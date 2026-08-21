-- Leaf-node delivery evidence: GitHub events (and later other sources)
-- matched by the repo-side insideout.yaml guide append rows here.
-- Evidence never auto-changes node status or proves outcomes (PRODUCT.md);
-- it is the human-reviewable record of matched activity.
CREATE TABLE IF NOT EXISTS insideout.roadmap_evidence (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  node_id uuid NOT NULL REFERENCES insideout.roadmap_nodes(id) ON DELETE CASCADE,
  kind text NOT NULL,
  detail text NOT NULL,
  source_url text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE insideout.roadmap_evidence ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.roadmap_evidence FORCE ROW LEVEL SECURITY;

-- Visibility and writes follow roadmap-node membership: a workspace
-- member reads; the project owner or a workspace admin writes (the API
-- writes with the webhook-resolved owner).
CREATE POLICY evidence_member_select ON insideout.roadmap_evidence
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM insideout.roadmap_nodes n
      JOIN insideout.projects p ON p.id = n.project_id
      WHERE n.id = roadmap_evidence.node_id
        AND insideout._is_member(p.workspace_id, insideout.current_user_id())
    )
  );

CREATE POLICY evidence_owner_insert ON insideout.roadmap_evidence
  FOR INSERT WITH CHECK (
    EXISTS (
      SELECT 1 FROM insideout.roadmap_nodes n
      JOIN insideout.projects p ON p.id = n.project_id
      WHERE n.id = node_id
        AND (p.owner_id = insideout.current_user_id()
             OR insideout._is_admin(p.workspace_id, insideout.current_user_id()))
    )
  );

-- Redeliveries are idempotent: same node + kind + detail writes once.
CREATE UNIQUE INDEX IF NOT EXISTS roadmap_evidence_dedupe
  ON insideout.roadmap_evidence (node_id, kind, detail);

GRANT SELECT, INSERT ON insideout.roadmap_evidence TO insideout_app;
