-- rv4 whole-change review (rv4-FR-F37, F22, F34, F23, F33): the rest of
-- DATA-01. 20261018140000 made a membership INSERT need a reason (the join
-- code for a member, the workspace creator for an admin) but left two doors
-- that reach the same state without it. This is a forward migration: that
-- file already ran on every database that took the wave-2 build, and an
-- edited file would not run there again. Idempotent: functions are CREATE OR
-- REPLACE, triggers are dropped before they are created. Runs as
-- insideout_owner (the migrate runner's role guard).

-- 1. UPDATE (F37, F22, F34). workspace_memberships_update is admin-only
-- USING with no column limit, and insideout_app has a table-wide UPDATE, so
-- an admin could re-point user_id (or workspace_id) of an existing row and put
-- any user into the workspace with no join code, no consent and no creator
-- check, or promote them in the same statement. The role is the only thing a
-- membership UPDATE has a reason to change (UpdateMemberRole); every other
-- column, one added later included, has to come out unchanged. A trigger and
-- not a column grant, for the reason in 20261018140000 (users): a column
-- grant is checked before the statement reads a row and would turn "no row"
-- for a member into "permission denied". A role RLS does not apply to
-- (insideout_owner, a superuser) is trusted with every column, the same line
-- RLS draws.
CREATE OR REPLACE FUNCTION insideout._pin_membership_columns()
RETURNS trigger
LANGUAGE plpgsql SET search_path = insideout, pg_catalog
AS $$
DECLARE
  v_editable constant text[] := ARRAY['role', 'updated_at'];
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND (rolbypassrls OR rolsuper)) THEN
    RETURN NEW;
  END IF;
  IF to_jsonb(NEW) - v_editable IS DISTINCT FROM to_jsonb(OLD) - v_editable THEN
    RAISE EXCEPTION 'insideout.workspace_memberships: this role may only change role'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  RETURN NEW;
END
$$;

DROP TRIGGER IF EXISTS workspace_memberships_pin_columns ON insideout.workspace_memberships;
CREATE TRIGGER workspace_memberships_pin_columns BEFORE UPDATE ON insideout.workspace_memberships
  FOR EACH ROW EXECUTE FUNCTION insideout._pin_membership_columns();

-- 2. The creator is a fact, not a permission (F23, F33). The admin branch of
-- _may_insert_membership trusted workspaces.creator_id for the life of the
-- workspace, but that column outlives the creator's membership (a creator who
-- left, or was removed, is still named by it) and workspaces_update let any
-- admin rewrite it. So a removed creator could insert themselves back as
-- admin with no join code, and an admin could make themselves "the creator"
-- and do the same after being removed, and then also pass the creator-only
-- workspaces_delete policy. Two changes:
--   a. creator_id cannot change (no flow does; DeleteWorkspace and the admin
--      branch only read it). An UPDATE that does not name creator_id never
--      fires the trigger, and writing the same value back passes it.
--   b. the admin row is for the first member of a workspace, which is what
--      createWorkspaceTx writes in the transaction that creates it: the
--      creator, and only while nobody belongs yet. A workspace that has
--      people gets new admins by promotion (UpdateMemberRole), never by
--      insert. The member branch is unchanged.
CREATE OR REPLACE FUNCTION insideout._pin_workspace_creator()
RETURNS trigger
LANGUAGE plpgsql SET search_path = insideout, pg_catalog
AS $$
BEGIN
  IF NEW.creator_id IS DISTINCT FROM OLD.creator_id
     AND NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND (rolbypassrls OR rolsuper)) THEN
    RAISE EXCEPTION 'insideout.workspaces: creator_id cannot be changed'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  RETURN NEW;
END
$$;

DROP TRIGGER IF EXISTS workspaces_pin_creator ON insideout.workspaces;
CREATE TRIGGER workspaces_pin_creator BEFORE UPDATE OF creator_id ON insideout.workspaces
  FOR EACH ROW EXECUTE FUNCTION insideout._pin_workspace_creator();

-- Same signature and attributes as in 20261018140000 (the policy keeps
-- pointing at it); only the admin branch gained the "nobody belongs yet" test.
CREATE OR REPLACE FUNCTION insideout._may_insert_membership(p_workspace_id uuid, p_user_id uuid, p_role text)
RETURNS boolean
LANGUAGE sql SECURITY DEFINER SET search_path = insideout, pg_catalog STABLE
AS $$
  SELECT EXISTS (
    SELECT 1 FROM insideout.workspaces w
    WHERE w.id = p_workspace_id
      AND (
        (p_role = 'admin' AND w.creator_id = p_user_id
            AND NOT EXISTS (SELECT 1 FROM insideout.workspace_memberships m WHERE m.workspace_id = w.id))
        OR (p_role = 'member' AND w.status = 'active'
            AND w.code = NULLIF(current_setting('app.join_code', true), ''))
      )
  )
$$;
