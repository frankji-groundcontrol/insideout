-- Proposal→structure application: a proposal may carry structured
-- items at propose time; accepting with apply creates them as real
-- roadmap nodes (a human decided — PRODUCT.md "Collaboration and
-- authority"). Items are recorded with the proposal, immutable.
CREATE TABLE IF NOT EXISTS insideout.proposal_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  update_id uuid NOT NULL REFERENCES insideout.project_updates(id) ON DELETE CASCADE,
  action text NOT NULL CHECK (action IN ('add_node')),
  title text NOT NULL,
  parent_hint text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE insideout.proposal_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.proposal_items FORCE ROW LEVEL SECURITY;

CREATE POLICY item_member_select ON insideout.proposal_items
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM insideout.project_updates u
      JOIN insideout.projects p ON p.id = u.project_id
      WHERE u.id = proposal_items.update_id
        AND insideout._is_member(p.workspace_id, insideout.current_user_id())
    )
  );

CREATE POLICY item_member_insert ON insideout.proposal_items
  FOR INSERT WITH CHECK (
    EXISTS (
      SELECT 1 FROM insideout.project_updates u
      JOIN insideout.projects p ON p.id = u.project_id
      WHERE u.id = update_id
        AND insideout._is_member(p.workspace_id, insideout.current_user_id())
    )
  );

GRANT SELECT, INSERT ON insideout.proposal_items TO insideout_app;
