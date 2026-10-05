# client-port

`CLIENT_PORT` — host port publishing the web client container
(docker-compose). Config companion: blank = compose default `5173`;
the file exists so the kind is always findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/compose/.secrets/client-port.key` (manifest
`compose/stack.yaml`).

## 2. How it is propagated

`.env` → docker-compose interpolation of the ports mapping for the
client service. Container-internal dev-server port stays fixed.

## 3. How to set it

A free host port (`5173`, `15173`). Blank = `5173` via the compose
`:-` fallback; a busy port fails `docker compose up` with a bind
error.

## 4. How to test it

`scripts/tests/keys/compose/test_client-port.py` (exists + integer
shape when non-empty). Live: `docker compose up`, then open
`http://localhost:<port>`.

## 5. How it is used

Host-side half of the client ports mapping — where the web client is
reachable from your machine.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
