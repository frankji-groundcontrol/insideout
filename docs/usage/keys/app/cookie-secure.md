# cookie-secure

`INSIDEOUT_COOKIE_SECURE` — the `Secure` flag on session cookies.
Config companion: blank = default **on**; the file exists so the kind
is always findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/cookie-secure.key` (manifest
`app/runtime.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → cookie construction in the session layer;
deployed value in the provider's variable store.

## 3. How to set it

Leave blank for secure cookies. Exactly `0` turns the flag off — the
plain-http local-dev escape hatch; any other value keeps it on.

## 4. How to test it

`scripts/tests/keys/app/test_cookie-secure.py` (exists + only `0` or
blank). Live: `curl -i` login and inspect `Set-Cookie` for `Secure`.

## 5. How it is used

Browsers refuse to store non-`Secure` cookies over https and store
`Secure` ones never over http — mismatches show up as login loops.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
