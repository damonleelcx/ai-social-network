-- 0014 · Buddies: finding company for one plan, beside the slower dating introductions.

CREATE TABLE buddy_requests (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind        text NOT NULL CHECK (kind IN ('movie','meal','show','travel','sports','games','business','study','other')),
  note        text NOT NULL DEFAULT '',
  when_text   text NOT NULL DEFAULT '',
  city_key    text NOT NULL,
  city_name   text NOT NULL DEFAULT '',
  same_gender boolean NOT NULL DEFAULT false,
  gender      text NOT NULL DEFAULT '',
  status      text NOT NULL DEFAULT 'open' CHECK (status IN ('open','filled','closed')),
  match_id    uuid REFERENCES matches(id) ON DELETE SET NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  expires_at  timestamptz NOT NULL
);
CREATE INDEX buddy_requests_board ON buddy_requests (city_key, created_at DESC) WHERE status = 'open';
CREATE INDEX buddy_requests_user ON buddy_requests (user_id, created_at DESC);

CREATE TABLE buddy_offers (
  request_id uuid NOT NULL REFERENCES buddy_requests(id) ON DELETE CASCADE,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (request_id, user_id)
);
CREATE INDEX buddy_offers_user ON buddy_offers (user_id);

-- What a match came from. Set means a plan, so dating coaching and taste learning stay out of it.
ALTER TABLE matches ADD COLUMN buddy_kind text;
