-- Fix "column reference workspace_id is ambiguous" persisting after
-- 20260720154500. That migration correctly moved cross-table checks
-- into SECURITY DEFINER functions to avoid alias collisions inside
-- their own subqueries, but passed *unqualified* column names (e.g.
-- `workspace_id`) as the function arguments. An unqualified argument is
-- still resolved in the outer, rewritten query's scope — and
-- GetProjectForMember's own JOIN brings both projects.workspace_id and
-- workspace_memberships.workspace_id into that scope, so the bare
-- reference is genuinely ambiguous regardless of it being a function
-- argument rather than a raw subquery. Every argument must be qualified
-- with its owning table's full name (Postgres correctly rebinds a
-- table-name-qualified reference to whichever alias that table actually
-- has in the rewritten query, unlike a bare column name).

DROP POLICY IF EXISTS projects_select ON insideout.projects;
CREATE POLICY projects_select ON insideout.projects
  FOR SELECT USING (insideout._is_member(insideout.projects.workspace_id, insideout.current_user_id()));

DROP POLICY IF EXISTS projects_insert ON insideout.projects;
CREATE POLICY projects_insert ON insideout.projects
  FOR INSERT WITH CHECK (
    created_by = insideout.current_user_id()
    AND insideout._is_member(insideout.projects.workspace_id, insideout.current_user_id())
  );

DROP POLICY IF EXISTS projects_update ON insideout.projects;
CREATE POLICY projects_update ON insideout.projects
  FOR UPDATE USING (
    owner_id = insideout.current_user_id()
    OR insideout._is_admin(insideout.projects.workspace_id, insideout.current_user_id())
  );

DROP POLICY IF EXISTS projects_delete ON insideout.projects;
CREATE POLICY projects_delete ON insideout.projects
  FOR DELETE USING (insideout._is_admin(insideout.projects.workspace_id, insideout.current_user_id()));

DROP POLICY IF EXISTS project_updates_select ON insideout.project_updates;
CREATE POLICY project_updates_select ON insideout.project_updates
  FOR SELECT USING (insideout._is_member(insideout._project_workspace(insideout.project_updates.project_id), insideout.current_user_id()));

DROP POLICY IF EXISTS project_updates_insert ON insideout.project_updates;
CREATE POLICY project_updates_insert ON insideout.project_updates
  FOR INSERT WITH CHECK (
    author_id = insideout.current_user_id()
    AND insideout._is_member(insideout._project_workspace(insideout.project_updates.project_id), insideout.current_user_id())
  );

DROP POLICY IF EXISTS project_updates_update ON insideout.project_updates;
CREATE POLICY project_updates_update ON insideout.project_updates
  FOR UPDATE USING (
    author_id = insideout.current_user_id()
    OR insideout._is_admin(insideout._project_workspace(insideout.project_updates.project_id), insideout.current_user_id())
  );

DROP POLICY IF EXISTS project_updates_delete ON insideout.project_updates;
CREATE POLICY project_updates_delete ON insideout.project_updates
  FOR DELETE USING (
    author_id = insideout.current_user_id()
    OR insideout._is_admin(insideout._project_workspace(insideout.project_updates.project_id), insideout.current_user_id())
  );

DROP POLICY IF EXISTS ideas_select ON insideout.ideas;
CREATE POLICY ideas_select ON insideout.ideas
  FOR SELECT USING (insideout._is_member(insideout.ideas.workspace_id, insideout.current_user_id()));

DROP POLICY IF EXISTS ideas_insert ON insideout.ideas;
CREATE POLICY ideas_insert ON insideout.ideas
  FOR INSERT WITH CHECK (
    author_id = insideout.current_user_id()
    AND insideout._is_member(insideout.ideas.workspace_id, insideout.current_user_id())
  );

DROP POLICY IF EXISTS ideas_update ON insideout.ideas;
CREATE POLICY ideas_update ON insideout.ideas
  FOR UPDATE USING (
    author_id = insideout.current_user_id()
    OR insideout._is_admin(insideout.ideas.workspace_id, insideout.current_user_id())
  );

DROP POLICY IF EXISTS prds_select ON insideout.prds;
CREATE POLICY prds_select ON insideout.prds
  FOR SELECT USING (insideout._is_member(insideout.prds.workspace_id, insideout.current_user_id()));

DROP POLICY IF EXISTS prds_update ON insideout.prds;
CREATE POLICY prds_update ON insideout.prds
  FOR UPDATE USING (
    author_id = insideout.current_user_id()
    OR insideout._is_admin(insideout.prds.workspace_id, insideout.current_user_id())
  );

DROP POLICY IF EXISTS prd_revisions_select ON insideout.prd_revisions;
CREATE POLICY prd_revisions_select ON insideout.prd_revisions
  FOR SELECT USING (insideout._is_member(insideout._prd_workspace(insideout.prd_revisions.prd_id), insideout.current_user_id()));

DROP POLICY IF EXISTS prd_revisions_insert ON insideout.prd_revisions;
CREATE POLICY prd_revisions_insert ON insideout.prd_revisions
  FOR INSERT WITH CHECK (
    created_by = insideout.current_user_id()
    AND insideout._prd_author(insideout.prd_revisions.prd_id) = insideout.current_user_id()
  );

DROP POLICY IF EXISTS agent_messages_all ON insideout.agent_messages;
CREATE POLICY agent_messages_all ON insideout.agent_messages
  FOR ALL
  USING (insideout._conversation_owner(insideout.agent_messages.conversation_id) = insideout.current_user_id())
  WITH CHECK (insideout._conversation_owner(insideout.agent_messages.conversation_id) = insideout.current_user_id());

DROP POLICY IF EXISTS workspaces_select ON insideout.workspaces;
CREATE POLICY workspaces_select ON insideout.workspaces
  FOR SELECT USING (
    creator_id = insideout.current_user_id()
    OR insideout._is_member(insideout.workspaces.id, insideout.current_user_id())
    OR (status = 'active' AND code = NULLIF(current_setting('app.join_code', true), ''))
  );

DROP POLICY IF EXISTS users_select ON insideout.users;
CREATE POLICY users_select ON insideout.users
  FOR SELECT USING (
    insideout.current_user_id() IS NULL
    OR id = insideout.current_user_id()
    OR insideout._shares_workspace(insideout.users.id, insideout.current_user_id())
  );

DROP POLICY IF EXISTS workspaces_update ON insideout.workspaces;
CREATE POLICY workspaces_update ON insideout.workspaces
  FOR UPDATE USING (
    creator_id = insideout.current_user_id()
    OR insideout._is_admin(insideout.workspaces.id, insideout.current_user_id())
  );

-- workspace_memberships policies are currently dormant (NO FORCE ROW
-- LEVEL SECURITY, see 20260720153000 — insideout_app bypasses them
-- entirely as the table owner), so this bare-argument bug never
-- actually triggers today. Fixed anyway for correctness if a future
-- lower-privileged role is ever added.
DROP POLICY IF EXISTS workspace_memberships_select ON insideout.workspace_memberships;
CREATE POLICY workspace_memberships_select ON insideout.workspace_memberships
  FOR SELECT USING (insideout._is_member(insideout.workspace_memberships.workspace_id, insideout.current_user_id()));

DROP POLICY IF EXISTS workspace_memberships_update ON insideout.workspace_memberships;
CREATE POLICY workspace_memberships_update ON insideout.workspace_memberships
  FOR UPDATE USING (insideout._is_admin(insideout.workspace_memberships.workspace_id, insideout.current_user_id()));

DROP POLICY IF EXISTS workspace_memberships_delete ON insideout.workspace_memberships;
CREATE POLICY workspace_memberships_delete ON insideout.workspace_memberships
  FOR DELETE USING (
    user_id = insideout.current_user_id()
    OR insideout._is_admin(insideout.workspace_memberships.workspace_id, insideout.current_user_id())
  );
