CREATE OR REPLACE FUNCTION juanleme.create_workspace(
  p_title TEXT,
  p_description TEXT DEFAULT '',
  p_cover_url TEXT DEFAULT NULL,
  p_status TEXT DEFAULT 'draft'
)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, juanleme
AS $$
DECLARE
  v_uid UUID := auth.uid();
  v_workspace_id UUID;
  v_code TEXT;
BEGIN
  IF v_uid IS NULL THEN
    RAISE EXCEPTION 'Not authenticated' USING ERRCODE = '28000';
  END IF;

  v_code := lpad(floor(random() * 1000000)::text, 6, '0');

  INSERT INTO juanleme.workspaces (title, description, cover_url, code, creator_id, status)
  VALUES (p_title, p_description, p_cover_url, v_code, v_uid, p_status)
  RETURNING id INTO v_workspace_id;

  INSERT INTO juanleme.workspace_memberships (workspace_id, user_id, role)
  VALUES (v_workspace_id, v_uid, 'admin');

  RETURN v_workspace_id;
END;
$$;

REVOKE ALL ON FUNCTION juanleme.create_workspace(TEXT, TEXT, TEXT, TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION juanleme.create_workspace(TEXT, TEXT, TEXT, TEXT) TO authenticated;
