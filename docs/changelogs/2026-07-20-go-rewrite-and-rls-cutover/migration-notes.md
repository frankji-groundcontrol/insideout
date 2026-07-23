# Migration Notes — Operating an Old JuanLeMe Deployment

If you ran the pre-rewrite JuanLeMe (卷了么) stack, here is what changed
underneath you.

## The `juanleme` schema is gone

Its real data (5 users, 4 workspaces, 7 memberships, 1 project) was migrated
into the `insideout` schema; the retired tables (`workshop_nodes`,
`documents`, `document_revisions`, `export_jobs`) and operational telemetry
were archived as JSON **outside the repository**, and then the whole
`juanleme` schema was dropped. There is no rollback path in the database;
recovery of retired-table data means the archive files, not SQL. See
[summary.md](summary.md) and [TODO.md](../../TODO.md) P1.

## Auth moved off Supabase

- The app no longer uses Supabase Auth in any form. All old Supabase
  sessions/tokens are invalid; every user gets a fresh session by logging in
  against the Go server (short-lived JWT + rotating refresh token, both
  httpOnly cookies).
- **Passwords keep working.** Migrated accounts carry their Supabase Auth
  bcrypt hashes; `server/internal/auth/password.go` verifies bcrypt for
  those accounts and silently re-hashes to argon2id on the first successful
  login. No password resets are needed.

## Frontend environment

The Nuxt app no longer reads any of:

- `NUXT_PUBLIC_SUPABASE_*` — the supabase service adapter and
  `@supabase/supabase-js` were deleted.
- `NUXT_PUBLIC_API_MODE` — there is no mode switch; the only backend is the
  Go API via the Nitro same-origin proxy (`/api/v1/**`).

Remove these from your deployment config; they are silently ignored.

## Server environment renames and additions

Authoritative list: `server/internal/config/config.go`.

| Old | New |
|---|---|
| `AI_BASE_URL` | `ANTHROPIC_BASE_URL` |
| `AI_AUTH_TOKEN` | `ANTHROPIC_AUTH_TOKEN` |

Unchanged/new: `DATABASE_URL` (required), `INSIDEOUT_JWT_SECRET` (required,
>= 32 chars), `INSIDEOUT_ADDR`, `INSIDEOUT_ACCESS_TTL`,
`INSIDEOUT_REFRESH_TTL`, `AI_MODEL` (default `claude-sonnet-4-20250514`),
`INSIDEOUT_COOKIE_SECURE`, `INSIDEOUT_DEV_CORS`. docker-compose additionally
uses `POSTGRES_PORT`, `POSTGRES_APP_PASSWORD`, `POSTGRES_SUPERUSER_PASSWORD`,
`SERVER_PORT`, `APP_PORT`.

## Other operational notes

- The `supabase/` directory (edge functions, old migrations, tests) no
  longer exists in the repo; there is nothing to deploy to Supabase.
- Migrations are applied with `go run ./cmd/insideout migrate` against
  `DATABASE_URL`. They work identically on a dedicated instance (bundled
  docker-compose `postgres:17`) or a shared instance where `insideout_app`
  owns only the `insideout` schema
  ([BUG-008](../../issues/2026-07-20-bug-008-shared-instance-db-provisioning.md)).
- If your `DATABASE_URL` goes through a transaction pooler, include
  `pgbouncer=true` so pgx switches to the simple query protocol
  (`server/internal/store/pool.go`).
- Exports are generated on demand and streamed as downloads; Supabase
  Storage is no longer used, so any storage buckets from the old deployment
  are orphaned and can be cleaned up.
