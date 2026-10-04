-- USER-05 / FEAT-04 (2026-10-04 remediation Wave 3): ideas are private to
-- their author by default. PRODUCT.md: "Capture remains one-step and
-- private to its author by default"; the README sells each member a
-- personal inbox. Reads become author-scoped; workspace admins keep read
-- access to match their existing drop authority (ideas_update). Write
-- policies are unchanged: insert still needs membership, update/convert
-- stay author-only, drop stays author-or-admin.
DROP POLICY IF EXISTS ideas_select ON insideout.ideas;
CREATE POLICY ideas_select ON insideout.ideas
  FOR SELECT USING (
    author_id = insideout.current_user_id()
    OR insideout._is_admin(insideout.ideas.workspace_id, insideout.current_user_id())
  );
