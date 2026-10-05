# cors-origins

`INSIDEOUT_CORS_ORIGINS` — extra allowed CORS origins (comma
separated). Config companion: blank = none; the file exists so the
kind is always findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/cors-origins.key` (manifest
`app/runtime.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → the server's CORS middleware; deployed
value in the provider's variable store.

## 3. How to set it

Comma-separated `scheme://host[:port]` origins, no trailing slashes.
Blank adds no extra origins; the dev token flow's localhost origin is
allowed independently.

## 4. How to test it

`scripts/tests/keys/app/test_cors-origins.py` (exists + CSV shape
when non-empty). Live: a browser fetch from a listed origin succeeds.

## 5. How it is used

Each entry becomes an allowed origin for credentialed cross-origin
requests to the API.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
