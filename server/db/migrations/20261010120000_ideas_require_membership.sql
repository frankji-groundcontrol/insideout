-- SEC-11: an idea's author keeps no rights over it once removed from the
-- workspace. ideas_select, ideas_update and prds_insert let the author branch
-- through on author_id alone, so a removed member could still read, edit and
-- drop their old ideas, and insert a PRD (visible to the members) into a
-- workspace they had left. The author branch now also needs current
-- membership; the admin branch already implies it.
--
-- Same shape as the neighbouring policies (20260720155500): membership comes
-- from the SECURITY DEFINER helpers with table-qualified arguments, never an
-- inline subquery on workspace_memberships. ideas_update has no WITH CHECK, so
-- Postgres reuses its USING expression for the new row.

DROP POLICY IF EXISTS ideas_select ON insideout.ideas;
CREATE POLICY ideas_select ON insideout.ideas
  FOR SELECT USING (
    (
      author_id = insideout.current_user_id()
      AND insideout._is_member(insideout.ideas.workspace_id, insideout.current_user_id())
    )
    OR insideout._is_admin(insideout.ideas.workspace_id, insideout.current_user_id())
  );

DROP POLICY IF EXISTS ideas_update ON insideout.ideas;
CREATE POLICY ideas_update ON insideout.ideas
  FOR UPDATE USING (
    (
      author_id = insideout.current_user_id()
      AND insideout._is_member(insideout.ideas.workspace_id, insideout.current_user_id())
    )
    OR insideout._is_admin(insideout.ideas.workspace_id, insideout.current_user_id())
  );

DROP POLICY IF EXISTS prds_insert ON insideout.prds;
CREATE POLICY prds_insert ON insideout.prds
  FOR INSERT WITH CHECK (
    author_id = insideout.current_user_id()
    AND insideout._is_member(insideout.prds.workspace_id, insideout.current_user_id())
  );
