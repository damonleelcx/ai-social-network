package dating

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/damonleelcx/ai-social-network/internal/i18n"
	"github.com/damonleelcx/ai-social-network/internal/moderation"
	"github.com/damonleelcx/ai-social-network/internal/notify"
	"github.com/damonleelcx/ai-social-network/internal/places"
	"github.com/damonleelcx/ai-social-network/internal/profile"
)

// DatePlan moves through: searching (Suki looks for places) → voting (both pick
// from up to 3) → scheduling (one proposes a time, the other confirms) →
// confirmed → completed. Either person can cancel at any point.
type DatePlan struct {
	ID          string            `json:"id"`
	MatchID     string            `json:"match_id"`
	Status      string            `json:"status"`
	Venues      []places.Venue    `json:"venues"`
	MyVote      string            `json:"my_vote"`
	OtherVoted  bool              `json:"other_voted"`
	Venue       *places.Venue     `json:"venue"`
	Constraints []string          `json:"constraints"`
	ProposedAt  *time.Time        `json:"proposed_at"`
	ProposedBy  string            `json:"proposed_by"`
	ByMe        bool              `json:"proposed_by_me"`
	ScheduledAt *time.Time        `json:"scheduled_at"`
	Person      profile.Public    `json:"person"`
	CreatedAt   time.Time         `json:"created_at"`
}

func (s *Service) dateMatch(ctx context.Context, dateID, userID string) (matchID, other string, err error) {
	if err = s.Pool.QueryRow(ctx, `SELECT match_id FROM date_plans WHERE id=$1`, dateID).Scan(&matchID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrNotFound
		}
		return "", "", err
	}
	other, _, err = s.otherOf(ctx, matchID, userID)
	return matchID, other, err
}

// DateFor returns a plan to one of its two people. It shows whether the other
// person has voted, not what for, so neither can steer the other's choice.
func (s *Service) DateFor(ctx context.Context, dateID, userID string) (*DatePlan, error) {
	matchID, other, err := s.dateMatch(ctx, dateID, userID)
	if err != nil {
		return nil, err
	}
	var d DatePlan
	var venues, votes, venue, cons []byte
	var proposedBy *string
	err = s.Pool.QueryRow(ctx, `SELECT id, status, venues, votes, venue, constraints, proposed_at, proposed_by::text, scheduled_at, created_at FROM date_plans WHERE id=$1`, dateID).
		Scan(&d.ID, &d.Status, &venues, &votes, &venue, &cons, &d.ProposedAt, &proposedBy, &d.ScheduledAt, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	d.MatchID = matchID
	d.Venues = []places.Venue{}
	_ = json.Unmarshal(venues, &d.Venues)
	d.Constraints = []string{}
	_ = json.Unmarshal(cons, &d.Constraints)
	vm := map[string]string{}
	_ = json.Unmarshal(votes, &vm)
	d.MyVote = vm[userID]
	d.OtherVoted = vm[other] != ""
	if venue != nil {
		_ = json.Unmarshal(venue, &d.Venue)
	}
	if proposedBy != nil {
		d.ByMe = *proposedBy == userID
		if d.ByMe {
			d.ProposedBy = "me"
		} else {
			d.ProposedBy = "them"
		}
	}
	op, err := profile.Get(ctx, s.Pool, other)
	if err != nil {
		return nil, err
	}
	ids, err := profile.ApprovedPhotoIDs(ctx, s.Pool, other)
	if err != nil {
		return nil, err
	}
	d.Person = op.Public(ids)
	return &d, nil
}

// Dates lists this person's plans, soonest and most recent first.
func (s *Service) Dates(ctx context.Context, userID string) ([]*DatePlan, error) {
	rows, err := s.Pool.Query(ctx, `SELECT d.id FROM date_plans d JOIN matches m ON m.id=d.match_id WHERE m.user_a=$1 OR m.user_b=$1
		ORDER BY CASE WHEN d.status IN ('searching','voting','scheduling','confirmed') THEN 0 ELSE 1 END, coalesce(d.scheduled_at, d.created_at) DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	out := []*DatePlan{}
	for _, id := range ids {
		d, err := s.DateFor(ctx, id, userID)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// StartDate opens a plan and starts the search. A match has one open plan at a
// time, so asking twice lands on the same plan.
func (s *Service) StartDate(ctx context.Context, matchID, userID string, constraints []string) (string, error) {
	other, status, err := s.otherOf(ctx, matchID, userID)
	if err != nil {
		return "", err
	}
	if status != "active" {
		return "", userErr("this conversation has ended")
	}
	if err := s.requireNotHeld(ctx, userID); err != nil {
		return "", err
	}
	// The other person can see these, so they pass the same filter as a message.
	for _, c := range constraints {
		if v := s.Mod.Check(ctx, userID, moderation.Chat, c); v.Blocked {
			s.Strike(ctx, userID, matchID, "date", v, c)
			return "", userErr("%s", moderation.BlockedMessage)
		}
	}
	if constraints == nil {
		constraints = []string{}
	}
	raw, _ := json.Marshal(constraints)
	var id string
	err = s.Pool.QueryRow(ctx, `INSERT INTO date_plans (match_id, created_by, constraints) VALUES ($1,$2,$3)
		ON CONFLICT (match_id) WHERE status IN ('searching','voting','scheduling','confirmed') DO NOTHING RETURNING id`, matchID, userID, raw).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.Pool.QueryRow(ctx, `SELECT id FROM date_plans WHERE match_id=$1 AND status IN ('searching','voting','scheduling','confirmed')`, matchID).Scan(&id)
		return id, err
	}
	if err != nil {
		return "", err
	}
	_ = s.SetStage(ctx, matchID, userID, "planning")
	s.chatNote(ctx, matchID, "Suki is looking for a place you'd both enjoy.", nil, map[string]any{"date_id": id}, "date-start:"+id)
	if s.Hooks.DatePlan != nil {
		s.Hooks.DatePlan(ctx, id, matchID, userID)
	}
	s.notifyUser(ctx, other, "dates")
	s.notifyUser(ctx, userID, "dates")
	return id, nil
}

// chatNote posts Suki's note on a date plan. One message is read by two people who may use
// different languages, so it stores its English template and is rendered for whoever reads it.
func (s *Service) chatNote(ctx context.Context, matchID, note string, vars map[string]string, meta map[string]any, key string) {
	meta["note"] = note
	if len(vars) > 0 {
		meta["note_vars"] = vars
	}
	body := renderNote("en", note, vars)
	if _, _, err := s.Chats.AddChat(ctx, matchID, "", "date", body, meta, key); err != nil {
		slog.Error("date note", "err", err)
	}
	a, b, _, err := s.Participants(ctx, matchID)
	if err == nil {
		s.notifyUser(ctx, a, "chat:"+matchID)
		s.notifyUser(ctx, b, "chat:"+matchID)
	}
}

// AddConstraint records a need ("not too loud", "wheelchair accessible").
func (s *Service) AddConstraint(ctx context.Context, dateID, userID, constraint string) error {
	matchID, _, err := s.dateMatch(ctx, dateID, userID)
	if err != nil {
		return err
	}
	if v := s.Mod.Check(ctx, userID, moderation.Chat, constraint); v.Blocked {
		s.Strike(ctx, userID, matchID, "date", v, constraint)
		return userErr("%s", moderation.BlockedMessage)
	}
	raw, _ := json.Marshal(constraint)
	_, err = s.Pool.Exec(ctx, `UPDATE date_plans SET constraints = constraints || jsonb_build_array($2::jsonb), updated_at=now() WHERE id=$1`, dateID, string(raw))
	return err
}

// SetCandidates stores what the search found, for the DatePlanner to choose from.
func (s *Service) SetCandidates(ctx context.Context, dateID string, venues []places.Venue) error {
	raw, _ := json.Marshal(venues)
	_, err := s.Pool.Exec(ctx, `UPDATE date_plans SET candidates=$2, updated_at=now() WHERE id=$1`, dateID, raw)
	return err
}

// PublishShortlist opens voting and tells both people.
func (s *Service) PublishShortlist(ctx context.Context, dateID string, venues []places.Venue) error {
	raw, _ := json.Marshal(venues)
	var matchID string
	err := s.Pool.QueryRow(ctx, `UPDATE date_plans SET venues=$2, status='voting', updated_at=now() WHERE id=$1 AND status='searching' RETURNING match_id`, dateID, raw).Scan(&matchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // a retried task, or the plan was cancelled meanwhile
	}
	if err != nil {
		return err
	}
	s.chatNote(ctx, matchID, "I found a few places you'd both like. Pick your favorite, and your pick stays private until you've both chosen.", nil, map[string]any{"date_id": dateID, "card": "vote"}, "date-vote:"+dateID)
	a, b, _, err := s.Participants(ctx, matchID)
	if err != nil {
		return err
	}
	for _, u := range []string{a, b} {
		if err := notify.Send(ctx, s.Pool, u, "date", "Date ideas are ready", "Suki found a few places for you both.", map[string]any{"screen": "date", "id": dateID}, "date-vote:"+dateID+":"+u); err != nil {
			return err
		}
		s.notifyUser(ctx, u, "dates")
	}
	return nil
}

// FailSearch tells both people honestly that no suitable place was found.
func (s *Service) FailSearch(ctx context.Context, dateID, why string) error {
	var matchID string
	err := s.Pool.QueryRow(ctx, `UPDATE date_plans SET status='failed', updated_at=now() WHERE id=$1 AND status='searching' RETURNING match_id`, dateID).Scan(&matchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	s.chatNote(ctx, matchID, "I couldn't find good places nearby just now. {why} You can ask me to try again, or pick somewhere yourselves.", map[string]string{"why": why}, map[string]any{"date_id": dateID}, "date-fail:"+dateID)
	if a, b, _, err := s.Participants(ctx, matchID); err == nil {
		s.notifyUser(ctx, a, "dates")
		s.notifyUser(ctx, b, "dates")
	}
	return nil
}

// Vote records a pick. When the two disagree, the earlier place on Suki's list
// wins and both are told which it was, so there is no awkward third round.
func (s *Service) Vote(ctx context.Context, dateID, userID, venueID string) error {
	matchID, other, err := s.dateMatch(ctx, dateID, userID)
	if err != nil {
		return err
	}
	if err := s.requireNotHeld(ctx, userID); err != nil {
		return err
	}
	var chosen *places.Venue
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var status string
		var venues, votes []byte
		if err := tx.QueryRow(ctx, `SELECT status, venues, votes FROM date_plans WHERE id=$1 FOR UPDATE`, dateID).Scan(&status, &venues, &votes); err != nil {
			return err
		}
		if status != "voting" {
			return userErr("voting on this date has finished")
		}
		var list []places.Venue
		_ = json.Unmarshal(venues, &list)
		vm := map[string]string{}
		_ = json.Unmarshal(votes, &vm)
		found := false
		for _, v := range list {
			found = found || v.ID == venueID
		}
		if !found {
			return userErr("that place isn't on the list")
		}
		vm[userID] = venueID
		raw, _ := json.Marshal(vm)
		if vm[other] == "" {
			_, err := tx.Exec(ctx, `UPDATE date_plans SET votes=$2, updated_at=now() WHERE id=$1`, dateID, raw)
			return err
		}
		for i := range list {
			if list[i].ID == vm[userID] || list[i].ID == vm[other] {
				chosen = &list[i]
				break
			}
		}
		rawV, _ := json.Marshal(chosen)
		_, err := tx.Exec(ctx, `UPDATE date_plans SET votes=$2, venue=$3, status='scheduling', updated_at=now() WHERE id=$1`, dateID, raw, rawV)
		return err
	})
	if err != nil {
		return err
	}
	if chosen != nil {
		s.chatNote(ctx, matchID, "It's {venue}! Now, when works for you both?", map[string]string{"venue": chosen.Name}, map[string]any{"date_id": dateID, "card": "time"}, "date-chosen:"+dateID)
	}
	s.notifyUser(ctx, other, "dates")
	s.notifyUser(ctx, userID, "dates")
	return nil
}

// ProposeTime suggests a time. It also moves a confirmed date back to needing the
// other person's yes, which is what rescheduling is.
func (s *Service) ProposeTime(ctx context.Context, dateID, userID string, at time.Time) error {
	matchID, other, err := s.dateMatch(ctx, dateID, userID)
	if err != nil {
		return err
	}
	if err := s.requireNotHeld(ctx, userID); err != nil {
		return err
	}
	lead := s.DateMinLead
	if lead == 0 {
		lead = time.Hour
	}
	if at.Before(time.Now().Add(lead)) {
		return userErr("pick a time a little further ahead")
	}
	if at.After(time.Now().Add(90 * 24 * time.Hour)) {
		return userErr("pick a time within the next three months")
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE date_plans SET proposed_at=$3, proposed_by=$2, status='scheduling', scheduled_at=NULL,
		reminded_24=false, reminded_2=false, checkin_sent=false, debrief_sent=false, updated_at=now()
		WHERE id=$1 AND status IN ('scheduling','confirmed')`, dateID, userID, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return userErr("this date can't take a time right now")
	}
	s.chatNote(ctx, matchID, "New time suggested. Does it work?", nil, map[string]any{"date_id": dateID, "card": "confirm", "at": at.UTC().Format(time.RFC3339)}, fmt.Sprintf("date-time:%s:%d", dateID, at.Unix()))
	if err := notify.Send(ctx, s.Pool, other, "date", "A time for your date", "Open SynAgent to say yes or suggest another.", map[string]any{"screen": "date", "id": dateID}, fmt.Sprintf("date-time:%s:%d:%s", dateID, at.Unix(), other)); err != nil {
		return err
	}
	s.notifyUser(ctx, other, "dates")
	s.notifyUser(ctx, userID, "dates")
	return nil
}

// ConfirmTime is the other person's yes. Only they can say it.
func (s *Service) ConfirmTime(ctx context.Context, dateID, userID string) error {
	matchID, other, err := s.dateMatch(ctx, dateID, userID)
	if err != nil {
		return err
	}
	if err := s.requireNotHeld(ctx, userID); err != nil {
		return err
	}
	var at time.Time
	err = s.Pool.QueryRow(ctx, `UPDATE date_plans SET status='confirmed', scheduled_at=proposed_at, updated_at=now()
		WHERE id=$1 AND status='scheduling' AND proposed_at > now() AND proposed_by<>$2 RETURNING scheduled_at`, dateID, userID).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return userErr("there's no upcoming time here for you to confirm")
	}
	if err != nil {
		return err
	}
	_ = s.SetStage(ctx, matchID, userID, "planned")
	s.chatNote(ctx, matchID, "You're on! Have a wonderful time. Remember to tell a friend where you'll be.", nil, map[string]any{"date_id": dateID, "card": "confirmed", "at": at.UTC().Format(time.RFC3339)}, fmt.Sprintf("date-ok:%s:%d", dateID, at.Unix()))
	s.learn(ctx, userID, other, "date_set", "date_set:"+dateID+":"+userID)
	s.learn(ctx, other, userID, "date_set", "date_set:"+dateID+":"+other)
	for _, u := range []string{userID, other} {
		if err := notify.Send(ctx, s.Pool, u, "date", "Your date is confirmed", "See you there. Suki will check in afterwards.", map[string]any{"screen": "date", "id": dateID}, fmt.Sprintf("date-ok:%s:%d:%s", dateID, at.Unix(), u)); err != nil {
			return err
		}
		s.notifyUser(ctx, u, "dates")
	}
	return nil
}

func (s *Service) CancelDate(ctx context.Context, dateID, userID, reason string) error {
	matchID, other, err := s.dateMatch(ctx, dateID, userID)
	if err != nil {
		return err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE date_plans SET status='cancelled', updated_at=now() WHERE id=$1 AND status IN ('searching','voting','scheduling','confirmed')`, dateID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	// The reason isn't posted: this note appears in Suki's name, and free text in it
	// would be a way round the message filter.
	s.chatNote(ctx, matchID, "This date plan was cancelled.", nil, map[string]any{"date_id": dateID}, "date-cancel:"+dateID)
	s.notifyUser(ctx, other, "dates")
	s.notifyUser(ctx, userID, "dates")
	return nil
}

// ── After the date ──────────────────────────────────────────────────────────

// CompleteDate marks a date as having happened and opens a private debrief for
// each person. Returns who to ask. Safe to call twice.
func (s *Service) CompleteDate(ctx context.Context, dateID string) ([][2]string, error) {
	var pairs [][2]string
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		// Guarded, so a date cancelled a moment ago stays cancelled.
		var matchID string
		err := tx.QueryRow(ctx, `UPDATE date_plans SET status='completed', updated_at=now() WHERE id=$1 AND status IN ('confirmed','completed') RETURNING match_id`, dateID).Scan(&matchID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		var a, b string
		if err := tx.QueryRow(ctx, `UPDATE matches SET stage='post_date' WHERE id=$1 RETURNING user_a, user_b`, matchID).Scan(&a, &b); err != nil {
			return err
		}
		pairs = [][2]string{{a, b}, {b, a}}
		for _, p := range pairs {
			if _, err := tx.Exec(ctx, `INSERT INTO debriefs (date_id, user_id, subject_id) VALUES ($1,$2,$3) ON CONFLICT (date_id, user_id) DO NOTHING`, dateID, p[0], p[1]); err != nil {
				return err
			}
		}
		return nil
	})
	return pairs, err
}

// Debrief is an open conversation about one date.
type Debrief struct {
	ID, DateID, SubjectID, SubjectName, VenueName string
	Notes                                         []string
}

// OpenDebrief returns this person's open debrief, if any.
func (s *Service) OpenDebrief(ctx context.Context, userID string) (*Debrief, error) {
	var d Debrief
	var notes, venue []byte
	err := s.Pool.QueryRow(ctx, `SELECT db.id, db.date_id, coalesce(db.subject_id::text,''), coalesce(p.display_name,''), db.notes, d.venue
		FROM debriefs db JOIN date_plans d ON d.id=db.date_id LEFT JOIN profiles p ON p.user_id=db.subject_id
		WHERE db.user_id=$1 AND db.status='open' ORDER BY db.created_at DESC LIMIT 1`, userID).
		Scan(&d.ID, &d.DateID, &d.SubjectID, &d.SubjectName, &notes, &venue)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(notes, &d.Notes)
	var v places.Venue
	if json.Unmarshal(venue, &v) == nil {
		d.VenueName = v.Name
	}
	return &d, nil
}

func (s *Service) AddDebriefNote(ctx context.Context, userID, note string) (string, error) {
	raw, _ := json.Marshal(note)
	var dateID string
	err := s.Pool.QueryRow(ctx, `UPDATE debriefs SET notes = notes || jsonb_build_array($2::jsonb)
		WHERE id = (SELECT id FROM debriefs WHERE user_id=$1 AND status='open' ORDER BY created_at DESC LIMIT 1) RETURNING date_id`, userID, string(raw)).Scan(&dateID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", userErr("there's no date to talk about right now")
	}
	return dateID, err
}

// CloseDebrief ends it. With nothing said it counts as skipped, and there is no
// evaluation at all: nobody is judged on silence.
func (s *Service) CloseDebrief(ctx context.Context, userID string, skip bool) (dateID string, analyzable bool, err error) {
	var notes []byte
	err = s.Pool.QueryRow(ctx, `SELECT date_id, notes FROM debriefs WHERE user_id=$1 AND status='open' ORDER BY created_at DESC LIMIT 1`, userID).Scan(&dateID, &notes)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	var list []string
	_ = json.Unmarshal(notes, &list)
	status := "closed"
	if skip || len(list) == 0 {
		status = "skipped"
	}
	_, err = s.Pool.Exec(ctx, `UPDATE debriefs SET status=$3, closed_at=now() WHERE date_id=$1 AND user_id=$2 AND status='open'`, dateID, userID, status)
	return dateID, status == "closed", err
}

// DebriefNotes returns what one person said about one date, for the analyst.
func (s *Service) DebriefNotes(ctx context.Context, dateID, userID string) (subjectID string, notes []string, err error) {
	var raw []byte
	err = s.Pool.QueryRow(ctx, `SELECT coalesce(subject_id::text,''), notes FROM debriefs WHERE date_id=$1 AND user_id=$2`, dateID, userID).Scan(&subjectID, &raw)
	if err != nil {
		return "", nil, err
	}
	_ = json.Unmarshal(raw, &notes)
	return subjectID, notes, nil
}

// MarkAnalyzed records that the analyst is done with a debrief.
func (s *Service) MarkAnalyzed(ctx context.Context, dateID, userID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE debriefs SET status='analyzed' WHERE date_id=$1 AND user_id=$2`, dateID, userID)
	return err
}

// ShareText is what the share sheet sends a trusted friend. It carries nothing
// about the other person beyond a first name and the venue.
func (s *Service) ShareText(ctx context.Context, dateID, userID string) (string, error) {
	d, err := s.DateFor(ctx, dateID, userID)
	if err != nil {
		return "", err
	}
	if d.Venue == nil || d.ScheduledAt == nil {
		return "", userErr("this date doesn't have a place and time yet")
	}
	addr := d.Venue.Address
	if addr != "" {
		addr = " (" + addr + ")"
	}
	return i18n.T(s.Lang(ctx, userID), "I'm meeting {name} at {venue}{address} on {time}. Just letting you know where I'll be.",
		"name", d.Person.DisplayName, "venue", d.Venue.Name, "address", addr, "time", d.ScheduledAt.UTC().Format("Mon Jan 2 at 15:04 UTC")), nil
}
