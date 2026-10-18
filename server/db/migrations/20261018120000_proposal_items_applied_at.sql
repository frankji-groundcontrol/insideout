-- rv4-FEAT-02: accepting a proposal with apply created its roadmap nodes
-- but never remembered that it had, so a second accept (a double click, a
-- retry, a second tab, the CLI after the web) created every node again.
-- applied_at is that memory: the apply claims an item by stamping it
-- (UPDATE ... WHERE applied_at IS NULL) in the same transaction that
-- creates its node, so an item is applied once and a rolled-back apply
-- leaves it unapplied. NULL = not applied yet (every pre-migration row).
ALTER TABLE insideout.proposal_items
  ADD COLUMN IF NOT EXISTS applied_at timestamptz;

-- Items stay immutable apart from that stamp. The default privileges of
-- migration 20260819190000 already hand the app role a table-wide UPDATE
-- here (only the missing UPDATE policy stopped it); revoke it, then grant
-- the one column the apply needs. REVOKE first: a table-level REVOKE also
-- drops column grants, so the pair is re-runnable and ends in one state.
REVOKE UPDATE ON insideout.proposal_items FROM insideout_app;
GRANT UPDATE (applied_at) ON insideout.proposal_items TO insideout_app;

-- Who may stamp an item: whoever may apply it, the project owner or a
-- workspace admin (the plan writers; the same people who decide). With no
-- WITH CHECK clause the USING predicate also checks the updated row.
DROP POLICY IF EXISTS item_applier_update ON insideout.proposal_items;
CREATE POLICY item_applier_update ON insideout.proposal_items
  FOR UPDATE USING (
    EXISTS (
      SELECT 1 FROM insideout.project_updates u
      JOIN insideout.projects p ON p.id = u.project_id
      WHERE u.id = proposal_items.update_id
        AND (p.owner_id = insideout.current_user_id()
             OR insideout._is_admin(p.workspace_id, insideout.current_user_id()))
    )
  );
