package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/damonleelcx/ai-social-network/internal/dating"
)

// A plan gets a picture of what it's about, drawn once and shared: two hotpot plans in the same
// city show the same picture, and the model draws it only once. Trips get their city's picture.
func TestPlansGetASharedScenePicture(t *testing.T) {
	e := setup(t)
	a := e.signup(t, "Ada", "ada@example.com")
	b := e.signup(t, "Bo", "bo@example.com")
	for i, x := range []*client{a, b} {
		x.verify()
		x.do("PATCH", "/v1/profile", map[string]any{"birthdate": "1994-05-01"}, 200, nil)
		x.do("POST", "/v1/profile/city", map[string]any{"query": "Testville"}, 200, nil)
		x.do("POST", "/v1/photos", photo(uint8(40+i*10)), 201, nil)
	}
	eventually(t, "photos to be approved", func() bool {
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM photos WHERE status='approved'`).Scan(&n)
		return n >= 2
	})

	var p1, p2 struct {
		Request dating.BuddyRequest `json:"request"`
	}
	a.do("POST", "/v1/buddies", map[string]any{"kind": "meal", "when": "Friday night", "note": "Spicy hotpot, anyone?", "expires_in_hours": 48}, 201, &p1)
	b.do("POST", "/v1/buddies", map[string]any{"kind": "meal", "when": "Saturday", "note": "Hotpot with a few people", "expires_in_hours": 48, "spots": 3}, 201, &p2)

	scene := func(c *client, id string) string {
		var one struct {
			Request dating.BuddyRequest `json:"request"`
		}
		c.do("GET", "/v1/buddies/"+id, nil, 200, &one)
		return one.Request.Scene
	}
	eventually(t, "both plans to get their picture", func() bool { return scene(a, p1.Request.ID) != "" && scene(b, p2.Request.ID) != "" })
	k1, k2 := scene(a, p1.Request.ID), scene(b, p2.Request.ID)
	if k1 != k2 {
		t.Fatalf("the same scene in the same city got two pictures: %s %s", k1, k2)
	}
	if n := e.images.drawn.Load(); n != 1 {
		t.Fatalf("the model drew %d pictures for one scene", n)
	}

	// Anyone can load it: it shows no one. A made-up key is not found.
	res, err := http.Get(e.api.URL + "/v1/scenes/" + k1 + ".jpg")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("scene: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	if res, _ := http.Get(e.api.URL + "/v1/scenes/000000000000000000000000.jpg"); res == nil || res.StatusCode != 404 {
		t.Fatal("a made-up scene was served")
	}

	// A trip gets its city's picture.
	day := func(n int) string { return time.Now().UTC().AddDate(0, 0, n).Format("2006-01-02") }
	var trip struct {
		Trip dating.Trip `json:"trip"`
	}
	a.do("POST", "/v1/trips", map[string]any{"city": "Farport", "starts_on": day(1), "ends_on": day(3)}, 201, &trip)
	eventually(t, "the trip to get its picture", func() bool {
		a.do("GET", "/v1/trips/"+trip.Trip.ID, nil, 200, &trip)
		return trip.Trip.Scene != "" && trip.Trip.Scene != k1
	})
}
