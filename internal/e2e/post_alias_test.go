package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// wx.request can't send PATCH, so the mini program POSTs to the same two routes. These
// must stay the same as PATCH: same handler, same sign-in rules, same answers.
func TestPostAliasesBehaveLikePatch(t *testing.T) {
	e := setup(t)
	methods := []string{"PATCH", "POST"}

	nobody := &client{t: t, e: e}
	for _, m := range methods {
		nobody.do(m, "/v1/settings", map[string]any{"suki_tone": "calm"}, 401, nil)
		nobody.do(m, "/v1/profile", map[string]any{"bio": "hello"}, 401, nil)
	}

	// Settings only need a session. The profile needs a verified email.
	c := e.signup(t, "Alias", "alias@example.com")
	for _, m := range methods {
		c.do(m, "/v1/settings", map[string]any{"suki_tone": "warm"}, 200, nil)
		c.do(m, "/v1/profile", map[string]any{"bio": "too early"}, 403, nil)
	}
	c.verify()

	// The same change through each method leaves the same settings, and answers with them.
	settingsVia := func(m string, change map[string]any) string {
		t.Helper()
		var out, stored map[string]any
		c.do(m, "/v1/settings", change, 200, &out)
		c.do("GET", "/v1/settings", nil, 200, &stored)
		a, _ := json.Marshal(out["settings"])
		b, _ := json.Marshal(stored["settings"])
		if string(a) != string(b) {
			t.Fatalf("%s /v1/settings answered %s, but stored %s", m, a, b)
		}
		return string(a)
	}
	change := map[string]any{"suki_tone": "direct", "daily_matches": 2, "suki_nickname": "Al"}
	viaPatch := settingsVia("PATCH", change)
	settingsVia("POST", map[string]any{"suki_tone": "calm", "daily_matches": 4, "suki_nickname": ""})
	if viaPost := settingsVia("POST", change); viaPost != viaPatch {
		t.Fatalf("settings differ:\nPATCH %s\nPOST  %s", viaPatch, viaPost)
	}
	for _, m := range methods {
		c.do(m, "/v1/settings", map[string]any{"made_up": true}, 422, nil)
		c.do(m, "/v1/settings", map[string]any{"daily_matches": 50}, 422, nil)
	}

	// The same for the profile, including what it refuses.
	profileVia := func(m string, patch map[string]any) string {
		t.Helper()
		var out, stored map[string]any
		c.do(m, "/v1/profile", patch, 200, &out)
		c.do("GET", "/v1/profile", nil, 200, &stored)
		a, _ := json.Marshal(out["profile"])
		b, _ := json.Marshal(stored["profile"])
		if string(a) != string(b) {
			t.Fatalf("%s /v1/profile answered %s, but stored %s", m, a, b)
		}
		return string(a)
	}
	patch := map[string]any{
		"bio":          "Here for real conversations.",
		"interests":    []string{"jazz", "ramen", "climbing"},
		"prompts":      []map[string]string{{"q": "My ideal Sunday", "a": "Long walks"}},
		"dealbreakers": map[string]any{"smoking": []string{"yes"}},
		"age_min":      25,
		"age_max":      45,
	}
	viaPatch = profileVia("PATCH", patch)
	profileVia("POST", map[string]any{"bio": "Something else.", "age_min": 30})
	if viaPost := profileVia("POST", patch); viaPost != viaPatch {
		t.Fatalf("profiles differ:\nPATCH %s\nPOST  %s", viaPatch, viaPost)
	}
	for _, m := range methods {
		c.do(m, "/v1/profile", map[string]any{"bio": "send nudes"}, 422, nil)
	}

	// An underage birthdate is stopped and held either way.
	for i, m := range methods {
		kid := e.signup(t, "Kid", []string{"kid-patch@example.com", "kid-post@example.com"}[i])
		kid.verify()
		kid.do(m, "/v1/profile", map[string]any{"birthdate": time.Now().AddDate(-16, 0, 0).Format("2006-01-02")}, 403, nil)
		var status string
		if err := e.pool.QueryRow(context.Background(), `SELECT status FROM profiles WHERE user_id=$1`, kid.id).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if status != "held" {
			t.Fatalf("%s: an underage profile has status %q", m, status)
		}
	}
}
