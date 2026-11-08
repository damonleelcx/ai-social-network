package billing

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Rewards are the credits people can earn without buying any. Each lands in credit_ledger
// with its own reason ('referral', 'invited', 'checkin'), never 'purchase'. A zero amount
// turns that reward off.
type Rewards struct {
	// InviterCredits and InviteeCredits are paid once, to each side, when an invited
	// friend's new account goes live. Sharing on its own never earns anything.
	InviterCredits int
	InviteeCredits int
	// InviteDailyCap and InviteTotalCap bound how many friends one person is paid for, per
	// China calendar day and ever. Past either, the friend still counts as joined, and
	// neither side is paid for that friend.
	InviteDailyCap int
	InviteTotalCap int
	// CheckInCredits is paid at most once per China calendar day.
	CheckInCredits int
}

// DefaultRewards are small next to the smallest pack (100 credits).
func DefaultRewards() Rewards {
	return Rewards{InviterCredits: 30, InviteeCredits: 30, InviteDailyCap: 5, InviteTotalCap: 50, CheckInCredits: 5}
}

func (s *Service) rewards() Rewards {
	if s.Rewards == nil {
		return DefaultRewards()
	}
	return *s.Rewards
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// China is Asia/Shanghai. China has had no daylight saving since 1991, so a fixed +8 is the
// same clock without depending on the image shipping a time zone database.
var China = time.FixedZone("Asia/Shanghai", 8*60*60)

// ChinaDay is the calendar day it is in China at t, as midnight there.
func ChinaDay(t time.Time) time.Time {
	y, m, d := t.In(China).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, China)
}

// CheckIn is today's check-in, as someone sees it.
type CheckIn struct {
	Enabled bool `json:"enabled"`
	// Day is today in China, YYYY-MM-DD.
	Day       string `json:"day"`
	CheckedIn bool   `json:"checked_in"`
	// Credits is what a check-in pays.
	Credits int `json:"credits"`
	// Earned is what this request paid: Credits the first time today, otherwise 0.
	Earned  int `json:"earned"`
	Balance int `json:"balance"`
}

func (s *Service) CheckInStatus(ctx context.Context, userID string) (*CheckIn, error) {
	day := ChinaDay(s.now())
	r := s.rewards()
	st := &CheckIn{Enabled: r.CheckInCredits > 0, Day: day.Format(time.DateOnly), Credits: r.CheckInCredits}
	err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM daily_checkins WHERE user_id=$1 AND day=$2),
		coalesce((SELECT balance FROM credit_balances WHERE user_id=$1),0)`, userID, day.Format(time.DateOnly)).Scan(&st.CheckedIn, &st.Balance)
	if err != nil {
		return nil, err
	}
	return st, nil
}

// DoCheckIn pays today's check-in once. Repeating it, from a double tap or two devices at
// once, pays nothing more: the second insert waits on the first and then finds the row.
func (s *Service) DoCheckIn(ctx context.Context, userID string) (*CheckIn, error) {
	r := s.rewards()
	if r.CheckInCredits <= 0 {
		return s.CheckInStatus(ctx, userID)
	}
	day := ChinaDay(s.now()).Format(time.DateOnly)
	earned := 0
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `INSERT INTO daily_checkins (user_id, day, credits) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, userID, day, r.CheckInCredits)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		earned = r.CheckInCredits
		return addCredits(ctx, tx, userID, r.CheckInCredits, "checkin", "checkin:"+userID+":"+day)
	})
	if err != nil {
		return nil, err
	}
	st, err := s.CheckInStatus(ctx, userID)
	if err != nil {
		return nil, err
	}
	// Status reads the clock again; keep the answer about the day that was checked in.
	st.Day, st.CheckedIn, st.Earned = day, true, earned
	return st, nil
}
