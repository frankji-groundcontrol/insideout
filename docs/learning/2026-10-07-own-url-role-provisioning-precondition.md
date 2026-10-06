# 2026-10-07 — The own-URL path's hidden precondition: provisioned roles with owner BYPASSRLS

## What was learned

Migrating a fresh own-URL database fails in two distinct, misleading
ways unless the database was provisioned the way
`docker/postgres-init/001-create-app-role.sh` provisions the bundled
one:

1. **`role "insideout_owner" does not exist`** — migrations run
   `SET LOCAL ROLE insideout_owner` (the two-role model: DEFINER
   objects must never be owned by a superuser), so the role must
   pre-exist on any database the migrations touch.
2. **`infinite recursion detected in policy for relation
   "workspace_memberships"`** — subtler: with the role present but
   missing **BYPASSRLS**, `insideout_owner` is itself subject to the
   very recursive RLS policy the migration edits, and PostgreSQL
   detects the recursion while applying the fix for it.

The bundled Docker path hides both because the init script runs
first. `setup.sh --postgres own` has always had the same hole; the
setup TUI's live exercise is what surfaced it.

## Evidence

[Setup TUI slice-2 changelog](../changelogs/2026-10-07-setup-tui-slice-2.md):
first live run failed with (1); after creating the roles without
BYPASSRLS it failed with (2); after
`ALTER ROLE insideout_owner BYPASSRLS` (matching the init script
exactly) the run applied all 23 migrations and booted.

## Scope

Any path that migrates an InsideOut database by URL — `setup.sh`,
`insideout setup --local --postgres own`, CI against a fresh
database, a new operator's self-hosted Postgres.

## When to apply

Before migrating a fresh own-URL database, reproduce the init
script's contract: two NOSUPERUSER LOGIN roles, owner with BYPASSRLS
owning the database, app with CONNECT and no public-schema rights.
Better: make the tool check and say so —
[issue](../issues/2026-10-07-own-url-fresh-db-bootstrap.md).
