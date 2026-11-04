package httpapi

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// The mobile app's web build. The image build replaces this folder with `expo export`'s
// output, so the checked-in index.html is only what a plain `go build` serves. "all:" because
// Expo puts its scripts under _expo, which embed otherwise skips.
//
//go:embed all:webapp
var webappFiles embed.FS

var webappTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".json":  "application/json",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".webp":  "image/webp",
	".ico":   "image/x-icon",
	".ttf":   "font/ttf",
	".woff2": "font/woff2",
}

// The session token lives in localStorage, so no script but ours may run here. Styles stay
// inline because react-native-web writes them at runtime.
const webappCSP = "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; " +
	"font-src 'self' data:; media-src blob:; connect-src 'self' blob:; manifest-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

var webapp = newWebapp(webappFiles)

// newWebapp serves the single-page app under /app. A path that isn't a file is a screen, and
// gets the page; a missing file stays a 404 so a stale script can't come back as HTML.
func newWebapp(files fs.FS) http.Handler {
	assets := map[string]*staticAsset{}
	err := fs.WalkDir(files, "webapp", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ctype, ok := webappTypes[path.Ext(name)]
		if !ok {
			return nil
		}
		body, err := fs.ReadFile(files, name)
		if err != nil {
			return err
		}
		a, err := newStaticAsset(body, ctype)
		if err != nil {
			return err
		}
		assets["/app/"+strings.TrimPrefix(name, "webapp/")] = a
		return nil
	})
	if err != nil {
		panic(err) // embedded at build time, so this can only be a bad build
	}
	page := assets["/app/index.html"]
	if page == nil {
		panic("webapp has no index.html")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a, ok := assets[r.URL.Path]
		if !ok && path.Ext(r.URL.Path) != "" {
			http.NotFound(w, r)
			return
		}
		h := w.Header()
		if !ok || a == page {
			a = page
			h.Set("Cache-Control", "no-cache")
			h.Set("Content-Security-Policy", webappCSP)
			h.Set("Referrer-Policy", "no-referrer")
		} else {
			// Expo hashes the file names of everything the page loads.
			h.Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		a.serve(w, r)
	})
}

type staticAsset struct {
	body, gz []byte
	ctype    string
	etag     string
}

func newStaticAsset(body []byte, ctype string) (*staticAsset, error) {
	sum := sha256.Sum256(body)
	a := &staticAsset{body: body, ctype: ctype, etag: hex.EncodeToString(sum[:8])}
	if strings.HasPrefix(ctype, "text/") || ctype == "image/svg+xml" || ctype == "application/json" || ctype == "font/ttf" {
		gz, err := gzipped(body)
		if err != nil {
			return nil, err
		}
		a.gz = gz
	}
	return a, nil
}

func (a *staticAsset) serve(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", a.ctype)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Vary", "Accept-Encoding")
	body, etag := a.body, a.etag
	if a.gz != nil && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		body, etag = a.gz, a.etag+"-gz"
		h.Set("Content-Encoding", "gzip")
	}
	h.Set("ETag", `"`+etag+`"`)
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(body))
}
