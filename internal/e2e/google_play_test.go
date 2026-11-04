package e2e

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/damonleelcx/ai-social-network/internal/billing"
)

// A stand-in for Google: sign-in, the purchase lookup, and consume.
func TestGooglePlayAddsCredits(t *testing.T) {
	e := setup(t)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()
	b := e.signup(t, "Bea", "bea@example.com")
	b.verify()

	var mu sync.Mutex
	consumed := map[string]int{}
	purchases := map[string]map[string]any{
		"good":    {"purchaseState": 0, "obfuscatedExternalAccountId": billing.GoogleAccountTag(a.id)},
		"pending": {"purchaseState": 2, "obfuscatedExternalAccountId": billing.GoogleAccountTag(a.id)},
		"void":    {"purchaseState": 1, "obfuscatedExternalAccountId": billing.GoogleAccountTag(a.id)},
	}
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_ = r.ParseForm()
			if parts := strings.Split(r.PostForm.Get("assertion"), "."); len(parts) != 3 {
				w.WriteHeader(400)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "expires_in": 3600})
			return
		}
		if r.Header.Get("Authorization") != "Bearer at" {
			w.WriteHeader(401)
			return
		}
		tok := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		if name, ok := strings.CutSuffix(tok, ":consume"); ok {
			mu.Lock()
			consumed[name]++
			mu.Unlock()
			w.WriteHeader(204)
			return
		}
		p, ok := purchases[tok]
		if !ok {
			w.WriteHeader(404)
			return
		}
		_ = json.NewEncoder(w).Encode(p)
	}))
	defer fake.Close()
	e.billing.Google = &billing.GooglePlay{PackageName: "space.herosagent.synagent", ClientEmail: "sa@example.com", PrivateKey: key, APIURL: fake.URL, TokenURL: fake.URL + "/token"}

	var st billing.Status
	a.do("GET", "/v1/credits", nil, 200, &st)
	if !st.GoogleEnabled || len(st.GoogleProducts) != 3 || st.GoogleAccount != billing.GoogleAccountTag(a.id) {
		t.Fatalf("status: %+v", st)
	}
	verify := func(who *client, product, token string, code int) map[string]any {
		out := map[string]any{}
		who.do("POST", "/v1/credits/google/verify", map[string]any{"product_id": product, "purchase_token": token}, code, &out)
		return out
	}
	balance := func() int {
		a.do("GET", "/v1/credits", nil, 200, &st)
		return st.Balance
	}

	// A made-up token, a made-up product, a cancelled purchase, and someone else's: nothing.
	verify(a, "credits_100", "made-up", 422)
	verify(a, "free_credits", "good", 422)
	verify(a, "credits_100", "void", 422)
	verify(b, "credits_100", "good", 422)
	if balance() != 0 || consumed["good"] != 0 {
		t.Fatalf("credits from a bad report: %d, consumed %d", st.Balance, consumed["good"])
	}

	// Not paid yet: nothing, but not a failure.
	if out := verify(a, "credits_100", "pending", 200); out["pending"] != true {
		t.Fatalf("pending: %v", out)
	}
	if balance() != 0 {
		t.Fatal("credits before a pending purchase finished")
	}

	// Paid, and reported twice: credited once, and consumed.
	verify(a, "credits_100", "good", 200)
	verify(a, "credits_100", "good", 200)
	if balance() != 100 {
		t.Fatalf("balance: %d", st.Balance)
	}
	mu.Lock()
	defer mu.Unlock()
	if consumed["good"] == 0 {
		t.Fatal("not consumed, so they couldn't buy this pack again")
	}
}
