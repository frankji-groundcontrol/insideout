-- B3 attribution (docs/plans/2026-07-24-roadmap-canvas-collab.md, D3/D6/D7/D10).
--
-- created_by / updated_by record who made a roadmap node and who last edited
-- it. Both are nullable on purpose: rows that predate this migration carry no
-- attribution and render an "unknown" initial rather than a fabricated
-- identity (D10). ON DELETE SET NULL keeps the node when its author is removed
-- — a deleted user must not drop their work out of the tree (D6).

ALTER TABLE insideout.roadmap_nodes
  ADD COLUMN created_by uuid REFERENCES insideout.users(id) ON DELETE SET NULL,
  ADD COLUMN updated_by uuid REFERENCES insideout.users(id) ON DELETE SET NULL;

-- Tighten the insert backstop (D7): a node's claimed creator must be the
-- authenticated caller, matching the sibling tables (projects.created_by,
-- project_updates.author_id). The store always sets created_by = actorID
-- inside withUserContext (app.user_id), so legitimate inserts pass; this only
-- rejects a spoofed or NULL created_by that an app-layer bug might otherwise
-- write.
DROP POLICY roadmap_nodes_insert ON insideout.roadmap_nodes;
CREATE POLICY roadmap_nodes_insert ON insideout.roadmap_nodes
  FOR INSERT WITH CHECK (
    created_by = insideout.current_user_id()
    AND EXISTS (
      SELECT 1 FROM insideout.projects p
      JOIN insideout.workspace_memberships m ON m.workspace_id = p.workspace_id
      WHERE p.id = insideout.roadmap_nodes.project_id AND m.user_id = insideout.current_user_id()
    )
  );
