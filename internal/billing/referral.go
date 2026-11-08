package billing

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"

	"github.com/jackc/pgx/v5"
)

// No 0/O or 1/I/L, because people read these out and type them.
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// newAccountWindow is how fresh an account must be to be attributed to an invite. Sign-up
// attributes at once; this only stops an existing account from being claimed later.
const newAccountWindow = "1 hour"

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

// ReferralCode is this person's code, made the first time it's asked for. It is random and
// opaque: share links carry it, never a user id or openid.
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
// it must never stop someone signing up. Only an account made moments ago can be
// attributed, only once, and never to itself.
func (s *Service) Attribute(ctx context.Context, inviteeID, code string) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil
	}
	_, err := s.Pool.Exec(ctx, `INSERT INTO referrals (invitee_id, referrer_id)
		SELECT i.id, c.user_id FROM referral_codes c JOIN users u ON u.id=c.user_id JOIN users i ON i.id=$1
		WHERE c.code=$2 AND c.user_id<>$1 AND u.deleted_at IS NULL
		  AND i.created_at > now() - interval '`+newAccountWindow+`'
		ON CONFLICT (invitee_id) DO NOTHING`, inviteeID, code)
	return err
}

// Rewarded says who just earned credits for a friend joining: the inviter, and the friend.
type Rewarded struct {
	UserID  string
	Credits int
	Balance int

	InviteeID      string
	InviteeCredits int
	InviteeBalance int
}

// RewardReferral pays both sides of an invitation, once. It runs when the invited friend's
// profile goes live, so sharing a link earns nothing, and neither does a bare sign-up. Nil
// means there's nothing to pay: nobody invited them, it's been decided already, or the
// inviter has reached a cap. A capped invitation is settled unpaid for good, so a later
// day's allowance never pays it.
func (s *Service) RewardReferral(ctx context.Context, inviteeID string) (*Rewarded, error) {
	r := s.rewards()
	now := s.now()
	var out *Rewarded
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var referrer string
		err := tx.QueryRow(ctx, `SELECT r.referrer_id FROM referrals r JOIN users u ON u.id=r.referrer_id
			WHERE r.invitee_id=$1 AND r.rewarded_at IS NULL AND r.skipped_at IS NULL AND u.deleted_at IS NULL FOR UPDATE OF r`, inviteeID).Scan(&referrer)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		// One inviter's friends are paid one at a time, so two going live together can't
		// both slip under a cap.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('referrer:' || $1::text, 0))`, referrer); err != nil {
			return err
		}
		var today, total int
		if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE rewarded_at >= $2), count(*)
			FROM referrals WHERE referrer_id=$1 AND rewarded_at IS NOT NULL`, referrer, ChinaDay(now)).Scan(&today, &total); err != nil {
			return err
		}
		if today >= r.InviteDailyCap || total >= r.InviteTotalCap || (r.InviterCredits <= 0 && r.InviteeCredits <= 0) {
			_, err := tx.Exec(ctx, `UPDATE referrals SET skipped_at=$2 WHERE invitee_id=$1`, inviteeID, now)
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE referrals SET rewarded_at=$2 WHERE invitee_id=$1`, inviteeID, now); err != nil {
			return err
		}
		out = &Rewarded{UserID: referrer, InviteeID: inviteeID}
		if r.InviterCredits > 0 {
			if err := addCredits(ctx, tx, referrer, r.InviterCredits, "referral", "referral:"+inviteeID); err != nil {
				return err
			}
			out.Credits = r.InviterCredits
		}
		if r.InviteeCredits > 0 {
			if err := addCredits(ctx, tx, inviteeID, r.InviteeCredits, "invited", "invited:"+inviteeID); err != nil {
				return err
			}
			out.InviteeCredits = r.InviteeCredits
		}
		return tx.QueryRow(ctx, `SELECT coalesce((SELECT balance FROM credit_balances WHERE user_id=$1),0),
			coalesce((SELECT balance FROM credit_balances WHERE user_id=$2),0)`, referrer, inviteeID).Scan(&out.Balance, &out.InviteeBalance)
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
	// CreditsForFriend is what the invited friend gets.
	CreditsForFriend int `json:"credits_for_friend"`
	Joined           int `json:"joined"`
	Rewarded         int `json:"rewarded"`
	CreditsEarned    int `json:"credits_earned"`
}

func (s *Service) ReferralStatus(ctx context.Context, userID string) (*ReferralStatus, error) {
	code, err := s.ReferralCode(ctx, userID)
	if err != nil {
		return nil, err
	}
	r := s.rewards()
	st := &ReferralStatus{Code: code, Link: s.InviteLink(code), CreditsPerFriend: r.InviterCredits, CreditsForFriend: r.InviteeCredits}
	err = s.Pool.QueryRow(ctx, `SELECT count(*), count(rewarded_at),
			(SELECT coalesce(sum(delta),0) FROM credit_ledger WHERE user_id=$1 AND reason='referral')
		FROM referrals WHERE referrer_id=$1`, userID).Scan(&st.Joined, &st.Rewarded, &st.CreditsEarned)
	if err != nil {
		return nil, err
	}
	return st, nil
}
