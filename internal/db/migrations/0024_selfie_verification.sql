-- 0024 · A live selfie checked against someone's main photo. The badge belongs to that photo:
-- change the main photo and it's gone until they check again. Selfies are never kept; only
-- the outcome, so attempts can be limited.

ALTER TABLE profiles ADD COLUMN verified_photo_id uuid REFERENCES photos(id) ON DELETE SET NULL;
ALTER TABLE profiles ADD COLUMN verified_at timestamptz;

CREATE TABLE selfie_checks (
  id         bigserial PRIMARY KEY,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  passed     boolean NOT NULL,
  reason     text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX selfie_checks_user ON selfie_checks (user_id, created_at DESC);
