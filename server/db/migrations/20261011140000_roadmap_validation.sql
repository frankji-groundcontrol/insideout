-- FEAT-04 / REL-07 (review 3): the loop's last leg. Evidence grows a
-- human-entered validation kind, and a done node can become validated —
-- a milestone a human confirms with a reason (PRODUCT.md: implemented /
-- released / validated; completion links to evidence and is overridden
-- by a human only with a reason). No sync may claim it: the store
-- refuses every generic transition into 'validated'.

-- A validation observation is an evidence row: kind 'validation', the
-- observed text in detail, the provenance in source_url, plus the date
-- the human saw it (webhook kinds never set it; NULL = machine row).
ALTER TABLE insideout.roadmap_evidence
  ADD COLUMN IF NOT EXISTS observed_at date;

-- 'validated' joins the CHECK-constrained machine. The only legal way
-- in is ValidateRoadmapNode (status must be 'done', reason required);
-- reopening to in_progress for the next version stays legal.
ALTER TABLE insideout.roadmap_nodes DROP CONSTRAINT roadmap_nodes_status_check;
ALTER TABLE insideout.roadmap_nodes ADD CONSTRAINT roadmap_nodes_status_check
  CHECK (status IN ('locked', 'pending', 'in_progress', 'done', 'validated'));

-- The validation milestone on the node: who confirmed, when, and the
-- reason they gave. '' / NULL for pre-migration rows; kept on reopen —
-- the next version starts without erasing history (PRODUCT.md item 10).
ALTER TABLE insideout.roadmap_nodes
  ADD COLUMN IF NOT EXISTS validation_reason text NOT NULL DEFAULT '';
ALTER TABLE insideout.roadmap_nodes
  ADD COLUMN IF NOT EXISTS validated_by uuid REFERENCES insideout.users(id) ON DELETE SET NULL;
ALTER TABLE insideout.roadmap_nodes
  ADD COLUMN IF NOT EXISTS validated_at timestamptz;
