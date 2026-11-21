-- rv5 row 2.6 (TREEB-01, TREEB-09, TREEX-08): delegating a node is recorded.
--
-- roadmap_nodes.owner_user_id stays the current responsible person; this
-- append-only table records every change of it: who held the node before
-- (from), who holds it after (to), who made the change (assigned_by), a note,
-- and when. kind is 'delegate' (a new owner), 'handback' (the owner returned
-- it to whoever delegated it) or 'cleared' (no owner any more, including a
-- member who left the workspace or erased their account).
CREATE TABLE insideout.roadmap_node_assignments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  node_id uuid NOT NULL REFERENCES insideout.roadmap_nodes(id) ON DELETE CASCADE,
  from_user_id uuid REFERENCES insideout.users(id) ON DELETE SET NULL,
  to_user_id uuid REFERENCES insideout.users(id) ON DELETE SET NULL,
  assigned_by uuid REFERENCES insideout.users(id) ON DELETE SET NULL,
  kind text NOT NULL CHECK (kind IN ('delegate', 'handback', 'cleared')),
  note text NOT NULL DEFAULT '' CHECK (char_length(note) <= 500),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.roadmap_node_assignments (node_id, created_at);
CREATE INDEX ON insideout.roadmap_node_assignments (from_user_id);
CREATE INDEX ON insideout.roadmap_node_assignments (to_user_id);
CREATE INDEX ON insideout.roadmap_node_assignments (assigned_by);

ALTER TABLE insideout.roadmap_node_assignments ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.roadmap_node_assignments FORCE ROW LEVEL SECURITY;

-- Every member of the node's workspace reads the history (owner decision 2:
-- a default view, not a wall). A row is written in the actor's own name, by
-- a current member; there is no UPDATE or DELETE policy, so the history is
-- append-only for the app role. _node_project is SECURITY DEFINER, so the
-- policy never queries roadmap_nodes under its own RLS (BUG-007).
CREATE POLICY roadmap_node_assignments_select ON insideout.roadmap_node_assignments
  FOR SELECT USING (
    insideout._is_member(insideout._project_workspace(insideout._node_project(node_id)), insideout.current_user_id())
  );

CREATE POLICY roadmap_node_assignments_insert ON insideout.roadmap_node_assignments
  FOR INSERT WITH CHECK (
    assigned_by = insideout.current_user_id()
    AND insideout._is_member(insideout._project_workspace(insideout._node_project(node_id)), insideout.current_user_id())
  );

GRANT SELECT, INSERT ON insideout.roadmap_node_assignments TO insideout_app;

-- Owners set before the history existed get one 'delegate' row each, by an
-- unknown hand, so a hand-back finds where the node came from (nowhere).
INSERT INTO insideout.roadmap_node_assignments (node_id, to_user_id, kind)
SELECT id, owner_user_id, 'delegate' FROM insideout.roadmap_nodes WHERE owner_user_id IS NOT NULL;

-- The project feed can name a node, and carries delegation, hand-back and
-- completion, so a delegator sees what became of what they handed out.
ALTER TABLE insideout.project_updates
  ADD COLUMN node_id uuid REFERENCES insideout.roadmap_nodes(id) ON DELETE SET NULL;
CREATE INDEX ON insideout.project_updates (node_id);

ALTER TABLE insideout.project_updates DROP CONSTRAINT project_updates_kind_check;
ALTER TABLE insideout.project_updates ADD CONSTRAINT project_updates_kind_check
  CHECK (kind IN ('progress', 'blocker', 'note', 'agent_checkpoint', 'agent_proposal',
                  'node_delegated', 'node_handed_back', 'node_done'));
