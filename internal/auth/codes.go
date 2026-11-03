package auth

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	mailer "github.com/damonleelcx/ai-social-network/internal/mail"
)

// Emailed six-digit codes. A code is short, so the defence is in the limits: it
// expires in 15 minutes, five wrong guesses kill it, and at most three can be
// requested per hour. The hash includes the user id, so a table of hashes cannot
// be reversed with one shared lookup table.

const (
	maxCodeAttempts = 5
	maxCodesPerHour = 3
)

func newCode() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%06d", binary.BigEndian.Uint64(b[:])%1_000_000)
}

func codeHash(userID, code string) string { return Hash(userID + ":" + strings.TrimSpace(code)) }

func (s *Service) issueCode(ctx context.Context, userID, purpose string) (string, error) {
	var recent int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM email_codes WHERE user_id=$1 AND purpose=$2 AND created_at > now()-interval '1 hour'`,
		userID, purpose).Scan(&recent); err != nil {
		return "", err
	}
	if recent >= maxCodesPerHour {
		return "", ErrTooManyCodes
	}
	code := newCode()
	_, err := s.Pool.Exec(ctx, `INSERT INTO email_codes (user_id, purpose, code_hash, expires_at) VALUES ($1,$2,$3, now()+interval '15 minutes')`,
		userID, purpose, codeHash(userID, code))
	return code, err
}

// consumeCode checks a code against the newest live one, counting wrong guesses.
func consumeCode(ctx context.Context, tx pgx.Tx, userID, purpose, code string) error {
	var id, hash string
	var attempts int
	err := tx.QueryRow(ctx, `SELECT id, code_hash, attempts FROM email_codes
		WHERE user_id=$1 AND purpose=$2 AND used_at IS NULL AND expires_at>now()
		ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, userID, purpose).Scan(&id, &hash, &attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCodeInvalid
	}
	if err != nil {
		return err
	}
	if attempts >= maxCodeAttempts {
		return ErrCodeInvalid
	}
	if hash != codeHash(userID, code) {
		if _, err := tx.Exec(ctx, `UPDATE email_codes SET attempts=attempts+1 WHERE id=$1`, id); err != nil {
			return err
		}
		return errWrongCode
	}
	_, err = tx.Exec(ctx, `UPDATE email_codes SET used_at=now() WHERE user_id=$1 AND purpose=$2 AND used_at IS NULL`, userID, purpose)
	return err
}

// errWrongCode commits the attempt count, then reads as ErrCodeInvalid outside.
var errWrongCode = errors.New("wrong code")

func (s *Service) withCode(ctx context.Context, userID, purpose, code string, then func(tx pgx.Tx) error) error {
	wrong := false
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := consumeCode(ctx, tx, userID, purpose, code); err != nil {
			if errors.Is(err, errWrongCode) {
				wrong = true
				return nil // keep the counted attempt
			}
			return err
		}
		return then(tx)
	})
	if wrong {
		return ErrCodeInvalid
	}
	return err
}

func (s *Service) SendVerification(ctx context.Context, userID, lang string) error {
	u, err := s.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.EmailVerified {
		return nil
	}
	code, err := s.issueCode(ctx, userID, "verify")
	if err != nil {
		return err
	}
	m := codeMail("Your SynAgent verification code", "Welcome! Here is the code to confirm your email address.", u.Name, code, "If you didn't sign up, you can ignore this email.", lang)
	m.To = u.Email
	_, err = s.Mailer.Send(ctx, m)
	return err
}

func (s *Service) VerifyEmail(ctx context.Context, userID, code string) (*User, error) {
	err := s.withCode(ctx, userID, "verify", code, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE users SET email_verified_at=coalesce(email_verified_at, now()), updated_at=now() WHERE id=$1`, userID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.UserByID(ctx, userID)
}

// ForgotPassword always succeeds from the caller's view: whether an account
// exists is not disclosed, and neither is hitting the hourly limit.
func (s *Service) ForgotPassword(ctx context.Context, email, lang string) error {
	var id, name, addr string
	err := s.Pool.QueryRow(ctx, `SELECT id, name, email FROM users WHERE lower(email)=lower($1) AND deleted_at IS NULL`, strings.TrimSpace(email)).Scan(&id, &name, &addr)
	if err != nil {
		return nil
	}
	code, err := s.issueCode(ctx, id, "reset")
	if errors.Is(err, ErrTooManyCodes) {
		return nil
	}
	if err != nil {
		return err
	}
	m := codeMail("Your SynAgent password reset code", "Here is the code to choose a new password.", name, code, "If you didn't ask for this, your password has not changed and you can ignore this email.", lang)
	m.To = addr
	_, err = s.Mailer.Send(ctx, m)
	return err
}

// ResetPassword consumes the code, sets the password, marks the email verified
// (they proved they receive mail there) and signs out every session.
func (s *Service) ResetPassword(ctx context.Context, email, code, password string) error {
	var userID, addr string
	err := s.Pool.QueryRow(ctx, `SELECT id, email FROM users WHERE lower(email)=lower($1) AND deleted_at IS NULL`, strings.TrimSpace(email)).Scan(&userID, &addr)
	if err != nil {
		return ErrCodeInvalid
	}
	// Checked only after the code: a weak-password answer before it would confirm that
	// the address has an account.
	return s.withCode(ctx, userID, "reset", code, func(tx pgx.Tx) error {
		if err := ValidatePassword(password, addr); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET password_hash=$2, email_verified_at=coalesce(email_verified_at, now()), updated_at=now() WHERE id=$1`,
			userID, HashPassword(password)); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
		return err
	})
}

func codeMail(subject, lead, name, code, foot, lang string) mailer.Message {
	greet := "Hi"
	if name != "" {
		greet = "Hi " + name
	}
	text := fmt.Sprintf("%s,\n\n%s\n\n    %s\n\nIt works for 15 minutes.\n\n%s\n\nSuki", greet, lead, code, foot)
	return mailer.Message{Subject: subject, Text: text, HTML: mailer.CodePage(lang, subject, greet+". "+lead, code, foot)}
}
