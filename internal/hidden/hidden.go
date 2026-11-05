// Package hidden is the only code allowed to touch evaluations, scores, ranks and
// learned taste. Nothing here may reach a person. internal/archtest enforces the
// boundary: the API and the conversational agent cannot import this package, and
// no other package may name the hidden schema in SQL.
//
// Fairness rules that live in this file:
//   - No feature for race, religion, politics or looks. Photos are never scored.
//   - Reputation moves a ranking by at most ±ReputationWeight, and only after
//     MinRaters different people, so one bad date cannot bury anyone.
//   - Raters are normalised against their own average: a harsh rater's 6 and a
//     generous rater's 9 can mean the same thing.
package hidden

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damonleelcx/ai-social-network/internal/profile"
)

const (
	ReputationWeight = 0.15
	MinRaters        = 3
	// PublishThreshold is the lowest final score worth a person's attention.
	PublishThreshold = 55.0
	learningRate     = 0.08
	shrinkage        = 5.0 // pseudo-raters at the neutral value
)

// ── Features ────────────────────────────────────────────────────────────────

// interestCategories groups free-text interests so taste generalises: someone who
// keeps liking hikers will also like climbers.
var interestCategories = map[string][]string{
	"outdoors": {"hik", "climb", "camp", "surf", "ski", "snowboard", "kayak", "trail", "nature", "garden", "bouldering"},
	"fitness":  {"run", "gym", "yoga", "pilates", "cycl", "swim", "tennis", "soccer", "basketball", "football", "volleyball", "boxing", "dance", "dancing", "sport"},
	"arts":     {"art", "museum", "paint", "draw", "photograph", "theat", "theater", "design", "ceramic", "pottery", "gallery"},
	"music":    {"music", "jazz", "concert", "guitar", "piano", "sing", "vinyl", "band", "dj", "opera"},
	"food":     {"food", "cook", "bak", "ramen", "sushi", "coffee", "wine", "beer", "restaurant", "brunch", "tea", "cocktail"},
	"tech":     {"tech", "code", "coding", "program", "robot", "ai", "startup", "science", "engineer"},
	"games":    {"game", "gaming", "board game", "chess", "poker", "dnd", "anime", "manga", "esports"},
	"reading":  {"book", "read", "poetry", "writ", "literature", "history", "philosophy"},
	"film":     {"film", "movie", "cinema", "tv", "documentar", "series"},
	"travel":   {"travel", "backpack", "language", "road trip", "explor"},
	"wellness": {"meditat", "mindful", "spa", "therapy", "journal"},
	"social":   {"volunteer", "comedy", "karaoke", "trivia", "party", "dogs", "cats", "pets"},
}

// Features describes one person as seen by another. The list is closed on
// purpose: anything not named here cannot become part of someone's "type".
func Features(viewer, other *profile.Profile) map[string]float64 {
	f := map[string]float64{
		"age_delta":        float64(other.Age-viewer.Age) / 10,
		"age_gap":          math.Abs(float64(other.Age-viewer.Age)) / 10,
		"interest_overlap": jaccard(viewer.Interests, other.Interests),
		"language_overlap": jaccard(viewer.Languages, other.Languages),
		"goal_match":       same(viewer.Goal, other.Goal),
		"smoking_match":    same(viewer.Smoking, other.Smoking),
		"drinking_match":   same(viewer.Drinking, other.Drinking),
		"kids_match":       same(viewer.WantsKids, other.WantsKids),
	}
	if viewer.HeightCM != nil && other.HeightCM != nil {
		f["height_delta"] = float64(*other.HeightCM-*viewer.HeightCM) / 20
	}
	for cat, stems := range interestCategories {
		for _, in := range other.Interests {
			if matchesStem(in, stems) {
				f["likes_"+cat] = 1
				break
			}
		}
	}
	return f
}

func matchesStem(interest string, stems []string) bool {
	for _, st := range stems {
		if strings.Contains(interest, st) {
			return true
		}
	}
	return false
}

func same(a, b string) float64 {
	if a != "" && a == b {
		return 1
	}
	return 0
}

func jaccard(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	set := map[string]bool{}
	for _, x := range a {
		set[x] = true
	}
	inter := 0
	for _, x := range b {
		if set[x] {
			inter++
		}
	}
	return float64(inter) / float64(len(a)+len(b)-inter)
}

// basePrior is what the model assumes before it knows anything about a person.
var basePrior = map[string]float64{"interest_overlap": 3, "goal_match": 1.2, "kids_match": 0.8, "age_gap": -0.6, "language_overlap": 0.4, "smoking_match": 0.3, "drinking_match": 0.2}

func predict(weights, feats map[string]float64) float64 {
	z := -0.8
	for k, v := range feats {
		z += (basePrior[k] + weights[k]) * v
	}
	return 1 / (1 + math.Exp(-z))
}

// ── Taste ───────────────────────────────────────────────────────────────────

// Signal kinds and how much each tells us. A date someone would repeat says more
// than a tap on a card, and silence says nothing.
var signalKinds = map[string]struct{ label, strength float64 }{
	"accept":     {1, 0.6},
	"pass":       {0, 0.6},
	"mutual":     {1, 0.4},
	"replied":    {1, 0.3},
	"date_set":   {1, 1},
	"date_good":  {1, 2.5},
	"date_bad":   {0, 2.5},
	"unmatch":    {0, 1},
}

// Signal teaches the model from something a person did. It is one online
// logistic-regression step, and the key makes a replayed event a no-op.
func Signal(ctx context.Context, pool *pgxpool.Pool, userID, subjectID, kind, key string) error {
	sk, ok := signalKinds[kind]
	if !ok {
		return fmt.Errorf("unknown signal %q", kind)
	}
	viewer, err := profile.Get(ctx, pool, userID)
	if err != nil {
		return err
	}
	other, err := profile.Get(ctx, pool, subjectID)
	if err != nil {
		return err
	}
	feats := Features(viewer, other)
	raw, _ := json.Marshal(feats)
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `INSERT INTO hidden.signals (user_id, subject_id, kind, label, strength, features, key)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (key) DO NOTHING`, userID, subjectID, kind, sk.label, sk.strength, raw, key)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO hidden.taste (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
			return err
		}
		var w []byte
		if err := tx.QueryRow(ctx, `SELECT weights FROM hidden.taste WHERE user_id=$1 FOR UPDATE`, userID).Scan(&w); err != nil {
			return err
		}
		weights := map[string]float64{}
		_ = json.Unmarshal(w, &weights)
		update(weights, feats, sk.label, sk.strength)
		w, _ = json.Marshal(weights)
		_, err = tx.Exec(ctx, `UPDATE hidden.taste SET weights=$2, signals=signals+1, updated_at=now() WHERE user_id=$1`, userID, w)
		return err
	})
}

func update(weights, feats map[string]float64, label, strength float64) {
	errTerm := label - predict(weights, feats)
	for k, v := range feats {
		weights[k] = clamp(weights[k]+learningRate*strength*errTerm*v, -4, 4)
	}
}

func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, x)) }

func tasteOf(ctx context.Context, pool *pgxpool.Pool, userID string) (map[string]float64, error) {
	var raw []byte
	err := pool.QueryRow(ctx, `SELECT weights FROM hidden.taste WHERE user_id=$1`, userID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return map[string]float64{}, nil
	}
	if err != nil {
		return nil, err
	}
	w := map[string]float64{}
	_ = json.Unmarshal(raw, &w)
	return w, nil
}

// ResetTaste forgets what was learned from behaviour. The person asked.
func ResetTaste(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM hidden.signals WHERE user_id=$1`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM hidden.taste WHERE user_id=$1`, userID)
		return err
	})
}

// TasteDigest describes learned leanings in words for the TasteLearner, with no
// numbers, so a number cannot end up in a note the person reads.
func TasteDigest(ctx context.Context, pool *pgxpool.Pool, userID string) (string, int, error) {
	w, err := tasteOf(ctx, pool, userID)
	if err != nil {
		return "", 0, err
	}
	var signals, noted int
	err = pool.QueryRow(ctx, `SELECT signals, noted_at_signals FROM hidden.taste WHERE user_id=$1`, userID).Scan(&signals, &noted)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", 0, err
	}
	type kv struct {
		k string
		v float64
	}
	var l []kv
	for k, v := range w {
		if math.Abs(v) >= 0.15 {
			l = append(l, kv{k, v})
		}
	}
	sort.Slice(l, func(i, j int) bool {
		if math.Abs(l[i].v) != math.Abs(l[j].v) {
			return math.Abs(l[i].v) > math.Abs(l[j].v)
		}
		return l[i].k < l[j].k
	})
	var b strings.Builder
	for i, x := range l {
		if i == 8 {
			break
		}
		dir := "leans toward"
		if x.v < 0 {
			dir = "leans away from"
		}
		fmt.Fprintf(&b, "- %s %s\n", dir, strings.ReplaceAll(strings.TrimPrefix(x.k, "likes_"), "_", " "))
	}
	return b.String(), signals - noted, nil
}

// MarkNoted records that notes now reflect the signals so far.
func MarkNoted(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	_, err := pool.Exec(ctx, `UPDATE hidden.taste SET noted_at_signals=signals WHERE user_id=$1`, userID)
	return err
}

// UsersDueForNotes lists people with enough new behaviour for fresh notes.
func UsersDueForNotes(ctx context.Context, pool *pgxpool.Pool, min, limit int) ([]string, error) {
	return strings1(ctx, pool, `SELECT user_id FROM hidden.taste WHERE signals - noted_at_signals >= $1 LIMIT $2`, min, limit)
}

func strings1(ctx context.Context, pool *pgxpool.Pool, q string, a ...any) ([]string, error) {
	rows, err := pool.Query(ctx, q, a...)
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

// ── Scout: filter and rank ──────────────────────────────────────────────────

// PackPerson is one candidate as the Matchmaker sees them: an alias and public
// facts, no name, no photo, no contact details.
type PackPerson struct {
	Alias       string           `json:"alias"`
	UserID      string           `json:"user_id"`
	Prior       float64          `json:"prior"`
	Age         int              `json:"age"`
	Gender      string           `json:"gender"`
	Goal        string           `json:"goal"`
	Bio         string           `json:"bio"`
	Occupation  string           `json:"occupation"`
	Smoking     string           `json:"smoking"`
	Drinking    string           `json:"drinking"`
	Kids        string           `json:"kids"`
	WantsKids   string           `json:"wants_kids"`
	Languages   []string         `json:"languages"`
	Interests   []string         `json:"interests"`
	Prompts     []profile.Prompt `json:"prompts"`
}

type Pack struct {
	Viewer     PackPerson   `json:"viewer"`
	Candidates []PackPerson `json:"candidates"`
}

func packPerson(p *profile.Profile, alias string, prior float64) PackPerson {
	return PackPerson{Alias: alias, UserID: p.UserID, Prior: prior, Age: p.Age, Gender: p.Gender, Goal: p.Goal, Bio: p.Bio, Occupation: p.Occupation,
		Smoking: p.Smoking, Drinking: p.Drinking, Kids: p.Kids, WantsKids: p.WantsKids, Languages: p.Languages, Interests: p.Interests, Prompts: p.Prompts}
}

func dealbroken(from, to *profile.Profile) bool {
	vals := map[string]string{"smoking": to.Smoking, "drinking": to.Drinking, "kids": to.Kids, "wants_kids": to.WantsKids, "religion": to.Religion, "politics": to.Politics}
	for field, banned := range from.Dealbreaker {
		for _, b := range banned {
			if strings.EqualFold(b, vals[field]) && vals[field] != "" {
				return true
			}
		}
	}
	return false
}

// eligibleSQL is the hard filter both the Scout and the Publisher rely on. Hard
// means hard: the same city, both people's stated gender and age preferences, no
// block either way, and never a pair that was already proposed.
const eligibleSQL = `FROM profiles p JOIN users u ON u.id = p.user_id
	WHERE p.status='active' AND u.deleted_at IS NULL AND u.email_verified_at IS NOT NULL
	  AND p.user_id <> $1 AND p.city_key = $2 AND p.city_key <> ''
	  AND p.gender = ANY($3::text[]) AND $4::text = ANY(p.seeking)
	  AND date_part('year', age(p.birthdate)) BETWEEN $5::int AND $6::int
	  AND p.age_min <= $7::int AND p.age_max >= $7::int
	  AND p.last_active_at > now() - interval '30 days'
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id AND ph.status='approved')
	  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id=$1 AND b.blocked_id=p.user_id) OR (b.blocker_id=p.user_id AND b.blocked_id=$1))
	  AND NOT EXISTS (SELECT 1 FROM proposals x WHERE x.user_a=least($1::uuid,p.user_id) AND x.user_b=greatest($1::uuid,p.user_id))
	  AND (SELECT count(*) FROM proposals x WHERE (x.user_a=p.user_id OR x.user_b=p.user_id) AND x.status='open') < 6`

// Gather runs the Scout: hard filters, then a ranking that blends the viewer's
// learned taste, the candidate's taste for the viewer, and bounded reputation.
// The top candidates go into a pack stored against this search.
func Gather(ctx context.Context, pool *pgxpool.Pool, goalID, viewerID string, topK int) (int, error) {
	v, err := profile.Get(ctx, pool, viewerID)
	if err != nil {
		return 0, err
	}
	rows, err := pool.Query(ctx, `SELECT `+profile.Cols+` `+eligibleSQL+` LIMIT 300`, viewerID, v.CityKey, v.Seeking, v.Gender, v.AgeMin, v.AgeMax, v.Age)
	if err != nil {
		return 0, err
	}
	var cands []*profile.Profile
	for rows.Next() {
		c, err := profile.Scan(rows)
		if err != nil {
			rows.Close()
			return 0, err
		}
		if !dealbroken(v, c) && !dealbroken(c, v) {
			cands = append(cands, c)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	vw, err := tasteOf(ctx, pool, viewerID)
	if err != nil {
		return 0, err
	}
	type scored struct {
		p     *profile.Profile
		score float64
	}
	list := make([]scored, 0, len(cands))
	for _, c := range cands {
		cw, err := tasteOf(ctx, pool, c.UserID)
		if err != nil {
			return 0, err
		}
		rep, err := reputationOf(ctx, pool, c.UserID)
		if err != nil {
			return 0, err
		}
		list = append(list, scored{c, RankScore(predict(vw, Features(v, c)), predict(cw, Features(c, v)), rep)})
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].score > list[j].score })
	if len(list) > topK {
		list = list[:topK]
	}
	pack := Pack{Viewer: packPerson(v, "you", 0)}
	for i, x := range list {
		pack.Candidates = append(pack.Candidates, packPerson(x.p, fmt.Sprintf("c%d", i+1), x.score))
	}
	raw, _ := json.Marshal(pack)
	_, err = pool.Exec(ctx, `INSERT INTO hidden.search_packs (goal_id, viewer_id, pack) VALUES ($1,$2,$3)
		ON CONFLICT (goal_id) DO UPDATE SET pack=EXCLUDED.pack`, goalID, viewerID, raw)
	return len(list), err
}

// RankScore blends how much each side should like the other (both matter: a match
// needs two yeses) and scales by bounded reputation.
func RankScore(viewerLikes, candidateLikes, reputation float64) float64 {
	return (0.6*viewerLikes + 0.4*candidateLikes) * (1 + ReputationWeight*clamp(reputation, -1, 1))
}

func GetPack(ctx context.Context, pool *pgxpool.Pool, goalID string) (*Pack, error) {
	var raw []byte
	if err := pool.QueryRow(ctx, `SELECT pack FROM hidden.search_packs WHERE goal_id=$1`, goalID).Scan(&raw); err != nil {
		return nil, err
	}
	var p Pack
	return &p, json.Unmarshal(raw, &p)
}

// ── Matchmaker and Critic ───────────────────────────────────────────────────

type Compat struct {
	CandidateID    string   `json:"candidate_id"`
	Alias          string   `json:"alias"`
	Score          float64  `json:"score"`
	Reasons        []string `json:"reasons"`
	Shared         []string `json:"shared"`
	PitchViewer    string   `json:"pitch_viewer"`
	PitchCandidate string   `json:"pitch_candidate"`
	Verdict        string   `json:"verdict"`
	Prior          float64  `json:"-"`
}

// RecordCompat stores one evaluation. The shared interests are checked against
// both profiles here, so the Matchmaker cannot invent common ground.
func RecordCompat(ctx context.Context, pool *pgxpool.Pool, goalID, alias string, score float64, reasons, shared []string, pitchViewer, pitchCand string) error {
	pack, err := GetPack(ctx, pool, goalID)
	if err != nil {
		return err
	}
	var c *PackPerson
	for i := range pack.Candidates {
		if pack.Candidates[i].Alias == alias {
			c = &pack.Candidates[i]
		}
	}
	if c == nil {
		return fmt.Errorf("no candidate %q in this search", alias)
	}
	if score < 0 || score > 100 {
		return errors.New("score must be from 0 to 100")
	}
	for _, s := range shared {
		if !hasFold(pack.Viewer.Interests, s) || !hasFold(c.Interests, s) {
			return fmt.Errorf("%q is not an interest both people listed; use only exact shared interests", s)
		}
	}
	for _, pitch := range []string{pitchViewer, pitchCand} {
		if LeaksNumbers(pitch) {
			return errors.New("a pitch may not contain a score, percentage, rank or rating; describe shared ground in words")
		}
	}
	r, _ := json.Marshal(nonNil(reasons))
	sh, _ := json.Marshal(nonNil(shared))
	_, err = pool.Exec(ctx, `INSERT INTO hidden.compat (goal_id, viewer_id, candidate_id, score, prior, reasons, shared, pitch_viewer, pitch_candidate)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (goal_id, candidate_id) DO UPDATE SET score=EXCLUDED.score, reasons=EXCLUDED.reasons, shared=EXCLUDED.shared,
		pitch_viewer=EXCLUDED.pitch_viewer, pitch_candidate=EXCLUDED.pitch_candidate, verdict='pending', critic_notes=''`,
		goalID, pack.Viewer.UserID, c.UserID, score, c.Prior, r, sh, pitchViewer, pitchCand)
	return err
}

func nonNil(l []string) []string {
	if l == nil {
		return []string{}
	}
	return l
}

func hasFold(l []string, s string) bool {
	for _, x := range l {
		if strings.EqualFold(x, s) {
			return true
		}
	}
	return false
}

// LeaksNumbers spots copy that would turn a hidden score into a visible one.
func LeaksNumbers(s string) bool {
	l := strings.ToLower(s)
	for _, w := range []string{"%", "score", "rating", "rated", "ranked", "rank ", "reputation", "percent", "/10", "/100", "out of 10", "out of 100"} {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

func Compats(ctx context.Context, pool *pgxpool.Pool, goalID string) ([]Compat, error) {
	pack, err := GetPack(ctx, pool, goalID)
	if err != nil {
		return nil, err
	}
	alias := map[string]string{}
	for _, c := range pack.Candidates {
		alias[c.UserID] = c.Alias
	}
	rows, err := pool.Query(ctx, `SELECT candidate_id, score, prior, reasons, shared, pitch_viewer, pitch_candidate, verdict FROM hidden.compat WHERE goal_id=$1 ORDER BY score DESC`, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Compat{}
	for rows.Next() {
		var c Compat
		var r, sh []byte
		if err := rows.Scan(&c.CandidateID, &c.Score, &c.Prior, &r, &sh, &c.PitchViewer, &c.PitchCandidate, &c.Verdict); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(r, &c.Reasons)
		_ = json.Unmarshal(sh, &c.Shared)
		c.Alias = alias[c.CandidateID]
		out = append(out, c)
	}
	return out, rows.Err()
}

// Verdict records the Critic's ruling on one evaluation.
func Verdict(ctx context.Context, pool *pgxpool.Pool, goalID, alias, verdict, notes string) error {
	pack, err := GetPack(ctx, pool, goalID)
	if err != nil {
		return err
	}
	for _, c := range pack.Candidates {
		if c.Alias == alias {
			tag, err := pool.Exec(ctx, `UPDATE hidden.compat SET verdict=$3, critic_notes=$4 WHERE goal_id=$1 AND candidate_id=$2`, goalID, c.UserID, verdict, notes)
			if err == nil && tag.RowsAffected() == 0 {
				return fmt.Errorf("%s has no evaluation yet", alias)
			}
			return err
		}
	}
	return fmt.Errorf("no candidate %q in this search", alias)
}

// Pending counts candidates without an evaluation, and evaluations without a ruling.
func Pending(ctx context.Context, pool *pgxpool.Pool, goalID string) (unevaluated, unruled []string, err error) {
	pack, err := GetPack(ctx, pool, goalID)
	if err != nil {
		return nil, nil, err
	}
	cs, err := Compats(ctx, pool, goalID)
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]string{}
	for _, c := range cs {
		seen[c.Alias] = c.Verdict
	}
	for _, c := range pack.Candidates {
		switch seen[c.Alias] {
		case "":
			unevaluated = append(unevaluated, c.Alias)
		case "pending":
			unruled = append(unruled, c.Alias)
		}
	}
	return unevaluated, unruled, nil
}

// FinalScore blends the model's judgement with the Scout's ranking.
func FinalScore(modelScore, prior float64) float64 { return 0.6*modelScore + 0.4*100*prior }

// Publishable is a pair ready to put to both people. It carries the copy and no
// numbers, so what leaves this package cannot leak a score.
type Publishable struct {
	CandidateID    string
	Shared         []string
	PitchViewer    string
	PitchCandidate string
	// Close marks a weaker fit, introduced because nobody better turned up. Its
	// pitches may not hold up, so the Publisher writes plain ones instead.
	Close bool
}

// closeFits is how many weaker fits one search may introduce.
const closeFits = 1

// Selected picks what this search may publish: approved, over the threshold, and
// still eligible now (someone may have blocked or paused since the search began),
// best first, up to limit. If none qualify, it falls back to the closest fit that
// the Critic didn't reject. Eligibility stays hard either way.
func Selected(ctx context.Context, pool *pgxpool.Pool, goalID, viewerID string, limit int) ([]Publishable, error) {
	cs, err := Compats(ctx, pool, goalID)
	if err != nil {
		return nil, err
	}
	v, err := profile.Get(ctx, pool, viewerID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(cs, func(i, j int) bool { return FinalScore(cs[i].Score, cs[i].Prior) > FinalScore(cs[j].Score, cs[j].Prior) })
	out := []Publishable{}
	close := []Publishable{}
	for _, c := range cs {
		if len(out) >= limit {
			break
		}
		strong := c.Verdict == "approved" && FinalScore(c.Score, c.Prior) >= PublishThreshold
		if !strong && c.Verdict != "approved" && c.Verdict != "close_fit" {
			continue
		}
		var ok bool
		err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 `+eligibleSQL+` AND p.user_id=$8::uuid)`, viewerID, v.CityKey, v.Seeking, v.Gender, v.AgeMin, v.AgeMax, v.Age, c.CandidateID).Scan(&ok)
		if err != nil {
			return nil, err
		}
		if ok && strong {
			out = append(out, Publishable{c.CandidateID, c.Shared, c.PitchViewer, c.PitchCandidate, false})
		} else if ok {
			close = append(close, Publishable{c.CandidateID, c.Shared, "", "", true})
		}
	}
	if len(out) > 0 {
		return out, nil
	}
	if len(close) > closeFits {
		close = close[:closeFits]
	}
	if len(close) > limit {
		close = close[:limit]
	}
	return close, nil
}

// ── After a date ────────────────────────────────────────────────────────────

type Evaluation struct {
	DateID, RaterID, SubjectID                                    string
	Chemistry, Respect, Conversation, Reliability, ProfileHonesty float64
	SeeAgain                                                      bool
	Evidence                                                      []string
}

// Overall weighs respect and reliability heavily on purpose: chemistry is
// personal, while how someone treated their date is not.
func (e *Evaluation) Overall() float64 {
	return 0.25*e.Chemistry + 0.3*e.Respect + 0.15*e.Conversation + 0.2*e.Reliability + 0.1*e.ProfileHonesty
}

// RecordEvaluation stores an evaluation once per person per date, recomputes the
// subject's reputation, and teaches the rater's taste.
func RecordEvaluation(ctx context.Context, pool *pgxpool.Pool, e *Evaluation) error {
	for _, v := range []float64{e.Chemistry, e.Respect, e.Conversation, e.Reliability, e.ProfileHonesty} {
		if v < 0 || v > 10 {
			return errors.New("every dimension must be from 0 to 10")
		}
	}
	ev, _ := json.Marshal(nonNil(e.Evidence))
	_, err := pool.Exec(ctx, `INSERT INTO hidden.evaluations (date_id, rater_id, subject_id, chemistry, respect, conversation, reliability, profile_honesty, overall, see_again, evidence)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (date_id, rater_id) DO UPDATE SET chemistry=EXCLUDED.chemistry, respect=EXCLUDED.respect, conversation=EXCLUDED.conversation,
		reliability=EXCLUDED.reliability, profile_honesty=EXCLUDED.profile_honesty, overall=EXCLUDED.overall, see_again=EXCLUDED.see_again, evidence=EXCLUDED.evidence`,
		e.DateID, e.RaterID, e.SubjectID, e.Chemistry, e.Respect, e.Conversation, e.Reliability, e.ProfileHonesty, e.Overall(), e.SeeAgain, ev)
	if err != nil {
		return err
	}
	if err := RecomputeReputation(ctx, pool, e.SubjectID); err != nil {
		return err
	}
	kind := "date_bad"
	if e.SeeAgain {
		kind = "date_good"
	}
	return Signal(ctx, pool, e.RaterID, e.SubjectID, kind, "date:"+e.DateID+":"+e.RaterID)
}

// RecomputeReputation averages rater-normalised scores, shrunk toward neutral,
// counting one rating per rater and nothing until MinRaters different people.
func RecomputeReputation(ctx context.Context, pool *pgxpool.Pool, subjectID string) error {
	rows, err := pool.Query(ctx, `SELECT e.rater_id, avg(e.overall),
		  (SELECT avg(o.overall) FROM hidden.evaluations o WHERE o.rater_id=e.rater_id),
		  (SELECT count(*) FROM hidden.evaluations o WHERE o.rater_id=e.rater_id)
		FROM hidden.evaluations e WHERE e.subject_id=$1 GROUP BY e.rater_id`, subjectID)
	if err != nil {
		return err
	}
	var obs []RaterObs
	for rows.Next() {
		var r RaterObs
		var id string
		if err := rows.Scan(&id, &r.Score, &r.RaterMean, &r.RaterCount); err != nil {
			rows.Close()
			return err
		}
		obs = append(obs, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `INSERT INTO hidden.reputation (user_id, score, raters, updated_at) VALUES ($1,$2,$3,now())
		ON CONFLICT (user_id) DO UPDATE SET score=EXCLUDED.score, raters=EXCLUDED.raters, updated_at=now()`, subjectID, ReputationFrom(obs), len(obs))
	return err
}

// RaterObs is one rater's view of a subject, plus that rater's own habits.
type RaterObs struct {
	Score, RaterMean float64
	RaterCount       int
}

// ReputationFrom turns observations into a value in [-1, 1]. A rater with few
// ratings has no meaningful average, so they are compared to the scale's middle.
func ReputationFrom(obs []RaterObs) float64 {
	if len(obs) < MinRaters {
		return 0
	}
	sum := 0.0
	for _, o := range obs {
		base := 6.0
		if o.RaterCount >= 3 {
			base = o.RaterMean
		}
		sum += o.Score - base
	}
	mean := sum / (float64(len(obs)) + shrinkage)
	return clamp(mean/2.5, -1, 1)
}

func reputationOf(ctx context.Context, pool *pgxpool.Pool, userID string) (float64, error) {
	var s float64
	err := pool.QueryRow(ctx, `SELECT score FROM hidden.reputation WHERE user_id=$1`, userID).Scan(&s)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return s, err
}

// HasEvaluation reports whether this rater's debrief was already turned into an
// evaluation. The verifier uses it: it never exposes the evaluation itself.
func HasEvaluation(ctx context.Context, pool *pgxpool.Pool, dateID, raterID string) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM hidden.evaluations WHERE date_id=$1 AND rater_id=$2)`, dateID, raterID).Scan(&ok)
	return ok, err
}
