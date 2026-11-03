package i18n

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestFallsBackToEnglish(t *testing.T) {
	if got := T("fr", "It's a match!"); got != "It's a match!" {
		t.Errorf("unknown language: %q", got)
	}
	if got := T("zh", "A model wrote this"); got != "A model wrote this" {
		t.Errorf("unknown text: %q", got)
	}
	if got := T("en", "You and {name} both said yes.", "name", "Leo"); got != "You and Leo both said yes." {
		t.Errorf("english: %q", got)
	}
}

func TestTranslates(t *testing.T) {
	if got := T("zh", "You and {name} both said yes.", "name", "Leo"); got != "你和 Leo 都说好了。" {
		t.Errorf("zh: %q", got)
	}
	if got := T("es", "It's a match!"); got != "¡Tienes un match!" {
		t.Errorf("es: %q", got)
	}
}

var placeholder = regexp.MustCompile(`\{[a-z]+\}`)

func holes(s string) string {
	h := placeholder.FindAllString(s, -1)
	sort.Strings(h)
	return strings.Join(h, " ")
}

// A translation that loses a placeholder would show a blank where a name belongs.
func TestCatalogIsComplete(t *testing.T) {
	for en, row := range catalog {
		for lang, c := range column {
			if strings.TrimSpace(row[c]) == "" {
				t.Errorf("%s missing: %q", lang, en)
			}
			if holes(row[c]) != holes(en) {
				t.Errorf("%s placeholders differ: %q -> %q", lang, en, row[c])
			}
		}
	}
}
