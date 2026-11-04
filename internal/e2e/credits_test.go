package e2e

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/damonleelcx/ai-social-network/internal/billing"
)

func (e *env) stripeEvent(t *testing.T, payload []byte, secret string) int {
	t.Helper()
	ts := fmt.Sprint(time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(payload)
	req, _ := http.NewRequest("POST", e.api.URL+"/v1/billing/stripe", strings.NewReader(string(payload)))
	req.Header.Set("Stripe-Signature", "t="+ts+",v1="+hex.EncodeToString(mac.Sum(nil)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode
}

// The whole money path: free until the fifth match, then out of credits, buy, spend.
func TestCreditsAfterFiveMatches(t *testing.T) {
	e := setup(t)
	var form string
	stripe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		form = string(b)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "cs_test_1", "url": "https://checkout.example/cs_test_1"})
	}))
	defer stripe.Close()
	e.billing.StripeKey, e.billing.WebhookSecret, e.billing.StripeURL = "sk_test_x", testWebhookSecret, stripe.URL

	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()
	var st billing.Status
	status := func() billing.Status {
		a.do("GET", "/v1/credits", nil, 200, &st)
		return st
	}
	if s := status(); !s.Free || s.Balance != 0 || s.FreeMatches != 5 {
		t.Fatalf("a new person: %+v", s)
	}
	body, _ := a.askSuki("hello there")
	if strings.Contains(body, "out of credits") {
		t.Fatal("charged before the free matches were used")
	}

	// Five matches, the last of them ended: unmatching mustn't refill the free ones.
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		o := e.signup(t, fmt.Sprintf("P%d", i), fmt.Sprintf("p%d@example.com", i))
		var pid string
		if err := e.pool.QueryRow(ctx, `INSERT INTO proposals (user_a,user_b,status) VALUES (least($1::uuid,$2::uuid),greatest($1::uuid,$2::uuid),'matched') RETURNING id`, a.id, o.id).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		status := "active"
		if i == 4 {
			status = "unmatched"
		}
		if _, err := e.pool.Exec(ctx, `INSERT INTO matches (proposal_id,user_a,user_b,status) VALUES ($1,least($2::uuid,$3::uuid),greatest($2::uuid,$3::uuid),$4)`, pid, a.id, o.id, status); err != nil {
			t.Fatal(err)
		}
	}
	if s := status(); s.Free || s.MatchesUsed != 5 {
		t.Fatalf("after five matches: %+v", s)
	}

	// Out of credits: Suki says so, with a way to buy, and no credit goes below zero.
	body, meta := a.askSuki("hello again")
	if !strings.Contains(body, "out of credits") {
		t.Fatalf("no out-of-credits reply: %q", body)
	}
	cards, _ := meta["cards"].([]any)
	if len(cards) != 1 || cards[0].(map[string]any)["kind"] != "buy_credits" {
		t.Fatalf("cards: %v", cards)
	}

	// Someone in trouble is never asked to pay.
	body, _ = a.askSuki("I'm scared he is threatening to hurt me")
	if strings.Contains(body, "out of credits") {
		t.Fatalf("a safety message was charged: %q", body)
	}

	// Buying: an unknown pack is refused, and a real one goes to the provider.
	a.do("POST", "/v1/credits/checkout", map[string]any{"package": "nope"}, 422, nil)
	var co struct {
		URL string `json:"url"`
	}
	a.do("POST", "/v1/credits/checkout", map[string]any{"package": "small"}, 200, &co)
	if co.URL != "https://checkout.example/cs_test_1" || !strings.Contains(form, "unit_amount%5D=999") {
		t.Fatalf("checkout: %q form %q", co.URL, form)
	}
	if status().Balance != 0 {
		t.Fatal("credits before payment")
	}

	var purchase string
	if err := e.pool.QueryRow(ctx, `SELECT id FROM credit_purchases WHERE user_id=$1`, a.id).Scan(&purchase); err != nil {
		t.Fatal(err)
	}
	event := func(amount int) []byte {
		raw, _ := json.Marshal(map[string]any{"type": "checkout.session.completed", "data": map[string]any{"object": map[string]any{
			"id": "cs_test_1", "payment_status": "paid", "amount_total": amount, "currency": "usd", "client_reference_id": purchase}}})
		return raw
	}
	// Someone else's payment on the same Stripe account: acknowledged, and nothing added.
	other, _ := json.Marshal(map[string]any{"type": "checkout.session.completed", "data": map[string]any{"object": map[string]any{
		"id": "cs_other", "payment_status": "paid", "amount_total": 999, "currency": "usd", "client_reference_id": "not-ours"}}})
	if c := e.stripeEvent(t, other, testWebhookSecret); c != 200 {
		t.Fatalf("another payment: %d", c)
	}
	// Forged, or the wrong amount: nothing.
	if c := e.stripeEvent(t, event(999), "whsec_wrong"); c != 400 {
		t.Fatalf("forged webhook: %d", c)
	}
	if c := e.stripeEvent(t, event(1), testWebhookSecret); c == 200 {
		t.Fatal("a wrong amount was accepted")
	}
	if status().Balance != 0 {
		t.Fatal("credits from a bad webhook")
	}
	// Paid, and delivered twice: credited once.
	for i := 0; i < 2; i++ {
		if c := e.stripeEvent(t, event(999), testWebhookSecret); c != 200 {
			t.Fatalf("webhook: %d", c)
		}
	}
	if s := status(); s.Balance != 100 {
		t.Fatalf("balance after paying: %d", s.Balance)
	}

	// Spending, and what Suki didn't answer comes back.
	body, _ = a.askSuki("are you there")
	if strings.Contains(body, "out of credits") {
		t.Fatalf("still out of credits: %q", body)
	}
	if s := status(); s.Balance != 99 {
		t.Fatalf("balance after a message: %d", s.Balance)
	}
	var turn string
	if err := e.pool.QueryRow(ctx, `SELECT substring(key from 6) FROM credit_ledger WHERE user_id=$1 AND reason='turn'`, a.id).Scan(&turn); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := e.billing.RefundTurn(ctx, a.id, turn); err != nil {
			t.Fatal(err)
		}
	}
	if s := status(); s.Balance != 100 {
		t.Fatalf("balance after a refund: %d", s.Balance)
	}
	var usage struct {
		Usage []billing.UsageRow `json:"usage"`
	}
	a.do("GET", "/v1/credits/usage", nil, 200, &usage)
	if len(usage.Usage) != 3 || usage.Usage[0].Reason != "refund" || usage.Usage[2].Reason != "purchase" {
		t.Fatalf("usage: %+v", usage.Usage)
	}
}

// With no payment provider nobody can buy, so nobody can be charged.
func TestNoChargeWithoutPayments(t *testing.T) {
	e := setup(t)
	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		o := e.signup(t, fmt.Sprintf("P%d", i), fmt.Sprintf("p%d@example.com", i))
		var pid string
		if err := e.pool.QueryRow(ctx, `INSERT INTO proposals (user_a,user_b,status) VALUES (least($1::uuid,$2::uuid),greatest($1::uuid,$2::uuid),'matched') RETURNING id`, a.id, o.id).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		if _, err := e.pool.Exec(ctx, `INSERT INTO matches (proposal_id,user_a,user_b) VALUES ($1,least($2::uuid,$3::uuid),greatest($2::uuid,$3::uuid))`, pid, a.id, o.id); err != nil {
			t.Fatal(err)
		}
	}
	body, _ := a.askSuki("hello")
	if strings.Contains(body, "out of credits") {
		t.Fatalf("charged with no way to buy: %q", body)
	}
	a.do("POST", "/v1/credits/checkout", map[string]any{"package": "small"}, 422, nil)
	if c := e.stripeEvent(t, []byte(`{}`), "anything"); c != 400 {
		t.Fatalf("webhook with payments off: %d", c)
	}
}

// A store only sells inside its own app, so only people using that app can be charged.
func TestChargesOnlyPeopleWhoCanBuy(t *testing.T) {
	e := setup(t)
	e.billing.Google = &billing.GooglePlay{PackageName: "space.herosagent.synagent"}
	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		o := e.signup(t, fmt.Sprintf("P%d", i), fmt.Sprintf("p%d@example.com", i))
		var pid string
		if err := e.pool.QueryRow(ctx, `INSERT INTO proposals (user_a,user_b,status) VALUES (least($1::uuid,$2::uuid),greatest($1::uuid,$2::uuid),'matched') RETURNING id`, a.id, o.id).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		if _, err := e.pool.Exec(ctx, `INSERT INTO matches (proposal_id,user_a,user_b) VALUES ($1,least($2::uuid,$3::uuid),greatest($2::uuid,$3::uuid))`, pid, a.id, o.id); err != nil {
			t.Fatal(err)
		}
	}
	var st billing.Status
	if a.do("GET", "/v1/credits", nil, 200, &st); !st.Free {
		t.Fatalf("charged on the web with no way to buy: %+v", st)
	}
	if body, _ := a.askSuki("hello"); strings.Contains(body, "out of credits") {
		t.Fatalf("stopped with no way to buy: %q", body)
	}

	// A made-up store changes nothing.
	a.store = "elsewhere"
	if a.do("GET", "/v1/credits", nil, 200, &st); !st.Free {
		t.Fatalf("a made-up store: %+v", st)
	}

	a.store = "play"
	if a.do("GET", "/v1/credits", nil, 200, &st); st.Free {
		t.Fatalf("free on the Play build: %+v", st)
	}
	if body, _ := a.askSuki("hello again"); !strings.Contains(body, "out of credits") {
		t.Fatalf("Play build with no credits: %q", body)
	}

	// They can buy on their phone, so moving to the web doesn't make it free.
	a.store = ""
	if a.do("GET", "/v1/credits", nil, 200, &st); st.Free {
		t.Fatalf("free again on the web: %+v", st)
	}

	// A store that isn't selling can't be the way to buy.
	e.billing.Google = nil
	if a.do("GET", "/v1/credits", nil, 200, &st); !st.Free {
		t.Fatalf("charged with the store closed: %+v", st)
	}
}

const testWebhookSecret = "whsec_test"
