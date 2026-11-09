package e2e

import (
	"context"
	"strings"
	"testing"
)

// Suki says something each time the app is opened, free of charge. Coming straight back is the
// same visit, and someone who asked her not to start things is left alone.
func TestSukiSpeaksWhenTheAppOpens(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()
	exec := func(q string) {
		t.Helper()
		if _, err := e.pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	turns := func() int {
		var n int
		if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM agent_turns WHERE intent='sys_app_open'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	const later = `UPDATE agent_turns SET created_at=created_at-interval '11 minutes'`

	// She has just welcomed someone who only now signed up.
	a.do("POST", "/v1/agent/opened", map[string]any{}, 200, nil)
	if n := turns(); n != 0 {
		t.Fatalf("%d turns right after signing up, want 0", n)
	}
	exec(`UPDATE users SET created_at=now()-interval '1 day'`)
	exec(later)

	before := a.countSuki()
	a.do("POST", "/v1/agent/opened", map[string]any{}, 200, nil)
	eventually(t, "Suki to say something", func() bool { return a.countSuki() > before })
	if body, _ := a.lastSuki(); !strings.Contains(body, "how's your week") {
		t.Fatalf("she said %q", body)
	}
	var charged int
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM credit_ledger`).Scan(&charged); err != nil {
		t.Fatal(err)
	}
	if charged != 0 {
		t.Fatalf("opening the app touched the ledger %d times", charged)
	}

	// Back within minutes: the same visit.
	a.do("POST", "/v1/agent/opened", map[string]any{}, 200, nil)
	if n := turns(); n != 1 {
		t.Fatalf("%d turns after coming straight back, want 1", n)
	}

	// Opening it again later is another visit.
	exec(later)
	before = a.countSuki()
	a.do("POST", "/v1/agent/opened", map[string]any{}, 200, nil)
	eventually(t, "Suki to speak again", func() bool { return a.countSuki() > before })

	// A message that fails its fact check never reaches them.
	exec(later)
	e.fake.inventOpening.Store(true)
	before = a.countSuki()
	a.do("POST", "/v1/agent/opened", map[string]any{}, 200, nil)
	eventually(t, "the turn to finish", func() bool {
		var done int
		_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM agent_turns WHERE intent='sys_app_open' AND status='done'`).Scan(&done)
		return done == 3
	})
	if n := a.countSuki(); n != before {
		t.Fatalf("a message that failed its check was sent: %d messages, want %d", n, before)
	}
	e.fake.inventOpening.Store(false)

	// Asked not to start things.
	exec(later)
	a.do("PATCH", "/v1/settings", map[string]any{"suki_proactive": "low"}, 200, nil)
	a.do("POST", "/v1/agent/opened", map[string]any{}, 200, nil)
	if n := turns(); n != 3 {
		t.Fatalf("%d turns with her set to low, want 3", n)
	}
}
