-- 0017 · Inviting friends. Each person gets one code, and each new account can name one
-- person who invited them. rewarded_at is set once, when the friend's profile goes live.

CREATE TABLE referral_codes (
  user_id    uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  code       text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE referrals (
  invitee_id  uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  referrer_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  rewarded_at timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now(),
  CHECK (invitee_id <> referrer_id)
);
CREATE INDEX referrals_referrer ON referrals (referrer_id, rewarded_at);

ALTER TABLE credit_ledger DROP CONSTRAINT credit_ledger_reason_check;
ALTER TABLE credit_ledger ADD CONSTRAINT credit_ledger_reason_check
  CHECK (reason IN ('purchase','turn','refund','grant','referral'));
