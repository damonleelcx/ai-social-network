package e2e

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/damonleelcx/ai-social-network/internal/billing"
)

// A daily check-in pays once per calendar day in China (Asia/Shanghai), however often or
// however concurrently it's asked for.
func TestDailyCheckIn(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	credits := billing.DefaultRewards().CheckInCredits
	var mu sync.Mutex
	// 07:59:59 UTC on 8 November is 15:59:59 in Shanghai.
	clock := time.Date(2026, 11, 8, 7, 59, 59, 0, time.UTC)
	setClock := func(t time.Time) { mu.Lock(); clock = t; mu.Unlock() }
	e.billing.Now = func() time.Time { mu.Lock(); defer mu.Unlock(); return clock }
	t.Cleanup(func() { e.billing.Now = nil })

	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()
	var st billing.CheckIn
	a.do("GET", "/v1/checkin", nil, 200, &st)
	if !st.Enabled || st.CheckedIn || st.Day != "2026-11-08" || st.Credits != credits || st.Balance != 0 {
		t.Fatalf("before: %+v", st)
	}
	a.do("POST", "/v1/checkin", nil, 200, &st)
	if !st.CheckedIn || st.Earned != credits || st.Balance != credits {
		t.Fatalf("first check-in: %+v", st)
	}
	// A double tap: done already, nothing more.
	a.do("POST", "/v1/checkin", nil, 200, &st)
	if !st.CheckedIn || st.Earned != 0 || st.Balance != credits {
		t.Fatalf("second check-in: %+v", st)
	}

	// 23:59:59 in Shanghai is still the same day, though it's a different hour in UTC.
	setClock(time.Date(2026, 11, 8, 15, 59, 59, 0, time.UTC))
	a.do("POST", "/v1/checkin", nil, 200, &st)
	if st.Earned != 0 || st.Day != "2026-11-08" {
		t.Fatalf("late the same China day: %+v", st)
	}
	// One second later it's 9 November in Shanghai (still 8 November in UTC): a new day.
	setClock(time.Date(2026, 11, 8, 16, 0, 0, 0, time.UTC))
	a.do("GET", "/v1/checkin", nil, 200, &st)
	if st.CheckedIn || st.Day != "2026-11-09" {
		t.Fatalf("new China day, before: %+v", st)
	}
	a.do("POST", "/v1/checkin", nil, 200, &st)
	if st.Earned != credits || st.Balance != 2*credits || st.Day != "2026-11-09" {
		t.Fatalf("new China day: %+v", st)
	}

	// Many devices at once on the next day: one payment.
	setClock(time.Date(2026, 11, 10, 1, 0, 0, 0, time.UTC))
	var wg sync.WaitGroup
	var paid atomic.Int32
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := e.billing.DoCheckIn(ctx, a.id)
			if err != nil {
				t.Error(err)
				return
			}
			if got.Earned > 0 {
				paid.Add(1)
			}
		}()
	}
	wg.Wait()
	if paid.Load() != 1 || e.balance(t, a.id) != 3*credits || e.ledgerRows(t, a.id, "checkin") != 3 {
		t.Fatalf("concurrent check-ins paid %d times, balance %d", paid.Load(), e.balance(t, a.id))
	}
	if e.ledgerRows(t, a.id, "purchase") != 0 {
		t.Fatal("check-in recorded as a purchase")
	}

	// Turned off, a check-in pays nothing and says so.
	off := billing.DefaultRewards()
	off.CheckInCredits = 0
	e.billing.Rewards = &off
	t.Cleanup(func() { e.billing.Rewards = nil })
	setClock(time.Date(2026, 11, 12, 1, 0, 0, 0, time.UTC))
	a.do("POST", "/v1/checkin", nil, 200, &st)
	if st.Enabled || st.Earned != 0 || e.balance(t, a.id) != 3*credits {
		t.Fatalf("check-in while off: %+v", st)
	}
}
