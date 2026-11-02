-- 0011 · Evaluations and scores. Nothing here ever reaches a person.
--
-- A separate schema makes the boundary checkable: internal/archtest fails the
-- build if SQL naming this schema appears outside internal/hidden, and if the
-- API or the conversational agent imports that package.
CREATE SCHEMA hidden;

-- The anonymised pack the Scout assembled for one search.
CREATE TABLE hidden.search_packs (
  goal_id    uuid PRIMARY KEY REFERENCES goals(id) ON DELETE CASCADE,
  viewer_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  pack       jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- One Matchmaker evaluation of one candidate, and the Critic's ruling on it.
CREATE TABLE hidden.compat (
  goal_id         uuid NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  viewer_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  candidate_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  score           double precision NOT NULL CHECK (score BETWEEN 0 AND 100),
  prior           double precision NOT NULL DEFAULT 0,
  reasons         jsonb NOT NULL DEFAULT '[]',
  shared          jsonb NOT NULL DEFAULT '[]',
  pitch_viewer    text NOT NULL DEFAULT '',
  pitch_candidate text NOT NULL DEFAULT '',
  verdict         text NOT NULL DEFAULT 'pending' CHECK (verdict IN ('pending','approved','rejected')),
  critic_notes    text NOT NULL DEFAULT '',
  created_at      timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (goal_id, candidate_id)
);

-- What one person's date said about the other, extracted from their debrief.
CREATE TABLE hidden.evaluations (
  date_id         uuid NOT NULL REFERENCES date_plans(id) ON DELETE CASCADE,
  rater_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  chemistry       double precision NOT NULL CHECK (chemistry BETWEEN 0 AND 10),
  respect         double precision NOT NULL CHECK (respect BETWEEN 0 AND 10),
  conversation    double precision NOT NULL CHECK (conversation BETWEEN 0 AND 10),
  reliability     double precision NOT NULL CHECK (reliability BETWEEN 0 AND 10),
  profile_honesty double precision NOT NULL CHECK (profile_honesty BETWEEN 0 AND 10),
  overall         double precision NOT NULL CHECK (overall BETWEEN 0 AND 10),
  see_again       boolean NOT NULL,
  evidence        jsonb NOT NULL DEFAULT '[]',
  created_at      timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (date_id, rater_id)
);
CREATE INDEX evaluations_subject ON hidden.evaluations (subject_id);
CREATE INDEX evaluations_rater ON hidden.evaluations (rater_id);

CREATE TABLE hidden.reputation (
  user_id    uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  score      double precision NOT NULL DEFAULT 0 CHECK (score BETWEEN -1 AND 1),
  raters     int NOT NULL DEFAULT 0,
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- The learned feature weights behind "your type".
CREATE TABLE hidden.taste (
  user_id          uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  weights          jsonb NOT NULL DEFAULT '{}',
  signals          int NOT NULL DEFAULT 0,
  noted_at_signals int NOT NULL DEFAULT 0,
  updated_at       timestamptz NOT NULL DEFAULT now()
);

-- The behaviour it learned from. key makes a replayed event one signal.
CREATE TABLE hidden.signals (
  id         bigserial PRIMARY KEY,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject_id uuid REFERENCES users(id) ON DELETE CASCADE,
  kind       text NOT NULL,
  label      double precision NOT NULL,
  strength   double precision NOT NULL,
  features   jsonb NOT NULL DEFAULT '{}',
  key        text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX signals_user ON hidden.signals (user_id, id DESC);
