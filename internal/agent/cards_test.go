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
