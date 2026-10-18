# access-ttl

`INSIDEOUT_ACCESS_TTL` — the lifetime of an access token as a Go duration.
Blank uses `15m`.

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/access-ttl.key` (manifest
`app/runtime.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

The root `.env` is exported by `scripts/dev.sh`; `config.Load` gives the
duration to the JWT token issuer. Compose passes it to the server container.

## 3. How to set it

Run `./scripts/env.sh edit`, choose `INSIDEOUT_ACCESS_TTL`, and enter a Go
duration such as `15m` or `1h`. Clear the row to use `15m`. Invalid strings
fail startup instead of silently selecting a different value.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

With a disposable account, keep a local `login.json` outside git and send it
to the login endpoint. The command below parses the token in memory and prints
only its lifetime (never the token):

```bash
curl -sS -H 'content-type: application/json' \
  --data-binary @login.json http://127.0.0.1:8080/api/v1/auth/login \
  | python3 -c 'import base64,json,sys; t=json.load(sys.stdin)["accessToken"]; p=t.split(".")[1]; p += "=" * (-len(p)%4); c=json.loads(base64.urlsafe_b64decode(p)); print(c["exp"]-c["iat"])'
```

Expected output is the configured number of seconds (`900` for `15m`). If
startup reports an invalid duration, enter a value such as `15m`; if the
observed lifetime differs, restart the server after editing the root `.env`.

## 5. How it is used

The expiry claim forces the client to use the refresh flow after the access
token expires. Keep it shorter than `INSIDEOUT_REFRESH_TTL`.

## 6. Installation guide

The local server setup is [Installation — Your own server](../../installation.md#your-own-server).
The variable contract is [environment.md](../../environment.md#go-backend),
and the key-by-key set step is [SETENV Step 4](../../../SETENV.md#step-4--local-dev-toggles).
