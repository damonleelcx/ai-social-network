package billing

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// GoogleProduct is a one-time consumable product set up in Play Console, with this same id.
// Prices live in Play Console, in each country's currency, so none are kept here.
type GoogleProduct struct {
	ID      string `json:"id"`
	Credits int    `json:"credits"`
}

var GoogleProducts = []GoogleProduct{
	{ID: "credits_100", Credits: 100},
	{ID: "credits_300", Credits: 300},
	{ID: "credits_1000", Credits: 1000},
}

// GooglePlay checks purchases with the Google Play Developer API, as a service account.
type GooglePlay struct {
	PackageName string
	ClientEmail string
	PrivateKey  *rsa.PrivateKey
	// Tests point these elsewhere. Empty means Google itself.
	APIURL   string
	TokenURL string

	mu      sync.Mutex
	token   string
	expires time.Time
}

// NewGooglePlay returns nil when there's no service account, and an error when it's unusable.
func NewGooglePlay(packageName, serviceAccountJSON string) (*GooglePlay, error) {
	if serviceAccountJSON == "" {
		return nil, nil
	}
	if packageName == "" {
		return nil, errors.New("google play: no package name")
	}
	var sa struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
	}
	if err := json.Unmarshal([]byte(serviceAccountJSON), &sa); err != nil {
		return nil, errors.New("google play: the service account isn't JSON")
	}
	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil || sa.ClientEmail == "" {
		return nil, errors.New("google play: the service account has no key or email")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("google play: the service account key can't be read")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("google play: the service account key isn't RSA")
	}
	return &GooglePlay{PackageName: packageName, ClientEmail: sa.ClientEmail, PrivateKey: key}, nil
}

// GoogleAccountTag ties a purchase to the person who made it, without giving Google their id.
func GoogleAccountTag(userID string) string {
	sum := sha256.Sum256([]byte("synagent:" + userID))
	return hex.EncodeToString(sum[:])
}

func (s *Service) googleToken(ctx context.Context) (string, error) {
	g := s.Google
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.token != "" && time.Now().Before(g.expires) {
		return g.token, nil
	}
	tokenURL := g.TokenURL
	if tokenURL == "" {
		tokenURL = "https://oauth2.googleapis.com/token"
	}
	enc := base64.RawURLEncoding
	now := time.Now().Unix()
	claims, err := json.Marshal(map[string]any{
		"iss": g.ClientEmail, "scope": "https://www.googleapis.com/auth/androidpublisher",
		"aud": tokenURL, "iat": now, "exp": now + 3600,
	})
	if err != nil {
		return "", err
	}
	unsigned := enc.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." + enc.EncodeToString(claims)
	sum := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(rand.Reader, g.PrivateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	form := url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"}, "assertion": {unsigned + "." + enc.EncodeToString(sig)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client().Do(req)
	if err != nil {
		return "", errors.New("google play: can't reach Google")
	}
	defer resp.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&out)
	if resp.StatusCode != http.StatusOK || out.AccessToken == "" {
		return "", fmt.Errorf("google play: sign-in status %d", resp.StatusCode)
	}
	g.token = out.AccessToken
	g.expires = time.Now().Add(time.Duration(out.ExpiresIn)*time.Second - time.Minute)
	return g.token, nil
}

// googleCall returns the response's status and body.
func (s *Service) googleCall(ctx context.Context, method, productID, purchaseToken, suffix string) (int, []byte, error) {
	g := s.Google
	access, err := s.googleToken(ctx)
	if err != nil {
		return 0, nil, err
	}
	base := g.APIURL
	if base == "" {
		base = "https://androidpublisher.googleapis.com"
	}
	u := fmt.Sprintf("%s/androidpublisher/v3/applications/%s/purchases/products/%s/tokens/%s%s", base,
		url.PathEscape(g.PackageName), url.PathEscape(productID), url.PathEscape(purchaseToken), suffix)
	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := s.client().Do(req)
	if err != nil {
		return 0, nil, errors.New("google play: can't reach Google")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, raw, err
}

// GoogleResult is what checking a purchase came to.
type GoogleResult struct {
	Pending bool
	// Paid is set the first time a purchase is credited, and nil on any repeat.
	Paid *Paid
}

// VerifyGooglePurchase asks Google about a purchase the app reports. Only Google's answer
// counts. The purchase has to be this person's, it adds credits once however often it's
// reported, and it's consumed only after they're added, so a failure here can be retried.
func (s *Service) VerifyGooglePurchase(ctx context.Context, userID, productID, purchaseToken string) (*GoogleResult, error) {
	if s.Google == nil {
		return nil, &UserError{"Buying credits isn't open yet."}
	}
	var product *GoogleProduct
	for i := range GoogleProducts {
		if GoogleProducts[i].ID == productID {
			product = &GoogleProducts[i]
		}
	}
	if product == nil {
		return nil, &UserError{"Pick one of the credit packs."}
	}
	if purchaseToken == "" || len(purchaseToken) > 4096 {
		return nil, &UserError{"That purchase couldn't be checked."}
	}
	status, raw, err := s.googleCall(ctx, http.MethodGet, productID, purchaseToken, "")
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound || status == http.StatusBadRequest || status == http.StatusGone {
		return nil, &UserError{"That purchase couldn't be checked."}
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("google play: status %d", status)
	}
	var p struct {
		PurchaseState               int    `json:"purchaseState"`
		ObfuscatedExternalAccountID string `json:"obfuscatedExternalAccountId"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	switch p.PurchaseState {
	case 0:
	case 2:
		return &GoogleResult{Pending: true}, nil
	default:
		return nil, &UserError{"That purchase was cancelled."}
	}
	if p.ObfuscatedExternalAccountID != GoogleAccountTag(userID) {
		slog.Warn("google purchase for another account", "user", userID)
		return nil, &UserError{"That purchase belongs to another account."}
	}
	sum := sha256.Sum256([]byte(purchaseToken))
	ref := "google:" + hex.EncodeToString(sum[:])
	res := &GoogleResult{}
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		// Play knows what was paid, in which currency, so the amount here is left at zero.
		tag, err := tx.Exec(ctx, `INSERT INTO credit_purchases (user_id, package, credits, amount_cents, currency, status, session_id, paid_at)
			VALUES ($1,$2,$3,0,'google_play','paid',$4,now()) ON CONFLICT (session_id) DO NOTHING`, userID, product.ID, product.Credits, ref)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		if err := addCredits(ctx, tx, userID, product.Credits, "purchase", "purchase:"+ref); err != nil {
			return err
		}
		paid := Paid{UserID: userID, Credits: product.Credits}
		if err := tx.QueryRow(ctx, `SELECT balance FROM credit_balances WHERE user_id=$1`, userID).Scan(&paid.Balance); err != nil {
			return err
		}
		res.Paid = &paid
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Already consumed is fine. Anything else is retried the next time the app reports it.
	if code, _, err := s.googleCall(ctx, http.MethodPost, productID, purchaseToken, ":consume"); err != nil || (code != http.StatusOK && code != http.StatusNoContent) {
		slog.Warn("google consume", "status", code, "err", err)
	}
	return res, nil
}
