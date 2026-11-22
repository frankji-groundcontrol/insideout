-- 2026-11-22 roadmap canvas plan, "Dependency contract": a node may wait on
-- other nodes of its project, across branches, so parallel work merges.
--
-- node_id waits on depends_on_id. The store refuses a link between a node and
-- its own ancestor or descendant (the tree already says that) and a link that
-- closes a cycle; both run behind the per-project advisory lock. Deleting
-- either node, or replacing the whole tree, drops the link.
CREATE TABLE insideout.roadmap_dependencies (
  node_id uuid NOT NULL REFERENCES insideout.roadmap_nodes(id) ON DELETE CASCADE,
  depends_on_id uuid NOT NULL REFERENCES insideout.roadmap_nodes(id) ON DELETE CASCADE,
  created_by uuid REFERENCES insideout.users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (node_id, depends_on_id),
  CHECK (node_id <> depends_on_id)
);

CREATE INDEX ON insideout.roadmap_dependencies (depends_on_id);
CREATE INDEX ON insideout.roadmap_dependencies (created_by);

ALTER TABLE insideout.roadmap_dependencies ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.roadmap_dependencies FORCE ROW LEVEL SECURITY;

-- Every member of the node's workspace reads the links (owner decision 2). A
-- link is written and removed by a writer of the waiting node, in the actor's
-- own name, and both ends sit in one project. The functions are SECURITY
-- DEFINER, so no policy here queries roadmap_nodes under its own RLS (BUG-007).
CREATE POLICY roadmap_dependencies_select ON insideout.roadmap_dependencies
  FOR SELECT USING (
    insideout._is_member(insideout._project_workspace(insideout._node_project(node_id)), insideout.current_user_id())
  );

CREATE POLICY roadmap_dependencies_insert ON insideout.roadmap_dependencies
  FOR INSERT WITH CHECK (
    created_by = insideout.current_user_id()
    AND insideout._can_write_node(node_id, insideout.current_user_id())
    AND insideout._node_project(depends_on_id) = insideout._node_project(node_id)
  );

CREATE POLICY roadmap_dependencies_delete ON insideout.roadmap_dependencies
  FOR DELETE USING (
    insideout._can_write_node(node_id, insideout.current_user_id())
  );

GRANT SELECT, INSERT, DELETE ON insideout.roadmap_dependencies TO insideout_app;
