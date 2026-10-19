-- rv4 wave 3 (DATA-03, read half): users.password_hash leaves the app role's
-- table reads. Idempotent: REVOKE and GRANT end in the same state when re-run,
-- the function is CREATE OR REPLACE. Runs as insideout_owner (the migrate
-- runner's role guard).

-- 20261018140000 closed the write half (users_pin_columns, the INSERT column
-- grant). The read half stayed open: users_select shows a user every
-- co-member's row, and 20260819190000 gave insideout_app table-wide SELECT, so
-- any co-member's stored credential was one `SELECT password_hash` away at the
-- database layer. No API response carries it, so this is defence in depth: it
-- takes the hash out of reach of a read bug (SELECT *, a join, a RETURNING, a
-- dump) and of any SQL the app role runs on the strength of a bug upstream.
--
-- A table-level privilege masks column grants, so the table-wide SELECT goes
-- first (the REVOKE also clears any column grant made earlier, which is why
-- re-running this file ends in the same state) and every column but
-- password_hash is granted by name. UPDATE stays table-wide, pinned by
-- users_pin_columns: `SET password_hash = $2 WHERE id = $1` needs only UPDATE
-- on the column and SELECT on id. A column added later needs its own GRANT in
-- its own migration; users_read_grant_test.go fails until it has one.
REVOKE SELECT ON insideout.users FROM insideout_app;
GRANT SELECT (id, email, username, avatar_url, bio, keywords, role, system_role,
              email_verified_at, meta, created_at, updated_at)
  ON insideout.users TO insideout_app;

-- Login still has to verify a password, and it is the only flow that does.
-- insideout._credential_for_email is the one door to the stored credential: a
-- SECURITY DEFINER function owned by insideout_owner (which can read the
-- column), search_path pinned (pg_temp last) so nothing a caller creates can
-- shadow a name inside it, EXECUTE for insideout_app only (PUBLIC holds
-- EXECUTE on every new function until it is revoked). It answers only in the
-- pre-auth system context, app.user_id unset: the context login and the CLI
-- run in, and the one users_select already trusts. Inside a request's own
-- transaction it returns no row, so a handler that forgets this cannot read a
-- co-member's hash by passing their email. The context is a convention the app
-- role could still satisfy (set_config('app.user_id', '', true)); the gain is
-- that no table read, SELECT *, join or RETURNING can expose the hash and
-- exactly one named path to it is left to audit.
CREATE OR REPLACE FUNCTION insideout._credential_for_email(p_email text)
RETURNS TABLE (id uuid, password_hash text)
LANGUAGE sql SECURITY DEFINER SET search_path = insideout, pg_temp STABLE
AS $$
  SELECT u.id, u.password_hash FROM insideout.users u
  WHERE u.email = p_email AND insideout.current_user_id() IS NULL
$$;

REVOKE ALL ON FUNCTION insideout._credential_for_email(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION insideout._credential_for_email(text) TO insideout_app;

-- Fail the migration, not production, when the column is still readable
-- through a route the REVOKE above cannot reach (an inherited role membership,
-- a grant to PUBLIC): the finding would otherwise ship as closed.
DO $$
BEGIN
  IF has_column_privilege('insideout_app', 'insideout.users', 'password_hash', 'SELECT') THEN
    RAISE EXCEPTION 'insideout_app can still SELECT insideout.users.password_hash (inherited from a role or PUBLIC?)';
  END IF;
END
$$;
