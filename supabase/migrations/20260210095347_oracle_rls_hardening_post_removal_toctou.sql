-- Oracle 审核后的 RLS 加固迁移
-- 修复项: 1) 离开工作区后仍可操作文档 2) TOCTOU 竞态条件 3) 工作区成员可查看彼此档案

-- ============================================================
-- 1) 文档 UPDATE/DELETE 必须验证当前成员身份（防止退出工作区后仍可操作）
-- ============================================================

DROP POLICY IF EXISTS documents_update_own ON juanleme.documents;
CREATE POLICY documents_update_own
ON juanleme.documents
FOR UPDATE
TO authenticated
USING (
  user_id = auth.uid()
  AND EXISTS (
    SELECT 1
    FROM juanleme.workshop_nodes wn
    WHERE wn.id = node_id
      AND juanleme.get_user_workspace_role(wn.workspace_id) IS NOT NULL
  )
)
WITH CHECK (
  user_id = auth.uid()
  AND EXISTS (
    SELECT 1
    FROM juanleme.workshop_nodes wn
    WHERE wn.id = node_id
      AND juanleme.get_user_workspace_role(wn.workspace_id) IS NOT NULL
  )
);

DROP POLICY IF EXISTS documents_delete_own ON juanleme.documents;
CREATE POLICY documents_delete_own
ON juanleme.documents
FOR DELETE
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

-- ============================================================
-- 2) document_revisions 同样需要验证成员身份
-- ============================================================

DROP POLICY IF EXISTS document_revisions_select_own_docs ON juanleme.document_revisions;
CREATE POLICY document_revisions_select_own_docs
ON juanleme.document_revisions
FOR SELECT
TO authenticated
USING (
  EXISTS (
    SELECT 1
    FROM juanleme.documents d
    JOIN juanleme.workshop_nodes wn ON wn.id = d.node_id
    WHERE d.id = document_id
      AND d.user_id = auth.uid()
      AND juanleme.get_user_workspace_role(wn.workspace_id) IS NOT NULL
  )
);

DROP POLICY IF EXISTS document_revisions_insert_own_docs ON juanleme.document_revisions;
CREATE POLICY document_revisions_insert_own_docs
ON juanleme.document_revisions
FOR INSERT
TO authenticated
WITH CHECK (
  EXISTS (
    SELECT 1
    FROM juanleme.documents d
    JOIN juanleme.workshop_nodes wn ON wn.id = d.node_id
    WHERE d.id = document_id
      AND d.user_id = auth.uid()
      AND juanleme.get_user_workspace_role(wn.workspace_id) IS NOT NULL
  )
);

-- ============================================================
-- 3) TOCTOU 加固: complete_node 使用 FOR KEY SHARE 锁
-- ============================================================

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

  SELECT wn.workspace_id INTO v_workspace_id
  FROM juanleme.workshop_nodes wn
  WHERE wn.id = p_node_id;

  IF v_workspace_id IS NULL THEN
    RAISE EXCEPTION 'Node not found' USING ERRCODE = 'P0002';
  END IF;

  -- TOCTOU 加固: 使用 FOR KEY SHARE 锁定成员行，防止并发删除
  PERFORM 1
  FROM juanleme.workspace_memberships
  WHERE workspace_id = v_workspace_id
    AND user_id = v_uid
  FOR KEY SHARE;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'Not a member of this workspace' USING ERRCODE = '42501';
  END IF;

  INSERT INTO juanleme.documents (node_id, user_id, content, status)
  VALUES (p_node_id, v_uid, p_content, 'completed')
  ON CONFLICT (node_id, user_id) DO UPDATE
  SET content = EXCLUDED.content,
      status = 'completed'
  RETURNING id INTO v_doc_id;

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

-- ============================================================
-- 4) TOCTOU 加固: get_workshop_roadmap 使用 FOR KEY SHARE 锁
-- ============================================================

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

  -- TOCTOU 加固: FOR KEY SHARE 保持成员锁到查询结束
  PERFORM 1
  FROM juanleme.workspace_memberships wm
  WHERE wm.workspace_id = p_workspace_id
    AND wm.user_id = v_uid
  FOR KEY SHARE;

  IF NOT FOUND THEN
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

-- ============================================================
-- 5) 工作区成员可查看彼此基本档案（产品需求：显示成员名）
-- ============================================================

CREATE POLICY profiles_select_workspace_member
ON juanleme.profiles
FOR SELECT
TO authenticated
USING (
  EXISTS (
    SELECT 1
    FROM juanleme.workspace_memberships my_ws
    JOIN juanleme.workspace_memberships their_ws
      ON their_ws.workspace_id = my_ws.workspace_id
    WHERE my_ws.user_id = auth.uid()
      AND their_ws.user_id = juanleme.profiles.id
  )
);
