package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// One account works in both copies of the server (US and China), but lives in one: its
// home region. The copy that holds an account answers for it, and the other forwards to
// it. These are the questions the forwarding copy asks.

// HasEmail says whether a live account here uses this address.
func (s *Service) HasEmail(ctx context.Context, email string) (bool, error) {
	var ok bool
	err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE lower(email)=lower($1) AND deleted_at IS NULL)`, strings.TrimSpace(email)).Scan(&ok)
	return ok, err
}

// HasEmailHash is HasEmail for an address the other copy only sends as Hash(lower(email)),
// so no address crosses between regions just to ask.
func (s *Service) HasEmailHash(ctx context.Context, hash string) (bool, error) {
	var ok bool
	err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE encode(sha256(convert_to(lower(email),'UTF8')),'hex')=$1 AND deleted_at IS NULL)`, hash).Scan(&ok)
	return ok, err
}

// EmailHash is how an address is asked about across regions.
func EmailHash(email string) string { return Hash(strings.ToLower(strings.TrimSpace(email))) }

// ExternalUser is the live account behind a sign-in identity, or ErrInvalidCredentials.
func (s *Service) ExternalUser(ctx context.Context, provider, subject string) (*User, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `SELECT i.user_id FROM external_identities i JOIN users u ON u.id=i.user_id
		WHERE i.provider=$1 AND i.subject=$2 AND u.deleted_at IS NULL`, provider, subject).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	return s.UserByID(ctx, id)
}
