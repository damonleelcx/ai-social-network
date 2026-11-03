// Package notify tells people something happened: in the app, by push, and for
// the moments worth an email. Notifications say what is waiting, never what
// someone wrote, because a lock screen is not private.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damonleelcx/ai-social-network/internal/i18n"
	"github.com/damonleelcx/ai-social-network/internal/mail"
)

// KindApprovalRequested remains for the engine's approval flow. No dating skill
// parks for approval today: a person's yes is a tap on a card.
const (
	KindApprovalRequested = "approval_requested"
	KindBuildReady        = "approval_requested"
)

// Pref is the settings key that switches a kind of notification off.
var prefFor = map[string]string{
	"proposal": "notify_matches", "match": "notify_matches", "message": "notify_messages", "hint": "notify_suki",
	"date": "notify_dates", "reminder": "notify_dates", "debrief": "notify_dates", "safety": "",
}

// emailKinds are worth an inbox when email is on. Chat messages never are.
var emailKinds = map[string]bool{"match": true, "reminder": true, "date": true}

// Send records a notification once per key, honouring the person's settings. It
// always lands in the in-app list; only push and email are optional.
func Send(ctx context.Context, pool *pgxpool.Pool, userID, kind, title, body string, data map[string]any, key string) error {
	raw, _ := json.Marshal(data)
	var nkey any
	if key != "" {
		nkey = key
	}
	enabled, email := true, false
	var prefs []byte
	var to string
	if err := pool.QueryRow(ctx, `SELECT coalesce(p.data,'{}'), u.email FROM users u LEFT JOIN user_preferences p ON p.user_id=u.id WHERE u.id=$1`, userID).Scan(&prefs, &to); err != nil {
		return err
	}
	settings := map[string]any{}
	_ = json.Unmarshal(prefs, &settings)
	lang, _ := settings["language"].(string)
	title, body = i18n.T(lang, title), i18n.T(lang, body)
	if p := prefFor[kind]; p != "" && settings[p] == false {
		enabled = false
	}
	email = enabled && emailKinds[kind] && settings["notify_email"] != false
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var pushed any
		if !enabled {
			pushed = time.Now()
		}
		tag, err := tx.Exec(ctx, `INSERT INTO notifications (user_id, kind, title, body, data, key, pushed_at) VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (key) DO NOTHING`, userID, kind, title, body, raw, nkey, pushed)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		if email && key != "" {
			params, _ := json.Marshal(map[string]any{"title": title, "body": body})
			if _, err := tx.Exec(ctx, `INSERT INTO outbox (key, kind, user_id, to_email, params) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (key) DO NOTHING`,
				"notify:"+key, kind, userID, to, params); err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `SELECT pg_notify('syn_user', $1)`, userID+"|notification")
		return err
	})
}

// ApprovalRequested keeps the engine's approval flow working: it files an in-app
// notification. See the note on KindApprovalRequested.
func ApprovalRequested(ctx context.Context, tx pgx.Tx, approvalID, kind string) (int64, error) {
	tag, err := tx.Exec(ctx, `INSERT INTO notifications (user_id, kind, title, data, key)
		SELECT a.user_id, $2, 'Suki needs your okay', jsonb_build_object('approval_id', a.id), 'approval:' || a.id
		FROM approvals a WHERE a.id=$1 ON CONFLICT (key) DO NOTHING`, approvalID, kind)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Deliver sends due outbox email. The claim pushes next_attempt_at forward in
// the same statement, so a second scheduler does not pick the row up again until
// the backoff expires. Returns rows sent.
func Deliver(ctx context.Context, pool *pgxpool.Pool, m mail.Mailer, origin string) int {
	rows, err := pool.Query(ctx, `
		UPDATE outbox SET attempts = attempts + 1,
		       next_attempt_at = now() + make_interval(secs => least(3600, 30 * power(2, attempts)))
		WHERE id IN (SELECT id FROM outbox WHERE sent_at IS NULL AND failed_at IS NULL AND next_attempt_at <= now()
		             ORDER BY id LIMIT 20 FOR UPDATE SKIP LOCKED)
		RETURNING id, key, kind, to_email, lang, params, attempts`)
	if err != nil {
		return 0
	}
	type row struct {
		id            int64
		key, kind, to string
		lang          string
		params        map[string]any
		attempts      int
	}
	var rs []row
	for rows.Next() {
		var r row
		var raw []byte
		if rows.Scan(&r.id, &r.key, &r.kind, &r.to, &r.lang, &raw, &r.attempts) == nil {
			_ = json.Unmarshal(raw, &r.params)
			rs = append(rs, r)
		}
	}
	rows.Close()
	sent := 0
	for _, r := range rs {
		msg, err := Render(r.kind, r.lang, r.params, origin)
		if err == nil {
			msg.To = r.to
			msg.MessageID = "outbox-" + r.key
			_, err = m.Send(ctx, msg)
		}
		c := context.WithoutCancel(ctx)
		if err == nil {
			_, _ = pool.Exec(c, `UPDATE outbox SET sent_at = now(), error = '' WHERE id = $1`, r.id)
			sent++
			continue
		}
		slog.Warn("outbox send failed", "key", r.key, "attempt", r.attempts, "err", err)
		if r.attempts >= 6 {
			_, _ = pool.Exec(c, `UPDATE outbox SET failed_at = now(), error = $2 WHERE id = $1`, r.id, err.Error())
		} else {
			_, _ = pool.Exec(c, `UPDATE outbox SET error = $2 WHERE id = $1`, r.id, err.Error())
		}
	}
	return sent
}

// Render turns an outbox row into an email.
func Render(kind, lang string, p map[string]any, origin string) (mail.Message, error) {
	title, _ := p["title"].(string)
	body, _ := p["body"].(string)
	if title == "" {
		return mail.Message{}, fmt.Errorf("outbox %s without a title", kind)
	}
	foot := i18n.T(lang, "You can turn these emails off in Settings > Notifications.")
	text := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\nSuki", title, body, i18n.T(lang, "Open SynAgent to see it."), foot)
	return mail.Message{Subject: title + " · SynAgent", Text: text,
		HTML: mail.Page(lang, title, body, i18n.T(lang, "Open SynAgent"), origin+"/open", foot)}, nil
}

// PushDue sends unsent notifications through Expo's push service. A token that
// Expo says is dead is removed; a failure leaves the row for the next tick, and
// anything older than an hour is dropped, since a stale buzz helps no one.
func PushDue(ctx context.Context, pool *pgxpool.Pool, url string, client *http.Client) int {
	if url == "" {
		return 0
	}
	if _, err := pool.Exec(ctx, `UPDATE notifications SET pushed_at=now() WHERE pushed_at IS NULL AND created_at < now()-interval '1 hour'`); err != nil {
		return 0
	}
	rows, err := pool.Query(ctx, `SELECT n.id, n.user_id, n.kind, n.title, n.body, n.data FROM notifications n
		WHERE n.pushed_at IS NULL ORDER BY n.id LIMIT 50`)
	if err != nil {
		return 0
	}
	type note struct {
		id                      int64
		user, kind, title, body string
		data                    map[string]any
	}
	var notes []note
	for rows.Next() {
		var n note
		var raw []byte
		if rows.Scan(&n.id, &n.user, &n.kind, &n.title, &n.body, &raw) == nil {
			_ = json.Unmarshal(raw, &n.data)
			notes = append(notes, n)
		}
	}
	rows.Close()
	sent := 0
	for _, n := range notes {
		tr, err := pool.Query(ctx, `SELECT token FROM push_tokens WHERE user_id=$1`, n.user)
		if err != nil {
			continue
		}
		var tokens []string
		for tr.Next() {
			var t string
			if tr.Scan(&t) == nil {
				tokens = append(tokens, t)
			}
		}
		tr.Close()
		ok := true
		for _, t := range tokens {
			if err := pushOne(ctx, pool, client, url, t, n.title, n.body, n.data); err != nil {
				slog.Warn("push failed", "err", err)
				ok = false
			}
		}
		if ok {
			_, _ = pool.Exec(ctx, `UPDATE notifications SET pushed_at=now() WHERE id=$1`, n.id)
			sent += len(tokens)
		}
	}
	return sent
}

func pushOne(ctx context.Context, pool *pgxpool.Pool, client *http.Client, url, token, title, body string, data map[string]any) error {
	payload, _ := json.Marshal(map[string]any{"to": token, "title": title, "body": body, "data": data, "sound": "default"})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("push service: status %d", resp.StatusCode)
	}
	var out struct {
		Data struct {
			Status  string `json:"status"`
			Details struct {
				Error string `json:"error"`
			} `json:"details"`
		} `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&out) == nil && out.Data.Details.Error == "DeviceNotRegistered" {
		_, _ = pool.Exec(ctx, `DELETE FROM push_tokens WHERE token=$1`, token)
	}
	return nil
}
