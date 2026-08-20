#!/bin/sh
# Bootstrap two NOSUPERUSER roles on a fresh data directory:
#   insideout_owner — owns the insideout schema and SECURITY DEFINER functions
#   insideout_app   — runtime login used by the Go server (subject to RLS)
# The image superuser (POSTGRES_USER) is used only here. DEFINER objects must
# never be owned by it. Idempotent so it can be re-run against an existing volume.
set -e

owner_pw="${INSIDEOUT_OWNER_PASSWORD:?set INSIDEOUT_OWNER_PASSWORD}"
app_pw="${INSIDEOUT_APP_PASSWORD:?set INSIDEOUT_APP_PASSWORD}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
	-v owner_pw="$owner_pw" -v app_pw="$app_pw" <<-EOSQL
SELECT format('CREATE ROLE insideout_owner LOGIN NOSUPERUSER PASSWORD %L', :'owner_pw')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'insideout_owner')
\gexec
SELECT format('CREATE ROLE insideout_app LOGIN NOSUPERUSER PASSWORD %L', :'app_pw')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'insideout_app')
\gexec

ALTER ROLE insideout_owner NOSUPERUSER LOGIN BYPASSRLS;
ALTER ROLE insideout_app NOSUPERUSER LOGIN NOBYPASSRLS;

ALTER DATABASE "$POSTGRES_DB" OWNER TO insideout_owner;
GRANT CONNECT ON DATABASE "$POSTGRES_DB" TO insideout_app;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM insideout_app;
EOSQL
