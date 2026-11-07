package e2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/damonleelcx/ai-social-network/internal/hidden"
)

// The whole loop: what someone accepts and passes on through the app changes who the
// next search ranks first, and more so the more they show it.
func TestSearchesLearnWhoSomeoneLikes(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.fake.verdict.Store("rejected") // Nobody is introduced by a search here. Only the ranking is under test.
	outdoors := []string{"hiking", "camping", "climbing"}
	arts := []string{"painting", "museums", "pottery"}
	person := func(name string, interests []string, shade uint8) *client {
		t.Helper()
		c := e.signup(t, name, name+"@example.com")
		c.verify()
		c.complete("man", []string{"woman"}, "1992-02-02", interests, shade)
		return c
	}
	active := func(c *client) {
		t.Helper()
		eventually(t, "their profile to go live", func() bool {
			var status string
			_ = e.pool.QueryRow(ctx, `SELECT status FROM profiles WHERE user_id=$1`, c.id).Scan(&status)
			return status == "active"
		})
	}
	hiker := person("hiker", outdoors, 10)
	artist := person("artist", arts, 20)
	active(hiker)
	active(artist)
	ana := e.signup(t, "Ana", "ana@example.com")
	ana.verify()
	ana.complete("woman", []string{"man"}, "1992-02-02", []string{"ramen", "coffee", "tea"}, 30)
	active(ana)

	// How far ahead the next search puts the hiker, using Ana's real ranking.
	var goal string
	eventually(t, "Ana to have a search", func() bool {
		return e.pool.QueryRow(ctx, `SELECT id FROM goals WHERE user_id=$1 AND skill='match-search' LIMIT 1`, ana.id).Scan(&goal) == nil
	})
	lead := func() float64 {
		t.Helper()
		if _, err := hidden.Gather(ctx, e.pool, goal, ana.id, 20); err != nil {
			t.Fatal(err)
		}
		pack, err := hidden.GetPack(ctx, e.pool, goal)
		if err != nil {
			t.Fatal(err)
		}
		priors := map[string]float64{}
		for _, c := range pack.Candidates {
			priors[c.UserID] = c.Prior
		}
		h, ok1 := priors[hiker.id]
		a, ok2 := priors[artist.id]
		if !ok1 || !ok2 {
			t.Fatalf("both should be candidates: %v", priors)
		}
		return h - a
	}
	decide := func(c *client, decision string) {
		t.Helper()
		var id string
		if err := e.pool.QueryRow(ctx, `INSERT INTO proposals (user_a,user_b,status,pitch_a,pitch_b) VALUES (least($1::uuid,$2::uuid),greatest($1::uuid,$2::uuid),'open','hello','hello') RETURNING id`, ana.id, c.id).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ana.do("POST", "/v1/proposals/"+id+"/decide", map[string]any{"decision": decision}, 200, nil)
	}
	teach := func(round int) {
		t.Helper()
		o := person(fmt.Sprintf("out%d", round), outdoors, uint8(40+round))
		p := person(fmt.Sprintf("art%d", round), arts, uint8(60+round))
		active(o)
		active(p)
		var before int
		_ = e.pool.QueryRow(ctx, `SELECT coalesce((SELECT signals FROM hidden.taste WHERE user_id=$1),0)`, ana.id).Scan(&before)
		decide(o, "accept")
		decide(p, "pass")
		eventually(t, "her choices to teach the model", func() bool {
			var n int
			_ = e.pool.QueryRow(ctx, `SELECT coalesce((SELECT signals FROM hidden.taste WHERE user_id=$1),0)`, ana.id).Scan(&n)
			return n >= before+2
		})
	}

	start := lead()
	teach(1)
	first := lead()
	teach(2)
	second := lead()
	if !(start < first && first < second) {
		t.Fatalf("the hiker's lead should grow as she shows her type: %v, %v, %v", start, first, second)
	}
	if second <= 0 {
		t.Fatalf("the hiker should now rank first, lead %v", second)
	}
}
