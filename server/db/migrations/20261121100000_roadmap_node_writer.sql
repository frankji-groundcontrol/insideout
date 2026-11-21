-- rv5 row 2.2 (TREEB-02, TREEX-01, TREEB-07): one write rule for the roadmap
-- tree, used by the Go store (requireNodeWriter, server/internal/store/authz.go)
-- and by row-level security, so the two cannot drift apart again.
--
-- A user may write a node when they are the project owner, a workspace admin
-- of the node's workspace, or a current workspace member who owns the node or
-- any ancestor of it (the subtree they were handed). The store adds the fine
-- points the database does not: a delegate may not move, delete or reassign
-- the node they were handed, only what is below it; validation and replacing
-- the whole tree stay with the project owner and admins.
--
-- The functions are SECURITY DEFINER, owned by insideout_owner (BYPASSRLS):
-- a policy on roadmap_nodes that queried roadmap_nodes directly would recurse
-- (SQLSTATE 42P17, BUG-007). STABLE, so a policy sees the tree as it was when
-- the statement started.

CREATE OR REPLACE FUNCTION insideout._is_plan_writer(p_project_id uuid, p_user_id uuid)
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
SET search_path = insideout, pg_temp
STABLE
AS $$
  SELECT EXISTS (
    SELECT 1 FROM insideout.projects p
    WHERE p.id = p_project_id
      AND (p.owner_id = p_user_id OR insideout._is_admin(p.workspace_id, p_user_id))
  )
$$;

CREATE OR REPLACE FUNCTION insideout._can_write_node(p_node_id uuid, p_user_id uuid)
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
SET search_path = insideout, pg_temp
STABLE
AS $$
  WITH RECURSIVE up AS (
    SELECT id, parent_id, project_id, owner_user_id
    FROM insideout.roadmap_nodes WHERE id = p_node_id
    UNION
    SELECT n.id, n.parent_id, n.project_id, n.owner_user_id
    FROM insideout.roadmap_nodes n JOIN up ON n.id = up.parent_id
  )
  SELECT EXISTS (
    SELECT 1 FROM insideout.roadmap_nodes n
    WHERE n.id = p_node_id AND insideout._is_plan_writer(n.project_id, p_user_id)
  ) OR EXISTS (
    SELECT 1 FROM up JOIN insideout.projects p ON p.id = up.project_id
    WHERE up.owner_user_id = p_user_id AND insideout._is_member(p.workspace_id, p_user_id)
  )
$$;

REVOKE ALL ON FUNCTION insideout._is_plan_writer(uuid, uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION insideout._can_write_node(uuid, uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION insideout._is_plan_writer(uuid, uuid) TO insideout_app;
GRANT EXECUTE ON FUNCTION insideout._can_write_node(uuid, uuid) TO insideout_app;

-- Writes follow the rule; SELECT stays workspace-wide (a manager reads every
-- branch). An insert is checked on its parent; a root needs a plan writer.
DROP POLICY IF EXISTS roadmap_nodes_insert ON insideout.roadmap_nodes;
CREATE POLICY roadmap_nodes_insert ON insideout.roadmap_nodes
  FOR INSERT WITH CHECK (
    created_by = insideout.current_user_id()
    AND CASE WHEN parent_id IS NULL
      THEN insideout._is_plan_writer(project_id, insideout.current_user_id())
      ELSE insideout._can_write_node(parent_id, insideout.current_user_id())
    END
  );

DROP POLICY IF EXISTS roadmap_nodes_update ON insideout.roadmap_nodes;
CREATE POLICY roadmap_nodes_update ON insideout.roadmap_nodes
  FOR UPDATE USING (insideout._can_write_node(id, insideout.current_user_id()));

DROP POLICY IF EXISTS roadmap_nodes_delete ON insideout.roadmap_nodes;
CREATE POLICY roadmap_nodes_delete ON insideout.roadmap_nodes
  FOR DELETE USING (insideout._can_write_node(id, insideout.current_user_id()));

-- A delegate attaches evidence (a validation observation) to nodes they may
-- write; the project owner and admins keep the right they had.
DROP POLICY IF EXISTS evidence_owner_insert ON insideout.roadmap_evidence;
CREATE POLICY evidence_owner_insert ON insideout.roadmap_evidence
  FOR INSERT WITH CHECK (insideout._can_write_node(node_id, insideout.current_user_id()));
