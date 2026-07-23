-- Idea → reality pivot (docs/plans/2026-07-22-idea-to-reality.md).
--
-- roadmap_nodes: a branched tree on a project. parent_id self-FK forms the
-- hierarchy — a root (the product/MVP) whose children are parallel
-- workstreams, each branchable further. Siblings share a parent_id and are
-- ordered by position. status drives the build state machine.
--
-- projects.repo_url: the GitHub repo a project syncs progress from (M4).

CREATE TABLE insideout.roadmap_nodes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id uuid NOT NULL REFERENCES insideout.projects(id) ON DELETE CASCADE,
  parent_id uuid REFERENCES insideout.roadmap_nodes(id) ON DELETE CASCADE,
  title text NOT NULL,
  description text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'pending'
    CHECK (status IN ('locked', 'pending', 'in_progress', 'done')),
  position int NOT NULL DEFAULT 0,
  meta jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.roadmap_nodes (project_id, parent_id, position);

CREATE TRIGGER set_updated_at
  BEFORE UPDATE ON insideout.roadmap_nodes
  FOR EACH ROW EXECUTE FUNCTION insideout.set_updated_at();

ALTER TABLE insideout.projects ADD COLUMN repo_url text NOT NULL DEFAULT '';

-- RLS: defense-in-depth backstop mirroring project_updates — a workspace
-- member of the node's project may read and write (the roadmap is
-- collaborative). FORCE is required because insideout_app owns the table.
ALTER TABLE insideout.roadmap_nodes ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.roadmap_nodes FORCE ROW LEVEL SECURITY;

CREATE POLICY roadmap_nodes_select ON insideout.roadmap_nodes
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM insideout.projects p
      JOIN insideout.workspace_memberships m ON m.workspace_id = p.workspace_id
      WHERE p.id = insideout.roadmap_nodes.project_id AND m.user_id = insideout.current_user_id()
    )
  );

CREATE POLICY roadmap_nodes_insert ON insideout.roadmap_nodes
  FOR INSERT WITH CHECK (
    EXISTS (
      SELECT 1 FROM insideout.projects p
      JOIN insideout.workspace_memberships m ON m.workspace_id = p.workspace_id
      WHERE p.id = insideout.roadmap_nodes.project_id AND m.user_id = insideout.current_user_id()
    )
  );

CREATE POLICY roadmap_nodes_update ON insideout.roadmap_nodes
  FOR UPDATE USING (
    EXISTS (
      SELECT 1 FROM insideout.projects p
      JOIN insideout.workspace_memberships m ON m.workspace_id = p.workspace_id
      WHERE p.id = insideout.roadmap_nodes.project_id AND m.user_id = insideout.current_user_id()
    )
  );

CREATE POLICY roadmap_nodes_delete ON insideout.roadmap_nodes
  FOR DELETE USING (
    EXISTS (
      SELECT 1 FROM insideout.projects p
      JOIN insideout.workspace_memberships m ON m.workspace_id = p.workspace_id
      WHERE p.id = insideout.roadmap_nodes.project_id AND m.user_id = insideout.current_user_id()
    )
  );
