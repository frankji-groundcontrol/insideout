# Postgres DSNs (app + owner)

`DATABASE_URL` / `DATABASE_OWNER_URL` — role-scoped connection strings
for the shared deployment Postgres (Supabase is today's host of the
instance; the kind is just Postgres).

## 1. Where the `.key` files are

`hosts/<host>/keys/database/.secrets/database-app-dsn.key` and
`database-owner-dsn.key` (manifest `database/postgres.yaml`). DSNs
embed their role's password — treat the files as passwords.

## 2. How it is propagated

`.env` → `dev.sh` export → server boot (app DSN) and `migrate` /
`scripts/backup-db.sh` (owner DSN). Deployed values live in the
hosting provider's variable store.

## 3. How to set them

From the managed Postgres instance's connection strings with the two
roles this repo creates: `insideout_app` (runtime) and `insideout_owner`
(migrations). Keep the two passwords distinct; rotating means changing
them at the database **and** in both places (`.env` + provider store)
together.

## 4. How to test them

- Owner DSN: `./scripts/dev.sh -C server go run ./cmd/insideout
  migrate` — must apply (or report "nothing to do, already up to
  date") against the real database.
- App DSN: boot the server and hit any authenticated route (401, not
  500, proves the pool is live).
- `pg_dump` via `scripts/backup-db.sh` exercises the owner DSN too —
  it fails against the app role by design.

## 5. How they are used

The app DSN is the runtime, RLS-subject connection (every request
runs as `insideout_app`); the owner DSN applies schema migrations and
takes backups. Never swap them — the app role must not own the schema.

## 6. Installation guide

[SETENV.md](../../../SETENV.md) Step 1 (both setup paths a/b) and the
role model in [local-development.md](../local-development.md).
