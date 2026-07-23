#!/bin/sh
# Creates the non-superuser insideout_app role and makes it the OWNER of
# the insideout database (and therefore of public within it, per Postgres
# default) — matching the provisioning model in
# docs/plans/2026-07-20-go-rewrite/01-database.md §2. Runs once, only
# against a fresh data directory, via the official postgres image's
# docker-entrypoint-initdb.d hook.
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
	CREATE ROLE insideout_app WITH LOGIN PASSWORD '$INSIDEOUT_APP_PASSWORD';
	ALTER DATABASE "$POSTGRES_DB" OWNER TO insideout_app;
EOSQL
