package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/damonleelcx/ai-social-network/internal/billing"
	"github.com/damonleelcx/ai-social-network/internal/workflows"
)

func (e *env) signupWithRef(t *testing.T, name, email, ref string) *client {
	t.Helper()
	c := &client{t: t, e: e, email: email}
	var r struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	c.do("POST", "/v1/auth/signup", map[string]any{"email": email, "password": "correct horse battery", "name": name, "accept_terms": true, "language": "en", "ref": ref}, 201, &r)
	c.token, c.id = r.Token, r.User.ID
	return c
}

// inviterCode makes someone with an invite code.
func (e *env) inviterCode(t *testing.T, email string) (string, string) {
	t.Helper()
	c := e.signup(t, "Inviter", email)
	c.verify()
	var st billing.ReferralStatus
	c.do("GET", "/v1/referral", nil, 200, &st)
	return c.id, st.Code
}

func (e *env) referrerOf(t *testing.T, invitee string) string {
	t.Helper()
	var id string
	if err := e.pool.QueryRow(context.Background(), `SELECT referrer_id FROM referrals WHERE invitee_id=$1`, invitee).Scan(&id); err != nil {
		t.Fatalf("no referral for %s: %v", invitee, err)
	}
	return id
}

func TestReferralCredits(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()
	var st billing.ReferralStatus
	a.do("GET", "/v1/referral", nil, 200, &st)
	if len(st.Code) != 8 || !strings.HasSuffix(st.Link, "/app/?ref="+st.Code) || st.CreditsPerFriend != billing.ReferralCredits {
		t.Fatalf("status: %+v", st)
	}
	var again billing.ReferralStatus
	a.do("GET", "/v1/referral", nil, 200, &again)
	if again.Code != st.Code {
		t.Fatalf("the code changed: %s then %s", st.Code, again.Code)
	}

	balance := func(id string) int {
		var n int
		_ = e.pool.QueryRow(ctx, `SELECT coalesce((SELECT balance FROM credit_balances WHERE user_id=$1),0)`, id).Scan(&n)
		return n
	}

	// Lower case, as someone might type it. Signing up pays nothing yet.
	b := e.signupWithRef(t, "Bo", "bo@example.com", strings.ToLower(st.Code))
	a.do("GET", "/v1/referral", nil, 200, &st)
	if st.Joined != 1 || st.Rewarded != 0 || balance(a.id) != 0 {
		t.Fatalf("after sign-up: %+v, balance %d", st, balance(a.id))
	}

	// An unfinished profile pays nothing.
	if err := workflows.AfterProfileChange(ctx, b.id); err != nil {
		t.Fatal(err)
	}
	if got := balance(a.id); got != 0 {
		t.Fatalf("paid for an unfinished profile: %d", got)
	}

	// Their profile going live pays, and only once.
	if _, err := e.pool.Exec(ctx, `UPDATE profiles SET birthdate='1995-01-01', gender='man', seeking='{woman}', city_key='tokyo', city_name='Tokyo', goal='dating', interests='{a,b,c}', bio='hi' WHERE user_id=$1`, b.id); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO photos (user_id, s3_key, mime, width, height, sha256, status) VALUES ($1,'k','image/jpeg',1,1,'x','approved')`, b.id); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := workflows.AfterProfileChange(ctx, b.id); err != nil {
			t.Fatal(err)
		}
		if err := workflows.RewardReferrer(ctx, b.id); err != nil {
			t.Fatal(err)
		}
	}
	if got := balance(a.id); got != billing.ReferralCredits {
		t.Fatalf("balance after reward: %d", got)
	}
	a.do("GET", "/v1/referral", nil, 200, &st)
	if st.Rewarded != 1 || st.CreditsEarned != billing.ReferralCredits {
		t.Fatalf("after reward: %+v", st)
	}
	var said int
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM credit_ledger WHERE user_id=$1 AND reason='referral'`, a.id).Scan(&said); err != nil || said != 1 {
		t.Fatalf("ledger rows: %d, %v", said, err)
	}

	// A code that doesn't exist doesn't stop anyone, and earns nobody anything.
	c := e.signupWithRef(t, "Cy", "cy@example.com", "NOPE1234")
	if err := workflows.RewardReferrer(ctx, c.id); err != nil {
		t.Fatal(err)
	}

	// Nobody can invite themselves.
	if err := e.billing.Attribute(ctx, a.id, st.Code); err != nil {
		t.Fatal(err)
	}
	if err := workflows.RewardReferrer(ctx, a.id); err != nil {
		t.Fatal(err)
	}
	if got := balance(a.id); got != billing.ReferralCredits {
		t.Fatalf("balance after self-referral: %d", got)
	}

	// Past the monthly cap the invitation still counts, without credits.
	for i := 1; i < billing.ReferralMonthlyCap; i++ {
		f := e.signupWithRef(t, "F", fmt.Sprintf("f%d@example.com", i), st.Code)
		if err := workflows.RewardReferrer(ctx, f.id); err != nil {
			t.Fatal(err)
		}
	}
	want := billing.ReferralCredits * billing.ReferralMonthlyCap
	if got := balance(a.id); got != want {
		t.Fatalf("at the cap: %d, want %d", got, want)
	}
	over := e.signupWithRef(t, "G", "g@example.com", st.Code)
	if err := workflows.RewardReferrer(ctx, over.id); err != nil {
		t.Fatal(err)
	}
	if got := balance(a.id); got != want {
		t.Fatalf("past the cap: %d, want %d", got, want)
	}
}
