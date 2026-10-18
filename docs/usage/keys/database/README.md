# Keys — database module

Per-key reference for the `database/` module of the key store. Each
file documents one key kind with the six parts (where the `.key` file
is, propagation, setting, testing, use, and the installation-guide
pointer). The module table, shared flow, and per-host state live in
the [keys README](../README.md).

Choose one database lane: `insideout setup --local --postgres bundled` keeps
the local roles and DSNs matched, while `--postgres own` accepts an existing
DSN at a hidden prompt. Verify runtime connectivity with `setup --check` and
role behavior with migrations and the real authorization test. The installation
sequence starts at [SETENV Step 1](../../../SETENV.md#step-1--database_url-required).

- [database-app-dsn.md](database-app-dsn.md) — `DATABASE_URL`
  (manifest `database/postgres.yaml`)
- [database-owner-dsn.md](database-owner-dsn.md) — `DATABASE_OWNER_URL`
  (`database/postgres.yaml`)
- [postgres-app-password.md](postgres-app-password.md) —
  `POSTGRES_APP_PASSWORD` (`database/local-stack.yaml`)
- [postgres-owner-password.md](postgres-owner-password.md) —
  `POSTGRES_OWNER_PASSWORD` (`database/local-stack.yaml`)
