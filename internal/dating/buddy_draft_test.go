package dating

import (
	"strings"
	"testing"
)

func TestParseBuddyDraft(t *testing.T) {
	in, err := ParseBuddyDraft("```json\n{\"kind\":\"movie\",\"when\":\"Saturday evening\",\"note\":\"A film\",\"expires_in_hours\":48,\"same_gender\":false,\"spots\":1,\"place\":\"the cinema\"}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if in.Kind != "movie" || in.When != "Saturday evening" || in.Note != "A film" || in.ExpiresInHours != 48 || in.Spots != 1 || in.Place != "the cinema" {
		t.Fatalf("%+v", in)
	}

	in, err = ParseBuddyDraft(`{"kind":"nope","when":"tonight","note":"hotpot","expires_in_hours":0,"spots":99,"place":"` + strings.Repeat("a", 200) + `"}`)
	if err != nil {
		t.Fatal(err)
	}
	if in.Kind != "other" || in.ExpiresInHours != 24 || in.Spots != 20 || len([]rune(in.Place)) != buddyPlaceMax {
		t.Fatalf("%+v", in)
	}

	if _, err := ParseBuddyDraft(`{"kind":"meal"}`); err == nil {
		t.Fatal("empty when and note should fail")
	}
	if _, err := ParseBuddyDraft("not json"); err == nil {
		t.Fatal("non-json should fail")
	}
}
