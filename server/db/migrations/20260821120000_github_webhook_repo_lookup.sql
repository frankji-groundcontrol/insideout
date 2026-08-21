-- GitHub webhook repo lookup without a user session. Deliveries arrive
-- unauthenticated (HMAC-verified at the API layer) while RLS scopes
-- projects to workspace members, so the app role would see nothing. A
-- DEFINER helper owned by insideout_owner resolves repo URL -> (project,
-- owner); the API layer then re-runs the normal per-project sync as that
-- owner, keeping every write inside regular RLS.
CREATE OR REPLACE FUNCTION insideout._projects_by_repo(p_repo_url text)
RETURNS TABLE (project_id uuid, owner_id uuid)
LANGUAGE sql
SECURITY DEFINER
SET search_path = insideout, pg_catalog
STABLE
AS $$
  SELECT p.id, p.owner_id
  FROM insideout.projects p
  WHERE p.repo_url = p_repo_url
    AND p.owner_id IS NOT NULL
$$;

GRANT EXECUTE ON FUNCTION insideout._projects_by_repo(text) TO insideout_app;
