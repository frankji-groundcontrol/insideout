# BUG-008: dedicated-instance assumptions broke against the real, shared-instance target

**Found**: 2026-07-20, during the InsideOut rewrite (P1), setting up the real `DATABASE_URL` against a shared Supabase project and getting the Go server to actually connect and run migrations for the first time.

## The target turned out to be a shared instance, not a dedicated one

The original plan (`docs/plans/2026-07-20-go-rewrite/01-database.md` §2) designed `insideout_app` to **own the whole database** (and therefore `public` within it), on the assumption of a dedicated PostgreSQL instance. The actual target — reached via the `supabase-community` MCP server — is a single Supabase project's Postgres shared with several unrelated tenants (`agentshome`, `botem`, `clawlendar`, `evolver`, `flowers`, `seebyear`, all as sibling schemas alongside `juanleme`/`insideout`). `insideout_app` can only own the `insideout` schema there; `public`, and the database itself, belong to `postgres`/`supabase_admin`.

**Fix**: `insideout_app` is created schema-scoped (`CREATE SCHEMA insideout AUTHORIZATION insideout_app`), never granted anything on `public` beyond the harmless default `USAGE`. Migration #1 (`20260720135749_schema_and_lockdown.sql`) dropped its `REVOKE CREATE ON SCHEMA public FROM PUBLIC` entirely — on a dedicated instance it's a no-op (the owner bypasses grants against the `PUBLIC` pseudo-role regardless), and on this shared instance it would either fail outright (not the owner) or, if it somehow succeeded, be a global cross-tenant change. "Never write to `public`" is enforced by the migrations simply never targeting it, not by an ineffective/risky `REVOKE`.

## Creating the role required a two-step `SET ROLE` dance

**Symptom**: `CREATE SCHEMA insideout AUTHORIZATION insideout_app` (executed as the `postgres` role via the MCP) failed with `ERROR: must be able to SET ROLE "insideout_app"`, even though `postgres` had just created that role.

**Root cause**: Supabase's `postgres` role is not a true superuser (`rolsuper = false`) — it manages roles via `CREATEROLE`, which on role-creation grants the creator membership with `ADMIN OPTION` but not the separate `SET` option (Postgres 16+ split role membership into independent `INHERIT`/`SET`/`ADMIN` grants). `ADMIN OPTION` lets you manage the role; it doesn't let you `SET ROLE` to it.

**Fix**: `GRANT insideout_app TO postgres WITH SET TRUE;` before the `CREATE SCHEMA ... AUTHORIZATION` — `postgres` already had `ADMIN OPTION` on the role it created, which is sufficient to grant itself the `SET` option too.

## Direct connection is IPv6-only and network-restricted from this environment

`db.<project-ref>.supabase.co:5432` only resolves an `AAAA` record (no `A`/IPv4), and the connection attempt from this sandboxed environment got reset mid-handshake — consistent with Supabase's per-project network restrictions blocking unrecognized source IPs, though never fully confirmed since the actual, working path was the pooler instead.

**Fix**: use Supabase's Supavisor pooler instead. Two further wrinkles finding the right one:
- The `aws-0-<region>` pooler generation didn't have this project's tenant registered (`FATAL: (ENOTFOUND) tenant/user ... not found`) — `aws-1-<region>` did. Confirmed by testing the well-known `postgres.<project-ref>` username at each generation prefix before trusting a custom role's connection string.
- Port 5432 (session pooler) also came back with the same generation issue; the working, dashboard-issued string was actually the **transaction pooler** on port **6543** with `?pgbouncer=true`.

## Transaction-pooling (`pgbouncer=true`) breaks pgx's default query mode

**Symptom**: not an error at connection time — this was pre-empted before it could bite, based on knowing PgBouncer's transaction-pooling mode multiplexes many client sessions over few backend connections, so server-side prepared statements from pgx's default extended-protocol caching (`QueryExecModeCacheStatement`) don't reliably survive across statements issued through it.

**Fix**: `internal/store/pool.go`'s `Open()` detects `pgbouncer=true` in the connection string and sets `cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol` for that case only — a dedicated/session-mode Postgres (bundled docker-compose `postgres`, or Supabase's session pooler on 5432) is unaffected either way and keeps the faster default. This is also the direct cause of BUG-007 item #5 (jsonb parameter encoding) — simple protocol mode has no server round-trip to learn a parameter's target column type.

## Why it matters

None of this was visible from the plan documents or from local `docker compose` testing (which uses a real dedicated instance where the original owns-the-whole-database design actually works). It only surfaced by actually pointing `DATABASE_URL` at the real target and trying to connect — a reminder that "the database" is not a fixed abstraction; a self-hosted dedicated instance and a shared managed-Postgres project have genuinely different provisioning models, and code that assumes one can silently misbehave (or simply refuse to connect) against the other.
