# postgres-owner-password

`POSTGRES_OWNER_PASSWORD` — the `insideout_owner` role password for
the **bundled docker-compose** Postgres only (setup path b).

## 1. Where the `.key` file is

`hosts/<host>/keys/database/.secrets/postgres-owner-password.key`
(manifest `database/local-stack.yaml`, sibling
[postgres-app-password.md](postgres-app-password.md)). Empty on every
host so far — mint on the next use of the disposable stack.

## 2. How it is propagated

`docker-compose.yml` interpolates it from `.env` at `up` time.
Nothing else reads it.

## 3. How to set it

Self-minted (`openssl rand -hex 32`) when the stack is next used. It
must equal the password embedded in `DATABASE_OWNER_URL` when that DSN
points at the compose stack (`localhost:5442`).

## 4. How to test it

`docker compose up -d` then `migrate` against `localhost:5442` — the
owner role answering proves it; `env.sh check` enforces the DSN
cross-check when the bundled setup is in use.

## 5. How it is used

`docker-compose.yml` creates `insideout_owner` in the ephemeral
container with this password; the owner DSN then authenticates for
migrations.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 5
links here.
