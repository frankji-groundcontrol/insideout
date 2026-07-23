-- Fix "infinite recursion detected in policy for relation
-- workspace_memberships" (SQLSTATE 42P17), found by running the real
-- authorization-checklist integration tests against a live database.
--
-- workspace_memberships_select/update/delete each need to answer "is the
-- current user ALSO a member of this row's workspace" — which requires
-- querying workspace_memberships from within its OWN policy. Postgres
-- detects that self-reference as a structural cycle at query-rewrite time
-- and refuses to plan it at all (this is not a hypothetical edge case —
-- it errors on the very first SELECT). Cross-table references (e.g.
-- projects_select querying workspace_memberships) don't have this
-- problem; only a table's own policy referencing itself does.
--
-- Standard fix: move the self-check into a SECURITY DEFINER function.
-- Wrapping it in a function stops the planner from inlining the
-- self-referential subquery into the same query plan it's already
-- rewriting, which is what triggers the cycle detection — confirmed
-- empirically against this database before writing this migration.

CREATE OR REPLACE FUNCTION insideout._is_member(p_workspace_id uuid, p_user_id uuid)
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
SET search_path = insideout, pg_catalog
STABLE
AS $$
  SELECT EXISTS (
    SELECT 1 FROM insideout.workspace_memberships
    WHERE workspace_id = p_workspace_id AND user_id = p_user_id
  )
$$;

CREATE OR REPLACE FUNCTION insideout._is_admin(p_workspace_id uuid, p_user_id uuid)
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
SET search_path = insideout, pg_catalog
STABLE
AS $$
  SELECT EXISTS (
    SELECT 1 FROM insideout.workspace_memberships
    WHERE workspace_id = p_workspace_id AND user_id = p_user_id AND role = 'admin'
  )
$$;

DROP POLICY IF EXISTS workspace_memberships_select ON insideout.workspace_memberships;
CREATE POLICY workspace_memberships_select ON insideout.workspace_memberships
  FOR SELECT USING (insideout._is_member(workspace_id, insideout.current_user_id()));

DROP POLICY IF EXISTS workspace_memberships_update ON insideout.workspace_memberships;
CREATE POLICY workspace_memberships_update ON insideout.workspace_memberships
  FOR UPDATE USING (insideout._is_admin(workspace_id, insideout.current_user_id()));

DROP POLICY IF EXISTS workspace_memberships_delete ON insideout.workspace_memberships;
CREATE POLICY workspace_memberships_delete ON insideout.workspace_memberships
  FOR DELETE USING (
    user_id = insideout.current_user_id()
    OR insideout._is_admin(workspace_id, insideout.current_user_id())
  );
