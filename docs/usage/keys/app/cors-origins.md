# cors-origins

`INSIDEOUT_CORS_ORIGINS` — a comma-separated allow-list for browser Origins
that call the API from a different origin. Blank means no extra origins.

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/cors-origins.key` (manifest
`app/runtime.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

The root `.env` is exported by `scripts/dev.sh`; `config.Load` splits the
list, and the API CORS middleware compares each request's `Origin`. Compose
passes the value to the server container.

## 3. How to set it

Run `./scripts/env.sh edit`, choose `INSIDEOUT_CORS_ORIGINS`, and enter origins
without trailing slashes, for example `http://localhost:5173,http://127.0.0.1:5173`.
Use the `localhost` token when any localhost or `127.0.0.1` port should match.
Clear the row when the client is same-origin through the compose nginx proxy.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

After the server is running, send a preflight from an allowed origin:

```bash
curl -sS -D - -o /dev/null -X OPTIONS \
  -H 'Origin: http://localhost:5173' \
  -H 'Access-Control-Request-Method: GET' \
  http://127.0.0.1:8080/api/v1/me \
  | grep -E 'Access-Control-Allow-(Origin|Credentials|Methods):'
```

Expected output includes the requested origin and
`Access-Control-Allow-Credentials: true`. Repeat with an origin that is not
listed; it must not receive `Access-Control-Allow-Origin`. If the header is
missing, add the exact scheme, host, and port, remove a trailing slash, and
restart the server.

## 5. How it is used

The allow-list is for credentialed browser requests. Native clients and the
hosted same-origin web client do not need it; setting it does not grant API
authentication by itself.

## 6. Installation guide

The local browser setup is [Installation — Your own server](../../installation.md#your-own-server).
The variable contract is [environment.md](../../environment.md#go-backend),
and the key-by-key set step is [SETENV Step 4](../../../SETENV.md#step-4--local-dev-toggles).
