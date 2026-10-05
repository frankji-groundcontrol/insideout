# postgres-port

`POSTGRES_PORT` — host port publishing the local Postgres container
(docker-compose). Config companion: blank = compose default `5442`;
the file exists so the kind is always findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/compose/.secrets/postgres-port.key` (manifest
`compose/stack.yaml`).

## 2. How it is propagated

`.env` → docker-compose interpolation of the ports mapping for the
db service. Container-internal 5432 stays fixed.

## 3. How to set it

A free host port (`5442`, `15442`). Blank = `5442` via the compose
`:-` fallback. If you change it, the local
[database-app-dsn](../database/database-app-dsn.md) host-port must match.

## 4. How to test it

`scripts/tests/keys/compose/test_postgres-port.py` (exists + integer
shape when non-empty). Live: `docker compose up`, then `psql` to
`localhost:<port>`.

## 5. How it is used

Publishes the dev database to your machine (psql, migration tools);
the application itself reaches the container over the compose
network, not through this port.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
