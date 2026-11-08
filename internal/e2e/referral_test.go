package e2e

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func (e *env) balance(t *testing.T, id string) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT coalesce((SELECT balance FROM credit_balances WHERE user_id=$1),0)`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (e *env) ledgerRows(t *testing.T, id, reason string) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM credit_ledger WHERE user_id=$1 AND reason=$2`, id, reason).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestReferralCredits(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	r := billing.DefaultRewards()
	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()
	var st billing.ReferralStatus
	a.do("GET", "/v1/referral", nil, 200, &st)
	if len(st.Code) != 8 || !strings.HasSuffix(st.Link, "/app/?ref="+st.Code) || st.CreditsPerFriend != r.InviterCredits || st.CreditsForFriend != r.InviteeCredits {
		t.Fatalf("status: %+v", st)
	}
	// The code is opaque: nothing in it is the person's id.
	if strings.Contains(strings.ToLower(a.id), strings.ToLower(st.Code)) {
		t.Fatalf("code %s looks like the user id", st.Code)
	}
	var again billing.ReferralStatus
	a.do("GET", "/v1/referral", nil, 200, &again)
	if again.Code != st.Code {
		t.Fatalf("the code changed: %s then %s", st.Code, again.Code)
	}

	// Lower case, as someone might type it. Signing up pays nothing yet, to either side.
	b := e.signupWithRef(t, "Bo", "bo@example.com", strings.ToLower(st.Code))
	a.do("GET", "/v1/referral", nil, 200, &st)
	if st.Joined != 1 || st.Rewarded != 0 || e.balance(t, a.id) != 0 || e.balance(t, b.id) != 0 {
		t.Fatalf("after sign-up: %+v, balances %d %d", st, e.balance(t, a.id), e.balance(t, b.id))
	}

	// An unfinished profile pays nothing.
	if err := workflows.AfterProfileChange(ctx, b.id); err != nil {
		t.Fatal(err)
	}
	if got := e.balance(t, a.id); got != 0 {
		t.Fatalf("paid for an unfinished profile: %d", got)
	}

	// Their profile going live pays both of them, and only once.
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
	if got := e.balance(t, a.id); got != r.InviterCredits {
		t.Fatalf("inviter balance after reward: %d", got)
	}
	if got := e.balance(t, b.id); got != r.InviteeCredits {
		t.Fatalf("invitee balance after reward: %d", got)
	}
	a.do("GET", "/v1/referral", nil, 200, &st)
	if st.Rewarded != 1 || st.CreditsEarned != r.InviterCredits {
		t.Fatalf("after reward: %+v", st)
	}
	// Earned credits are told apart from bought ones by their reason.
	if n := e.ledgerRows(t, a.id, "referral"); n != 1 {
		t.Fatalf("inviter ledger rows: %d", n)
	}
	if n := e.ledgerRows(t, b.id, "invited"); n != 1 {
		t.Fatalf("invitee ledger rows: %d", n)
	}
	if n := e.ledgerRows(t, b.id, "purchase") + e.ledgerRows(t, a.id, "purchase"); n != 0 {
		t.Fatalf("earned credits recorded as purchases: %d", n)
	}
	var usage struct {
		Usage []billing.UsageRow `json:"usage"`
	}
	b.verify()
	b.do("GET", "/v1/credits/usage", nil, 200, &usage)
	if len(usage.Usage) != 1 || usage.Usage[0].Reason != "invited" {
		t.Fatalf("invitee usage: %+v", usage.Usage)
	}

	// A code that doesn't exist doesn't stop anyone, and earns nobody anything.
	c := e.signupWithRef(t, "Cy", "cy@example.com", "NOPE1234")
	if err := workflows.RewardReferrer(ctx, c.id); err != nil {
		t.Fatal(err)
	}
	if got := e.balance(t, c.id); got != 0 {
		t.Fatalf("balance with a made-up code: %d", got)
	}

	// Nobody can invite themselves.
	if err := e.billing.Attribute(ctx, a.id, st.Code); err != nil {
		t.Fatal(err)
	}
	if err := workflows.RewardReferrer(ctx, a.id); err != nil {
		t.Fatal(err)
	}
	if got := e.balance(t, a.id); got != r.InviterCredits {
		t.Fatalf("balance after self-referral: %d", got)
	}
}

// An account that already existed can't be claimed by an invite, whether by signing in
// with a code or by being attributed later.
func TestReferralExistingAccountEarnsNothing(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	inviter, code := e.inviterCode(t, "inv@example.com")
	old := e.signup(t, "Old", "old@example.com")
	old.verify()

	// Signing in again with a code is not a new account.
	old.do("POST", "/v1/auth/signin", map[string]any{"email": "old@example.com", "password": "correct horse battery", "ref": code}, 200, nil)
	// Even if something tried to attribute it, an account older than moments ago is refused.
	if _, err := e.pool.Exec(ctx, `UPDATE users SET created_at=now() - interval '2 days' WHERE id=$1`, old.id); err != nil {
		t.Fatal(err)
	}
	if err := e.billing.Attribute(ctx, old.id, code); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM referrals WHERE invitee_id=$1`, old.id).Scan(&n); err != nil || n != 0 {
		t.Fatalf("existing account attributed: %d, %v", n, err)
	}
	if err := workflows.RewardReferrer(ctx, old.id); err != nil {
		t.Fatal(err)
	}
	if e.balance(t, inviter) != 0 || e.balance(t, old.id) != 0 {
		t.Fatalf("existing account paid: %d %d", e.balance(t, inviter), e.balance(t, old.id))
	}

	// A new account can only ever name one inviter: the first wins.
	_, other := e.inviterCode(t, "other@example.com")
	fresh := e.signupWithRef(t, "Fresh", "fresh@example.com", code)
	if err := e.billing.Attribute(ctx, fresh.id, other); err != nil {
		t.Fatal(err)
	}
	if got := e.referrerOf(t, fresh.id); got != inviter {
		t.Fatalf("a second code took over the invitation: %s", got)
	}
}

// Paying the same friend from many places at once pays each side exactly once.
func TestReferralRewardIsIdempotentUnderConcurrency(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	r := billing.DefaultRewards()
	inviter, code := e.inviterCode(t, "inv@example.com")
	b := e.signupWithRef(t, "Bo", "bo@example.com", code)
	var wg sync.WaitGroup
	var paid atomic.Int32
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := e.billing.RewardReferral(ctx, b.id)
			if err != nil {
				t.Error(err)
			}
			if got != nil {
				paid.Add(1)
			}
		}()
	}
	wg.Wait()
	if paid.Load() != 1 || e.balance(t, inviter) != r.InviterCredits || e.balance(t, b.id) != r.InviteeCredits {
		t.Fatalf("paid %d times, balances %d %d", paid.Load(), e.balance(t, inviter), e.balance(t, b.id))
	}
}

// The inviter is paid for at most InviteDailyCap friends per China day and InviteTotalCap
// ever. A friend past a cap still counts as joined, and is settled unpaid for good.
func TestReferralCaps(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	rw := billing.Rewards{InviterCredits: 30, InviteeCredits: 30, InviteDailyCap: 2, InviteTotalCap: 3}
	e.billing.Rewards = &rw
	// 23:00 in Shanghai on 8 November.
	var mu sync.Mutex
	clock := time.Date(2026, 11, 8, 15, 0, 0, 0, time.UTC)
	setClock := func(t time.Time) { mu.Lock(); clock = t; mu.Unlock() }
	e.billing.Now = func() time.Time { mu.Lock(); defer mu.Unlock(); return clock }
	t.Cleanup(func() { e.billing.Rewards, e.billing.Now = nil, nil })

	inviter, code := e.inviterCode(t, "inv@example.com")
	friends := make([]*client, 5)
	for i := range friends {
		friends[i] = e.signupWithRef(t, "F", fmt.Sprintf("f%d@example.com", i), code)
	}

	// Four friends go live at once on day one: the daily cap holds under concurrency.
	var wg sync.WaitGroup
	for _, f := range friends[:4] {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if _, err := e.billing.RewardReferral(ctx, id); err != nil {
				t.Error(err)
			}
		}(f.id)
	}
	wg.Wait()
	if got := e.balance(t, inviter); got != 2*rw.InviterCredits {
		t.Fatalf("day one, inviter: %d, want %d", got, 2*rw.InviterCredits)
	}
	paidFriends := 0
	for _, f := range friends[:4] {
		switch e.balance(t, f.id) {
		case rw.InviteeCredits:
			paidFriends++
		case 0:
		default:
			t.Fatalf("friend balance %d", e.balance(t, f.id))
		}
	}
	if paidFriends != 2 {
		t.Fatalf("friends paid on day one: %d", paidFriends)
	}

	// Past midnight in Shanghai (16:00 UTC) is a new day. Friends capped yesterday stay unpaid.
	setClock(time.Date(2026, 11, 8, 16, 0, 1, 0, time.UTC))
	for _, f := range friends[:4] {
		if _, err := e.billing.RewardReferral(ctx, f.id); err != nil {
			t.Fatal(err)
		}
	}
	if got := e.balance(t, inviter); got != 2*rw.InviterCredits {
		t.Fatalf("a capped friend was paid later: %d", got)
	}
	// A new friend today is paid: three in total, the lifetime cap.
	if _, err := e.billing.RewardReferral(ctx, friends[4].id); err != nil {
		t.Fatal(err)
	}
	if got := e.balance(t, inviter); got != 3*rw.InviterCredits {
		t.Fatalf("day two: %d", got)
	}
	// Past the lifetime cap nobody is paid, even on a fresh day, and the friend still counts.
	setClock(time.Date(2026, 11, 11, 4, 0, 0, 0, time.UTC))
	late := e.signupWithRef(t, "Late", "late@example.com", code)
	if _, err := e.billing.RewardReferral(ctx, late.id); err != nil {
		t.Fatal(err)
	}
	if e.balance(t, inviter) != 3*rw.InviterCredits || e.balance(t, late.id) != 0 {
		t.Fatalf("past the total cap: %d %d", e.balance(t, inviter), e.balance(t, late.id))
	}
	st, err := e.billing.ReferralStatus(ctx, inviter)
	if err != nil {
		t.Fatal(err)
	}
	if st.Joined != 6 || st.Rewarded != 3 || st.CreditsEarned != 3*rw.InviterCredits {
		t.Fatalf("status: %+v", st)
	}
}
