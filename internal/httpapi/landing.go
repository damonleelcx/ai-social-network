package httpapi

import (
	"bytes"
	"compress/gzip"
	"embed"
	"html"
	"io/fs"
	"net/http"
	"path"
	"strings"
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

// landing serves the marketing site out of the binary. Text is gzipped once, up front:
// three.js alone is half a megabyte, and it's a page people open on their phones.
var landing = newLanding("")

// ICPFooter is the mainland China filing notice the home page must show when it is
// served from China: the ICP filing number, linked to the MIIT lookup, and the value-added
// telecom licence number. Empty values are left out; both empty means no notice.
func ICPFooter(beian, licence string) string {
	var parts []string
	if beian != "" {
		parts = append(parts, `<a href="https://beian.miit.gov.cn/" target="_blank" rel="noopener">`+html.EscapeString(beian)+`</a>`)
	}
	if licence != "" {
		parts = append(parts, "增值电信业务经营许可证："+html.EscapeString(licence))
	}
	if len(parts) == 0 {
		return ""
	}
	return `<p class="icp" lang="zh-CN">` + strings.Join(parts, " · ") + "</p>\n"
}

// newLanding builds the site. A non-empty footer is added inside the home page's footer.
func newLanding(footer string) http.Handler {
	assets := map[string]*staticAsset{}
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
		if footer != "" && name == "landing/index.html" {
			body = bytes.Replace(body, []byte("</footer>"), []byte(footer+"</footer>"), 1)
		}
		a, err := newStaticAsset(body, ctype)
		if err != nil {
			return err
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
		// File names aren't hashed, and the page, its styles and its scripts have to change
		// together, so those revalidate every time. An unchanged file costs a 304.
		if strings.HasPrefix(a.ctype, "text/") {
			h.Set("Cache-Control", "no-cache")
		} else {
			h.Set("Cache-Control", "public, max-age=3600")
		}
		a.serve(w, r)
	})
}

func gzipped(body []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := gz.Write(body); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
