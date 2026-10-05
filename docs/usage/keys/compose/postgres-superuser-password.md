# postgres-superuser-password

`POSTGRES_SUPERUSER_PASSWORD` — bootstrap password of the local
Postgres superuser (docker-compose). Config companion: blank =
compose's local dev default; the file exists so the kind is always
findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/compose/.secrets/postgres-superuser-password.key`
(manifest `compose/stack.yaml`).

## 2. How it is propagated

`.env` → docker-compose interpolation of the db service's
`POSTGRES_PASSWORD`. Applied only when the data volume is first
created.

## 3. How to set it

Any string for a local-only stack. Blank uses the compose `:-` dev
default. Changing it later does nothing until the volume is
recreated.

## 4. How to test it

`scripts/tests/keys/compose/test_postgres-superuser-password.py`
(exists + non-placeholder when non-empty). Live: `psql` as the
superuser with the value.

## 5. How it is used

Superuser access to the local dev database for admin tasks. The
application connects with the
[postgres-app-password](../database/postgres-app-password.md), not
this value; production runs on the managed database behind
[database-app-dsn](../database/database-app-dsn.md).

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
