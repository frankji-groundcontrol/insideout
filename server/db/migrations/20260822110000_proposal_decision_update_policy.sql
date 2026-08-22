-- The proposal-decision upsert (re-deciding reverses a decision, per
-- PRODUCT.md's Decision Log) needs an UPDATE policy; the row holds the
-- latest decision while each decide appends a timeline note (history).
ALTER TABLE insideout.proposal_decisions ENABLE ROW LEVEL SECURITY;
CREATE POLICY decision_owner_update ON insideout.proposal_decisions
  FOR UPDATE USING (
    EXISTS (
      SELECT 1 FROM insideout.project_updates u
      JOIN insideout.projects p ON p.id = u.project_id
      WHERE u.id = proposal_decisions.update_id
        AND (p.owner_id = insideout.current_user_id()
             OR insideout._is_admin(p.workspace_id, insideout.current_user_id()))
    )
  ) WITH CHECK (
    EXISTS (
      SELECT 1 FROM insideout.project_updates u
      JOIN insideout.projects p ON p.id = u.project_id
      WHERE u.id = update_id
        AND (p.owner_id = insideout.current_user_id()
             OR insideout._is_admin(p.workspace_id, insideout.current_user_id()))
    )
  );

GRANT UPDATE ON insideout.proposal_decisions TO insideout_app;
