package i18n

import (
	"regexp"
	"testing"
)

// The China edition never frames the app as dating, in any language.
var chinaBanned = regexp.MustCompile(`(?i)\bdat(e|es|ed|ing)\b|\bmatch(es|ed)?\b|matchmak|swipe|wingman|winggirl|约会|红娘|相亲|恋爱|僚机|데이트|매치|매칭|연애|소개팅|デート|マッチ|恋愛|婚活|仲人|\bcitas?\b|emparej|\bpareja|casamenter`)

func TestChinaEditionWording(t *testing.T) {
	chinaEdition = true
	defer func() { chinaEdition = false }()
	for english := range catalog {
		for _, lang := range []string{"en", "zh", "ko", "ja", "es"} {
			if got := T(lang, english); chinaBanned.MatchString(got) {
				t.Errorf("%s: %q says %q", lang, english, got)
			}
		}
	}
	for english, row := range chinaText {
		if _, ok := catalog[english]; !ok {
			t.Errorf("China wording for a sentence the catalog doesn't have: %q", english)
		}
		if row["en"] == "" {
			t.Errorf("no English China wording for %q", english)
		}
	}
}
