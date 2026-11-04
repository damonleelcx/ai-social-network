package billing

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	// ReferralCredits is what someone earns for each friend who joins.
	ReferralCredits = 20
	// ReferralMonthlyCap limits what one person can earn in 30 days, so that making
	// accounts doesn't become a way to get credits for free.
	ReferralMonthlyCap = 20
)

// No 0/O or 1/I/L, because people read these out and type them.
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

func newReferralCode() (string, error) {
	var b strings.Builder
	for i := 0; i < 8; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeAlphabet))))
		if err != nil {
			return "", err
		}
		b.WriteByte(codeAlphabet[n.Int64()])
	}
	return b.String(), nil
}

// ReferralCode is this person's code, made the first time it's asked for.
func (s *Service) ReferralCode(ctx context.Context, userID string) (string, error) {
	var last error
	for i := 0; i < 5; i++ {
		code, err := newReferralCode()
		if err != nil {
			return "", err
		}
		var got string
		// No row back means the new code was someone else's, so try another.
		last = s.Pool.QueryRow(ctx, `WITH made AS (
				INSERT INTO referral_codes (user_id, code) VALUES ($1,$2) ON CONFLICT DO NOTHING RETURNING code)
			SELECT code FROM made UNION ALL SELECT code FROM referral_codes WHERE user_id=$1 LIMIT 1`, userID, code).Scan(&got)
		if last == nil {
			return got, nil
		}
		if !errors.Is(last, pgx.ErrNoRows) {
			return "", last
		}
	}
	return "", last
}

// InviteLink opens the web app, which works on any device without installing anything.
func (s *Service) InviteLink(code string) string {
	return strings.TrimRight(s.PublicOrigin, "/") + "/app/?ref=" + code
}

// Attribute records who invited a new account. A code that doesn't exist is ignored:
// it must never stop someone signing up.
func (s *Service) Attribute(ctx context.Context, inviteeID, code string) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil
	}
	_, err := s.Pool.Exec(ctx, `INSERT INTO referrals (invitee_id, referrer_id)
		SELECT $1, c.user_id FROM referral_codes c JOIN users u ON u.id=c.user_id
		WHERE c.code=$2 AND c.user_id<>$1 AND u.deleted_at IS NULL
		ON CONFLICT (invitee_id) DO NOTHING`, inviteeID, code)
	return err
}

// Rewarded says who just earned credits for a friend.
type Rewarded struct {
	UserID  string
	Credits int
	Balance int
}

// RewardReferral pays whoever invited this person, once. It runs when their profile goes
// live, so a bare sign-up earns nothing. Nil means there's nothing to pay: nobody invited
// them, it's been paid, or the inviter has reached the cap.
func (s *Service) RewardReferral(ctx context.Context, inviteeID string) (*Rewarded, error) {
	var out *Rewarded
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var referrer string
		err := tx.QueryRow(ctx, `SELECT r.referrer_id FROM referrals r JOIN users u ON u.id=r.referrer_id
			WHERE r.invitee_id=$1 AND r.rewarded_at IS NULL AND u.deleted_at IS NULL FOR UPDATE OF r`, inviteeID).Scan(&referrer)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		var recent int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM referrals WHERE referrer_id=$1 AND rewarded_at > now() - interval '30 days'`, referrer).Scan(&recent); err != nil {
			return err
		}
		if recent >= ReferralMonthlyCap {
			return nil
		}
		if _, err := tx.Exec(ctx, `UPDATE referrals SET rewarded_at=now() WHERE invitee_id=$1`, inviteeID); err != nil {
			return err
		}
		if err := addCredits(ctx, tx, referrer, ReferralCredits, "referral", "referral:"+inviteeID); err != nil {
			return err
		}
		var balance int
		if err := tx.QueryRow(ctx, `SELECT balance FROM credit_balances WHERE user_id=$1`, referrer).Scan(&balance); err != nil {
			return err
		}
		out = &Rewarded{UserID: referrer, Credits: ReferralCredits, Balance: balance}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ReferralStatus is what someone sees about their invitations.
type ReferralStatus struct {
	Code             string `json:"code"`
	Link             string `json:"link"`
	CreditsPerFriend int    `json:"credits_per_friend"`
	Joined           int    `json:"joined"`
	Rewarded         int    `json:"rewarded"`
	CreditsEarned    int    `json:"credits_earned"`
}

func (s *Service) ReferralStatus(ctx context.Context, userID string) (*ReferralStatus, error) {
	code, err := s.ReferralCode(ctx, userID)
	if err != nil {
		return nil, err
	}
	st := &ReferralStatus{Code: code, Link: s.InviteLink(code), CreditsPerFriend: ReferralCredits}
	err = s.Pool.QueryRow(ctx, `SELECT count(*), count(rewarded_at),
			(SELECT coalesce(sum(delta),0) FROM credit_ledger WHERE user_id=$1 AND reason='referral')
		FROM referrals WHERE referrer_id=$1`, userID).Scan(&st.Joined, &st.Rewarded, &st.CreditsEarned)
	if err != nil {
		return nil, err
	}
	return st, nil
}
