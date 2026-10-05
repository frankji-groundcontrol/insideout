# database-app-dsn

`DATABASE_URL` — the runtime Postgres DSN, `insideout_app` role, for
the shared deployment instance (Supabase is today's host; the kind is
just Postgres).

## 1. Where the `.key` file is

`hosts/<host>/keys/database/.secrets/database-app-dsn.key` (manifest
`database/postgres.yaml`, sibling key
[database-owner-dsn.md](database-owner-dsn.md)). The DSN embeds the
role's password — treat the file as a password.

## 2. How it is propagated

`.env` → `dev.sh` export → the server's pool at boot. Deployed value
rides the hosting provider's variable store.

## 3. How to set it

From the managed instance's connection string for the `insideout_app`
role (the repo creates it; keep its password distinct from the owner
role's). Rotating means changing it at the database **and** in both
consumers (`.env` + provider store) together.

## 4. How to test it

Boot the server and hit any authenticated route — 401 (not 500)
proves the pool is live and the role answers.

## 5. How it is used

The runtime, RLS-subject connection — every request runs as
`insideout_app`. It must never own the schema; that is the owner
DSN's job.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 1
links here. Contract row in
[environment.md](../../environment.md).
