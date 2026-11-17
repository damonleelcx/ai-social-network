package e2e

import "testing"

// Someone who signed up but never entered the emailed code can't use Suki, however they come
// back: with the session they got at sign-up, or by signing in again. /v1/me keeps saying the
// address isn't confirmed, so the app sends them back to the code screen.
func TestUnconfirmedEmailCannotSkipTheCode(t *testing.T) {
	e := setup(t)
	a := e.signup(t, "Ada", "ada@example.com")

	check := func(c *client) {
		t.Helper()
		var me struct {
			User struct {
				EmailVerified bool `json:"email_verified"`
			} `json:"user"`
		}
		c.do("GET", "/v1/me", nil, 200, &me)
		if me.User.EmailVerified {
			t.Fatal("email shows as verified without the code")
		}
		c.do("GET", "/v1/agent/messages", nil, 403, nil)
		c.do("POST", "/v1/agent/messages", map[string]any{"text": "hi"}, 403, nil)
		c.do("POST", "/v1/agent/opened", map[string]any{}, 403, nil)
		c.do("GET", "/v1/matches", nil, 403, nil)
	}
	check(a)

	// Signing in again changes nothing.
	b := &client{t: t, e: e, email: a.email}
	var r struct {
		Token string `json:"token"`
	}
	b.do("POST", "/v1/auth/signin", map[string]any{"email": a.email, "password": "correct horse battery"}, 200, &r)
	b.token = r.Token
	check(b)

	// The code is what lets them in.
	b.verify()
	b.do("GET", "/v1/agent/messages", nil, 200, nil)
}
