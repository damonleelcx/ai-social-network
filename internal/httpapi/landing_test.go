package httpapi

import (
	"compress/gzip"
	"io"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func getLanding(t *testing.T, url string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	landing.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
	return w
}

func TestLandingSaysWhatIsTrue(t *testing.T) {
	w := getLanding(t, "/")
	if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("home: %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	page := w.Body.String()
	for _, want := range []string{"isn't in the App Store", "adults 18 and over", "Suki is an AI", "/legal/privacy", "/legal/terms", "/legal/delete-account", "download/SynAgent.apk", "download/SynAgent-unsigned.ipa", "not signed"} {
		if !strings.Contains(page, want) {
			t.Errorf("home page lost %q", want)
		}
	}
}

// Scores stay hidden, even made-up ones: the sample cards are words only.
func TestLandingShowsNoScores(t *testing.T) {
	percent := regexp.MustCompile(`\d+\s*%\s*match|% match|match score|compatibility`)
	for _, f := range []string{"/", "/landing/app.js"} {
		if m := percent.FindString(strings.ToLower(getLanding(t, f).Body.String())); m != "" {
			t.Errorf("%s shows a score: %q", f, m)
		}
	}
}

func TestLandingAssetsResolve(t *testing.T) {
	page := getLanding(t, "/").Body.String()
	css := getLanding(t, "/landing/style.css").Body.String()
	urls := regexp.MustCompile(`/landing/[\w./-]+`).FindAllString(page, -1)
	for _, m := range regexp.MustCompile(`url\((fonts/[\w.-]+|img/[\w.-]+)\)`).FindAllStringSubmatch(css, -1) {
		urls = append(urls, "/landing/"+m[1])
	}
	urls = append(urls, "/landing/scene.js", "/landing/vendor/three.js", "/landing/img/suki.png", "/landing/img/suki-mono.png")
	for _, u := range urls {
		w := getLanding(t, u)
		if w.Code != 200 || w.Header().Get("Content-Type") == "" {
			t.Errorf("%s: %d %q", u, w.Code, w.Header().Get("Content-Type"))
		}
	}
	if w := getLanding(t, "/landing/nope.js"); w.Code != 404 {
		t.Errorf("missing file: %d", w.Code)
	}
	if w := getLanding(t, "/landing/vendor/README.md"); w.Code != 404 {
		t.Errorf("repo notes are being served: %d", w.Code)
	}
}

// A new page with last hour's stylesheet renders broken, so the two can't be cached apart.
func TestLandingCodeIsNeverStale(t *testing.T) {
	for _, f := range []string{"/", "/landing/style.css", "/landing/app.js", "/landing/scene.js", "/landing/vendor/three.js"} {
		if got := getLanding(t, f).Header().Get("Cache-Control"); got != "no-cache" {
			t.Errorf("%s: Cache-Control %q", f, got)
		}
	}
	if got := getLanding(t, "/landing/img/suki.png").Header().Get("Cache-Control"); got != "public, max-age=3600" {
		t.Errorf("image: Cache-Control %q", got)
	}
}

func TestLandingGzipAndRevalidation(t *testing.T) {
	plain := getLanding(t, "/landing/app.js").Body.String()

	r := httptest.NewRequest("GET", "/landing/app.js", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	landing.ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("not gzipped: %v", w.Header())
	}
	etag := w.Header().Get("ETag")
	gz, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(gz)
	if err != nil || string(got) != plain {
		t.Errorf("gzip round trip: %v, %d vs %d bytes", err, len(got), len(plain))
	}

	r = httptest.NewRequest("GET", "/landing/app.js", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	r.Header.Set("If-None-Match", etag)
	w = httptest.NewRecorder()
	landing.ServeHTTP(w, r)
	if w.Code != 304 {
		t.Errorf("revalidation: %d", w.Code)
	}

	// Images are already compressed.
	r = httptest.NewRequest("GET", "/landing/img/suki.png", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w = httptest.NewRecorder()
	landing.ServeHTTP(w, r)
	if w.Header().Get("Content-Encoding") != "" {
		t.Errorf("png gzipped again")
	}
}
