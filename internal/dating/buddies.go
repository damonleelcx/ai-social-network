package dating

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/damonleelcx/ai-social-network/internal/i18n"
	"github.com/damonleelcx/ai-social-network/internal/moderation"
	"github.com/damonleelcx/ai-social-network/internal/notify"
	"github.com/damonleelcx/ai-social-network/internal/profile"
	"github.com/damonleelcx/ai-social-network/internal/safety"
)

// Buddies are the fast path beside introductions: someone wants company for one plan,
// people in the same city say they're in, and the person who posted picks one. The picked
// pair gets an ordinary match, so chat, blocking and reporting all work as they always do.
// All of it happens in the conversation with Suki: she brings posts and offers as cards,
// and the person's tap on a card is what acts.
const (
	buddyNoteMax     = 200
	buddyWhenMax     = 60
	buddyMaxOpen     = 3
	buddyPostsPerDay = 5
	buddyMaxHours    = 14 * 24
	buddyFanOut      = 20
	buddyPingsPerDay = 3
	buddyBoardSize   = 50
	buddyMaxSpots    = 20
	buddyPlaceMax    = 80
)

// buddyKindNames are the kinds of plan, with the words used in messages.
var buddyKindNames = map[string]string{
	"movie": "a movie", "meal": "a meal", "show": "a show", "travel": "a trip", "sports": "sports",
	"games": "games", "business": "business", "study": "studying", "other": "a plan",
}

// BuddyInput is what someone asks for. When is their own words, shown as written: nobody's
// time zone is known here, so the app or Suki works out how long the post stays up.
//
// Spots is how many people can join: 1 (or 0) means the poster picks one, more makes it a
// group anyone it reaches can join at once. Place is where, if they know. TripID posts it in
// the city of one of their trips instead of their home city.
type BuddyInput struct {
	Kind           string `json:"kind"`
	Note           string `json:"note"`
	When           string `json:"when"`
	ExpiresInHours int    `json:"expires_in_hours"`
	SameGender     bool   `json:"same_gender"`
	Spots          int    `json:"spots,omitempty"`
	Place          string `json:"place,omitempty"`
	TripID         string `json:"trip_id,omitempty"`
}

type BuddyOffer struct {
	Person    profile.Public `json:"person"`
	CreatedAt time.Time      `json:"created_at"`
}

type BuddyRequest struct {
	ID         string          `json:"id"`
	Kind       string          `json:"kind"`
	Note       string          `json:"note"`
	When       string          `json:"when"`
	CityName   string          `json:"city_name"`
	SameGender bool            `json:"same_gender"`
	Status     string          `json:"status"`
	MatchID    string          `json:"match_id,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	ExpiresAt  time.Time       `json:"expires_at"`
	Offered    bool            `json:"offered"`
	Person     *profile.Public `json:"person,omitempty"`
	Offers     []BuddyOffer    `json:"offers,omitempty"`
	// Groups: how many can join, how many are going (the poster included), and, for the
	// people in it, who they are.
	Spots   int              `json:"spots"`
	Place   string           `json:"place,omitempty"`
	Going   int              `json:"going"`
	Joined  bool             `json:"joined,omitempty"`
	Members []profile.Public `json:"members,omitempty"`
}

// Group says whether people join this plan at once rather than being picked.
func (b *BuddyRequest) Group() bool { return b.Spots > 1 }

// BuddyMissing lists what this person still needs before they can post or join. It asks for
// less than dating does: enough for the other person to know who is coming.
func (s *Service) BuddyMissing(ctx context.Context, userID string) ([]string, error) {
	p, err := profile.Get(ctx, s.Pool, userID)
	if err != nil {
		return nil, err
	}
	var approved int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id=$1 AND status='approved'`, userID).Scan(&approved); err != nil {
		return nil, err
	}
	m := []string{}
	if p.DisplayName == "" {
		m = append(m, "display_name")
	}
	if p.Birthdate == "" {
		m = append(m, "birthdate")
	}
	if p.CityKey == "" {
		m = append(m, "city")
	}
	if approved == 0 {
		m = append(m, "photo")
	}
	return m, nil
}

func (s *Service) requireBuddyReady(ctx context.Context, userID string) error {
	if err := s.requireNotHeld(ctx, userID); err != nil {
		return err
	}
	m, err := s.BuddyMissing(ctx, userID)
	if err != nil {
		return err
	}
	if len(m) > 0 {
		return userErr("Add your name, birthday, city and a photo first, so people know who's coming.")
	}
	return nil
}

// PostBuddy opens a request and tells a few people nearby.
func (s *Service) PostBuddy(ctx context.Context, userID string, in BuddyInput) (*BuddyRequest, error) {
	if err := s.requireBuddyReady(ctx, userID); err != nil {
		return nil, err
	}
	if _, ok := buddyKindNames[in.Kind]; !ok {
		return nil, userErr("Pick what kind of plan this is.")
	}
	in.Note, in.When = strings.TrimSpace(in.Note), strings.TrimSpace(in.When)
	if utf8.RuneCountInString(in.Note) > buddyNoteMax {
		return nil, userErr("That note is too long.")
	}
	if in.When == "" || utf8.RuneCountInString(in.When) > buddyWhenMax {
		return nil, userErr("Say when, in a few words.")
	}
	if in.ExpiresInHours < 1 || in.ExpiresInHours > buddyMaxHours {
		return nil, userErr("A post can stay up from an hour to two weeks.")
	}
	if in.Spots == 0 {
		in.Spots = 1
	}
	if in.Spots < 1 || in.Spots > buddyMaxSpots {
		return nil, userErr("A group can have up to 20 people joining.")
	}
	in.Place = strings.TrimSpace(in.Place)
	if utf8.RuneCountInString(in.Place) > buddyPlaceMax {
		return nil, userErr("Keep the place to its name.")
	}
	text := in.When + "\n" + in.Note + "\n" + in.Place
	if v := s.Mod.Check(ctx, userID, moderation.Chat, text); v.Blocked {
		s.Strike(ctx, userID, "", "buddy", v, text)
		return nil, userErr("%s", moderation.BlockedMessage)
	}
	// Everyone in the city reads this, so what a chat only warns about is refused here.
	if safety.ScanChat(text) != nil {
		return nil, userErr("Keep it to the plan itself. Money requests, threats and outside contact details can't go in a post.")
	}
	p, err := profile.Get(ctx, s.Pool, userID)
	if err != nil {
		return nil, err
	}
	if in.SameGender && p.Gender == "" {
		return nil, userErr("Add your gender first, so I know who to show this to.")
	}
	// A plan for a trip goes up in the trip's city, and only while the trip isn't over.
	cityKey, cityName := p.CityKey, p.CityName
	var tripID any
	if in.TripID != "" {
		err := s.Pool.QueryRow(ctx, `SELECT city_key, city_name FROM trips WHERE id=$1 AND user_id=$2 AND ends_on>=current_date`,
			in.TripID, userID).Scan(&cityKey, &cityName)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, userErr("That trip is over or isn't yours.")
		}
		if err != nil {
			return nil, err
		}
		tripID = in.TripID
	}
	var id string
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		// One at a time per person, so two quick posts can't both pass the limits.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('buddy:' || $1::text))`, userID); err != nil {
			return err
		}
		var open, today int
		if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status='open' AND expires_at>now()), count(*) FILTER (WHERE created_at>now()-interval '1 day')
			FROM buddy_requests WHERE user_id=$1`, userID).Scan(&open, &today); err != nil {
			return err
		}
		if open >= buddyMaxOpen {
			return userErr("You already have %d posts up. Close one first.", buddyMaxOpen)
		}
		if today >= buddyPostsPerDay {
			return userErr("That's enough posts for today. Try again tomorrow.")
		}
		return tx.QueryRow(ctx, `INSERT INTO buddy_requests (user_id, kind, note, when_text, city_key, city_name, same_gender, gender, expires_at, spots, place, trip_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8, now() + make_interval(hours => $9::int), $10, $11, $12) RETURNING id`,
			userID, in.Kind, in.Note, in.When, cityKey, cityName, in.SameGender, p.Gender, in.ExpiresInHours, in.Spots, in.Place, tripID).Scan(&id)
	})
	if err != nil {
		return nil, err
	}
	s.tellNearby(ctx, userID, id, cityKey, in, p)
	s.notifyUser(ctx, userID, "buddies")
	return s.buddyRequest(ctx, id, userID)
}

// tellNearby pings a few recently active people. A post is on the board whether or not this
// reaches anyone, and nobody gets more than a few of these a day.
func (s *Service) tellNearby(ctx context.Context, userID, requestID, cityKey string, in BuddyInput, p *profile.Profile) {
	ctx = context.WithoutCancel(ctx)
	// The people who live there, and the people whose trip there is on while the post is up.
	rows, err := s.Pool.Query(ctx, `SELECT p.user_id FROM profiles p JOIN users u ON u.id=p.user_id
		WHERE (p.city_key=$2 OR EXISTS (SELECT 1 FROM trips t WHERE t.user_id=p.user_id AND t.city_key=$2
		       AND t.ends_on>=current_date AND t.starts_on<=(now()+make_interval(hours => $7::int))::date))
		  AND p.user_id<>$1 AND p.status<>'held' AND u.deleted_at IS NULL AND u.email_verified_at IS NOT NULL
		  AND p.last_active_at > now()-interval '14 days'
		  AND (NOT $3::bool OR p.gender=$4)
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id=p.user_id AND ph.status='approved')
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id=$1 AND b.blocked_id=p.user_id) OR (b.blocker_id=p.user_id AND b.blocked_id=$1))
		  AND (SELECT count(*) FROM notifications n WHERE n.user_id=p.user_id AND n.kind='buddy' AND n.created_at>now()-interval '1 day') < $5
		ORDER BY p.last_active_at DESC LIMIT $6`, userID, cityKey, in.SameGender, p.Gender, buddyPingsPerDay, buddyFanOut, in.ExpiresInHours)
	if err != nil {
		slog.Error("buddy fan-out", "err", err)
		return
	}
	var people []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			people = append(people, id)
		}
	}
	rows.Close()
	for _, id := range people {
		lang := s.Lang(ctx, id)
		line := "{name} is looking for company for {plan}: {when}."
		if in.Spots > 1 {
			line = "{name} is getting a group together for {plan}: {when}."
		}
		body := i18n.T(lang, line, "name", p.DisplayName, "plan", i18n.T(lang, buddyKindNames[in.Kind]), "when", in.When)
		if err := s.PostFromSuki(ctx, id, body, "playful", []map[string]any{{"kind": "buddy_request", "request_id": requestID}}, "buddy:"+requestID+":"+id); err != nil {
			slog.Error("buddy announce", "err", err)
		}
		if err := notify.Send(ctx, s.Pool, id, "buddy", "Someone nearby wants company", body,
			map[string]any{"screen": "agent"}, "buddy:"+requestID+":"+id); err != nil {
			slog.Error("buddy notify", "err", err)
		}
	}
}

const buddyCols = `r.id, r.user_id, r.kind, r.note, r.when_text, r.city_name, r.same_gender,
	CASE WHEN r.status='open' AND r.expires_at<=now() THEN 'closed' ELSE r.status END,
	coalesce(r.match_id::text,''), r.created_at, r.expires_at, r.spots, r.place,
	1 + (SELECT count(*) FROM buddy_members bm WHERE bm.request_id=r.id)`

// buddyDest is where buddyCols scan to, with any extra columns after them.
func buddyDest(b *BuddyRequest, owner *string, extra ...any) []any {
	return append([]any{&b.ID, owner, &b.Kind, &b.Note, &b.When, &b.CityName, &b.SameGender, &b.Status, &b.MatchID, &b.CreatedAt, &b.ExpiresAt,
		&b.Spots, &b.Place, &b.Going}, extra...)
}

func scanBuddy(row pgx.Row) (*BuddyRequest, string, error) {
	var b BuddyRequest
	var owner string
	err := row.Scan(buddyDest(&b, &owner)...)
	return &b, owner, err
}

func (s *Service) publicOf(ctx context.Context, userID string) (*profile.Public, error) {
	p, err := profile.Get(ctx, s.Pool, userID)
	if err != nil {
		return nil, err
	}
	ids, err := profile.ApprovedPhotoIDs(ctx, s.Pool, userID)
	if err != nil {
		return nil, err
	}
	pub := p.Public(ids)
	return &pub, nil
}

// buddyVisible is the one rule for who can see a post: open, same city, the same gender if
// that was asked for, no block either way, and no match between them that already ended.
// $1 is the viewer. The query it goes in joins buddy_requests r and profiles me.
//
// Same city means their home city, or a city they have a trip to that's on before the post
// comes down. In a group, a block with anyone already in it hides it too.
const buddyVisible = `r.status='open' AND r.expires_at>now() AND r.user_id<>$1
	AND ((r.city_key=me.city_key AND me.city_key<>'')
	     OR EXISTS (SELECT 1 FROM trips t WHERE t.user_id=$1 AND t.city_key=r.city_key AND t.ends_on>=current_date AND t.starts_on<=r.expires_at::date))
	AND (NOT r.same_gender OR r.gender=me.gender)
	AND EXISTS (SELECT 1 FROM profiles o JOIN users u ON u.id=o.user_id WHERE o.user_id=r.user_id AND o.status<>'held' AND u.deleted_at IS NULL)
	AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id=$1 AND b.blocked_id=r.user_id) OR (b.blocker_id=r.user_id AND b.blocked_id=$1))
	AND NOT EXISTS (SELECT 1 FROM buddy_members bm JOIN blocks b ON (b.blocker_id=$1 AND b.blocked_id=bm.user_id) OR (b.blocker_id=bm.user_id AND b.blocked_id=$1)
	     WHERE bm.request_id=r.id)
	AND NOT EXISTS (SELECT 1 FROM matches m WHERE m.user_a=least($1::uuid,r.user_id) AND m.user_b=greatest($1::uuid,r.user_id) AND m.status<>'active')`

// BuddyBoard is what this person can join. Until they're ready it stays empty: a person
// with no photo shouldn't be able to look through people who have one.
func (s *Service) BuddyBoard(ctx context.Context, userID string) ([]*BuddyRequest, []string, error) {
	missing, err := s.BuddyMissing(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	out := []*BuddyRequest{}
	if len(missing) > 0 {
		return out, missing, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+buddyCols+`, `+buddyIn+`
		FROM buddy_requests r JOIN profiles me ON me.user_id=$1
		WHERE `+buddyVisible+` ORDER BY r.created_at DESC LIMIT $2`, userID, buddyBoardSize)
	if err != nil {
		return nil, nil, err
	}
	owners := []string{}
	for rows.Next() {
		var b BuddyRequest
		var owner string
		if err := rows.Scan(buddyDest(&b, &owner, &b.Offered)...); err != nil {
			rows.Close()
			return nil, nil, err
		}
		out = append(out, &b)
		owners = append(owners, owner)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	for i, b := range out {
		if b.Person, err = s.publicOf(ctx, owners[i]); err != nil {
			return nil, nil, err
		}
	}
	return out, missing, nil
}

// MyBuddyRequests is this person's own posts, with who has said they're in.
func (s *Service) MyBuddyRequests(ctx context.Context, userID string) ([]*BuddyRequest, error) {
	rows, err := s.Pool.Query(ctx, `SELECT r.id FROM buddy_requests r WHERE r.user_id=$1 ORDER BY r.created_at DESC LIMIT 20`, userID)
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
	out := []*BuddyRequest{}
	for _, id := range ids {
		b, err := s.buddyRequest(ctx, id, userID)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// buddyRequest is the owner's view of one post.
func (s *Service) buddyRequest(ctx context.Context, id, ownerID string) (*BuddyRequest, error) {
	b, _, err := scanBuddy(s.Pool.QueryRow(ctx, `SELECT `+buddyCols+` FROM buddy_requests r WHERE r.id=$1 AND r.user_id=$2`, id, ownerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if b.Group() {
		// The group chat shows who posted it, the poster included, so it knows whose plan it is.
		b.Joined = true
		if b.Person, err = s.publicOf(ctx, ownerID); err != nil {
			return nil, err
		}
		b.Members, err = s.groupMembers(ctx, id, ownerID)
		return b, err
	}
	if b.Status != "open" {
		return b, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT o.user_id, o.created_at FROM buddy_offers o WHERE o.request_id=$1
		AND NOT EXISTS (SELECT 1 FROM blocks x WHERE (x.blocker_id=$2 AND x.blocked_id=o.user_id) OR (x.blocker_id=o.user_id AND x.blocked_id=$2))
		AND NOT EXISTS (SELECT 1 FROM profiles h WHERE h.user_id=o.user_id AND h.status='held')
		ORDER BY o.created_at`, id, ownerID)
	if err != nil {
		return nil, err
	}
	type offer struct {
		user string
		at   time.Time
	}
	var offers []offer
	for rows.Next() {
		var o offer
		if err := rows.Scan(&o.user, &o.at); err != nil {
			rows.Close()
			return nil, err
		}
		offers = append(offers, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, o := range offers {
		pub, err := s.publicOf(ctx, o.user)
		if err != nil {
			return nil, err
		}
		b.Offers = append(b.Offers, BuddyOffer{Person: *pub, CreatedAt: o.at})
	}
	return b, nil
}

// BuddyView is one post for one of Suki's cards: the owner's view with who's in, or
// the view anyone it's visible to gets. Anyone else, and a post that has closed, gets not found.
func (s *Service) BuddyView(ctx context.Context, id, viewerID string) (*BuddyRequest, error) {
	if b, err := s.buddyRequest(ctx, id, viewerID); err == nil {
		return b, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if b, err := s.memberView(ctx, id, viewerID); err == nil {
		return b, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	var b BuddyRequest
	var owner string
	err := s.Pool.QueryRow(ctx, `SELECT `+buddyCols+`, `+buddyIn+`
		FROM buddy_requests r JOIN profiles me ON me.user_id=$1 WHERE r.id=$2 AND `+buddyVisible, viewerID, id).
		Scan(buddyDest(&b, &owner, &b.Offered)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if b.Person, err = s.publicOf(ctx, owner); err != nil {
		return nil, err
	}
	return &b, nil
}

// OfferBuddy says "I'm in". The poster hears about it and decides.
func (s *Service) OfferBuddy(ctx context.Context, requestID, userID string) error {
	if err := s.requireBuddyReady(ctx, userID); err != nil {
		return err
	}
	var owner, when, kind string
	var spots int
	err := s.Pool.QueryRow(ctx, `SELECT r.user_id, r.when_text, r.kind, r.spots FROM buddy_requests r JOIN profiles me ON me.user_id=$1
		WHERE r.id=$2 AND `+buddyVisible, userID, requestID).Scan(&owner, &when, &kind, &spots)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if spots > 1 {
		return s.joinGroup(ctx, requestID, userID)
	}
	tag, err := s.Pool.Exec(ctx, `INSERT INTO buddy_offers (request_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, requestID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	name, err := s.displayName(ctx, userID)
	if err != nil {
		return err
	}
	lang := s.Lang(ctx, owner)
	body := i18n.T(lang, "{name} wants to join you for {plan}.", "name", name, "plan", i18n.T(lang, buddyKindNames[kind]))
	if err := s.PostFromSuki(ctx, owner, body, "happy", []map[string]any{{"kind": "buddy_offer", "request_id": requestID, "user_id": userID}}, "buddy-offer:"+requestID+":"+userID); err != nil {
		slog.Error("buddy offer announce", "err", err)
	}
	if err := notify.Send(ctx, s.Pool, owner, "buddy_offer", "Someone's in", body,
		map[string]any{"screen": "agent"}, "buddy-offer:"+requestID+":"+userID); err != nil {
		slog.Error("buddy offer notify", "err", err)
	}
	s.notifyUser(ctx, owner, "buddies")
	return nil
}

// WithdrawBuddyOffer takes an offer back while the post is still open.
// In a group, it's leaving the group.
func (s *Service) WithdrawBuddyOffer(ctx context.Context, requestID, userID string) error {
	if left, err := s.leaveGroup(ctx, requestID, userID); err != nil || left {
		return err
	}
	var owner string
	err := s.Pool.QueryRow(ctx, `DELETE FROM buddy_offers o USING buddy_requests r WHERE o.request_id=r.id AND o.request_id=$1 AND o.user_id=$2 AND r.status='open' RETURNING r.user_id`,
		requestID, userID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	s.notifyUser(ctx, owner, "buddies")
	return nil
}

// CloseBuddy takes a post down. A group can be called off even once it's full, and
// everyone in it is told.
func (s *Service) CloseBuddy(ctx context.Context, requestID, userID string) error {
	var kind string
	var spots int
	err := s.Pool.QueryRow(ctx, `UPDATE buddy_requests SET status='closed' WHERE id=$1 AND user_id=$2
		AND (status='open' OR (status='filled' AND spots>1)) RETURNING kind, spots`, requestID, userID).Scan(&kind, &spots)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	s.notifyUser(ctx, userID, "buddies")
	if spots > 1 {
		s.groupCalledOff(ctx, requestID, userID, kind)
	}
	return nil
}

// PickBuddy is the poster choosing someone who said they're in. Both have now said yes to
// each other, so this makes the same match two yeses to an introduction would, and a pair
// who already have a match keep it. A match that ended stays ended.
func (s *Service) PickBuddy(ctx context.Context, requestID, ownerID, pickedID string) (string, error) {
	if err := s.requireNotHeld(ctx, ownerID); err != nil {
		return "", err
	}
	a, b := pairKey(ownerID, pickedID)
	blocked, err := s.blockedEitherWay(ctx, a, b)
	if err != nil {
		return "", err
	}
	if blocked {
		return "", ErrNotFound
	}
	var kind, matchID string
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT r.kind FROM buddy_requests r JOIN buddy_offers o ON o.request_id=r.id AND o.user_id=$3
			JOIN profiles h ON h.user_id=o.user_id AND h.status<>'held'
			WHERE r.id=$1 AND r.user_id=$2 AND r.status='open' AND r.expires_at>now() FOR UPDATE OF r`, requestID, ownerID, pickedID).Scan(&kind)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var ended bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM matches WHERE user_a=$1 AND user_b=$2 AND status<>'active')`, a, b).Scan(&ended); err != nil {
			return err
		}
		if ended {
			return ErrNotFound
		}
		var proposalID string
		if err := tx.QueryRow(ctx, `INSERT INTO proposals (user_a, user_b, decision_a, decision_b, status) VALUES ($1,$2,'accept','accept','matched')
			ON CONFLICT (user_a, user_b) DO UPDATE SET decision_a='accept', decision_b='accept', status='matched' RETURNING id`, a, b).Scan(&proposalID); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `INSERT INTO matches (proposal_id, user_a, user_b, buddy_kind) VALUES ($1,$2,$3,$4)
			ON CONFLICT (proposal_id) DO UPDATE SET proposal_id=EXCLUDED.proposal_id RETURNING id`, proposalID, a, b, kind).Scan(&matchID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE buddy_requests SET status='filled', match_id=$2 WHERE id=$1`, requestID, matchID)
		return err
	})
	if err != nil {
		return "", err
	}
	for _, x := range [][2]string{{ownerID, pickedID}, {pickedID, ownerID}} {
		name, err := s.displayName(ctx, x[1])
		if err != nil {
			slog.Error("buddy announce", "err", err)
			continue
		}
		lang := s.Lang(ctx, x[0])
		plan := i18n.T(lang, buddyKindNames[kind])
		body := i18n.T(lang, "You and {name} are on for {plan}! Say hi and sort out the details. For a first meet-up, pick somewhere public and tell a friend.", "name", name, "plan", plan)
		if err := s.PostFromSuki(ctx, x[0], body, "happy", []map[string]any{{"kind": "match", "match_id": matchID}}, "buddy:"+requestID); err != nil {
			slog.Error("buddy announce", "err", err)
		}
		if err := notify.Send(ctx, s.Pool, x[0], "match", "You've got company", i18n.T(lang, "You and {name} are on for {plan}!", "name", name, "plan", plan),
			map[string]any{"screen": "chat", "id": matchID}, "buddy-match:"+requestID+":"+x[0]); err != nil {
			slog.Error("buddy notify", "err", err)
		}
		s.notifyUser(ctx, x[0], "matches")
		s.notifyUser(ctx, x[0], "buddies")
	}
	return matchID, nil
}
