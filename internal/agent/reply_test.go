package agent

import "testing"

func TestParseReplyStripsMoodMarker(t *testing.T) {
	for _, c := range []struct {
		in, body, mood string
		replies        []string
	}{
		{"活泼、长期固定，收到啦。\n\n<<mood=happy|>>", "活泼、长期固定，收到啦。", "happy", nil},
		{"Saved! <<mood=happy|replies=Tell me more>>", "Saved!", "happy", []string{"Tell me more"}},
		{"Hey\n<<mood=playful|replies=Good idea;Something else>>", "Hey", "playful", []string{"Good idea", "Something else"}},
		{"Just the words <<mood=thinking>>", "Just the words", "thinking", nil},
		{"No marker", "No marker", "happy", nil},
		{"Cut off <<mood=pouty|", "Cut off", "pouty", nil},
	} {
		body, mood, replies := parseReply(c.in)
		if body != c.body || mood != c.mood || !sameReplies(replies, c.replies) {
			t.Errorf("parseReply(%q) = %q, %q, %#v\nwant %q, %q, %#v", c.in, body, mood, replies, c.body, c.mood, c.replies)
		}
		if got := DisplayBody(c.in); got != c.body {
			t.Errorf("DisplayBody(%q) = %q, want %q", c.in, got, c.body)
		}
	}
}

func sameReplies(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
