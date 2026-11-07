package tts

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// CosyVoice is Alibaba Cloud's speech synthesis (DashScope, Bailian), for the China copy,
// where Fish Audio can't be reached. DashScope only offers it over a WebSocket: start a
// task, send the text, finish, and the MP3 arrives as binary frames until task-finished.
type CosyVoice struct {
	Endpoint string
	APIKey   string
	Model    string
	Voice    string
	Timeout  time.Duration
}

// DefaultCosyVoiceEndpoint is DashScope's mainland inference socket.
const DefaultCosyVoiceEndpoint = "wss://dashscope.aliyuncs.com/api-ws/v1/inference"

func NewCosyVoice(endpoint, apiKey, model, voice string) (*CosyVoice, error) {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(voice) == "" {
		return nil, ErrNotConfigured
	}
	if endpoint == "" {
		endpoint = DefaultCosyVoiceEndpoint
	}
	if model == "" {
		model = "cosyvoice-v2"
	}
	return &CosyVoice{Endpoint: endpoint, APIKey: strings.TrimSpace(apiKey), Model: model, Voice: strings.TrimSpace(voice), Timeout: 45 * time.Second}, nil
}

type cosyHeader struct {
	Action       string `json:"action,omitempty"`
	TaskID       string `json:"task_id"`
	Streaming    string `json:"streaming,omitempty"`
	Event        string `json:"event,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type cosyMessage struct {
	Header  cosyHeader     `json:"header"`
	Payload map[string]any `json:"payload"`
}

func (c *CosyVoice) SpeakMP3(ctx context.Context, text string) ([]byte, string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, "", errors.New("tts: nothing to speak")
	}
	if rs := []rune(text); len(rs) > maxVendorChars {
		text = string(rs[:maxVendorChars])
	}
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, c.Endpoint, &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"bearer " + c.APIKey}},
	})
	if err != nil {
		return nil, "", fmt.Errorf("tts: dashscope unreachable: %w", err)
	}
	defer conn.CloseNow()
	conn.SetReadLimit(16 << 20)

	id := taskID()
	send := func(action string, payload map[string]any) error {
		b, _ := json.Marshal(cosyMessage{Header: cosyHeader{Action: action, TaskID: id, Streaming: "duplex"}, Payload: payload})
		return conn.Write(ctx, websocket.MessageText, b)
	}
	if err := send("run-task", map[string]any{
		"task_group": "audio", "task": "tts", "function": "SpeechSynthesizer", "model": c.Model,
		"parameters": map[string]any{"text_type": "PlainText", "voice": c.Voice, "format": "mp3", "sample_rate": 22050},
		"input":      map[string]any{},
	}); err != nil {
		return nil, "", fmt.Errorf("tts: start: %w", err)
	}

	var audio bytes.Buffer
	sent := false
	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("tts: dashscope read: %w", err)
		}
		if typ == websocket.MessageBinary {
			audio.Write(data)
			continue
		}
		var m cosyMessage
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, "", fmt.Errorf("tts: dashscope sent %q", truncate(string(data), 200))
		}
		switch m.Header.Event {
		case "task-started":
			if !sent {
				sent = true
				if err := send("continue-task", map[string]any{"input": map[string]any{"text": text}}); err != nil {
					return nil, "", fmt.Errorf("tts: send text: %w", err)
				}
				if err := send("finish-task", map[string]any{"input": map[string]any{}}); err != nil {
					return nil, "", fmt.Errorf("tts: finish: %w", err)
				}
			}
		case "task-finished":
			conn.Close(websocket.StatusNormalClosure, "")
			if audio.Len() == 0 {
				return nil, "", errors.New("tts: dashscope returned no audio")
			}
			return audio.Bytes(), "audio/mpeg", nil
		case "task-failed":
			return nil, "", fmt.Errorf("tts: dashscope failed: %s %s", m.Header.ErrorCode, m.Header.ErrorMessage)
		}
	}
}

func taskID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
