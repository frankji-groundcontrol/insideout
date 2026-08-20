-- Split insideout_owner (schema + SECURITY DEFINER) from insideout_app
-- (runtime). DEFINER functions must not be owned by a superuser; the
-- migrate runner SET LOCAL ROLE insideout_owner before this file runs.
--
-- Restores FORCE RLS on workspace_memberships: _is_member/_is_admin now
-- run as the table owner (insideout_owner), so they no longer recurse
-- under FORCE the way they did when insideout_app owned both the table
-- and the DEFINER helpers (see 20260720153000).

DO $$
BEGIN
  IF current_user <> 'insideout_owner' THEN
    RAISE EXCEPTION 'must run as insideout_owner, not %', current_user;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'insideout_owner' AND rolsuper) THEN
    RAISE EXCEPTION 'insideout_owner must not be a superuser';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'insideout_owner' AND rolbypassrls) THEN
    RAISE EXCEPTION 'insideout_owner must have BYPASSRLS (not superuser) so SECURITY DEFINER helpers can read through FORCE RLS';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'insideout_app') THEN
    RAISE EXCEPTION 'role insideout_app does not exist';
  END IF;
END $$;

ALTER SCHEMA insideout OWNER TO insideout_owner;

-- Leftover objects from the single-role era (insideout_app owned everything).
-- Requires this session to be able to reassign; on a dedicated instance the
-- bootstrap superuser SET ROLE path in the migrate runner handles new
-- clusters. Existing volumes need one superuser REASSIGN OWNED first.
DO $$
DECLARE
  obj record;
BEGIN
  FOR obj IN
    SELECT c.relname AS name, c.relkind
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_roles r ON r.oid = c.relowner
    WHERE n.nspname = 'insideout' AND r.rolname <> 'insideout_owner'
      AND c.relkind IN ('r', 'S', 'v', 'm', 'p')
  LOOP
    RAISE EXCEPTION 'insideout.% is not owned by insideout_owner. As a bootstrap superuser run REASSIGN OWNED BY <current owner> TO insideout_owner, then re-migrate.',
      obj.name;
  END LOOP;
END $$;

GRANT USAGE ON SCHEMA insideout TO insideout_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA insideout TO insideout_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA insideout TO insideout_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA insideout TO insideout_app;

ALTER DEFAULT PRIVILEGES FOR ROLE insideout_owner IN SCHEMA insideout
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO insideout_app;
ALTER DEFAULT PRIVILEGES FOR ROLE insideout_owner IN SCHEMA insideout
  GRANT USAGE, SELECT ON SEQUENCES TO insideout_app;
ALTER DEFAULT PRIVILEGES FOR ROLE insideout_owner IN SCHEMA insideout
  GRANT EXECUTE ON FUNCTIONS TO insideout_app;

REVOKE CREATE ON SCHEMA insideout FROM insideout_app;
REVOKE ALL ON SCHEMA insideout FROM PUBLIC;

-- Membership helpers are DEFINER-as-owner, so FORCE is safe again.
ALTER TABLE insideout.workspace_memberships FORCE ROW LEVEL SECURITY;
