-- rv4-COMP-05: a GitHub repo binds to one project. Idempotent: functions are
-- CREATE OR REPLACE, the rewrite only touches rows that are not yet canonical,
-- the index is IF NOT EXISTS. Runs as insideout_owner (the migrate runner's
-- role guard), so the UPDATE below sees every workspace's projects.
--
-- The webhook found projects by exact text (`repo_url = p_repo_url`) and every
-- project holding that text received the delivery, so two owners who had typed
-- the same repo each received the other's commit lines. The lookup now ignores
-- letter case, names at most one project, and the PUT handler refuses a second
-- binding (store.SetProjectRepo). This file prepares the rows that already
-- exist for that.

-- 1. The canonical spelling, in SQL. Mirrors github.ParseRepoURL followed by
-- github.CanonicalRepoURL line by line, because the rewrite below must produce
-- exactly the string the Go side would have stored, and refuse exactly what it
-- refuses: surrounding Unicode white space (strings.TrimSpace), one trailing
-- "/" and one trailing ".git", a scheme, a host that must be github.com (after
-- a scheme, or when the first segment holds a dot), then owner/repo made of
-- [A-Za-z0-9._-] and not of dots alone. NULL means "not a repo URL". The case
-- is kept as typed. TestCanonicalRepoURLInSQLMatchesGo runs one corpus through
-- both and demands the same answer.
CREATE OR REPLACE FUNCTION insideout._canonical_repo_url(p_url text)
RETURNS text
LANGUAGE plpgsql IMMUTABLE STRICT
SET search_path = insideout, pg_catalog
AS $$
DECLARE
  c_ws constant text := '[\t\n\v\f\r \u0085   -     　]+';
  s text := regexp_replace(regexp_replace(p_url, '^' || c_ws, ''), c_ws || '$', '');
  scheme boolean := false;
  pos integer;
  parts text[];
BEGIN
  IF right(s, 1) = '/' THEN s := left(s, -1); END IF;
  IF right(s, 4) = '.git' THEN s := left(s, -4); END IF;
  pos := strpos(s, '://');
  IF pos > 0 THEN
    s := substr(s, pos + 3);
    scheme := true;
  END IF;
  parts := regexp_split_to_array(s, '/');
  IF scheme OR strpos(parts[1], '.') > 0 THEN
    -- ASCII-only case folding: lower() follows the database locale, which
    -- would not agree with strings.EqualFold under a Turkish one.
    IF translate(parts[1], 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz') <> 'github.com' THEN
      RETURN NULL;
    END IF;
    parts := parts[2:cardinality(parts)];
  END IF;
  IF cardinality(parts) < 2 THEN
    RETURN NULL;
  END IF;
  FOR i IN 1..2 LOOP
    IF parts[i] !~ '^[A-Za-z0-9._-]+$' OR btrim(parts[i], '.') = '' THEN
      RETURN NULL;
    END IF;
  END LOOP;
  RETURN 'https://github.com/' || parts[1] || '/' || parts[2];
END
$$;

GRANT EXECUTE ON FUNCTION insideout._canonical_repo_url(text) TO insideout_app;

-- 2. Rewrite what was stored before the PUT handler stored the canonical form
-- (owner/repo, .git, a trailing slash, a pasted /tree/main). Only rows the
-- function accepts and that are not already canonical change; a value it
-- refuses (a foreign host, a dot-only segment) is left exactly as stored, and
-- the sync answers it with a 400 instead of reading anything. This does not
-- merge or delete projects: rows that now share a repo keep sharing it, and
-- step 3 makes the earliest-created one the only recipient. The row's
-- updated_at moves with the rewrite (the set_updated_at trigger).
UPDATE insideout.projects p
SET repo_url = c.canonical
FROM (
  SELECT id, insideout._canonical_repo_url(repo_url) AS canonical
  FROM insideout.projects
  WHERE repo_url <> ''
) c
WHERE p.id = c.id
  AND c.canonical IS NOT NULL
  AND c.canonical <> p.repo_url;

-- 3. The webhook lookup: case-insensitive, and at most one project, the one
-- created first (id breaks a tie), so the answer does not depend on row order.
-- store.SetProjectRepo asks this same function who holds a repo before it
-- binds another project to it, so "who receives deliveries" and "who holds the
-- repo" cannot disagree. The empty URL never matches: an unlinked project has
-- repo_url = '' and must not read as "every project without a repo". Same
-- signature and grant as 20260821120000_github_webhook_repo_lookup.sql.
CREATE OR REPLACE FUNCTION insideout._projects_by_repo(p_repo_url text)
RETURNS TABLE (project_id uuid, owner_id uuid)
LANGUAGE sql
SECURITY DEFINER
SET search_path = insideout, pg_catalog
STABLE
AS $$
  SELECT p.id, p.owner_id
  FROM insideout.projects p
  WHERE p_repo_url <> ''
    AND p.repo_url <> ''
    AND lower(p.repo_url) = lower(p_repo_url)
    AND p.owner_id IS NOT NULL
  ORDER BY p.created_at, p.id
  LIMIT 1
$$;

GRANT EXECUTE ON FUNCTION insideout._projects_by_repo(text) TO insideout_app;

-- The lookup ran as a sequential scan of every project per delivery and now
-- also per bind. Partial, because most projects have no repo. Not UNIQUE:
-- hosted rows may already share a repo (see step 2), and one-per-repo is
-- enforced under an advisory lock in store.SetProjectRepo instead.
CREATE INDEX IF NOT EXISTS projects_repo_url_lower_idx
  ON insideout.projects (lower(repo_url))
  WHERE repo_url <> '';
