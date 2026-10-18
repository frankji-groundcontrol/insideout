# dev-cors

`INSIDEOUT_DEV_CORS` — the permissive development CORS switch. Blank (or any
value other than `1`) leaves it off.

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/dev-cors.key` (manifest
`app/runtime.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

The root `.env` is exported by `scripts/dev.sh`; `config.Load` enables the
reflecting CORS middleware only when the value is exactly `1`. Compose passes
the variable to the server container.

## 3. How to set it

Run `./scripts/env.sh edit`, choose `INSIDEOUT_DEV_CORS`, and enter exactly
`1` for a short-lived localhost experiment. Clear it when the experiment is
finished. A non-local `INSIDEOUT_PUBLIC_URL` rejects this setting at startup;
use the explicit origin list instead for a LAN client.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

With a loopback `INSIDEOUT_PUBLIC_URL` and the server running, send a preflight
from an otherwise unlisted origin:

```bash
curl -sS -D - -o /dev/null -X OPTIONS \
  -H 'Origin: http://example.invalid' \
  -H 'Access-Control-Request-Method: GET' \
  http://127.0.0.1:8080/api/v1/me \
  | grep -E 'Access-Control-Allow-(Origin|Credentials):'
```

Expected output reflects `http://example.invalid` and includes
`Access-Control-Allow-Credentials: true`. A startup error naming
`INSIDEOUT_DEV_CORS` means `INSIDEOUT_PUBLIC_URL` is not loopback; clear this
key and set the allowed browser origin in `INSIDEOUT_CORS_ORIGINS` instead.

## 5. How it is used

The middleware reflects arbitrary request origins with credentials. That is
useful for local debugging and unsafe on a hosted deployment; it does not
replace authentication.

## 6. Installation guide

The local browser setup is [Installation — Your own server](../../installation.md#your-own-server).
The variable contract is [environment.md](../../environment.md#go-backend),
and the key-by-key set step is [SETENV Step 4](../../../SETENV.md#step-4--local-dev-toggles).
