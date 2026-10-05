# server-port

`SERVER_PORT` — host port publishing the server container
(docker-compose). Config companion: blank = compose default `8080`;
the file exists so the kind is always findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/compose/.secrets/server-port.key` (manifest
`compose/stack.yaml`).

## 2. How it is propagated

`.env` → docker-compose interpolation of the ports mapping for the
server service. Server-internal port stays fixed.

## 3. How to set it

A free host port (`8080`, `18080`). Blank = `8080` via the compose
`:-` fallback; a busy port fails `docker compose up` with a bind
error.

## 4. How to test it

`scripts/tests/keys/compose/test_server-port.py` (exists + integer
shape when non-empty). Live: `docker compose up`, then curl the
published port's health endpoint.

## 5. How it is used

Only the host-side half of the compose ports mapping — it changes
where the stack is reachable from your machine, not where the server
listens inside the container (that is [addr](../app/addr.md)).

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
