package profile

import "strings"

var numerals = strings.NewReplacer(
	"〇", "0", "零", "0", "一", "1", "二", "2", "三", "3", "四", "4", "五", "5", "六", "6", "七", "7", "八", "8", "九", "9",
	"０", "0", "１", "1", "２", "2", "３", "3", "４", "4", "５", "5", "６", "6", "７", "7", "８", "8", "９", "9",
)

// StatesBirthYear reports whether someone's own words contain a birthdate's year in full.
// A birthdate is saved once and can't be changed, and it comes to us through a model reading
// their message, so it needs more than the model's word for it. Two-digit years don't count:
// "95-05-05" could be read either way.
func StatesBirthYear(text, birthdate string) bool {
	if len(birthdate) < 4 {
		return false
	}
	year := birthdate[:4]
	for _, run := range strings.FieldsFunc(numerals.Replace(text), func(r rune) bool { return r < '0' || r > '9' }) {
		// 19950312 and 03121995 are one run of digits.
		if run == year || (len(run) == 8 && (strings.HasPrefix(run, year) || strings.HasSuffix(run, year))) {
			return true
		}
	}
	return false
}
