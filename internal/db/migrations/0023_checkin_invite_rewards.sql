-- 0023 · Credits people earn rather than buy: a daily check-in, and a welcome for both sides
-- when an invited friend's profile goes live. Each lands in credit_ledger with its own reason,
-- so earned credits stay distinguishable from purchased ones.

-- One row per person per calendar day in China time (Asia/Shanghai). The primary key is what
-- makes a double tap or two devices at once pay only once.
CREATE TABLE daily_checkins (
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  day        date NOT NULL,
  credits    int NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, day)
);

-- A referral is decided once: paid (rewarded_at) or, past the inviter's caps, settled without
-- pay (skipped_at). Either way it is never looked at again.
ALTER TABLE referrals ADD COLUMN skipped_at timestamptz;

ALTER TABLE credit_ledger DROP CONSTRAINT credit_ledger_reason_check;
ALTER TABLE credit_ledger ADD CONSTRAINT credit_ledger_reason_check
  CHECK (reason IN ('purchase','turn','refund','grant','referral','invited','checkin'));
