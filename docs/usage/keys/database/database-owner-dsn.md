# database-owner-dsn

`DATABASE_OWNER_URL` — the migration and backup DSN for the
`insideout_owner` role. It is a non-superuser with ownership of the
`insideout` schema and its `SECURITY DEFINER` objects.

## 1. Where the `.key` file is

`hosts/<host>/keys/database/.secrets/database-owner-dsn.key` (manifest
`database/postgres.yaml`). The DSN embeds a password; keep the file mode
`0600` and never print its value.

## 2. How it is propagated

The root `.env` is exported by `scripts/dev.sh`; `insideout migrate` and
`scripts/backup-db.sh` use this DSN. When it is empty, migrations fall back to
`DATABASE_URL`, which must then already connect as `insideout_owner`.

## 3. How to set it

For a new bundled stack, `insideout setup --local --postgres bundled` writes
the owner DSN from `POSTGRES_OWNER_PASSWORD`. For an existing instance, run
`insideout setup --local --postgres own` and provide the DSN at the hidden
prompt, or select `DATABASE_OWNER_URL` in `./scripts/env.sh edit`. The role
must be `insideout_owner`; do not paste a superuser DSN into this key.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

Run migrations through the normal environment bridge:

```bash
./scripts/env.sh check server
./scripts/dev.sh -C server go run ./cmd/insideout migrate
```

Expected output is either `migrations applied (...)` or
`nothing to do, already up to date`. If it says the connection is using
`insideout_app`, set `DATABASE_OWNER_URL` to the owner DSN and retry. A
permission or authentication error means the owner role/password is wrong;
repair it at the database and update both consumers without printing the DSN.

## 5. How it is used

Only migrations and backups use this connection. Keeping it separate prevents
the runtime role from acquiring DDL rights and preserves the schema/RLS
boundary.

## 6. Installation guide

The two database paths are [Installation — Database choices](../../installation.md#database-choices).
The key-by-key setup is [SETENV Step 1](../../../SETENV.md#step-1--database_url-required),
and the variable contract is [environment.md](../../environment.md#database-setups).
