# postgres-owner-password

`POSTGRES_OWNER_PASSWORD` — the password used to create `insideout_owner` in
the bundled compose PostgreSQL service. It is conditional: remote-DSN users
can leave it empty.

## 1. Where the `.key` file is

`hosts/<host>/keys/database/.secrets/postgres-owner-password.key` (manifest
`database/local-stack.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

Compose maps this value to `INSIDEOUT_OWNER_PASSWORD` for the postgres init
script. The script reads it only when the data volume is initialized; the
server uses the password embedded in `DATABASE_OWNER_URL` for migrations.

## 3. How to set it

For a bundled stack, run `insideout setup --local --postgres bundled`, or use
`./scripts/env.sh edit` to set this key and make the password in
`DATABASE_OWNER_URL` identical. Keep it empty for a remote DSN. Changing it
after the volume exists does not change the role password; rotate the role or
recreate the disposable volume deliberately.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

After `docker compose up -d postgres`, authenticate as the owner role through
the container's environment, without placing the password in process
arguments:

```bash
docker compose exec -T postgres sh -c \
  'PGPASSWORD="$INSIDEOUT_OWNER_PASSWORD" psql -h 127.0.0.1 -U insideout_owner -d insideout -Atqc "select current_user"'
```

Expected output is `insideout_owner`, followed by the migration command in
[database-owner-dsn.md](database-owner-dsn.md). If the role cannot log in or
migrations fail, repair the role password and `DATABASE_OWNER_URL` together;
never substitute the app or image superuser for the migration role.

## 5. How it is used

The init script creates the non-superuser owner role. It owns the
`insideout` schema and migration objects, while the app role remains the
runtime connection.

## 6. Installation guide

The bundled path is [Installation — Database choices](../../installation.md#database-choices).
The key-by-key setup is [SETENV Step 5](../../../SETENV.md#step-5--only-if-you-run-docker-compose),
and the variable contract is [environment.md](../../environment.md#docker-compose-only).
