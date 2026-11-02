package agent

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/damonleelcx/ai-social-network/internal/moderation"
	"github.com/damonleelcx/ai-social-network/internal/safety"
)

const (
	// streamHoldback is how far text stays behind the model. Every guard runs on the
	// held text before any of it is released, so a phrase the guards catch is seen whole
	// before its first word could reach the person. It has to be longer than any of them.
	streamHoldback = 48
	streamInterval = 70 * time.Millisecond
	// Postgres NOTIFY payloads are limited to 8000 bytes.
	streamMaxChunk = 400
)

// unsafeDraft is the check a draft must pass before anyone sees it.
func unsafeDraft(s string) bool { return safety.LeaksScore(s) || moderation.Explicit(s) }

// streamer sends a reply to the person's devices as it's written. Streaming is best
// effort: it's a live preview, and the stored message is the truth. A lost event costs
// nothing, because the final message replaces whatever is on screen.
type streamer struct {
	r       *Runner
	ctx     context.Context
	t       *Turn
	text    strings.Builder // everything the model has written in this step
	sent    int             // runes released so far
	offset  int             // UTF-16 units released, which is what the client counts
	stopped bool
	last    time.Time
}

func (s *streamer) emit(data map[string]any) {
	data["turn"] = s.t.ID
	raw, _ := json.Marshal(data)
	s.r.signal(s.ctx, s.t.UserID, "stream:"+string(raw))
}

// reset tells the client to drop what it has for this turn.
func (s *streamer) reset() {
	if s.offset > 0 {
		s.emit(map[string]any{"reset": true, "offset": 0, "text": ""})
	}
	s.sent, s.offset = 0, 0
}

// visible cuts the mood line, which is for the app and not for reading. A lone "<" could
// be its start, so it's held too.
func visible(s string) string {
	if i := strings.Index(s, "<<"); i >= 0 {
		return s[:i]
	}
	return strings.TrimSuffix(s, "<")
}

// delta takes one piece from the model.
func (s *streamer) delta(d string) {
	if s.stopped {
		return
	}
	s.text.WriteString(d)
	draft := visible(s.text.String())
	if unsafeDraft(draft) {
		s.stopped = true
		s.reset()
		return
	}
	if time.Since(s.last) < streamInterval {
		return
	}
	runes := []rune(draft)
	end := len(runes) - streamHoldback
	if end <= s.sent {
		return
	}
	// Release whole words, so the text doesn't appear mid-word.
	for end > s.sent && runes[end-1] != ' ' && runes[end-1] != '\n' {
		end--
	}
	if end > s.sent {
		s.release(runes, end)
	}
}

func (s *streamer) release(runes []rune, end int) {
	for s.sent < end {
		stop := min(end, s.sent+streamMaxChunk)
		chunk := runes[s.sent:stop]
		s.emit(map[string]any{"offset": s.offset, "text": string(chunk)})
		s.offset += len(utf16.Encode(chunk))
		s.sent = stop
	}
	s.last = time.Now()
}

// finish releases the rest of an approved reply, just before it's stored.
func (s *streamer) finish(body string) {
	if s.stopped {
		return
	}
	runes := []rune(body)
	if s.sent > len(runes) || !strings.HasPrefix(body, string([]rune(visible(s.text.String()))[:s.sent])) {
		s.reset() // the stored text differs from what was shown; the final message fixes it
		return
	}
	s.release(runes, len(runes))
}
