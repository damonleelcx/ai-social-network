package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

// fakeLLM stands in for the model. It recognises which agent is calling from the
// tools it was offered, and behaves like a well-behaved model, so the tests exercise
// everything around the model: routing, tools, verifiers, retries, guards.
type fakeLLM struct {
	calls      atomic.Int64
	down       atomic.Bool
	holdMain   atomic.Bool
	visionNude atomic.Bool
}

func newFakeLLM() *fakeLLM { return &fakeLLM{} }

type fakeMsg struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func (m fakeMsg) text() string {
	var s string
	if json.Unmarshal(m.Content, &s) == nil {
		return s
	}
	return string(m.Content)
}

var (
	reAlias  = regexp.MustCompile(`"alias": "(c\d+)"`)
	reVenue  = regexp.MustCompile(`"id":\s*"(osm-[a-z]+-\d+)"`)
	reReport = regexp.MustCompile(`report_id: ([0-9a-f-]{36})`)
)

func (f *fakeLLM) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.calls.Add(1)
	if f.down.Load() {
		http.Error(w, `{"error":"bad gateway"}`, 400) // a permanent failure, no slow retries
		return
	}
	var body struct {
		Model    string    `json:"model"`
		Stream   bool      `json:"stream"`
		Messages []fakeMsg `json:"messages"`
		Tools    []struct {
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		} `json:"tools"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Stream {
		w = &sseWriter{w}
	}
	has := map[string]bool{}
	for _, t := range body.Tools {
		has[t.Function.Name] = true
	}
	all := ""
	for _, m := range body.Messages {
		all += m.text() + "\n"
	}
	// Suki's per-call reminder isn't part of the conversation.
	if n := len(body.Messages); n > 1 && strings.HasPrefix(body.Messages[n-1].text(), "If this is your reply") {
		body.Messages = body.Messages[:n-1]
	}
	last := body.Messages[len(body.Messages)-1]
	sys := body.Messages[0].text()
	user := ""
	for i := len(body.Messages) - 1; i >= 0; i-- {
		if body.Messages[i].Role == "user" {
			user = body.Messages[i].text()
			break
		}
	}
	// The vision check is multi-part content.
	if strings.Contains(string(last.Content), "image_url") {
		nude := f.visionNude.Load()
		reply(w, fmt.Sprintf(`{"person_visible":true,"face_visible":true,"one_main_person":true,"appears_under_18":false,"nudity_or_sexual":%v,"violent_or_hateful":false,"contact_info_visible":false,"tip":"Nice light"}`, nude), nil)
		return
	}
	if strings.Contains(sys, "content moderator") {
		flag := strings.Contains(user, "NSFWMODEL")
		reply(w, fmt.Sprintf(`{"sexual_explicit":%v,"sexual_solicitation":false,"minors":false,"graphic_violence":false}`, flag), nil)
		return
	}
	if strings.Contains(sys, "You extract profile facts") {
		switch {
		case strings.Contains(user, "creative types"):
			reply(w, `{"taste_note":"Goes for creative types","occupation":"should not be saved"}`, nil)
		case strings.Contains(user, "my friend was born"):
			reply(w, `{"birthdate":"1990-03-02"}`, nil)
		case strings.Contains(user, "misread my birthday"):
			reply(w, `{"birthdate":"2005-03-02"}`, nil)
		case strings.Contains(user, "read my birthday"):
			reply(w, `{"birthdate":"1995-03-02"}`, nil)
		case strings.Contains(user, "extractable city"):
			reply(w, `{"city":"Testville","goal":"not_a_goal","interests":["origami"]}`, nil)
		default:
			reply(w, `{}`, nil)
		}
		return
	}
	if strings.Contains(sys, "Classify a message") {
		reply(w, `{"intent":"`+routeFor(user)+`"}`, nil)
		return
	}
	if f.holdMain.Load() {
		// A stuck model: the crash test takes the turn away from this worker.
		deadline := time.Now().Add(20 * time.Second)
		for f.holdMain.Load() && time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
		}
		http.Error(w, "gone", 500)
		return
	}
	done := last.Role == "tool"
	switch {
	case has["compat_record"]:
		if done || strings.Contains(all, "VERIFICATION") && false {
			reply(w, "Evaluated.", nil)
			return
		}
		var calls []any
		seen := map[string]bool{}
		for _, m := range reAlias.FindAllStringSubmatch(user, -1) {
			if seen[m[1]] || m[1] == "you" {
				continue
			}
			seen[m[1]] = true
			calls = append(calls, call(m[1], "compat_record", map[string]any{"alias": m[1], "score": 88, "reasons": []string{"shared interests"}, "shared": []string{"jazz"},
				"pitch_for_viewer": "You both light up about live jazz.", "pitch_for_candidate": "You two share a love of jazz."}))
		}
		reply(w, "", calls)
	case has["critic_verdict"]:
		if done {
			reply(w, "Reviewed.", nil)
			return
		}
		var calls []any
		seen := map[string]bool{}
		for _, m := range reAlias.FindAllStringSubmatch(user, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				calls = append(calls, call(m[1], "critic_verdict", map[string]any{"alias": m[1], "verdict": "approved", "notes": "fine"}))
			}
		}
		reply(w, "", calls)
	case has["venue_shortlist"]:
		if done {
			reply(w, "Shortlisted.", nil)
			return
		}
		var picks []map[string]any
		seen := map[string]bool{}
		for _, m := range reVenue.FindAllStringSubmatch(user, -1) {
			if !seen[m[1]] && len(picks) < 3 {
				seen[m[1]] = true
				picks = append(picks, map[string]any{"venue_id": m[1], "reason": "You both love jazz, and it's easy to talk here."})
			}
		}
		reply(w, "", []any{call("v", "venue_shortlist", map[string]any{"picks": picks})})
	case has["evaluation_record"]:
		if done {
			reply(w, "Recorded.", nil)
			return
		}
		reply(w, "", []any{call("e", "evaluation_record", map[string]any{"chemistry": 9, "respect": 9, "conversation": 8, "reliability": 6, "profile_honesty": 6, "see_again": true, "evidence": []string{"we laughed a lot"}, "safety_concern": false})})
	case has["taste_note_add"] && has["coach_suggest"] == false && strings.Contains(sys, "TasteLearner"):
		reply(w, "No new notes.", nil)
	case has["safety_report"] && strings.Contains(sys, "Sentinel"):
		if done {
			reply(w, "Assessed.", nil)
			return
		}
		m := reReport.FindStringSubmatch(all)
		if m == nil {
			reply(w, "No report id.", nil)
			return
		}
		reply(w, "", []any{call("s", "safety_report", map[string]any{"report_id": m[1], "severity": "high", "details": "Asked for money early in the chat."})})
	case has["coach_suggest"] && strings.Contains(sys, "conversation coach"):
		if done {
			reply(w, "Posted.", nil)
			return
		}
		reply(w, "", []any{call("c", "coach_suggest", map[string]any{"match_id": "a-guess-from-the-model", "note": "You both love jazz. Open with that!", "suggestions": []string{"What's the best live jazz you've seen?"}})})
	default:
		f.suki(w, body.Messages, has, user, last, done, all)
	}
}

// suki plays the conversational agent.
func (f *fakeLLM) suki(w http.ResponseWriter, msgs []fakeMsg, has map[string]bool, user string, last fakeMsg, done bool, all string) {
	if last.Role == "system" && strings.Contains(last.text(), "Rewrite your last message") {
		reply(w, "Your score is 9/10, still. <<mood=happy>>", nil) // a stubborn model
		return
	}
	if last.Role == "system" && strings.Contains(last.text(), "EVENT") {
		reply(w, "So, how did it go? <<mood=playful|replies=Loved it;It was okay;Not for me>>", nil)
		return
	}
	switch {
	case done:
		reply(w, "Saved! <<mood=happy|replies=Tell me more>>", nil)
	case strings.Contains(user, "leak please"):
		reply(w, "Between us, your score is 9/10. <<mood=playful>>", nil)
	case strings.Contains(user, "minor correction") && has["profile_save"]:
		reply(w, "", []any{call("p", "profile_save", map[string]any{"birthdate": time.Now().AddDate(-16, 0, 0).Format("2006-01-02")})})
	case strings.Contains(user, "correct my birthday") && has["profile_save"]:
		reply(w, "", []any{call("p", "profile_save", map[string]any{"birthdate": "1995-10-20"})})
	case strings.Contains(user, "nurse") && has["profile_save"]:
		reply(w, "", []any{call("p", "profile_save", map[string]any{"occupation": "nurse"})})
	case strings.Contains(user, "all done") && has["debrief_close"]:
		reply(w, "", []any{call("d", "debrief_close", map[string]any{"skip": false})})
	default:
		reply(w, "Tell me more! <<mood=thinking>>", nil)
	}
}

func routeFor(text string) string {
	switch {
	case strings.Contains(text, "nurse"):
		return "profile_work_education"
	case strings.Contains(text, "correct my birthday"), strings.Contains(text, "my friend was born"), strings.Contains(text, "minor correction"):
		return "profile_basics"
	case strings.Contains(text, "laughed"), strings.Contains(text, "all done"):
		return "debrief_answer"
	}
	return "meta_smalltalk"
}

func call(id, name string, args any) map[string]any {
	raw, _ := json.Marshal(args)
	return map[string]any{"id": id, "type": "function", "function": map[string]any{"name": name, "arguments": string(raw)}}
}

type sseWriter struct{ http.ResponseWriter }

func reply(w http.ResponseWriter, content string, calls []any) {
	if sw, ok := w.(*sseWriter); ok {
		w = sw.ResponseWriter
		w.Header().Set("Content-Type", "text/event-stream")
		send := func(delta map[string]any) {
			raw, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": delta}}})
			fmt.Fprintf(w, "data: %s\n\n", raw)
		}
		for i := 0; i < len(content); i += 4 {
			send(map[string]any{"content": content[i:min(i+4, len(content))]})
			time.Sleep(3 * time.Millisecond)
		}
		for i, c := range calls {
			m := c.(map[string]any)
			// Two pieces, as real models send them.
			f := m["function"].(map[string]any)
			args := f["arguments"].(string)
			half := len(args) / 2
			send(map[string]any{"tool_calls": []any{map[string]any{"index": i, "id": m["id"], "type": "function", "function": map[string]any{"name": f["name"], "arguments": args[:half]}}}})
			send(map[string]any{"tool_calls": []any{map[string]any{"index": i, "function": map[string]any{"arguments": args[half:]}}}})
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		return
	}
	msg := map[string]any{"role": "assistant", "content": content}
	if len(calls) > 0 {
		msg["tool_calls"] = calls
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"model": "fake", "choices": []any{map[string]any{"message": msg, "finish_reason": "stop"}},
		"usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5}})
}

// fakeGeo stands in for Nominatim and Overpass.
func fakeGeo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if strings.HasSuffix(r.URL.Path, "/overpass") {
		_, _ = w.Write([]byte(`{"elements":[
			{"type":"node","id":1,"lat":1,"lon":2,"tags":{"amenity":"bar","name":"Blue Note","addr:street":"Main St"}},
			{"type":"node","id":2,"lat":1,"lon":2,"tags":{"amenity":"restaurant","name":"Ramen Lab"}},
			{"type":"node","id":3,"lat":1,"lon":2,"tags":{"leisure":"park","name":"River Park"}},
			{"type":"node","id":4,"lat":1,"lon":2,"tags":{"amenity":"cafe","name":"Cafe One"}},
			{"type":"node","id":5,"lat":1,"lon":2,"tags":{"amenity":"cafe","name":"Private Cafe","access":"private"}}]}`))
		return
	}
	_, _ = w.Write([]byte(`[{"lat":"1.0","lon":"2.0","address":{"city":"Testville","state":"Test","country":"Testland","country_code":"tt"}}]`))
}
