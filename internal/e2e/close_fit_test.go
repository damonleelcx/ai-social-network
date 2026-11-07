package e2e

import (
	"strings"
	"testing"

	"github.com/damonleelcx/ai-social-network/internal/dating"
)

func proposalsOf(c *client) []dating.Proposal {
	var props struct {
		Proposals []dating.Proposal `json:"proposals"`
	}
	c.do("GET", "/v1/proposals", nil, 200, &props)
	return props.Proposals
}

// Nobody strong in the city: Suki still introduces the closest fit, in her own plain words.
func TestClosestFitWhenNobodyStrong(t *testing.T) {
	e := setup(t)
	e.fake.verdict.Store("close_fit")
	a := e.signup(t, "Ana", "ana@example.com")
	a.verify()
	b := e.signup(t, "Ben", "ben@example.com")
	b.verify()
	c := e.signup(t, "Cal", "cal@example.com")
	c.verify()
	b.complete("man", []string{"woman"}, "1992-02-02", []string{"jazz", "ramen", "hiking"}, 20)
	settled := func(who *client) {
		t.Helper()
		eventually(t, "their search to finish", func() bool {
			body, _ := who.lastSuki()
			return strings.Contains(body, "keep me company")
		})
	}
	settled(b)
	c.complete("man", []string{"woman"}, "1991-02-02", []string{"jazz", "ramen", "hiking"}, 30)
	settled(c)
	// Last, so only her search can find anyone.
	a.complete("woman", []string{"man"}, "1994-05-01", []string{"jazz", "ramen", "hiking"}, 10)

	eventually(t, "the closest fit to be introduced", func() bool { return len(proposalsOf(a)) > 0 })
	props := proposalsOf(a)
	if len(props) != 1 {
		t.Fatalf("one weak fit per search, got %d", len(props))
	}
	if !strings.Contains(props[0].Pitch, "Testville") || strings.Contains(props[0].Pitch, "jazz") {
		t.Fatalf("should be the plain pitch: %q", props[0].Pitch)
	}
}

// A rejected pair is never the fallback.
func TestRejectedIsNeverTheClosestFit(t *testing.T) {
	e := setup(t)
	e.fake.verdict.Store("rejected")
	a := e.signup(t, "Ana", "ana@example.com")
	a.verify()
	b := e.signup(t, "Ben", "ben@example.com")
	b.verify()
	b.complete("man", []string{"woman"}, "1992-02-02", []string{"jazz", "ramen", "hiking"}, 20)
	a.complete("woman", []string{"man"}, "1994-05-01", []string{"jazz", "ramen", "hiking"}, 10)

	eventually(t, "Suki to invite a chat", func() bool {
		body, _ := a.lastSuki()
		return strings.Contains(body, "keep me company")
	})
	if n := len(proposalsOf(a)); n != 0 {
		t.Fatalf("a rejected pair was introduced: %d", n)
	}
}

// Nobody in the city at all: Suki invites a chat.
func TestSukiInvitesAChatWhenNobodyIsThere(t *testing.T) {
	e := setup(t)
	a := e.signup(t, "Ana", "ana@example.com")
	a.verify()
	a.complete("woman", []string{"man"}, "1994-05-01", []string{"jazz", "ramen", "hiking"}, 10)

	eventually(t, "Suki to invite a chat", func() bool {
		body, _ := a.lastSuki()
		return strings.Contains(body, "keep me company")
	})
}
