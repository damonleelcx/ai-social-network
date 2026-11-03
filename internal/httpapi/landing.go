package httpapi

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed landing
var landingFiles embed.FS

// Spelled out because the distroless image has no MIME database.
var landingTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".woff2": "font/woff2",
}

type landingAsset struct {
	body, gz []byte
	ctype    string
	etag     string
}

// landing serves the marketing site out of the binary. Text is gzipped once, up front:
// three.js alone is half a megabyte, and it's a page people open on their phones.
var landing = newLanding()

func newLanding() http.Handler {
	assets := map[string]*landingAsset{}
	err := fs.WalkDir(landingFiles, "landing", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ctype, ok := landingTypes[path.Ext(name)]
		if !ok {
			return nil // licences and notes ship in the repo, not on the site
		}
		body, err := landingFiles.ReadFile(name)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(body)
		a := &landingAsset{body: body, ctype: ctype, etag: hex.EncodeToString(sum[:8])}
		if strings.HasPrefix(ctype, "text/") || ctype == "image/svg+xml" {
			var buf bytes.Buffer
			gz, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
			if _, err := gz.Write(body); err != nil {
				return err
			}
			if err := gz.Close(); err != nil {
				return err
			}
			a.gz = buf.Bytes()
		}
		assets["/"+name] = a
		return nil
	})
	if err != nil {
		panic(err) // embedded at build time, so this can only be a bad build
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path
		if name == "/" {
			name = "/landing/index.html"
		}
		a, ok := assets[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		h := w.Header()
		h.Set("Content-Type", a.ctype)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Vary", "Accept-Encoding")
		// File names aren't hashed, and the page, its styles and its scripts have to change
		// together, so those revalidate every time. An unchanged file costs a 304.
		if strings.HasPrefix(a.ctype, "text/") {
			h.Set("Cache-Control", "no-cache")
		} else {
			h.Set("Cache-Control", "public, max-age=3600")
		}
		body, etag := a.body, a.etag
		if a.gz != nil && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			body, etag = a.gz, a.etag+"-gz"
			h.Set("Content-Encoding", "gzip")
		}
		h.Set("ETag", `"`+etag+`"`)
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(body))
	})
}
