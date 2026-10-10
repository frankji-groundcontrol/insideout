-- SEC-11, the rest of it: a PRD's, a project update's and a PRD snapshot's
-- author keeps no rights over them once removed from the workspace.
-- 20261010120000 made the author branch of ideas_select, ideas_update and
-- prds_insert need current membership; four siblings still passed on
-- author_id (or created_by) alone, so a removed member could still edit the
-- PRDs they wrote, edit or delete their project updates, and snapshot their
-- PRDs. The author branch now also needs current membership; the admin branch
-- already implies it.
--
-- Same shape as 20261010120000: membership comes from the SECURITY DEFINER
-- helpers with table-qualified arguments, never an inline subquery on
-- workspace_memberships. prds_update and project_updates_update have no WITH
-- CHECK, so Postgres reuses their USING expression for the new row.

DROP POLICY IF EXISTS prds_update ON insideout.prds;
CREATE POLICY prds_update ON insideout.prds
  FOR UPDATE USING (
    (
      author_id = insideout.current_user_id()
      AND insideout._is_member(insideout.prds.workspace_id, insideout.current_user_id())
    )
    OR insideout._is_admin(insideout.prds.workspace_id, insideout.current_user_id())
  );

DROP POLICY IF EXISTS project_updates_update ON insideout.project_updates;
CREATE POLICY project_updates_update ON insideout.project_updates
  FOR UPDATE USING (
    (
      author_id = insideout.current_user_id()
      AND insideout._is_member(insideout._project_workspace(insideout.project_updates.project_id), insideout.current_user_id())
    )
    OR insideout._is_admin(insideout._project_workspace(insideout.project_updates.project_id), insideout.current_user_id())
  );

DROP POLICY IF EXISTS project_updates_delete ON insideout.project_updates;
CREATE POLICY project_updates_delete ON insideout.project_updates
  FOR DELETE USING (
    (
      author_id = insideout.current_user_id()
      AND insideout._is_member(insideout._project_workspace(insideout.project_updates.project_id), insideout.current_user_id())
    )
    OR insideout._is_admin(insideout._project_workspace(insideout.project_updates.project_id), insideout.current_user_id())
  );

DROP POLICY IF EXISTS prd_revisions_insert ON insideout.prd_revisions;
CREATE POLICY prd_revisions_insert ON insideout.prd_revisions
  FOR INSERT WITH CHECK (
    created_by = insideout.current_user_id()
    AND insideout._prd_author(insideout.prd_revisions.prd_id) = insideout.current_user_id()
    AND insideout._is_member(insideout._prd_workspace(insideout.prd_revisions.prd_id), insideout.current_user_id())
  );
