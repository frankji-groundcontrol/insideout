-- Time-first roadmap (PRODUCT.md "Time is a first-class constraint"):
-- an execution item without a deadline cannot enter Now. Nullable —
-- deadlines are commitments, not defaults.
ALTER TABLE insideout.roadmap_nodes ADD COLUMN IF NOT EXISTS deadline timestamptz;
