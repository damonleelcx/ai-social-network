package tts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTenlaiPiecesFitOneRequest(t *testing.T) {
	zh := strings.Repeat("今天天气很好，我们一起去看展吧。", 12)
	for _, p := range tenlaiPieces(zh) {
		if n := utf8.RuneCountInString(p); n > tenlaiCJKChars {
			t.Fatalf("piece of %d chars: %q", n, p)
		}
		if !strings.HasSuffix(p, "。") {
			t.Errorf("piece not cut at a sentence end: %q", p)
		}
	}
	if got := strings.Join(tenlaiPieces(zh), ""); got != zh {
		t.Errorf("pieces lost text")
	}
	en := strings.Repeat("Shall we see the new show this weekend? ", 20)
	for _, p := range tenlaiPieces(en) {
		if n := utf8.RuneCountInString(p); n > tenlaiOtherChars {
			t.Fatalf("piece of %d chars", n)
		}
	}
	if p := tenlaiPieces(strings.Repeat("好", 250)); len(p) != 3 {
		t.Errorf("unbroken text: %d pieces", len(p))
	}
}

func TestTenlaiJoinsPiecesInOrder(t *testing.T) {
	var srv *httptest.Server
	var texts []string
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/audio/") {
			_, _ = w.Write([]byte(strings.TrimPrefix(r.URL.Path, "/audio/")))
			return
		}
		if r.Header.Get("Authorization") != "Bearer k" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"401","message":"bad key"}}`))
			return
		}
		var in struct {
			Model, Text  string
			VoiceSetting struct {
				VoiceID int `json:"voice_id"`
			} `json:"voice_setting"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		if in.Model != "tenlai-tts-v1.0" || in.VoiceSetting.VoiceID != 502001 {
			t.Errorf("request %+v", in)
		}
		texts = append(texts, in.Text)
		_, _ = w.Write([]byte(`{"status":"completed","data":{"audio_url":"` + srv.URL + `/audio/` + string(rune('a'+len(texts)-1)) + `"}}`))
	}))
	defer srv.Close()
	c, err := NewTenlai(srv.URL, "k", "", "502001")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("今天天气很好，我们一起去看展吧。", 12)
	b, ct, err := c.SpeakMP3(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	if ct != "audio/mpeg" || string(b) != "abc"[:len(texts)] || len(texts) < 2 {
		t.Errorf("got %q %s from %d requests", b, ct, len(texts))
	}
	bad, _ := NewTenlai(srv.URL, "wrong", "", "502001")
	if _, _, err := bad.SpeakMP3(context.Background(), "你好"); err == nil || !strings.Contains(err.Error(), "bad key") {
		t.Errorf("bad key: %v", err)
	}
	if _, err := NewTenlai(srv.URL, "k", "", "longxiaochun_v2"); err == nil {
		t.Error("a non-numeric voice was accepted")
	}
}
