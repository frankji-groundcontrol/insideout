# server-port

`SERVER_PORT` — the host port published for the API container. Blank uses
compose's `8080` default.

## 1. Where the `.key` file is

`hosts/<host>/keys/compose/.secrets/server-port.key` (manifest
`compose/stack.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

Docker Compose substitutes the value in the server ports mapping. The API
still listens on `:8080` inside the container; `INSIDEOUT_ADDR` controls a
bare process instead.

## 3. How to set it

Run `./scripts/env.sh edit`, choose `SERVER_PORT`, and enter a free host TCP
port such as `8080` or `18080`. Clear it to restore `8080`. If a local client
calls the API directly, update its `API_BASE` to the selected port.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

Start the API container and probe the published host port:

```bash
./scripts/env.sh check
docker compose up -d server
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/healthz
```

Expected output is `200` (substitute the chosen port). A bind error means the
port is busy; choose a free port and restart the server service. A `503` from
compose usually means the database service is not healthy yet.

## 5. How it is used

The host mapping makes the API reachable from local tools and a direct Flutter
web run. It does not change the port used by the server-to-database network.

## 6. Installation guide

The compose path is [Installation — Your own server](../../installation.md#your-own-server).
The key-by-key setup is [SETENV Step 5](../../../SETENV.md#step-5--only-if-you-run-docker-compose),
and the variable contract is [environment.md](../../environment.md#docker-compose-only).
