# cookie-secure

`INSIDEOUT_COOKIE_SECURE` — whether auth cookies carry the `Secure` flag.
Blank (or any value other than `0`) keeps the secure default.

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/cookie-secure.key` (manifest
`app/runtime.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

The root `.env` is exported by `scripts/dev.sh`; `config.Load` turns the value
into the cookie option used by login, refresh, and reset responses. Compose
passes the same variable to the server container.

## 3. How to set it

Run `./scripts/env.sh edit`, choose `INSIDEOUT_COOKIE_SECURE`, and enter
exactly `0` only for a plain-HTTP local browser. Clear the row (recommended
for HTTPS) to restore secure cookies.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

Use a disposable account and sign in through the local browser, then inspect
the login response in the browser Network panel. The `Set-Cookie` header must
include `Secure` when the value is blank and must omit it when the value is
exactly `0`. A header-only command is safe once you have a local request body
file (kept outside git):

```bash
curl -sS -D - -o /dev/null -H 'content-type: application/json' \
  --data-binary @login.json http://127.0.0.1:8080/api/v1/auth/login \
  | sed -n 's/^Set-Cookie: insideout_[^=]*=[^;]*;\(.*\)$/\1/p'
```

The filtered output contains `Secure` when the flag is on and omits it when
the value is exactly `0`; the token value is never printed. If an HTTP browser
does not stay signed in while the header contains `Secure`, set the value to
`0` for that local run. If an HTTPS deployment omits `Secure`, clear the value
and restart it; do not leave insecure cookies on a shared or hosted service.

## 5. How it is used

The auth handlers apply the flag to access, refresh, magic-link, and reset
cookies. It is independent of bearer tokens used by the CLI and MCP server.

## 6. Installation guide

The local browser setup is [Installation — Your own server](../../installation.md#your-own-server).
The variable contract is [environment.md](../../environment.md#go-backend),
and the key-by-key set step is [SETENV Step 4](../../../SETENV.md#step-4--local-dev-toggles).
