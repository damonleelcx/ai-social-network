-- 0003 · Engine additions: deterministic tool tasks, and what a goal works on
-- (a match, a date or a photo).
ALTER TABLE goals ADD COLUMN subject_id text;
ALTER TABLE tasks DROP CONSTRAINT tasks_kind_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_kind_check CHECK (kind IN ('plan','llm','wait','finish','tool'));
ALTER TABLE goals DROP CONSTRAINT goals_domain_check;
ALTER TABLE goals ADD CONSTRAINT goals_domain_check CHECK (domain IN ('match','coach','date','debrief','taste','safety','photo','general'));
ALTER TABLE users DROP CONSTRAINT users_role_check;
UPDATE users SET role = 'member' WHERE role = 'player';
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('member','moderator','admin'));
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'member';
ALTER TABLE users ADD COLUMN terms_accepted_at timestamptz;

-- Every chat message lives in MongoDB (internal/chatstore), Suki's included.
-- conversations stays here as the anchor goals and turns point at.
DROP TABLE messages;

-- A six-digit emailed code replaces a link: it works on any phone, and needs
-- no universal-link setup. Only its hash is stored and guesses are counted.
DROP TABLE email_tokens;
CREATE TABLE email_codes (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  purpose    text NOT NULL CHECK (purpose IN ('verify','reset')),
  code_hash  text NOT NULL,
  attempts   int NOT NULL DEFAULT 0,
  expires_at timestamptz NOT NULL,
  used_at    timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX email_codes_user ON email_codes (user_id, purpose, created_at DESC);

-- Failed sign-ins, for throttling by address.
CREATE TABLE auth_failures (
  email      text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX auth_failures_email ON auth_failures (lower(email), created_at);
