-- 0021 · Group plans and trips.
--
-- A buddy post with more than one spot is a group: people who say they're in join at once,
-- up to the number of spots, and everyone in it shares one group chat (in MongoDB, thread
-- "group:<post id>"). A post with one spot still works as before: the poster picks one person.
--
-- A trip is someone saying they'll be in another city for some days. While a trip is on or
-- about to start, plans posted in that city reach them too, and they can post plans there.

CREATE TABLE trips (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  city_key    text NOT NULL,
  city_name   text NOT NULL DEFAULT '',
  lat         double precision NOT NULL DEFAULT 0,
  lon         double precision NOT NULL DEFAULT 0,
  starts_on   date NOT NULL,
  ends_on     date NOT NULL,
  note        text NOT NULL DEFAULT '',
  created_at  timestamptz NOT NULL DEFAULT now(),
  CHECK (ends_on >= starts_on)
);
CREATE INDEX trips_city ON trips (city_key, ends_on);
CREATE INDEX trips_user ON trips (user_id, ends_on);

ALTER TABLE buddy_requests ADD COLUMN spots int NOT NULL DEFAULT 1 CHECK (spots BETWEEN 1 AND 20);
ALTER TABLE buddy_requests ADD COLUMN place text NOT NULL DEFAULT '';
ALTER TABLE buddy_requests ADD COLUMN trip_id uuid REFERENCES trips(id) ON DELETE SET NULL;

-- Who has joined a group plan. The poster isn't a row here: they're always in it.
CREATE TABLE buddy_members (
  request_id uuid NOT NULL REFERENCES buddy_requests(id) ON DELETE CASCADE,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  joined_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (request_id, user_id)
);
CREATE INDEX buddy_members_user ON buddy_members (user_id);
