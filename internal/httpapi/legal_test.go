package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicPages(t *testing.T) {
	for _, tc := range []struct{ title, body, want string }{
		{"Terms of Use", termsHTML, "at least 18 years old"},
		{"Privacy Policy", privacyHTML, "Beijing, China"},
		{"Delete your SynAgent account", deleteHTML, "up to 30 days"},
	} {
		w := httptest.NewRecorder()
		legalPage(tc.title, tc.body)(w, httptest.NewRequest("GET", "/", nil))
		page := w.Body.String()
		if w.Code != 200 || !strings.Contains(page, "<title>"+tc.title+"</title>") || !strings.Contains(page, tc.want) {
			t.Errorf("%s: %d, want %q in the page", tc.title, w.Code, tc.want)
		}
		if strings.Contains(strings.ToLower(page), "draft") {
			t.Errorf("%s still reads as a draft", tc.title)
		}
	}
}
