# Setup: database roles on your own server, and the bundled owner DSN

Two defects in `insideout setup --local`, plus the file split that made room
for the fix.

## Own-URL path: roles are probed, and provisioned on consent

A fresh own-URL database lacks the two roles that
`docker/postgres-init/001-create-app-role.sh` creates for the bundled
database, so the first migration failed halfway with
`role "insideout_owner" does not exist`
([issue](../../issues/2026-10-07-own-url-fresh-db-bootstrap.md)). Setup now
asks PostgreSQL before it migrates:

- both roles present: migrate exactly as before;
- both missing and the connecting role can plausibly create them: at a
  terminal, a prompt that lists what will change (the owner switch and the
  `REVOKE CREATE ON SCHEMA public FROM PUBLIC` are not invisible on a shared
  database); with `--create-roles`, no prompt (the agent path, where stdin is
  the DSN). One transaction creates the roles, passwords are generated into
  `.env` and never printed, and `DATABASE_URL` / `DATABASE_OWNER_URL` are
  rewritten to the new roles;
- anything else (a piped run without the flag, a role without the privilege,
  only one of the two roles present): stop before migrating and print the SQL
  to run, with `<owner-password>` / `<app-password>` placeholders.

PostgreSQL decides privilege: the attempt runs in one transaction and
SQLSTATE `42501` rolls it back whole. The booted server now also receives
`DATABASE_OWNER_URL`.

Files: `server/cmd/insideout/cli_setup_roles.go` (new), plus the
`--create-roles` flag in `cli_setup.go`.

## Own-URL path: a database the roles cannot use stops setup before it migrates

With both roles already on the cluster (a first database was set up earlier),
setup went straight to the migration, which died on a second database
`insideout_owner` could neither own nor create in: "permission denied for
database", with no hint. It is the same defect class as the fresh-database
bootstrap. Setup now probes three more catalog facts on the connection it was
given (who owns the database, whether the `insideout` schema exists, whether
`insideout_owner` may CREATE in the database) and, with both roles present,
stops before any migration when the schema is missing and the owner cannot
create it. It prints the per-database SQL (`ALTER DATABASE ... OWNER`, `GRANT
CONNECT`, the `public` schema REVOKEs) and the shared-instance alternative
(pre-create the schema), and changes nothing. `--create-roles` consents to
creating roles, not to handing over a database, so it does not change this.

Ownership alone is deliberately not the test. The migration's only
database-level need is CREATE for the `insideout` schema, and only while the
schema is missing, so a database someone else owns still migrates once an
admin pre-created the schema or granted CREATE (the shared-instance layout in
[database-and-rls](../../architecture/database-and-rls.md)). Stopping on "owner
is not `insideout_owner`" would break those layouts; the must-not-stop
scenarios fail under that variant. Test `d582924`, fix `65b0824`.

## Bundled path: the owner DSN was never written

`insideout setup --local --postgres bundled` and `scripts/setup.sh --local
--postgres bundled` wrote only the app-role `DATABASE_URL`. The app role may
not apply migrations (SECURITY DEFINER objects must not be owned by it), so
the same run ended with

    migrate: connected as "insideout_app"; use DATABASE_OWNER_URL

No earlier walk could have caught it: the bundled path needs Docker, and the
live walk of 2026-10-07 covered only the own-URL path with a privileged DSN.
Both setups now also write `DATABASE_OWNER_URL` (`insideout_owner`, the
`POSTGRES_OWNER_PASSWORD` or its dev default, same host, port and database),
and the Go flow attaches it before migrating. A new test pins the written
value (red before the fix); the shell script was checked on a scratch env file
the same way.

## File split

`cli_setup_local.go` (about 410 lines) is now `cli_setup_local.go` (173),
`cli_setup_local_env.go` (117) and `cli_setup_local_boot.go` (132), a pure move.

## How it was verified

- **Split:** the pure-move check printed `PURE MOVE` (344 significant lines on
  both sides).
- **Roleless cluster** (a private PostgreSQL 18 cluster, no env file): the
  `roleless`-tagged suite passes: provisioning per the init script and
  migrating, a piped run stopping with the SQL, a role without the privilege
  stopping before any migration, roles-present unchanged, partial roles, and
  the probe against the catalog. Run it with
  `go test -tags roleless ./cmd/insideout/ -run Roleless`.
- **Real terminal** (pseudo-terminal, same cluster): declining the offer exits
  1 after printing the SQL; accepting creates the roles (owner: not superuser,
  BYPASSRLS; app: not superuser, not BYPASSRLS), applies the migrations and
  exits 0. Neither transcript contains the DSN or a generated password.
- **Bundled flow, with Docker stubbed out** (a no-op `docker` on `PATH`; the
  database is real PostgreSQL 18): the binary built before the fix stops at
  `connected as "insideout_app"; use DATABASE_OWNER_URL`; the binary built
  after it writes both DSNs, applies the pending migrations and exits 0.
- **Not-ready databases** (private scratch cluster, build tag `roleless`):
  a database made by a superuser, one made by another role, and connecting as
  `insideout_owner` itself all stop before migrating, name the owner, print the
  SQL and change nothing (also with `--create-roles`); the printed statements
  then really fix it. Must-not-stop guards: a second database owned by
  `insideout_owner`, and the shared-instance layouts (schema pre-created by an
  admin; CREATE granted to `insideout_owner`). Red on `d582924`: the stop
  scenarios failed (setup died mid-migration and printed no SQL); the three
  guards passed.
- `go test ./... -count=1 -v`: 566 PASS, 0 FAIL, 1 SKIP
  (`TestReadSecretTerminal`) against the scratch database.

## Operator notes

- A fresh own-URL database no longer needs hand provisioning if the connecting
  role may create roles; otherwise setup prints the SQL.
- Existing bundled checkouts that already ran setup keep working only if their
  `.env` carries `DATABASE_OWNER_URL`; re-run `insideout setup --local
  --postgres bundled` (it rewrites `DATABASE_URL` and now adds the owner DSN).

## Limits

Managed services whose admin role has CREATEROLE and BYPASSRLS but cannot
`SET ROLE` stop with the SQL instead of provisioning (`ALTER DATABASE ...
OWNER` needs `SET ROLE`). The remaining own-URL cases are tracked in the
[issue record](../../issues/2026-10-07-own-url-fresh-db-bootstrap.md).
