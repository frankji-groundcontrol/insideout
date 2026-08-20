-- One-shot operator script: run as the cluster bootstrap superuser (the
-- docker image's POSTGRES_USER, or the shared-instance admin), never as a
-- role that will own SECURITY DEFINER functions.
--
--   psql -v owner_pw='...' -v app_pw='...' -f server/db/provision_roles.sql
--
-- Then point DATABASE_OWNER_URL at insideout_owner and DATABASE_URL at
-- insideout_app, and run `insideout migrate`.

SELECT format('CREATE ROLE insideout_owner LOGIN NOSUPERUSER PASSWORD %L', :'owner_pw')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'insideout_owner')
\gexec
SELECT format('CREATE ROLE insideout_app LOGIN NOSUPERUSER PASSWORD %L', :'app_pw')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'insideout_app')
\gexec

ALTER ROLE insideout_owner NOSUPERUSER LOGIN BYPASSRLS;
ALTER ROLE insideout_app NOSUPERUSER LOGIN NOBYPASSRLS;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'insideout_app')
     AND EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'insideout_owner') THEN
    -- Move leftover single-role objects. Harmless if app owns nothing.
    BEGIN
      EXECUTE 'REASSIGN OWNED BY insideout_app TO insideout_owner';
    EXCEPTION WHEN OTHERS THEN
      RAISE NOTICE 'REASSIGN OWNED skipped: %', SQLERRM;
    END;
  END IF;
END $$;
