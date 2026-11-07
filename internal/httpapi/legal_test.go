package httpapi

import (
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestPublicPages(t *testing.T) {
	for _, tc := range []struct{ page, title, want string }{
		{"terms", "Terms of Use", "at least 18 years old"},
		{"privacy", "Privacy Policy", "Beijing, China"},
		{"delete-account", "Delete your SynAgent account", "up to 30 days"},
		{"contact", "Contact us", "support@heros-agent.space"},
		{"child-safety", "Child safety standards", "National Center for Missing &amp; Exploited Children"},
	} {
		w := httptest.NewRecorder()
		legalPage(tc.page, false)(w, httptest.NewRequest("GET", "/", nil))
		page := w.Body.String()
		if w.Code != 200 || !strings.Contains(page, "<title>"+tc.title+"</title>") || !strings.Contains(page, tc.want) {
			t.Errorf("%s: %d, want %q in the page", tc.page, w.Code, tc.want)
		}
		if strings.Contains(strings.ToLower(page), "draft") {
			t.Errorf("%s still reads as a draft", tc.page)
		}
	}
}

func TestLegalLanguagePicked(t *testing.T) {
	for _, tc := range []struct{ query, accept, want string }{
		{"", "", "en"},
		{"?lang=ja", "", "ja"},
		{"?lang=ja", "ko", "ja"},
		{"", "zh-CN,zh;q=0.9,en;q=0.8", "zh"},
		{"", "fr-FR,es;q=0.9", "es"},
		{"?lang=xx", "fr", "en"},
	} {
		r := httptest.NewRequest("GET", "/"+tc.query, nil)
		r.Header.Set("Accept-Language", tc.accept)
		w := httptest.NewRecorder()
		legalPage("terms", false)(w, r)
		if !strings.Contains(w.Body.String(), `<html lang="`+tc.want+`">`) {
			t.Errorf("%q %q: want %s", tc.query, tc.accept, tc.want)
		}
	}
}

var (
	legalTag  = regexp.MustCompile(`<(h1|h2|p|ul|li|table|tr|th|td|strong)[ >]`)
	legalAttr = regexp.MustCompile(`(?:href|id)="[^"]*"`)
)

// A translation has to say the same things in the same places, and link to the same ones.
func TestLegalTranslationsMatchEnglish(t *testing.T) {
	for _, page := range []string{"terms", "privacy", "delete-account", "contact", "child-safety", "open"} {
		en, err := legalFiles.ReadFile("legal/en/" + page + ".html")
		if err != nil {
			t.Fatal(err)
		}
		wantTags := strings.Join(legalTag.FindAllString(string(en), -1), "")
		wantAttrs := strings.Join(legalAttr.FindAllString(string(en), -1), " ")
		for _, l := range legalLanguages[1:] {
			text, err := legalFiles.ReadFile("legal/" + l.code + "/" + page + ".html")
			if err != nil {
				t.Errorf("%s %s: %v", l.code, page, err)
				continue
			}
			if got := strings.Join(legalTag.FindAllString(string(text), -1), ""); got != wantTags {
				t.Errorf("%s %s: the structure differs from English", l.code, page)
			}
			if got := strings.Join(legalAttr.FindAllString(string(text), -1), " "); got != wantAttrs {
				t.Errorf("%s %s: the links differ from English", l.code, page)
			}
			w := httptest.NewRecorder()
			legalPage(page, false)(w, httptest.NewRequest("GET", "/?lang="+l.code, nil))
			if !strings.Contains(w.Body.String(), l.note) {
				t.Errorf("%s %s: no note that English controls", l.code, page)
			}
		}
	}
}
