package e2e

import (
	"context"
	"testing"
)

// A live selfie that matches the main photo earns the badge; one that doesn't says why and
// earns nothing; attempts are limited; and nothing of the selfie is kept.
func TestSelfieVerification(t *testing.T) {
	e := setup(t)
	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()

	type result struct {
		Verified bool   `json:"verified"`
		Reason   string `json:"reason"`
	}
	var r result
	// Nothing to compare with yet.
	a.do("POST", "/v1/verify/selfie", photo(90), 422, nil)

	a.do("POST", "/v1/photos", photo(40), 201, nil)
	eventually(t, "the photo to be approved", func() bool {
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM photos WHERE status='approved'`).Scan(&n)
		return n == 1
	})
	var photos int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM photos`).Scan(&photos)

	e.fake.selfieMismatch.Store(true)
	a.do("POST", "/v1/verify/selfie", photo(90), 200, &r)
	if r.Verified || r.Reason == "" {
		t.Fatalf("a mismatch: %+v", r)
	}
	verified := func() bool {
		var me struct {
			Profile struct {
				Verified bool `json:"verified"`
			} `json:"profile"`
		}
		a.do("GET", "/v1/me", nil, 200, &me)
		return me.Profile.Verified
	}
	if verified() {
		t.Fatal("verified after a mismatch")
	}

	e.fake.selfieMismatch.Store(false)
	a.do("POST", "/v1/verify/selfie", photo(90), 200, &r)
	if !r.Verified || !verified() {
		t.Fatalf("a match didn't verify: %+v", r)
	}

	// The selfie was only looked at: no new photo, nothing stored.
	var after int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM photos`).Scan(&after)
	if after != photos {
		t.Fatal("the selfie was kept as a photo")
	}

	// Five tries a day.
	for i := 0; i < 3; i++ {
		a.do("POST", "/v1/verify/selfie", photo(90), 200, &r)
	}
	a.do("POST", "/v1/verify/selfie", photo(90), 422, nil)
}
