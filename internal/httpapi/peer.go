package httpapi

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/damonleelcx/ai-social-network/internal/auth"
)

// Two copies of this server run: the US one and the China one (for the WeChat mini program,
// whose data stays in China). An account lives in one of them, its home region, and works
// from both: the copy that doesn't hold it forwards its requests to the one that does.
//
//   - A session says where it lives: China's tokens start with "cn.", the US's have no prefix.
//   - Before there is a session (signing in, a forgotten password), an address this copy
//     doesn't know is tried at the other copy.
//   - Signing up checks the other copy for the address first, by hash only.
//   - Signing in with WeChat or Google asks the other copy for a session when the identity
//     is theirs.
//
// The copies trust each other through a shared secret in a header, and a forwarded request
// is never forwarded again.

const (
	peerHeader   = "X-Synagent-Peer"
	peerIPHeader = "X-Synagent-Client-Ip"
)

// Peer is the other copy. Nil means this copy runs alone.
type Peer struct {
	Region string // this copy: "us" or "cn"
	Secret string
	base   *url.URL
	proxy  *httputil.ReverseProxy
	client *http.Client
}

func NewPeer(region, peerURL, secret string) (*Peer, error) {
	if region != "us" && region != "cn" {
		return nil, fmt.Errorf("peer: region must be us or cn, not %q", region)
	}
	if len(secret) < 32 {
		return nil, errors.New("peer: the shared secret must be at least 32 characters")
	}
	u, err := url.Parse(strings.TrimRight(peerURL, "/"))
	if err != nil || u.Scheme != "https" && u.Scheme != "http" || u.Host == "" {
		return nil, fmt.Errorf("peer: bad url %q", peerURL)
	}
	p := &Peer{Region: region, Secret: secret, base: u, client: &http.Client{Timeout: 20 * time.Second}}
	p.proxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(u)
			pr.Out.Host = u.Host
			pr.Out.Header.Set(peerHeader, secret)
			pr.Out.Header.Set(peerIPHeader, clientIP(pr.In))
		},
		// Suki's replies stream.
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error("peer forward", "path", r.URL.Path, "err", err)
			fail(w, 502, "We're having a moment. Please try again shortly.")
		},
	}
	return p, nil
}

// HomeOf is the region a session token belongs to.
func HomeOf(token string) string {
	if strings.HasPrefix(token, "cn.") {
		return "cn"
	}
	return "us"
}

// TokenPrefix is what a region puts in front of the sessions it issues.
func TokenPrefix(region string) string {
	if region == "cn" {
		return "cn."
	}
	return ""
}

func (p *Peer) trusts(r *http.Request) bool {
	got := r.Header.Get(peerHeader)
	return got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(p.Secret)) == 1
}

type ctxPeer struct{}

// fromPeer says the other copy forwarded this request, so it must not go back.
func fromPeer(r *http.Request) bool { v, _ := r.Context().Value(ctxPeer{}).(bool); return v }

// requestToken is the session a request carries, in a header or, for a browser's socket,
// in its subprotocol.
func requestToken(r *http.Request) string {
	if t := bearer(r); t != "" {
		return t
	}
	for _, p := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
		if t, ok := strings.CutPrefix(strings.TrimSpace(p), "bearer."); ok {
			return t
		}
	}
	return ""
}

// byAddress are the requests without a session that name an account by its address.
var byAddress = map[string]bool{"/v1/auth/signin": true, "/v1/auth/forgot": true, "/v1/auth/reset": true}

// route sends each request to the copy that holds its account.
func (s *Server) route(next http.Handler) http.Handler {
	p := s.Peer
	if p == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p.trusts(r) {
			// The other copy saw the person; rate limits and session records use their address.
			if ip := r.Header.Get(peerIPHeader); ip != "" {
				r.Header.Set("X-Forwarded-For", ip)
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxPeer{}, true)))
			return
		}
		r.Header.Del(peerHeader)
		r.Header.Del(peerIPHeader)
		if strings.HasPrefix(r.URL.Path, "/internal/") {
			http.NotFound(w, r)
			return
		}
		if t := requestToken(r); t != "" && HomeOf(t) != p.Region {
			p.proxy.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodPost && byAddress[r.URL.Path] {
			body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
			if err != nil {
				fail(w, 400, "bad request")
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			var b struct{ Email string }
			_ = json.Unmarshal(body, &b)
			if b.Email != "" {
				here, err := s.Auth.HasEmail(r.Context(), b.Email)
				if err != nil {
					handleErr(w, r, err)
					return
				}
				if !here {
					p.proxy.ServeHTTP(w, r)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// call is a request from this copy to the other one's private API.
func (p *Peer) call(ctx context.Context, path string, in any, r *http.Request) (*http.Response, error) {
	b, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.base.String()+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(peerHeader, p.Secret)
	if r != nil {
		req.Header.Set(peerIPHeader, clientIP(r))
		req.Header.Set("User-Agent", r.UserAgent())
	}
	return p.client.Do(req)
}

// EmailTaken asks whether the other copy has an account with this address.
func (p *Peer) EmailTaken(ctx context.Context, email string) (bool, error) {
	resp, err := p.call(ctx, "/internal/peer/email", map[string]string{"email_hash": auth.EmailHash(email)}, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false, fmt.Errorf("peer email check: %d", resp.StatusCode)
	}
	var out struct{ Exists bool }
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out.Exists, err
}

// externalSession asks the other copy to sign in an identity it holds. It writes the other
// copy's answer and returns true if it did; false means the identity isn't theirs.
func (s *Server) externalSession(w http.ResponseWriter, r *http.Request, provider, subject string) bool {
	if s.Peer == nil || fromPeer(r) {
		return false
	}
	if _, err := s.Auth.ExternalUser(r.Context(), provider, subject); err == nil {
		return false // it's ours
	}
	resp, err := s.Peer.call(r.Context(), "/internal/peer/session", map[string]string{"provider": provider, "subject": subject}, r)
	if err != nil {
		slog.Warn("peer session", "err", err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
	return true
}

func (s *Server) peerOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Peer == nil || !fromPeer(r) {
			http.NotFound(w, r)
			return
		}
		next(w, r)
	}
}

// POST /internal/peer/email {email_hash} → {exists}
func (s *Server) peerEmail(w http.ResponseWriter, r *http.Request) {
	var b struct {
		EmailHash string `json:"email_hash"`
	}
	if !read(w, r, &b) {
		return
	}
	ok, err := s.Auth.HasEmailHash(r.Context(), b.EmailHash)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"exists": ok})
}

// POST /internal/peer/session {provider, subject} → a session, or 404 if the identity isn't here.
func (s *Server) peerSession(w http.ResponseWriter, r *http.Request) {
	var b struct{ Provider, Subject string }
	if !read(w, r, &b) {
		return
	}
	u, err := s.Auth.ExternalUser(r.Context(), b.Provider, b.Subject)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		handleErr(w, r, err)
		return
	}
	s.issue(w, r, 200, u)
}
