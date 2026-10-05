# addr

`INSIDEOUT_ADDR` — the server's listen address. Config companion:
blank = catalog default `:8080` in effect; the file exists so the kind
is always findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/addr.key` (manifest
`app/runtime.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → the server's listen call; deployed value
in the provider's variable store.

## 3. How to set it

Host:port (`:8080`, `127.0.0.1:9000`). Blank falls back to `:8080`,
then to the platform `PORT` on Railway/Heroku-style deploys
(`config.go` `listenAddr`).

## 4. How to test it

`scripts/tests/keys/app/test_addr.py` (exists + host:port shape when
non-empty). Live: `curl` the health endpoint on the chosen address.

## 5. How it is used

Passed to the HTTP listener at boot; a bad value fails startup with a
bind error.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
