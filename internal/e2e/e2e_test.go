// Package e2e walks the whole product over real HTTP against real Postgres and
// MongoDB, with the two outside services (the model, the map) faked. It proves the
// journey works, and that no score ever reaches a client along the way.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damonleelcx/ai-social-network/internal/agent"
	"github.com/damonleelcx/ai-social-network/internal/auth"
	"github.com/damonleelcx/ai-social-network/internal/chatstore"
	"github.com/damonleelcx/ai-social-network/internal/dating"
	"github.com/damonleelcx/ai-social-network/internal/db"
	"github.com/damonleelcx/ai-social-network/internal/engine"
	"github.com/damonleelcx/ai-social-network/internal/httpapi"
	"github.com/damonleelcx/ai-social-network/internal/llm"
	"github.com/damonleelcx/ai-social-network/internal/mail"
	"github.com/damonleelcx/ai-social-network/internal/photostore"
	"github.com/damonleelcx/ai-social-network/internal/places"
	"github.com/damonleelcx/ai-social-network/internal/workflows"
)

// ── Harness ─────────────────────────────────────────────────────────────────

type capturedMail struct {
	mu   sync.Mutex
	msgs []mail.Message
}

func (c *capturedMail) Send(_ context.Context, m mail.Message) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, m)
	return "id", nil
}

func (c *capturedMail) Enabled() bool { return true }

var reCode = regexp.MustCompile(`\b\d{6}\b`)

func (c *capturedMail) lastCode(t *testing.T, to string) string {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := len(c.msgs) - 1; i >= 0; i-- {
		if c.msgs[i].To == to {
			if m := reCode.FindString(c.msgs[i].Text); m != "" {
				return m
			}
		}
	}
	t.Fatalf("no code mailed to %s", to)
	return ""
}

type env struct {
	api      *httptest.Server
	pool     *pgxpool.Pool
	chats    *chatstore.Store
	mail     *capturedMail
	bodies   *bytes.Buffer
	bodiesMu sync.Mutex
	store    *engine.Store
	model    *engine.Model
	svc      *dating.Service
	fake     *fakeLLM
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	photos   photostore.Store
}

func setup(t *testing.T) *env {
	t.Helper()
	base := os.Getenv("SYN_TEST_DATABASE_URL")
	if base == "" {
		base = "postgres://syn@127.0.0.1:55870/syn?sslmode=disable"
	}
	mongoURL := os.Getenv("SYN_TEST_MONGO_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://127.0.0.1:55871"
	}
	ctx0, c0 := context.WithTimeout(context.Background(), 3*time.Second)
	defer c0()
	conn, err := pgx.Connect(ctx0, base)
	if err != nil {
		t.Skipf("no test postgres: %v", err)
	}
	_ = conn.Close(ctx0)
	// One database per test process: this test wipes it, and two runs can overlap.
	name := fmt.Sprintf("syn_e2e_%d", os.Getpid())
	url, err := db.EnsureDatabase(context.Background(), base, name)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DROP SCHEMA IF EXISTS hidden CASCADE; DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	chats, err := chatstore.Open(ctx, mongoURL, "syn_e2e_"+fmt.Sprint(time.Now().UnixNano()))
	if err != nil {
		t.Skipf("no test mongo: %v", err)
	}
	fake := newFakeLLM()
	llmSrv := httptest.NewServer(fake)
	geo := httptest.NewServer(http.HandlerFunc(fakeGeo))
	store := &engine.Store{Pool: pool, Conversations: chats}
	client := llm.New(llmSrv.URL, "test")
	client.Retries = 0
	model := &engine.Model{Client: client, Store: store}
	photos := &photostore.Disk{Dir: t.TempDir()}
	plc := &places.Client{Pool: pool, NominatimURL: geo.URL, OverpassURL: geo.URL + "/overpass"}
	svc := &dating.Service{Pool: pool, Chats: chats, DateMinLead: time.Millisecond}
	workflows.Init(&workflows.Deps{Pool: pool, Store: store, Model: model, Chats: chats, Photos: photos, Places: plc, Dating: svc, FastModel: "fast", DebriefDelay: time.Millisecond})
	m := &capturedMail{}
	a := &auth.Service{Pool: pool, Mailer: m, PublicOrigin: "http://test", SessionTTL: time.Hour}
	hub := httpapi.NewHub()
	srv := &httpapi.Server{Pool: pool, Auth: a, Chats: chats, Photos: photos, Places: plc, Dating: svc, Hub: hub}
	a.BeforeDelete = srv.BeforeDelete
	e := &env{pool: pool, chats: chats, mail: m, bodies: &bytes.Buffer{}, store: store, model: model, svc: svc, fake: fake, cancel: cancel, photos: photos}
	// Every response body is kept, so the test can search all of them for scores.
	handler := srv.Handler()
	e.api = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &recorder{ResponseWriter: w, env: e}
		handler.ServeHTTP(rec, r)
	}))
	planner := &engine.Planner{Store: store, Model: model, LLM: "main"}
	for i := 0; i < 3; i++ {
		w := &engine.Worker{ID: fmt.Sprintf("w%d", i), Store: store, Model: model, Planner: planner, LLM: "main", Mailer: m, Lease: 20 * time.Second, Wake: make(chan struct{})}
		e.wg.Add(1)
		go func() { defer e.wg.Done(); w.Run(ctx) }()
	}
	for i := 0; i < 2; i++ {
		r := &agent.Runner{ID: fmt.Sprintf("t%d", i), Pool: pool, Chats: chats, Model: model, Dating: svc, LLM: "main", FastLLM: "fast", Lease: 20 * time.Second, Wake: make(chan struct{})}
		e.wg.Add(1)
		go func() { defer e.wg.Done(); r.Run(ctx) }()
	}
	e.wg.Add(1)
	go func() { defer e.wg.Done(); engine.Listen(ctx, pool, nil, hub.Publish) }()
	sched := &engine.Scheduler{Store: store, Model: model, FastLLM: "fast", Mailer: m, PublicOrigin: "http://test", Interval: 200 * time.Millisecond}
	e.wg.Add(1)
	go func() { defer e.wg.Done(); sched.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		done := make(chan struct{})
		go func() { e.wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(15 * time.Second):
			t.Log("workers slow to stop")
		}
		_ = chats.Drop(context.Background())
		e.api.Close()
		llmSrv.Close()
		geo.Close()
		pool.Close()
	})
	return e
}

type recorder struct {
	http.ResponseWriter
	env *env
}

func (r *recorder) Write(b []byte) (int, error) {
	if r.Header().Get("Content-Type") == "application/json" {
		r.env.bodiesMu.Lock()
		r.env.bodies.Write(b)
		r.env.bodiesMu.Unlock()
	}
	return r.ResponseWriter.Write(b)
}

func (r *recorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

type client struct {
	t     *testing.T
	e     *env
	token string
	id    string
	email string
}

func (c *client) do(method, path string, body any, want int, out any) {
	c.t.Helper()
	var rd io.Reader
	if raw, ok := body.([]byte); ok {
		rd = bytes.NewReader(raw)
	} else if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.e.api.URL+path, rd)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if want != 0 && resp.StatusCode != want {
		c.t.Fatalf("%s %s: status %d, want %d: %s", method, path, resp.StatusCode, want, raw)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			c.t.Fatalf("%s %s: %v: %s", method, path, err, raw)
		}
	}
}

func (e *env) signup(t *testing.T, name, email string) *client {
	t.Helper()
	c := &client{t: t, e: e, email: email}
	var r struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	c.do("POST", "/v1/auth/signup", map[string]any{"email": email, "password": "correct horse battery", "name": name, "accept_terms": true}, 201, &r)
	c.token, c.id = r.Token, r.User.ID
	return c
}

func (c *client) verify() {
	c.t.Helper()
	c.do("POST", "/v1/auth/verify", map[string]any{"code": c.e.mail.lastCode(c.t, c.email)}, 200, nil)
}

func photo(shade uint8) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 400, 400))
	for x := 0; x < 400; x++ {
		for y := 0; y < 400; y++ {
			img.Set(x, y, color.RGBA{shade, uint8(x / 2), uint8(y / 2), 255})
		}
	}
	var b bytes.Buffer
	_ = jpeg.Encode(&b, img, nil)
	return b.Bytes()
}

func eventually(t *testing.T, what string, f func() bool) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if f() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", what)
}

func (c *client) complete(gender string, seeking []string, birth string, interests []string, shade uint8) {
	c.t.Helper()
	c.do("PATCH", "/v1/profile", map[string]any{"birthdate": birth, "gender": gender, "seeking": seeking, "goal": "long_term", "interests": interests,
		"bio": "Here for real conversations.", "age_min": 25, "age_max": 45, "smoking": "no"}, 200, nil)
	c.do("POST", "/v1/profile/city", map[string]any{"query": "Testville"}, 200, nil)
	c.do("POST", "/v1/photos", photo(shade), 201, nil)
}

func (c *client) lastSuki() (string, map[string]any) {
	var r struct {
		Messages []chatstore.Message `json:"messages"`
	}
	c.do("GET", "/v1/agent/messages", nil, 200, &r)
	for i := len(r.Messages) - 1; i >= 0; i-- {
		if r.Messages[i].Role == "assistant" {
			return r.Messages[i].Body, r.Messages[i].Meta
		}
	}
	return "", nil
}

func (c *client) askSuki(text string) (string, map[string]any) {
	c.t.Helper()
	before := c.countSuki()
	c.do("POST", "/v1/agent/messages", map[string]any{"text": text, "client_id": fmt.Sprint(time.Now().UnixNano())}, 202, nil)
	eventually(c.t, "Suki to answer "+text, func() bool { return c.countSuki() > before })
	return c.lastSuki()
}

func (c *client) countSuki() int {
	var r struct {
		Messages []chatstore.Message `json:"messages"`
	}
	c.do("GET", "/v1/agent/messages", nil, 200, &r)
	n := 0
	for _, m := range r.Messages {
		if m.Role == "assistant" {
			n++
		}
	}
	return n
}

// ── The journey ─────────────────────────────────────────────────────────────

func TestAuthFlow(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Rae", "rae@example.com")

	// Signing up without accepting the terms is refused.
	(&client{t: t, e: e}).do("POST", "/v1/auth/signup", map[string]any{"email": "x@example.com", "password": "correct horse battery"}, 422, nil)
	// A duplicate address is refused, in any case.
	(&client{t: t, e: e}).do("POST", "/v1/auth/signup", map[string]any{"email": "RAE@example.com", "password": "correct horse battery", "accept_terms": true}, 422, nil)
	// A weak password is refused.
	(&client{t: t, e: e}).do("POST", "/v1/auth/signup", map[string]any{"email": "y@example.com", "password": "short", "accept_terms": true}, 422, nil)

	// Unverified: almost everything is closed.
	c.do("GET", "/v1/profile", nil, 403, nil)
	c.do("GET", "/v1/me", nil, 200, nil)
	// A wrong code doesn't verify, and five wrong codes kill the code.
	c.do("POST", "/v1/auth/verify", map[string]any{"code": "000000"}, 422, nil)
	c.do("GET", "/v1/profile", nil, 403, nil)
	good := e.mail.lastCode(t, c.email)
	for i := 0; i < 4; i++ {
		c.do("POST", "/v1/auth/verify", map[string]any{"code": "000000"}, 422, nil)
	}
	c.do("POST", "/v1/auth/verify", map[string]any{"code": good}, 422, nil)
	// A fresh code works.
	c.do("POST", "/v1/auth/resend", nil, 200, nil)
	c.verify()
	c.do("GET", "/v1/profile", nil, 200, nil)

	// Sign out ends the session.
	old := c.token
	c.do("POST", "/v1/auth/signout", nil, 200, nil)
	c.do("GET", "/v1/me", nil, 401, nil)
	// A wrong password fails, and doesn't reveal whether the address exists.
	s := &client{t: t, e: e}
	s.do("POST", "/v1/auth/signin", map[string]any{"email": c.email, "password": "wrong password here"}, 401, nil)
	s.do("POST", "/v1/auth/signin", map[string]any{"email": "nobody@example.com", "password": "wrong password here"}, 401, nil)

	// Forgot password: same answer for a real and unknown address.
	s.do("POST", "/v1/auth/forgot", map[string]any{"email": "nobody@example.com"}, 200, nil)
	s.do("POST", "/v1/auth/forgot", map[string]any{"email": c.email}, 200, nil)
	code := e.mail.lastCode(t, c.email)
	s.do("POST", "/v1/auth/reset", map[string]any{"email": c.email, "code": "000000", "password": "a whole new password"}, 422, nil)
	s.do("POST", "/v1/auth/reset", map[string]any{"email": c.email, "code": code, "password": "a whole new password"}, 200, nil)
	// A used code can't be replayed.
	s.do("POST", "/v1/auth/reset", map[string]any{"email": c.email, "code": code, "password": "yet another password"}, 422, nil)
	var r struct {
		Token string `json:"token"`
	}
	s.do("POST", "/v1/auth/signin", map[string]any{"email": c.email, "password": "a whole new password"}, 200, &r)
	if r.Token == "" || r.Token == old {
		t.Fatal("no fresh token")
	}
}

func TestUserMapOnlyShowsCrowds(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	add := func(prefix string, n int, lat, lon float64, status string, verified bool) {
		for i := 0; i < n; i++ {
			var id string
			var at any
			if verified {
				at = time.Now()
			}
			if err := e.pool.QueryRow(ctx, `INSERT INTO users (email, password_hash, email_verified_at) VALUES ($1, 'x', $2) RETURNING id`,
				fmt.Sprintf("%s%d@test", prefix, i), at).Scan(&id); err != nil {
				t.Fatal(err)
			}
			if _, err := e.pool.Exec(ctx, `INSERT INTO profiles (user_id, display_name, city_key, city_name, city_lat, city_lon, status) VALUES ($1, 'Secret Name', 'k', 'Secret City', $2, $3, $4)`,
				id, lat, lon, status); err != nil {
				t.Fatal(err)
			}
		}
	}
	add("tokyo", 7, 35.68, 139.69, "active", true)
	add("reykjavik", 4, 64.15, -21.94, "active", true) // too few to show
	add("lima", 5, -12.05, -77.04, "paused", true)     // stepped back
	add("oslo", 5, 59.91, 10.75, "onboarding", false)  // never verified

	res, err := http.Get(e.api.URL + "/v1/public/user-map") // no session
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		t.Fatalf("status %d: %s", res.StatusCode, body)
	}
	var got struct {
		Total int `json:"total"`
		Cells []struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
			N   int     `json:"n"`
		} `json:"cells"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.Total != 16 {
		t.Fatalf("total = %d, want the 16 verified accounts", got.Total)
	}
	// Seven people in Tokyo come back as a rounded five, at the middle of the cell.
	if len(got.Cells) != 1 || got.Cells[0].Lat != 37.5 || got.Cells[0].Lon != 137.5 || got.Cells[0].N != 5 {
		t.Fatalf("cells = %+v", got.Cells)
	}
	if strings.Contains(string(body), "Secret") || strings.Contains(string(body), "139.69") {
		t.Fatalf("leaked detail: %s", body)
	}
}

func TestUnderageIsStopped(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Kid", "kid@example.com")
	c.verify()
	birth := time.Now().AddDate(-16, 0, 0).Format("2006-01-02")
	c.do("PATCH", "/v1/profile", map[string]any{"birthdate": birth}, 403, nil)
	var st string
	if err := e.pool.QueryRow(context.Background(), `SELECT status FROM profiles WHERE user_id=$1`, c.id).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "held" {
		t.Fatalf("status %s", st)
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM reports WHERE subject_id=$1 AND severity='critical'`, c.id).Scan(&n)
	if n != 1 {
		t.Fatalf("%d reports", n)
	}
}

func TestWholeJourney(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	// Two people in one city who suit each other, one who shares nothing and has
	// the wrong gender preference, and one in another city.
	ana := e.signup(t, "Ana", "ana@example.com")
	ben := e.signup(t, "Ben", "ben@example.com")
	wrong := e.signup(t, "Cy", "cy@example.com")
	ana.verify()
	ben.verify()
	wrong.verify()

	// Suki greets them after verification, and can save what they say.
	if body, _ := ana.lastSuki(); !strings.Contains(body, "Suki") {
		t.Fatalf("no welcome: %q", body)
	}
	body, meta := ana.askSuki("I work as a nurse")
	if !strings.Contains(body, "Saved") || meta["mood"] != "happy" {
		t.Fatalf("reply %q %v", body, meta)
	}
	var prof struct {
		Profile struct {
			Occupation string `json:"occupation"`
		} `json:"profile"`
	}
	ana.do("GET", "/v1/profile", nil, 200, &prof)
	if prof.Profile.Occupation != "nurse" {
		t.Fatalf("Suki's tool didn't save: %+v", prof)
	}

	// Asking for a score gets the fixed deflection, with no model call.
	calls := e.fake.calls.Load()
	if body, _ := ana.askSuki("what's my score?"); body == "" || !strings.Contains(body, "no number") {
		t.Fatalf("score probe: %q", body)
	}
	if e.fake.calls.Load() != calls {
		t.Fatal("the score probe reached the model")
	}
	// A leaking draft never reaches the person, even if the rewrite leaks too.
	if body, _ := ana.askSuki("leak please"); strings.Contains(body, "9/10") || strings.Contains(strings.ToLower(body), "your score") {
		t.Fatalf("leak reached the person: %q", body)
	}

	// Onboarding completes, photos are reviewed, and matching starts by itself.
	ana.complete("woman", []string{"man"}, "1994-05-01", []string{"jazz", "ramen", "hiking"}, 10)
	ben.complete("man", []string{"woman"}, "1992-02-02", []string{"jazz", "ramen", "chess"}, 20)
	wrong.complete("man", []string{"man"}, "1990-02-02", []string{"jazz", "ramen", "golf"}, 30)

	var props struct {
		Proposals []dating.Proposal `json:"proposals"`
	}
	eventually(t, "Ana to get an introduction", func() bool {
		ana.do("GET", "/v1/proposals", nil, 200, &props)
		return len(props.Proposals) > 0
	})
	if len(props.Proposals) != 1 || props.Proposals[0].Person.DisplayName != "Ben" {
		t.Fatalf("wrong proposals: %+v", props.Proposals)
	}
	p := props.Proposals[0]
	if p.Pitch == "" || len(p.Shared) == 0 {
		t.Fatalf("pitch missing: %+v", p)
	}
	if strings.Contains(p.Person.Bio, "birthdate") {
		t.Fatal("private data in a card")
	}
	// Someone with no introduction can't see a stranger's photo, and neither can Ben's
	// photo be seen before it's approved by people outside the proposal.
	photoID := p.Person.PhotoIDs[0]
	wrong.do("GET", "/v1/media/"+photoID, nil, 404, nil)
	ana.do("GET", "/v1/media/"+photoID, nil, 200, nil)
	// Nobody can read a proposal that isn't theirs.
	wrong.do("GET", "/v1/proposals/"+p.ID, nil, 404, nil)

	// A malformed id is simply not found.
	ana.do("GET", "/v1/proposals/not-a-uuid", nil, 404, nil)
	ana.do("GET", "/v1/matches/not-a-uuid", nil, 404, nil)
	ana.do("GET", "/v1/dates/not-a-uuid", nil, 404, nil)

	// One yes makes no match, and the other person isn't told who said what.
	var dec struct {
		MatchID string `json:"match_id"`
	}
	ana.do("POST", "/v1/proposals/"+p.ID+"/decide", map[string]any{"decision": "accept"}, 200, &dec)
	if dec.MatchID != "" {
		t.Fatal("a match after one yes")
	}
	var benProp struct {
		Proposal dating.Proposal `json:"proposal"`
	}
	ben.do("GET", "/v1/proposals/"+p.ID, nil, 200, &benProp)
	if benProp.Proposal.Decision != "" {
		t.Fatal("Ben sees a decision of his he never made")
	}
	// Chat is closed until both say yes.
	ben.do("POST", "/v1/proposals/"+p.ID+"/decide", map[string]any{"decision": "accept"}, 200, &dec)
	if dec.MatchID == "" {
		t.Fatal("no match after two yeses")
	}
	matchID := dec.MatchID
	wrong.do("GET", "/v1/matches/"+matchID, nil, 404, nil)
	wrong.do("POST", "/v1/matches/"+matchID+"/messages", map[string]any{"text": "hi"}, 404, nil)

	// The coach writes each of them a private hint with suggestions.
	var hints struct {
		Hints []dating.Hint `json:"hints"`
	}
	eventually(t, "the coach's hint", func() bool {
		ana.do("GET", "/v1/matches/"+matchID+"/hints", nil, 200, &hints)
		return len(hints.Hints) > 0
	})
	if len(hints.Hints[0].Suggestions) == 0 {
		t.Fatalf("hint without suggestions: %+v", hints.Hints[0])
	}
	var benHints struct {
		Hints []dating.Hint `json:"hints"`
	}
	eventually(t, "Ben's hint", func() bool {
		ben.do("GET", "/v1/matches/"+matchID+"/hints", nil, 200, &benHints)
		return len(benHints.Hints) > 0
	})
	if benHints.Hints[0].ID == hints.Hints[0].ID {
		t.Fatal("a private hint was shared")
	}
	wrong.do("GET", "/v1/matches/"+matchID+"/hints", nil, 404, nil)

	// They talk. A retried send lands on one message.
	for i := 0; i < 2; i++ {
		ana.do("POST", "/v1/matches/"+matchID+"/messages", map[string]any{"text": "Jazz on Sunday?", "client_id": "same"}, 201, nil)
	}
	ben.do("POST", "/v1/matches/"+matchID+"/messages", map[string]any{"text": "Yes please", "client_id": "b1"}, 201, nil)
	var chat struct {
		Messages []chatstore.Message `json:"messages"`
	}
	ben.do("GET", "/v1/matches/"+matchID+"/messages", nil, 200, &chat)
	if len(chat.Messages) != 2 {
		t.Fatalf("retried send duplicated: %d messages", len(chat.Messages))
	}
	// Messages really are in MongoDB.
	stored, err := e.chats.ChatRecent(ctx, matchID, 10)
	if err != nil || len(stored) != 2 {
		t.Fatalf("mongo has %d, err %v", len(stored), err)
	}
	ben.do("POST", "/v1/matches/"+matchID+"/read", map[string]any{"seq": chat.Messages[0].Seq}, 200, nil)
	ana.do("GET", "/v1/matches/"+matchID+"/messages", nil, 200, &chat)
	if chat.Messages[0].ReadAt == nil {
		t.Fatal("read receipt missing")
	}

	// A money request warns the recipient privately and reaches a human.
	ben.do("POST", "/v1/matches/"+matchID+"/messages", map[string]any{"text": "can you send me $500 for rent", "client_id": "b2"}, 201, nil)
	eventually(t, "the moderators' report", func() bool {
		var n int
		_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM reports WHERE subject_id=$1 AND reason='money_request'`, ben.id).Scan(&n)
		return n == 1
	})
	ana.do("GET", "/v1/matches/"+matchID+"/hints", nil, 200, &hints)
	found := false
	for _, h := range hints.Hints {
		found = found || strings.Contains(h.Body, "money")
	}
	if !found {
		t.Fatal("Ana wasn't warned")
	}

	// A date: real places, both vote, a time is proposed and only the other confirms.
	var ds struct {
		DateID string `json:"date_id"`
	}
	ana.do("POST", "/v1/matches/"+matchID+"/date", map[string]any{"constraints": []string{"not too loud"}}, 201, &ds)
	var again struct {
		DateID string `json:"date_id"`
	}
	ben.do("POST", "/v1/matches/"+matchID+"/date", nil, 201, &again)
	if again.DateID != ds.DateID {
		t.Fatal("two open plans for one match")
	}
	var dp struct {
		Date dating.DatePlan `json:"date"`
	}
	eventually(t, "the shortlist", func() bool {
		ana.do("GET", "/v1/dates/"+ds.DateID, nil, 200, &dp)
		return dp.Date.Status == "voting"
	})
	if len(dp.Date.Venues) == 0 || len(dp.Date.Venues) > 3 || dp.Date.Venues[0].Reason == "" {
		t.Fatalf("shortlist: %+v", dp.Date.Venues)
	}
	first, second := dp.Date.Venues[0].ID, dp.Date.Venues[len(dp.Date.Venues)-1].ID
	ana.do("POST", "/v1/dates/"+ds.DateID+"/vote", map[string]any{"venue_id": "not-on-the-list"}, 422, nil)
	ana.do("POST", "/v1/dates/"+ds.DateID+"/vote", map[string]any{"venue_id": second}, 200, nil)
	ben.do("GET", "/v1/dates/"+ds.DateID, nil, 200, &dp)
	if dp.Date.MyVote != "" || !dp.Date.OtherVoted {
		t.Fatalf("a vote leaked or wasn't counted: %+v", dp.Date)
	}
	ben.do("POST", "/v1/dates/"+ds.DateID+"/vote", map[string]any{"venue_id": first}, 200, &dp)
	if dp.Date.Status != "scheduling" || dp.Date.Venue == nil || dp.Date.Venue.ID != first {
		t.Fatalf("after both voted: %+v", dp.Date)
	}
	at := time.Now().Add(50 * time.Millisecond).UTC().Format(time.RFC3339Nano)
	ana.do("POST", "/v1/dates/"+ds.DateID+"/time", map[string]any{"at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)}, 422, nil)
	ana.do("POST", "/v1/dates/"+ds.DateID+"/time", map[string]any{"at": time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)}, 200, nil)
	ana.do("POST", "/v1/dates/"+ds.DateID+"/confirm", nil, 422, nil) // can't confirm your own proposal
	_ = at
	ben.do("POST", "/v1/dates/"+ds.DateID+"/confirm", nil, 200, &dp)
	if dp.Date.Status != "confirmed" || dp.Date.ScheduledAt == nil {
		t.Fatalf("confirm: %+v", dp.Date)
	}
	var share struct {
		Text string `json:"text"`
	}
	ana.do("GET", "/v1/dates/"+ds.DateID+"/share", nil, 200, &share)
	if !strings.Contains(share.Text, "Ben") {
		t.Fatalf("share: %s", share.Text)
	}

	// Time passes: move the date into the past, and let the timer wake things up.
	if _, err := e.pool.Exec(ctx, `UPDATE date_plans SET scheduled_at=now()-interval '2 hours', reminded_24=false, reminded_2=true WHERE id=$1`, ds.DateID); err != nil {
		t.Fatal(err)
	}
	workflows.Cron(ctx)
	eventually(t, "Suki to ask Ana how it went", func() bool {
		var n int
		_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM debriefs WHERE date_id=$1`, ds.DateID).Scan(&n)
		b, _ := ana.lastSuki()
		return n == 2 && strings.Contains(b, "how did it go")
	})
	var checkins int
	_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE key LIKE 'checkin:%'`).Scan(&checkins)
	if checkins != 2 {
		t.Fatalf("%d safety check-ins", checkins)
	}

	// Ana tells Suki about it in her own words; her words become a hidden evaluation.
	ana.askSuki("we laughed a lot and he was kind")
	ana.askSuki("all done, I'd see him again")
	eventually(t, "the evaluation", func() bool {
		var n int
		_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM hidden.evaluations WHERE rater_id=$1 AND subject_id=$2`, ana.id, ben.id).Scan(&n)
		return n == 1
	})
	var overall float64
	var see bool
	if err := e.pool.QueryRow(ctx, `SELECT overall, see_again FROM hidden.evaluations WHERE rater_id=$1`, ana.id).Scan(&overall, &see); err != nil {
		t.Fatal(err)
	}
	if !see || overall <= 0 {
		t.Fatalf("evaluation %v %v", overall, see)
	}
	// One rater can't change a reputation.
	var rep float64
	eventually(t, "reputation to be recomputed", func() bool {
		return e.pool.QueryRow(ctx, `SELECT score FROM hidden.reputation WHERE user_id=$1`, ben.id).Scan(&rep) == nil
	})
	if rep != 0 {
		t.Fatalf("one rating moved reputation to %v", rep)
	}
	// The behaviour taught the taste model.
	eventually(t, "the taste model to learn", func() bool {
		var signals int
		_ = e.pool.QueryRow(ctx, `SELECT signals FROM hidden.taste WHERE user_id=$1`, ana.id).Scan(&signals)
		return signals >= 3
	})

	// Settings and memory work.
	var st struct {
		Settings map[string]any `json:"settings"`
	}
	ana.do("PATCH", "/v1/settings", map[string]any{"suki_tone": "direct", "daily_matches": 2}, 200, &st)
	if st.Settings["suki_tone"] != "direct" {
		t.Fatalf("settings: %+v", st.Settings)
	}
	ana.do("PATCH", "/v1/settings", map[string]any{"made_up": true}, 422, nil)
	ana.do("PATCH", "/v1/settings", map[string]any{"daily_matches": 50}, 422, nil)
	ana.do("POST", "/v1/memories/reset", nil, 200, nil)
	var left int
	_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM hidden.taste WHERE user_id=$1`, ana.id).Scan(&left)
	if left != 0 {
		t.Fatal("reset left learned taste behind")
	}

	// Export contains their words and no score.
	ana.do("GET", "/v1/account/export", nil, 200, &map[string]any{})

	// Every timeline event is free of scores, because moderators can read them.
	var evs int
	_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM events WHERE data::text ~* '"(score|overall|prior|reputation)"'`).Scan(&evs)
	if evs != 0 {
		t.Errorf("%d events hold a score", evs)
	}

	// Nothing any client received, over this entire journey, carried a score.
	e.bodiesMu.Lock()
	all := e.bodies.String()
	e.bodiesMu.Unlock()
	for _, banned := range []string{`"score"`, `"prior"`, `"overall"`, `"reputation"`, `"chemistry"`, `"respect"`, `"weights"`, `"verdict"`, `"see_again"`, `"reasons"`, `"critic_notes"`, `"evidence"`, `"features"`} {
		if strings.Contains(all, banned) {
			t.Errorf("a client received %s", banned)
		}
	}
	if len(all) < 2000 {
		t.Fatalf("the scan saw only %d bytes", len(all))
	}

	// Block ends everything at once, and the pair is never proposed again.
	ana.do("POST", "/v1/blocks", map[string]any{"user_id": ben.id}, 200, nil)
	ana.do("POST", "/v1/matches/"+matchID+"/messages", map[string]any{"text": "still there?"}, 422, nil)
	ana.do("GET", "/v1/media/"+photoID, nil, 404, nil)

	// Deleting an account removes the chats and photos too, as promised.
	ben.do("POST", "/v1/account/delete", map[string]any{"password": "wrong password here"}, 401, nil)
	ben.do("POST", "/v1/account/delete", map[string]any{"password": "correct horse battery"}, 200, nil)
	ben.do("GET", "/v1/me", nil, 401, nil)
	stored, err = e.chats.ChatRecent(ctx, matchID, 10)
	if err != nil || len(stored) != 0 {
		t.Fatalf("mongo kept %d messages after deletion, err %v", len(stored), err)
	}
	var rows int
	_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id=$1`, ben.id).Scan(&rows)
	if rows != 0 {
		t.Fatalf("%d photo rows after deletion", rows)
	}
}

func TestDealbreakersAndCityKeepPeopleApart(t *testing.T) {
	e := setup(t)
	a := e.signup(t, "Ada", "ada@example.com")
	b := e.signup(t, "Bo", "bo@example.com")
	a.verify()
	b.verify()
	a.do("PATCH", "/v1/profile", map[string]any{"dealbreakers": map[string]any{"smoking": []string{"no"}}}, 200, nil) // nobody who doesn't smoke
	a.complete("woman", []string{"man"}, "1994-05-01", []string{"jazz", "ramen", "hiking"}, 40)
	b.complete("man", []string{"woman"}, "1992-02-02", []string{"jazz", "ramen", "chess"}, 50)
	eventually(t, "both searches to finish", func() bool {
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM goals WHERE skill='match-search' AND status='completed'`).Scan(&n)
		return n >= 2
	})
	var props struct {
		Proposals []dating.Proposal `json:"proposals"`
	}
	a.do("GET", "/v1/proposals", nil, 200, &props)
	if len(props.Proposals) != 0 {
		t.Fatalf("a dealbreaker was ignored: %+v", props.Proposals)
	}
}

// Kill the worker mid-turn. Another picks it up and the person gets exactly one reply.
func TestTurnSurvivesWorkerCrash(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Dee", "dee@example.com")
	c.verify()
	e.fake.holdMain.Store(true)
	before := c.countSuki()
	c.do("POST", "/v1/agent/messages", map[string]any{"text": "I work as a nurse", "client_id": "crash1"}, 202, nil)
	eventually(t, "a worker to take the turn", func() bool {
		var leased int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM agent_turns WHERE status='leased'`).Scan(&leased)
		return leased == 1
	})
	// Simulate the crash: the lease expires while the worker has gone quiet.
	if _, err := e.pool.Exec(context.Background(), `UPDATE agent_turns SET status='ready', lease_owner=NULL, lease_expires_at=NULL, lease_epoch=lease_epoch+1 WHERE status='leased'`); err != nil {
		t.Fatal(err)
	}
	e.fake.holdMain.Store(false)
	eventually(t, "the turn to finish", func() bool {
		var done int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM agent_turns WHERE status='done'`).Scan(&done)
		return done == 1
	})
	time.Sleep(500 * time.Millisecond)
	if got := c.countSuki() - before; got != 1 {
		t.Fatalf("%d replies after a crash, want exactly 1", got)
	}
}

// With the model down, Suki says so after her retries rather than going silent,
// and the deterministic safety path still works.
func TestModelOutage(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Eli", "eli@example.com")
	c.verify()
	e.fake.down.Store(true)
	if body, _ := c.askSuki("I don't want to be here anymore"); !strings.Contains(body, "988") {
		t.Fatalf("crisis reply needs no model: %q", body)
	}
	before := c.countSuki()
	c.do("POST", "/v1/agent/messages", map[string]any{"text": "hello there", "client_id": "o1"}, 202, nil)
	eventually(t, "an honest failure message", func() bool { return c.countSuki() > before })
	if body, _ := c.lastSuki(); !strings.Contains(body, "foggy") {
		t.Fatalf("reply: %q", body)
	}
}

func TestPhotoReviewFailsClosed(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Fay", "fay@example.com")
	c.verify()
	e.fake.visionNude.Store(true)
	var r struct {
		ID string `json:"id"`
	}
	c.do("POST", "/v1/photos", photo(60), 201, &r)
	var ps struct {
		Photos []struct {
			ID, Status, Reason string
		} `json:"photos"`
	}
	eventually(t, "the review", func() bool {
		c.do("GET", "/v1/photos", nil, 200, &ps)
		return len(ps.Photos) == 1 && ps.Photos[0].Status != "pending"
	})
	if ps.Photos[0].Status != "rejected" {
		t.Fatalf("status %s", ps.Photos[0].Status)
	}
	// EXIF can't survive: the stored file is our own re-encode.
	c.do("POST", "/v1/photos", []byte("not an image"), 422, nil)
}

// NSFW content stops at every door, and ordinary flirting gets through.
func TestNSFWIsFiltered(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	a := e.signup(t, "Gia", "gia@example.com")
	b := e.signup(t, "Hal", "hal@example.com")
	a.verify()
	b.verify()

	// Profile text, by word list and by the model, in any field.
	a.do("PATCH", "/v1/profile", map[string]any{"bio": "send nudes"}, 422, nil)
	a.do("PATCH", "/v1/profile", map[string]any{"bio": "NSFWMODEL phrased cleverly"}, 422, nil)
	a.do("PATCH", "/v1/profile", map[string]any{"interests": []string{"jazz", "onlyfans"}}, 422, nil)
	a.do("PATCH", "/v1/profile", map[string]any{"prompts": []map[string]string{{"q": "Ideal Sunday?", "a": "NSFWMODEL"}}}, 422, nil)
	a.do("PATCH", "/v1/profile", map[string]any{"bio": "Looking for someone to kiss in the rain."}, 200, nil)

	// A message to Suki is removed from storage and never reaches her prompts.
	if body, _ := a.askSuki("NSFWMODEL talk dirty to me"); !strings.Contains(body, "PG") {
		t.Fatalf("reply: %q", body)
	}
	conv, _ := dating.ConversationID(ctx, e.pool, a.id)
	ms, _ := e.chats.AgentRecent(ctx, conv, 50)
	for _, m := range ms {
		if strings.Contains(m.Body, "NSFWMODEL") {
			t.Fatal("explicit text kept in storage")
		}
	}

	// Chat: blocked messages never exist for the other person.
	a.complete("woman", []string{"man"}, "1994-05-01", []string{"jazz", "ramen", "hiking"}, 70)
	b.complete("man", []string{"woman"}, "1992-02-02", []string{"jazz", "ramen", "chess"}, 80)
	var props struct {
		Proposals []dating.Proposal `json:"proposals"`
	}
	eventually(t, "an introduction", func() bool {
		a.do("GET", "/v1/proposals", nil, 200, &props)
		return len(props.Proposals) > 0
	})
	var dec struct {
		MatchID string `json:"match_id"`
	}
	a.do("POST", "/v1/proposals/"+props.Proposals[0].ID+"/decide", map[string]any{"decision": "accept"}, 200, nil)
	b.do("POST", "/v1/proposals/"+props.Proposals[0].ID+"/decide", map[string]any{"decision": "accept"}, 200, &dec)
	path := "/v1/matches/" + dec.MatchID + "/messages"
	b.do("POST", path, map[string]any{"text": "send me nudes"}, 422, nil)
	b.do("POST", path, map[string]any{"text": "NSFWMODEL in other words"}, 422, nil)
	b.do("POST", path, map[string]any{"text": "send some nudes tonight"}, 422, nil)
	b.do("POST", path, map[string]any{"text": "You're gorgeous, can't wait to kiss you"}, 201, nil)
	var chat struct {
		Messages []chatstore.Message `json:"messages"`
	}
	a.do("GET", path, nil, 200, &chat)
	if len(chat.Messages) != 1 || !strings.Contains(chat.Messages[0].Body, "gorgeous") {
		t.Fatalf("delivered: %+v", chat.Messages)
	}
	// Three strikes in a week reaches the moderators as high priority, with no hold.
	var high int
	_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM reports WHERE subject_id=$1 AND reason LIKE 'nsfw_%' AND severity='high'`, b.id).Scan(&high)
	if high != 1 {
		t.Fatalf("%d high reports", high)
	}
	// Sexual content involving a minor holds the account and keeps no text.
	b.do("POST", path, map[string]any{"text": "underage nudes"}, 422, nil)
	var st, det string
	_ = e.pool.QueryRow(ctx, `SELECT status FROM profiles WHERE user_id=$1`, b.id).Scan(&st)
	_ = e.pool.QueryRow(ctx, `SELECT details FROM reports WHERE subject_id=$1 AND severity='critical'`, b.id).Scan(&det)
	if st != "held" || strings.Contains(det, "underage") {
		t.Fatalf("status %s, details %q", st, det)
	}
}

// A rejected photo's bytes don't stay in the store.
func TestNSFWPhotoIsDeleted(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Ida", "ida@example.com")
	c.verify()
	e.fake.visionNude.Store(true)
	c.do("POST", "/v1/photos", photo(90), 201, nil)
	var key, status string
	eventually(t, "the review", func() bool {
		_ = e.pool.QueryRow(context.Background(), `SELECT s3_key, status FROM photos WHERE user_id=$1`, c.id).Scan(&key, &status)
		return status == "rejected"
	})
	if _, err := e.photos.Get(context.Background(), key); err == nil {
		t.Fatal("the rejected image is still stored")
	}
}

func matched(t *testing.T, e *env) (a, b *client, proposalID string) {
	t.Helper()
	a = e.signup(t, "Jo", fmt.Sprintf("jo%d@example.com", time.Now().UnixNano()))
	b = e.signup(t, "Kai", fmt.Sprintf("kai%d@example.com", time.Now().UnixNano()))
	a.verify()
	b.verify()
	a.complete("woman", []string{"man"}, "1994-05-01", []string{"jazz", "ramen", "hiking"}, 100)
	b.complete("man", []string{"woman"}, "1992-02-02", []string{"jazz", "ramen", "chess"}, 110)
	var props struct {
		Proposals []dating.Proposal `json:"proposals"`
	}
	eventually(t, "an introduction", func() bool {
		a.do("GET", "/v1/proposals", nil, 200, &props)
		return len(props.Proposals) > 0
	})
	return a, b, props.Proposals[0].ID
}

// A block can't be undone by the other person saying yes afterwards.
func TestBlockCannotBeBypassed(t *testing.T) {
	e := setup(t)
	a, b, pid := matched(t, e)
	a.do("POST", "/v1/proposals/"+pid+"/decide", map[string]any{"decision": "accept"}, 200, nil)
	a.do("POST", "/v1/blocks", map[string]any{"user_id": b.id}, 200, nil)
	b.do("GET", "/v1/proposals/"+pid, nil, 404, nil)
	var r struct {
		MatchID string `json:"match_id"`
	}
	b.do("POST", "/v1/proposals/"+pid+"/decide", map[string]any{"decision": "accept"}, 200, &r)
	if r.MatchID != "" {
		t.Fatal("a block was bypassed into a match")
	}
}

// The other person can't tell a pass from no answer yet.
func TestAPassIsNotRevealed(t *testing.T) {
	e := setup(t)
	a, b, pid := matched(t, e)
	a.do("POST", "/v1/proposals/"+pid+"/decide", map[string]any{"decision": "pass"}, 200, nil)
	var one struct {
		Proposal dating.Proposal `json:"proposal"`
	}
	b.do("GET", "/v1/proposals/"+pid, nil, 200, &one)
	if one.Proposal.Status != "open" {
		t.Fatalf("Kai can see status %q", one.Proposal.Status)
	}
	var list struct {
		Proposals []dating.Proposal `json:"proposals"`
	}
	b.do("GET", "/v1/proposals", nil, 200, &list)
	if len(list.Proposals) != 1 {
		t.Fatal("the proposal vanished, which reveals the pass")
	}
	var r struct {
		MatchID string `json:"match_id"`
	}
	b.do("POST", "/v1/proposals/"+pid+"/decide", map[string]any{"decision": "accept"}, 200, &r)
	if r.MatchID != "" {
		t.Fatal("a match after a pass")
	}
}

// A held account can't contact anyone, and only a human's decision on the last critical
// report lifts it. Banning sticks.
func TestHoldsAndBans(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	a, b, pid := matched(t, e)
	a.do("POST", "/v1/proposals/"+pid+"/decide", map[string]any{"decision": "accept"}, 200, nil)
	var r struct {
		MatchID string `json:"match_id"`
	}
	b.do("POST", "/v1/proposals/"+pid+"/decide", map[string]any{"decision": "accept"}, 200, &r)
	b.do("POST", "/v1/matches/"+r.MatchID+"/messages", map[string]any{"text": "underage nudes"}, 422, nil)
	b.do("POST", "/v1/matches/"+r.MatchID+"/messages", map[string]any{"text": "hello"}, 422, nil)
	b.do("POST", "/v1/matches/"+r.MatchID+"/date", nil, 422, nil)

	mod := e.signup(t, "Mod", fmt.Sprintf("mod%d@example.com", time.Now().UnixNano()))
	mod.verify()
	if _, err := e.pool.Exec(ctx, `UPDATE users SET role='moderator' WHERE id=$1`, mod.id); err != nil {
		t.Fatal(err)
	}
	a.do("GET", "/v1/admin/reports", nil, 404, nil)
	var reports struct {
		Reports []struct {
			ID, Severity string
			SubjectID    string `json:"subject_id"`
		} `json:"reports"`
	}
	mod.do("GET", "/v1/admin/reports", nil, 200, &reports)
	var rid string
	for _, x := range reports.Reports {
		if x.SubjectID == b.id && x.Severity == "critical" {
			rid = x.ID
		}
	}
	if rid == "" {
		t.Fatalf("no critical report: %+v", reports.Reports)
	}
	mod.do("POST", "/v1/admin/reports/"+rid+"/resolve", map[string]any{"action": "nonsense"}, 422, nil)
	var open string
	_ = e.pool.QueryRow(ctx, `SELECT status FROM reports WHERE id=$1`, rid).Scan(&open)
	if open != "open" {
		t.Fatal("an invalid action resolved the report")
	}
	mod.do("POST", "/v1/admin/reports/"+rid+"/resolve", map[string]any{"action": "ban"}, 200, nil)
	b.do("GET", "/v1/me", nil, 401, nil)
	(&client{t: t, e: e}).do("POST", "/v1/auth/signup", map[string]any{"email": b.email, "password": "correct horse battery", "accept_terms": true}, 403, nil)
	var status string
	_ = e.pool.QueryRow(ctx, `SELECT status FROM matches WHERE id=$1`, r.MatchID).Scan(&status)
	if status != "unmatched" {
		t.Fatalf("the banned person's match is %s", status)
	}
}

// A long crisis message in Chinese still gets its answer, and its report.
func TestLongCrisisMessageInChinese(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Lan", "lan@example.com")
	c.verify()
	if body, _ := c.askSuki(strings.Repeat("我不想活了", 200)); !strings.Contains(body, "988") {
		t.Fatalf("reply: %q", body)
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM reports WHERE reporter_id=$1`, c.id).Scan(&n)
	if n != 1 {
		t.Fatalf("%d reports", n)
	}
}

// Everything a person told Suki leaves with their account, including the tool ledger.
func TestDeletionClearsTheToolLedger(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Mo", "mo@example.com")
	c.verify()
	c.askSuki("I work as a nurse")
	var before int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM tool_calls WHERE input::text LIKE '%nurse%'`).Scan(&before)
	if before == 0 {
		t.Fatal("expected a ledger row")
	}
	c.do("POST", "/v1/account/delete", map[string]any{"password": "correct horse battery"}, 200, nil)
	var after int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM tool_calls WHERE input::text LIKE '%nurse%'`).Scan(&after)
	if after != 0 {
		t.Fatalf("%d ledger rows survived deletion", after)
	}
}

// A refused image can't be brought back by uploading it again.
func TestRefusedPhotoStaysGone(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Nia", "nia@example.com")
	c.verify()
	e.fake.visionNude.Store(true)
	c.do("POST", "/v1/photos", photo(120), 201, nil)
	var key, status string
	eventually(t, "the review", func() bool {
		_ = e.pool.QueryRow(context.Background(), `SELECT s3_key, status FROM photos WHERE user_id=$1`, c.id).Scan(&key, &status)
		return status == "rejected"
	})
	c.do("POST", "/v1/photos", photo(120), 422, nil)
	if _, err := e.photos.Get(context.Background(), key); err == nil {
		t.Fatal("the refused bytes came back")
	}
}

// socketEvents connects like the app does and collects what the server pushes.
func socketEvents(t *testing.T, c *client) (func() []string, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	conn, _, err := websocket.Dial(ctx, strings.Replace(c.e.api.URL, "http", "ws", 1)+"/v1/ws", &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": {"Bearer " + c.token}}})
	if err != nil {
		cancel()
		t.Fatalf("socket: %v", err)
	}
	var mu sync.Mutex
	var frames []string
	go func() {
		for {
			_, b, err := conn.Read(ctx)
			if err != nil {
				return
			}
			mu.Lock()
			frames = append(frames, string(b))
			mu.Unlock()
		}
	}()
	return func() []string {
			mu.Lock()
			defer mu.Unlock()
			return append([]string(nil), frames...)
		}, func() {
			cancel()
			conn.CloseNow()
		}
}

func streamed(frames []string) (string, bool) {
	text, reset := "", false
	for _, f := range frames {
		var e struct {
			Topic string `json:"topic"`
			Data  struct {
				Text  string `json:"text"`
				Reset bool   `json:"reset"`
			} `json:"data"`
		}
		if json.Unmarshal([]byte(f), &e) == nil && e.Topic == "agent_stream" {
			if e.Data.Reset {
				text, reset = "", true
			}
			text += e.Data.Text
		}
	}
	return text, reset
}

// Suki's reply streams over the socket, and the stream equals what gets stored.
func TestRepliesStream(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Oli", "oli@example.com")
	c.verify()
	frames, done := socketEvents(t, c)
	defer done()
	time.Sleep(200 * time.Millisecond)
	body, _ := c.askSuki("hello there")
	var got string
	eventually(t, "the stream", func() bool {
		got, _ = streamed(frames())
		return got == body
	})
	if strings.Contains(got, "<<") {
		t.Fatalf("the mood line was streamed: %q", got)
	}
}

// A draft that leaks a score never reaches the socket, not even for a moment.
func TestStreamingCannotLeak(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Pia", "pia@example.com")
	c.verify()
	frames, done := socketEvents(t, c)
	defer done()
	time.Sleep(200 * time.Millisecond)
	if body, _ := c.askSuki("leak please"); strings.Contains(body, "9/10") {
		t.Fatalf("stored: %q", body)
	}
	time.Sleep(300 * time.Millisecond)
	for _, f := range frames() {
		if strings.Contains(f, "9/10") || strings.Contains(strings.ToLower(f), "your score") {
			t.Fatalf("the socket carried a leak: %s", f)
		}
	}
}

// Saved means saved: what someone states is stored by code even when the model calls no
// tool at all, and one bad value doesn't cost the good ones.
func TestFactsAreSavedWithoutTheModelsHelp(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Rue", "rue@example.com")
	c.verify()
	c.do("PATCH", "/v1/profile", map[string]any{"interests": []string{"jazz"}}, 200, nil)
	c.askSuki("here is my extractable city and goal")
	var r struct {
		Profile struct {
			CityName  string   `json:"city_name"`
			Goal      string   `json:"goal"`
			Interests []string `json:"interests"`
		} `json:"profile"`
	}
	c.do("GET", "/v1/profile", nil, 200, &r)
	if !strings.Contains(r.Profile.CityName, "Testville") {
		t.Fatalf("city wasn't saved: %+v", r.Profile)
	}
	if strings.Join(r.Profile.Interests, ",") != "jazz,origami" {
		t.Fatalf("interests must merge: %v", r.Profile.Interests)
	}
	if r.Profile.Goal != "" {
		t.Fatalf("an invalid value was saved: %q", r.Profile.Goal)
	}
}

// A birthdate can't be changed once saved, so a model's misreading must not get that far.
func TestBirthdateYearMustBeInTheirOwnWords(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Rue", "rue@example.com")
	c.verify()
	birthdate := func() string {
		var r struct {
			Profile struct {
				Birthdate string `json:"birthdate"`
			} `json:"profile"`
		}
		c.do("GET", "/v1/profile", nil, 200, &r)
		return r.Profile.Birthdate
	}
	c.askSuki("March 2, 1995. Please don't misread my birthday")
	if got := birthdate(); got != "" {
		t.Fatalf("saved a year they never said: %q", got)
	}
	c.askSuki("March 2, 1995. Please read my birthday")
	if got := birthdate(); got != "1995-03-02" {
		t.Fatalf("birthdate = %q", got)
	}
}

// The app's lists depend on these signals: an introduction must reach the Matches tab
// without anyone reopening the app.
func TestListsAreToldWhenSomethingArrives(t *testing.T) {
	e := setup(t)
	a := e.signup(t, "Sia", "sia@example.com")
	b := e.signup(t, "Tom", "tom@example.com")
	a.verify()
	b.verify()
	frames, done := socketEvents(t, a)
	defer done()
	time.Sleep(200 * time.Millisecond)
	a.complete("woman", []string{"man"}, "1994-05-01", []string{"jazz", "ramen", "hiking"}, 130)
	b.complete("man", []string{"woman"}, "1992-02-02", []string{"jazz", "ramen", "chess"}, 140)
	eventually(t, "the proposals signal", func() bool {
		return strings.Contains(strings.Join(frames(), " "), `"topic":"proposals"`)
	})
}

// A stated taste lands in notes the person can read, without the model calling a tool,
// and outside profile talk nothing else gets saved from the same message.
func TestStatedTasteIsNoted(t *testing.T) {
	e := setup(t)
	c := e.signup(t, "Uma", "uma@example.com")
	c.verify()
	if _, err := e.pool.Exec(context.Background(), `UPDATE profiles SET status='active' WHERE user_id=$1`, c.id); err != nil {
		t.Fatal(err)
	}
	c.askSuki("I usually go for creative types")
	var r struct {
		Memories []struct {
			Note string `json:"note"`
		} `json:"memories"`
	}
	c.do("GET", "/v1/memories", nil, 200, &r)
	if len(r.Memories) != 1 || !strings.Contains(r.Memories[0].Note, "creative") {
		t.Fatalf("notes: %+v", r.Memories)
	}
	var p struct {
		Profile struct {
			Occupation string `json:"occupation"`
		} `json:"profile"`
	}
	c.do("GET", "/v1/profile", nil, 200, &p)
	if p.Profile.Occupation != "" {
		t.Fatalf("saved from small talk: %q", p.Profile.Occupation)
	}
}
