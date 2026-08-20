# 2026-08-19 — insideout_owner + insideout_app (no superuser DEFINER)

Status: **in flight**. Reverses go-rewrite D2 (single role `insideout_app`).

## Why

User: use both `insideout_owner` and `insideout_app`; `SECURITY DEFINER`
must not be owned by a superuser.

D2 made `insideout_app` own the schema *and* connect at runtime. DEFINER
helpers (`_is_member`, …) then ran as the table owner, so FORCE RLS still
applied inside them and membership policies recursed. The workaround was
`NO FORCE` on `workspace_memberships`, which drops RLS for the connecting
role.

Correct model:

| Role | Superuser | Login | Owns | Connects |
| --- | --- | --- | --- | --- |
| bootstrap (`postgres` image user) | yes | yes | nothing in `insideout` | init only |
| `insideout_owner` | **no** | yes (migrate) | schema, tables, DEFINER functions | `DATABASE_OWNER_URL` |
| `insideout_app` | no | yes | nothing | `DATABASE_URL` (Go runtime) |

DEFINER functions keep `SET search_path = insideout, pg_catalog` and are
owned by `insideout_owner`, so they bypass RLS as the table owner without
cluster-wide superuser rights. `insideout_app` is subject to RLS, including
`workspace_memberships` FORCE again.

## Checklist

- [x] Open this plan; reverse D2 in architecture docs
- [x] Docker init creates both NOSUPERUSER roles
- [x] `DATABASE_OWNER_URL` for migrate; `DATABASE_URL` stays the app
- [x] Migrate refuses to apply DDL as a superuser (SET ROLE owner)
- [x] Migration: grants, default privileges, FORCE membership RLS
- [x] Real Postgres: authz tests as `insideout_app`; role-owner assertions
- [x] Operator provision documented (`server/db/provision_roles.sql`)

## Sources

- User request this session
- [BUG-007](../issues/2026-07-20-bug-007-rls-against-real-postgres.md) recursion
- Historical D2: [01-database.md](2026-07-20-go-rewrite/01-database.md) §2
