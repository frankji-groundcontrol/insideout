# postgres-app-password

`POSTGRES_APP_PASSWORD` — the `insideout_app` role password for the
**bundled docker-compose** Postgres only (setup path b in SETENV.md).

## 1. Where the `.key` file is

`hosts/<host>/keys/database/.secrets/postgres-app-password.key`
(manifest `database/local-stack.yaml`, sibling
[postgres-owner-password.md](postgres-owner-password.md)). Separate
from the DSN keys because it belongs to a different installation.

## 2. How it is propagated

`docker-compose.yml` interpolates it from `.env` at `up` time to
provision the container's role. Nothing else reads it.

## 3. How to set it

Self-minted (`openssl rand -hex 32`) when the stack is next used. One
hard rule: it must equal the password embedded in `DATABASE_URL` when
that DSN points at the compose stack (`localhost:5442`). Staying empty
while on a remote DSN is by design.

## 4. How to test it

`./scripts/env.sh check` cross-checks it against `DATABASE_URL`
automatically; then `docker compose up -d` plus any authenticated
request proves the role answers.

## 5. How it is used

`docker-compose.yml` creates `insideout_app` in the ephemeral
container with this password; the app DSN then authenticates as that
role.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 5
links here.
