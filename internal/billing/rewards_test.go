package billing

import (
	"testing"
	"time"
)

func TestChinaDay(t *testing.T) {
	for _, c := range []struct {
		at   time.Time
		want string
	}{
		{time.Date(2026, 11, 8, 15, 59, 59, 0, time.UTC), "2026-11-08"},
		{time.Date(2026, 11, 8, 16, 0, 0, 0, time.UTC), "2026-11-09"},
		{time.Date(2026, 12, 31, 16, 0, 0, 0, time.UTC), "2027-01-01"},
		{time.Date(2026, 11, 8, 0, 30, 0, 0, time.FixedZone("PST", -8*3600)), "2026-11-08"},
		// Midsummer: no daylight saving shift in China.
		{time.Date(2026, 7, 1, 15, 59, 59, 0, time.UTC), "2026-07-01"},
	} {
		if got := ChinaDay(c.at).Format(time.DateOnly); got != c.want {
			t.Errorf("%s: %s, want %s", c.at, got, c.want)
		}
	}
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		at := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
		if _, off := at.In(loc).Zone(); off != 8*3600 {
			t.Errorf("Asia/Shanghai offset is %d", off)
		}
	}
}

func TestDefaultRewardsAreSmall(t *testing.T) {
	r := DefaultRewards()
	smallest := Packages[0].Credits
	if r.CheckInCredits <= 0 || r.CheckInCredits*10 > smallest {
		t.Errorf("check-in %d is not small next to a %d-credit pack", r.CheckInCredits, smallest)
	}
	if r.InviterCredits <= 0 || r.InviteeCredits <= 0 || r.InviterCredits > smallest/2 || r.InviteeCredits > smallest/2 {
		t.Errorf("invite rewards %d/%d out of proportion to a %d-credit pack", r.InviterCredits, r.InviteeCredits, smallest)
	}
	if r.InviteDailyCap <= 0 || r.InviteTotalCap < r.InviteDailyCap {
		t.Errorf("caps: %+v", r)
	}
	s := &Service{}
	if s.rewards() != r {
		t.Error("nil Rewards should mean the defaults")
	}
}
