package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/damonleelcx/ai-social-network/internal/externalauth"
	"github.com/damonleelcx/ai-social-network/internal/httpapi"
)

// Two copies, US and China, each with its own database, linked as peers. An account lives
// in one and works from both; nothing about it is stored in the other.
func TestOneAccountTwoRegions(t *testing.T) {
	us := setup(t)
	cn := setup(t)
	const secret = "0123456789abcdef0123456789abcdef-test"
	link := func(e *env, region string, other *env) {
		p, err := httpapi.NewPeer(region, other.api.URL, secret)
		if err != nil {
			t.Fatal(err)
		}
		e.srv.Peer = p
		e.auth.TokenPrefix = httpapi.TokenPrefix(region)
	}
	link(us, "us", cn)
	link(cn, "cn", us)
	// Handler() is built once in setup, so rebuild both with their peers.
	us.rehandle()
	cn.rehandle()

	countUsers := func(e *env, email string) int {
		var n int
		if err := e.pool.QueryRow(t.Context(), `SELECT count(*) FROM users WHERE lower(email)=lower($1)`, email).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// Someone signs up in the app (US) and later opens the mini program (China).
	ada := us.signup(t, "Ada", "ada@example.com")
	ada.verify()
	viaCN := &client{t: t, e: cn}
	var r authReply
	viaCN.do("POST", "/v1/auth/signin", map[string]any{"email": "ada@example.com", "password": "correct horse battery"}, 200, &r)
	if strings.HasPrefix(r.Token, "cn.") || r.User.ID != ada.id {
		t.Fatalf("signing in through China should give Ada's US session, got %+v", r)
	}
	viaCN.token = r.Token
	var me struct {
		User struct{ ID string } `json:"user"`
	}
	viaCN.do("GET", "/v1/me", nil, 200, &me)
	if me.User.ID != ada.id {
		t.Fatalf("China answered for someone else: %+v", me)
	}
	viaCN.do("GET", "/v1/profile", nil, 200, nil)
	// The live socket goes through too, and Suki's reply arrives on it.
	frames, stop := socketEvents(t, viaCN)
	defer stop()
	before := len(frames())
	viaCN.askSuki("hello from China")
	eventually(t, "a socket event through China", func() bool { return len(frames()) > before })
	if countUsers(cn, "ada@example.com") != 0 {
		t.Fatal("Ada's account was copied into China")
	}
	// The address is taken in both places.
	(&client{t: t, e: cn}).do("POST", "/v1/auth/signup", map[string]any{"email": "ADA@example.com", "password": "another horse battery", "name": "x", "accept_terms": true}, 422, nil)
	// A wrong password through China is still just wrong.
	(&client{t: t, e: cn}).do("POST", "/v1/auth/signin", map[string]any{"email": "ada@example.com", "password": "nope nope nope"}, 401, nil)

	// Someone signs up in China and later uses the app (US).
	bo := cn.signup(t, "Bo", "bo@example.cn")
	if !strings.HasPrefix(bo.token, "cn.") {
		t.Fatalf("China's sessions say they're China's: %q", bo.token)
	}
	bo.verify()
	viaUS := &client{t: t, e: us}
	viaUS.do("POST", "/v1/auth/signin", map[string]any{"email": "bo@example.cn", "password": "correct horse battery"}, 200, &r)
	if !strings.HasPrefix(r.Token, "cn.") {
		t.Fatalf("Bo's session should be China's, got %q", r.Token)
	}
	viaUS.token = r.Token
	viaUS.do("GET", "/v1/profile", nil, 200, nil)
	if countUsers(us, "bo@example.cn") != 0 {
		t.Fatal("Bo's account was copied into the US")
	}

	// A WeChat identity that already lives in the US signs in through China and lands there.
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"openid": strings.TrimPrefix(r.URL.Query().Get("js_code"), "code-")})
	}))
	defer fake.Close()
	us.srv.WeChatLogin = &externalauth.WeChat{AppID: "app", AppSecret: "s", URL: fake.URL}
	cn.srv.WeChatLogin = &externalauth.WeChat{AppID: "app", AppSecret: "s", URL: fake.URL}
	var old authReply
	(&client{t: t, e: us}).do("POST", "/v1/auth/wechat", map[string]any{"code": "code-wx1", "accept_terms": true}, 201, &old)
	var back authReply
	(&client{t: t, e: cn}).do("POST", "/v1/auth/wechat", map[string]any{"code": "code-wx1"}, 200, &back)
	if back.User.ID != old.User.ID || strings.HasPrefix(back.Token, "cn.") {
		t.Fatalf("WeChat user should be found in the US: %+v vs %+v", back, old)
	}
	// Someone new to WeChat signs up in China.
	var fresh authReply
	(&client{t: t, e: cn}).do("POST", "/v1/auth/wechat", map[string]any{"code": "code-wx2", "accept_terms": true}, 201, &fresh)
	if !strings.HasPrefix(fresh.Token, "cn.") {
		t.Fatalf("new WeChat account should live in China: %q", fresh.Token)
	}

	// The private API answers only the other copy.
	(&client{t: t, e: us}).do("POST", "/internal/peer/email", map[string]any{"email_hash": "x"}, 404, nil)
}
