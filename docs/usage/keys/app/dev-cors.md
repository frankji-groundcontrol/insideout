# dev-cors

`INSIDEOUT_DEV_CORS` — permissive dev CORS switch. Config companion:
blank = default **off**; the file exists so the kind is always
findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/dev-cors.key` (manifest
`app/runtime.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → the server's CORS middleware; deployed
value in the provider's variable store.

## 3. How to set it

Exactly `1` turns permissive CORS on (allow all origins — local
experiments only). Blank or any other value keeps it off.

## 4. How to test it

`scripts/tests/keys/app/test_dev-cors.py` (exists + only `1` or
blank). Live with `1`: any origin gets `Access-Control-Allow-Origin:
*`-style open behavior.

## 5. How it is used

Read once at boot (`DevPermissiveCORS`); never enable on a deployed
instance — it defeats the origin allowlist. The server refuses to start
with `1` unless `INSIDEOUT_PUBLIC_URL` is localhost, 127.0.0.1 or ::1;
for a LAN address list the browser origin in `INSIDEOUT_CORS_ORIGINS`
instead.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
