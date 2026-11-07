package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/damonleelcx/ai-social-network/internal/chatstore"
	"github.com/damonleelcx/ai-social-network/internal/dating"
)

// A group plan and a trip, end to end: people join a group at once up to its spots, talk in
// one group chat, and someone visiting from another city finds the plan through their trip.
func TestGroupPlansAndTrips(t *testing.T) {
	e := setup(t)
	a := e.signup(t, "Ada", "ada@example.com")
	b := e.signup(t, "Bo", "bo@example.com")
	c := e.signup(t, "Cy", "cy@example.com")
	d := e.signup(t, "Di", "di@example.com")
	for i, x := range []*client{a, b, c, d} {
		x.verify()
		x.do("PATCH", "/v1/profile", map[string]any{"birthdate": "1994-05-01"}, 200, nil)
		city := "Testville"
		if x == d {
			city = "Farport"
		}
		x.do("POST", "/v1/profile/city", map[string]any{"query": city}, 200, nil)
		x.do("POST", "/v1/photos", photo(uint8(40+i*10)), 201, nil)
	}
	eventually(t, "photos to be approved", func() bool {
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM photos WHERE status='approved'`).Scan(&n)
		return n >= 4
	})

	var posted struct {
		Request dating.BuddyRequest `json:"request"`
	}
	hike := map[string]any{"kind": "sports", "when": "Saturday morning", "note": "Easy hike, then coffee.", "expires_in_hours": 72, "spots": 2, "place": "River Park"}
	a.do("POST", "/v1/buddies", map[string]any{"kind": "sports", "when": "Saturday", "expires_in_hours": 72, "spots": 21}, 422, nil)
	a.do("POST", "/v1/buddies", hike, 201, &posted)
	id := posted.Request.ID
	if !posted.Request.Group() || posted.Request.Going != 1 || posted.Request.Place != "River Park" {
		t.Fatalf("posted: %+v", posted.Request)
	}

	// Di lives elsewhere and can't see it.
	var board buddyLists
	d.do("GET", "/v1/buddies", nil, 200, &board)
	if len(board.Requests) != 0 {
		t.Fatalf("another city saw the plan: %+v", board.Requests)
	}
	d.do("GET", "/v1/groups/"+id+"/messages", nil, 404, nil)

	// Joining is at once: nobody picks. Twice is once.
	b.do("POST", "/v1/buddies/"+id+"/offer", map[string]any{}, 200, nil)
	b.do("POST", "/v1/buddies/"+id+"/offer", map[string]any{}, 200, nil)
	c.do("GET", "/v1/buddies", nil, 200, &board)
	if len(board.Requests) != 1 || board.Requests[0].Going != 2 || board.Requests[0].Spots != 2 {
		t.Fatalf("board: %+v", board.Requests)
	}
	c.do("POST", "/v1/buddies/"+id+"/offer", map[string]any{}, 200, nil)

	// Now it's full, so it's off the board, and everyone in it sees everyone.
	var one struct {
		Request dating.BuddyRequest `json:"request"`
	}
	b.do("GET", "/v1/buddies/"+id, nil, 200, &one)
	if one.Request.Status != "filled" || !one.Request.Joined || len(one.Request.Members) != 3 || one.Request.Going != 3 {
		t.Fatalf("member view: %+v", one.Request)
	}
	if !sukiCardOf(b, "group") || !sukiCardOf(a, "group") {
		t.Fatal("Suki didn't bring the group chat card")
	}

	// The group chat.
	b.do("POST", "/v1/groups/"+id+"/messages", map[string]any{"text": "Meet at the gate at 9?", "client_id": "1"}, 201, nil)
	b.do("POST", "/v1/groups/"+id+"/messages", map[string]any{"text": "Meet at the gate at 9?", "client_id": "1"}, 201, nil)
	var chat dating.GroupChat
	a.do("GET", "/v1/groups/"+id+"/messages", nil, 200, &chat)
	texts, joins := 0, 0
	for _, m := range chat.Messages {
		switch m.Role {
		case "text":
			texts++
			if m.Meta["name"] != "Bo" {
				t.Fatalf("no sender name: %+v", m)
			}
		case "system":
			joins++
		}
	}
	if texts != 1 || joins != 2 || !chat.Open {
		t.Fatalf("chat: %d texts, %d joins, open %v: %+v", texts, joins, chat.Open, chat.Messages)
	}

	// Di's trip to Testville. The dates and the city are checked.
	today := time.Now().UTC()
	day := func(n int) string { return today.AddDate(0, 0, n).Format("2006-01-02") }
	d.do("POST", "/v1/trips", map[string]any{"city": "Testville", "starts_on": day(3), "ends_on": day(1)}, 422, nil)
	d.do("POST", "/v1/trips", map[string]any{"city": "Farport", "starts_on": day(0), "ends_on": day(3)}, 422, nil)
	var trip struct {
		Trip dating.Trip `json:"trip"`
	}
	d.do("POST", "/v1/trips", map[string]any{"city": "Testville", "starts_on": day(0), "ends_on": day(5), "note": "First time here!"}, 201, &trip)
	if trip.Trip.CityName == "" || trip.Trip.Going != 0 {
		t.Fatalf("trip: %+v", trip.Trip)
	}

	// The group is full, so Di still sees nothing. Once Bo leaves, there's room, and Di finds it.
	d.do("GET", "/v1/buddies", nil, 200, &board)
	if len(board.Requests) != 0 {
		t.Fatal("a full group was on the board")
	}
	b.do("DELETE", "/v1/buddies/"+id+"/offer", nil, 200, nil)
	b.do("GET", "/v1/groups/"+id+"/messages", nil, 404, nil)
	d.do("GET", "/v1/trips/"+trip.Trip.ID, nil, 200, &trip)
	if trip.Trip.Plans != 1 {
		t.Fatalf("plans on the trip: %+v", trip.Trip)
	}
	d.do("POST", "/v1/buddies/"+id+"/offer", map[string]any{}, 200, nil)
	d.do("GET", "/v1/groups/"+id+"/messages", nil, 200, &chat)

	// Ada and Cy both go to Farport, over days that overlap. Each sees a count, never a name,
	// and only other travellers count.
	var trips struct {
		Trips []dating.Trip `json:"trips"`
	}
	a.do("POST", "/v1/trips", map[string]any{"city": "Farport", "starts_on": day(1), "ends_on": day(2)}, 201, nil)
	c.do("POST", "/v1/trips", map[string]any{"city": "Farport", "starts_on": day(2), "ends_on": day(9)}, 201, nil)
	a.do("GET", "/v1/trips", nil, 200, &trips)
	if len(trips.Trips) != 1 || trips.Trips[0].Going != 1 {
		t.Fatalf("Ada's trips: %+v", trips.Trips)
	}
	var trending struct {
		Cities []dating.TripCity `json:"cities"`
	}
	a.do("GET", "/v1/trips/trending", nil, 200, &trending)
	if len(trending.Cities) != 2 || trending.Cities[0].Going != 2 {
		t.Fatalf("trending: %+v", trending.Cities)
	}
	a.do("DELETE", "/v1/trips/"+trips.Trips[0].ID, nil, 200, nil)
	b.do("DELETE", "/v1/trips/"+trip.Trip.ID, nil, 404, nil)

	// Calling it off closes the chat for everyone in it, and Suki tells them.
	a.do("POST", "/v1/buddies/"+id+"/close", map[string]any{}, 200, nil)
	c.do("POST", "/v1/groups/"+id+"/messages", map[string]any{"text": "Oh no", "client_id": "2"}, 422, nil)
	c.do("GET", "/v1/groups/"+id+"/messages", nil, 200, &chat)
	if chat.Open {
		t.Fatal("a called-off group's chat is still open")
	}
	if !sukiSaid(c, "called off") {
		t.Fatal("Suki didn't tell Cy the plan was called off")
	}
}

func sukiMessages(x *client) []chatstore.Message {
	var r struct {
		Messages []chatstore.Message `json:"messages"`
	}
	x.do("GET", "/v1/agent/messages", nil, 200, &r)
	return r.Messages
}

// sukiCardOf says whether Suki has brought this person a card of this kind.
func sukiCardOf(x *client, kind string) bool {
	for _, m := range sukiMessages(x) {
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

func sukiSaid(x *client, words string) bool {
	for _, m := range sukiMessages(x) {
		if strings.Contains(m.Body, words) {
			return true
		}
	}
	return false
}
