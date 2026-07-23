-- Fix "column reference ... is ambiguous" (SQLSTATE 42702), found by
-- running the authorization-checklist integration tests against a live
-- database. Postgres merges an RLS policy's USING/WITH CHECK expression
-- textually into whatever query touches the table — so a policy's own
-- subquery alias (e.g. `m` for workspace_memberships) collides whenever
-- the *application* query happens to alias a table the same way
-- (GetProjectForMember joins workspace_memberships AS m too). This
-- isn't a one-off — nearly every policy in 20260720150000 has this
-- shape, so instead of chasing individual collisions, every cross-table
-- policy expression is rewritten to call a SECURITY DEFINER function
-- instead of an inline EXISTS/JOIN subquery. A function call introduces
-- no alias into the outer query at all, so there's nothing left to
-- collide — this is the same fix already proven for workspaces_select's
-- FOR KEY SHARE issue in 20260720151500, applied uniformly everywhere.

CREATE OR REPLACE FUNCTION insideout._project_workspace(p_project_id uuid)
RETURNS uuid LANGUAGE sql SECURITY DEFINER SET search_path = insideout, pg_catalog STABLE
AS $$ SELECT workspace_id FROM insideout.projects WHERE id = p_project_id $$;

CREATE OR REPLACE FUNCTION insideout._prd_workspace(p_prd_id uuid)
RETURNS uuid LANGUAGE sql SECURITY DEFINER SET search_path = insideout, pg_catalog STABLE
AS $$ SELECT workspace_id FROM insideout.prds WHERE id = p_prd_id $$;

CREATE OR REPLACE FUNCTION insideout._prd_author(p_prd_id uuid)
RETURNS uuid LANGUAGE sql SECURITY DEFINER SET search_path = insideout, pg_catalog STABLE
AS $$ SELECT author_id FROM insideout.prds WHERE id = p_prd_id $$;

CREATE OR REPLACE FUNCTION insideout._shares_workspace(p_user_a uuid, p_user_b uuid)
RETURNS boolean LANGUAGE sql SECURITY DEFINER SET search_path = insideout, pg_catalog STABLE
AS $$
  SELECT EXISTS (
    SELECT 1 FROM insideout.workspace_memberships wa
    JOIN insideout.workspace_memberships wb ON wa.workspace_id = wb.workspace_id
    WHERE wa.user_id = p_user_a AND wb.user_id = p_user_b
  )
$$;

-- users
DROP POLICY IF EXISTS users_select ON insideout.users;
CREATE POLICY users_select ON insideout.users
  FOR SELECT USING (
    insideout.current_user_id() IS NULL
    OR id = insideout.current_user_id()
    OR insideout._shares_workspace(id, insideout.current_user_id())
  );

-- projects
DROP POLICY IF EXISTS projects_select ON insideout.projects;
CREATE POLICY projects_select ON insideout.projects
  FOR SELECT USING (insideout._is_member(workspace_id, insideout.current_user_id()));

DROP POLICY IF EXISTS projects_insert ON insideout.projects;
CREATE POLICY projects_insert ON insideout.projects
  FOR INSERT WITH CHECK (
    created_by = insideout.current_user_id()
    AND insideout._is_member(workspace_id, insideout.current_user_id())
  );

DROP POLICY IF EXISTS projects_update ON insideout.projects;
CREATE POLICY projects_update ON insideout.projects
  FOR UPDATE USING (
    owner_id = insideout.current_user_id()
    OR insideout._is_admin(workspace_id, insideout.current_user_id())
  );

DROP POLICY IF EXISTS projects_delete ON insideout.projects;
CREATE POLICY projects_delete ON insideout.projects
  FOR DELETE USING (insideout._is_admin(workspace_id, insideout.current_user_id()));

-- project_updates
DROP POLICY IF EXISTS project_updates_select ON insideout.project_updates;
CREATE POLICY project_updates_select ON insideout.project_updates
  FOR SELECT USING (insideout._is_member(insideout._project_workspace(project_id), insideout.current_user_id()));

DROP POLICY IF EXISTS project_updates_insert ON insideout.project_updates;
CREATE POLICY project_updates_insert ON insideout.project_updates
  FOR INSERT WITH CHECK (
    author_id = insideout.current_user_id()
    AND insideout._is_member(insideout._project_workspace(project_id), insideout.current_user_id())
  );

DROP POLICY IF EXISTS project_updates_update ON insideout.project_updates;
CREATE POLICY project_updates_update ON insideout.project_updates
  FOR UPDATE USING (
    author_id = insideout.current_user_id()
    OR insideout._is_admin(insideout._project_workspace(project_id), insideout.current_user_id())
  );

DROP POLICY IF EXISTS project_updates_delete ON insideout.project_updates;
CREATE POLICY project_updates_delete ON insideout.project_updates
  FOR DELETE USING (
    author_id = insideout.current_user_id()
    OR insideout._is_admin(insideout._project_workspace(project_id), insideout.current_user_id())
  );

-- ideas
DROP POLICY IF EXISTS ideas_select ON insideout.ideas;
CREATE POLICY ideas_select ON insideout.ideas
  FOR SELECT USING (insideout._is_member(workspace_id, insideout.current_user_id()));

DROP POLICY IF EXISTS ideas_insert ON insideout.ideas;
CREATE POLICY ideas_insert ON insideout.ideas
  FOR INSERT WITH CHECK (
    author_id = insideout.current_user_id()
    AND insideout._is_member(workspace_id, insideout.current_user_id())
  );

DROP POLICY IF EXISTS ideas_update ON insideout.ideas;
CREATE POLICY ideas_update ON insideout.ideas
  FOR UPDATE USING (
    author_id = insideout.current_user_id()
    OR insideout._is_admin(workspace_id, insideout.current_user_id())
  );

-- prds
DROP POLICY IF EXISTS prds_select ON insideout.prds;
CREATE POLICY prds_select ON insideout.prds
  FOR SELECT USING (insideout._is_member(workspace_id, insideout.current_user_id()));

DROP POLICY IF EXISTS prds_update ON insideout.prds;
CREATE POLICY prds_update ON insideout.prds
  FOR UPDATE USING (
    author_id = insideout.current_user_id()
    OR insideout._is_admin(workspace_id, insideout.current_user_id())
  );

-- prd_revisions
DROP POLICY IF EXISTS prd_revisions_select ON insideout.prd_revisions;
CREATE POLICY prd_revisions_select ON insideout.prd_revisions
  FOR SELECT USING (insideout._is_member(insideout._prd_workspace(prd_id), insideout.current_user_id()));

DROP POLICY IF EXISTS prd_revisions_insert ON insideout.prd_revisions;
CREATE POLICY prd_revisions_insert ON insideout.prd_revisions
  FOR INSERT WITH CHECK (
    created_by = insideout.current_user_id()
    AND insideout._prd_author(prd_id) = insideout.current_user_id()
  );

-- agent_messages (agent_conversations_all/ai_runs_all have no cross-table
-- reference at all, so they're not at risk — only agent_messages_all is)
CREATE OR REPLACE FUNCTION insideout._conversation_owner(p_conversation_id uuid)
RETURNS uuid LANGUAGE sql SECURITY DEFINER SET search_path = insideout, pg_catalog STABLE
AS $$ SELECT user_id FROM insideout.agent_conversations WHERE id = p_conversation_id $$;

DROP POLICY IF EXISTS agent_messages_all ON insideout.agent_messages;
CREATE POLICY agent_messages_all ON insideout.agent_messages
  FOR ALL
  USING (insideout._conversation_owner(conversation_id) = insideout.current_user_id())
  WITH CHECK (insideout._conversation_owner(conversation_id) = insideout.current_user_id());
