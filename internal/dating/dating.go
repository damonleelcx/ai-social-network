// Package dating is everything two people do together: proposals, matches, chat,
// date plans and safety actions. Nothing it returns carries a score. It tells the
// hidden side what happened through the Learner interface, one way only.
package dating

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damonleelcx/ai-social-network/internal/chatstore"
	"github.com/damonleelcx/ai-social-network/internal/i18n"
	"github.com/damonleelcx/ai-social-network/internal/moderation"
	"github.com/damonleelcx/ai-social-network/internal/notify"
	"github.com/damonleelcx/ai-social-network/internal/profile"
	"github.com/damonleelcx/ai-social-network/internal/safety"
)

// Learner receives behaviour for the taste model. It returns nothing a caller
// could show anyone.
type Learner func(ctx context.Context, userID, subjectID, kind, key string)

// Hooks start background work. They are set by internal/workflows, which owns the
// goals, so this package needs no knowledge of the engine.
type Hooks struct {
	Coach      func(ctx context.Context, matchID, userID, trigger string)
	DatePlan   func(ctx context.Context, dateID, matchID, userID string)
	SafetyScan func(ctx context.Context, matchID, reportID string)
}

// Scener draws a picture for a plan or a trip in the background (internal/scenes).
type Scener interface {
	Post(requestID string)
	Trip(tripID string)
	// The previews name the picture for a card that hasn't been confirmed yet, and start drawing it.
	PreviewPost(ctx context.Context, userID, kind, note, place, cityKey, cityName string) string
	PreviewTrip(ctx context.Context, cityKey, cityName string) string
}

type Service struct {
	Pool  *pgxpool.Pool
	Chats *chatstore.Store
	Learn Learner
	Hooks Hooks
	// Scenes gives plans and trips a picture. Nil leaves them with the poster's photo.
	Scenes Scener
	// Mod filters NSFW text. Nil means the word list alone.
	Mod *moderation.Filter
	// DateMinLead is the earliest a date can be set from now.
	DateMinLead time.Duration
}

// UserError is a message safe to show the person.
type UserError struct{ Msg string }

func (e *UserError) Error() string { return e.Msg }

func userErr(f string, a ...any) error { return &UserError{fmt.Sprintf(f, a...)} }

var ErrNotFound = errors.New("not found")

func (s *Service) learn(ctx context.Context, user, subject, kind, key string) {
	if s.Learn != nil {
		s.Learn(ctx, user, subject, kind, key)
	}
}

func (s *Service) notifyUser(ctx context.Context, userID, topic string) {
	_, _ = s.Pool.Exec(context.WithoutCancel(ctx), `SELECT pg_notify('syn_user', $1)`, userID+"|"+topic)
}

// ConversationID returns this person's one conversation with Suki.
func ConversationID(ctx context.Context, pool *pgxpool.Pool, userID string) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `INSERT INTO conversations (user_id, title) VALUES ($1, 'Suki')
		ON CONFLICT (user_id) DO UPDATE SET updated_at=conversations.updated_at RETURNING id`, userID).Scan(&id)
	return id, err
}

// PostFromSuki writes a fixed message from Suki into someone's conversation.
func (s *Service) PostFromSuki(ctx context.Context, userID, body, mood string, cards []map[string]any, key string) error {
	conv, err := ConversationID(ctx, s.Pool, userID)
	if err != nil {
		return err
	}
	body = i18n.T(s.Lang(ctx, userID), body)
	meta := map[string]any{"mood": mood}
	if len(cards) > 0 {
		meta["cards"] = cards
	}
	if _, _, err := s.Chats.AddAgent(ctx, conv, "assistant", body, meta, key); err != nil {
		return err
	}
	s.notifyUser(ctx, userID, "agent")
	return nil
}

// ── Proposals ───────────────────────────────────────────────────────────────

// Proposal is one person's view of a pair. Only the other person's public card
// and the pitch: no numbers, and nothing about how they decided.
type Proposal struct {
	ID        string         `json:"id"`
	Status    string         `json:"status"`
	Pitch     string         `json:"pitch"`
	Shared    []string       `json:"shared"`
	Decision  string         `json:"my_decision"`
	Person    profile.Public `json:"person"`
	CreatedAt time.Time      `json:"created_at"`
	ExpiresAt time.Time      `json:"expires_at"`
	MatchID   string         `json:"match_id,omitempty"`
}

func pairKey(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

// Publish creates a proposal for a pair, once ever. created is false when the
// pair already has one (a retried task, or the other person's search got there
// first), and nobody is told twice.
func (s *Service) Publish(ctx context.Context, viewerID, candidateID, pitchViewer, pitchCandidate string, shared []string, goalID string) (string, bool, error) {
	a, b := pairKey(viewerID, candidateID)
	pa, pb := pitchViewer, pitchCandidate
	if a != viewerID {
		pa, pb = pitchCandidate, pitchViewer
	}
	if shared == nil {
		shared = []string{}
	}
	raw, _ := json.Marshal(shared)
	var id string
	err := s.Pool.QueryRow(ctx, `INSERT INTO proposals (user_a, user_b, pitch_a, pitch_b, shared, goal_id) VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (user_a, user_b) DO NOTHING RETURNING id`, a, b, pa, pb, raw, goalID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.Pool.QueryRow(ctx, `SELECT id FROM proposals WHERE user_a=$1 AND user_b=$2`, a, b).Scan(&id)
		return id, false, err
	}
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

// Announce tells both people about a proposal. Each write is keyed, so a retried
// task cannot announce twice.
func (s *Service) Announce(ctx context.Context, proposalID string) error {
	var a, b, pa, pb string
	if err := s.Pool.QueryRow(ctx, `SELECT user_a, user_b, pitch_a, pitch_b FROM proposals WHERE id=$1`, proposalID).Scan(&a, &b, &pa, &pb); err != nil {
		return err
	}
	for _, x := range []struct{ user, other, pitch string }{{a, b, pa}, {b, a, pb}} {
		name, err := s.displayName(ctx, x.other)
		if err != nil {
			return err
		}
		body := i18n.T(s.Lang(ctx, x.user), "I found someone! Meet {name}.", "name", name) + " " + x.pitch
		cards := []map[string]any{{"kind": "proposal", "proposal_id": proposalID}}
		if err := s.PostFromSuki(ctx, x.user, body, "happy", cards, "proposal:"+proposalID); err != nil {
			return err
		}
		if err := notify.Send(ctx, s.Pool, x.user, "proposal", "Suki found someone for you", "Take a look when you have a moment.",
			map[string]any{"screen": "proposal", "id": proposalID}, "proposal:"+proposalID+":"+x.user); err != nil {
			return err
		}
		s.notifyUser(ctx, x.user, "proposals")
	}
	return nil
}

func (s *Service) displayName(ctx context.Context, userID string) (string, error) {
	var n string
	err := s.Pool.QueryRow(ctx, `SELECT display_name FROM profiles WHERE user_id=$1`, userID).Scan(&n)
	return n, err
}

const proposalCols = `id, user_a, user_b, pitch_a, pitch_b, shared, coalesce(decision_a,''), coalesce(decision_b,''), status, created_at, expires_at,
	coalesce((SELECT m.id::text FROM matches m WHERE m.proposal_id=proposals.id),'')`

func (s *Service) scanProposal(ctx context.Context, row pgx.Row, viewer string) (*Proposal, error) {
	var p Proposal
	var a, b, pa, pb, da, db string
	var shared []byte
	if err := row.Scan(&p.ID, &a, &b, &pa, &pb, &shared, &da, &db, &p.Status, &p.CreatedAt, &p.ExpiresAt, &p.MatchID); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(shared, &p.Shared)
	other := b
	p.Pitch, p.Decision = pa, da
	if viewer == b {
		other = a
		p.Pitch, p.Decision = pb, db
	}
	blocked, err := s.blockedEitherWay(ctx, a, b)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, pgx.ErrNoRows
	}
	// Someone who hasn't answered can't tell a pass from no answer yet.
	if p.Status == "declined" && p.Decision == "" {
		p.Status = "open"
	}
	op, err := profile.Get(ctx, s.Pool, other)
	if err != nil {
		return nil, err
	}
	ids, err := profile.ApprovedPhotoIDs(ctx, s.Pool, other)
	if err != nil {
		return nil, err
	}
	p.Person = op.Public(ids)
	return &p, nil
}

// OpenProposals lists what is waiting on this person. A proposal they already
// answered disappears, and it stays silent whether the other person answered.
func (s *Service) OpenProposals(ctx context.Context, userID string) ([]*Proposal, error) {
	// Ordered by id, which is random: an order by score or time would reveal rank within
	// a batch. A proposal the other person passed stays until it expires, for the same
	// reason a pass isn't announced; one with a block in it goes.
	rows, err := s.Pool.Query(ctx, `SELECT id FROM proposals p WHERE p.status IN ('open','declined') AND p.expires_at>now()
		AND ((p.user_a=$1 AND p.decision_a IS NULL) OR (p.user_b=$1 AND p.decision_b IS NULL))
		AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id=p.user_a AND b.blocked_id=p.user_b) OR (b.blocker_id=p.user_b AND b.blocked_id=p.user_a))
		ORDER BY p.id LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	out := []*Proposal{}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		p, err := s.ProposalFor(ctx, id, userID)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// ProposalFor returns a proposal if this person is one of the pair.
func (s *Service) ProposalFor(ctx context.Context, id, userID string) (*Proposal, error) {
	p, err := s.scanProposal(ctx, s.Pool.QueryRow(ctx, `SELECT `+proposalCols+` FROM proposals WHERE id=$1 AND (user_a=$2 OR user_b=$2)`, id, userID), userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

// Decide records one person's answer. It tells them only what they need: a match
// exists, or nothing happens yet. A pass is never announced to the other person,
// and they can't tell a pass from no answer.
func (s *Service) Decide(ctx context.Context, proposalID, userID, decision string) (matchID string, err error) {
	if err := s.requireNotHeld(ctx, userID); err != nil {
		return "", err
	}
	if decision != "accept" && decision != "pass" {
		return "", userErr("decision must be accept or pass")
	}
	var a, b, status string
	var da, db *string
	var newMatch bool
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT user_a, user_b, decision_a, decision_b, status FROM proposals WHERE id=$1 AND (user_a=$2 OR user_b=$2) FOR UPDATE`,
			proposalID, userID).Scan(&a, &b, &da, &db, &status); err != nil {
			return ErrNotFound
		}
		if status == "expired" {
			return userErr("this introduction has expired")
		}
		if status == "matched" && decision == "pass" {
			return userErr("you've already answered this one")
		}
		mine, col := &da, "decision_a"
		if userID == b {
			mine, col = &db, "decision_b"
		}
		if *mine != nil {
			if **mine != decision {
				return userErr("you've already answered this one")
			}
			return tx.QueryRow(ctx, `SELECT coalesce((SELECT id::text FROM matches WHERE proposal_id=$1),'')`, proposalID).Scan(&matchID)
		}
		*mine = &decision
		// Only an open proposal can become a match. If the other person passed or
		// blocked, this answer is recorded and nothing happens, and they see the same
		// response as anyone still waiting for a reply.
		next := status
		if status == "open" {
			if decision == "pass" {
				next = "declined"
			} else if da != nil && db != nil && *da == "accept" && *db == "accept" {
				next = "matched"
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE proposals SET `+col+`=$2, status=$3 WHERE id=$1`, proposalID, decision, next); err != nil {
			return err
		}
		if next == "matched" {
			newMatch = true
			return tx.QueryRow(ctx, `INSERT INTO matches (proposal_id, user_a, user_b) VALUES ($1,$2,$3) RETURNING id`, proposalID, a, b).Scan(&matchID)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	other := a
	if userID == a {
		other = b
	}
	if decision == "accept" {
		s.learn(ctx, userID, other, "accept", "decide:"+proposalID+":"+userID)
	} else {
		s.learn(ctx, userID, other, "pass", "decide:"+proposalID+":"+userID)
	}
	if newMatch {
		s.onMatch(ctx, matchID, a, b)
	}
	// Only the person who answered is pinged, unless it made a match: a ping to the
	// other person would reveal that they had been answered, and how.
	s.notifyUser(ctx, userID, "proposals")
	if newMatch {
		s.notifyUser(ctx, other, "proposals")
	}
	return matchID, nil
}

func (s *Service) onMatch(ctx context.Context, matchID, a, b string) {
	s.learn(ctx, a, b, "mutual", "mutual:"+matchID+":"+a)
	s.learn(ctx, b, a, "mutual", "mutual:"+matchID+":"+b)
	for _, x := range [][2]string{{a, b}, {b, a}} {
		name, err := s.displayName(ctx, x[1])
		if err != nil {
			slog.Error("match announce", "err", err)
			continue
		}
		lang := s.Lang(ctx, x[0])
		body := i18n.T(lang, "It's a match with {name}! You both said yes. I'm thinking of an opener for you right now.", "name", name)
		if err := s.PostFromSuki(ctx, x[0], body, "surprised", []map[string]any{{"kind": "match", "match_id": matchID}}, "match:"+matchID); err != nil {
			slog.Error("match announce", "err", err)
		}
		if err := notify.Send(ctx, s.Pool, x[0], "match", "It's a match!", i18n.T(lang, "You and {name} both said yes.", "name", name),
			map[string]any{"screen": "chat", "id": matchID}, "match:"+matchID+":"+x[0]); err != nil {
			slog.Error("match notify", "err", err)
		}
		if s.Hooks.Coach != nil {
			s.Hooks.Coach(ctx, matchID, x[0], "match_created")
		}
		s.notifyUser(ctx, x[0], "matches")
	}
}

// ExpireProposals closes unanswered proposals. No signal is recorded: silence
// doesn't say anything about the other person.
func (s *Service) ExpireProposals(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `UPDATE proposals SET status='expired' WHERE status='open' AND expires_at<now()`)
	return err
}

// ── Matches ─────────────────────────────────────────────────────────────────

type Match struct {
	ID            string         `json:"id"`
	Stage         string         `json:"stage"`
	Status        string         `json:"status"`
	Person        profile.Public `json:"person"`
	CreatedAt     time.Time      `json:"created_at"`
	LastMessageAt *time.Time     `json:"last_message_at"`
	LastPreview   string         `json:"last_preview"`
	Unread        int64          `json:"unread"`
	OpenDateID    string         `json:"open_date_id,omitempty"`
	// Activity is the kind of plan this match came from. Empty for an introduction.
	Activity string `json:"activity,omitempty"`
}

// Participants returns the pair, for authorisation everywhere else.
func (s *Service) Participants(ctx context.Context, matchID string) (a, b, status string, err error) {
	err = s.Pool.QueryRow(ctx, `SELECT user_a, user_b, status FROM matches WHERE id=$1`, matchID).Scan(&a, &b, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return
}

func (s *Service) otherOf(ctx context.Context, matchID, userID string) (string, string, error) {
	a, b, status, err := s.Participants(ctx, matchID)
	if err != nil {
		return "", "", err
	}
	switch userID {
	case a:
		return b, status, nil
	case b:
		return a, status, nil
	}
	return "", "", ErrNotFound
}

func (s *Service) matchFor(ctx context.Context, id, userID string) (*Match, error) {
	var m Match
	var a, b string
	err := s.Pool.QueryRow(ctx, `SELECT m.id, m.stage, m.status, m.user_a, m.user_b, m.created_at, m.last_message_at, m.last_preview,
		coalesce((SELECT d.id::text FROM date_plans d WHERE d.match_id=m.id AND d.status IN ('searching','voting','scheduling','confirmed') LIMIT 1),''),
		coalesce(m.buddy_kind,'')
		FROM matches m WHERE m.id=$1 AND (m.user_a=$2 OR m.user_b=$2)`, id, userID).
		Scan(&m.ID, &m.Stage, &m.Status, &a, &b, &m.CreatedAt, &m.LastMessageAt, &m.LastPreview, &m.OpenDateID, &m.Activity)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	other := b
	if userID == b {
		other = a
	}
	blocked, err := s.blockedEitherWay(ctx, a, b)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrNotFound
	}
	op, err := profile.Get(ctx, s.Pool, other)
	if err != nil {
		return nil, err
	}
	ids, err := profile.ApprovedPhotoIDs(ctx, s.Pool, other)
	if err != nil {
		return nil, err
	}
	m.Person = op.Public(ids)
	m.Unread, _ = s.Chats.Unread(ctx, id, userID)
	return &m, nil
}

func (s *Service) MatchFor(ctx context.Context, id, userID string) (*Match, error) { return s.matchFor(ctx, id, userID) }

func (s *Service) Matches(ctx context.Context, userID string) ([]*Match, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id FROM matches WHERE (user_a=$1 OR user_b=$1) AND status='active'
		ORDER BY coalesce(last_message_at, created_at) DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := []*Match{}
	for _, id := range ids {
		m, err := s.matchFor(ctx, id, userID)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// Unmatch ends a match for both people. Open date plans are cancelled with it.
func (s *Service) Unmatch(ctx context.Context, matchID, userID string) error {
	other, status, err := s.otherOf(ctx, matchID, userID)
	if err != nil {
		return err
	}
	if status != "active" {
		return nil
	}
	if err := s.closeMatch(ctx, matchID); err != nil {
		return err
	}
	s.learn(ctx, userID, other, "unmatch", "unmatch:"+matchID+":"+userID)
	s.notifyUser(ctx, other, "matches")
	s.notifyUser(ctx, userID, "matches")
	return nil
}

func (s *Service) closeMatch(ctx context.Context, matchID string) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE matches SET status='unmatched' WHERE id=$1`, matchID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE date_plans SET status='cancelled', updated_at=now() WHERE match_id=$1 AND status IN ('searching','voting','scheduling','confirmed')`, matchID)
		return err
	})
}

// Strike records blocked NSFW content for the moderators. One slip is a note; a
// pattern, or anything sexual involving a minor, needs a person. The minor case holds
// the account at once and keeps none of the text.
func (s *Service) Strike(ctx context.Context, userID, matchID, where string, v moderation.Verdict, text string) {
	r := Report{SubjectID: userID, MatchID: matchID, Source: "sentinel", Reason: "nsfw_" + where + ":" + v.Category, Details: truncateRunes(text, 200)}
	if v.Minors() {
		r.Severity, r.Details = "critical", "(text not retained)"
	} else {
		var recent int
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM reports WHERE subject_id=$1 AND reason LIKE 'nsfw_%' AND created_at > now()-interval '7 days'`, userID).Scan(&recent)
		if recent >= 2 {
			r.Severity = "high"
		}
	}
	if _, err := s.FileReport(ctx, r); err != nil {
		slog.Error("nsfw strike", "err", err)
	}
}

// Touch tells a person's devices that something changed.
func (s *Service) Touch(ctx context.Context, userID, topic string) { s.notifyUser(ctx, userID, topic) }

// ── Chat ────────────────────────────────────────────────────────────────────

const maxMessage = 2000

// CoachThresholds are the message counts at which Suki takes another look.
var CoachThresholds = []int{4, 10, 20, 40}

// SendMessage stores a message from one person to the other. The stored text is
// delivered as written; a scan may add a private warning for the recipient.
func (s *Service) SendMessage(ctx context.Context, matchID, userID, body, clientID string) (*chatstore.Message, error) {
	other, status, err := s.otherOf(ctx, matchID, userID)
	if err != nil {
		return nil, err
	}
	if status != "active" {
		return nil, userErr("this conversation has ended")
	}
	if err := s.requireNotHeld(ctx, userID); err != nil {
		return nil, err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, userErr("write something first")
	}
	if utf8.RuneCountInString(body) > maxMessage {
		return nil, userErr("that message is too long")
	}
	// Checked before storing: a blocked message never exists for the other person.
	if v := s.Mod.Check(ctx, userID, moderation.Chat, body); v.Blocked {
		s.Strike(ctx, userID, matchID, "chat", v, body)
		return nil, userErr("%s", moderation.BlockedMessage)
	}
	var key string
	if clientID != "" {
		key = userID + ":" + clientID
	}
	m, _, err := s.Chats.AddChat(ctx, matchID, userID, "text", body, nil, key)
	if err != nil {
		return nil, err
	}
	// Everything after the MongoDB write is keyed or guarded by seq, and runs on a retry
	// too. If Postgres fails on the first try, the retry still does the threat scan, the
	// notification and the preview instead of returning early.
	if f := safety.ScanChat(body); f != nil {
		s.flag(ctx, matchID, userID, other, m.Seq, f, body)
	}
	var count, coachedAt int
	var buddy bool
	preview := truncateRunes(body, 80)
	err = s.Pool.QueryRow(ctx, `UPDATE matches SET last_message_at=now(), last_preview=$2, message_count=message_count+1, stall_nudged_at=NULL, last_seq=$3 WHERE id=$1 AND last_seq<$3
		RETURNING message_count, coached_at_count, buddy_kind IS NOT NULL`, matchID, preview, m.Seq).Scan(&count, &coachedAt, &buddy)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, nil // already counted: this is a retry of a finished send
	}
	if err != nil {
		return nil, err
	}
	if count >= 2 && !buddy {
		// The first exchange in each direction says the conversation is alive.
		s.learn(ctx, userID, other, "replied", "replied:"+matchID+":"+userID)
	}
	_ = notify.Send(ctx, s.Pool, other, "message", "New message", "You have a new message.",
		map[string]any{"screen": "chat", "id": matchID}, fmt.Sprintf("msg:%s:%d", matchID, m.Seq))
	s.notifyUser(ctx, other, "chat:"+matchID)
	s.notifyUser(ctx, userID, "chat:"+matchID)
	if count == 1 {
		// The opener went out: move past icebreaking, no model needed.
		_, _ = s.Pool.Exec(ctx, `UPDATE matches SET stage='getting_to_know' WHERE id=$1 AND stage='icebreaker'`, matchID)
	}
	for _, t := range CoachThresholds {
		if buddy {
			break // A plan with a stranger isn't a romance to coach.
		}
		if coachedAt < t && count >= t && s.Hooks.Coach != nil {
			tag, err := s.Pool.Exec(ctx, `UPDATE matches SET coached_at_count=$2 WHERE id=$1 AND coached_at_count<$2`, matchID, t)
			if err == nil && tag.RowsAffected() == 1 {
				s.Hooks.Coach(ctx, matchID, userID, fmt.Sprintf("count_%d", t))
				s.Hooks.Coach(ctx, matchID, other, fmt.Sprintf("count_%d", t))
			}
			break
		}
	}
	return m, nil
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// flag warns the recipient privately and, for anything serious, files a report
// for a human. The sender isn't told; that would teach them to reword it.
func (s *Service) flag(ctx context.Context, matchID, sender, recipient string, seq int64, f *safety.Finding, body string) {
	key := fmt.Sprintf("scan:%s:%d", matchID, seq)
	if f.Warn != "" {
		raw, _ := json.Marshal([]string{})
		_, err := s.Pool.Exec(ctx, `INSERT INTO coach_hints (match_id, user_id, body, suggestions, key) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (key) DO NOTHING`,
			matchID, recipient, f.Warn, raw, key)
		if err != nil {
			slog.Error("scan hint", "err", err)
		}
		s.notifyUser(ctx, recipient, "chat:"+matchID)
	}
	if f.Severity == "normal" {
		return
	}
	id, err := s.FileReport(ctx, Report{SubjectID: sender, MatchID: matchID, MessageSeq: seq, Source: "sentinel", Reason: f.Reason, Details: truncateRunes(body, 500), Severity: f.Severity, Key: key})
	if err != nil {
		slog.Error("scan report", "err", err)
		return
	}
	if s.Hooks.SafetyScan != nil && id != "" {
		s.Hooks.SafetyScan(ctx, matchID, id)
	}
}

// Messages returns a thread to a participant.
func (s *Service) Messages(ctx context.Context, matchID, userID string, after int64, limit int) ([]chatstore.Message, error) {
	other, _, err := s.otherOf(ctx, matchID, userID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	var ms []chatstore.Message
	if after == 0 {
		ms, err = s.Chats.ChatRecent(ctx, matchID, limit)
	} else {
		ms, err = s.Chats.ChatAfter(ctx, matchID, after, limit)
	}
	if err != nil {
		return nil, err
	}
	localizeNotes(ms, s.Lang(ctx, userID))
	if Prefs(ctx, s.Pool, other)["read_receipts"] == false {
		for i := range ms {
			if ms[i].SenderID == userID {
				ms[i].ReadAt = nil
			}
		}
	}
	return ms, nil
}

// MarkRead clears the unread count. Messages hides the read time from the other
// person if this person turned read receipts off.
func (s *Service) MarkRead(ctx context.Context, matchID, userID string, upto int64) error {
	other, _, err := s.otherOf(ctx, matchID, userID)
	if err != nil {
		return err
	}
	n, err := s.Chats.MarkRead(ctx, matchID, userID, upto)
	if err != nil {
		return err
	}
	if n > 0 {
		s.notifyUser(ctx, other, "chat:"+matchID)
	}
	return nil
}

// Typing tells the other person. It writes nothing.
func (s *Service) Typing(ctx context.Context, matchID, userID string) error {
	other, status, err := s.otherOf(ctx, matchID, userID)
	if err != nil {
		return err
	}
	if status == "active" {
		s.notifyUser(ctx, other, "typing:"+matchID)
	}
	return nil
}

// ── Coach hints ─────────────────────────────────────────────────────────────

type Hint struct {
	ID          string    `json:"id"`
	Body        string    `json:"body"`
	Suggestions []string  `json:"suggestions"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Service) Hints(ctx context.Context, matchID, userID string) ([]Hint, error) {
	if _, _, err := s.otherOf(ctx, matchID, userID); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, body, suggestions, created_at FROM coach_hints
		WHERE match_id=$1 AND user_id=$2 AND dismissed_at IS NULL ORDER BY created_at DESC LIMIT 5`, matchID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Hint{}
	for rows.Next() {
		var h Hint
		var raw []byte
		if err := rows.Scan(&h.ID, &h.Body, &raw, &h.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &h.Suggestions)
		if h.Suggestions == nil {
			h.Suggestions = []string{}
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Service) DismissHint(ctx context.Context, id, userID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE coach_hints SET dismissed_at=now() WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

// AddHint stores a private hint, once per key.
func (s *Service) AddHint(ctx context.Context, matchID, userID, body string, suggestions []string, key string) error {
	if _, _, err := s.otherOf(ctx, matchID, userID); err != nil {
		return err
	}
	if suggestions == nil {
		suggestions = []string{}
	}
	raw, _ := json.Marshal(suggestions)
	tag, err := s.Pool.Exec(ctx, `INSERT INTO coach_hints (match_id, user_id, body, suggestions, key) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (key) DO NOTHING`,
		matchID, userID, body, raw, key)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 1 {
		s.notifyUser(ctx, userID, "chat:"+matchID)
	}
	return nil
}

// Prefs reads settings with their defaults applied by the caller.
func Prefs(ctx context.Context, pool *pgxpool.Pool, userID string) map[string]any {
	out := map[string]any{}
	var raw []byte
	if pool.QueryRow(ctx, `SELECT data FROM user_preferences WHERE user_id=$1`, userID).Scan(&raw) == nil {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

// CoachCanRead reports whether a person allows Suki to read their messages for
// coaching. It is on by default and stated in the privacy policy; turning it off
// removes their words from both people's coaching.
func CoachCanRead(prefs map[string]any) bool { return prefs["chat_coach_read"] != false }

// ConsentedThread is the recent conversation as the Coach may see it: messages
// from anyone who opted out are replaced with a placeholder, so the Coach knows
// the turn happened but not what was said.
func (s *Service) ConsentedThread(ctx context.Context, matchID, viewerID string, n int) (string, error) {
	other, _, err := s.otherOf(ctx, matchID, viewerID)
	if err != nil {
		return "", err
	}
	ms, err := s.Chats.ChatRecent(ctx, matchID, n)
	if err != nil {
		return "", err
	}
	otherName, _ := s.displayName(ctx, other)
	can := map[string]bool{viewerID: CoachCanRead(Prefs(ctx, s.Pool, viewerID)), other: CoachCanRead(Prefs(ctx, s.Pool, other))}
	var b strings.Builder
	for _, m := range ms {
		switch {
		case m.SenderID == "":
			fmt.Fprintf(&b, "[Suki to both] %s\n", m.Body)
		case !can[m.SenderID]:
			fmt.Fprintf(&b, "%s: (message hidden by their privacy setting)\n", whoName(m.SenderID, viewerID, otherName))
		default:
			fmt.Fprintf(&b, "%s: %s\n", whoName(m.SenderID, viewerID, otherName), m.Body)
		}
	}
	return b.String(), nil
}

func whoName(sender, viewer, otherName string) string {
	if sender == viewer {
		return "You"
	}
	return otherName
}

// PostPrompt puts one of Suki's questions into a shared chat, only when both
// people allow it.
func (s *Service) PostPrompt(ctx context.Context, matchID, viewerID, body, key string) (bool, error) {
	other, status, err := s.otherOf(ctx, matchID, viewerID)
	if err != nil || status != "active" {
		return false, err
	}
	if Prefs(ctx, s.Pool, viewerID)["chat_prompts"] == false || Prefs(ctx, s.Pool, other)["chat_prompts"] == false {
		return false, nil
	}
	_, created, err := s.Chats.AddChat(ctx, matchID, "", "prompt", body, nil, key)
	if err != nil {
		return false, err
	}
	if created {
		s.notifyUser(ctx, viewerID, "chat:"+matchID)
		s.notifyUser(ctx, other, "chat:"+matchID)
	}
	return created, nil
}

var stageOrder = map[string]int{"icebreaker": 0, "getting_to_know": 1, "date_ready": 2, "planning": 3, "planned": 4, "post_date": 5}

// SetStage only moves a conversation forward: a model's opinion can't undo a
// date that is already planned.
func (s *Service) SetStage(ctx context.Context, matchID, userID, stage string) error {
	if _, ok := stageOrder[stage]; !ok {
		return userErr("unknown stage %q", stage)
	}
	if _, _, err := s.otherOf(ctx, matchID, userID); err != nil {
		return err
	}
	var cur string
	if err := s.Pool.QueryRow(ctx, `SELECT stage FROM matches WHERE id=$1`, matchID).Scan(&cur); err != nil {
		return err
	}
	if stageOrder[stage] <= stageOrder[cur] {
		return nil
	}
	_, err := s.Pool.Exec(ctx, `UPDATE matches SET stage=$2 WHERE id=$1`, matchID, stage)
	return err
}

// ── Blocks and reports ──────────────────────────────────────────────────────

// Block cuts all contact at once: the match ends, open dates are cancelled and
// neither person is ever proposed to the other again.
func (s *Service) Block(ctx context.Context, blocker, blocked string) error {
	if blocker == blocked {
		return userErr("you can't block yourself")
	}
	if _, err := s.Pool.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, blocker, blocked); err != nil {
		return err
	}
	a, b := pairKey(blocker, blocked)
	rows, err := s.Pool.Query(ctx, `SELECT id FROM matches WHERE user_a=$1 AND user_b=$2 AND status='active'`, a, b)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		if err := s.closeMatch(ctx, id); err != nil {
			return err
		}
	}
	if _, err := s.Pool.Exec(ctx, `UPDATE proposals SET status='declined' WHERE user_a=$1 AND user_b=$2 AND status='open'`, a, b); err != nil {
		return err
	}
	s.notifyUser(ctx, blocked, "matches")
	s.notifyUser(ctx, blocker, "matches")
	return nil
}

func (s *Service) Unblock(ctx context.Context, blocker, blocked string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id=$1 AND blocked_id=$2`, blocker, blocked)
	return err
}

type BlockedUser struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
}

func (s *Service) Blocked(ctx context.Context, userID string) ([]BlockedUser, error) {
	rows, err := s.Pool.Query(ctx, `SELECT b.blocked_id, coalesce(p.display_name,'') FROM blocks b LEFT JOIN profiles p ON p.user_id=b.blocked_id WHERE b.blocker_id=$1 ORDER BY b.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var x BlockedUser
		if err := rows.Scan(&x.UserID, &x.DisplayName); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

type Report struct {
	ReporterID, SubjectID, MatchID string
	MessageSeq                     int64
	Source, Reason, Details        string
	Severity, Key                  string
}

// FileReport adds to the moderators' queue. A critical one also holds the subject
// from new introductions until a human has looked. It doesn't punish anyone.
func (s *Service) FileReport(ctx context.Context, r Report) (string, error) {
	if r.Severity == "" {
		r.Severity = "normal"
	}
	if r.Source == "" {
		r.Source = "user"
	}
	var key any
	if r.Key != "" {
		key = r.Key
	}
	var seq any
	if r.MessageSeq > 0 {
		seq = r.MessageSeq
	}
	var id string
	err := s.Pool.QueryRow(ctx, `INSERT INTO reports (reporter_id, subject_id, match_id, message_seq, source, reason, details, severity, key)
		VALUES (nullif($1,'')::uuid, nullif($2,'')::uuid, nullif($3,'')::uuid, $4, $5, $6, $7, $8, $9) ON CONFLICT (key) DO NOTHING RETURNING id`,
		r.ReporterID, r.SubjectID, r.MatchID, seq, r.Source, r.Reason, r.Details, r.Severity, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if r.Severity == "critical" && r.SubjectID != "" {
		if _, err := s.Pool.Exec(ctx, `UPDATE profiles SET status='held', updated_at=now() WHERE user_id=$1`, r.SubjectID); err != nil {
			return id, err
		}
	}
	return id, nil
}

// HoldForReview stops someone being proposed, pending a moderator.
func (s *Service) HoldForReview(ctx context.Context, userID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE profiles SET status='held', updated_at=now() WHERE user_id=$1`, userID)
	return err
}

func (s *Service) blockedEitherWay(ctx context.Context, a, b string) (bool, error) {
	var blocked bool
	err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM blocks WHERE (blocker_id=$1 AND blocked_id=$2) OR (blocker_id=$2 AND blocked_id=$1))`, a, b).Scan(&blocked)
	return blocked, err
}

// requireNotHeld stops an account that a moderator must look at from contacting
// anyone, including one that may belong to a minor.
func (s *Service) requireNotHeld(ctx context.Context, userID string) error {
	var status string
	err := s.Pool.QueryRow(ctx, `SELECT status FROM profiles WHERE user_id=$1`, userID).Scan(&status)
	if err == nil && status == "held" {
		return userErr("Your account is paused until someone on our team has taken a look.")
	}
	return nil
}

// CloseEverything ends a banned person's connections, so no one is left waiting on
// them or planning to meet them.
func (s *Service) CloseEverything(ctx context.Context, userID string) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE buddy_requests SET status='closed' WHERE user_id=$1 AND status='open'`, userID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE date_plans SET status='cancelled', updated_at=now() WHERE status IN ('searching','voting','scheduling','confirmed')
			AND match_id IN (SELECT id FROM matches WHERE user_a=$1 OR user_b=$1)`, userID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE matches SET status='unmatched' WHERE (user_a=$1 OR user_b=$1) AND status='active'`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE proposals SET status='declined' WHERE (user_a=$1 OR user_b=$1) AND status='open'`, userID)
		return err
	})
}

// AnnounceForGoal re-announces what a search published. Announcements are keyed, so
// this is how a retried task finishes a crash between publishing and announcing.
func (s *Service) AnnounceForGoal(ctx context.Context, goalID string) error {
	rows, err := s.Pool.Query(ctx, `SELECT id FROM proposals WHERE goal_id=$1`, goalID)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		if err := s.Announce(ctx, id); err != nil {
			return err
		}
	}
	return rows.Err()
}

// CanSeePhoto is the one gate on another person's photo: your own, someone Suki has
// introduced you to and is still waiting on, a current match, someone whose buddy post
// you can see, or someone who said they're in on yours. After an unmatch or a
// block the answer is the same 404, so a person can't tell which happened.
func (s *Service) CanSeePhoto(ctx context.Context, viewerID, ownerID string) (bool, error) {
	if viewerID == ownerID {
		return true, nil
	}
	a, b := pairKey(viewerID, ownerID)
	var ok bool
	err := s.Pool.QueryRow(ctx, `SELECT (EXISTS(SELECT 1 FROM proposals WHERE user_a=$1 AND user_b=$2 AND status IN ('open','declined') AND expires_at>now())
			OR EXISTS(SELECT 1 FROM matches WHERE user_a=$1 AND user_b=$2 AND status='active')
			OR EXISTS(SELECT 1 FROM buddy_requests r JOIN profiles me ON me.user_id=$3 WHERE r.user_id=$4 AND r.status='open' AND r.expires_at>now()
				AND r.city_key=me.city_key AND (NOT r.same_gender OR r.gender=me.gender))
			OR EXISTS(SELECT 1 FROM buddy_requests r JOIN buddy_offers o ON o.request_id=r.id WHERE r.user_id=$3 AND o.user_id=$4 AND r.status='open' AND r.expires_at>now()))
		AND NOT EXISTS(SELECT 1 FROM blocks WHERE (blocker_id=$1 AND blocked_id=$2) OR (blocker_id=$2 AND blocked_id=$1))`, a, b, viewerID, ownerID).Scan(&ok)
	return ok, err
}
