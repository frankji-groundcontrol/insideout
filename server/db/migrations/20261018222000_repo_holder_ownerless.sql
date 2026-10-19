-- rv4-FR-F193 / F205 / F200 / F121: a project holds its repo whether or not it
-- has an owner. Idempotent (CREATE OR REPLACE, same signature and grant), runs
-- as insideout_owner like every migration, and goes forward: 20261018180000
-- already exists on databases that took it, so it is not edited in place.
--
-- insideout._projects_by_repo answers three questions with one row: which
-- project a delivery goes to (store.ProjectsByRepo), which project holds a repo
-- when another one asks to bind it (claimRepo), and whether a sync may use the
-- installation token of the repo (commitToken). 20261018180000 added
-- `AND p.owner_id IS NOT NULL` to it, so a project that kept its repo_url and
-- lost its owner stopped being the holder: a second project could bind the
-- same repo, and the deliveries moved to it. PATCH /projects/{pid} clears the
-- owner when no ownerId is sent, and RemoveMember leaves owner_id pointing at
-- somebody who is no longer a member, so both states are reachable.
--
-- The holder is now the project that carries the repo and was created first (id
-- breaks a tie), owned or not. The second column is the user a delivery acts
-- as: the project's owner while that owner is still a member of the project's
-- workspace, NULL otherwise (no owner, or an owner who left). A delivery has no
-- identity of its own, and everything done as a non-member is "not found" under
-- RLS, so the webhook skips a holder whose actor is NULL instead of running a
-- sync that can only fail. An admin can still sync it by hand and can still
-- release the link.
CREATE OR REPLACE FUNCTION insideout._projects_by_repo(p_repo_url text)
RETURNS TABLE (project_id uuid, owner_id uuid)
LANGUAGE sql
SECURITY DEFINER
SET search_path = insideout, pg_catalog
STABLE
AS $$
  SELECT p.id, m.user_id
  FROM insideout.projects p
  LEFT JOIN insideout.workspace_memberships m
    ON m.workspace_id = p.workspace_id AND m.user_id = p.owner_id
  WHERE p_repo_url <> ''
    AND p.repo_url <> ''
    AND lower(p.repo_url) = lower(p_repo_url)
  ORDER BY p.created_at, p.id
  LIMIT 1
$$;

GRANT EXECUTE ON FUNCTION insideout._projects_by_repo(text) TO insideout_app;
