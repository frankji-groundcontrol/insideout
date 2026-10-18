# addr

`INSIDEOUT_ADDR` — the server listen address. Leave it empty to use the
default `:8080`; set it only when the process must listen on another local
address or port.

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/addr.key` (manifest
`app/runtime.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

The value moves from the root `.env` through `scripts/dev.sh` into
`config.Load`. Compose publishes the server container separately with
`SERVER_PORT`; `INSIDEOUT_ADDR` controls the address inside a bare server
process (and is fixed to `:8080` inside compose).

## 3. How to set it

From the repository root run `./scripts/env.sh edit`, choose
`INSIDEOUT_ADDR`, and enter a value such as `:9000` or `127.0.0.1:9000`.
Clear the row to restore the `:8080` default. The fallback platform `PORT` is
used only when this value is empty.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

Run `./scripts/env.sh check server`, start the server in one terminal, and
probe the selected port from another:

```bash
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:9000/healthz
```

Expected output is `200`. A bind error means the address is already in use or
is not available on this machine; choose a free address, clear the value, and
run the check again.

## 5. How it is used

`config.Load` passes the value to the HTTP listener. An invalid address or a
busy port stops the server at bind time; it does not change the compose host
mapping.

## 6. Installation guide

The end-to-end local path is [Installation — Your own server](../../installation.md#your-own-server).
The variable contract is [environment.md](../../environment.md#go-backend),
and the key-by-key set step is [SETENV Step 4](../../../SETENV.md#step-4--local-dev-toggles).
