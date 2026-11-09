package e2e

import "testing"

// A card is shown only once its facts have been checked against what the person said. The last
// message is the real failure: a map search put a Malaysia trip in another country.
func TestCardsAreFactCheckedBeforeTheyShow(t *testing.T) {
	e := setup(t)
	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()
	shown := func(text string) bool {
		t.Helper()
		_, meta := a.askSuki(text)
		cards, _ := meta["cards"].([]any)
		return len(cards) == 1 && cards[0].(map[string]any)["kind"] == "trip_post"
	}

	if !shown("trip please, Farport tomorrow") {
		t.Fatal("a card that checks out wasn't shown")
	}
	if shown("trip please, unreadable verdict") {
		t.Fatal("a card that couldn't be checked was shown")
	}
	if shown("trip please, 明天去马来西亚") {
		t.Fatal("a card that failed its check was shown")
	}
}
