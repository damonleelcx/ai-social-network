package e2e

import (
	"context"
	"testing"

	"github.com/damonleelcx/ai-social-network/internal/chatstore"
	"github.com/damonleelcx/ai-social-network/internal/dating"
)

type buddyLists struct {
	Requests []dating.BuddyRequest `json:"requests"`
	Missing  []string              `json:"missing"`
}

// A plan from post to chat: tonight's movie, without any dating preferences at all.
func TestBuddyPlanBecomesAChat(t *testing.T) {
	e := setup(t)
	a := e.signup(t, "Ada", "ada@example.com")
	b := e.signup(t, "Bo", "bo@example.com")
	c := e.signup(t, "Cy", "cy@example.com")
	a.verify()
	b.verify()
	c.verify()

	movie := map[string]any{"kind": "movie", "when": "Tonight after 7", "note": "New sci-fi film, I'd rather not go alone.", "expires_in_hours": 12}

	// Without a city or a photo you can neither post nor look.
	a.do("POST", "/v1/buddies", movie, 422, nil)
	var board buddyLists
	b.do("GET", "/v1/buddies", nil, 200, &board)
	if len(board.Missing) == 0 || len(board.Requests) != 0 {
		t.Fatalf("an unfinished profile saw the board: %+v", board)
	}

	// Only what the plan needs: no seeking, goal or interests.
	for i, x := range []*client{a, b, c} {
		x.do("PATCH", "/v1/profile", map[string]any{"birthdate": "1994-05-01"}, 200, nil)
		x.do("POST", "/v1/profile/city", map[string]any{"query": "Testville"}, 200, nil)
		x.do("POST", "/v1/photos", photo(uint8(40+i*10)), 201, nil)
	}
	eventually(t, "photos to be approved", func() bool {
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM photos WHERE status='approved'`).Scan(&n)
		return n >= 3
	})

	var posted struct {
		Request dating.BuddyRequest `json:"request"`
	}
	a.do("POST", "/v1/buddies", movie, 201, &posted)
	id := posted.Request.ID

	// A contact detail in a post is refused, because the whole city reads it.
	a.do("POST", "/v1/buddies", map[string]any{"kind": "meal", "when": "Friday", "note": "add me on whatsapp", "expires_in_hours": 12}, 422, nil)

	b.do("GET", "/v1/buddies", nil, 200, &board)
	if len(board.Requests) != 1 || board.Requests[0].ID != id || board.Requests[0].Person == nil || board.Requests[0].Person.DisplayName != "Ada" {
		t.Fatalf("board: %+v", board)
	}
	a.do("GET", "/v1/buddies", nil, 200, &board)
	if len(board.Requests) != 0 {
		t.Fatal("your own post shouldn't be on your board")
	}

	// You can't see someone's photo until they're in front of you, and then you can.
	photoID := func(x *client) string {
		var ph struct {
			ID string
		}
		_ = e.pool.QueryRow(context.Background(), `SELECT id FROM photos WHERE user_id=(SELECT id FROM users WHERE email=$1)`, x.email).Scan(&ph.ID)
		return ph.ID
	}
	b.do("GET", "/v1/media/"+photoID(a), nil, 200, nil)

	// Both say they're in, and a stranger's pick goes nowhere.
	b.do("POST", "/v1/buddies/"+id+"/offer", map[string]any{}, 200, nil)
	c.do("POST", "/v1/buddies/"+id+"/offer", map[string]any{}, 200, nil)
	c.do("DELETE", "/v1/buddies/"+id+"/offer", nil, 200, nil)
	c.do("POST", "/v1/buddies/"+id+"/offer", map[string]any{}, 200, nil)

	var mine buddyLists
	a.do("GET", "/v1/buddies/mine", nil, 200, &mine)
	if len(mine.Requests) != 1 || len(mine.Requests[0].Offers) != 2 {
		t.Fatalf("mine: %+v", mine)
	}
	bID := mine.Requests[0].Offers[0].Person.UserID

	// It all happens in the conversation: people hear from Suki, with a card to tap.
	sukiCard := func(x *client, kind string) bool {
		var r struct {
			Messages []chatstore.Message `json:"messages"`
		}
		x.do("GET", "/v1/agent/messages", nil, 200, &r)
		for _, m := range r.Messages {
			if cards, ok := m.Meta["cards"].([]any); ok {
				for _, c := range cards {
					if c.(map[string]any)["kind"] == kind {
						return true
					}
				}
			}
		}
		return false
	}
	if !sukiCard(a, "buddy_offer") {
		t.Fatal("the poster didn't hear about the offer from Suki")
	}
	if !sukiCard(b, "buddy_request") {
		t.Fatal("someone nearby didn't hear about the post from Suki")
	}
	var one struct {
		Request dating.BuddyRequest `json:"request"`
	}
	b.do("GET", "/v1/buddies/"+id, nil, 200, &one)
	if !one.Request.Offered || one.Request.Person == nil {
		t.Fatalf("card view: %+v", one.Request)
	}

	b.do("POST", "/v1/buddies/"+id+"/pick", map[string]any{"user_id": bID}, 404, nil)
	a.do("POST", "/v1/buddies/"+id+"/pick", map[string]any{"user_id": a.id}, 404, nil)

	var picked struct {
		MatchID string `json:"match_id"`
	}
	a.do("POST", "/v1/buddies/"+id+"/pick", map[string]any{"user_id": bID}, 200, &picked)
	if picked.MatchID == "" {
		t.Fatal("no match")
	}

	// It's an ordinary match, marked as a plan, and the chat works from the first message.
	var m struct {
		Match dating.Match `json:"match"`
	}
	b.do("GET", "/v1/matches/"+picked.MatchID, nil, 200, &m)
	if m.Match.Activity != "movie" || m.Match.Person.DisplayName != "Ada" {
		t.Fatalf("match: %+v", m.Match)
	}
	b.do("POST", "/v1/matches/"+picked.MatchID+"/messages", map[string]any{"text": "Which cinema?", "client_id": "1"}, 201, nil)

	// The post has closed for the person who wasn't picked, and a plan teaches nothing about taste.
	c.do("GET", "/v1/buddies", nil, 200, &board)
	if len(board.Requests) != 0 {
		t.Fatalf("a filled post is still up: %+v", board)
	}
	c.do("POST", "/v1/buddies/"+id+"/offer", map[string]any{}, 404, nil)
	var learned int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM matches WHERE id=$1 AND coached_at_count=0`, picked.MatchID).Scan(&learned)
	if learned != 1 {
		t.Fatal("a plan was coached like a romance")
	}

	// A block takes the other person's posts off the board, and their offers off yours.
	var second struct {
		Request dating.BuddyRequest `json:"request"`
	}
	a.do("POST", "/v1/buddies", map[string]any{"kind": "travel", "when": "Next month", "expires_in_hours": 300}, 201, &second)
	c.do("GET", "/v1/buddies", nil, 200, &board)
	if len(board.Requests) != 1 {
		t.Fatalf("board before the block: %+v", board)
	}
	c.do("POST", "/v1/blocks", map[string]any{"user_id": a.id}, 200, nil)
	c.do("GET", "/v1/buddies", nil, 200, &board)
	if len(board.Requests) != 0 {
		t.Fatalf("a blocked person's post is still visible: %+v", board)
	}
	c.do("POST", "/v1/buddies/"+second.Request.ID+"/offer", map[string]any{}, 404, nil)

	// Posting stops at three open plans.
	a.do("POST", "/v1/buddies", map[string]any{"kind": "study", "when": "Sunday", "expires_in_hours": 48}, 201, nil)
	a.do("POST", "/v1/buddies", map[string]any{"kind": "games", "when": "Sunday", "expires_in_hours": 48}, 201, nil)
	a.do("POST", "/v1/buddies", map[string]any{"kind": "sports", "when": "Sunday", "expires_in_hours": 48}, 422, nil)

	// Closing takes it off the board.
	a.do("POST", "/v1/buddies/"+second.Request.ID+"/close", map[string]any{}, 200, nil)
	b.do("GET", "/v1/buddies", nil, 200, &board)
	for _, r := range board.Requests {
		if r.ID == second.Request.ID {
			t.Fatal("a closed post is still up")
		}
	}
}
