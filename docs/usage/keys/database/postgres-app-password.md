# postgres-app-password

`POSTGRES_APP_PASSWORD` — the password used to create `insideout_app` in the
bundled compose PostgreSQL service. It is conditional: remote-DSN users can
leave it empty.

## 1. Where the `.key` file is

`hosts/<host>/keys/database/.secrets/postgres-app-password.key` (manifest
`database/local-stack.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

Compose maps this value to `INSIDEOUT_APP_PASSWORD` for the postgres init
script. The script reads it only when the data volume is initialized; the Go
server uses the password embedded in `DATABASE_URL` instead.

## 3. How to set it

For a bundled stack, run `insideout setup --local --postgres bundled`, or use
`./scripts/env.sh edit` to set this key and make the password in
`DATABASE_URL` identical. Keep `POSTGRES_APP_PASSWORD` empty when
`DATABASE_URL` points to a remote instance. Changing it after the volume
exists does not change the role password; rotate the role or recreate the
disposable volume deliberately.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

After `docker compose up -d postgres`, authenticate as the app role without
placing its password in the process arguments:

```bash
docker compose exec -T postgres sh -c \
  'PGPASSWORD="$INSIDEOUT_APP_PASSWORD" psql -h 127.0.0.1 -U insideout_app -d insideout -Atqc "select current_user"'
```

Expected output is `insideout_app`, followed by
`./scripts/env.sh check --db` and the `TestAuthz` command in
[database-app-dsn.md](database-app-dsn.md). If authentication fails, compare
the value used to initialize the volume with the password in the app DSN;
update both together or recreate the disposable volume.

## 5. How it is used

The init script creates the non-superuser runtime role with this password. The
role owns runtime data access only; migrations use `insideout_owner`.

## 6. Installation guide

The bundled path is [Installation — Database choices](../../installation.md#database-choices).
The key-by-key setup is [SETENV Step 5](../../../SETENV.md#step-5--only-if-you-run-docker-compose),
and the variable contract is [environment.md](../../environment.md#docker-compose-only).
