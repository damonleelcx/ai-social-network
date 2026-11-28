package dating

import (
	"encoding/json"
	"strings"

	"github.com/damonleelcx/ai-social-network/internal/llm"
)

// ParseBuddyDraft reads the model's JSON for one plan. It does not post anything.
// When and note can each be empty, but not both: the person still edits the form.
func ParseBuddyDraft(raw string) (BuddyInput, error) {
	var v struct {
		Kind  string `json:"kind"`
		When  string `json:"when"`
		Note  string `json:"note"`
		Hours int    `json:"expires_in_hours"`
		Same  bool   `json:"same_gender"`
		Spots int    `json:"spots"`
		Place string `json:"place"`
	}
	if json.Unmarshal([]byte(llm.ExtractJSON(raw)), &v) != nil {
		return BuddyInput{}, userErr("I couldn't turn that into a plan. Say a little more about what and when.")
	}
	v.When = clipRunes(v.When, buddyWhenMax)
	v.Note = clipRunes(v.Note, buddyNoteMax)
	v.Place = clipRunes(v.Place, buddyPlaceMax)
	if v.When == "" && v.Note == "" {
		return BuddyInput{}, userErr("I couldn't turn that into a plan. Say a little more about what and when.")
	}
	if _, ok := buddyKindNames[v.Kind]; !ok {
		v.Kind = "other"
	}
	if v.Hours < 1 {
		v.Hours = 24
	} else if v.Hours > buddyMaxHours {
		v.Hours = buddyMaxHours
	}
	if v.Spots < 1 {
		v.Spots = 1
	} else if v.Spots > buddyMaxSpots {
		v.Spots = buddyMaxSpots
	}
	return BuddyInput{
		Kind: v.Kind, Note: v.Note, When: v.When, ExpiresInHours: v.Hours,
		SameGender: v.Same, Spots: v.Spots, Place: v.Place,
	}, nil
}

func clipRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return string(r)
}
