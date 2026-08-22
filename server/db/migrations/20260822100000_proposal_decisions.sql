-- Proposal acceptance (agent vocabulary follow-on): the human decision
-- on an agent proposal, as an immutable decision-log record
-- (PRODUCT.md Decision Log). Insert-only — history is never rewritten.
CREATE TABLE IF NOT EXISTS insideout.proposal_decisions (
  update_id uuid PRIMARY KEY REFERENCES insideout.project_updates(id) ON DELETE CASCADE,
  decision text NOT NULL CHECK (decision IN ('accepted', 'rejected')),
  reason text NOT NULL DEFAULT '',
  decided_by uuid NOT NULL REFERENCES insideout.users(id),
  decided_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE insideout.proposal_decisions ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.proposal_decisions FORCE ROW LEVEL SECURITY;

-- Visibility follows the proposal's project membership; the decision
-- is written by the project owner or a workspace admin (humans decide).
CREATE POLICY decision_member_select ON insideout.proposal_decisions
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM insideout.project_updates u
      JOIN insideout.projects p ON p.id = u.project_id
      WHERE u.id = proposal_decisions.update_id
        AND insideout._is_member(p.workspace_id, insideout.current_user_id())
    )
  );

CREATE POLICY decision_owner_insert ON insideout.proposal_decisions
  FOR INSERT WITH CHECK (
    EXISTS (
      SELECT 1 FROM insideout.project_updates u
      JOIN insideout.projects p ON p.id = u.project_id
      WHERE u.id = update_id
        AND (p.owner_id = insideout.current_user_id()
             OR insideout._is_admin(p.workspace_id, insideout.current_user_id()))
    )
  );

GRANT SELECT, INSERT ON insideout.proposal_decisions TO insideout_app;
