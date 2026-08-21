-- Agent vocabulary v1: checkpoints and proposals land in the project
-- timeline as clearly typed records (agents never apply strategic
-- changes — PRODUCT.md "Collaboration and authority").
ALTER TABLE insideout.project_updates DROP CONSTRAINT project_updates_kind_check;
ALTER TABLE insideout.project_updates ADD CONSTRAINT project_updates_kind_check
  CHECK (kind IN ('progress', 'blocker', 'note', 'agent_checkpoint', 'agent_proposal'));
