package profile

import "testing"

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
		// The bug: they said 1995 and the model wrote 2005.
		{"I was born on March 2, 1995", "2005-03-02", false},
		{"一九九五年三月二日", "2005-03-02", false},
		// A short year or a matching day isn't enough.
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
