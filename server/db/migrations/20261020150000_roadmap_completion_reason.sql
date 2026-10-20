-- Completion links to verifiable evidence and may be overridden by a
-- human only with a reason (PRODUCT.md). The store, not this column,
-- refuses in_progress → done unless a roadmap_evidence row exists or
-- this one-sentence reason is written on the same update. '' for every
-- row that has not been overridden.
ALTER TABLE insideout.roadmap_nodes
  ADD COLUMN IF NOT EXISTS completion_reason text NOT NULL DEFAULT '';
