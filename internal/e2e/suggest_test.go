package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/damonleelcx/ai-social-network/internal/workflows"
)

// Suki sends suggestions for free, even to someone with no credits left, and it's their
// answer that costs a credit.
func TestSukiSuggestsSomethingToDo(t *testing.T) {
	e := setup(t)
	e.billing.StripeKey, e.billing.WebhookSecret = "sk_test_x", testWebhookSecret
	ctx := context.Background()

	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()
	quiet := e.signup(t, "Quiet", "quiet@example.com")
	quiet.verify()
	var aID, quietID string
	for email, id := range map[string]*string{"ada@example.com": &aID, "quiet@example.com": &quietID} {
		if err := e.pool.QueryRow(ctx, `SELECT id FROM users WHERE email=$1`, email).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	quiet.do("PATCH", "/v1/settings", map[string]any{"suki_proactive": "low"}, 200, nil)

	// Ada has used her free match and has no credits.
	var pid string
	if err := e.pool.QueryRow(ctx, `INSERT INTO proposals (user_a,user_b,status) VALUES (least($1::uuid,$2::uuid),greatest($1::uuid,$2::uuid),'matched') RETURNING id`, aID, quietID).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO matches (proposal_id,user_a,user_b,status) VALUES ($1,least($2::uuid,$3::uuid),greatest($2::uuid,$3::uuid),'unmatched')`, pid, aID, quietID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `UPDATE profiles SET status='active', last_active_at=now()`); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `UPDATE agent_turns SET created_at=now()-interval '2 hours'`); err != nil {
		t.Fatal(err)
	}
	count := func(q, id string) int {
		var n int
		if err := e.pool.QueryRow(ctx, q, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	const turns = `SELECT count(*) FROM agent_turns WHERE user_id=$1 AND intent='sys_suggest_next'`

	workflows.Cron(ctx)
	eventually(t, "Suki's suggestion", func() bool {
		b, _ := a.lastSuki()
		return strings.Contains(b, "say hi to someone")
	})
	if n := count(`SELECT count(*) FROM notifications WHERE user_id=$1 AND key LIKE 'suggest:%'`, aID); n != 1 {
		t.Fatalf("%d notifications, want 1", n)
	}
	if n := count(`SELECT count(*) FROM credit_ledger WHERE user_id=$1`, aID); n != 0 {
		t.Fatalf("a suggestion touched the ledger %d times", n)
	}
	if n := count(turns, quietID); n != 0 {
		t.Fatalf("low proactivity got %d suggestions", n)
	}

	// Not again the same day, even once the interruption check has passed.
	if _, err := e.pool.Exec(ctx, `UPDATE agent_turns SET created_at=now()-interval '2 hours'`); err != nil {
		t.Fatal(err)
	}
	workflows.Cron(ctx)
	time.Sleep(200 * time.Millisecond)
	if n := count(turns, aID); n != 1 {
		t.Fatalf("%d suggestions in a day, want 1", n)
	}

	// A day later, but they've just spoken to her: their answer is charged, and she doesn't
	// interrupt them with another suggestion.
	if _, err := e.pool.Exec(ctx, `UPDATE agent_turns SET created_at=now()-interval '1 day'`); err != nil {
		t.Fatal(err)
	}
	if body, _ := a.askSuki("good idea"); !strings.Contains(body, "out of credits") {
		t.Fatalf("answering should cost a credit: %q", body)
	}
	workflows.Cron(ctx)
	time.Sleep(200 * time.Millisecond)
	if n := count(turns, aID); n != 1 {
		t.Fatalf("interrupted a conversation: %d suggestions", n)
	}
}
