# refresh-ttl

`INSIDEOUT_REFRESH_TTL` — refresh-token lifetime (Go duration).
Config companion: blank = default `720h`; the file exists so the kind
is always findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/refresh-ttl.key` (manifest
`app/runtime.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → token minting in the auth layer; deployed
value in the provider's variable store.

## 3. How to set it

A Go duration string (`720h`, `168h`). Blank = `720h` (30 days);
garbage fails startup with a parse error (fail-fast).

## 4. How to test it

`scripts/tests/keys/app/test_refresh-ttl.py` (exists + duration shape
when non-empty). Live: decoded refresh-token `exp` differs by the set
amount.

## 5. How it is used

Expiry claim on refresh tokens; it bounds how long a login survives
before re-auth is forced. Keep it ≥ access-ttl.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
