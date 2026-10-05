# database-owner-dsn

`DATABASE_OWNER_URL` — the migration Postgres DSN, `insideout_owner`
role (NOSUPERUSER — never a superuser for DEFINER objects).

## 1. Where the `.key` file is

`hosts/<host>/keys/database/.secrets/database-owner-dsn.key` (manifest
`database/postgres.yaml`, sibling key
[database-app-dsn.md](database-app-dsn.md)). Embeds the role's
password.

## 2. How it is propagated

`.env` → `dev.sh` export → `migrate` and `scripts/backup-db.sh`.
Deployed value in the provider's variable store. Empty is legal: it
falls back to `DATABASE_URL`, which must then already be the owner.

## 3. How to set it

From the managed instance's connection string for the
`insideout_owner` role. Rotate together with the app DSN (change at
the database, then both consumers).

## 4. How to test it

`./scripts/dev.sh -C server go run ./cmd/insideout migrate` — must
apply or report "nothing to do, already up to date" against the real
database. `scripts/backup-db.sh` exercises it too (pg_dump fails
against the app role by design).

## 5. How it is used

Applies schema migrations and takes backups — the only DSN with DDL
rights, which is why it exists separately from the runtime role.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 1
links here. Contract row in
[environment.md](../../environment.md).
