package e2e

import (
	"context"
	"testing"
	"time"
)

// Someone with a finished profile sees what Suki thinks they're looking for. A line only reaches
// them if its words are really theirs and the check agrees, and what they take out stays out.
func TestSukiShowsWhatSheThinksTheyAreLookingFor(t *testing.T) {
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
	a.askSuki("I'd never date a smoker, and my dog Rex comes first. We had wine once.")
	exec(`UPDATE profiles SET status='active'`)
	exec(`UPDATE users SET created_at=now()-interval '1 day'`)

	type read struct {
		Read []struct {
			ID, Kind, Text, Said string
		} `json:"read"`
	}
	get := func() read {
		t.Helper()
		var r read
		a.do("GET", "/v1/looking-for", nil, 200, &r)
		return r
	}

	// If she doesn't actually show the card, she doesn't say anything, and it stays to do.
	const later = `UPDATE agent_turns SET created_at=created_at-interval '11 minutes'`
	exec(later)
	e.fake.skipLooking.Store(true)
	before := a.countSuki()
	a.do("POST", "/v1/agent/opened", map[string]any{}, 200, nil)
	eventually(t, "the turn to finish", func() bool {
		var n int
		_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM agent_turns WHERE intent='sys_looking_for' AND status='done'`).Scan(&n)
		return n == 1
	})
	if n := a.countSuki(); n != before {
		t.Fatalf("she said %d things with no card", n-before)
	}
	e.fake.skipLooking.Store(false)

	exec(later)
	before = a.countSuki()
	a.do("POST", "/v1/agent/opened", map[string]any{}, 200, nil)
	eventually(t, "Suki to show the card", func() bool { return a.countSuki() > before })
	_, meta := a.lastSuki()
	cards, _ := meta["cards"].([]any)
	if len(cards) != 1 || cards[0].(map[string]any)["kind"] != "looking_for" {
		t.Fatalf("cards = %v", meta["cards"])
	}

	r := get()
	if len(r.Read) != 2 || r.Read[0].Kind != "toward" || r.Read[0].Said != "my dog Rex" || r.Read[1].Kind != "away" || r.Read[1].Said != "never date a smoker" {
		t.Fatalf("read = %+v", r.Read)
	}

	// Once shown, opening the app goes back to her usual hello.
	exec(later)
	a.do("POST", "/v1/agent/opened", map[string]any{}, 200, nil)
	eventually(t, "her usual hello", func() bool {
		var n int
		_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM agent_turns WHERE intent='sys_app_open' AND status='done'`).Scan(&n)
		return n == 1
	})

	// Taken out, and not written back when she next rereads them.
	a.do("DELETE", "/v1/looking-for/read/"+r.Read[1].ID, nil, 200, nil)
	if got := get(); len(got.Read) != 1 || got.Read[0].Kind != "toward" {
		t.Fatalf("after removing: %+v", got.Read)
	}
	exec(`UPDATE looking_reads SET shown_at=NULL`)
	exec(later)
	done := func() int {
		var n int
		_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM agent_turns WHERE intent='sys_looking_for' AND status='done'`).Scan(&n)
		return n
	}
	was := done()
	a.do("POST", "/v1/agent/opened", map[string]any{}, 200, nil)
	eventually(t, "her second read", func() bool { return done() > was })
	time.Sleep(50 * time.Millisecond)
	for _, l := range get().Read {
		if l.Kind == "away" {
			t.Fatalf("a line they removed came back: %+v", l)
		}
	}

	// Nothing she worked out touches their stated dealbreakers.
	var flags int
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM profiles, jsonb_object_keys(dealbreakers) WHERE true`).Scan(&flags); err != nil {
		t.Fatal(err)
	}
	if flags != 0 {
		t.Fatalf("her read wrote %d dealbreakers", flags)
	}
}
