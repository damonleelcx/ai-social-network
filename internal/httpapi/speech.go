package httpapi

// Suki's voice.
//
// GET  /v1/agent/speech → {enabled}
// POST /v1/agent/speech → audio/mpeg, for one of:
//   {seq}          one of Suki's own messages in the caller's conversation
//   {sample:true}  the fixed "Hear Suki" line
//
// It only ever speaks text the server already holds as Suki's, so it can't be used
// as a general text-to-speech service. There's a per-minute limit and a daily
// character allowance per account, charged only for lines that reach the vendor.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/damonleelcx/ai-social-network/internal/dating"
	"github.com/damonleelcx/ai-social-network/internal/i18n"
	"github.com/damonleelcx/ai-social-network/internal/tts"
)

const speechSample = "Hi, I'm Suki! I'm so glad you're here. Let's find you someone wonderful."

var errSpeechCap = errors.New("Suki has talked a lot today. Her voice will be back tomorrow, and you can still read her messages.")

func (s *Server) speechStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]bool{"enabled": s.Speech != nil})
}

func (s *Server) chargeSpeech(ctx context.Context, userID string, chars int) error {
	var total int
	err := s.Pool.QueryRow(ctx, `INSERT INTO tts_usage (user_id, day, chars) VALUES ($1, (now() AT TIME ZONE 'UTC')::date, $2)
		ON CONFLICT (user_id, day) DO UPDATE SET chars = tts_usage.chars + EXCLUDED.chars
			WHERE tts_usage.chars + EXCLUDED.chars <= $3
		RETURNING chars`, userID, chars, s.SpeechDailyChars).Scan(&total)
	if errors.Is(err, pgx.ErrNoRows) {
		return errSpeechCap
	}
	return err
}

func (s *Server) speak(w http.ResponseWriter, r *http.Request) {
	if s.Speech == nil {
		fail(w, 503, "voice is not available")
		return
	}
	var b struct {
		Seq    int64 `json:"seq"`
		Sample bool  `json:"sample"`
	}
	if !read(w, r, &b) {
		return
	}
	u := userOf(r)
	text := i18n.T(s.lang(r, u.ID), speechSample)
	if !b.Sample {
		conv, err := dating.ConversationID(r.Context(), s.Pool, u.ID)
		if err != nil {
			handleErr(w, r, err)
			return
		}
		ms, err := s.Chats.AgentAfter(r.Context(), conv, b.Seq-1, 1)
		if err != nil {
			handleErr(w, r, err)
			return
		}
		if len(ms) == 0 || ms[0].Seq != b.Seq || ms[0].Role != "assistant" {
			fail(w, 404, "not found")
			return
		}
		text = ms[0].Body
	}
	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
	defer cancel()
	audio, ct, err := s.Speech.SpeakCharged(ctx, text, func(n int) error { return s.chargeSpeech(ctx, u.ID, n) })
	switch {
	case err == nil:
	case errors.Is(err, tts.ErrEmpty):
		fail(w, 422, "nothing to say")
		return
	case errors.Is(err, errSpeechCap):
		fail(w, 429, errSpeechCap.Error())
		return
	default:
		slog.Error("speech", "user", u.ID, "err", err)
		fail(w, 502, "Suki's voice is unavailable right now")
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Length", fmt.Sprint(len(audio)))
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = w.Write(audio)
}
