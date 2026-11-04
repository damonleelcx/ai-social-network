// Package auth: accounts, passwords (argon2id), sessions and emailed tokens.
// Only hashes of secrets are stored — a database read yields no live session,
// no usable reset link and no password.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"

	mailer "github.com/damonleelcx/ai-social-network/internal/mail"
)

type User struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	Name          string     `json:"name"`
	Role          string     `json:"role"` // member | moderator | admin
	EmailVerified bool       `json:"email_verified"`
	// HasPassword is false for someone who only signs in through another service.
	HasPassword bool `json:"has_password"`
	CreatedAt     time.Time  `json:"created_at"`
	Admin         bool       `json:"admin"`
	VerifiedAt    *time.Time `json:"-"`
}

type Service struct {
	Pool         *pgxpool.Pool
	Mailer       mailer.Mailer
	PublicOrigin string
	SessionTTL   time.Duration
	// AdminEmails are operators (moderation). An address
	// counts only once it is verified, so signing up with an admin's address
	// grants nothing.
	AdminEmails []string
	// BeforeDelete runs after DeleteAccount has checked the password and
	// before the account row goes: chat messages (MongoDB) and photos (S3) are
	// outside the Postgres cascade, so they go here.
	// An error stops the deletion (nothing is lost; it can be retried).
	BeforeDelete func(ctx context.Context, userID string) error
}

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("an account with this email already exists")
	ErrWeakPassword       = errors.New("password must be at least 10 characters")
	ErrBadEmail           = errors.New("that email address does not look valid")
	ErrTokenInvalid       = errors.New("this link is invalid or has expired")
	ErrPasswordTooLong    = errors.New("password is too long")
	ErrPasswordIsEmail    = errors.New("password must not be your email address")
	ErrBadSessionID       = errors.New("bad session id")
	ErrCodeInvalid        = errors.New("that code is wrong or has expired")
	ErrTooManyCodes       = errors.New("too many codes requested; try again in an hour")
	ErrTermsRequired      = errors.New("please accept the terms to continue")
	ErrTooManyAttempts    = errors.New("too many attempts; try again in 15 minutes")
	ErrBanned             = errors.New("this email can't be used on SynAgent")
)

// UserFacing reports whether err is one of this package's own messages,
// written for people. Anything else (a database error, say) must be logged
// and replaced by a generic message, never shown.
func UserFacing(err error) bool {
	for _, e := range []error{ErrInvalidCredentials, ErrEmailTaken, ErrWeakPassword, ErrBadEmail, ErrTokenInvalid,
		ErrPasswordTooLong, ErrPasswordIsEmail, ErrBadSessionID, ErrCodeInvalid, ErrTooManyCodes, ErrTermsRequired, ErrTooManyAttempts, ErrBanned} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

// ── Passwords ──────────────────────────────────────────────────────────────

const (
	argonTime    = 2
	argonMemory  = 64 * 1024
	argonThreads = 2
	argonKeyLen  = 32
)

func HashPassword(pw string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	key := argon2.IDKey([]byte(pw), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
}

func CheckPassword(hash, pw string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[4])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[5])
	if err1 != nil || err2 != nil {
		return false
	}
	got := argon2.IDKey([]byte(pw), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// dummyHash equalises sign-in timing for unknown emails, so response time does
// not reveal which addresses have accounts.
var dummyHash = HashPassword("timing-equaliser-not-a-real-password")

func ValidatePassword(pw, email string) error {
	n := utf8.RuneCountInString(pw)
	if n < 10 {
		return ErrWeakPassword
	}
	if n > 200 {
		return ErrPasswordTooLong
	}
	if strings.EqualFold(pw, email) {
		return ErrPasswordIsEmail
	}
	return nil
}

func NormalizeEmail(e string) (string, error) {
	e = strings.TrimSpace(e)
	a, err := mail.ParseAddress(e)
	if err != nil || a.Address != e || !strings.Contains(e[strings.LastIndexByte(e, '@')+1:], ".") || len(e) > 254 {
		return "", ErrBadEmail
	}
	return e, nil
}

func token() (raw, hash string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, Hash(raw)
}

func Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// ── Accounts ───────────────────────────────────────────────────────────────

func (s *Service) SignUp(ctx context.Context, email, password, name, lang string, acceptTerms bool) (*User, error) {
	if !acceptTerms {
		return nil, ErrTermsRequired
	}
	email, err := NormalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if err := ValidatePassword(password, email); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > 80 {
		name = string([]rune(name)[:80])
	}
	var banned bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM banned_emails WHERE email_hash=$1)`, Hash(strings.ToLower(email))).Scan(&banned); err != nil {
		return nil, err
	}
	if banned {
		return nil, ErrBanned
	}
	var id string
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `INSERT INTO users (email, password_hash, name, terms_accepted_at) VALUES ($1,$2,$3, now()) RETURNING id`,
			email, HashPassword(password), name).Scan(&id)
		if err != nil {
			if strings.Contains(err.Error(), "23505") {
				return ErrEmailTaken
			}
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO user_preferences (user_id, data) VALUES ($1, jsonb_build_object('language', $2::text))`,
			id, lang)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := s.SendVerification(ctx, id, lang); err != nil {
		// The account exists; the person can ask for another link.
		return s.UserByID(ctx, id)
	}
	return s.UserByID(ctx, id)
}

func (s *Service) SignIn(ctx context.Context, email, password string) (*User, error) {
	var failures int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM auth_failures WHERE lower(email)=lower($1) AND created_at > now()-interval '15 minutes'`,
		strings.TrimSpace(email)).Scan(&failures); err != nil {
		return nil, err
	}
	if failures >= 10 {
		return nil, ErrTooManyAttempts
	}
	fail := func() {
		_, _ = s.Pool.Exec(ctx, `INSERT INTO auth_failures (email) VALUES ($1)`, strings.TrimSpace(email))
	}
	var id, hash string
	err := s.Pool.QueryRow(ctx, `SELECT id, password_hash FROM users WHERE lower(email)=lower($1) AND deleted_at IS NULL`,
		strings.TrimSpace(email)).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		CheckPassword(dummyHash, password)
		fail()
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !CheckPassword(hash, password) {
		fail()
		return nil, ErrInvalidCredentials
	}
	_, _ = s.Pool.Exec(ctx, `DELETE FROM auth_failures WHERE lower(email)=lower($1)`, strings.TrimSpace(email))
	return s.UserByID(ctx, id)
}

func (s *Service) UserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.Pool.QueryRow(ctx, `SELECT u.id, u.email, u.name, u.role, u.email_verified_at, u.created_at, u.password_hash <> ''
		FROM users u WHERE u.id=$1 AND u.deleted_at IS NULL`, id).
		Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.VerifiedAt, &u.CreatedAt, &u.HasPassword)
	if err != nil {
		return nil, err
	}
	u.EmailVerified = u.VerifiedAt != nil
	for _, a := range s.AdminEmails {
		if strings.EqualFold(a, u.Email) && u.EmailVerified {
			u.Admin = true
		}
	}
	if u.Role == "admin" {
		u.Admin = true
	}
	return &u, nil
}

// Ban ends an account for good. Only a hash of the address is kept, enough to refuse it
// at sign-up and of no other use.
func (s *Service) Ban(ctx context.Context, userID string) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var email string
		err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id=$1`, userID).Scan(&email)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO banned_emails (email_hash) VALUES ($1) ON CONFLICT DO NOTHING`, Hash(strings.ToLower(email))); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE users SET deleted_at=now(), email=email||'.banned.'||id WHERE id=$1 AND deleted_at IS NULL`, userID)
		return err
	})
}

// ── Sessions ───────────────────────────────────────────────────────────────

func (s *Service) CreateSession(ctx context.Context, userID, ua, ip string) (string, time.Time, error) {
	raw, hash := token()
	exp := time.Now().Add(s.SessionTTL)
	_, err := s.Pool.Exec(ctx, `INSERT INTO sessions (id, user_id, expires_at, user_agent, ip) VALUES ($1,$2,$3,$4,$5)`,
		hash, userID, exp, truncate(ua, 300), ip)
	return raw, exp, err
}

// SessionUser resolves a cookie to a user and slides last_seen at most once a
// minute, so every request is not a write.
func (s *Service) SessionUser(ctx context.Context, raw string) (*User, string, error) {
	if raw == "" {
		return nil, "", ErrInvalidCredentials
	}
	hash := Hash(raw)
	var userID string
	err := s.Pool.QueryRow(ctx, `UPDATE sessions SET last_seen_at = CASE WHEN last_seen_at < now()-interval '1 minute' THEN now() ELSE last_seen_at END
		WHERE id=$1 AND revoked_at IS NULL AND expires_at > now() RETURNING user_id`, hash).Scan(&userID)
	// Only a session that isn't there means signed out. A database error must not
	// read as that, or a brief outage would sign everyone out of the app.
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrInvalidCredentials
	}
	if err != nil {
		return nil, "", err
	}
	u, err := s.UserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrInvalidCredentials // the account has gone
	}
	return u, hash, err
}

func (s *Service) RevokeSession(ctx context.Context, userID, sessionHash string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE id=$1 AND user_id=$2`, sessionHash, userID)
	return err
}

func (s *Service) RevokeAllSessions(ctx context.Context, userID, except string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND id<>$2 AND revoked_at IS NULL`, userID, except)
	return err
}

type Session struct {
	ID        string    `json:"id"`
	UserAgent string    `json:"user_agent"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen_at"`
	Current   bool      `json:"current"`
}

func (s *Service) Sessions(ctx context.Context, userID, current string) ([]Session, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, user_agent, ip, created_at, last_seen_at FROM sessions
		WHERE user_id=$1 AND revoked_at IS NULL AND expires_at>now() ORDER BY last_seen_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var x Session
		if err := rows.Scan(&x.ID, &x.UserAgent, &x.IP, &x.CreatedAt, &x.LastSeen); err != nil {
			return nil, err
		}
		x.Current = x.ID == current
		x.ID = x.ID[:16] // enough to address it; never the whole hash
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Service) RevokeSessionPrefix(ctx context.Context, userID, prefix string) error {
	if len(prefix) != 16 {
		return ErrBadSessionID
	}
	_, err := s.Pool.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND left(id,16)=$2`, userID, prefix)
	return err
}

func (s *Service) ChangePassword(ctx context.Context, userID, current, next, keepSession string) error {
	var hash, email string
	if err := s.Pool.QueryRow(ctx, `SELECT password_hash, email FROM users WHERE id=$1`, userID).Scan(&hash, &email); err != nil {
		return err
	}
	if !CheckPassword(hash, current) {
		return ErrInvalidCredentials
	}
	if err := ValidatePassword(next, email); err != nil {
		return err
	}
	if _, err := s.Pool.Exec(ctx, `UPDATE users SET password_hash=$2, updated_at=now() WHERE id=$1`, userID, HashPassword(next)); err != nil {
		return err
	}
	return s.RevokeAllSessions(ctx, userID, keepSession)
}

// DeleteAccount removes the user and, by cascade, everything they own.
// Chat messages and photo bytes live outside Postgres; BeforeDelete removes them.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	var hash string
	if err := s.Pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id=$1`, userID).Scan(&hash); err != nil {
		return err
	}
	if !CheckPassword(hash, password) {
		return ErrInvalidCredentials
	}
	return s.deleteUser(ctx, userID)
}

func (s *Service) deleteUser(ctx context.Context, userID string) error {
	// Sign every session out first, so nothing can sit down again while the
	// account is being taken apart.
	if _, err := s.Pool.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID); err != nil {
		return err
	}
	if s.BeforeDelete != nil {
		if err := s.BeforeDelete(ctx, userID); err != nil {
			return fmt.Errorf("hand over before deletion: %w", err)
		}
	}
	_, err := s.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	return err
}

// ── Mail templates (English / 中文 / 한국어 / 日本語) ─────────────────────────
// Suki writes them. They carry a link and nothing else of value: the token is
// single-use and short-lived, and only its hash is stored.

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
