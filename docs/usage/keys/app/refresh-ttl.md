# refresh-ttl

`INSIDEOUT_REFRESH_TTL` — the lifetime of a refresh token as a Go duration.
Blank uses `720h` (30 days).

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/refresh-ttl.key` (manifest
`app/runtime.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

The root `.env` is exported by `scripts/dev.sh`; `config.Load` gives the
duration to refresh-token issuance and rotation. Compose passes it to the
server container.

## 3. How to set it

Run `./scripts/env.sh edit`, choose `INSIDEOUT_REFRESH_TTL`, and enter a Go
duration such as `720h` or `168h`. Clear the row to use 30 days. Invalid
strings fail startup; keep this duration longer than the access-token TTL.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

Use a disposable account and a local `login.json` kept outside git. Refresh
tokens are opaque, so inspect only the cookie's `Max-Age`; the filter below
never prints the token value:

```bash
curl -sS -D - -o /dev/null -H 'content-type: application/json' \
  --data-binary @login.json http://127.0.0.1:8080/api/v1/auth/login \
  | sed -n 's/^Set-Cookie: insideout_refresh=[^;]*;.*Max-Age=\([0-9]*\).*/\1/p'
```

Expected output is the configured number of seconds (`2592000` for `720h`).
If the value is shorter than the access TTL or startup rejects it, correct the
duration, restart the server, and repeat the disposable-account check.

## 5. How it is used

Refresh expiry bounds how long a session can be renewed. Expired or rotated
refresh tokens require a new login.

## 6. Installation guide

The local server setup is [Installation — Your own server](../../installation.md#your-own-server).
The variable contract is [environment.md](../../environment.md#go-backend),
and the key-by-key set step is [SETENV Step 4](../../../SETENV.md#step-4--local-dev-toggles).
