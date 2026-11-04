package externalauth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Google checks the ID tokens Google signs for this app's sign-in.
type Google struct {
	// ClientID is the OAuth web client id the app asks for tokens with. A token issued to
	// any other app is refused.
	ClientID string
	// CertsURL is for tests. Empty means Google itself.
	CertsURL string
	HTTP     *http.Client

	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

// GoogleUser is what a checked token says.
type GoogleUser struct {
	Subject string
	Email   string
	Name    string
}

func (g *Google) refresh(ctx context.Context) error {
	url := g.CertsURL
	if url == "" {
		url = "https://www.googleapis.com/oauth2/v3/certs"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := g.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("google sign-in: can't reach Google")
	}
	defer resp.Body.Close()
	var set struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<18)).Decode(&set); err != nil {
		return errors.New("google sign-in: unreadable keys")
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		n, err1 := base64.RawURLEncoding.DecodeString(k.N)
		e, err2 := base64.RawURLEncoding.DecodeString(k.E)
		if err1 != nil || err2 != nil {
			continue
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	if len(keys) == 0 {
		return errors.New("google sign-in: no keys")
	}
	g.keys, g.fetched = keys, time.Now()
	return nil
}

func (g *Google) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	stale := time.Since(g.fetched) > time.Hour
	if k, ok := g.keys[kid]; ok && !stale {
		return k, nil
	}
	// Google rotates its keys, so an unknown id may be a new one. At most one fetch a minute,
	// so made-up ids can't make this hammer Google.
	if stale || time.Since(g.fetched) > time.Minute {
		if err := g.refresh(ctx); err != nil {
			return nil, err
		}
	}
	if k, ok := g.keys[kid]; ok {
		return k, nil
	}
	return nil, ErrRejected
}

// Verify checks the signature, issuer, audience and expiry, and that Google has verified the
// address. Anything wrong with the token itself is ErrRejected.
func (g *Google) Verify(ctx context.Context, idToken string, now time.Time) (*GoogleUser, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, ErrRejected
	}
	head, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrRejected
	}
	var h struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	// Only RS256: the algorithm can't be left for the token to choose.
	if json.Unmarshal(head, &h) != nil || h.Alg != "RS256" {
		return nil, ErrRejected
	}
	key, err := g.key(ctx, h.Kid)
	if err != nil {
		return nil, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrRejected
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], sig) != nil {
		return nil, ErrRejected
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrRejected
	}
	var c struct {
		Iss           string `json:"iss"`
		Aud           string `json:"aud"`
		Sub           string `json:"sub"`
		Exp           int64  `json:"exp"`
		Email         string `json:"email"`
		EmailVerified any    `json:"email_verified"`
		Name          string `json:"name"`
	}
	if json.Unmarshal(body, &c) != nil {
		return nil, ErrRejected
	}
	if (c.Iss != "https://accounts.google.com" && c.Iss != "accounts.google.com") || c.Aud != g.ClientID || c.Sub == "" || now.Unix() >= c.Exp {
		return nil, ErrRejected
	}
	if c.Email == "" || (c.EmailVerified != true && c.EmailVerified != "true") {
		return nil, ErrRejected
	}
	return &GoogleUser{Subject: c.Sub, Email: c.Email, Name: c.Name}, nil
}
