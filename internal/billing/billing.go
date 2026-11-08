// Package billing is credits: who pays for talking to Suki, what they've bought, and what
// they've used. A person's first matches are free. After that each message to Suki costs a
// credit. With no payment provider configured nobody can buy any, so nobody is charged.
package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// FreeMatches is how many matches a person gets before messages cost anything.
	FreeMatches = 1
	// CostPerMessage is what one message to Suki costs.
	CostPerMessage = 1

	signatureTolerance = 5 * time.Minute
)

type Package struct {
	ID          string `json:"id"`
	Credits     int    `json:"credits"`
	AmountCents int    `json:"amount_cents"`
	Currency    string `json:"currency"`
}

// Packages are what's for sale.
var Packages = []Package{
	{ID: "small", Credits: 100, AmountCents: 999, Currency: "usd"},
	{ID: "medium", Credits: 300, AmountCents: 2999, Currency: "usd"},
	{ID: "large", Credits: 1000, AmountCents: 9999, Currency: "usd"},
}

type Service struct {
	Pool *pgxpool.Pool
	// Stripe. Both empty means buying and charging are off.
	StripeKey     string
	WebhookSecret string
	// StripeURL is for tests. Empty means Stripe itself.
	StripeURL    string
	PublicOrigin string
	HTTP         *http.Client
	// WeChat is WeChat Pay for the mini program. Nil means it's off.
	WeChat *WeChat
	// Google is Google Play Billing for the Play build of the Android app. Nil means it's off.
	Google *GooglePlay
	// Rewards are earned credits (invites, check-in). Nil means DefaultRewards.
	Rewards *Rewards
	// Now is for tests. Nil means time.Now.
	Now func() time.Time
}

// ErrBadSignature means a webhook didn't come from the payment provider.
var ErrBadSignature = errors.New("bad signature")

// UserError is a message safe to show the person.
type UserError struct{ Msg string }

func (e *UserError) Error() string { return e.Msg }

func (s *Service) stripeEnabled() bool { return s != nil && s.StripeKey != "" && s.WebhookSecret != "" }

// Enabled says whether credits are on sale anywhere. Until they are, nobody is charged. Someone
// who can't buy where they are is sent to where they can.
func (s *Service) Enabled() bool {
	return s.stripeEnabled() || (s != nil && (s.WeChat != nil || s.Google != nil))
}

type Status struct {
	Balance         int             `json:"balance"`
	MatchesUsed     int             `json:"matches_used"`
	FreeMatches     int             `json:"free_matches"`
	Free            bool            `json:"free"`
	CostPerMessage  int             `json:"cost_per_message"`
	PaymentsEnabled bool            `json:"payments_enabled"`
	Packages        []Package       `json:"packages"`
	WeChatEnabled   bool            `json:"wechat_enabled"`
	WeChatPackages  []Package       `json:"wechat_packages"`
	GoogleEnabled   bool            `json:"google_enabled"`
	GoogleProducts  []GoogleProduct `json:"google_products"`
	GoogleAccount   string          `json:"google_account"`
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// isFree counts every match they've ever had, ended ones too, so unmatching can't refill it.
func (s *Service) isFree(ctx context.Context, q queryer, userID string) (bool, int, error) {
	var n int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM matches WHERE user_a=$1 OR user_b=$1`, userID).Scan(&n); err != nil {
		return false, 0, err
	}
	return n < FreeMatches || !s.Enabled(), n, nil
}

func (s *Service) Status(ctx context.Context, userID string) (*Status, error) {
	free, n, err := s.isFree(ctx, s.Pool, userID)
	if err != nil {
		return nil, err
	}
	st := &Status{MatchesUsed: n, FreeMatches: FreeMatches, Free: free, CostPerMessage: CostPerMessage, PaymentsEnabled: s.stripeEnabled(), Packages: Packages, WeChatEnabled: s.WeChat != nil, WeChatPackages: WeChatPackages,
		GoogleEnabled: s.Google != nil, GoogleProducts: GoogleProducts, GoogleAccount: GoogleAccountTag(userID)}
	err = s.Pool.QueryRow(ctx, `SELECT coalesce((SELECT balance FROM credit_balances WHERE user_id=$1),0)`, userID).Scan(&st.Balance)
	return st, err
}

// ChargeTurn takes payment for one message to Suki. False means they're out of credits. A
// retried turn lands on the same ledger row and isn't charged twice.
func (s *Service) ChargeTurn(ctx context.Context, userID, turnID string) (bool, error) {
	ok := true
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		free, _, err := s.isFree(ctx, tx, userID)
		if err != nil || free {
			return err
		}
		var charged bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM credit_ledger WHERE key=$1)`, "turn:"+turnID).Scan(&charged); err != nil || charged {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE credit_balances SET balance=balance-$2, updated_at=now() WHERE user_id=$1 AND balance>=$2`, userID, CostPerMessage)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			ok = false
			return nil
		}
		_, err = tx.Exec(ctx, `INSERT INTO credit_ledger (user_id, delta, reason, key) VALUES ($1,$2,'turn',$3)`, userID, -CostPerMessage, "turn:"+turnID)
		return err
	})
	return ok, err
}

// RefundTurn gives a charge back when Suki couldn't answer. It does nothing if nothing was
// charged, or if it was already refunded.
func (s *Service) RefundTurn(ctx context.Context, userID, turnID string) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var delta int
		err := tx.QueryRow(ctx, `SELECT delta FROM credit_ledger WHERE key=$1`, "turn:"+turnID).Scan(&delta)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return addCredits(ctx, tx, userID, -delta, "refund", "refund:"+turnID)
	})
}

// addCredits moves a balance once per key.
func addCredits(ctx context.Context, tx pgx.Tx, userID string, delta int, reason, key string) error {
	tag, err := tx.Exec(ctx, `INSERT INTO credit_ledger (user_id, delta, reason, key) VALUES ($1,$2,$3,$4) ON CONFLICT (key) DO NOTHING`, userID, delta, reason, key)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO credit_balances (user_id, balance) VALUES ($1,$2)
		ON CONFLICT (user_id) DO UPDATE SET balance=credit_balances.balance+EXCLUDED.balance, updated_at=now()`, userID, delta)
	return err
}

// Grant adds credits by hand, for support. The key makes it safe to repeat.
func (s *Service) Grant(ctx context.Context, userID string, credits int, key string) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		return addCredits(ctx, tx, userID, credits, "grant", "grant:"+key)
	})
}

type UsageRow struct {
	Delta     int       `json:"delta"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// Usage is the newest changes to their balance.
func (s *Service) Usage(ctx context.Context, userID string, limit int) ([]UsageRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT delta, reason, created_at FROM credit_ledger WHERE user_id=$1 ORDER BY id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UsageRow{}
	for rows.Next() {
		var u UsageRow
		if err := rows.Scan(&u.Delta, &u.Reason, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Service) stripeBase() string {
	if s.StripeURL != "" {
		return s.StripeURL
	}
	return "https://api.stripe.com"
}

// Checkout starts a purchase and returns the provider's payment page.
func (s *Service) Checkout(ctx context.Context, userID, packageID string) (string, error) {
	if !s.stripeEnabled() {
		return "", &UserError{"Buying credits isn't open yet."}
	}
	var pkg *Package
	for i := range Packages {
		if Packages[i].ID == packageID {
			pkg = &Packages[i]
		}
	}
	if pkg == nil {
		return "", &UserError{"Pick one of the credit packs."}
	}
	var id string
	if err := s.Pool.QueryRow(ctx, `INSERT INTO credit_purchases (user_id, package, credits, amount_cents, currency) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		userID, pkg.ID, pkg.Credits, pkg.AmountCents, pkg.Currency).Scan(&id); err != nil {
		return "", err
	}
	form := url.Values{
		"mode":                                   {"payment"},
		"success_url":                            {s.PublicOrigin + "/credits/done"},
		"cancel_url":                             {s.PublicOrigin + "/credits/done?cancelled=1"},
		"client_reference_id":                    {id},
		"metadata[purchase_id]":                  {id},
		"line_items[0][quantity]":                {"1"},
		"line_items[0][price_data][currency]":    {pkg.Currency},
		"line_items[0][price_data][unit_amount]": {strconv.Itoa(pkg.AmountCents)},
		"line_items[0][price_data][product_data][name]": {fmt.Sprintf("SynAgent: %d credits", pkg.Credits)},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.stripeBase()+"/v1/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(s.StripeKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := s.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		// The body can repeat what we sent, and that stays out of logs and errors.
		return "", fmt.Errorf("payment provider: status %d", resp.StatusCode)
	}
	var session struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &session); err != nil || session.URL == "" {
		return "", errors.New("payment provider: no checkout page")
	}
	if _, err := s.Pool.Exec(ctx, `UPDATE credit_purchases SET session_id=$2 WHERE id=$1`, id, session.ID); err != nil {
		return "", err
	}
	return session.URL, nil
}

// Paid is a purchase that has just been completed.
type Paid struct {
	UserID  string
	Credits int
	Balance int
}

// HandleWebhook takes the provider's report of a payment. It returns nil Paid for anything
// that adds nothing, including a repeat of an event it has already handled.
func (s *Service) HandleWebhook(ctx context.Context, payload []byte, signature string, now time.Time) (*Paid, error) {
	if !s.stripeEnabled() || !validSignature(payload, signature, s.WebhookSecret, now) {
		return nil, ErrBadSignature
	}
	var ev struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID            string `json:"id"`
				PaymentStatus string `json:"payment_status"`
				AmountTotal   int    `json:"amount_total"`
				Currency      string `json:"currency"`
				Reference     string `json:"client_reference_id"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	o := ev.Data.Object
	if (ev.Type != "checkout.session.completed" && ev.Type != "checkout.session.async_payment_succeeded") || o.PaymentStatus != "paid" {
		return nil, nil
	}
	return s.completePurchase(ctx, o.Reference, o.ID, o.AmountTotal, o.Currency)
}

// completePurchase adds what was paid for, once. It returns nil Paid when there's nothing
// to add. reference is the purchase id, with or without its hyphens.
func (s *Service) completePurchase(ctx context.Context, reference, providerID string, amount int, paidCurrency string) (*Paid, error) {
	var paid *Paid
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var userID *string
		var credits, cents int
		var currency, status string
		err := tx.QueryRow(ctx, `SELECT user_id, credits, amount_cents, currency, status FROM credit_purchases WHERE replace(id::text,'-','')=replace($1,'-','') FOR UPDATE`, reference).
			Scan(&userID, &credits, &cents, &currency, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			// Some other payment on the same Stripe account. Sending it again won't help.
			slog.Warn("payment for an unknown purchase", "session", providerID)
			return nil
		}
		if err != nil {
			return err
		}
		if status == "paid" {
			return nil
		}
		if amount != cents || !strings.EqualFold(paidCurrency, currency) {
			return fmt.Errorf("purchase %s: paid %d %s, expected %d %s", reference, amount, paidCurrency, cents, currency)
		}
		if _, err := tx.Exec(ctx, `UPDATE credit_purchases SET status='paid', paid_at=now(), session_id=$2 WHERE replace(id::text,'-','')=replace($1,'-','')`, reference, providerID); err != nil {
			return err
		}
		if userID == nil {
			return nil // They deleted their account while paying.
		}
		if err := addCredits(ctx, tx, *userID, credits, "purchase", "purchase:"+strings.ReplaceAll(reference, "-", "")); err != nil {
			return err
		}
		p := Paid{UserID: *userID, Credits: credits}
		if err := tx.QueryRow(ctx, `SELECT balance FROM credit_balances WHERE user_id=$1`, *userID).Scan(&p.Balance); err != nil {
			return err
		}
		paid = &p
		return nil
	})
	return paid, err
}

// validSignature checks Stripe's signature: an HMAC of "timestamp.payload", recent enough
// that a recorded request can't be replayed later.
func validSignature(payload []byte, header, secret string, now time.Time) bool {
	var ts string
	var sigs []string
	for _, part := range strings.Split(header, ",") {
		k, v, _ := strings.Cut(strings.TrimSpace(part), "=")
		switch k {
		case "t":
			ts = v
		case "v1":
			sigs = append(sigs, v)
		}
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	if d := now.Sub(time.Unix(sec, 0)); d > signatureTolerance || d < -signatureTolerance {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(payload)
	want := mac.Sum(nil)
	for _, sig := range sigs {
		if got, err := hex.DecodeString(sig); err == nil && hmac.Equal(got, want) {
			return true
		}
	}
	return false
}
