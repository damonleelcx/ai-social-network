package tts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"
)

// A fake DashScope: checks the key and the protocol order, then streams two audio chunks.
func fakeDashScope(t *testing.T, fail bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "bearer k" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		ctx := r.Context()
		read := func() cosyMessage {
			_, b, err := c.Read(ctx)
			if err != nil {
				t.Errorf("read: %v", err)
			}
			var m cosyMessage
			_ = json.Unmarshal(b, &m)
			return m
		}
		event := func(id, ev string) {
			b, _ := json.Marshal(cosyMessage{Header: cosyHeader{TaskID: id, Event: ev, ErrorCode: "X", ErrorMessage: "bad voice"}})
			_ = c.Write(ctx, websocket.MessageText, b)
		}
		run := read()
		if run.Header.Action != "run-task" || run.Payload["model"] != "cosyvoice-v2" {
			t.Errorf("first message %+v", run)
		}
		if fail {
			event(run.Header.TaskID, "task-failed")
			return
		}
		event(run.Header.TaskID, "task-started")
		if m := read(); m.Header.Action != "continue-task" || m.Payload["input"].(map[string]any)["text"] != "你好" {
			t.Errorf("continue %+v", m)
		}
		if m := read(); m.Header.Action != "finish-task" {
			t.Errorf("finish %+v", m)
		}
		_ = c.Write(ctx, websocket.MessageBinary, []byte("ID3"))
		_ = c.Write(ctx, websocket.MessageBinary, []byte("mp3"))
		event(run.Header.TaskID, "task-finished")
	}))
}

func TestCosyVoiceSpeaks(t *testing.T) {
	s := fakeDashScope(t, false)
	defer s.Close()
	c, err := NewCosyVoice("ws"+strings.TrimPrefix(s.URL, "http"), "k", "", "longxiaochun_v2")
	if err != nil {
		t.Fatal(err)
	}
	audio, ct, err := c.SpeakMP3(context.Background(), "你好")
	if err != nil || ct != "audio/mpeg" || string(audio) != "ID3mp3" {
		t.Fatalf("got %q %q %v", audio, ct, err)
	}
}

func TestCosyVoiceReportsFailure(t *testing.T) {
	s := fakeDashScope(t, true)
	defer s.Close()
	c, _ := NewCosyVoice("ws"+strings.TrimPrefix(s.URL, "http"), "k", "", "v")
	if _, _, err := c.SpeakMP3(context.Background(), "你好"); err == nil || !strings.Contains(err.Error(), "bad voice") {
		t.Fatalf("want the vendor's reason, got %v", err)
	}
}

func TestCosyVoiceNeedsKeyAndVoice(t *testing.T) {
	if _, err := NewCosyVoice("", "", "", "v"); err != ErrNotConfigured {
		t.Fatal(err)
	}
}
