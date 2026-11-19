package scenes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Both image providers: the request each expects, and where each puts the picture.
func TestAskSpeaksEachProvider(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		if strings.Contains(r.URL.Path, "/hunyuan-image/") {
			if in["size"] != "1472x1140" || in["messages"] == nil || in["input"] != nil {
				t.Errorf("hunyuan request %v", in)
			}
			_, _ = w.Write([]byte(`{"choices":[{"delta":{"image":{"url":"https://cos.example/h.png"}},"finish_reason":null}]}`))
			return
		}
		if in["parameters"] == nil || in["input"] == nil {
			t.Errorf("dashscope request %v", in)
		}
		_, _ = w.Write([]byte(`{"output":{"choices":[{"message":{"content":[{"image":"https://oss.example/d.png"}]}}]}}`))
	}))
	defer srv.Close()
	for path, want := range map[string]string{
		"/v1/wand/hunyuan-image/v35-generation":                  "https://cos.example/h.png",
		"/api/v1/services/aigc/multimodal-generation/generation": "https://oss.example/d.png",
	} {
		s := &Service{ImageURL: srv.URL + path, ImageKey: "k", ImageModel: "m"}
		got, err := s.ask(context.Background(), "a scene")
		if err != nil || got != want {
			t.Errorf("%s: %q %v", path, got, err)
		}
	}
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error":{"message":"prompt refused"}}`))
	}))
	defer failing.Close()
	s := &Service{ImageURL: failing.URL + "/v1/wand/hunyuan-image/v35-generation"}
	if _, err := s.ask(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "prompt refused") {
		t.Errorf("error: %v", err)
	}
}
