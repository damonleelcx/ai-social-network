package profile

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var numerals = strings.NewReplacer(
	"〇", "0", "零", "0", "一", "1", "二", "2", "三", "3", "四", "4", "五", "5", "六", "6", "七", "7", "八", "8", "九", "9",
	"０", "0", "１", "1", "２", "2", "３", "3", "４", "4", "５", "5", "６", "6", "７", "7", "８", "8", "９", "9",
)

// Year first, then month, then day: 95/10/20, 95-10-20, 95.10.20, 95年10月20日.
var statedDate = regexp.MustCompile(`(?:^|[^\d])(\d{2}|\d{4})\s*[/\-\.年／．]\s*(\d{1,2})\s*[/\-\.月／．]\s*(\d{1,2})\s*[日号]?`)

// StatesBirthYear reports whether someone's own words contain a birthdate's year in full.
// A birthdate is saved once and can't be changed, and it comes to us through a model reading
// their message, so it needs more than the model's word for it. A two-digit year counts only
// as part of a complete year-first date, such as 95/10/20 for 1995-10-20.
func StatesBirthYear(text, birthdate string) bool {
	if len(birthdate) < 4 {
		return false
	}
	if stated, ok := StatedBirthdate(text, time.Now()); ok && stated == birthdate {
		return true
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

// StatedBirthdate reads one year-first date from text. 95/10/20 is 1995-10-20 when that
// is the only century that makes them 18 to 110. Month-first dates are left alone.
func StatedBirthdate(text string, now time.Time) (string, bool) {
	var dates []string
	for _, m := range statedDate.FindAllStringSubmatch(numerals.Replace(text), -1) {
		if d, ok := normalizeStated(m[1], m[2], m[3], now); ok {
			dates = append(dates, d)
		}
	}
	if len(dates) != 1 {
		return "", false
	}
	return dates[0], true
}

func normalizeStated(yearRaw, monthRaw, dayRaw string, now time.Time) (string, bool) {
	month, errM := strconv.Atoi(monthRaw)
	day, errD := strconv.Atoi(dayRaw)
	year, errY := strconv.Atoi(yearRaw)
	if errM != nil || errD != nil || errY != nil || month < 1 || month > 12 || day < 1 || day > 31 {
		return "", false
	}
	if len(yearRaw) == 2 {
		var ok bool
		year, ok = expandBirthYear(year, month, day, now)
		if !ok {
			return "", false
		}
	}
	date := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return "", false
	}
	if age := AgeOn(date, now); age < 18 || age > 110 {
		return "", false
	}
	return date, true
}

// expandBirthYear picks 19xx or 20xx when only one of them is a plausible birthday.
func expandBirthYear(yy, month, day int, now time.Time) (int, bool) {
	var hit int
	n := 0
	for _, y := range []int{1900 + yy, 2000 + yy} {
		date := fmt.Sprintf("%04d-%02d-%02d", y, month, day)
		if _, err := time.Parse("2006-01-02", date); err != nil {
			continue
		}
		if age := AgeOn(date, now); age >= 18 && age <= 110 {
			hit = y
			n++
		}
	}
	if n != 1 {
		return 0, false
	}
	return hit, true
}
