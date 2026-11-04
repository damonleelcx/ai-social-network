package dating

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/damonleelcx/ai-social-network/internal/chatstore"
	"github.com/damonleelcx/ai-social-network/internal/i18n"
)

type setting struct {
	Default any
	Enum    []string
	Min     int
	Max     int
}

// Settings is the whole list of what can be set, by the settings screens and by
// asking Suki. Anything else is refused.
var Settings = map[string]setting{
	// App
	"language":        {Default: "en", Enum: []string{"en", "zh", "ko", "ja", "es"}},
	"theme":           {Default: "system", Enum: []string{"system", "light", "dark"}},
	"notify_matches":  {Default: true},
	"notify_messages": {Default: true},
	"notify_dates":    {Default: true},
	"notify_suki":     {Default: true},
	"notify_email":    {Default: true},
	"read_receipts":   {Default: true},
	"daily_matches":   {Default: 3, Min: 1, Max: 5},
	// Agent
	"suki_tone":       {Default: "playful", Enum: []string{"playful", "warm", "direct", "calm"}},
	"suki_emoji":      {Default: "some", Enum: []string{"none", "some", "lots"}},
	"suki_proactive":  {Default: "normal", Enum: []string{"low", "normal", "high"}},
	"suki_nickname":   {Default: ""},
	"mood_avatar":     {Default: true},
	"suki_voice":      {Default: true},
	"suki_voice_auto": {Default: false},
	"chat_coach":      {Default: true},
	"chat_coach_read": {Default: true},
	"chat_prompts":    {Default: true},
	"memory_enabled":  {Default: true},
	"debrief_enabled": {Default: true},
}

// GetSettings returns every setting, with defaults filled in.
// Lang is the language someone reads the app in.
func (s *Service) Lang(ctx context.Context, userID string) string {
	l, _ := GetSettings(ctx, s.Pool, userID)["language"].(string)
	return l
}

// localizeNotes renders date notes for whoever is reading. Older notes have no template and
// keep the English they were written in.
func localizeNotes(ms []chatstore.Message, lang string) {
	for i := range ms {
		note, ok := ms[i].Meta["note"].(string)
		if !ok {
			continue
		}
		vars := map[string]string{}
		switch v := ms[i].Meta["note_vars"].(type) {
		case map[string]any:
			for k, x := range v {
				vars[k], _ = x.(string)
			}
		case map[string]string:
			vars = v
		case bson.M:
			for k, x := range v {
				vars[k], _ = x.(string)
			}
		case bson.D:
			for _, e := range v {
				vars[e.Key], _ = e.Value.(string)
			}
		}
		ms[i].Body = renderNote(lang, note, vars)
	}
}

func renderNote(lang, note string, vars map[string]string) string {
	kv := make([]string, 0, len(vars)*2)
	for k, v := range vars {
		kv = append(kv, k, i18n.T(lang, v))
	}
	return i18n.T(lang, note, kv...)
}

func GetSettings(ctx context.Context, pool *pgxpool.Pool, userID string) map[string]any {
	stored := Prefs(ctx, pool, userID)
	out := map[string]any{}
	for k, s := range Settings {
		if v, ok := stored[k]; ok {
			out[k] = v
		} else {
			out[k] = s.Default
		}
	}
	return out
}

// PatchSettings validates and merges. Nothing is written if any key is bad.
func PatchSettings(ctx context.Context, pool *pgxpool.Pool, userID string, patch map[string]any) (map[string]any, error) {
	for k, v := range patch {
		s, ok := Settings[k]
		if !ok {
			return nil, userErr("there's no setting called %q", k)
		}
		if err := s.check(k, v); err != nil {
			return nil, err
		}
	}
	raw, _ := json.Marshal(patch)
	if _, err := pool.Exec(ctx, `INSERT INTO user_preferences (user_id, data) VALUES ($1,$2)
		ON CONFLICT (user_id) DO UPDATE SET data = user_preferences.data || EXCLUDED.data, updated_at=now()`, userID, raw); err != nil {
		return nil, err
	}
	return GetSettings(ctx, pool, userID), nil
}

func (s setting) check(k string, v any) error {
	switch d := s.Default.(type) {
	case bool:
		if _, ok := v.(bool); !ok {
			return userErr("%s must be on or off", k)
		}
	case int:
		f, ok := v.(float64)
		if !ok || f != float64(int(f)) || int(f) < s.Min || int(f) > s.Max {
			return userErr("%s must be a whole number from %d to %d", k, s.Min, s.Max)
		}
	case string:
		str, ok := v.(string)
		if !ok {
			return userErr("%s must be text", k)
		}
		if len(s.Enum) > 0 {
			for _, e := range s.Enum {
				if e == str {
					return nil
				}
			}
			return userErr("%s must be one of %s", k, strings.Join(s.Enum, ", "))
		}
		if utf8.RuneCountInString(str) > 30 {
			return userErr("%s is too long", k)
		}
		_ = d
	default:
		return fmt.Errorf("setting %s has an unknown type", k)
	}
	return nil
}
