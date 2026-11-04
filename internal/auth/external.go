package auth

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"

	"crypto/sha256"

	"github.com/jackc/pgx/v5"
)

// External is someone another service has vouched for.
type External struct {
	Provider string // wechat | google
	Subject  string // that service's permanent id for them
	// Email is one the service has verified. Empty if it doesn't give one.
	Email       string
	Name        string
	Lang        string
	AcceptTerms bool
}

// syntheticEmail is for people with no address. It's unique per person, so a ban can hold
// against it, and .invalid is a domain that can never receive mail.
func syntheticEmail(provider, subject string) string {
	sum := sha256.Sum256([]byte(provider + ":" + subject))
	return provider + "-" + hex.EncodeToString(sum[:16]) + "@external.invalid"
}

// SignInExternal finds or creates the account behind an identity. New people have to have
// accepted the terms. If the service's verified address already has an account, the identity
// joins it only if that address was verified there too. Otherwise whoever signed up with it
// might not own it, and joining would let them keep a password into this person's account.
func (s *Service) SignInExternal(ctx context.Context, x External) (*User, bool, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `SELECT i.user_id FROM external_identities i JOIN users u ON u.id=i.user_id
		WHERE i.provider=$1 AND i.subject=$2 AND u.deleted_at IS NULL`, x.Provider, x.Subject).Scan(&id)
	if err == nil {
		u, err := s.UserByID(ctx, id)
		return u, false, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	email := syntheticEmail(x.Provider, x.Subject)
	if x.Email != "" {
		if email, err = NormalizeEmail(x.Email); err != nil {
			return nil, false, err
		}
	}
	var banned bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM banned_emails WHERE email_hash=$1)`, Hash(strings.ToLower(email))).Scan(&banned); err != nil {
		return nil, false, err
	}
	if banned {
		return nil, false, ErrBanned
	}
	var verified bool
	err = s.Pool.QueryRow(ctx, `SELECT id, email_verified_at IS NOT NULL FROM users WHERE lower(email)=lower($1) AND deleted_at IS NULL`, email).Scan(&id, &verified)
	switch {
	case err == nil:
		if !verified {
			return nil, false, ErrEmailTaken
		}
		if _, err := s.Pool.Exec(ctx, `INSERT INTO external_identities (provider, subject, user_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, x.Provider, x.Subject, id); err != nil {
			return nil, false, err
		}
		u, err := s.UserByID(ctx, id)
		return u, false, err
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, false, err
	}
	if !x.AcceptTerms {
		return nil, false, ErrTermsRequired
	}
	name := strings.TrimSpace(x.Name)
	if utf8.RuneCountInString(name) > 80 {
		name = string([]rune(name)[:80])
	}
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		// No password: password_hash stays empty, and CheckPassword never accepts that.
		if err := tx.QueryRow(ctx, `INSERT INTO users (email, password_hash, name, terms_accepted_at, email_verified_at) VALUES ($1,'',$2,now(),now()) RETURNING id`,
			email, name).Scan(&id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO external_identities (provider, subject, user_id) VALUES ($1,$2,$3)`, x.Provider, x.Subject, id); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO user_preferences (user_id, data) VALUES ($1, jsonb_build_object('language', $2::text))`, id, x.Lang)
		return err
	})
	if err != nil {
		return nil, false, err
	}
	u, err := s.UserByID(ctx, id)
	return u, true, err
}

// DeleteAccountExternal is DeleteAccount for someone proving who they are through the
// service they sign in with.
func (s *Service) DeleteAccountExternal(ctx context.Context, userID, provider, subject string) error {
	var ok bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM external_identities WHERE provider=$1 AND subject=$2 AND user_id=$3)`, provider, subject, userID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrInvalidCredentials
	}
	return s.deleteUser(ctx, userID)
}
