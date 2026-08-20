# 2026-08-19 — insideout_owner + insideout_app

## What changed

Reversed go-rewrite D2 (single role `insideout_app`). Runtime and ownership
are split:

- `insideout_owner` — NOSUPERUSER, `BYPASSRLS`, owns the `insideout` schema
  and `SECURITY DEFINER` helpers. Migrations connect as this role
  (`DATABASE_OWNER_URL`). A bootstrap superuser may apply DDL only after
  `SET LOCAL ROLE insideout_owner`, so DEFINER functions are never
  superuser-owned.
- `insideout_app` — NOSUPERUSER, runtime `DATABASE_URL`. Granted DML and
  EXECUTE; subject to RLS. `workspace_memberships` FORCE RLS is restored
  because `_is_member` now runs as the owner.

Docker init and `server/db/provision_roles.sql` create both roles.
Existing single-role volumes need one superuser `REASSIGN OWNED BY
insideout_app TO insideout_owner` before migrate.

## Verification

- `cd server && go test ./internal/config/`
- `python3 scripts/test_env_catalog.py`
- `cd server && go test ./internal/config/` — pass
- `python3 scripts/test_env_catalog.py` — 47 passed
- Real Postgres: create `insideout_owner` (NOSUPERUSER `BYPASSRLS`) +
  `insideout_app`, migrate as owner, `go test ./internal/store/ -run
  'TestAuthz|TestRoles' -v` as `insideout_app` — all pass. `requireMember`
  no longer uses `FOR KEY SHARE` (zero-row EvalPlanQual under FORCE RLS).

## Plan

[`docs/plans/2026-08-19-owner-app-roles.md`](../plans/2026-08-19-owner-app-roles.md)
