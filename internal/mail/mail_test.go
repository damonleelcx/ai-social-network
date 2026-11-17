package mail

import (
	"html"
	"strings"
	"testing"
)

func TestPageIsSukiBranded(t *testing.T) {
	for _, lang := range []string{"en", "zh", "ko", "ja", "fr"} {
		page := Page(lang, "It's a match!", "You both said yes.", "Open Suki", "https://example.test/open", "foot")
		for _, want := range []string{">SUKI<", "— Suki", "Suki"} {
			if !strings.Contains(page, want) {
				t.Errorf("%s: missing %q", lang, want)
			}
		}
		for _, stale := range []string{"PLAY WITH AGENTS", "Play money", "葵", "아오이", "あおい"} {
			if strings.Contains(page, stale) {
				t.Errorf("%s: still says %q", lang, stale)
			}
		}
		if !strings.Contains(page, html.EscapeString(footerNote[htmlLang(lang)])) {
			t.Errorf("%s: footer missing", lang)
		}
	}
}
