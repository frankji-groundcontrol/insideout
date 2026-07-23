-- Fix "new row violates row-level security policy for table workspaces"
-- on CreateWorkspace, found by running the authorization-checklist
-- integration tests against a live database.
--
-- INSERT ... RETURNING implicitly re-checks the table's SELECT policy on
-- the newly inserted row (Postgres requires the returned row to be
-- visible, not just insertable). CreateWorkspace's INSERT INTO workspaces
-- (with RETURNING) runs in the same transaction as, but strictly before,
-- the INSERT that makes the creator a member — so at the exact moment
-- RETURNING is evaluated, workspaces_select's membership-only check has
-- nothing to find yet. The creator should always be able to see a
-- workspace they just created regardless of that ordering, so add an
-- explicit creator_id branch (every other INSERT ... RETURNING path in
-- this schema already has pre-existing membership by the time it runs —
-- this is the only one that creates the workspace and its first
-- membership in two separate statements).

DROP POLICY IF EXISTS workspaces_select ON insideout.workspaces;
CREATE POLICY workspaces_select ON insideout.workspaces
  FOR SELECT USING (
    creator_id = insideout.current_user_id()
    OR EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m
      WHERE m.workspace_id = insideout.workspaces.id AND m.user_id = insideout.current_user_id()
    )
    OR (
      status = 'active'
      AND code = NULLIF(current_setting('app.join_code', true), '')
    )
  );
