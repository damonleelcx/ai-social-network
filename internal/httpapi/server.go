// Package httpapi is the mobile app's only door: JSON over HTTP, plus one socket
// that says what changed. It can't import internal/hidden, and archtest enforces
// that, so a response can't carry what this package can't read.
package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damonleelcx/ai-social-network/internal/agent"
	"github.com/damonleelcx/ai-social-network/internal/auth"
	"github.com/damonleelcx/ai-social-network/internal/billing"
	"github.com/damonleelcx/ai-social-network/internal/chatstore"
	"github.com/damonleelcx/ai-social-network/internal/dating"
	"github.com/damonleelcx/ai-social-network/internal/externalauth"
	"github.com/damonleelcx/ai-social-network/internal/i18n"
	"github.com/damonleelcx/ai-social-network/internal/photostore"
	"github.com/damonleelcx/ai-social-network/internal/places"
	"github.com/damonleelcx/ai-social-network/internal/profile"
	"github.com/damonleelcx/ai-social-network/internal/tts"
	"github.com/damonleelcx/ai-social-network/internal/workflows"
)

type Server struct {
	Pool   *pgxpool.Pool
	Auth   *auth.Service
	Chats  *chatstore.Store
	Photos photostore.Store
	Places *places.Client
	Dating *dating.Service
	Hub    *Hub
	// Billing is credits. Checkout and the webhook refuse when no provider is configured.
	Billing *billing.Service
	// WeChatLogin is sign-in from the mini program. Nil means it's off.
	WeChatLogin *externalauth.WeChat
	// GoogleLogin is sign-in from the Android app. Nil means it's off.
	GoogleLogin *externalauth.Google
	// Speech is nil when no voice is configured.
	Speech           *tts.Service
	SpeechDailyChars int
	limit            limiter
	mapCache         userMapCache
}

// ── Socket fan-out ──────────────────────────────────────────────────────────

// Hub delivers "something changed" to a person's connected devices. Workers run
// elsewhere, so events arrive through Postgres NOTIFY and every web pod hears them.
type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan string]bool
}

func NewHub() *Hub { return &Hub{subs: map[string]map[chan string]bool{}} }

func (h *Hub) add(user string) chan string {
	ch := make(chan string, 32)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[user] == nil {
		h.subs[user] = map[chan string]bool{}
	}
	h.subs[user][ch] = true
	return ch
}

func (h *Hub) remove(user string, ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subs[user], ch)
	if len(h.subs[user]) == 0 {
		delete(h.subs, user)
	}
}

// Publish takes a NOTIFY payload: "<user id>" or "<user id>|<topic>".
func (h *Hub) Publish(payload string) {
	user, topic, _ := strings.Cut(payload, "|")
	if topic == "" {
		topic = "agent"
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[user] {
		select {
		case ch <- topic:
		default: // a slow client refetches on its next event anyway
		}
	}
}

// ── Plumbing ────────────────────────────────────────────────────────────────

type ctxKey int

const (
	keyUser ctxKey = iota
	keySession
)

func userOf(r *http.Request) *auth.User { u, _ := r.Context().Value(keyUser).(*auth.User); return u }

func sessionOf(r *http.Request) string { s, _ := r.Context().Value(keySession).(string); return s }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, msg string) { writeJSON(w, status, map[string]string{"error": msg}) }

// handleErr shows messages written for people, and logs the rest.
func handleErr(w http.ResponseWriter, r *http.Request, err error) {
	var ue *dating.UserError
	var ve *profile.ValidationError
	switch {
	// 22P02 is Postgres refusing a malformed uuid: an id in a URL that can't exist.
	case errors.Is(err, dating.ErrNotFound), strings.Contains(err.Error(), "22P02"):
		fail(w, 404, "not found")
	case errors.As(err, &ue):
		fail(w, 422, ue.Msg)
	case errors.As(err, &ve):
		fail(w, 422, ve.Msg)
	case errors.Is(err, profile.ErrUnderage):
		fail(w, 403, err.Error())
	case errors.Is(err, auth.ErrInvalidCredentials):
		fail(w, 401, err.Error())
	case errors.Is(err, auth.ErrBanned):
		fail(w, 403, err.Error())
	case errors.Is(err, auth.ErrTooManyAttempts), errors.Is(err, auth.ErrTooManyCodes):
		fail(w, 429, err.Error())
	case auth.UserFacing(err):
		fail(w, 422, err.Error())
	default:
		slog.Error("request failed", "path", r.URL.Path, "err", err)
		fail(w, 500, "Something went wrong on our side. Please try again.")
	}
}

func read(w http.ResponseWriter, r *http.Request, into any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(into); err != nil && !errors.Is(err, io.EOF) {
		fail(w, 400, "that request didn't make sense")
		return false
	}
	return true
}

func clientIP(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		// The last hop is what our own proxy saw; the first is whatever the client sent.
		parts := strings.Split(f, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// limiter is a fixed-window counter per key. It's per pod, so the real ceiling is
// the limit times the pods, which is fine for abuse limits; budgets are in Postgres.
type limiter struct {
	mu   sync.Mutex
	seen map[string]*window
}

type window struct {
	start time.Time
	n     int
}

func (l *limiter) allow(key string, max int, per time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.seen == nil {
		l.seen = map[string]*window{}
	}
	if len(l.seen) > 100_000 {
		l.seen = map[string]*window{}
	}
	w := l.seen[key]
	now := time.Now()
	if w == nil || now.Sub(w.start) > per {
		l.seen[key] = &window{now, 1}
		return true
	}
	w.n++
	return w.n <= max
}

func (s *Server) limited(name string, max int, per time.Duration, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := name + ":" + clientIP(r)
		if u := userOf(r); u != nil {
			key = name + ":" + u.ID
		}
		if !s.limit.allow(key, max, per) {
			fail(w, 429, "Slow down a little and try again in a moment.")
			return
		}
		next(w, r)
	}
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if t, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(t)
	}
	return ""
}

// socketProtocol is what the server answers to a browser's WebSocket, which has to name
// a subprotocol it can get back before it will open.
const socketProtocol = "synagent"

// browserSocketAuth lets a browser, which can't set headers on a WebSocket, send its token as
// a "bearer.<token>" subprotocol. Not a query parameter: those end up in access logs.
func browserSocketAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			for _, p := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
				if t, ok := strings.CutPrefix(strings.TrimSpace(p), "bearer."); ok {
					r.Header.Set("Authorization", "Bearer "+t)
				}
			}
		}
		next(w, r)
	}
}

// signedIn requires a session. Being signed in is not enough for most things:
// see verified.
func (s *Server) signedIn(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, hash, err := s.Auth.SessionUser(r.Context(), bearer(r))
		if errors.Is(err, auth.ErrInvalidCredentials) {
			fail(w, 401, "please sign in")
			return
		}
		if err != nil {
			slog.Error("session lookup", "err", err)
			fail(w, 503, "We're having a moment. Please try again shortly.")
			return
		}
		ctx := context.WithValue(context.WithValue(r.Context(), keyUser, u), keySession, hash)
		next(w, r.WithContext(ctx))
	}
}

// verified requires a confirmed email, and keeps "last active" honest.
func (s *Server) verified(next http.HandlerFunc) http.HandlerFunc {
	return s.signedIn(func(w http.ResponseWriter, r *http.Request) {
		u := userOf(r)
		if !u.EmailVerified {
			fail(w, 403, "please verify your email first")
			return
		}
		_, _ = s.Pool.Exec(r.Context(), `UPDATE profiles SET last_active_at=now() WHERE user_id=$1 AND last_active_at < now()-interval '5 minutes'`, u.ID)
		next(w, r)
	})
}

func (s *Server) moderator(next http.HandlerFunc) http.HandlerFunc {
	return s.verified(func(w http.ResponseWriter, r *http.Request) {
		u := userOf(r)
		if !u.Admin && u.Role != "moderator" {
			fail(w, 404, "not found")
			return
		}
		next(w, r)
	})
}

// Handler is the whole API.
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	v := s.verified
	lim := s.limited

	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]bool{"ok": true}) })
	m.HandleFunc("GET /readyz", s.ready)
	m.Handle("GET /{$}", landing)
	m.Handle("GET /landing/", landing)
	m.Handle("GET /app", webapp)
	m.Handle("GET /app/", webapp)
	m.HandleFunc("GET /legal/terms", legalPage("terms"))
	m.HandleFunc("GET /legal/privacy", legalPage("privacy"))
	m.HandleFunc("GET /legal/contact", legalPage("contact"))
	m.HandleFunc("GET /legal/delete-account", legalPage("delete-account"))
	m.HandleFunc("GET /open", legalPage("open"))

	m.HandleFunc("GET /v1/public/user-map", lim("user-map", 60, time.Minute, s.userMap))

	// Accounts
	m.HandleFunc("POST /v1/auth/signup", lim("signup", 30, time.Hour, s.signup))
	m.HandleFunc("POST /v1/auth/signin", lim("signin", 120, time.Minute, s.signin))
	m.HandleFunc("POST /v1/auth/signout", s.signedIn(s.signout))
	m.HandleFunc("POST /v1/auth/verify", s.signedIn(lim("verify", 20, 10*time.Minute, s.verifyEmail)))
	m.HandleFunc("POST /v1/auth/resend", s.signedIn(lim("resend", 5, 10*time.Minute, s.resend)))
	m.HandleFunc("POST /v1/auth/google", lim("signin", 120, time.Minute, s.googleSignIn))
	m.HandleFunc("POST /v1/auth/google/delete", s.signedIn(lim("delete", 5, time.Hour, s.googleDelete)))
	m.HandleFunc("POST /v1/auth/wechat", lim("signin", 120, time.Minute, s.wechatSignIn))
	m.HandleFunc("POST /v1/auth/wechat/delete", s.signedIn(lim("delete", 5, time.Hour, s.wechatDelete)))
	m.HandleFunc("POST /v1/auth/forgot", lim("forgot", 60, 10*time.Minute, s.forgot))
	m.HandleFunc("POST /v1/auth/reset", lim("reset", 60, 10*time.Minute, s.reset))
	m.HandleFunc("POST /v1/auth/change-password", s.signedIn(lim("change", 10, time.Hour, s.changePassword)))
	m.HandleFunc("GET /v1/auth/sessions", s.signedIn(s.sessions))
	m.HandleFunc("DELETE /v1/auth/sessions/{id}", s.signedIn(s.revokeSession))
	m.HandleFunc("POST /v1/account/delete", s.signedIn(lim("delete", 5, time.Hour, s.deleteAccount)))
	m.HandleFunc("GET /v1/account/export", v(lim("export", 5, time.Hour, s.export)))
	m.HandleFunc("GET /v1/me", s.signedIn(s.me))

	// Profile and photos
	m.HandleFunc("GET /v1/profile", v(s.getProfile))
	m.HandleFunc("PATCH /v1/profile", v(s.patchProfile))
	m.HandleFunc("POST /v1/profile/city", v(lim("city", 20, time.Hour, s.setCity)))
	m.HandleFunc("POST /v1/profile/pause", v(s.pause))
	m.HandleFunc("POST /v1/profile/bio-draft/accept", v(s.acceptBio))
	m.HandleFunc("GET /v1/photos", v(s.listPhotos))
	m.HandleFunc("POST /v1/photos", v(lim("photo", 30, time.Hour, s.uploadPhoto)))
	m.HandleFunc("POST /v1/photos/{id}/primary", v(s.primaryPhoto))
	m.HandleFunc("DELETE /v1/photos/{id}", v(s.deletePhoto))
	m.HandleFunc("GET /v1/media/{id}", v(s.media))

	// Suki
	m.HandleFunc("GET /v1/agent/messages", v(s.agentMessages))
	m.HandleFunc("POST /v1/agent/messages", v(lim("agent", 30, time.Minute, s.agentSend)))
	m.HandleFunc("GET /v1/agent/speech", v(s.speechStatus))
	m.HandleFunc("POST /v1/agent/speech", v(lim("speech", 20, time.Minute, s.speak)))

	// Credits. The webhook signs itself, and the page is only somewhere to land after paying.
	m.HandleFunc("GET /v1/credits", v(s.creditStatus))
	m.HandleFunc("GET /v1/credits/usage", v(s.creditUsage))
	m.HandleFunc("GET /v1/referral", v(s.referralStatus))
	m.HandleFunc("POST /v1/credits/checkout", v(lim("checkout", 20, time.Hour, s.creditCheckout)))
	m.HandleFunc("POST /v1/billing/stripe", s.stripeWebhook)
	m.HandleFunc("POST /v1/credits/wechat/prepay", v(lim("checkout", 20, time.Hour, s.wechatPrepay)))
	m.HandleFunc("POST /v1/billing/wechat", s.wechatNotify)
	m.HandleFunc("POST /v1/credits/google/verify", v(lim("google-verify", 60, time.Hour, s.googleVerify)))
	m.HandleFunc("GET /credits/done", s.creditsDone)

	// Buddies
	m.HandleFunc("GET /v1/buddies", v(s.buddyBoard))
	m.HandleFunc("GET /v1/buddies/mine", v(s.buddyMine))
	m.HandleFunc("GET /v1/buddies/{id}", v(s.buddyGet))
	m.HandleFunc("POST /v1/buddies", v(lim("buddy_post", 10, time.Hour, s.buddyPost)))
	m.HandleFunc("POST /v1/buddies/{id}/offer", v(lim("buddy_offer", 30, time.Hour, s.buddyOffer)))
	m.HandleFunc("DELETE /v1/buddies/{id}/offer", v(s.buddyWithdraw))
	m.HandleFunc("POST /v1/buddies/{id}/pick", v(lim("buddy_pick", 30, time.Hour, s.buddyPick)))
	m.HandleFunc("POST /v1/buddies/{id}/close", v(s.buddyClose))

	// Introductions and matches
	m.HandleFunc("GET /v1/proposals", v(s.proposals))
	m.HandleFunc("GET /v1/proposals/{id}", v(s.proposal))
	m.HandleFunc("POST /v1/proposals/{id}/decide", v(s.decide))
	m.HandleFunc("GET /v1/matches", v(s.matches))
	m.HandleFunc("GET /v1/matches/{id}", v(s.match))
	m.HandleFunc("POST /v1/matches/{id}/unmatch", v(s.unmatch))
	m.HandleFunc("GET /v1/matches/{id}/messages", v(s.chatMessages))
	m.HandleFunc("POST /v1/matches/{id}/messages", v(lim("chat", 60, time.Minute, s.chatSend)))
	m.HandleFunc("POST /v1/matches/{id}/read", v(s.chatRead))
	m.HandleFunc("POST /v1/matches/{id}/typing", v(lim("typing", 60, time.Minute, s.chatTyping)))
	m.HandleFunc("GET /v1/matches/{id}/hints", v(s.hints))
	m.HandleFunc("POST /v1/matches/{id}/coach", v(lim("coach", 10, time.Hour, s.askCoach)))
	m.HandleFunc("POST /v1/hints/{id}/dismiss", v(s.dismissHint))

	// Dates
	m.HandleFunc("GET /v1/dates", v(s.dates))
	m.HandleFunc("POST /v1/matches/{id}/date", v(lim("date", 10, time.Hour, s.startDate)))
	m.HandleFunc("GET /v1/dates/{id}", v(s.date))
	m.HandleFunc("POST /v1/dates/{id}/vote", v(s.vote))
	m.HandleFunc("POST /v1/dates/{id}/time", v(s.proposeTime))
	m.HandleFunc("POST /v1/dates/{id}/confirm", v(s.confirmTime))
	m.HandleFunc("POST /v1/dates/{id}/cancel", v(s.cancelDate))
	m.HandleFunc("GET /v1/dates/{id}/share", v(s.shareDate))

	// Safety
	m.HandleFunc("GET /v1/blocks", v(s.blocks))
	m.HandleFunc("POST /v1/blocks", v(s.block))
	m.HandleFunc("DELETE /v1/blocks/{id}", v(s.unblock))
	m.HandleFunc("POST /v1/reports", v(lim("report", 20, time.Hour, s.report)))

	// Settings and memory
	m.HandleFunc("GET /v1/settings", s.signedIn(s.getSettings))
	m.HandleFunc("PATCH /v1/settings", s.signedIn(s.patchSettings))
	m.HandleFunc("GET /v1/memories", v(s.memories))
	m.HandleFunc("DELETE /v1/memories/{id}", v(s.deleteMemory))
	m.HandleFunc("POST /v1/memories/reset", v(s.resetMemories))

	// Notifications
	m.HandleFunc("GET /v1/notifications", v(s.notifications))
	m.HandleFunc("POST /v1/notifications/read", v(s.readNotifications))
	m.HandleFunc("POST /v1/push-tokens", v(s.pushToken))
	m.HandleFunc("GET /v1/ws", browserSocketAuth(s.verified(s.socket)))

	// Moderators: where the humans in the loop act. Agents have no route here.
	m.HandleFunc("GET /v1/admin/reports", s.moderator(s.adminReports))
	m.HandleFunc("POST /v1/admin/reports/{id}/resolve", s.moderator(s.adminResolve))
	m.HandleFunc("GET /v1/admin/photos", s.moderator(s.adminPhotos))
	m.HandleFunc("POST /v1/admin/photos/{id}/decide", s.moderator(s.adminPhotoDecide))
	m.HandleFunc("GET /v1/admin/metrics", s.moderator(s.adminMetrics))
	m.HandleFunc("POST /v1/admin/users/{id}/credits", s.moderator(s.adminGrantCredits))
	m.HandleFunc("GET /v1/admin/goals/{id}/timeline", s.moderator(s.adminTimeline))

	return logRequests(m)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		if r.URL.Path != "/healthz" && r.URL.Path != "/readyz" {
			slog.Info("http", "method", r.Method, "path", r.Pattern, "status", sw.status, "ms", time.Since(start).Milliseconds())
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(s int) { w.status = s; w.ResponseWriter.WriteHeader(s) }

// Unwrap lets the socket upgrade reach the real connection.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.Pool.Ping(ctx); err != nil {
		fail(w, 503, "postgres")
		return
	}
	if err := s.Chats.Ping(ctx); err != nil {
		fail(w, 503, "mongodb")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ── Accounts ────────────────────────────────────────────────────────────────

type authResp struct {
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	User      *auth.User `json:"user"`
}

func (s *Server) issue(w http.ResponseWriter, r *http.Request, status int, u *auth.User) {
	tok, exp, err := s.Auth.CreateSession(r.Context(), u.ID, r.UserAgent(), clientIP(r))
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, status, authResp{tok, exp, u})
}

// wechatOpenID says who a wx.login code belongs to. It has answered if false comes back.
func (s *Server) wechatOpenID(w http.ResponseWriter, r *http.Request, code string) (string, bool) {
	if s.WeChatLogin == nil {
		fail(w, 422, "WeChat sign-in isn't open yet.")
		return "", false
	}
	id, err := s.WeChatLogin.OpenID(r.Context(), code)
	if errors.Is(err, externalauth.ErrRejected) {
		fail(w, 422, "WeChat couldn't confirm who you are. Please try again.")
		return "", false
	}
	if err != nil {
		handleErr(w, r, err)
		return "", false
	}
	return id, true
}

// googleUser checks an ID token. It has answered if nil comes back.
func (s *Server) googleUser(w http.ResponseWriter, r *http.Request, token string) *externalauth.GoogleUser {
	if s.GoogleLogin == nil {
		fail(w, 422, "Google sign-in isn't open yet.")
		return nil
	}
	u, err := s.GoogleLogin.Verify(r.Context(), token, time.Now())
	if errors.Is(err, externalauth.ErrRejected) {
		fail(w, 422, "Google couldn't confirm who you are. Please try again.")
		return nil
	}
	if err != nil {
		handleErr(w, r, err)
		return nil
	}
	return u
}

func (s *Server) googleSignIn(w http.ResponseWriter, r *http.Request) {
	var b struct {
		IDToken     string `json:"id_token"`
		Language    string
		Ref         string
		AcceptTerms bool `json:"accept_terms"`
	}
	if !read(w, r, &b) {
		return
	}
	g := s.googleUser(w, r, b.IDToken)
	if g == nil {
		return
	}
	if b.Language == "" {
		b.Language = "en"
	}
	u, created, err := s.Auth.SignInExternal(r.Context(), auth.External{Provider: "google", Subject: g.Subject, Email: g.Email, Name: g.Name, Lang: b.Language, AcceptTerms: b.AcceptTerms})
	if err != nil {
		handleErr(w, r, err)
		return
	}
	status := 200
	if created {
		status = 201
		s.attributeReferral(r, u.ID, b.Ref)
		if g.Name != "" {
			if _, err := profile.Save(r.Context(), s.Pool, u.ID, map[string]any{"display_name": g.Name}); err != nil {
				slog.Warn("google signup name", "err", err)
			}
		}
	}
	s.issue(w, r, status, u)
}

func (s *Server) googleDelete(w http.ResponseWriter, r *http.Request) {
	var b struct {
		IDToken string `json:"id_token"`
	}
	if !read(w, r, &b) {
		return
	}
	g := s.googleUser(w, r, b.IDToken)
	if g == nil {
		return
	}
	if err := s.Auth.DeleteAccountExternal(r.Context(), userOf(r).ID, "google", g.Subject); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) wechatSignIn(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Code, Language, Ref string
		AcceptTerms         bool `json:"accept_terms"`
	}
	if !read(w, r, &b) {
		return
	}
	openid, ok := s.wechatOpenID(w, r, b.Code)
	if !ok {
		return
	}
	if b.Language == "" {
		b.Language = "zh"
	}
	u, created, err := s.Auth.SignInExternal(r.Context(), auth.External{Provider: "wechat", Subject: openid, Lang: b.Language, AcceptTerms: b.AcceptTerms})
	if err != nil {
		handleErr(w, r, err)
		return
	}
	status := 200
	if created {
		status = 201
		s.attributeReferral(r, u.ID, b.Ref)
	}
	s.issue(w, r, status, u)
}

func (s *Server) wechatDelete(w http.ResponseWriter, r *http.Request) {
	var b struct{ Code string }
	if !read(w, r, &b) {
		return
	}
	openid, ok := s.wechatOpenID(w, r, b.Code)
	if !ok {
		return
	}
	if err := s.Auth.DeleteAccountExternal(r.Context(), userOf(r).ID, "wechat", openid); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Email, Password, Name, Language, Ref string
		AcceptTerms                          bool `json:"accept_terms"`
	}
	if !read(w, r, &b) {
		return
	}
	if b.Language == "" {
		b.Language = "en"
	}
	u, err := s.Auth.SignUp(r.Context(), b.Email, b.Password, b.Name, b.Language, b.AcceptTerms)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	if b.Name != "" {
		if _, err := profile.Save(r.Context(), s.Pool, u.ID, map[string]any{"display_name": b.Name}); err != nil {
			slog.Warn("signup name", "err", err)
		}
	}
	s.attributeReferral(r, u.ID, b.Ref)
	s.issue(w, r, 201, u)
}

func (s *Server) signin(w http.ResponseWriter, r *http.Request) {
	var b struct{ Email, Password string }
	if !read(w, r, &b) {
		return
	}
	u, err := s.Auth.SignIn(r.Context(), b.Email, b.Password)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	s.issue(w, r, 200, u)
}

func (s *Server) signout(w http.ResponseWriter, r *http.Request) {
	u := userOf(r)
	_, _ = s.Pool.Exec(r.Context(), `DELETE FROM push_tokens WHERE user_id=$1`, u.ID)
	if err := s.Auth.RevokeSession(r.Context(), u.ID, sessionOf(r)); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var b struct{ Code string }
	if !read(w, r, &b) {
		return
	}
	u, err := s.Auth.VerifyEmail(r.Context(), userOf(r).ID, b.Code)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	if err := s.Dating.PostFromSuki(r.Context(), u.ID, "Hi, I'm Suki! I'm an AI matchmaker, and I'm going to help you meet people you actually click with: someone to date, or just company for a movie or a trip. No endless swiping, just good introductions. Let's get to know each other. What should I call you?", "playful", nil, "welcome"); err != nil {
		slog.Warn("welcome", "err", err)
	}
	writeJSON(w, 200, map[string]any{"user": u})
}

func (s *Server) lang(r *http.Request, userID string) string {
	l, _ := dating.GetSettings(r.Context(), s.Pool, userID)["language"].(string)
	return l
}

func (s *Server) resend(w http.ResponseWriter, r *http.Request) {
	u := userOf(r)
	if err := s.Auth.SendVerification(r.Context(), u.ID, s.lang(r, u.ID)); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) forgot(w http.ResponseWriter, r *http.Request) {
	var b struct{ Email string }
	if !read(w, r, &b) {
		return
	}
	if err := s.Auth.ForgotPassword(r.Context(), b.Email, "en"); err != nil {
		slog.Error("forgot password", "err", err) // same answer either way
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) reset(w http.ResponseWriter, r *http.Request) {
	var b struct{ Email, Code, Password string }
	if !read(w, r, &b) {
		return
	}
	if err := s.Auth.ResetPassword(r.Context(), b.Email, b.Code, b.Password); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var b struct{ Current, Next string }
	if !read(w, r, &b) {
		return
	}
	if err := s.Auth.ChangePassword(r.Context(), userOf(r).ID, b.Current, b.Next, sessionOf(r)); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) sessions(w http.ResponseWriter, r *http.Request) {
	l, err := s.Auth.Sessions(r.Context(), userOf(r).ID, sessionOf(r))
	if err != nil {
		handleErr(w, r, err)
		return
	}
	if l == nil {
		l = []auth.Session{}
	}
	writeJSON(w, 200, map[string]any{"sessions": l})
}

func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request) {
	if err := s.Auth.RevokeSessionPrefix(r.Context(), userOf(r).ID, r.PathValue("id")); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	var b struct{ Password string }
	if !read(w, r, &b) {
		return
	}
	if err := s.Auth.DeleteAccount(r.Context(), userOf(r).ID, b.Password); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// BeforeDelete removes what the Postgres cascade can't reach: photo bytes in S3
// and chat threads in MongoDB. Both are done before any row goes, so a failure
// leaves the account intact to retry. Threads belonging to this person's matches
// go for both people: the conversation doesn't survive the match.
func (s *Server) BeforeDelete(ctx context.Context, userID string) error {
	photos, err := profile.Photos(ctx, s.Pool, userID)
	if err != nil {
		return err
	}
	for _, p := range photos {
		if err := s.Photos.Delete(ctx, p.S3Key); err != nil {
			return err
		}
	}
	convs, err := ids(ctx, s.Pool, `SELECT id FROM conversations WHERE user_id=$1`, userID)
	if err != nil {
		return err
	}
	matches, err := ids(ctx, s.Pool, `SELECT id FROM matches WHERE user_a=$1 OR user_b=$1`, userID)
	if err != nil {
		return err
	}
	return s.Chats.DeleteThreads(ctx, convs, matches)
}

func ids(ctx context.Context, pool *pgxpool.Pool, q string, a ...any) ([]string, error) {
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

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u := userOf(r)
	out := map[string]any{"user": u}
	if u.EmailVerified {
		p, err := profile.Get(r.Context(), s.Pool, u.ID)
		if err != nil {
			handleErr(w, r, err)
			return
		}
		m, err := profile.Missing(r.Context(), s.Pool, p)
		if err != nil {
			handleErr(w, r, err)
			return
		}
		out["profile"] = p
		out["missing"] = m
	}
	writeJSON(w, 200, out)
}

// ── Profile and photos ──────────────────────────────────────────────────────

func (s *Server) getProfile(w http.ResponseWriter, r *http.Request) {
	u := userOf(r)
	p, err := profile.Get(r.Context(), s.Pool, u.ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	m, err := profile.Missing(r.Context(), s.Pool, p)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"profile": p, "missing": m})
}

func (s *Server) patchProfile(w http.ResponseWriter, r *http.Request) {
	var patch map[string]any
	if !read(w, r, &patch) {
		return
	}
	u := userOf(r)
	p, err := profile.Save(r.Context(), s.Pool, u.ID, patch)
	if errors.Is(err, profile.ErrUnderage) {
		_, _ = s.Dating.FileReport(r.Context(), dating.Report{SubjectID: u.ID, Source: "onboarding", Reason: "underage_birthdate", Severity: "critical", Key: "underage:" + u.ID})
	}
	if err != nil {
		handleErr(w, r, err)
		return
	}
	if err := workflows.AfterProfileChange(r.Context(), u.ID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"profile": p})
}

func (s *Server) setCity(w http.ResponseWriter, r *http.Request) {
	var b struct{ Query string }
	if !read(w, r, &b) {
		return
	}
	u := userOf(r)
	c, err := s.Places.ResolveCity(r.Context(), b.Query)
	if errors.Is(err, places.ErrNoCity) {
		fail(w, 422, "I couldn't find that city. Try adding the country.")
		return
	}
	if err != nil {
		handleErr(w, r, err)
		return
	}
	if err := profile.SetCity(r.Context(), s.Pool, u.ID, c.Key, c.Name, c.Lat, c.Lon); err != nil {
		handleErr(w, r, err)
		return
	}
	if err := workflows.AfterProfileChange(r.Context(), u.ID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"city_name": c.Name})
}

func (s *Server) pause(w http.ResponseWriter, r *http.Request) {
	var b struct{ Paused bool }
	if !read(w, r, &b) {
		return
	}
	if err := profile.SetPaused(r.Context(), s.Pool, userOf(r).ID, b.Paused); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"paused": b.Paused})
}

// acceptBio is the tap that turns Suki's draft into the person's bio.
func (s *Server) acceptBio(w http.ResponseWriter, r *http.Request) {
	u := userOf(r)
	p, err := profile.Get(r.Context(), s.Pool, u.ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	if p.BioDraft == "" {
		fail(w, 422, "there's no draft to accept")
		return
	}
	if _, err := profile.Save(r.Context(), s.Pool, u.ID, map[string]any{"bio": p.BioDraft, "bio_draft": ""}); err != nil {
		handleErr(w, r, err)
		return
	}
	if err := workflows.AfterProfileChange(r.Context(), u.ID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) listPhotos(w http.ResponseWriter, r *http.Request) {
	l, err := profile.Photos(r.Context(), s.Pool, userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"photos": l})
}

const maxPhotos = 6

// uploadPhoto takes raw image bytes. The stored image is always our own re-encode,
// so EXIF (a phone's GPS tag) is gone before anything is written.
func (s *Server) uploadPhoto(w http.ResponseWriter, r *http.Request) {
	u := userOf(r)
	var count int
	if err := s.Pool.QueryRow(r.Context(), `SELECT count(*) FROM photos WHERE user_id=$1`, u.ID).Scan(&count); err != nil {
		handleErr(w, r, err)
		return
	}
	if count >= maxPhotos {
		fail(w, 422, "You can have up to 6 photos. Remove one first.")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, photostore.MaxUpload))
	if err != nil {
		fail(w, 413, "That photo is too large (12 MB at most).")
		return
	}
	out, width, height, err := photostore.Normalize(raw)
	if err != nil {
		fail(w, 422, err.Error())
		return
	}
	sum := sha256.Sum256(out)
	hash := hex.EncodeToString(sum[:])
	// A refused image stays refused. Without this, uploading it again would put the
	// deleted bytes back in storage.
	var prior, priorReason string
	err = s.Pool.QueryRow(r.Context(), `SELECT status, reason FROM photos WHERE user_id=$1 AND sha256=$2`, u.ID, hash).Scan(&prior, &priorReason)
	if err == nil && prior == "rejected" {
		fail(w, 422, "That photo was already declined. "+priorReason)
		return
	}
	var id string
	key := "photos/" + u.ID + "/" + hash + ".jpg"
	// Write the bytes first: a row must never point at nothing. If the insert then
	// fails, a retry overwrites the same key.
	if err := s.Photos.Put(r.Context(), key, out, "image/jpeg"); err != nil {
		handleErr(w, r, err)
		return
	}
	err = s.Pool.QueryRow(r.Context(), `INSERT INTO photos (user_id, s3_key, mime, width, height, sha256) VALUES ($1,$2,'image/jpeg',$3,$4,$5)
		ON CONFLICT (user_id, sha256) DO UPDATE SET sha256=EXCLUDED.sha256 RETURNING id`, u.ID, key, width, height, hash).Scan(&id)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	if err := workflows.StartPhotoReview(r.Context(), u.ID, id); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 201, map[string]string{"id": id, "status": "pending"})
}

func (s *Server) primaryPhoto(w http.ResponseWriter, r *http.Request) {
	if err := profile.SetPrimary(r.Context(), s.Pool, userOf(r).ID, r.PathValue("id")); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) deletePhoto(w http.ResponseWriter, r *http.Request) {
	key, err := profile.RemovePhoto(r.Context(), s.Pool, userOf(r).ID, r.PathValue("id"))
	if err != nil {
		handleErr(w, r, err)
		return
	}
	if err := s.Photos.Delete(r.Context(), key); err != nil {
		slog.Error("photo delete", "key", key, "err", err)
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// media streams a photo after checking this person may see it. Others only ever
// get approved photos; the owner also sees pending ones.
func (s *Server) media(w http.ResponseWriter, r *http.Request) {
	u := userOf(r)
	var owner, key, status string
	err := s.Pool.QueryRow(r.Context(), `SELECT user_id, s3_key, status FROM photos WHERE id=$1`, r.PathValue("id")).Scan(&owner, &key, &status)
	if err != nil {
		fail(w, 404, "not found")
		return
	}
	ok, err := s.Dating.CanSeePhoto(r.Context(), u.ID, owner)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	reviewing := status == "review" && (u.Admin || u.Role == "moderator")
	if !reviewing && (!ok || (owner != u.ID && status != "approved")) {
		fail(w, 404, "not found")
		return
	}
	data, err := s.Photos.Get(r.Context(), key)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(data)
}

// ── Suki ────────────────────────────────────────────────────────────────────

const seqOverlap = 5

func afterParam(r *http.Request) int64 {
	n, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	return n
}

func (s *Server) agentMessages(w http.ResponseWriter, r *http.Request) {
	conv, err := dating.ConversationID(r.Context(), s.Pool, userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	var l []chatstore.Message
	// A little overlap: seq is taken before the write, so a concurrent writer can land
	// just behind the newest message. Clients merge by seq.
	if after := afterParam(r); after > 0 {
		l, err = s.Chats.AgentAfter(r.Context(), conv, max(after-seqOverlap, 0), 200)
	} else {
		l, err = s.Chats.AgentRecent(r.Context(), conv, 100)
	}
	if err != nil {
		handleErr(w, r, err)
		return
	}
	var pending int
	_ = s.Pool.QueryRow(r.Context(), `SELECT count(*) FROM agent_turns WHERE conversation_id=$1 AND status IN ('ready','leased')`, conv).Scan(&pending)
	writeJSON(w, 200, map[string]any{"messages": l, "thinking": pending > 0})
}

func (s *Server) agentSend(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Text     string `json:"text"`
		ClientID string `json:"client_id"`
	}
	if !read(w, r, &b) {
		return
	}
	b.Text = strings.TrimSpace(b.Text)
	if b.Text == "" || len(b.Text) > 4000 {
		fail(w, 422, "write something, up to about 4000 characters")
		return
	}
	m, err := agent.Enqueue(r.Context(), s.Pool, s.Chats, userOf(r).ID, b.Text, b.ClientID, nil)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 202, map[string]any{"message": m})
}

// ── Credits ──

func (s *Server) creditStatus(w http.ResponseWriter, r *http.Request) {
	st, err := s.Billing.Status(r.Context(), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, st)
}

// attributeReferral never fails the sign-up it's part of.
func (s *Server) attributeReferral(r *http.Request, userID, code string) {
	if err := s.Billing.Attribute(r.Context(), userID, code); err != nil {
		slog.Warn("referral", "err", err)
	}
}

func (s *Server) referralStatus(w http.ResponseWriter, r *http.Request) {
	st, err := s.Billing.ReferralStatus(r.Context(), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, st)
}

func (s *Server) creditUsage(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Billing.Usage(r.Context(), userOf(r).ID, 50)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"usage": rows})
}

func (s *Server) creditCheckout(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Package string `json:"package"`
	}
	if !read(w, r, &b) {
		return
	}
	url, err := s.Billing.Checkout(r.Context(), userOf(r).ID, b.Package)
	var ue *billing.UserError
	if errors.As(err, &ue) {
		fail(w, 422, ue.Msg)
		return
	}
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"url": url})
}

// stripeWebhook answers 400 to what isn't really from Stripe, and 500 to what it should send
// again. Delivering an event twice adds credits once.
func (s *Server) stripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		fail(w, 400, "too large")
		return
	}
	paid, err := s.Billing.HandleWebhook(r.Context(), payload, r.Header.Get("Stripe-Signature"), time.Now())
	if errors.Is(err, billing.ErrBadSignature) {
		fail(w, 400, "bad signature")
		return
	}
	if err != nil {
		slog.Error("stripe webhook", "err", err)
		fail(w, 500, "try again")
		return
	}
	s.announcePaid(r, paid)
	writeJSON(w, 200, map[string]any{"received": true})
}

func (s *Server) announcePaid(r *http.Request, paid *billing.Paid) {
	if paid == nil {
		return
	}
	ctx := r.Context()
	lang := s.Dating.Lang(ctx, paid.UserID)
	body := i18n.T(lang, "Thanks! I added {credits} credits, so you have {balance} now.", "credits", strconv.Itoa(paid.Credits), "balance", strconv.Itoa(paid.Balance))
	if err := s.Dating.PostFromSuki(ctx, paid.UserID, body, "happy", nil, "credits:"+strconv.Itoa(paid.Balance)+":"+strconv.Itoa(paid.Credits)); err != nil {
		slog.Error("credits announce", "err", err)
	}
	s.Dating.Touch(ctx, paid.UserID, "credits")
}

func (s *Server) wechatPrepay(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Package string `json:"package"`
		Code    string `json:"code"`
	}
	if !read(w, r, &b) {
		return
	}
	p, err := s.Billing.WeChatPrepay(r.Context(), userOf(r).ID, b.Package, b.Code)
	var ue *billing.UserError
	if errors.As(err, &ue) {
		fail(w, 422, ue.Msg)
		return
	}
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, p)
}

func (s *Server) googleVerify(w http.ResponseWriter, r *http.Request) {
	var b struct {
		ProductID     string `json:"product_id"`
		PurchaseToken string `json:"purchase_token"`
	}
	if !read(w, r, &b) {
		return
	}
	res, err := s.Billing.VerifyGooglePurchase(r.Context(), userOf(r).ID, b.ProductID, b.PurchaseToken)
	var ue *billing.UserError
	if errors.As(err, &ue) {
		fail(w, 422, ue.Msg)
		return
	}
	if err != nil {
		handleErr(w, r, err)
		return
	}
	s.announcePaid(r, res.Paid)
	writeJSON(w, 200, map[string]any{"pending": res.Pending})
}

// wechatNotify answers anything but 2xx to have WeChat send it again.
func (s *Server) wechatNotify(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		fail(w, 400, "too large")
		return
	}
	paid, err := s.Billing.HandleWeChatNotify(r.Context(), payload, r.Header, time.Now())
	if errors.Is(err, billing.ErrBadSignature) {
		fail(w, 400, "bad signature")
		return
	}
	if err != nil {
		slog.Error("wechat notify", "err", err)
		fail(w, 500, "try again")
		return
	}
	s.announcePaid(r, paid)
	w.WriteHeader(http.StatusNoContent)
}

// adminGrantCredits is for support. The key makes a repeated request harmless.
func (s *Server) adminGrantCredits(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Credits int    `json:"credits"`
		Key     string `json:"key"`
	}
	if !read(w, r, &b) {
		return
	}
	if b.Credits <= 0 || b.Credits > 10000 || b.Key == "" {
		fail(w, 422, "credits must be 1 to 10000, with a key")
		return
	}
	if err := s.Billing.Grant(r.Context(), r.PathValue("id"), b.Credits, b.Key); err != nil {
		handleErr(w, r, err)
		return
	}
	s.Dating.Touch(r.Context(), r.PathValue("id"), "credits")
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) creditsDone(w http.ResponseWriter, r *http.Request) {
	line := "Thanks! Your credits will show up in a moment. You can go back to SynAgent now."
	if r.URL.Query().Get("cancelled") != "" {
		line = "Nothing was charged. You can go back to SynAgent now."
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	_, _ = io.WriteString(w, `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>SynAgent</title></head>`+
		`<body style="font:18px/1.5 system-ui;max-width:32em;margin:20vh auto;padding:0 16px;text-align:center"><p>`+line+`</p><p><a href="/app/suki">Open SynAgent</a></p></body></html>`)
}

// ── Buddies ──

func (s *Server) buddyBoard(w http.ResponseWriter, r *http.Request) {
	list, missing, err := s.Dating.BuddyBoard(r.Context(), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"requests": list, "missing": missing})
}

func (s *Server) buddyMine(w http.ResponseWriter, r *http.Request) {
	list, err := s.Dating.MyBuddyRequests(r.Context(), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"requests": list})
}

func (s *Server) buddyGet(w http.ResponseWriter, r *http.Request) {
	b, err := s.Dating.BuddyView(r.Context(), r.PathValue("id"), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"request": b})
}

func (s *Server) buddyPost(w http.ResponseWriter, r *http.Request) {
	var in dating.BuddyInput
	if !read(w, r, &in) {
		return
	}
	b, err := s.Dating.PostBuddy(r.Context(), userOf(r).ID, in)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 201, map[string]any{"request": b})
}

func (s *Server) buddyOffer(w http.ResponseWriter, r *http.Request) {
	if err := s.Dating.OfferBuddy(r.Context(), r.PathValue("id"), userOf(r).ID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) buddyWithdraw(w http.ResponseWriter, r *http.Request) {
	if err := s.Dating.WithdrawBuddyOffer(r.Context(), r.PathValue("id"), userOf(r).ID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) buddyPick(w http.ResponseWriter, r *http.Request) {
	var b struct {
		UserID string `json:"user_id"`
	}
	if !read(w, r, &b) {
		return
	}
	id, err := s.Dating.PickBuddy(r.Context(), r.PathValue("id"), userOf(r).ID, b.UserID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"match_id": id})
}

func (s *Server) buddyClose(w http.ResponseWriter, r *http.Request) {
	if err := s.Dating.CloseBuddy(r.Context(), r.PathValue("id"), userOf(r).ID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// ── Introductions and matches ───────────────────────────────────────────────

func (s *Server) proposals(w http.ResponseWriter, r *http.Request) {
	l, err := s.Dating.OpenProposals(r.Context(), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"proposals": l})
}

func (s *Server) proposal(w http.ResponseWriter, r *http.Request) {
	p, err := s.Dating.ProposalFor(r.Context(), r.PathValue("id"), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"proposal": p})
}

func (s *Server) decide(w http.ResponseWriter, r *http.Request) {
	var b struct{ Decision string }
	if !read(w, r, &b) {
		return
	}
	id, err := s.Dating.Decide(r.Context(), r.PathValue("id"), userOf(r).ID, b.Decision)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"match_id": id})
}

func (s *Server) matches(w http.ResponseWriter, r *http.Request) {
	l, err := s.Dating.Matches(r.Context(), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"matches": l})
}

func (s *Server) match(w http.ResponseWriter, r *http.Request) {
	m, err := s.Dating.MatchFor(r.Context(), r.PathValue("id"), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"match": m})
}

func (s *Server) unmatch(w http.ResponseWriter, r *http.Request) {
	if err := s.Dating.Unmatch(r.Context(), r.PathValue("id"), userOf(r).ID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) chatMessages(w http.ResponseWriter, r *http.Request) {
	after := afterParam(r)
	if after > 0 {
		after = max(after-seqOverlap, 1)
	}
	l, err := s.Dating.Messages(r.Context(), r.PathValue("id"), userOf(r).ID, after, 200)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"messages": l})
}

func (s *Server) chatSend(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Text     string `json:"text"`
		ClientID string `json:"client_id"`
	}
	if !read(w, r, &b) {
		return
	}
	m, err := s.Dating.SendMessage(r.Context(), r.PathValue("id"), userOf(r).ID, b.Text, b.ClientID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 201, map[string]any{"message": m})
}

func (s *Server) chatRead(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Seq int64 `json:"seq"`
	}
	if !read(w, r, &b) {
		return
	}
	if err := s.Dating.MarkRead(r.Context(), r.PathValue("id"), userOf(r).ID, b.Seq); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) chatTyping(w http.ResponseWriter, r *http.Request) {
	if err := s.Dating.Typing(r.Context(), r.PathValue("id"), userOf(r).ID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) hints(w http.ResponseWriter, r *http.Request) {
	l, err := s.Dating.Hints(r.Context(), r.PathValue("id"), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"hints": l})
}

// askCoach is the "ask Suki" button in a chat. The answer lands as a hint.
func (s *Server) askCoach(w http.ResponseWriter, r *http.Request) {
	u := userOf(r)
	id := r.PathValue("id")
	if _, err := s.Dating.MatchFor(r.Context(), id, u.ID); err != nil {
		handleErr(w, r, err)
		return
	}
	if err := workflows.StartCoach(r.Context(), id, u.ID, "asked_for_help"); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 202, map[string]bool{"ok": true})
}

func (s *Server) dismissHint(w http.ResponseWriter, r *http.Request) {
	if err := s.Dating.DismissHint(r.Context(), r.PathValue("id"), userOf(r).ID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ── Dates ───────────────────────────────────────────────────────────────────

func (s *Server) dates(w http.ResponseWriter, r *http.Request) {
	l, err := s.Dating.Dates(r.Context(), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"dates": l})
}

func (s *Server) startDate(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Constraints []string `json:"constraints"`
	}
	if !read(w, r, &b) {
		return
	}
	id, err := s.Dating.StartDate(r.Context(), r.PathValue("id"), userOf(r).ID, b.Constraints)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 201, map[string]string{"date_id": id})
}

func (s *Server) date(w http.ResponseWriter, r *http.Request) {
	d, err := s.Dating.DateFor(r.Context(), r.PathValue("id"), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"date": d})
}

func (s *Server) vote(w http.ResponseWriter, r *http.Request) {
	var b struct {
		VenueID string `json:"venue_id"`
	}
	if !read(w, r, &b) {
		return
	}
	if err := s.Dating.Vote(r.Context(), r.PathValue("id"), userOf(r).ID, b.VenueID); err != nil {
		handleErr(w, r, err)
		return
	}
	s.date(w, r)
}

func (s *Server) proposeTime(w http.ResponseWriter, r *http.Request) {
	var b struct {
		At time.Time `json:"at"`
	}
	if !read(w, r, &b) {
		return
	}
	if err := s.Dating.ProposeTime(r.Context(), r.PathValue("id"), userOf(r).ID, b.At); err != nil {
		handleErr(w, r, err)
		return
	}
	s.date(w, r)
}

func (s *Server) confirmTime(w http.ResponseWriter, r *http.Request) {
	if err := s.Dating.ConfirmTime(r.Context(), r.PathValue("id"), userOf(r).ID); err != nil {
		handleErr(w, r, err)
		return
	}
	s.date(w, r)
}

func (s *Server) cancelDate(w http.ResponseWriter, r *http.Request) {
	var b struct{ Reason string }
	if !read(w, r, &b) {
		return
	}
	if err := s.Dating.CancelDate(r.Context(), r.PathValue("id"), userOf(r).ID, b.Reason); err != nil {
		handleErr(w, r, err)
		return
	}
	s.date(w, r)
}

func (s *Server) shareDate(w http.ResponseWriter, r *http.Request) {
	text, err := s.Dating.ShareText(r.Context(), r.PathValue("id"), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]string{"text": text})
}

// ── Safety ──────────────────────────────────────────────────────────────────

func (s *Server) blocks(w http.ResponseWriter, r *http.Request) {
	l, err := s.Dating.Blocked(r.Context(), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"blocked": l})
}

func (s *Server) block(w http.ResponseWriter, r *http.Request) {
	var b struct {
		UserID string `json:"user_id"`
	}
	if !read(w, r, &b) {
		return
	}
	if err := s.Dating.Block(r.Context(), userOf(r).ID, b.UserID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) unblock(w http.ResponseWriter, r *http.Request) {
	if err := s.Dating.Unblock(r.Context(), userOf(r).ID, r.PathValue("id")); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	var b struct {
		UserID     string `json:"user_id"`
		MatchID    string `json:"match_id"`
		MessageSeq int64  `json:"message_seq"`
		Reason     string `json:"reason"`
		Details    string `json:"details"`
	}
	if !read(w, r, &b) {
		return
	}
	if b.Reason == "" || len(b.Details) > 2000 {
		fail(w, 422, "tell us what happened")
		return
	}
	u := userOf(r)
	if ok, err := s.Dating.CanSeePhoto(r.Context(), u.ID, b.UserID); err != nil || !ok {
		// Blocked first is common, and a report must still go through.
		var known bool
		_ = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM proposals WHERE (user_a=$1 AND user_b=$2) OR (user_a=$2 AND user_b=$1))`, u.ID, b.UserID).Scan(&known)
		if !known {
			fail(w, 404, "not found")
			return
		}
	}
	if _, err := s.Dating.FileReport(r.Context(), dating.Report{ReporterID: u.ID, SubjectID: b.UserID, MatchID: b.MatchID, MessageSeq: b.MessageSeq, Source: "user", Reason: b.Reason, Details: b.Details}); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 201, map[string]bool{"ok": true})
}

// ── Settings and memory ─────────────────────────────────────────────────────

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"settings": dating.GetSettings(r.Context(), s.Pool, userOf(r).ID)})
}

func (s *Server) patchSettings(w http.ResponseWriter, r *http.Request) {
	var patch map[string]any
	if !read(w, r, &patch) {
		return
	}
	st, err := dating.PatchSettings(r.Context(), s.Pool, userOf(r).ID, patch)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"settings": st})
}

func (s *Server) memories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT id, content, created_at FROM memories WHERE user_id=$1 ORDER BY created_at DESC LIMIT 200`, userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, c string
		var at time.Time
		if err := rows.Scan(&id, &c, &at); err != nil {
			handleErr(w, r, err)
			return
		}
		out = append(out, map[string]any{"id": id, "note": c, "created_at": at})
	}
	writeJSON(w, 200, map[string]any{"memories": out})
}

func (s *Server) deleteMemory(w http.ResponseWriter, r *http.Request) {
	if _, err := s.Pool.Exec(r.Context(), `DELETE FROM memories WHERE id=$1 AND user_id=$2`, r.PathValue("id"), userOf(r).ID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) resetMemories(w http.ResponseWriter, r *http.Request) {
	if err := workflows.ResetTaste(r.Context(), userOf(r).ID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ── Notifications and the socket ────────────────────────────────────────────

func (s *Server) notifications(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT id, kind, title, body, data, read_at, created_at FROM notifications WHERE user_id=$1 ORDER BY id DESC LIMIT 100`, userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var kind, title, body string
		var data []byte
		var readAt *time.Time
		var at time.Time
		if err := rows.Scan(&id, &kind, &title, &body, &data, &readAt, &at); err != nil {
			handleErr(w, r, err)
			return
		}
		out = append(out, map[string]any{"id": id, "kind": kind, "title": title, "body": body, "data": json.RawMessage(data), "read": readAt != nil, "created_at": at})
	}
	writeJSON(w, 200, map[string]any{"notifications": out})
}

func (s *Server) readNotifications(w http.ResponseWriter, r *http.Request) {
	if _, err := s.Pool.Exec(r.Context(), `UPDATE notifications SET read_at=now() WHERE user_id=$1 AND read_at IS NULL`, userOf(r).ID); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) pushToken(w http.ResponseWriter, r *http.Request) {
	var b struct{ Token, Platform string }
	if !read(w, r, &b) {
		return
	}
	if !strings.HasPrefix(b.Token, "ExponentPushToken[") || len(b.Token) > 200 {
		fail(w, 422, "that isn't a push token")
		return
	}
	if _, err := s.Pool.Exec(r.Context(), `INSERT INTO push_tokens (token, user_id, platform) VALUES ($1,$2,$3)
		ON CONFLICT (token) DO UPDATE SET user_id=EXCLUDED.user_id, platform=EXCLUDED.platform`, b.Token, userOf(r).ID, b.Platform); err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// socket says what changed; the client refetches. So a dropped frame can't lose a
// message, and there is one read path, whose access checks are all in one place.
func (s *Server) socket(w http.ResponseWriter, r *http.Request) {
	u := userOf(r)
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true, Subprotocols: []string{socketProtocol}}) // native clients send no Origin; auth is the bearer token
	if err != nil {
		return
	}
	defer c.CloseNow()
	ch := s.Hub.add(u.ID)
	defer s.Hub.remove(u.ID, ch)
	ctx := c.CloseRead(r.Context())
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case topic := <-ch:
			wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			frame := `{"topic":` + strconv.Quote(topic) + `}`
			if data, ok := strings.CutPrefix(topic, "stream:"); ok {
				frame = `{"topic":"agent_stream","data":` + data + `}` // a live preview of Suki's reply
			}
			err := c.Write(wctx, websocket.MessageText, []byte(frame))
			cancel()
			if err != nil {
				return
			}
		case <-ping.C:
			wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.Ping(wctx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

// ── Moderators ──────────────────────────────────────────────────────────────

func (s *Server) adminReports(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT r.id, coalesce(r.reporter_id::text,''), coalesce(r.subject_id::text,''), coalesce(r.match_id::text,''), r.source, r.reason, r.details, r.severity, r.status, r.action, r.created_at
		FROM reports r WHERE r.status='open' ORDER BY CASE r.severity WHEN 'critical' THEN 0 WHEN 'high' THEN 1 ELSE 2 END, r.created_at LIMIT 100`)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, rep, sub, match, src, reason, det, sev, st, act string
		var at time.Time
		if err := rows.Scan(&id, &rep, &sub, &match, &src, &reason, &det, &sev, &st, &act, &at); err != nil {
			handleErr(w, r, err)
			return
		}
		out = append(out, map[string]any{"id": id, "reporter_id": rep, "subject_id": sub, "match_id": match, "source": src, "reason": reason, "details": det, "severity": sev, "status": st, "action": act, "created_at": at})
	}
	writeJSON(w, 200, map[string]any{"reports": out})
}

// adminResolve is the one place a person can be warned, released or banned, and
// only a human moderator reaches it.
func (s *Server) adminResolve(w http.ResponseWriter, r *http.Request) {
	var b struct{ Action, Note string }
	if !read(w, r, &b) {
		return
	}
	switch b.Action {
	case "dismiss", "release", "hold", "ban":
	default:
		fail(w, 422, "action must be dismiss, release, hold or ban")
		return
	}
	id := r.PathValue("id")
	var subject string
	if err := s.Pool.QueryRow(r.Context(), `UPDATE reports SET status='resolved', action=$2, details=details || CASE WHEN $3='' THEN '' ELSE ' | ' || $3 END, resolved_by=$4, resolved_at=now() WHERE id=$1 RETURNING coalesce(subject_id::text,'')`,
		id, b.Action, b.Note, userOf(r).ID).Scan(&subject); err != nil {
		fail(w, 404, "not found")
		return
	}
	switch b.Action {
	case "dismiss", "release":
		if subject != "" {
			// Another critical report still open keeps the hold.
			if _, err := s.Pool.Exec(r.Context(), `UPDATE profiles SET status=CASE WHEN completed_at IS NULL THEN 'onboarding' ELSE 'active' END
				WHERE user_id=$1 AND status='held' AND NOT EXISTS (SELECT 1 FROM reports WHERE subject_id=$1 AND status='open' AND severity='critical')`, subject); err != nil {
				handleErr(w, r, err)
				return
			}
		}
	case "hold":
		if subject != "" {
			if err := s.Dating.HoldForReview(r.Context(), subject); err != nil {
				handleErr(w, r, err)
				return
			}
		}
	case "ban":
		if subject != "" {
			if err := s.Dating.HoldForReview(r.Context(), subject); err != nil {
				handleErr(w, r, err)
				return
			}
			if err := s.Dating.CloseEverything(r.Context(), subject); err != nil {
				handleErr(w, r, err)
				return
			}
			if err := s.Auth.Ban(r.Context(), subject); err != nil {
				handleErr(w, r, err)
				return
			}
		}
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// adminPhotos lists photos the vision check couldn't settle.
func (s *Server) adminPhotos(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT id, user_id, created_at FROM photos WHERE status='review' ORDER BY created_at LIMIT 100`)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, user string
		var at time.Time
		if err := rows.Scan(&id, &user, &at); err != nil {
			handleErr(w, r, err)
			return
		}
		out = append(out, map[string]any{"id": id, "user_id": user, "created_at": at})
	}
	writeJSON(w, 200, map[string]any{"photos": out})
}

// adminPhotoDecide is a person settling one of those photos.
func (s *Server) adminPhotoDecide(w http.ResponseWriter, r *http.Request) {
	var b struct{ Decision, Reason string }
	if !read(w, r, &b) {
		return
	}
	if b.Decision != "approved" && b.Decision != "rejected" {
		fail(w, 422, "decision must be approved or rejected")
		return
	}
	id := r.PathValue("id")
	var key string
	if err := s.Pool.QueryRow(r.Context(), `SELECT s3_key FROM photos WHERE id=$1 AND status='review'`, id).Scan(&key); err != nil {
		fail(w, 404, "not found")
		return
	}
	owner, err := profile.RecordReview(r.Context(), s.Pool, id, b.Decision, b.Reason, "")
	if err != nil {
		handleErr(w, r, err)
		return
	}
	if b.Decision == "rejected" {
		if err := s.Photos.Delete(r.Context(), key); err != nil {
			slog.Error("photo delete", "err", err)
		}
	} else if err := workflows.AfterProfileChange(r.Context(), owner); err != nil {
		handleErr(w, r, err)
		return
	}
	s.Dating.Touch(r.Context(), owner, "photos")
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// adminMetrics is system-level reliability: completion, retries, recovery, human
// intervention, spend. It contains counts only, never a score.
func (s *Server) adminMetrics(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{}
	q := func(name, sql string) {
		var n float64
		if err := s.Pool.QueryRow(r.Context(), sql).Scan(&n); err != nil {
			slog.Warn("metric", "name", name, "err", err)
		}
		out[name] = n
	}
	q("users", `SELECT count(*) FROM users WHERE deleted_at IS NULL`)
	q("active_profiles", `SELECT count(*) FROM profiles WHERE status='active'`)
	q("proposals_7d", `SELECT count(*) FROM proposals WHERE created_at > now()-interval '7 days'`)
	q("matches_7d", `SELECT count(*) FROM matches WHERE created_at > now()-interval '7 days'`)
	q("dates_confirmed", `SELECT count(*) FROM date_plans WHERE status IN ('confirmed','completed')`)
	q("goals_completed_7d", `SELECT count(*) FROM goals WHERE status='completed' AND updated_at > now()-interval '7 days'`)
	q("goals_needing_attention", `SELECT count(*) FROM goals WHERE status='needs_attention'`)
	q("mission_completion_rate_7d", `SELECT coalesce(avg((status='completed')::int),0) FROM goals WHERE created_at > now()-interval '7 days' AND status NOT IN ('planning','active')`)
	q("task_success_rate_7d", `SELECT coalesce(avg((status='succeeded')::int),0) FROM tasks WHERE created_at > now()-interval '7 days' AND status IN ('succeeded','failed')`)
	q("leases_reclaimed_7d", `SELECT count(*) FROM events WHERE type='lease.reclaimed' AND created_at > now()-interval '7 days'`)
	q("task_retries_7d", `SELECT count(*) FROM events WHERE type='task.retry' AND created_at > now()-interval '7 days'`)
	q("turns_failed_7d", `SELECT count(*) FROM agent_turns WHERE status='failed' AND created_at > now()-interval '7 days'`)
	q("turn_p95_ms_24h", `SELECT coalesce(percentile_cont(0.95) WITHIN GROUP (ORDER BY extract(epoch FROM finished_at-created_at)*1000),0) FROM agent_turns WHERE status='done' AND finished_at > now()-interval '24 hours'`)
	q("tokens_today", `SELECT coalesce(sum(prompt_tokens+completion_tokens),0) FROM llm_calls WHERE created_at > date_trunc('day', now())`)
	q("llm_errors_today", `SELECT count(*) FROM llm_calls WHERE error<>'' AND created_at > date_trunc('day', now())`)
	q("open_reports", `SELECT count(*) FROM reports WHERE status='open'`)
	q("open_reports_critical", `SELECT count(*) FROM reports WHERE status='open' AND severity='critical'`)
	writeJSON(w, 200, out)
}

// adminTimeline is the execution timeline of one goal: what each agent did, why,
// what changed. Event data never holds a score (see workflows).
func (s *Server) adminTimeline(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT type, data, created_at FROM events WHERE goal_id=$1 ORDER BY id LIMIT 500`, r.PathValue("id"))
	if err != nil {
		handleErr(w, r, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var t string
		var data []byte
		var at time.Time
		if err := rows.Scan(&t, &data, &at); err != nil {
			handleErr(w, r, err)
			return
		}
		out = append(out, map[string]any{"type": t, "data": json.RawMessage(data), "at": at})
	}
	writeJSON(w, 200, map[string]any{"events": out})
}

// export is what a person can take with them: their words, not others'.
func (s *Server) export(w http.ResponseWriter, r *http.Request) {
	u := userOf(r)
	p, err := profile.Get(r.Context(), s.Pool, u.ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	photos, err := profile.Photos(r.Context(), s.Pool, u.ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	conv, err := dating.ConversationID(r.Context(), s.Pool, u.ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	matches, err := ids(r.Context(), s.Pool, `SELECT id FROM matches WHERE user_a=$1 OR user_b=$1`, u.ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	agentMsgs, chats, err := s.Chats.Export(r.Context(), u.ID, conv, matches)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	buddies, err := s.Dating.MyBuddyRequests(r.Context(), u.ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	credits, err := s.Billing.Usage(r.Context(), u.ID, 10000)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	notes, err := ids(r.Context(), s.Pool, `SELECT content FROM memories WHERE user_id=$1`, u.ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"account": map[string]any{"email": u.Email, "created_at": u.CreatedAt}, "profile": p, "photos": photos,
		"settings": dating.GetSettings(r.Context(), s.Pool, u.ID), "buddy_posts": buddies, "credit_history": credits, "suki_conversation": agentMsgs, "messages_you_sent": chats, "suki_notes": notes})
}
