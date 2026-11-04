package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/damonleelcx/ai-social-network/internal/externalauth"
)

type authReply struct {
	Token string `json:"token"`
	User  struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		HasPassword   bool   `json:"has_password"`
	} `json:"user"`
}

func TestWeChatSignUpAndSignIn(t *testing.T) {
	e := setup(t)
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In the fake, the code says who's signing in.
		code := r.URL.Query().Get("js_code")
		if !strings.HasPrefix(code, "code-") {
			_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 40029})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"openid": strings.TrimPrefix(code, "code-")})
	}))
	defer fake.Close()

	anon := &client{t: t, e: e}
	// Not configured: refused, and nothing is created.
	anon.do("POST", "/v1/auth/wechat", map[string]any{"code": "code-ada", "accept_terms": true}, 422, nil)
	e.srv.WeChatLogin = &externalauth.WeChat{AppID: "app", AppSecret: "secret", URL: fake.URL}

	anon.do("POST", "/v1/auth/wechat", map[string]any{"code": "bad", "accept_terms": true}, 422, nil)
	// Someone new has to accept the terms first.
	anon.do("POST", "/v1/auth/wechat", map[string]any{"code": "code-ada"}, 422, nil)

	var first authReply
	anon.do("POST", "/v1/auth/wechat", map[string]any{"code": "code-ada", "accept_terms": true, "language": "zh"}, 201, &first)
	if first.Token == "" || !first.User.EmailVerified || first.User.HasPassword || !strings.HasSuffix(first.User.Email, ".invalid") {
		t.Fatalf("new account: %+v", first)
	}
	a := &client{t: t, e: e, token: first.Token, id: first.User.ID}
	// Past the verified gate at once, and in the language they asked for.
	a.do("GET", "/v1/profile", nil, 200, nil)
	body, _ := a.askSuki("hello")
	if body == "" {
		t.Fatal("Suki didn't answer a WeChat account")
	}

	// Coming back finds the same account, with no terms needed.
	var again authReply
	anon.do("POST", "/v1/auth/wechat", map[string]any{"code": "code-ada"}, 200, &again)
	if again.User.ID != first.User.ID {
		t.Fatalf("a second account: %s", again.User.ID)
	}
	// Someone else gets their own.
	var other authReply
	inviter, code := e.inviterCode(t, "inviter@example.com")
	anon.do("POST", "/v1/auth/wechat", map[string]any{"code": "code-bea", "accept_terms": true, "ref": code}, 201, &other)
	if got := e.referrerOf(t, other.User.ID); got != inviter {
		t.Fatalf("invited by %s, want %s", got, inviter)
	}
	if other.User.ID == first.User.ID {
		t.Fatal("two people on one account")
	}
	// There's no password to guess.
	anon.do("POST", "/v1/auth/signin", map[string]any{"email": first.User.Email, "password": ""}, 401, nil)

	// Deleting takes this person's own WeChat, and nobody else's.
	a.do("POST", "/v1/auth/wechat/delete", map[string]any{"code": "code-bea"}, 401, nil)
	a.do("GET", "/v1/profile", nil, 200, nil)
	a.do("POST", "/v1/auth/wechat/delete", map[string]any{"code": "code-ada"}, 200, nil)
	a.do("GET", "/v1/profile", nil, 401, nil)
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE id=$1`, first.User.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("account still there: %d %v", n, err)
	}

	// Banned: they can't sign back in, or come back as new.
	if err := e.auth.Ban(context.Background(), other.User.ID); err != nil {
		t.Fatal(err)
	}
	anon.do("POST", "/v1/auth/wechat", map[string]any{"code": "code-bea", "accept_terms": true}, 403, nil)
}
