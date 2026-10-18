# database-app-dsn

`DATABASE_URL` — the runtime PostgreSQL DSN for the `insideout_app` role.
The role is a non-superuser and is subject to the app's RLS policies.

## 1. Where the `.key` file is

`hosts/<host>/keys/database/.secrets/database-app-dsn.key` (manifest
`database/postgres.yaml`). The DSN embeds a password; keep the file mode
`0600` and never print its value.

## 2. How it is propagated

The root `.env` is exported by `scripts/dev.sh`; the Go server opens its
runtime pool from `DATABASE_URL`. Compose passes it into the server container.
`DATABASE_OWNER_URL` is separate and is used for migrations when present.

## 3. How to set it

For a new local stack, use `insideout setup --local --postgres bundled` so the
app DSN, owner DSN, and role passwords stay matched. For an existing instance,
run `insideout setup --local --postgres own` and enter the DSN at the hidden
prompt, or use `./scripts/env.sh edit` and paste it there. Preserve
`pgbouncer=true` when the provider requires transaction pooling; the DSN's
role must be `insideout_app`.

## 4. How to test it

Before the live check, run the placement and shape suite from the repository root:

```bash
python3 -m unittest discover -s scripts/tests/keys -t . -p 'test_*.py'
```

It reports the key kind and never prints secret values.

These checks use the value without displaying it:

```bash
./scripts/env.sh check --db
insideout setup --check
./scripts/dev.sh -C server go test ./internal/store/... -run TestAuthz -v
```

Expected output includes `database ... accepts connections` and the real
authorization tests pass. A plain HTTP `401` from an unauthenticated route
proves only API routing; it is not a database check. If the DB probe fails,
confirm the host, role, SSL mode, and password at the database, then update
the DSN and rerun the checks. A role or RLS failure belongs to the database
owner or migration setup, not to the API URL alone.

## 5. How it is used

Every authenticated request uses this pool. Migrations must not run as this
role; keep DDL and `SECURITY DEFINER` ownership on `insideout_owner`.

## 6. Installation guide

The two database paths are [Installation — Database choices](../../installation.md#database-choices).
The key-by-key setup is [SETENV Step 1](../../../SETENV.md#step-1--database_url-required),
and the variable contract is [environment.md](../../environment.md#database-setups).
