-- 0013 · Fixes from the independent review.

-- The tool ledger holds what people said, so it belongs to them and leaves with them.
ALTER TABLE tool_calls ADD COLUMN user_id uuid REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX tool_calls_user ON tool_calls (user_id);

-- Only a hash, enough to refuse a banned address at sign-up.
CREATE TABLE banned_emails (
  email_hash text PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- At most one running turn per person, enforced where a race can't get past it.
CREATE UNIQUE INDEX agent_turns_one_leased ON agent_turns (user_id) WHERE status = 'leased';

-- The newest message seq already counted, so a retried send isn't counted twice.
ALTER TABLE matches ADD COLUMN last_seq bigint NOT NULL DEFAULT 0;

-- A completed date whose debrief questions haven't all gone out is found by this.
CREATE INDEX date_plans_debrief_due ON date_plans (id) WHERE status = 'completed' AND NOT debrief_sent;
