-- 0016 · Signing in with another service. Someone with no password has an empty
-- password_hash, which CheckPassword never accepts.

CREATE TABLE external_identities (
  provider   text NOT NULL CHECK (provider IN ('wechat','google')),
  subject    text NOT NULL,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (provider, subject)
);
CREATE INDEX external_identities_user ON external_identities (user_id);
