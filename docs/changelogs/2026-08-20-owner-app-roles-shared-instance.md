# 2026-08-20 — Owner/app roles rolled out to the shared Supabase instance

The two-role model from
[2026-08-19](2026-08-19-owner-app-roles.md) is now live on the shared
Supabase instance (the one Railway `server` runs on). Plan:
[owner-app-roles](../plans/2026-08-19-owner-app-roles.md).

## What was applied

Everything ran as the instance admin (Supabase MCP `postgres` session) in
one transaction, verified after commit:

- `CREATE ROLE insideout_owner LOGIN NOSUPERUSER BYPASSRLS` (no login
  password yet — usable via `SET ROLE` only), `GRANT insideout_owner TO
  postgres` for admin sessions.
- Scoped ownership transfer: `insideout` schema, all 16 tables, and all 12
  functions to `insideout_owner` (indexes, TOAST, and composite row types
  follow their tables). A blanket `REASSIGN OWNED BY insideout_app` was
  deliberately avoided — the instance hosts other applications' roles.
- `20260819190000_owner_app_grants.sql` applied verbatim after `SET ROLE
  insideout_owner`, and recorded in `insideout.schema_migrations` by exact
  filename, matching how `store.Migrate` keeps its ledger (17 rows now).
  This restores FORCE RLS on `workspace_memberships`.

## Deviations from `provision_roles.sql` worth knowing

- PG 17 rejects `GRANT insideout_owner TO postgres WITH ADMIN OPTION`
  ("ADMIN option cannot be granted back to your own grantor"); a plain
  membership grant is sufficient for admin `SET ROLE`.
- The operator script's psql-variable passwords were skipped: the owner
  role intentionally has no password at the time of writing (see
  follow-ups). The script remains the reference for dedicated/docker
  instances.

## Verification

- Instance queries: 0 relations and 0 functions in `insideout` still owned
  by another role; `workspace_memberships` has FORCE RLS; owner role is
  `NOSUPERUSER + BYPASSRLS + LOGIN`; 17 migrations recorded.
- Runtime as `insideout_app` (psql via the working pooler DSN): SELECT on
  `schema_migrations` returns 17; `workspaces` returns 0 rows under RLS
  without error; `_is_member(...)` (SECURITY DEFINER, owner-owned)
  executes and returns a boolean.
- Hosted Railway API: `GET /healthz` → `{"status":"ok"}` after the
  cutover. The running server was never restarted.
- Local `.env` `DATABASE_URL` replaced with the known-good app DSN
  (5432 session pooler); `scripts/env.sh check server` passes. This
  resolves the SASL failure noted in `docs/HANDOFF.md`.

The DB-gated `authz_test.go` battery was not re-run against the shared
instance (it writes test data); its prior real-Postgres run is recorded in
the 2026-08-19 changelog.

## Follow-ups (resolved the same day)

1. **Done**: `insideout_owner` login password set through the admin MCP
   session (user-authorized; the generated value now exists only in both
   machines' `.env` and Railway's `DATABASE_OWNER_URL` — never committed).
   Railway `server` received `DATABASE_OWNER_URL` (session pooler, 5432).
2. **Deploy gotcha recorded**: `railway redeploy` re-runs the last
   successful *image* — here the 2026-08-18 build, whose migrate still
   executed on the main pool as `insideout_app`. With CREATE revoked from
   the app role by the grants migration, both redeploys crash-looped
   (`permission denied for schema insideout`). The fix was deploying
   current `main` with `railway up --service server`: deployment SUCCESS,
   boot reached `listening`, `/healthz` 200. Autodeploy is off — ship
   server changes with `railway up --service server`.
3. Optional, still open: rotate the `insideout_app` password (it was
   echoed once in a local agent transcript during diagnosis) and update
   Railway `DATABASE_URL` plus both `.env` files in the same pass.
4. Hardening idea (not done): let `Migrate` pass as `insideout_app` when
   zero migrations are pending, so a missing owner URL degrades to a
   warning instead of a boot failure.
