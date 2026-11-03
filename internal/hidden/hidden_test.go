package hidden

import (
	"math"
	"testing"

	"github.com/damonleelcx/ai-social-network/internal/profile"
)

func TestReputationNeedsThreeDifferentRaters(t *testing.T) {
	bad := RaterObs{Score: 1, RaterMean: 7, RaterCount: 5}
	if got := ReputationFrom([]RaterObs{bad, bad}); got != 0 {
		t.Fatalf("two raters moved reputation to %v", got)
	}
	if got := ReputationFrom([]RaterObs{bad, bad, bad}); got >= 0 {
		t.Fatalf("three bad dates should lower it, got %v", got)
	}
}

func TestHarshRaterIsNormalised(t *testing.T) {
	// A 6 from someone who averages 4 is praise. A 6 from someone who averages 9 isn't.
	fromHarsh := []RaterObs{{6, 4, 5}, {6, 4, 5}, {6, 4, 5}}
	fromKind := []RaterObs{{6, 9, 5}, {6, 9, 5}, {6, 9, 5}}
	if h, k := ReputationFrom(fromHarsh), ReputationFrom(fromKind); h <= 0 || k >= 0 {
		t.Fatalf("harsh rater %v should raise, kind rater %v should lower", h, k)
	}
}

func TestReputationIsBounded(t *testing.T) {
	terrible := make([]RaterObs, 500)
	for i := range terrible {
		terrible[i] = RaterObs{0, 10, 10}
	}
	r := ReputationFrom(terrible)
	if r < -1 || r > 1 {
		t.Fatalf("reputation out of range: %v", r)
	}
	// However bad, it can only move the ranking by the stated weight.
	base := RankScore(0.7, 0.7, 0)
	worst := RankScore(0.7, 0.7, -5)
	if math.Abs(worst-base)/base > ReputationWeight+1e-9 {
		t.Fatalf("reputation moved the ranking by %v", math.Abs(worst-base)/base)
	}
}

func TestLearningShiftsTowardWhatTheyLike(t *testing.T) {
	viewer := &profile.Profile{Age: 30, Interests: []string{"hiking"}, Goal: "long_term"}
	hiker := &profile.Profile{Age: 30, Interests: []string{"climbing", "trail running"}, Goal: "long_term"}
	gamer := &profile.Profile{Age: 30, Interests: []string{"esports", "anime"}, Goal: "long_term"}
	w := map[string]float64{}
	before := predict(w, Features(viewer, hiker)) - predict(w, Features(viewer, gamer))
	for i := 0; i < 40; i++ {
		update(w, Features(viewer, hiker), 1, 0.6)
		update(w, Features(viewer, gamer), 0, 0.6)
	}
	after := predict(w, Features(viewer, hiker)) - predict(w, Features(viewer, gamer))
	if after <= before {
		t.Fatalf("taste didn't learn: gap %v then %v", before, after)
	}
	if w["likes_outdoors"] <= 0 || w["likes_games"] >= 0 {
		t.Fatalf("expected outdoors up and games down: %+v", w)
	}
}

func TestFeaturesCannotCarryProtectedAttributes(t *testing.T) {
	a := &profile.Profile{Age: 30, Religion: "x", Politics: "y", Gender: "woman"}
	b := &profile.Profile{Age: 31, Religion: "x", Politics: "y", Gender: "man"}
	for k := range Features(a, b) {
		for _, banned := range []string{"religion", "politics", "gender", "race", "ethnic", "photo", "look"} {
			if k == banned {
				t.Fatalf("feature %s exists", k)
			}
		}
	}
}

func TestDealbreakersApplyOnlyToStatedValues(t *testing.T) {
	from := &profile.Profile{Dealbreaker: map[string][]string{"smoking": {"yes"}}}
	if !dealbroken(from, &profile.Profile{Smoking: "yes"}) {
		t.Fatal("a smoker should be excluded")
	}
	if dealbroken(from, &profile.Profile{Smoking: "no"}) {
		t.Fatal("a non-smoker should pass")
	}
	if dealbroken(from, &profile.Profile{}) {
		t.Fatal("no answer isn't a dealbreaker")
	}
}

func TestPitchCheckCatchesNumbers(t *testing.T) {
	for _, s := range []string{"You're a 92% match", "Your compatibility score is high", "rated highly", "9/10 for sure", "ranked first"} {
		if !LeaksNumbers(s) {
			t.Errorf("should be caught: %s", s)
		}
	}
	if LeaksNumbers("You both love live jazz and quiet Sundays.") {
		t.Error("clean copy was caught")
	}
}
