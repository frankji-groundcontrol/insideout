-- Product version control, Stage 1: the human Commit (PRODUCT.md
-- "Product version control"). A commit freezes one prd_revisions
-- snapshot as an immutable version and records why it exists. Working
-- versions stay mutable; no UPDATE/DELETE paths exist for commits —
-- immutability is structural.
CREATE TABLE IF NOT EXISTS insideout.prd_commits (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  prd_id uuid NOT NULL REFERENCES insideout.prds(id) ON DELETE CASCADE,
  revision int NOT NULL,
  name text NOT NULL,
  primary_audience text NOT NULL,
  change_summary text NOT NULL DEFAULT '',
  unresolved jsonb NOT NULL DEFAULT '[]'::jsonb,
  decision_note text NOT NULL DEFAULT '',
  diff jsonb NOT NULL DEFAULT '{}'::jsonb,
  committed_by uuid NOT NULL REFERENCES insideout.users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (prd_id, revision)
);

ALTER TABLE insideout.prd_commits ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.prd_commits FORCE ROW LEVEL SECURITY;

-- Visibility follows the PRD's workspace membership; writes are the
-- Driver (PRD author) or a workspace admin, matching the revision
-- policies.
CREATE POLICY prd_commit_member_select ON insideout.prd_commits
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM insideout.prds p
      WHERE p.id = prd_commits.prd_id
        AND insideout._is_member(p.workspace_id, insideout.current_user_id())
    )
  );

CREATE POLICY prd_commit_driver_insert ON insideout.prd_commits
  FOR INSERT WITH CHECK (
    EXISTS (
      SELECT 1 FROM insideout.prds p
      WHERE p.id = prd_id
        AND (p.author_id = insideout.current_user_id()
             OR insideout._is_admin(p.workspace_id, insideout.current_user_id()))
    )
  );

GRANT SELECT, INSERT ON insideout.prd_commits TO insideout_app;
