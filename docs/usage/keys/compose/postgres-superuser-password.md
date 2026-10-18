# postgres-superuser-password

`POSTGRES_SUPERUSER_PASSWORD` — the bootstrap password for the `postgres`
role in the bundled compose database. Blank uses the local development
default. The API never uses this credential.

## 1. Where the `.key` file is

`hosts/<host>/keys/compose/.secrets/postgres-superuser-password.key`
(manifest `compose/stack.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

Compose maps this value to the postgres image's `POSTGRES_PASSWORD` only when
the container initializes a fresh data volume. The value is not consulted by
the Go server.

## 3. How to set it

Run `./scripts/env.sh edit`, choose `POSTGRES_SUPERUSER_PASSWORD`, and enter a
locally generated password for a disposable volume. Clear it only when the
documented dev default is acceptable. Changing it after the volume exists does
not rotate the database role; rotate the role explicitly or recreate the
disposable volume.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

After starting a fresh bundled database, test the role without placing the
password in shell arguments:

```bash
docker compose exec -T postgres sh -c \
  'PGPASSWORD="$POSTGRES_PASSWORD" psql -h 127.0.0.1 -U postgres -d insideout -Atqc "select 1"'
```

Expected output is `1`. If authentication fails after editing the key, the
existing volume still has its original password; restore that password or
recreate the disposable volume before retrying. Keep this role away from the
application DSN.

## 5. How it is used

The image superuser runs the init scripts that create `insideout_owner` and
`insideout_app`. It should be used only for local bootstrap and maintenance.

## 6. Installation guide

The compose path is [Installation — Database choices](../../installation.md#database-choices).
The key-by-key setup is [SETENV Step 5](../../../SETENV.md#step-5--only-if-you-run-docker-compose),
and the variable contract is [environment.md](../../environment.md#docker-compose-only).
