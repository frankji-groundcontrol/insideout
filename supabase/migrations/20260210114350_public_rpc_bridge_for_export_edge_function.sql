CREATE OR REPLACE FUNCTION public.export_resolve_workspace(p_workshop_id uuid)
RETURNS TABLE (
  project_id uuid,
  workspace_id uuid,
  title text,
  description text
)
LANGUAGE sql
SECURITY INVOKER
STABLE
SET search_path = pg_catalog
AS $function$
  SELECT
    p.id AS project_id,
    p.workspace_id,
    w.title,
    w.description
  FROM juanleme.projects AS p
  JOIN juanleme.workspaces AS w
    ON w.id = p.workspace_id
  WHERE (p.id = p_workshop_id OR p.workspace_id = p_workshop_id)
    AND EXISTS (
      SELECT 1
      FROM juanleme.workspace_memberships AS wm
      WHERE wm.workspace_id = p.workspace_id
        AND wm.user_id = auth.uid()
    )
  ORDER BY p.created_at ASC
  LIMIT 1
$function$;

CREATE OR REPLACE FUNCTION public.export_get_workspace(p_workspace_id uuid)
RETURNS TABLE (
  id uuid,
  title text,
  description text
)
LANGUAGE sql
SECURITY INVOKER
STABLE
SET search_path = pg_catalog
AS $function$
  SELECT
    w.id,
    w.title,
    w.description
  FROM juanleme.workspaces AS w
  WHERE w.id = p_workspace_id
    AND EXISTS (
      SELECT 1
      FROM juanleme.workspace_memberships AS wm
      WHERE wm.workspace_id = w.id
        AND wm.user_id = auth.uid()
    )
  LIMIT 1
$function$;

CREATE OR REPLACE FUNCTION public.export_get_workspace_nodes_me(p_workspace_id uuid)
RETURNS TABLE (
  id uuid,
  title text,
  description text,
  "order" integer,
  content jsonb
)
LANGUAGE sql
SECURITY INVOKER
STABLE
SET search_path = pg_catalog
AS $function$
  SELECT
    wn.id,
    wn.title,
    wn.description,
    wn."order",
    d.content
  FROM juanleme.workshop_nodes AS wn
  LEFT JOIN juanleme.documents AS d
    ON d.node_id = wn.id
   AND d.user_id = auth.uid()
  WHERE wn.workspace_id = p_workspace_id
    AND EXISTS (
      SELECT 1
      FROM juanleme.workspace_memberships AS wm
      WHERE wm.workspace_id = wn.workspace_id
        AND wm.user_id = auth.uid()
    )
  ORDER BY wn."order" ASC
$function$;

CREATE OR REPLACE FUNCTION public.export_find_idempotent_job(
  p_workspace_id uuid,
  p_user_id uuid,
  p_idempotency_key text
)
RETURNS TABLE (
  id uuid,
  status text,
  format text,
  created_at timestamptz,
  artifact_url text,
  artifact_path text,
  error_message text
)
LANGUAGE sql
SECURITY INVOKER
STABLE
SET search_path = pg_catalog
AS $function$
  SELECT
    ej.id,
    ej.status,
    ej.format,
    ej.created_at,
    NULL::text AS artifact_url,
    ej.artifact_path,
    ej.error_message
  FROM juanleme.export_jobs AS ej
  WHERE ej.workspace_id = p_workspace_id
    AND ej.user_id = p_user_id
    AND ej.idempotency_key = p_idempotency_key
  ORDER BY ej.created_at DESC
  LIMIT 1
$function$;

CREATE OR REPLACE FUNCTION public.export_create_job_service(
  p_workspace_id uuid,
  p_user_id uuid,
  p_format text,
  p_status text DEFAULT 'pending',
  p_idempotency_key text DEFAULT NULL
)
RETURNS TABLE (
  id uuid,
  status text,
  format text,
  created_at timestamptz
)
LANGUAGE sql
SECURITY INVOKER
VOLATILE
SET search_path = pg_catalog
AS $function$
  INSERT INTO juanleme.export_jobs (workspace_id, user_id, format, status, idempotency_key)
  VALUES (p_workspace_id, p_user_id, p_format, p_status, p_idempotency_key)
  RETURNING export_jobs.id, export_jobs.status, export_jobs.format, export_jobs.created_at
$function$;

CREATE OR REPLACE FUNCTION public.export_update_job_failure_service(
  p_job_id uuid,
  p_error_message text
)
RETURNS boolean
LANGUAGE sql
SECURITY INVOKER
VOLATILE
SET search_path = pg_catalog
AS $function$
  WITH upd AS (
    UPDATE juanleme.export_jobs
    SET status = 'failed',
        error_message = left(p_error_message, 2000),
        updated_at = now()
    WHERE id = p_job_id
    RETURNING 1
  )
  SELECT EXISTS (SELECT 1 FROM upd)
$function$;

CREATE OR REPLACE FUNCTION public.export_update_job_success_service(
  p_job_id uuid,
  p_artifact_path text
)
RETURNS boolean
LANGUAGE sql
SECURITY INVOKER
VOLATILE
SET search_path = pg_catalog
AS $function$
  WITH upd AS (
    UPDATE juanleme.export_jobs
    SET status = 'succeeded',
        artifact_path = p_artifact_path,
        error_message = NULL,
        updated_at = now()
    WHERE id = p_job_id
    RETURNING 1
  )
  SELECT EXISTS (SELECT 1 FROM upd)
$function$;

REVOKE ALL ON FUNCTION public.export_resolve_workspace(uuid) FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION public.export_get_workspace(uuid) FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION public.export_get_workspace_nodes_me(uuid) FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION public.export_find_idempotent_job(uuid, uuid, text) FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION public.export_create_job_service(uuid, uuid, text, text, text) FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION public.export_update_job_failure_service(uuid, text) FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION public.export_update_job_success_service(uuid, text) FROM PUBLIC, anon, authenticated, service_role;

GRANT EXECUTE ON FUNCTION public.export_resolve_workspace(uuid) TO authenticated;
GRANT EXECUTE ON FUNCTION public.export_get_workspace(uuid) TO authenticated;
GRANT EXECUTE ON FUNCTION public.export_get_workspace_nodes_me(uuid) TO authenticated;

GRANT EXECUTE ON FUNCTION public.export_find_idempotent_job(uuid, uuid, text) TO service_role;
GRANT EXECUTE ON FUNCTION public.export_create_job_service(uuid, uuid, text, text, text) TO service_role;
GRANT EXECUTE ON FUNCTION public.export_update_job_failure_service(uuid, text) TO service_role;
GRANT EXECUTE ON FUNCTION public.export_update_job_success_service(uuid, text) TO service_role;
