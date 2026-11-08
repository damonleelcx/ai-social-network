-- 0022 · What people wrote about themselves, in the languages others read it in. Keyed by a hash
-- of the words and the language, so a change to a profile simply misses and is translated anew.
-- Each row belongs to the person it translates and goes with their account.

CREATE TABLE translations (
  key        text PRIMARY KEY,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  lang       text NOT NULL,
  body       jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX translations_user ON translations (user_id);
