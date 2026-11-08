package agent

import "testing"

func TestOnePostCardATurn(t *testing.T) {
	drafted := map[string]any{"kind": "buddy_post", "note": "drafted"}
	cards := addCard([]map[string]any{drafted}, map[string]any{"kind": "buddy_post", "note": "from the model"})
	if len(cards) != 1 || cards[0]["note"] != "drafted" {
		t.Fatalf("post cards: %v", cards)
	}
	cards = addCard(cards, map[string]any{"kind": "photo_upload"})
	cards = addCard(cards, map[string]any{"kind": "photo_upload"})
	if len(cards) != 3 {
		t.Fatalf("other cards should all show: %v", cards)
	}
}

// "Singapore this weekend" drew both a travel post and a trip card. The trip is the right one.
func TestTripWinsOverATravelPost(t *testing.T) {
	travel := map[string]any{"kind": "buddy_post", "activity": "travel"}
	trip := map[string]any{"kind": "trip_post", "city": "Singapore"}
	cards := addCard(addCard(nil, travel), trip)
	if len(cards) != 1 || cards[0]["kind"] != "trip_post" {
		t.Fatalf("drafted first: %v", cards)
	}
	cards = addCard(addCard(nil, trip), travel)
	if len(cards) != 1 || cards[0]["kind"] != "trip_post" {
		t.Fatalf("trip first: %v", cards)
	}
	movie := map[string]any{"kind": "buddy_post", "activity": "movie"}
	if cards = addCard(addCard(nil, movie), trip); len(cards) != 2 {
		t.Fatalf("a plan that isn't travel stays: %v", cards)
	}
}
