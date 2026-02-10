-- =====================================================
-- juanleme RLS + policy matrix + views + RPC helpers
-- =====================================================

-- 1) Enable RLS on all domain tables
ALTER TABLE juanleme.profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE juanleme.workspaces ENABLE ROW LEVEL SECURITY;
ALTER TABLE juanleme.workspace_memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE juanleme.projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE juanleme.workshop_nodes ENABLE ROW LEVEL SECURITY;
ALTER TABLE juanleme.documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE juanleme.document_revisions ENABLE ROW LEVEL SECURITY;
ALTER TABLE juanleme.ai_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE juanleme.export_jobs ENABLE ROW LEVEL SECURITY;

-- 2) Workspace role helper
CREATE OR REPLACE FUNCTION juanleme.get_user_workspace_role(p_workspace_id UUID)
RETURNS TEXT
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = pg_catalog, juanleme
AS $$
  SELECT role
  FROM juanleme.workspace_memberships
  WHERE workspace_id = p_workspace_id
    AND user_id = auth.uid();
$$;

REVOKE ALL ON FUNCTION juanleme.get_user_workspace_role(UUID) FROM PUBLIC;

-- 3) RLS policies (deny-by-default)

-- profiles
DROP POLICY IF EXISTS profiles_select_own ON juanleme.profiles;
DROP POLICY IF EXISTS profiles_insert_own ON juanleme.profiles;
DROP POLICY IF EXISTS profiles_update_own_user_role ON juanleme.profiles;
DROP POLICY IF EXISTS profiles_update_own_admin_role ON juanleme.profiles;

CREATE POLICY profiles_select_own
ON juanleme.profiles
FOR SELECT
TO authenticated
USING (id = auth.uid());

CREATE POLICY profiles_insert_own
ON juanleme.profiles
FOR INSERT
TO authenticated
WITH CHECK (id = auth.uid());

CREATE POLICY profiles_update_own_user_role
ON juanleme.profiles
FOR UPDATE
TO authenticated
USING (id = auth.uid() AND role = 'user')
WITH CHECK (id = auth.uid() AND role = 'user');

CREATE POLICY profiles_update_own_admin_role
ON juanleme.profiles
FOR UPDATE
TO authenticated
USING (id = auth.uid() AND role = 'admin')
WITH CHECK (id = auth.uid() AND role = 'admin');

-- workspaces
DROP POLICY IF EXISTS workspaces_select_member ON juanleme.workspaces;
DROP POLICY IF EXISTS workspaces_insert_creator ON juanleme.workspaces;
DROP POLICY IF EXISTS workspaces_update_admin_or_creator ON juanleme.workspaces;
DROP POLICY IF EXISTS workspaces_delete_creator ON juanleme.workspaces;

CREATE POLICY workspaces_select_member
ON juanleme.workspaces
FOR SELECT
TO authenticated
USING (juanleme.get_user_workspace_role(id) IS NOT NULL);

CREATE POLICY workspaces_insert_creator
ON juanleme.workspaces
FOR INSERT
TO authenticated
WITH CHECK (creator_id = auth.uid());

CREATE POLICY workspaces_update_admin_or_creator
ON juanleme.workspaces
FOR UPDATE
TO authenticated
USING (juanleme.get_user_workspace_role(id) = 'admin' OR creator_id = auth.uid())
WITH CHECK (juanleme.get_user_workspace_role(id) = 'admin' OR creator_id = auth.uid());

CREATE POLICY workspaces_delete_creator
ON juanleme.workspaces
FOR DELETE
TO authenticated
USING (creator_id = auth.uid());

-- workspace_memberships
DROP POLICY IF EXISTS workspace_memberships_select_member ON juanleme.workspace_memberships;
DROP POLICY IF EXISTS workspace_memberships_insert_deny ON juanleme.workspace_memberships;
DROP POLICY IF EXISTS workspace_memberships_update_admin ON juanleme.workspace_memberships;
DROP POLICY IF EXISTS workspace_memberships_delete_admin_or_self ON juanleme.workspace_memberships;

CREATE POLICY workspace_memberships_select_member
ON juanleme.workspace_memberships
FOR SELECT
TO authenticated
USING (juanleme.get_user_workspace_role(workspace_id) IS NOT NULL);

CREATE POLICY workspace_memberships_insert_deny
ON juanleme.workspace_memberships
FOR INSERT
TO authenticated
WITH CHECK (false);

CREATE POLICY workspace_memberships_update_admin
ON juanleme.workspace_memberships
FOR UPDATE
TO authenticated
USING (juanleme.get_user_workspace_role(workspace_id) = 'admin')
WITH CHECK (juanleme.get_user_workspace_role(workspace_id) = 'admin');

CREATE POLICY workspace_memberships_delete_admin_or_self
ON juanleme.workspace_memberships
FOR DELETE
TO authenticated
USING (juanleme.get_user_workspace_role(workspace_id) = 'admin' OR user_id = auth.uid());

-- projects
DROP POLICY IF EXISTS projects_select_member ON juanleme.projects;
DROP POLICY IF EXISTS projects_insert_admin ON juanleme.projects;
DROP POLICY IF EXISTS projects_update_admin_or_creator ON juanleme.projects;
DROP POLICY IF EXISTS projects_delete_admin ON juanleme.projects;

CREATE POLICY projects_select_member
ON juanleme.projects
FOR SELECT
TO authenticated
USING (juanleme.get_user_workspace_role(workspace_id) IS NOT NULL);

CREATE POLICY projects_insert_admin
ON juanleme.projects
FOR INSERT
TO authenticated
WITH CHECK (juanleme.get_user_workspace_role(workspace_id) = 'admin' AND created_by = auth.uid());

CREATE POLICY projects_update_admin_or_creator
ON juanleme.projects
FOR UPDATE
TO authenticated
USING (juanleme.get_user_workspace_role(workspace_id) = 'admin' OR created_by = auth.uid())
WITH CHECK (juanleme.get_user_workspace_role(workspace_id) = 'admin' OR created_by = auth.uid());

CREATE POLICY projects_delete_admin
ON juanleme.projects
FOR DELETE
TO authenticated
USING (juanleme.get_user_workspace_role(workspace_id) = 'admin');

-- workshop_nodes
DROP POLICY IF EXISTS workshop_nodes_select_member ON juanleme.workshop_nodes;
DROP POLICY IF EXISTS workshop_nodes_insert_admin ON juanleme.workshop_nodes;
DROP POLICY IF EXISTS workshop_nodes_update_admin ON juanleme.workshop_nodes;
DROP POLICY IF EXISTS workshop_nodes_delete_admin ON juanleme.workshop_nodes;

CREATE POLICY workshop_nodes_select_member
ON juanleme.workshop_nodes
FOR SELECT
TO authenticated
USING (juanleme.get_user_workspace_role(workspace_id) IS NOT NULL);

CREATE POLICY workshop_nodes_insert_admin
ON juanleme.workshop_nodes
FOR INSERT
TO authenticated
WITH CHECK (juanleme.get_user_workspace_role(workspace_id) = 'admin');

CREATE POLICY workshop_nodes_update_admin
ON juanleme.workshop_nodes
FOR UPDATE
TO authenticated
USING (juanleme.get_user_workspace_role(workspace_id) = 'admin')
WITH CHECK (juanleme.get_user_workspace_role(workspace_id) = 'admin');

CREATE POLICY workshop_nodes_delete_admin
ON juanleme.workshop_nodes
FOR DELETE
TO authenticated
USING (juanleme.get_user_workspace_role(workspace_id) = 'admin');

-- documents
DROP POLICY IF EXISTS documents_select_own_member ON juanleme.documents;
DROP POLICY IF EXISTS documents_insert_own_member ON juanleme.documents;
DROP POLICY IF EXISTS documents_update_own ON juanleme.documents;
DROP POLICY IF EXISTS documents_delete_own ON juanleme.documents;

CREATE POLICY documents_select_own_member
ON juanleme.documents
FOR SELECT
TO authenticated
USING (
  user_id = auth.uid()
  AND EXISTS (
    SELECT 1
    FROM juanleme.workshop_nodes wn
    WHERE wn.id = node_id
      AND juanleme.get_user_workspace_role(wn.workspace_id) IS NOT NULL
  )
);

CREATE POLICY documents_insert_own_member
ON juanleme.documents
FOR INSERT
TO authenticated
WITH CHECK (
  user_id = auth.uid()
  AND EXISTS (
    SELECT 1
    FROM juanleme.workshop_nodes wn
    WHERE wn.id = node_id
      AND juanleme.get_user_workspace_role(wn.workspace_id) IS NOT NULL
  )
);

CREATE POLICY documents_update_own
ON juanleme.documents
FOR UPDATE
TO authenticated
USING (user_id = auth.uid())
WITH CHECK (user_id = auth.uid());

CREATE POLICY documents_delete_own
ON juanleme.documents
FOR DELETE
TO authenticated
USING (user_id = auth.uid());

-- document_revisions
DROP POLICY IF EXISTS document_revisions_select_own_docs ON juanleme.document_revisions;
DROP POLICY IF EXISTS document_revisions_insert_own_docs ON juanleme.document_revisions;

CREATE POLICY document_revisions_select_own_docs
ON juanleme.document_revisions
FOR SELECT
TO authenticated
USING (
  EXISTS (
    SELECT 1
    FROM juanleme.documents d
    WHERE d.id = document_id
      AND d.user_id = auth.uid()
  )
);

CREATE POLICY document_revisions_insert_own_docs
ON juanleme.document_revisions
FOR INSERT
TO authenticated
WITH CHECK (
  EXISTS (
    SELECT 1
    FROM juanleme.documents d
    WHERE d.id = document_id
      AND d.user_id = auth.uid()
  )
);

-- ai_runs
DROP POLICY IF EXISTS ai_runs_select_own ON juanleme.ai_runs;
DROP POLICY IF EXISTS ai_runs_insert_deny ON juanleme.ai_runs;
DROP POLICY IF EXISTS ai_runs_update_deny ON juanleme.ai_runs;
DROP POLICY IF EXISTS ai_runs_delete_deny ON juanleme.ai_runs;

CREATE POLICY ai_runs_select_own
ON juanleme.ai_runs
FOR SELECT
TO authenticated
USING (user_id = auth.uid());

CREATE POLICY ai_runs_insert_deny
ON juanleme.ai_runs
FOR INSERT
TO authenticated
WITH CHECK (false);

CREATE POLICY ai_runs_update_deny
ON juanleme.ai_runs
FOR UPDATE
TO authenticated
USING (false)
WITH CHECK (false);

CREATE POLICY ai_runs_delete_deny
ON juanleme.ai_runs
FOR DELETE
TO authenticated
USING (false);

-- export_jobs
DROP POLICY IF EXISTS export_jobs_select_own ON juanleme.export_jobs;
DROP POLICY IF EXISTS export_jobs_insert_deny ON juanleme.export_jobs;
DROP POLICY IF EXISTS export_jobs_update_deny ON juanleme.export_jobs;
DROP POLICY IF EXISTS export_jobs_delete_deny ON juanleme.export_jobs;

CREATE POLICY export_jobs_select_own
ON juanleme.export_jobs
FOR SELECT
TO authenticated
USING (user_id = auth.uid());

CREATE POLICY export_jobs_insert_deny
ON juanleme.export_jobs
FOR INSERT
TO authenticated
WITH CHECK (false);

CREATE POLICY export_jobs_update_deny
ON juanleme.export_jobs
FOR UPDATE
TO authenticated
USING (false)
WITH CHECK (false);

CREATE POLICY export_jobs_delete_deny
ON juanleme.export_jobs
FOR DELETE
TO authenticated
USING (false);

-- 4) Helper views (security invoker by default)
CREATE OR REPLACE VIEW juanleme.v_workshop_summary
WITH (security_invoker = true) AS
SELECT
  w.id,
  w.title,
  w.description,
  w.cover_url,
  w.code,
  w.creator_id,
  w.status,
  w.created_at,
  (
    SELECT count(*)
    FROM juanleme.workspace_memberships wm
    WHERE wm.workspace_id = w.id
  ) AS member_count,
  EXISTS (
    SELECT 1
    FROM juanleme.workspace_memberships wm
    WHERE wm.workspace_id = w.id
      AND wm.user_id = auth.uid()
  ) AS is_joined
FROM juanleme.workspaces w;

CREATE OR REPLACE VIEW juanleme.v_roadmap_with_status
WITH (security_invoker = true) AS
SELECT
  wn.id,
  wn.workspace_id,
  wn.title,
  wn.description,
  wn."order",
  wn.created_at,
  CASE
    WHEN d.status = 'completed' THEN 'completed'
    WHEN d.status = 'submitted' THEN 'completed'
    WHEN d.status = 'draft' THEN 'in_progress'
    WHEN d.id IS NOT NULL THEN 'in_progress'
    ELSE 'pending'
  END AS status,
  d.content
FROM juanleme.workshop_nodes wn
LEFT JOIN juanleme.documents d
  ON d.node_id = wn.id
 AND d.user_id = auth.uid();

-- 5) RPC functions
CREATE OR REPLACE FUNCTION juanleme.join_workshop(p_code TEXT)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, juanleme
AS $$
DECLARE
  v_workspace_id UUID;
  v_membership_id UUID;
  v_uid UUID := auth.uid();
BEGIN
  IF v_uid IS NULL THEN
    RAISE EXCEPTION 'Not authenticated' USING ERRCODE = '28000';
  END IF;

  SELECT id
  INTO v_workspace_id
  FROM juanleme.workspaces
  WHERE code = p_code
    AND status = 'active';

  IF v_workspace_id IS NULL THEN
    RAISE EXCEPTION 'Invalid invite code' USING ERRCODE = 'P0002';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM juanleme.workspace_memberships
    WHERE workspace_id = v_workspace_id
      AND user_id = v_uid
  ) THEN
    RAISE EXCEPTION 'Already a member' USING ERRCODE = '23505';
  END IF;

  INSERT INTO juanleme.workspace_memberships (workspace_id, user_id, role)
  VALUES (v_workspace_id, v_uid, 'member')
  RETURNING id INTO v_membership_id;

  RETURN v_membership_id;
END;
$$;

CREATE OR REPLACE FUNCTION juanleme.complete_node(p_node_id UUID, p_content JSONB)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, juanleme
AS $$
DECLARE
  v_uid UUID := auth.uid();
  v_workspace_id UUID;
  v_doc_id UUID;
  v_revision INT;
  v_result JSONB;
BEGIN
  IF v_uid IS NULL THEN
    RAISE EXCEPTION 'Not authenticated' USING ERRCODE = '28000';
  END IF;

  SELECT wn.workspace_id
  INTO v_workspace_id
  FROM juanleme.workshop_nodes wn
  WHERE wn.id = p_node_id;

  IF v_workspace_id IS NULL THEN
    RAISE EXCEPTION 'Node not found' USING ERRCODE = 'P0002';
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM juanleme.workspace_memberships
    WHERE workspace_id = v_workspace_id
      AND user_id = v_uid
  ) THEN
    RAISE EXCEPTION 'Not a member of this workspace' USING ERRCODE = '42501';
  END IF;

  INSERT INTO juanleme.documents (node_id, user_id, content, status)
  VALUES (p_node_id, v_uid, p_content, 'completed')
  ON CONFLICT (node_id, user_id) DO UPDATE
  SET content = EXCLUDED.content,
      status = 'completed'
  RETURNING id INTO v_doc_id;

  -- Serialize revision assignment for this document.
  PERFORM 1
  FROM juanleme.documents
  WHERE id = v_doc_id
  FOR UPDATE;

  SELECT COALESCE(MAX(revision), 0) + 1
  INTO v_revision
  FROM juanleme.document_revisions
  WHERE document_id = v_doc_id;

  INSERT INTO juanleme.document_revisions (document_id, content, revision)
  VALUES (v_doc_id, p_content, v_revision);

  v_result := jsonb_build_object(
    'document_id', v_doc_id,
    'revision', v_revision,
    'status', 'completed'
  );

  RETURN v_result;
END;
$$;

CREATE OR REPLACE FUNCTION juanleme.get_workshop_roadmap(p_workspace_id UUID)
RETURNS TABLE (
  id UUID,
  workspace_id UUID,
  title TEXT,
  description TEXT,
  "order" INT,
  status TEXT,
  content JSONB,
  created_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
STABLE
SET search_path = pg_catalog, juanleme
AS $$
DECLARE
  v_uid UUID := auth.uid();
BEGIN
  IF v_uid IS NULL THEN
    RAISE EXCEPTION 'Not authenticated' USING ERRCODE = '28000';
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM juanleme.workspace_memberships wm
    WHERE wm.workspace_id = p_workspace_id
      AND wm.user_id = v_uid
  ) THEN
    RAISE EXCEPTION 'Not a member of this workspace' USING ERRCODE = '42501';
  END IF;

  RETURN QUERY
  SELECT
    wn.id,
    wn.workspace_id,
    wn.title,
    wn.description,
    wn."order",
    CASE
      WHEN d.status = 'completed' THEN 'completed'::TEXT
      WHEN d.status = 'submitted' THEN 'completed'::TEXT
      WHEN d.status = 'draft' THEN 'in_progress'::TEXT
      WHEN d.id IS NOT NULL THEN 'in_progress'::TEXT
      ELSE 'pending'::TEXT
    END AS status,
    d.content,
    wn.created_at
  FROM juanleme.workshop_nodes wn
  LEFT JOIN juanleme.documents d
    ON d.node_id = wn.id
   AND d.user_id = v_uid
  WHERE wn.workspace_id = p_workspace_id
  ORDER BY wn."order" ASC;
END;
$$;

REVOKE ALL ON FUNCTION juanleme.join_workshop(TEXT) FROM PUBLIC;
REVOKE ALL ON FUNCTION juanleme.complete_node(UUID, JSONB) FROM PUBLIC;
REVOKE ALL ON FUNCTION juanleme.get_workshop_roadmap(UUID) FROM PUBLIC;

-- 6) Grants
REVOKE ALL ON TABLE
  juanleme.profiles,
  juanleme.workspaces,
  juanleme.workspace_memberships,
  juanleme.projects,
  juanleme.workshop_nodes,
  juanleme.documents,
  juanleme.document_revisions,
  juanleme.ai_runs,
  juanleme.export_jobs
FROM anon, authenticated;

REVOKE ALL ON TABLE
  juanleme.v_workshop_summary,
  juanleme.v_roadmap_with_status
FROM anon, authenticated;

REVOKE ALL ON FUNCTION juanleme.join_workshop(TEXT) FROM anon, authenticated;
REVOKE ALL ON FUNCTION juanleme.complete_node(UUID, JSONB) FROM anon, authenticated;
REVOKE ALL ON FUNCTION juanleme.get_workshop_roadmap(UUID) FROM anon, authenticated;
REVOKE ALL ON FUNCTION juanleme.get_user_workspace_role(UUID) FROM anon, authenticated;

GRANT SELECT, INSERT, UPDATE ON juanleme.profiles TO authenticated;
GRANT SELECT, INSERT, UPDATE, DELETE ON juanleme.workspaces TO authenticated;
GRANT SELECT, UPDATE, DELETE ON juanleme.workspace_memberships TO authenticated;
GRANT SELECT, INSERT, UPDATE, DELETE ON juanleme.projects TO authenticated;
GRANT SELECT, INSERT, UPDATE, DELETE ON juanleme.workshop_nodes TO authenticated;
GRANT SELECT, INSERT, UPDATE, DELETE ON juanleme.documents TO authenticated;
GRANT SELECT, INSERT ON juanleme.document_revisions TO authenticated;
GRANT SELECT ON juanleme.ai_runs TO authenticated;
GRANT SELECT ON juanleme.export_jobs TO authenticated;

GRANT EXECUTE ON FUNCTION juanleme.join_workshop(TEXT) TO authenticated;
GRANT EXECUTE ON FUNCTION juanleme.complete_node(UUID, JSONB) TO authenticated;
GRANT EXECUTE ON FUNCTION juanleme.get_workshop_roadmap(UUID) TO authenticated;
GRANT EXECUTE ON FUNCTION juanleme.get_user_workspace_role(UUID) TO authenticated;

GRANT SELECT ON juanleme.v_workshop_summary TO authenticated;
GRANT SELECT ON juanleme.v_roadmap_with_status TO authenticated;
