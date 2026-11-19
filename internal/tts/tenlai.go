package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Tenlai is Tencent's speech synthesis (天籁, on TokenHub), for the China copy: the same
// platform and key as its Hunyuan chat model. One request reads at most 100 Chinese
// characters (about 300 letters of English), so a longer line goes in pieces, cut at
// sentence ends, and the MP3s are joined in order.
type Tenlai struct {
	Endpoint string
	APIKey   string
	Model    string
	Voice    int
	Timeout  time.Duration
	HTTP     *http.Client
}

// DefaultTenlaiEndpoint is TokenHub's synchronous text-to-voice.
const DefaultTenlaiEndpoint = "https://tokenhub.tencentmaas.com/v1/tenlai/ttsproxy/text_to_voice"

const (
	tenlaiCJKChars   = 100
	tenlaiOtherChars = 300
)

func NewTenlai(endpoint, apiKey, model, voice string) (*Tenlai, error) {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(voice) == "" {
		return nil, ErrNotConfigured
	}
	id, err := strconv.Atoi(strings.TrimSpace(voice))
	if err != nil {
		return nil, fmt.Errorf("tts: a tenlai voice is a number, not %q", voice)
	}
	if endpoint == "" {
		endpoint = DefaultTenlaiEndpoint
	}
	if model == "" {
		model = "tenlai-tts-v1.0"
	}
	return &Tenlai{Endpoint: endpoint, APIKey: strings.TrimSpace(apiKey), Model: model, Voice: id, Timeout: 45 * time.Second}, nil
}

func (c *Tenlai) SpeakMP3(ctx context.Context, text string) ([]byte, string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, "", errors.New("tts: nothing to speak")
	}
	if rs := []rune(text); len(rs) > maxVendorChars {
		text = string(rs[:maxVendorChars])
	}
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	var audio bytes.Buffer
	for _, piece := range tenlaiPieces(text) {
		b, err := c.speak(ctx, piece)
		if err != nil {
			return nil, "", err
		}
		audio.Write(b)
	}
	if audio.Len() == 0 {
		return nil, "", errors.New("tts: tenlai returned no audio")
	}
	return audio.Bytes(), "audio/mpeg", nil
}

func (c *Tenlai) speak(ctx context.Context, text string) ([]byte, error) {
	body, _ := json.Marshal(map[string]any{
		"model":         c.Model,
		"text":          text,
		"voice_setting": map[string]any{"voice_id": c.Voice, "speed": 1, "volume": 0},
		"audio_setting": map[string]any{"sample_rate": 24000},
		"output_format": "mp3",
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("tts: tokenhub unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	var out struct {
		Data struct {
			AudioURL string `json:"audio_url"`
			Audio    string `json:"audio"`
		} `json:"data"`
		Status string `json:"status"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.Error != nil {
		return nil, fmt.Errorf("tts: tenlai %d: %s %s", resp.StatusCode, out.Error.Code, truncate(out.Error.Message, 200))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tts: tenlai %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	if out.Status != "" && out.Status != "completed" {
		return nil, fmt.Errorf("tts: tenlai status %s", out.Status)
	}
	got := out.Data.AudioURL
	if got == "" {
		got = out.Data.Audio
	}
	if strings.HasPrefix(got, "http://") || strings.HasPrefix(got, "https://") {
		return c.fetch(ctx, got)
	}
	// The docs call the same field base64 audio in one place; take that too.
	b, err := base64.StdEncoding.DecodeString(got)
	if err != nil || len(b) == 0 {
		return nil, errors.New("tts: tenlai sent no audio")
	}
	return b, nil
}

func (c *Tenlai) fetch(ctx context.Context, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("tts: fetching tenlai audio: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tts: fetching tenlai audio: %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 16<<20))
}

func (c *Tenlai) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: c.Timeout}
}

// tenlaiPieces cuts text into pieces one request can read: at sentence ends where it can,
// then at commas, then anywhere. Text with any Chinese, Japanese or Korean in it gets the
// tighter limit.
func tenlaiPieces(text string) []string {
	limit := tenlaiOtherChars
	for _, r := range text {
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
			limit = tenlaiCJKChars
			break
		}
	}
	var pieces []string
	rs := []rune(text)
	for len(rs) > limit {
		cut := lastBreak(rs[:limit], "。！？!?.；;\n")
		if cut == 0 {
			cut = lastBreak(rs[:limit], "，,、：: ")
		}
		if cut == 0 {
			cut = limit
		}
		if p := strings.TrimSpace(string(rs[:cut])); p != "" {
			pieces = append(pieces, p)
		}
		rs = rs[cut:]
	}
	if p := strings.TrimSpace(string(rs)); p != "" {
		pieces = append(pieces, p)
	}
	return pieces
}

// lastBreak is the position just after the last of marks in rs, or 0.
func lastBreak(rs []rune, marks string) int {
	for i := len(rs) - 1; i > 0; i-- {
		if strings.ContainsRune(marks, rs[i]) {
			return i + 1
		}
	}
	return 0
}
