package e2e

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/damonleelcx/ai-social-network/internal/externalauth"
)

func TestGoogleSignUpAndSignIn(t *testing.T) {
	e := setup(t)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	stranger, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	enc := base64.RawURLEncoding
	certs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": "k1", "n": enc.EncodeToString(key.N.Bytes()), "e": enc.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
	}))
	defer certs.Close()

	claims := func(sub, email string) map[string]any {
		return map[string]any{"iss": "https://accounts.google.com", "aud": "web-client", "sub": sub, "email": email,
			"email_verified": true, "name": "Ada G", "exp": time.Now().Add(time.Hour).Unix()}
	}
	signWith := func(k *rsa.PrivateKey, alg string, c map[string]any) string {
		head, _ := json.Marshal(map[string]string{"alg": alg, "kid": "k1"})
		body, _ := json.Marshal(c)
		unsigned := enc.EncodeToString(head) + "." + enc.EncodeToString(body)
		sum := sha256.Sum256([]byte(unsigned))
		sig, _ := rsa.SignPKCS1v15(rand.Reader, k, crypto.SHA256, sum[:])
		return unsigned + "." + enc.EncodeToString(sig)
	}
	tok := func(c map[string]any) string { return signWith(key, "RS256", c) }
	with := func(c map[string]any, k string, v any) map[string]any {
		c[k] = v
		return c
	}

	anon := &client{t: t, e: e}
	in := func(token string, terms bool, want int) authReply {
		var r authReply
		anon.do("POST", "/v1/auth/google", map[string]any{"id_token": token, "accept_terms": terms}, want, &r)
		return r
	}

	// Off until it's configured.
	in(tok(claims("g1", "ada@example.com")), true, 422)
	e.srv.GoogleLogin = &externalauth.Google{ClientID: "web-client", CertsURL: certs.URL}

	// Tokens that mustn't get in.
	in("not.a.token", true, 422)
	in(signWith(stranger, "RS256", claims("g1", "ada@example.com")), true, 422)
	in(signWith(key, "none", claims("g1", "ada@example.com")), true, 422)
	in(tok(with(claims("g1", "ada@example.com"), "aud", "another-app")), true, 422)
	in(tok(with(claims("g1", "ada@example.com"), "iss", "https://evil.example")), true, 422)
	in(tok(with(claims("g1", "ada@example.com"), "exp", time.Now().Add(-time.Hour).Unix())), true, 422)
	in(tok(with(claims("g1", "ada@example.com"), "email_verified", false)), true, 422)

	// Someone new accepts the terms first, and is verified at once with their real address.
	good := tok(claims("g1", "ada@example.com"))
	in(good, false, 422)
	first := in(good, true, 201)
	if first.Token == "" || !first.User.EmailVerified || first.User.HasPassword || first.User.Email != "ada@example.com" {
		t.Fatalf("new account: %+v", first)
	}
	if again := in(good, false, 200); again.User.ID != first.User.ID {
		t.Fatalf("a second account: %s", again.User.ID)
	}

	// An address someone verified with us: Google's account joins it, and their password stays.
	b := e.signup(t, "Bea", "bea@example.com")
	b.verify()
	joined := in(tok(claims("g2", "bea@example.com")), false, 200)
	if joined.User.ID != b.id || !joined.User.HasPassword {
		t.Fatalf("join: %+v", joined)
	}
	// An address nobody verified: whoever signed up with it may not own it, so no joining.
	c := e.signup(t, "Cy", "cy@example.com")
	in(tok(claims("g3", "cy@example.com")), true, 422)
	_ = c

	// Deleting takes this person's own Google account.
	a := &client{t: t, e: e, token: first.Token, id: first.User.ID}
	a.do("POST", "/v1/auth/google/delete", map[string]any{"id_token": tok(claims("g2", "bea@example.com"))}, 401, nil)
	a.do("POST", "/v1/auth/google/delete", map[string]any{"id_token": good}, 200, nil)
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE id=$1`, first.User.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("account still there: %d %v", n, err)
	}
}
