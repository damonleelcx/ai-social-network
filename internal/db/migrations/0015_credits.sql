-- 0015 · Credits: a person's first matches are free, then talking to Suki uses credits.

CREATE TABLE credit_balances (
  user_id    uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  balance    int NOT NULL DEFAULT 0 CHECK (balance >= 0),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- Every change to a balance, once. The key is what makes a retry harmless.
CREATE TABLE credit_ledger (
  id         bigserial PRIMARY KEY,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  delta      int NOT NULL,
  reason     text NOT NULL CHECK (reason IN ('purchase','turn','refund','grant')),
  key        text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX credit_ledger_user ON credit_ledger (user_id, created_at DESC);

-- Kept after an account is deleted, without the person: these are financial records.
CREATE TABLE credit_purchases (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      uuid REFERENCES users(id) ON DELETE SET NULL,
  package      text NOT NULL,
  credits      int NOT NULL,
  amount_cents int NOT NULL,
  currency     text NOT NULL,
  status       text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','paid')),
  session_id   text UNIQUE,
  created_at   timestamptz NOT NULL DEFAULT now(),
  paid_at      timestamptz
);
CREATE INDEX credit_purchases_user ON credit_purchases (user_id, created_at DESC);
