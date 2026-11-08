package dating

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/damonleelcx/ai-social-network/internal/moderation"
	"github.com/damonleelcx/ai-social-network/internal/places"
	"github.com/damonleelcx/ai-social-network/internal/safety"
)

// Trips: someone says they'll be in another city for some days. While the trip is on or
// about to start, plans posted there reach them and they can post plans there, so they can
// line up company before they land. Nobody is shown as a name on a trip: other travellers see
// only how many people are going, and meet through plans, where everyone chooses to join.
const (
	tripMaxOpen    = 5
	tripMaxDays    = 90
	tripMaxAhead   = 365
	tripNoteMax    = 200
	tripTrendDays  = 30
	tripTrendLimit = 10
)

type TripInput struct {
	City     string `json:"city"`
	StartsOn string `json:"starts_on"` // yyyy-mm-dd
	EndsOn   string `json:"ends_on"`
	Note     string `json:"note,omitempty"`
}

type Trip struct {
	ID       string `json:"id"`
	CityName string `json:"city_name"`
	StartsOn string `json:"starts_on"`
	EndsOn   string `json:"ends_on"`
	Note     string `json:"note,omitempty"`
	// Going is how many other people have a trip there at the same time.
	Going int `json:"going"`
	// Plans is how many open plans there are there now that this person could join.
	Plans int `json:"plans"`
	// Scene is the key of the city's picture, once it's drawn.
	Scene string `json:"scene,omitempty"`
}

// TripCity is a city with how many people are travelling there soon.
type TripCity struct {
	CityName string `json:"city_name"`
	Going    int    `json:"going"`
}

func parseDay(s string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(s))
}

// CheckTrip validates what someone asked for before the city is looked up, so a bad date
// doesn't cost a lookup.
func CheckTrip(in TripInput) error {
	from, err1 := parseDay(in.StartsOn)
	to, err2 := parseDay(in.EndsOn)
	if err1 != nil || err2 != nil {
		return userErr("Give the dates as year, month and day.")
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	switch {
	case to.Before(from):
		return userErr("The trip has to end after it starts.")
	case to.Before(today.AddDate(0, 0, -1)):
		return userErr("That trip is already over.")
	case from.After(today.AddDate(0, 0, tripMaxAhead)):
		return userErr("Add trips up to a year ahead.")
	case to.Sub(from) > tripMaxDays*24*time.Hour:
		return userErr("A trip can be up to 90 days. For longer, change your city instead.")
	}
	if utf8.RuneCountInString(strings.TrimSpace(in.Note)) > tripNoteMax {
		return userErr("That note is too long.")
	}
	if strings.TrimSpace(in.City) == "" {
		return userErr("Which city?")
	}
	return nil
}

// AddTrip saves a trip to a city the caller has already found.
func (s *Service) AddTrip(ctx context.Context, userID string, city *places.City, in TripInput) (*Trip, error) {
	if err := s.requireNotHeld(ctx, userID); err != nil {
		return nil, err
	}
	if err := CheckTrip(in); err != nil {
		return nil, err
	}
	in.Note = strings.TrimSpace(in.Note)
	if in.Note != "" {
		if v := s.Mod.Check(ctx, userID, moderation.Chat, in.Note); v.Blocked {
			s.Strike(ctx, userID, "", "trip", v, in.Note)
			return nil, userErr("%s", moderation.BlockedMessage)
		}
		if safety.ScanChat(in.Note) != nil {
			return nil, userErr("Keep it to the trip itself. Money requests, threats and outside contact details can't go in a note.")
		}
	}
	var home string
	if err := s.Pool.QueryRow(ctx, `SELECT city_key FROM profiles WHERE user_id=$1`, userID).Scan(&home); err != nil {
		return nil, err
	}
	if home != "" && home == city.Key {
		return nil, userErr("That's your own city. A trip is for somewhere else.")
	}
	var id string
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('trip:' || $1::text))`, userID); err != nil {
			return err
		}
		var open int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM trips WHERE user_id=$1 AND ends_on>=current_date`, userID).Scan(&open); err != nil {
			return err
		}
		if open >= tripMaxOpen {
			return userErr("You already have 5 trips coming up. Remove one first.")
		}
		return tx.QueryRow(ctx, `INSERT INTO trips (user_id, city_key, city_name, lat, lon, starts_on, ends_on, note)
			VALUES ($1,$2,$3,$4,$5,$6::date,$7::date,$8) RETURNING id`,
			userID, city.Key, city.Name, city.Lat, city.Lon, strings.TrimSpace(in.StartsOn), strings.TrimSpace(in.EndsOn), in.Note).Scan(&id)
	})
	if err != nil {
		return nil, err
	}
	s.notifyUser(ctx, userID, "trips")
	if s.Scenes != nil {
		s.Scenes.Trip(id)
	}
	return s.trip(ctx, id, userID)
}

// tripCols reads a trip t for its owner $2, with the others there at the same time and the
// plans they could join there now. Counts only: travellers aren't listed by name.
var tripCols = `t.id, t.city_name, to_char(t.starts_on,'YYYY-MM-DD'), to_char(t.ends_on,'YYYY-MM-DD'), t.note,
	(SELECT count(DISTINCT o.user_id) FROM trips o JOIN profiles p ON p.user_id=o.user_id JOIN users u ON u.id=o.user_id
	  WHERE o.city_key=t.city_key AND o.user_id<>$2 AND o.starts_on<=t.ends_on AND o.ends_on>=t.starts_on AND o.ends_on>=current_date
	    AND p.status<>'held' AND u.deleted_at IS NULL
	    AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id=$2 AND b.blocked_id=o.user_id) OR (b.blocker_id=o.user_id AND b.blocked_id=$2))),
	(SELECT count(*) FROM buddy_requests r JOIN profiles me ON me.user_id=$2 WHERE r.city_key=t.city_key AND ` + tripPlanVisible + `),
	coalesce((SELECT si.key FROM scene_images si WHERE si.key=t.scene_key AND si.status='ready'),'')`

// tripPlanVisible is buddyVisible for the plans in a trip's city: the same rule, with $2 as
// the viewer.
var tripPlanVisible = strings.ReplaceAll(buddyVisible, "$1", "$2")

func scanTrip(row pgx.Row) (*Trip, error) {
	var t Trip
	err := row.Scan(&t.ID, &t.CityName, &t.StartsOn, &t.EndsOn, &t.Note, &t.Going, &t.Plans, &t.Scene)
	return &t, err
}

func (s *Service) trip(ctx context.Context, id, userID string) (*Trip, error) {
	t, err := scanTrip(s.Pool.QueryRow(ctx, `SELECT `+tripCols+` FROM trips t WHERE t.id=$1 AND t.user_id=$2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

// MyTrips is this person's trips that aren't over, soonest first.
func (s *Service) MyTrips(ctx context.Context, userID string) ([]*Trip, error) {
	rows, err := s.Pool.Query(ctx, `SELECT t.id FROM trips t WHERE t.user_id=$1 AND t.ends_on>=current_date ORDER BY t.starts_on LIMIT 20`, userID)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	out := []*Trip{}
	for _, id := range ids {
		t, err := s.trip(ctx, id, userID)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// TripView is one of this person's own trips, for Suki's card.
func (s *Service) TripView(ctx context.Context, id, userID string) (*Trip, error) {
	return s.trip(ctx, id, userID)
}

// RemoveTrip deletes one of this person's trips. Plans they posted for it stay up.
func (s *Service) RemoveTrip(ctx context.Context, id, userID string) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM trips WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	s.notifyUser(ctx, userID, "trips")
	return nil
}

// TrendingTrips is where people are going in the next few weeks, busiest first.
func (s *Service) TrendingTrips(ctx context.Context) ([]TripCity, error) {
	rows, err := s.Pool.Query(ctx, `SELECT min(t.city_name), count(DISTINCT t.user_id) AS going
		FROM trips t JOIN profiles p ON p.user_id=t.user_id JOIN users u ON u.id=t.user_id
		WHERE t.ends_on>=current_date AND t.starts_on<=current_date+$1::int AND p.status<>'held' AND u.deleted_at IS NULL
		GROUP BY t.city_key ORDER BY going DESC, min(t.city_name) LIMIT $2`, tripTrendDays, tripTrendLimit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (TripCity, error) {
		var c TripCity
		err := r.Scan(&c.CityName, &c.Going)
		return c, err
	})
}

// TripCityOf is where a plan for one of this person's trips is, for finding places there.
func (s *Service) TripCityOf(ctx context.Context, id, userID string) (*places.City, error) {
	var c places.City
	err := s.Pool.QueryRow(ctx, `SELECT city_key, city_name, lat, lon FROM trips WHERE id=$1 AND user_id=$2 AND ends_on>=current_date`, id, userID).
		Scan(&c.Key, &c.Name, &c.Lat, &c.Lon)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

// HomeCity is where this person lives, or nil before they've said.
func (s *Service) HomeCity(ctx context.Context, userID string) (*places.City, error) {
	var c places.City
	err := s.Pool.QueryRow(ctx, `SELECT city_key, city_name, coalesce(city_lat,0), coalesce(city_lon,0) FROM profiles WHERE user_id=$1`, userID).
		Scan(&c.Key, &c.Name, &c.Lat, &c.Lon)
	if err != nil {
		return nil, err
	}
	if c.Key == "" {
		return nil, nil
	}
	return &c, nil
}
