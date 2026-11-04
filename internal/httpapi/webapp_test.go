package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWebapp(t *testing.T) {
	h := newWebapp(fstest.MapFS{
		"webapp/index.html":             {Data: []byte("<title>page</title>")},
		"webapp/_expo/static/js/app.js": {Data: []byte("run()")},
		"webapp/assets/secret.map":      {Data: []byte("not served")},
	})
	get := func(p string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		return w
	}
	for _, p := range []string{"/app/", "/app", "/app/chat/123", "/app/index.html"} {
		w := get(p)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "page") {
			t.Errorf("%s: %d %q", p, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-cache" || !strings.Contains(w.Header().Get("Content-Security-Policy"), "script-src 'self'") {
			t.Errorf("%s: headers %v", p, w.Header())
		}
	}
	w := get("/app/_expo/static/js/app.js")
	if w.Code != 200 || w.Body.String() != "run()" || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("script: %d %v", w.Code, w.Header())
	}
	for _, p := range []string{"/app/_expo/static/js/gone.js", "/app/assets/secret.map"} {
		if w := get(p); w.Code != 404 {
			t.Errorf("%s: %d, want 404 rather than the page", p, w.Code)
		}
	}
}

func TestBrowserSocketToken(t *testing.T) {
	var got string
	h := browserSocketAuth(func(_ http.ResponseWriter, r *http.Request) { got = bearer(r) })
	for _, c := range []struct{ header, auth, want string }{
		{"synagent, bearer.abc-_1", "", "abc-_1"},
		{"bearer.abc", "Bearer native", "native"},
		{"synagent", "", ""},
		{"", "", ""},
	} {
		r := httptest.NewRequest(http.MethodGet, "/v1/ws", nil)
		if c.header != "" {
			r.Header.Set("Sec-WebSocket-Protocol", c.header)
		}
		if c.auth != "" {
			r.Header.Set("Authorization", c.auth)
		}
		h(httptest.NewRecorder(), r)
		if got != c.want {
			t.Errorf("%q: token %q, want %q", c.header, got, c.want)
		}
	}
}
