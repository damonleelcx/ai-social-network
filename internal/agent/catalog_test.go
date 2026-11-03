package agent_test

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/damonleelcx/ai-social-network/internal/agent"
	"github.com/damonleelcx/ai-social-network/internal/skills"
	"github.com/damonleelcx/ai-social-network/internal/tools"
	_ "github.com/damonleelcx/ai-social-network/internal/workflows"
)

func catalog(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../../docs/01-intents-tools-skills.md")
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

// rowsIn returns the first-column names of the tables under one numbered section.
func rowsIn(doc, start, end string) map[string]bool {
	i := strings.Index(doc, start)
	j := strings.Index(doc, end)
	out := map[string]bool{}
	if i < 0 || j < 0 {
		return out
	}
	for _, m := range regexp.MustCompile("(?m)^\\| `([a-z_-]+)` \\|").FindAllStringSubmatch(doc[i:j], -1) {
		out[m[1]] = true
	}
	return out
}

func diff(a, b map[string]bool) []string {
	out := []string{}
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func TestCatalogMatchesIntents(t *testing.T) {
	doc := rowsIn(catalog(t), "## 1. Intents", "### 1.12")
	code := map[string]bool{}
	for _, d := range agent.Domains {
		for _, i := range d.Intents {
			code[i] = true
		}
	}
	for _, i := range agent.SystemIntents {
		code[i] = true
	}
	if m := diff(doc, code); len(m) > 0 {
		t.Errorf("in the catalog but not the code: %v", m)
	}
	if m := diff(code, doc); len(m) > 0 {
		t.Errorf("in the code but not the catalog: %v", m)
	}
	if len(code) < 80 {
		t.Errorf("only %d intents", len(code))
	}
}

func TestCatalogMatchesTools(t *testing.T) {
	doc := rowsIn(catalog(t), "## 2. Tools", "## 3. Skills")
	code := map[string]bool{}
	for _, n := range tools.Names() {
		code[n] = true
	}
	if m := diff(doc, code); len(m) > 0 {
		t.Errorf("in the catalog but not registered: %v", m)
	}
	if m := diff(code, doc); len(m) > 0 {
		t.Errorf("registered but not in the catalog: %v", m)
	}
}

func TestCatalogMatchesSkills(t *testing.T) {
	doc := rowsIn(catalog(t), "## 3. Skills", "## 4. Agents")
	code := map[string]bool{}
	for _, s := range skills.All() {
		if s.Name != skills.Fallback {
			code[s.Name] = true
		}
	}
	if m := diff(doc, code); len(m) > 0 {
		t.Errorf("in the catalog but not registered: %v", m)
	}
	if m := diff(code, doc); len(m) > 0 {
		t.Errorf("registered but not in the catalog: %v", m)
	}
}

// The heart of the hidden-scoring design: no tool that touches a score or an
// evaluation can ever be offered to the agent people talk to.
func TestSukiCannotHoldAHiddenTool(t *testing.T) {
	forbidden := map[string]bool{"candidates_gather": true, "compat_record": true, "critic_verdict": true, "proposal_publish": true,
		"evaluation_record": true, "taste_mark_noted": true, "photo_vision_review": true, "venue_shortlist": true, "venue_search": true, "chat_stage_set": true, "chat_prompt_post": true}
	for _, name := range agent.AllSukiTools() {
		if forbidden[name] {
			t.Errorf("Suki can hold %s", name)
		}
		if _, ok := tools.Get(name); !ok {
			t.Errorf("Suki's list names %s, which isn't registered", name)
		}
	}
}

func TestEveryDomainToolExistsAndSkillsUseRegisteredTools(t *testing.T) {
	for _, s := range skills.All() {
		for _, st := range s.Steps {
			if st.Tool != "" {
				if _, ok := tools.Get(st.Tool); !ok {
					t.Errorf("skill %s runs unknown tool %s", s.Name, st.Tool)
				}
			}
			for _, n := range st.Tools {
				if _, ok := tools.Get(n); !ok {
					t.Errorf("skill %s offers unknown tool %s", s.Name, n)
				}
			}
		}
	}
}
