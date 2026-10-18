# postgres-port

`POSTGRES_PORT` — the host port published for the bundled PostgreSQL
container. Blank uses compose's `5442` default.

## 1. Where the `.key` file is

`hosts/<host>/keys/compose/.secrets/postgres-port.key` (manifest
`compose/stack.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

Docker Compose substitutes the value in the postgres ports mapping. PostgreSQL
continues to listen on `5432` inside the container; the app uses the compose
service name internally and does not use this host mapping.

## 3. How to set it

Run `./scripts/env.sh edit`, choose `POSTGRES_PORT`, and enter a free host TCP
port such as `5442` or `15442`. Clear it to restore `5442`. If a bundled DSN
already exists, its host port must match this value; `insideout setup --local
--postgres bundled` rewrites both DSNs safely.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

Start only the database and probe the selected host port:

```bash
./scripts/env.sh check
docker compose up -d postgres
pg_isready -h 127.0.0.1 -p 5442 -d insideout
```

Expected output ends with `accepting connections` (substitute the chosen
port). A refusal means the container is not healthy or the host port is busy;
inspect `docker compose ps postgres`, then choose a free port and update the
bundled DSNs together.

## 5. How it is used

The host mapping serves local `psql`, setup, and migration tools. Containers
reach PostgreSQL over the compose network at `postgres:5432`.

## 6. Installation guide

The compose path is [Installation — Your own server](../../installation.md#your-own-server).
The key-by-key setup is [SETENV Step 5](../../../SETENV.md#step-5--only-if-you-run-docker-compose),
and the variable contract is [environment.md](../../environment.md#docker-compose-only).
