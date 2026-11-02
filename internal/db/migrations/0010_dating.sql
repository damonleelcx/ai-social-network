-- 0010 · The dating domain, visible side. Scores live in schema hidden (0011).
-- Photo bytes live in S3 and chat messages in MongoDB; these are the metadata.

CREATE TABLE profiles (
  user_id        uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  display_name   text NOT NULL DEFAULT '',
  birthdate      date,
  gender         text NOT NULL DEFAULT '',
  pronouns       text NOT NULL DEFAULT '',
  seeking        text[] NOT NULL DEFAULT '{}',
  city_key       text NOT NULL DEFAULT '',
  city_name      text NOT NULL DEFAULT '',
  city_lat       double precision,
  city_lon       double precision,
  age_min        int NOT NULL DEFAULT 18 CHECK (age_min >= 18),
  age_max        int NOT NULL DEFAULT 99 CHECK (age_max <= 120),
  goal           text NOT NULL DEFAULT '',
  bio            text NOT NULL DEFAULT '',
  bio_draft      text NOT NULL DEFAULT '',
  occupation     text NOT NULL DEFAULT '',
  education      text NOT NULL DEFAULT '',
  height_cm      int,
  smoking        text NOT NULL DEFAULT '',
  drinking       text NOT NULL DEFAULT '',
  kids           text NOT NULL DEFAULT '',
  wants_kids     text NOT NULL DEFAULT '',
  religion       text NOT NULL DEFAULT '',
  politics       text NOT NULL DEFAULT '',
  languages      text[] NOT NULL DEFAULT '{}',
  interests      text[] NOT NULL DEFAULT '{}',
  prompts        jsonb NOT NULL DEFAULT '[]',
  dealbreakers   jsonb NOT NULL DEFAULT '{}',
  -- onboarding: not proposed yet. active: proposed. paused: the person's choice.
  -- held: a moderator must look first (e.g. said they are under 18).
  status         text NOT NULL DEFAULT 'onboarding' CHECK (status IN ('onboarding','active','paused','held')),
  completed_at   timestamptz,
  last_active_at timestamptz NOT NULL DEFAULT now(),
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX profiles_city ON profiles (city_key, status);

CREATE TABLE photos (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  s3_key     text NOT NULL,
  mime       text NOT NULL,
  width      int NOT NULL DEFAULT 0,
  height     int NOT NULL DEFAULT 0,
  sha256     text NOT NULL,
  status     text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected','review')),
  reason     text NOT NULL DEFAULT '',
  tips       text NOT NULL DEFAULT '',
  is_primary boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, sha256)
);
CREATE INDEX photos_user ON photos (user_id, created_at);
CREATE UNIQUE INDEX photos_one_primary ON photos (user_id) WHERE is_primary;

-- One running conversation with Suki per person.
CREATE UNIQUE INDEX conversations_one_per_user ON conversations (user_id);

-- The queue behind a turn with Suki. Same discipline as tasks: skip locked,
-- lease, fencing epoch, and a key so a retried POST or timer is one turn.
-- message_seq is the user message in MongoDB this turn answers.
CREATE TABLE agent_turns (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  conversation_id  uuid NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  kind             text NOT NULL CHECK (kind IN ('message','system')),
  message_seq      bigint NOT NULL DEFAULT 0,
  prompt           text NOT NULL DEFAULT '',
  key              text NOT NULL UNIQUE,
  status           text NOT NULL DEFAULT 'ready' CHECK (status IN ('ready','leased','done','failed')),
  attempts         int NOT NULL DEFAULT 0,
  run_after        timestamptz NOT NULL DEFAULT now(),
  lease_owner      text,
  lease_epoch      bigint NOT NULL DEFAULT 0,
  lease_expires_at timestamptz,
  error            text NOT NULL DEFAULT '',
  intent           text NOT NULL DEFAULT '',
  created_at       timestamptz NOT NULL DEFAULT now(),
  finished_at      timestamptz
);
CREATE INDEX agent_turns_ready ON agent_turns (run_after) WHERE status = 'ready';
CREATE INDEX agent_turns_leased ON agent_turns (lease_expires_at) WHERE status = 'leased';

-- A pair Suki put to both people. user_a < user_b, so a pair is one row and can
-- only ever be proposed once.
CREATE TABLE proposals (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  pitch_a     text NOT NULL DEFAULT '',
  pitch_b     text NOT NULL DEFAULT '',
  shared      jsonb NOT NULL DEFAULT '[]',
  decision_a  text CHECK (decision_a IN ('accept','pass')),
  decision_b  text CHECK (decision_b IN ('accept','pass')),
  status      text NOT NULL DEFAULT 'open' CHECK (status IN ('open','matched','declined','expired')),
  goal_id     uuid,
  created_at  timestamptz NOT NULL DEFAULT now(),
  expires_at  timestamptz NOT NULL DEFAULT now() + interval '7 days',
  CHECK (user_a < user_b),
  UNIQUE (user_a, user_b)
);
CREATE INDEX proposals_a ON proposals (user_a, status);
CREATE INDEX proposals_b ON proposals (user_b, status);

CREATE TABLE matches (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  proposal_id      uuid NOT NULL UNIQUE REFERENCES proposals(id) ON DELETE CASCADE,
  user_a           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  stage            text NOT NULL DEFAULT 'icebreaker' CHECK (stage IN ('icebreaker','getting_to_know','date_ready','planning','planned','post_date')),
  status           text NOT NULL DEFAULT 'active' CHECK (status IN ('active','unmatched')),
  created_at       timestamptz NOT NULL DEFAULT now(),
  last_message_at  timestamptz,
  last_preview     text NOT NULL DEFAULT '',
  message_count    int NOT NULL DEFAULT 0,
  coached_at_count int NOT NULL DEFAULT 0,
  stall_nudged_at  timestamptz
);
CREATE INDEX matches_a ON matches (user_a, status);
CREATE INDEX matches_b ON matches (user_b, status);

-- Suki's private word to one person about one conversation.
CREATE TABLE coach_hints (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id     uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body         text NOT NULL,
  suggestions  jsonb NOT NULL DEFAULT '[]',
  key          text NOT NULL UNIQUE,
  dismissed_at timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX coach_hints_open ON coach_hints (match_id, user_id) WHERE dismissed_at IS NULL;

CREATE TABLE date_plans (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id     uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  created_by   uuid REFERENCES users(id) ON DELETE SET NULL,
  status       text NOT NULL DEFAULT 'searching' CHECK (status IN ('searching','voting','scheduling','confirmed','completed','cancelled','failed')),
  constraints  jsonb NOT NULL DEFAULT '[]',
  candidates   jsonb NOT NULL DEFAULT '[]',
  venues       jsonb NOT NULL DEFAULT '[]',
  votes        jsonb NOT NULL DEFAULT '{}',
  venue        jsonb,
  proposed_at  timestamptz,
  proposed_by  uuid REFERENCES users(id) ON DELETE SET NULL,
  scheduled_at timestamptz,
  reminded_24  boolean NOT NULL DEFAULT false,
  reminded_2   boolean NOT NULL DEFAULT false,
  checkin_sent boolean NOT NULL DEFAULT false,
  debrief_sent boolean NOT NULL DEFAULT false,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX date_plans_one_open ON date_plans (match_id) WHERE status IN ('searching','voting','scheduling','confirmed');
CREATE INDEX date_plans_due ON date_plans (scheduled_at) WHERE status = 'confirmed';

-- What one person told Suki about one date. Private to Suki's analysts.
CREATE TABLE debriefs (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  date_id    uuid NOT NULL REFERENCES date_plans(id) ON DELETE CASCADE,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject_id uuid REFERENCES users(id) ON DELETE SET NULL,
  notes      jsonb NOT NULL DEFAULT '[]',
  status     text NOT NULL DEFAULT 'open' CHECK (status IN ('open','closed','analyzed','skipped')),
  created_at timestamptz NOT NULL DEFAULT now(),
  closed_at  timestamptz,
  UNIQUE (date_id, user_id)
);

CREATE TABLE blocks (
  blocker_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (blocker_id, blocked_id)
);
CREATE INDEX blocks_blocked ON blocks (blocked_id);

CREATE TABLE reports (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id uuid REFERENCES users(id) ON DELETE SET NULL,
  subject_id  uuid REFERENCES users(id) ON DELETE SET NULL,
  match_id    uuid REFERENCES matches(id) ON DELETE SET NULL,
  message_seq bigint,
  source      text NOT NULL DEFAULT 'user' CHECK (source IN ('user','suki','sentinel','debrief','onboarding')),
  reason      text NOT NULL,
  details     text NOT NULL DEFAULT '',
  severity    text NOT NULL DEFAULT 'normal' CHECK (severity IN ('normal','high','critical')),
  status      text NOT NULL DEFAULT 'open' CHECK (status IN ('open','resolved')),
  action      text NOT NULL DEFAULT '',
  resolved_by uuid REFERENCES users(id) ON DELETE SET NULL,
  key         text UNIQUE,
  created_at  timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz
);
CREATE INDEX reports_open ON reports (created_at) WHERE status = 'open';

CREATE TABLE notifications (
  id         bigserial PRIMARY KEY,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind       text NOT NULL,
  title      text NOT NULL,
  body       text NOT NULL DEFAULT '',
  data       jsonb NOT NULL DEFAULT '{}',
  key        text UNIQUE,
  read_at    timestamptz,
  pushed_at  timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notifications_user ON notifications (user_id, id DESC);
CREATE INDEX notifications_unpushed ON notifications (id) WHERE pushed_at IS NULL;

CREATE TABLE push_tokens (
  token      text PRIMARY KEY,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  platform   text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX push_tokens_user ON push_tokens (user_id);

CREATE TABLE cached_lookups (
  key        text PRIMARY KEY,
  data       jsonb NOT NULL,
  fetched_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE feedback (
  id         bigserial PRIMARY KEY,
  user_id    uuid REFERENCES users(id) ON DELETE SET NULL,
  body       text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
