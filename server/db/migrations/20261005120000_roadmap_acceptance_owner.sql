-- Gap slices 16/17 (multirole review of the planning flow): done becomes
-- checkable and owned. Each node carries the sentence that says what done
-- means, and a Now execution item has exactly one human owner.
-- Acceptance defaults to '' (existing rows carry no acceptance); the owner
-- is nullable — pre-migration rows and removed users (ON DELETE SET NULL)
-- leave it unset.
ALTER TABLE insideout.roadmap_nodes ADD COLUMN IF NOT EXISTS acceptance text NOT NULL DEFAULT '';
ALTER TABLE insideout.roadmap_nodes ADD COLUMN IF NOT EXISTS owner_user_id uuid REFERENCES insideout.users(id) ON DELETE SET NULL;
