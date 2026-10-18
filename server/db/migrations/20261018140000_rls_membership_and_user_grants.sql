-- rv4 wave 2 (DATA-01, DATA-03): the write side of row-level security.
-- Idempotent: functions are CREATE OR REPLACE, policies and triggers are
-- dropped before they are created. Runs as insideout_owner (the migrate
-- runner's role guard).

-- DATA-01. workspace_memberships_insert only checked
-- `user_id = current_user_id()`, so any signed-in session could INSERT itself
-- as an ADMIN of any workspace it knew the id of (workspace ids are returned
-- to members and appear in URLs and exports): admins read and write every
-- row of the workspace. A membership row now needs a reason, and the reason
-- depends on the role:
--   admin  only the workspace's creator, which is what createWorkspaceTx
--          writes right after inserting the workspace;
--   member only a caller holding the workspace's join code in app.join_code,
--          which is what JoinWorkspace sets (and workspaces_select already
--          reads for the same purpose).
-- It lives in a SECURITY DEFINER helper, like _is_member and _is_admin, so
-- reading workspaces from inside a workspace_memberships policy cannot recurse
-- through workspaces_select (see 20260720151500). The column arguments are
-- qualified with the table name for the alias reason in 20260720155500.
CREATE OR REPLACE FUNCTION insideout._may_insert_membership(p_workspace_id uuid, p_user_id uuid, p_role text)
RETURNS boolean
LANGUAGE sql SECURITY DEFINER SET search_path = insideout, pg_catalog STABLE
AS $$
  SELECT EXISTS (
    SELECT 1 FROM insideout.workspaces w
    WHERE w.id = p_workspace_id
      AND (
        (p_role = 'admin' AND w.creator_id = p_user_id)
        OR (p_role = 'member' AND w.status = 'active'
            AND w.code = NULLIF(current_setting('app.join_code', true), ''))
      )
  )
$$;

DROP POLICY IF EXISTS workspace_memberships_insert ON insideout.workspace_memberships;
CREATE POLICY workspace_memberships_insert ON insideout.workspace_memberships
  FOR INSERT WITH CHECK (
    user_id = insideout.current_user_id()
    AND insideout._may_insert_membership(
      insideout.workspace_memberships.workspace_id,
      insideout.current_user_id(),
      insideout.workspace_memberships.role)
  );

-- DATA-03. 20260819190000 gave insideout_app table-wide INSERT and UPDATE on
-- insideout.users, while users_insert is WITH CHECK (true) and users_update
-- only pins the row to its owner. Nothing said which columns the app may
-- write, so one statement could create an admin (INSERT ... system_role =
-- 'admin') or promote itself (UPDATE ... SET system_role = 'admin'), and
-- rewrite role, email_verified_at, email or meta: the columns that gate
-- /api/v1/admin and sign-in.
--
-- INSERT is limited by column grants. CreateUser writes email, password_hash
-- and username and nothing else; every other column, including one added
-- later, is refused with 42501 before a row exists. The REVOKE comes first:
-- a table-level privilege masks column grants, and the REVOKE also clears any
-- column grant made earlier, so re-running this file ends in the same state.
REVOKE INSERT ON insideout.users FROM insideout_app;
GRANT INSERT (email, password_hash, username) ON insideout.users TO insideout_app;

-- UPDATE is limited by a pinning trigger, not by column grants. A column grant
-- is checked before the statement reads a row, so promote-admin run without
-- DATABASE_OWNER_URL (an UPDATE of someone else's system_role on the app pool,
-- which RLS answers with "no row", which the CLI turns into its "set
-- DATABASE_OWNER_URL" hint) would start failing with "permission denied". The
-- trigger only sees rows RLS already let through. It compares the new row with
-- the old one after dropping the columns a profile edit really changes
-- (UpdateProfile, UpdatePasswordHash; updated_at belongs to set_updated_at);
-- anything else, a column added later included, has to come out unchanged.
-- A role RLS does not apply to (insideout_owner, which runs migrations and
-- promote-admin; a superuser) is trusted with every column: the same line
-- RLS draws.
CREATE OR REPLACE FUNCTION insideout._pin_user_columns()
RETURNS trigger
LANGUAGE plpgsql SET search_path = insideout, pg_catalog
AS $$
DECLARE
  v_editable constant text[] := ARRAY['username', 'avatar_url', 'bio', 'keywords', 'password_hash', 'updated_at'];
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND (rolbypassrls OR rolsuper)) THEN
    RETURN NEW;
  END IF;
  IF to_jsonb(NEW) - v_editable IS DISTINCT FROM to_jsonb(OLD) - v_editable THEN
    RAISE EXCEPTION 'insideout.users: this role may only change username, avatar_url, bio, keywords and password_hash'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  RETURN NEW;
END
$$;

DROP TRIGGER IF EXISTS users_pin_columns ON insideout.users;
CREATE TRIGGER users_pin_columns BEFORE UPDATE ON insideout.users
  FOR EACH ROW EXECUTE FUNCTION insideout._pin_user_columns();
