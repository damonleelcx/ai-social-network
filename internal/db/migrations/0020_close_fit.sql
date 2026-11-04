-- 0020 · A verdict for a pair that's a weaker fit but fine to introduce.

ALTER TABLE hidden.compat DROP CONSTRAINT IF EXISTS compat_verdict_check;
ALTER TABLE hidden.compat ADD CONSTRAINT compat_verdict_check CHECK (verdict IN ('pending','approved','close_fit','rejected'));
