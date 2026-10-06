-- Proposal items were ordered by created_at, but a batch of inserts in
-- one statement shares one now(), so the id tie-break reordered them
-- when a human applied the proposal (BUG-019). Position is the order
-- the proposer named, assigned at insert.
ALTER TABLE insideout.proposal_items
  ADD COLUMN IF NOT EXISTS position integer NOT NULL DEFAULT 0;