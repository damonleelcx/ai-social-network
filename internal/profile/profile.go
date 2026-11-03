// Package profile is what a person chooses to say about themselves. Both Suki's
// tools and the settings screen write through Save, so one set of rules applies.
package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Prompt struct {
	Q string `json:"q"`
	A string `json:"a"`
}

type Profile struct {
	UserID      string              `json:"user_id"`
	DisplayName string              `json:"display_name"`
	Birthdate   string              `json:"birthdate"` // yyyy-mm-dd, or ""
	Age         int                 `json:"age"`
	Gender      string              `json:"gender"`
	Pronouns    string              `json:"pronouns"`
	Seeking     []string            `json:"seeking"`
	CityKey     string              `json:"city_key"`
	CityName    string              `json:"city_name"`
	CityLat     *float64            `json:"-"`
	CityLon     *float64            `json:"-"`
	AgeMin      int                 `json:"age_min"`
	AgeMax      int                 `json:"age_max"`
	Goal        string              `json:"goal"`
	Bio         string              `json:"bio"`
	BioDraft    string              `json:"bio_draft"`
	Occupation  string              `json:"occupation"`
	Education   string              `json:"education"`
	HeightCM    *int                `json:"height_cm"`
	Smoking     string              `json:"smoking"`
	Drinking    string              `json:"drinking"`
	Kids        string              `json:"kids"`
	WantsKids   string              `json:"wants_kids"`
	Religion    string              `json:"religion"`
	Politics    string              `json:"politics"`
	Languages   []string            `json:"languages"`
	Interests   []string            `json:"interests"`
	Prompts     []Prompt            `json:"prompts"`
	Dealbreaker map[string][]string `json:"dealbreakers"`
	Status      string              `json:"status"`
}

// Public is what another person may see, and only someone Suki proposed to them
// or matched them with. No birthdate, no exact age range settings, no dealbreakers.
type Public struct {
	UserID      string   `json:"user_id"`
	DisplayName string   `json:"display_name"`
	Age         int      `json:"age"`
	Gender      string   `json:"gender"`
	Pronouns    string   `json:"pronouns"`
	CityName    string   `json:"city_name"`
	Goal        string   `json:"goal"`
	Bio         string   `json:"bio"`
	Occupation  string   `json:"occupation"`
	Education   string   `json:"education"`
	HeightCM    *int     `json:"height_cm"`
	Smoking     string   `json:"smoking"`
	Drinking    string   `json:"drinking"`
	Kids        string   `json:"kids"`
	WantsKids   string   `json:"wants_kids"`
	Languages   []string `json:"languages"`
	Interests   []string `json:"interests"`
	Prompts     []Prompt `json:"prompts"`
	PhotoIDs    []string `json:"photo_ids"`
}

func (p *Profile) Public(photoIDs []string) Public {
	if photoIDs == nil {
		photoIDs = []string{}
	}
	return Public{UserID: p.UserID, DisplayName: p.DisplayName, Age: p.Age, Gender: p.Gender, Pronouns: p.Pronouns, CityName: p.CityName,
		Goal: p.Goal, Bio: p.Bio, Occupation: p.Occupation, Education: p.Education, HeightCM: p.HeightCM, Smoking: p.Smoking,
		Drinking: p.Drinking, Kids: p.Kids, WantsKids: p.WantsKids, Languages: p.Languages, Interests: p.Interests, Prompts: p.Prompts, PhotoIDs: photoIDs}
}

const Cols = `p.user_id, p.display_name, coalesce(to_char(p.birthdate,'YYYY-MM-DD'),''), p.gender, p.pronouns, p.seeking, p.city_key, p.city_name, p.city_lat, p.city_lon,
	p.age_min, p.age_max, p.goal, p.bio, p.bio_draft, p.occupation, p.education, p.height_cm, p.smoking, p.drinking, p.kids, p.wants_kids, p.religion, p.politics,
	p.languages, p.interests, p.prompts, p.dealbreakers, p.status`

// Scan reads one row selected with Cols.
func Scan(row pgx.Row) (*Profile, error) {
	var p Profile
	var prompts, deal []byte
	err := row.Scan(&p.UserID, &p.DisplayName, &p.Birthdate, &p.Gender, &p.Pronouns, &p.Seeking, &p.CityKey, &p.CityName, &p.CityLat, &p.CityLon,
		&p.AgeMin, &p.AgeMax, &p.Goal, &p.Bio, &p.BioDraft, &p.Occupation, &p.Education, &p.HeightCM, &p.Smoking, &p.Drinking, &p.Kids, &p.WantsKids, &p.Religion, &p.Politics,
		&p.Languages, &p.Interests, &prompts, &deal, &p.Status)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(prompts, &p.Prompts)
	_ = json.Unmarshal(deal, &p.Dealbreaker)
	if p.Prompts == nil {
		p.Prompts = []Prompt{}
	}
	if p.Dealbreaker == nil {
		p.Dealbreaker = map[string][]string{}
	}
	p.Age = AgeOn(p.Birthdate, time.Now())
	return &p, nil
}

// AgeOn is a person's age in whole years, or 0 when the birthdate is unknown.
func AgeOn(birthdate string, now time.Time) int {
	b, err := time.Parse("2006-01-02", birthdate)
	if err != nil {
		return 0
	}
	age := now.Year() - b.Year()
	if now.Month() < b.Month() || (now.Month() == b.Month() && now.Day() < b.Day()) {
		age--
	}
	return age
}

// Get returns the profile, creating an empty one the first time.
func Get(ctx context.Context, pool *pgxpool.Pool, userID string) (*Profile, error) {
	if _, err := pool.Exec(ctx, `INSERT INTO profiles (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return nil, err
	}
	return Scan(pool.QueryRow(ctx, `SELECT `+Cols+` FROM profiles p WHERE p.user_id=$1`, userID))
}

// ValidationError is a message for the person (or, through a tool, for Suki).
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func invalid(f string, a ...any) error { return &ValidationError{fmt.Sprintf(f, a...)} }

// ErrUnderage stops onboarding. The caller holds the account for a moderator.
var ErrUnderage = errors.New("SynAgent is only for adults aged 18 or over")

var enums = map[string][]string{
	"gender":     {"woman", "man", "nonbinary", "other"},
	"goal":       {"long_term", "marriage", "short_term", "casual", "friends_first", "not_sure"},
	"smoking":    {"no", "sometimes", "yes"},
	"drinking":   {"no", "sometimes", "yes"},
	"kids":       {"none", "have"},
	"wants_kids": {"yes", "no", "open"},
}

var textLimits = map[string]int{"display_name": 40, "pronouns": 30, "bio": 600, "bio_draft": 600, "occupation": 80, "education": 80, "religion": 40, "politics": 40}

var column = map[string]string{"display_name": "display_name", "pronouns": "pronouns", "bio": "bio", "bio_draft": "bio_draft", "occupation": "occupation",
	"education": "education", "religion": "religion", "politics": "politics", "gender": "gender", "goal": "goal", "smoking": "smoking", "drinking": "drinking",
	"kids": "kids", "wants_kids": "wants_kids"}

func strs(v any) ([]string, bool) {
	switch x := v.(type) {
	case []string:
		return x, true
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			s, ok := e.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	}
	return nil, false
}

func num(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case json.Number:
		n, err := x.Int64()
		return int(n), err == nil
	}
	return 0, false
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// TextCheck rejects NSFW profile text. It is set at startup by internal/workflows,
// and returns the error to show the person, or nil.
var TextCheck func(ctx context.Context, userID, text string) error

// patchText gathers everything in a patch that other people could read.
func patchText(patch map[string]any) string {
	var b strings.Builder
	for _, k := range []string{"display_name", "pronouns", "bio", "bio_draft", "occupation", "education", "religion", "politics"} {
		if s, ok := patch[k].(string); ok {
			b.WriteString(s + "\n")
		}
	}
	for _, k := range []string{"interests", "languages"} {
		if l, ok := strs(patch[k]); ok {
			b.WriteString(strings.Join(l, ", ") + "\n")
		}
	}
	if v, ok := patch["prompts"]; ok {
		raw, _ := json.Marshal(v)
		var ps []Prompt
		if json.Unmarshal(raw, &ps) == nil {
			for _, x := range ps {
				b.WriteString(x.Q + " " + x.A + "\n")
			}
		}
	}
	return b.String()
}

// Save applies a partial update. Unknown keys are refused rather than ignored,
// so a mistake by a client or the model is visible. city_* fields are set only by
// SetCity, from a resolved place.
func Save(ctx context.Context, pool *pgxpool.Pool, userID string, patch map[string]any) (*Profile, error) {
	cur, err := Get(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	if TextCheck != nil {
		if text := patchText(patch); strings.TrimSpace(text) != "" {
			if err := TextCheck(ctx, userID, text); err != nil {
				return nil, err
			}
		}
	}
	sets := []string{}
	args := []any{userID}
	set := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
	}
	ageMin, ageMax := cur.AgeMin, cur.AgeMax
	for k, v := range patch {
		switch k {
		case "display_name", "pronouns", "bio", "bio_draft", "occupation", "education", "religion", "politics":
			s, ok := v.(string)
			if !ok {
				return nil, invalid("%s must be text", k)
			}
			s = strings.TrimSpace(s)
			if utf8.RuneCountInString(s) > textLimits[k] {
				return nil, invalid("%s is too long (limit %d characters)", k, textLimits[k])
			}
			if k == "display_name" && s == "" {
				return nil, invalid("a name can't be empty")
			}
			set(column[k], s)
		case "gender", "goal", "smoking", "drinking", "kids", "wants_kids":
			s, ok := v.(string)
			if !ok || !contains(enums[k], s) {
				return nil, invalid("%s must be one of %s", k, strings.Join(enums[k], ", "))
			}
			set(column[k], s)
		case "birthdate":
			s, _ := v.(string)
			if cur.Birthdate != "" && cur.Birthdate != s {
				return nil, invalid("a birthdate can only be set once; contact support if it is wrong")
			}
			b, err := time.Parse("2006-01-02", s)
			if err != nil {
				return nil, invalid("birthdate must look like 1996-03-02")
			}
			age := AgeOn(s, time.Now())
			if age < 18 {
				if _, err := pool.Exec(ctx, `UPDATE profiles SET status='held', updated_at=now() WHERE user_id=$1`, userID); err != nil {
					return nil, err
				}
				return nil, ErrUnderage
			}
			if age > 110 {
				return nil, invalid("that birthdate doesn't look right")
			}
			set("birthdate", b)
		case "seeking":
			l, ok := strs(v)
			if !ok || len(l) == 0 {
				return nil, invalid("seeking must list at least one of woman, man, nonbinary, other")
			}
			for _, x := range l {
				if !contains(enums["gender"], x) {
					return nil, invalid("seeking must list woman, man, nonbinary or other")
				}
			}
			set("seeking", l)
		case "age_min", "age_max":
			n, ok := num(v)
			if !ok || n < 18 || n > 120 {
				return nil, invalid("%s must be a number from 18 to 120", k)
			}
			if k == "age_min" {
				ageMin = n
			} else {
				ageMax = n
			}
			set(k, n)
		case "height_cm":
			n, ok := num(v)
			if !ok || n < 120 || n > 230 {
				return nil, invalid("height_cm must be between 120 and 230")
			}
			set("height_cm", n)
		case "languages", "interests":
			l, ok := strs(v)
			if !ok {
				return nil, invalid("%s must be a list", k)
			}
			l = normList(l)
			if len(l) > 25 {
				return nil, invalid("%s can hold at most 25 entries", k)
			}
			set(k, l)
		case "prompts":
			raw, _ := json.Marshal(v)
			var ps []Prompt
			if err := json.Unmarshal(raw, &ps); err != nil || len(ps) > 6 {
				return nil, invalid("prompts must be up to 6 question and answer pairs")
			}
			for _, x := range ps {
				if strings.TrimSpace(x.Q) == "" || strings.TrimSpace(x.A) == "" || utf8.RuneCountInString(x.A) > 300 || utf8.RuneCountInString(x.Q) > 120 {
					return nil, invalid("each prompt needs a question and an answer of up to 300 characters")
				}
			}
			raw, _ = json.Marshal(ps)
			set("prompts", raw)
		case "dealbreakers":
			raw, _ := json.Marshal(v)
			var d map[string][]string
			if err := json.Unmarshal(raw, &d); err != nil {
				return nil, invalid("dealbreakers must map a field to the values you won't accept")
			}
			for f := range d {
				if !contains(DealbreakerFields, f) {
					return nil, invalid("dealbreakers can cover %s", strings.Join(DealbreakerFields, ", "))
				}
			}
			raw, _ = json.Marshal(d)
			set("dealbreakers", raw)
		default:
			return nil, invalid("unknown profile field %q", k)
		}
	}
	if ageMin > ageMax {
		return nil, invalid("the youngest age can't be above the oldest")
	}
	if len(sets) == 0 {
		return cur, nil
	}
	sets = append(sets, "updated_at=now()")
	if _, err := pool.Exec(ctx, `UPDATE profiles SET `+strings.Join(sets, ", ")+` WHERE user_id=$1`, args...); err != nil {
		return nil, err
	}
	return Get(ctx, pool, userID)
}

// DealbreakerFields are the attributes a person can rule out. Religion and
// politics are here because a person may state them; they are never learned.
var DealbreakerFields = []string{"smoking", "drinking", "kids", "wants_kids", "religion", "politics"}

func normList(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" || seen[s] || utf8.RuneCountInString(s) > 40 {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// SetCity stores a place that places.Resolve returned.
func SetCity(ctx context.Context, pool *pgxpool.Pool, userID, key, name string, lat, lon float64) error {
	_, err := pool.Exec(ctx, `UPDATE profiles SET city_key=$2, city_name=$3, city_lat=$4, city_lon=$5, updated_at=now() WHERE user_id=$1`, userID, key, name, lat, lon)
	return err
}

// Missing lists what is still needed before Suki can propose this person.
func Missing(ctx context.Context, pool *pgxpool.Pool, p *Profile) ([]string, error) {
	var photos, approved int
	if err := pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE status='approved') FROM photos WHERE user_id=$1`, p.UserID).Scan(&photos, &approved); err != nil {
		return nil, err
	}
	m := []string{}
	add := func(cond bool, name string) {
		if cond {
			m = append(m, name)
		}
	}
	add(p.DisplayName == "", "display_name")
	add(p.Birthdate == "", "birthdate")
	add(p.Gender == "", "gender")
	add(len(p.Seeking) == 0, "seeking")
	add(p.CityKey == "", "city")
	add(p.Goal == "", "goal")
	add(len(p.Interests) < 3, "interests (at least 3)")
	add(p.Bio == "" && len(p.Prompts) == 0, "bio or a prompt answer")
	add(photos == 0, "photo")
	if photos > 0 && approved == 0 {
		m = append(m, "photo (waiting for review)")
	}
	return m, nil
}

// ActivateIfComplete moves a finished onboarding to active. It reports whether
// this call did so, which is the moment the first match search starts.
func ActivateIfComplete(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	p, err := Get(ctx, pool, userID)
	if err != nil || p.Status != "onboarding" {
		return false, err
	}
	m, err := Missing(ctx, pool, p)
	if err != nil || len(m) > 0 {
		return false, err
	}
	tag, err := pool.Exec(ctx, `UPDATE profiles SET status='active', completed_at=now(), updated_at=now() WHERE user_id=$1 AND status='onboarding'`, userID)
	return tag.RowsAffected() == 1, err
}

// SetPaused pauses or resumes being proposed. A held account stays held.
func SetPaused(ctx context.Context, pool *pgxpool.Pool, userID string, paused bool) error {
	from, to := "active", "paused"
	if !paused {
		from, to = "paused", "active"
	}
	_, err := pool.Exec(ctx, `UPDATE profiles SET status=$3, updated_at=now() WHERE user_id=$1 AND status=$2`, userID, from, to)
	return err
}

// ── Photos (metadata; the bytes are in photostore) ──────────────────────────

type Photo struct {
	ID        string `json:"id"`
	UserID    string `json:"-"`
	S3Key     string `json:"-"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
	Tips      string `json:"tips"`
	IsPrimary bool   `json:"is_primary"`
}

func Photos(ctx context.Context, pool *pgxpool.Pool, userID string) ([]Photo, error) {
	rows, err := pool.Query(ctx, `SELECT id, user_id, s3_key, width, height, status, reason, tips, is_primary FROM photos
		WHERE user_id=$1 ORDER BY is_primary DESC, created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Photo{}
	for rows.Next() {
		var x Photo
		if err := rows.Scan(&x.ID, &x.UserID, &x.S3Key, &x.Width, &x.Height, &x.Status, &x.Reason, &x.Tips, &x.IsPrimary); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// ApprovedPhotoIDs are the only photos anyone else can see, main photo first.
func ApprovedPhotoIDs(ctx context.Context, pool *pgxpool.Pool, userID string) ([]string, error) {
	rows, err := pool.Query(ctx, `SELECT id FROM photos WHERE user_id=$1 AND status='approved' ORDER BY is_primary DESC, created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// RecordReview settles a photo. The first approved photo becomes the main one.
func RecordReview(ctx context.Context, pool *pgxpool.Pool, photoID, status, reason, tips string) (userID string, err error) {
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `UPDATE photos SET status=$2, reason=$3, tips=$4 WHERE id=$1 RETURNING user_id`, photoID, status, reason, tips).Scan(&userID); err != nil {
			return err
		}
		if status != "approved" {
			_, err := tx.Exec(ctx, `UPDATE photos SET is_primary=false WHERE id=$1`, photoID)
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE photos SET is_primary=true WHERE id=$1 AND NOT EXISTS (SELECT 1 FROM photos o WHERE o.user_id=photos.user_id AND o.is_primary)`, photoID)
		return err
	})
	return userID, err
}

// SetPrimary makes an approved photo the main one.
func SetPrimary(ctx context.Context, pool *pgxpool.Pool, userID, photoID string) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var status string
		if err := tx.QueryRow(ctx, `SELECT status FROM photos WHERE id=$1 AND user_id=$2`, photoID, userID).Scan(&status); err != nil {
			return invalid("that photo isn't yours")
		}
		if status != "approved" {
			return invalid("only an approved photo can be your main photo")
		}
		if _, err := tx.Exec(ctx, `UPDATE photos SET is_primary=false WHERE user_id=$1 AND is_primary`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE photos SET is_primary=true WHERE id=$1`, photoID)
		return err
	})
}

// RemovePhoto deletes the row and promotes the next approved photo. It returns
// the storage key for the caller to delete.
func RemovePhoto(ctx context.Context, pool *pgxpool.Pool, userID, photoID string) (string, error) {
	var key string
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `DELETE FROM photos WHERE id=$1 AND user_id=$2 RETURNING s3_key`, photoID, userID).Scan(&key); err != nil {
			return invalid("that photo isn't yours")
		}
		_, err := tx.Exec(ctx, `UPDATE photos SET is_primary=true WHERE id = (SELECT id FROM photos WHERE user_id=$1 AND status='approved' ORDER BY created_at LIMIT 1)
			AND NOT EXISTS (SELECT 1 FROM photos o WHERE o.user_id=$1 AND o.is_primary)`, userID)
		return err
	})
	return key, err
}
