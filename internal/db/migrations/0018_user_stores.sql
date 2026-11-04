-- 0018 · Which store apps a person has used. Credits can only be bought through a store from inside its app.

CREATE TABLE user_stores (
  user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  store         text NOT NULL CHECK (store IN ('play','wechat')),
  first_seen_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, store)
);
