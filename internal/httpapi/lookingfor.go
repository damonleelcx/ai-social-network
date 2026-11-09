package httpapi

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damonleelcx/ai-social-network/internal/profile"
	"github.com/damonleelcx/ai-social-network/internal/workflows"
)

// LookingFor is what Suki understands someone wants, in three parts, built only from what they
// told her: who and what they're looking for, what they won't accept, and the notes she keeps on
// what they like. Nothing from hidden evaluations: they can see, and correct, every line.
type LookingFor struct {
	Must     LookingMust         `json:"must"`
	RedFlags map[string][]string `json:"red_flags"`
	Nice     []LookingForNote    `json:"nice"`
	// Read is Suki's own reading of what they've said. Every line cites their words.
	Read []workflows.LookingLine `json:"read"`
}

type LookingMust struct {
	Seeking []string `json:"seeking"`
	AgeMin  int      `json:"age_min,omitempty"`
	AgeMax  int      `json:"age_max,omitempty"`
	Goal    string   `json:"goal,omitempty"`
	City    string   `json:"city,omitempty"`
}

type LookingForNote struct {
	ID   string `json:"id"`
	Note string `json:"note"`
}

// ReadLookingFor builds the card's contents for one person.
func ReadLookingFor(ctx context.Context, pool *pgxpool.Pool, userID string) (*LookingFor, error) {
	p, err := profile.Get(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	out := &LookingFor{
		Must:     LookingMust{Seeking: p.Seeking, AgeMin: p.AgeMin, AgeMax: p.AgeMax, Goal: p.Goal, City: p.CityName},
		RedFlags: map[string][]string{},
		Nice:     []LookingForNote{},
	}
	if out.Must.Seeking == nil {
		out.Must.Seeking = []string{}
	}
	for k, v := range p.Dealbreaker {
		if len(v) > 0 {
			out.RedFlags[k] = v
		}
	}
	rows, err := pool.Query(ctx, `SELECT id, content FROM memories WHERE user_id=$1 AND kind='preference' ORDER BY created_at DESC LIMIT 30`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var n LookingForNote
		if err := rows.Scan(&n.ID, &n.Note); err != nil {
			return nil, err
		}
		out.Nice = append(out.Nice, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out.Read, err = workflows.LookingLines(ctx, userID)
	return out, err
}

// dropLookingLine takes one of Suki's lines off the card, and she won't say it again.
func (s *Server) dropLookingLine(w http.ResponseWriter, r *http.Request) {
	if err := workflows.DropLookingLine(r.Context(), userOf(r).ID, r.PathValue("id")); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) lookingFor(w http.ResponseWriter, r *http.Request) {
	lf, err := ReadLookingFor(r.Context(), s.Pool, userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, lf)
}

// dropRedFlag takes one of their dealbreakers away, from the card.
func (s *Server) dropRedFlag(w http.ResponseWriter, r *http.Request) {
	u := userOf(r)
	p, err := profile.Get(r.Context(), s.Pool, u.ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	deal := p.Dealbreaker
	if deal == nil {
		deal = map[string][]string{}
	}
	delete(deal, r.PathValue("field"))
	if _, err := profile.Save(r.Context(), s.Pool, u.ID, map[string]any{"dealbreakers": deal}); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
