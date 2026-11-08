package e2e

import (
	"context"
	"strings"
	"testing"
)

type translation struct {
	Lang       string `json:"lang"`
	Translated bool   `json:"translated"`
	Words      struct {
		Bio       string   `json:"bio"`
		CityName  string   `json:"city_name"`
		Interests []string `json:"interests"`
		Prompts   []struct {
			Q string `json:"q"`
			A string `json:"a"`
		} `json:"prompts"`
	} `json:"words"`
}

// Someone you were introduced to, in your language: made once, then served from the cache, and
// only to people with a reason to see them.
func TestProfileTranslation(t *testing.T) {
	e := setup(t)
	a := e.signup(t, "Ada", "ada@example.com")
	b := e.signup(t, "Bo", "bo@example.com")
	c := e.signup(t, "Cy", "cy@example.com")
	for _, x := range []*client{a, b, c} {
		x.verify()
	}
	b.do("PATCH", "/v1/profile", map[string]any{"bio": "I love pool and karaoke.", "interests": []string{"pool", "karaoke"},
		"prompts": []map[string]string{{"q": "What does your ideal Sunday look like?", "a": "Gaming or exploring"}}}, 200, nil)
	b.do("POST", "/v1/profile/city", map[string]any{"query": "Testville"}, 200, nil)

	// No introduction yet: nothing to read.
	a.do("GET", "/v1/people/"+b.id+"/translation?lang=zh", nil, 404, nil)

	if _, err := e.pool.Exec(context.Background(), `INSERT INTO proposals (user_a, user_b) VALUES (least($1::uuid,$2::uuid), greatest($1::uuid,$2::uuid))`, a.id, b.id); err != nil {
		t.Fatal(err)
	}
	a.do("GET", "/v1/people/"+b.id+"/translation?lang=fr", nil, 422, nil)

	var tr translation
	a.do("GET", "/v1/people/"+b.id+"/translation?lang=zh", nil, 200, &tr)
	if !tr.Translated || tr.Words.Bio != "译:I love pool and karaoke." || len(tr.Words.Interests) != 2 || tr.Words.Interests[0] != "译:pool" ||
		len(tr.Words.Prompts) != 1 || tr.Words.Prompts[0].Q != "译:What does your ideal Sunday look like?" || !strings.HasPrefix(tr.Words.CityName, "译:") {
		t.Fatalf("translation: %+v", tr)
	}

	// The second read is the cache, not the model.
	calls := e.fake.translations.Load()
	a.do("GET", "/v1/people/"+b.id+"/translation?lang=zh", nil, 200, &tr)
	if e.fake.translations.Load() != calls || tr.Words.Bio != "译:I love pool and karaoke." {
		t.Fatal("the same profile was translated twice")
	}

	// A change to the profile is translated anew.
	b.do("PATCH", "/v1/profile", map[string]any{"bio": "Hot springs on Sundays."}, 200, nil)
	a.do("GET", "/v1/people/"+b.id+"/translation?lang=zh", nil, 200, &tr)
	if tr.Words.Bio != "译:Hot springs on Sundays." {
		t.Fatalf("stale translation: %q", tr.Words.Bio)
	}

	// A stranger gets nothing, and neither does someone who blocked or was blocked.
	c.do("GET", "/v1/people/"+b.id+"/translation?lang=zh", nil, 404, nil)
	b.do("POST", "/v1/blocks", map[string]any{"user_id": a.id}, 200, nil)
	a.do("GET", "/v1/people/"+b.id+"/translation?lang=zh", nil, 404, nil)

	// Each translation belongs to the person it translates, so it goes with their account.
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM translations WHERE user_id=$1`, b.id).Scan(&n)
	if n == 0 {
		t.Fatal("nothing was cached")
	}
}
