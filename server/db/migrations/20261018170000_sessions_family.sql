-- rv4-SEC-04: a refresh token is one-use, but presenting a spent one only
-- refused that request; the newest token of the same login kept working, so a
-- thief who rotated first outlived the owner noticing. family_id names the
-- lineage: the id of the session a login created, carried unchanged to every
-- session rotated from it, so reuse of any spent token can end the whole
-- lineage at once.
--
-- NULL = a session written before this column existed. The code reads it as a
-- family of its own id (COALESCE(family_id, id)) and the first rotation after
-- the upgrade starts the family, so there is no backfill of a table that
-- grows by one row per refresh, and no NOT NULL for old rows to violate.
-- No foreign key either: the value is a lineage name, and retention deletes
-- the oldest session of a lineage long before its newest.
ALTER TABLE insideout.sessions
  ADD COLUMN IF NOT EXISTS family_id uuid;

-- Ending a lineage reads its live rows by family_id.
CREATE INDEX IF NOT EXISTS sessions_family_id_idx
  ON insideout.sessions (family_id);
