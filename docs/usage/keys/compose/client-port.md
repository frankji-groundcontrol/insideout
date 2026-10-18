# client-port

`CLIENT_PORT` — the host port published for the web client container. Blank
uses compose's `5173` default.

## 1. Where the `.key` file is

`hosts/<host>/keys/compose/.secrets/client-port.key` (manifest
`compose/stack.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

Docker Compose reads the root `.env` and substitutes this value in the client
ports mapping. Nginx still listens on `8080` inside the container; this key
changes only the host side.

## 3. How to set it

Run `./scripts/env.sh edit`, choose `CLIENT_PORT`, and enter a free host TCP
port such as `5173` or `15173`. Clear it to restore `5173`. If the port is
changed, set `INSIDEOUT_PUBLIC_URL` to the same browser origin when emailed
links must open on that port.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

Validate without rendering secrets and start the stack:

```bash
./scripts/env.sh check
docker compose up -d client
docker compose ps client
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:5173/healthz
```

Expected output is `200` and the client service is `healthy` (substitute the
chosen port in the URL). A bind error means another process owns the port;
choose a free one and restart the client service.

## 5. How it is used

The published port is where a browser reaches the Flutter web client. It does
not alter the API's internal `:8080` listener or the database port.

## 6. Installation guide

The compose path is [Installation — Your own server](../../installation.md#your-own-server).
The key-by-key setup is [SETENV Step 5](../../../SETENV.md#step-5--only-if-you-run-docker-compose),
and the variable contract is [environment.md](../../environment.md#docker-compose-only).
