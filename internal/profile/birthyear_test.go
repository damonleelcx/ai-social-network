package profile

import (
	"testing"
	"time"
)

func TestStatesBirthYear(t *testing.T) {
	for _, c := range []struct {
		text, date string
		want       bool
	}{
		{"I was born on March 2, 1995", "1995-03-02", true},
		{"1995-03-02", "1995-03-02", true},
		{"3/2/1995", "1995-03-02", true},
		{"19950302", "1995-03-02", true},
		{"03021995", "1995-03-02", true},
		{"我是1995年3月2日出生的", "1995-03-02", true},
		{"一九九五年三月二日", "1995-03-02", true},
		{"１９９５年３月２日", "1995-03-02", true},
		{"1995년 3월 2일", "1995-03-02", true},
		// Year first, including a two-digit year: 95/10/20 is 20 October 1995.
		{"95/10/20", "1995-10-20", true},
		{"95-10-20", "1995-10-20", true},
		{"95.10.20", "1995-10-20", true},
		{"95年10月20日", "1995-10-20", true},
		{"95年10月20号", "1995-10-20", true},
		{"９５/１０/２０", "1995-10-20", true},
		{"我的生日是95/10/20", "1995-10-20", true},
		{"95/10/20", "1995-03-02", false},
		{"95-05-05", "1995-05-05", true},
		// The bug: they said 1995 and the model wrote 2005.
		{"I was born on March 2, 1995", "2005-03-02", false},
		{"一九九五年三月二日", "2005-03-02", false},
		// Month-first, or a two-digit year read as the wrong century, isn't enough.
		{"3/2/95", "1995-03-02", false},
		{"95-05-05", "2005-05-05", false},
		{"born in 119950", "1995-03-02", false},
		{"", "1995-03-02", false},
		{"March 2, 1995", "", false},
	} {
		if got := StatesBirthYear(c.text, c.date); got != c.want {
			t.Errorf("StatesBirthYear(%q, %q) = %v, want %v", c.text, c.date, got, c.want)
		}
	}
}

func TestStatedBirthdate(t *testing.T) {
	now := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		text, want string
		ok         bool
	}{
		{"95/10/20", "1995-10-20", true},
		{"08/01/01", "2008-01-01", true},
		{"08/11/24", "", false}, // 2008-11-24 is still 17; 1908 is too old
		{"95/10/20 and 96/01/01", "", false},
		{"10/20/95", "", false},
	} {
		got, ok := StatedBirthdate(c.text, now)
		if ok != c.ok || got != c.want {
			t.Errorf("StatedBirthdate(%q) = %q, %v, want %q, %v", c.text, got, ok, c.want, c.ok)
		}
	}
}
