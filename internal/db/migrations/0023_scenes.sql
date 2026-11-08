-- 0023 · Pictures of what plans and trips are about, drawn once per scene and city and shared
-- by every plan that is the same idea (internal/scenes). They show no one: only the scene.

CREATE TABLE scene_images (
  key        text PRIMARY KEY,
  scene      text NOT NULL,
  city_key   text NOT NULL DEFAULT '',
  city_name  text NOT NULL DEFAULT '',
  status     text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','ready','failed')),
  attempts   int NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX scene_images_day ON scene_images (created_at);

ALTER TABLE buddy_requests ADD COLUMN scene_key text;
ALTER TABLE trips ADD COLUMN scene_key text;
