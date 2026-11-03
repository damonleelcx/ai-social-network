package dating

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/damonleelcx/ai-social-network/internal/chatstore"
)

func TestDateNotesFollowTheReader(t *testing.T) {
	stored := func() []chatstore.Message {
		return []chatstore.Message{
			{Role: "date", Body: "It's Cafe! Now, when works for you both?", Meta: map[string]any{
				"note": "It's {venue}! Now, when works for you both?", "note_vars": bson.D{{Key: "venue", Value: "Cafe"}}}},
			{Role: "date", Body: "x", Meta: map[string]any{
				"note":      "I couldn't find good places nearby just now. {why} You can ask me to try again, or pick somewhere yourselves.",
				"note_vars": map[string]any{"why": "The search didn't work this time."}}},
			{Role: "date", Body: "An old note in English"},
			{Role: "text", Body: "你好", Meta: map[string]any{"client_id": "c"}},
		}
	}

	// The same stored messages read differently for two people on one match.
	zh := stored()
	localizeNotes(zh, "zh")
	if zh[0].Body != "就去 Cafe！那么，你们什么时候方便？" {
		t.Errorf("zh venue: %q", zh[0].Body)
	}
	if zh[1].Body != "我暂时没在附近找到合适的地方。这次搜索没有成功。你们可以让我再试一次，或者自己选个地方。" {
		t.Errorf("zh why: %q", zh[1].Body)
	}
	en := stored()
	localizeNotes(en, "en")
	if en[0].Body != "It's Cafe! Now, when works for you both?" {
		t.Errorf("en: %q", en[0].Body)
	}
	if en[1].Body != "I couldn't find good places nearby just now. The search didn't work this time. You can ask me to try again, or pick somewhere yourselves." {
		t.Errorf("en why: %q", en[1].Body)
	}
	if zh[2].Body != "An old note in English" || zh[3].Body != "你好" {
		t.Errorf("messages without a template changed: %q %q", zh[2].Body, zh[3].Body)
	}
}
