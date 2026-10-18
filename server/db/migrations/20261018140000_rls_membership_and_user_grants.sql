-- rv4 wave 2 (DATA-01, DATA-03): the write side of row-level security.
-- Idempotent: functions are CREATE OR REPLACE, policies and triggers are
-- dropped before they are created. Runs as insideout_owner (the migrate
-- runner's role guard).

-- DATA-01. workspace_memberships_insert only checked
-- `user_id = current_user_id()`, so any signed-in session could INSERT itself
-- as an ADMIN of any workspace it knew the id of (workspace ids are returned
-- to members and appear in URLs and exports): admins read and write every
-- row of the workspace. A membership row now needs a reason, and the reason
-- depends on the role:
--   admin  only the workspace's creator, which is what createWorkspaceTx
--          writes right after inserting the workspace;
--   member only a caller holding the workspace's join code in app.join_code,
--          which is what JoinWorkspace sets (and workspaces_select already
--          reads for the same purpose).
-- It lives in a SECURITY DEFINER helper, like _is_member and _is_admin, so
-- reading workspaces from inside a workspace_memberships policy cannot recurse
-- through workspaces_select (see 20260720151500). The column arguments are
-- qualified with the table name for the alias reason in 20260720155500.
CREATE OR REPLACE FUNCTION insideout._may_insert_membership(p_workspace_id uuid, p_user_id uuid, p_role text)
RETURNS boolean
LANGUAGE sql SECURITY DEFINER SET search_path = insideout, pg_catalog STABLE
AS $$
  SELECT EXISTS (
    SELECT 1 FROM insideout.workspaces w
    WHERE w.id = p_workspace_id
      AND (
        (p_role = 'admin' AND w.creator_id = p_user_id)
        OR (p_role = 'member' AND w.status = 'active'
            AND w.code = NULLIF(current_setting('app.join_code', true), ''))
      )
  )
$$;

DROP POLICY IF EXISTS workspace_memberships_insert ON insideout.workspace_memberships;
CREATE POLICY workspace_memberships_insert ON insideout.workspace_memberships
  FOR INSERT WITH CHECK (
    user_id = insideout.current_user_id()
    AND insideout._may_insert_membership(
      insideout.workspace_memberships.workspace_id,
      insideout.current_user_id(),
      insideout.workspace_memberships.role)
  );
