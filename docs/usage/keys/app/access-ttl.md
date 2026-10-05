# access-ttl

`INSIDEOUT_ACCESS_TTL` — session access-token lifetime (Go duration).
Config companion: blank = default `15m`; the file exists so the kind
is always findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/access-ttl.key` (manifest
`app/runtime.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → token minting in the auth layer; deployed
value in the provider's variable store.

## 3. How to set it

A Go duration string (`15m`, `30m`, `1h`). Blank = `15m`; garbage
fails startup with a parse error (fail-fast, no silent default).

## 4. How to test it

`scripts/tests/keys/app/test_access-ttl.py` (exists + duration shape
when non-empty). Live: decoded access-token `exp` differs by the set
amount.

## 5. How it is used

Expiry claim on every access token; expiry forces the client onto the
refresh flow.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
