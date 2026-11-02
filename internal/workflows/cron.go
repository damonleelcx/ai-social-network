package workflows

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/damonleelcx/ai-social-network/internal/dating"
	"github.com/damonleelcx/ai-social-network/internal/hidden"
	"github.com/damonleelcx/ai-social-network/internal/notify"
)

// Cron is the set of wake-ups. It runs about once a minute on the scheduler
// leader. Every action is keyed or guarded by a flag in the row, so a restart in
// the middle just does the remainder on the next tick.
func Cron(ctx context.Context) {
	for name, f := range map[string]func(context.Context) error{
		"expire": d.Dating.ExpireProposals, "search": dailySearch, "dates": dateTimers, "debriefs-open": unfinishedDebriefs, "stuck": stuckGoals,
		"debriefs": staleDebriefs, "stalled": stalledChats, "taste": tasteRefresh, "inactive": inactiveUsers,
	} {
		if err := f(ctx); err != nil && ctx.Err() == nil {
			slog.Error("cron", "job", name, "err", err)
		}
	}
}

func userIDs(ctx context.Context, q string, a ...any) ([]string, error) {
	rows, err := d.Pool.Query(ctx, q, a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// dailySearch: active people without a search in the last 20 hours. Bounded per
// tick so a big backlog spreads out rather than spiking model spend.
func dailySearch(ctx context.Context) error {
	ids, err := userIDs(ctx, `SELECT p.user_id FROM profiles p JOIN users u ON u.id=p.user_id
		WHERE p.status='active' AND u.deleted_at IS NULL AND p.last_active_at > now()-interval '14 days'
		AND NOT EXISTS (SELECT 1 FROM goals g WHERE g.user_id=p.user_id AND g.skill='match-search' AND g.created_at > now()-interval '20 hours')
		LIMIT 20`)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := StartSearch(ctx, id, false); err != nil {
			return err
		}
	}
	return nil
}

func dateTimers(ctx context.Context) error {
	rows, err := d.Pool.Query(ctx, `SELECT d.id, d.scheduled_at, d.reminded_24, d.reminded_2, d.checkin_sent, m.user_a, m.user_b FROM date_plans d JOIN matches m ON m.id=d.match_id
		WHERE d.status='confirmed' AND d.scheduled_at < now() + interval '24 hours'`)
	if err != nil {
		return err
	}
	type due struct {
		id          string
		at          time.Time
		r24, r2, ci bool
		a, b        string
	}
	var list []due
	for rows.Next() {
		var x due
		if err := rows.Scan(&x.id, &x.at, &x.r24, &x.r2, &x.ci, &x.a, &x.b); err != nil {
			rows.Close()
			return err
		}
		list = append(list, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	now := time.Now()
	for _, x := range list {
		people := []string{x.a, x.b}
		data := map[string]any{"screen": "date", "id": x.id}
		if !x.r24 && x.at.After(now) {
			for _, u := range people {
				if err := notify.Send(ctx, d.Pool, u, "reminder", "Your date is tomorrow", "Tell a friend your plans, and have fun.", data, fmt.Sprintf("remind24:%s:%d:%s", x.id, x.at.Unix(), u)); err != nil {
					return err
				}
			}
			if _, err := d.Pool.Exec(ctx, `UPDATE date_plans SET reminded_24=true WHERE id=$1`, x.id); err != nil {
				return err
			}
		}
		if !x.r2 && x.at.After(now) && x.at.Before(now.Add(2*time.Hour)) {
			for _, u := range people {
				if err := notify.Send(ctx, d.Pool, u, "reminder", "Your date is coming up", "It's almost time.", data, fmt.Sprintf("remind2:%s:%d:%s", x.id, x.at.Unix(), u)); err != nil {
					return err
				}
			}
			if _, err := d.Pool.Exec(ctx, `UPDATE date_plans SET reminded_2=true WHERE id=$1`, x.id); err != nil {
				return err
			}
		}
		if !x.ci && now.After(x.at.Add(time.Hour)) {
			for _, u := range people {
				if err := notify.Send(ctx, d.Pool, u, "safety", "Everything okay?", "Just checking in. If anything feels off, open SynAgent and tell Suki.", data, fmt.Sprintf("checkin:%s:%d:%s", x.id, x.at.Unix(), u)); err != nil {
					return err
				}
			}
			if _, err := d.Pool.Exec(ctx, `UPDATE date_plans SET checkin_sent=true WHERE id=$1`, x.id); err != nil {
				return err
			}
		}
		if now.After(x.at.Add(d.DebriefDelay)) {
			if err := OpenDebriefs(ctx, x.id); err != nil {
				return err
			}
		}
	}
	return nil
}

// OpenDebriefs completes a date and has Suki ask each person how it went, unless
// they switched debriefs off. Keys make a second call harmless.
func OpenDebriefs(ctx context.Context, dateID string) error {
	pairs, err := d.Dating.CompleteDate(ctx, dateID)
	if err != nil {
		return err
	}
	for _, p := range pairs {
		st := dating.GetSettings(ctx, d.Pool, p[0])
		if st["debrief_enabled"] == false {
			if _, _, err := d.Dating.CloseDebrief(ctx, p[0], true); err != nil {
				return err
			}
			continue
		}
		if err := EnqueueSystemTurn(ctx, p[0], "debrief_start", "Their date just happened. Ask them, warmly and in one short question, how it went. Don't ask for a rating. Offer quick replies such as Loved it, It was okay, Not for me.", "debrief:"+dateID+":"+p[0]); err != nil {
			return err
		}
		if err := notify.Send(ctx, d.Pool, p[0], "debrief", "How did it go?", "Suki would love to hear about it.", map[string]any{"screen": "agent"}, "debrief:"+dateID+":"+p[0]); err != nil {
			return err
		}
	}
	// Set last: a crash before here means the next tick finishes the job, and every
	// step above is keyed.
	_, err = d.Pool.Exec(ctx, `UPDATE date_plans SET debrief_sent=true WHERE id=$1 AND status='completed'`, dateID)
	return err
}

// unfinishedDebriefs picks up dates completed by a run that crashed before asking.
func unfinishedDebriefs(ctx context.Context) error {
	ids, err := userIDs(ctx, `SELECT id FROM date_plans WHERE status='completed' AND NOT debrief_sent LIMIT 20`)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := OpenDebriefs(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// stuckGoals handles a goal parked for a budget or an ambiguous side effect, which
// doesn't pass through planHook. The person shouldn't wait forever on it: a photo goes
// to a human, and a date search says it failed.
func stuckGoals(ctx context.Context) error {
	ids, err := userIDs(ctx, `SELECT g.id FROM goals g WHERE g.status='needs_attention' AND g.updated_at < now()-interval '5 minutes' AND (
		(g.skill='photo-review' AND EXISTS (SELECT 1 FROM photos p WHERE p.id::text=g.subject_id AND p.status='pending')) OR
		(g.skill='date-plan' AND EXISTS (SELECT 1 FROM date_plans x WHERE x.id::text=g.subject_id AND x.status='searching'))) LIMIT 20`)
	if err != nil {
		return err
	}
	for _, id := range ids {
		g, err := d.Store.Goal(ctx, id)
		if err != nil {
			return err
		}
		if err := onGoalFailed(ctx, g, g.AttentionReason); err != nil {
			return err
		}
	}
	return nil
}

// EnqueueSystemTurn asks Suki to speak up without a user message.
func EnqueueSystemTurn(ctx context.Context, userID, intent, prompt, key string) error {
	conv, err := dating.ConversationID(ctx, d.Pool, userID)
	if err != nil {
		return err
	}
	_, err = d.Pool.Exec(ctx, `INSERT INTO agent_turns (user_id, conversation_id, kind, prompt, intent, key) VALUES ($1,$2,'system',$3,$4,$5) ON CONFLICT (key) DO NOTHING`,
		userID, conv, prompt, intent, key)
	if err == nil {
		_, _ = d.Pool.Exec(ctx, `SELECT pg_notify('syn_turn', $1)`, userID)
	}
	return err
}

// staleDebriefs: someone talked and wandered off, so analyse what they said. Nobody
// answered after a week, so drop it without judging anyone.
func staleDebriefs(ctx context.Context) error {
	rows, err := d.Pool.Query(ctx, `SELECT user_id, jsonb_array_length(notes) FROM debriefs
		WHERE status='open' AND ((jsonb_array_length(notes) > 0 AND created_at < now()-interval '24 hours') OR created_at < now()-interval '7 days') LIMIT 50`)
	if err != nil {
		return err
	}
	var users []string
	for rows.Next() {
		var u string
		var n int
		if err := rows.Scan(&u, &n); err != nil {
			rows.Close()
			return err
		}
		users = append(users, u)
	}
	rows.Close()
	for _, u := range users {
		dateID, analyzable, err := d.Dating.CloseDebrief(ctx, u, false)
		if err != nil {
			return err
		}
		if analyzable {
			if err := StartDebriefAnalysis(ctx, u, dateID); err != nil {
				return err
			}
		}
	}
	return nil
}

// stalledChats: a conversation that went quiet for 48 hours gets one nudge. A
// person who set Suki's proactivity to low isn't nudged.
func stalledChats(ctx context.Context) error {
	rows, err := d.Pool.Query(ctx, `UPDATE matches SET stall_nudged_at=now() WHERE id IN (
		SELECT id FROM matches WHERE status='active' AND stage IN ('icebreaker','getting_to_know','date_ready')
		AND coalesce(last_message_at, created_at) < now()-interval '48 hours' AND stall_nudged_at IS NULL LIMIT 20)
		RETURNING id, user_a, user_b`)
	if err != nil {
		return err
	}
	type m struct{ id, a, b string }
	var list []m
	for rows.Next() {
		var x m
		if err := rows.Scan(&x.id, &x.a, &x.b); err != nil {
			rows.Close()
			return err
		}
		list = append(list, x)
	}
	rows.Close()
	for _, x := range list {
		for _, u := range []string{x.a, x.b} {
			if dating.GetSettings(ctx, d.Pool, u)["suki_proactive"] == "low" {
				continue
			}
			if err := StartCoach(ctx, x.id, u, "stalled"); err != nil {
				return err
			}
		}
	}
	return nil
}

func tasteRefresh(ctx context.Context) error {
	ids, err := hidden.UsersDueForNotes(ctx, d.Pool, 10, 10)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := start(ctx, id, "taste-update", "taste", "Learn what they like", "Write up what this person's behaviour suggests they like.", ""); err != nil {
			return fmt.Errorf("taste-update: %w", err)
		}
	}
	return nil
}

// inactiveUsers stop being proposed through the eligibility filter's last-active
// window; open proposals to them simply expire. Nothing to change here beyond
// closing their open proposals early, so nobody waits on someone who has gone.
func inactiveUsers(ctx context.Context) error {
	_, err := d.Pool.Exec(ctx, `UPDATE proposals SET status='expired' WHERE status='open' AND (
		user_a IN (SELECT user_id FROM profiles WHERE last_active_at < now()-interval '30 days') OR
		user_b IN (SELECT user_id FROM profiles WHERE last_active_at < now()-interval '30 days'))`)
	return err
}
