-- 0012 · Characters of Suki's voice per account per day, for the daily allowance.
CREATE TABLE tts_usage (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  day     date NOT NULL,
  chars   int NOT NULL DEFAULT 0,
  PRIMARY KEY (user_id, day)
);
