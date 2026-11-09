-- 0026 · What Suki thinks someone is looking for, in her words.
--
-- One row a person. `lines` is what the card shows. `removed` is what they took out, so she
-- doesn't say it again. These are her reading of what they told her: nothing here filters
-- matches, and only the person sees it.

CREATE TABLE looking_reads (
  user_id     uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  lines       jsonb NOT NULL DEFAULT '[]',
  removed     jsonb NOT NULL DEFAULT '[]',
  source_hash text NOT NULL DEFAULT '',
  shown_at    timestamptz,
  updated_at  timestamptz NOT NULL DEFAULT now()
);
