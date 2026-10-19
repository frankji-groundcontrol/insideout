# Wave 3, users read grant — review-4 remediation

Branch `rv4/w3-users-read-grant`, base `53a0956` (the tip of `w3-auth-network`,
the cluster this one is ordered after), two commits `ea1d675` (red) and
`fdd0a56` (fix), then this fragment. Findings are cited as `rv4-<ID>` because
review IDs restart per review. This is the read half of rv4-DATA-03, carried as
R2-02 in the [open-findings record](../../issues/2026-10-18-review-4-open-findings.md):
wave 2 ([w2-rls.md](w2-rls.md)) closed the write half and recorded this as its
follow-up. The one migration slot the cluster was given, `20261018200000`, is
used and nothing else touches `server/db/migrations/`. No address, host,
secret, token or database URL is spelled here; every account in the runs below
was a throwaway on a private database.

## Dispositions

| ID | Status | Red | Fix | Note |
| --- | --- | --- | --- | --- |
| rv4-DATA-03, read half (R2-02) | `FIXED` | `ea1d675` | `fdd0a56` | no table read can return `password_hash`; login reads it through one definer function. The ceilings below say what that does and does not stop. |

With this row rv4-DATA-03 is closed in both halves.

## What changed

- **The app role cannot read `users.password_hash`** (red `ea1d675`, fix
  `fdd0a56`). `users_select` shows a user every co-member's row and
  `insideout_app` held table-wide `SELECT` on `insideout.users`, so a
  co-member's stored credential was one statement away for anything that ran
  SQL as the app role: a `SELECT *`, a join, a `RETURNING`, a dump taken through
  the pool. No API response carries it, so this was defence in depth, the same
  standing the write half had. The migration `20261018200000_users_read_grants.sql`
  runs `REVOKE SELECT ON insideout.users FROM insideout_app` and then grants
  `SELECT` on the twelve other columns by name (`id`, `email`, `username`,
  `avatar_url`, `bio`, `keywords`, `role`, `system_role`, `email_verified_at`,
  `meta`, `created_at`, `updated_at`). Any statement that reads the column is
  refused with SQLSTATE 42501, for every actor, the row's own owner included.
- **Login reads the credential through one named door.**
  `insideout._credential_for_email(text)` returns `(id, password_hash)`. It is
  `SECURITY DEFINER`, owned by `insideout_owner`, has `search_path` pinned,
  grants `EXECUTE` to `insideout_app` only, and answers only when no actor is
  set (the pre-auth context login runs in). `Store.GetUserByEmail` selects the
  user columns from `insideout.users` joined to the function, so one statement
  returns the user and the credential. An unknown email is the same
  `ErrNotFound` after the same single indexed lookup, so login's status, body
  and work are as before; `TestLogin_UnknownEmailCostsLikeWrongPassword` passes
  (median 13.15 ms for an unknown email, 13.07 ms for a known one with a wrong
  password).
- **Everything else in `store/users.go` stopped selecting the column.**
  `userColumns` lost `password_hash`, and `scanUser` takes extra destinations
  for columns selected after it. `CreateUser`'s and `UpdateProfile`'s
  `RETURNING` and `GetUserByID` use `userColumns`, so none of them can fail on
  the revoked column or carry it; `User.PasswordHash` is filled by
  `GetUserByEmail` alone. `UpdatePasswordHash` is unchanged: `UPDATE ... SET
  password_hash = $2 WHERE id = $1` needs `UPDATE` on the column and `SELECT` on
  `id`, nothing more. No `SELECT *` or `RETURNING *` on `users` is left in the Go
  tree (the other readers, in `roadmap.go`, `prd_commits.go` and
  `memberships.go`, name `username` or `email`). `api/auth.go` needed no edit and
  keeps what `w3-auth-network` did to it.
- **The read side of RLS is walked and the grant is guarded.**
  `server/internal/store/rls_read_walk_test.go` runs twelve read-shaped
  statements as `insideout_app` in a co-member's context, each of which must be
  refused with 42501: a column read of a co-member, of one's own row, and with
  no actor; `SELECT *`; a whole-row `to_jsonb(u)`; a join through memberships;
  a scalar subquery; a `WHERE password_hash LIKE '%'` oracle; `INSERT ...
  RETURNING` the column and `*`; `UPDATE ... RETURNING` the column; and `UPDATE
  ... SET bio = password_hash`. Its controls: the non-secret columns stay
  readable to a co-member and invisible to an outsider, and the function
  returns the id and stored hash for a known email with no actor, nothing for an
  unknown email, and nothing once an actor is set.
  `server/internal/store/users_read_grant_test.go` takes every `users` column
  from `pg_attribute` and requires `has_column_privilege('insideout_app', ...,
  'SELECT')` for all but `password_hash`, so a column added later without its
  own grant fails a test, not production. It also pins the function's shape
  (below) and that only `GetUserByEmail` carries the credential.
  `server/internal/api/auth_login_credential_test.go` drives the real login
  route over real PostgreSQL.
- **One existing walk case changed.** `rls_write_walk_test.go` proved the
  `password_hash` `UPDATE` with `password_hash = password_hash || '!'`, which
  reads the column and is now refused for that reason. It assigns a fresh value
  instead, so the case still proves what it was written for: the pinning
  trigger lets this column be written.

## Design notes

**The door takes an email and answers only before there is an actor.** Login is
the one flow that verifies a password and all it holds is an email, so the
function is keyed by email and returns two columns; the rest of the user comes
from the table, under RLS, as before. Its `WHERE` also requires
`insideout.current_user_id() IS NULL`: the system context `users_select`
already trusts and login runs in. Inside a request's own transaction (which
always sets `app.user_id`) it returns no row, so a handler that passed a
co-member's email by mistake still gets nothing. The function needs
`insideout_owner` to read through `FORCE ROW LEVEL SECURITY`, the contract the
setup role probes (`cli_setup_roles.go`) already enforce.

**Definer hygiene.** `search_path = insideout, pg_temp` (`pg_temp` last, the
documented order, so nothing a caller creates in a temporary schema can shadow a
name; `pg_catalog` is searched first regardless, every name inside is
schema-qualified, and the app role holds no `CREATE` on the schema, per
`20260819190000`). `PUBLIC` holds `EXECUTE` on every new function until it is
revoked, and that migration's default privileges also hand `EXECUTE` on new
functions to `insideout_app`, so the file runs `REVOKE ALL ... FROM PUBLIC`
before it grants `insideout_app` by name. The guard test reads the function's
ACL with `aclexplode` and fails on any grantee but `insideout_app` and the
owner.

**`REVOKE` first, columns by name, `UPDATE` untouched.** A table-level privilege
masks column grants, and a table-level `REVOKE` also clears column grants made
earlier, so re-running the file ends in the same state. `UPDATE` stays
table-wide because `users_pin_columns` already fixes what it can change, and a
column-level `UPDATE` grant would fail before RLS answers (the wave-2 reason).
`DELETE` and the three-column `INSERT` grant are not touched.

**`GetUserByEmail` stays credential-bearing; login did not get a reader of its
own.** A second method for login, leaving `GetUserByEmail` free of the
credential, would narrow the door to one route. It was not done because
`cmd/insideout/promote_admin_test.go` (lines 73 and 148) verifies the stored
password through `u.PasswordHash` from `GetUserByEmail`, that file is outside
this cluster, and `./cmd/...` is a required gate. The cost is that the other
by-email lookups (magic-link request, forgot-password, login-code request and
verify, `seed`, `promote-admin`) go through the function too and receive a hash
they ignore. They run pre-auth and already loaded it before this change; the
authenticated paths (`GetUserByID`, `UpdateProfile`, `CreateUser`) no longer
do.

**The guard reads `pg_attribute`, not `information_schema.columns`.** That view
lists only columns the current role has a privilege on, which is exactly the
ungranted column the test exists to catch.

**The migration asserts its own result.** The last statement is a `DO` block
that raises if `insideout_app` can still `SELECT` the column, so a route the
`REVOKE` cannot reach (a role membership that inherits `SELECT`, a grant to
`PUBLIC`) fails the deploy instead of shipping the finding as closed.

## Evidence

All runs are against a real PostgreSQL 18 cluster on a port of its own,
migrated from this worktree, with DSNs that exist only for the run. A skipped
database test counts as red; none skipped in the scoped runs.

| Finding | Red | Green |
| --- | --- | --- |
| rv4-DATA-03, read half | at `ea1d675` (base schema): `TestRLSReadWalk_UsersCredential` fails all twelve cases, each `rls_read_walk_test.go:64: <case>: want a refusal (SQLSTATE 42501), got rows=1 err=<nil>`; `TestRLSReadWalk_CredentialHelper` fails with `function insideout._credential_for_email(unknown) does not exist (SQLSTATE 42883)`; `TestUsersReadGrant_Columns`: `insideout_app can SELECT users.password_hash`; `TestUsersReadGrant_CredentialFunction`: `insideout._credential_for_email(text) does not exist`; `TestUsersReadGrant_OnlyLoginCarriesCredential`: `CreateUser`, `GetUserByID` and `UpdateProfile` each `returned the stored credential`; `TestLoginCredential_RouteNeedsNoTableRead`: `SELECT password_hash as the pool role: want permission denied (42501), got <nil>`. `TestLoginCredential_LegacyBcryptUpgradesInPlace` passes on the base, by design: it is the control that the bcrypt-to-argon2id upgrade survives the revoke | at `fdd0a56` all of them pass, with `TestRLSWalk`, the three `TestRLSWriteWalk_*`, `TestLogin_UnknownEmailCostsLikeWrongPassword`, the round-trip tests for magic link, password reset, invitation and login code, and the invitation and workspace-join lifecycles |

Outside the Go suite, on the same cluster:

- **psql as `insideout_app`** (one transaction, rolled back). A co-member reads
  another member's non-secret columns (a row comes back). `SELECT
  password_hash`, `SELECT *` and a whole-row `to_jsonb(u)` each fail with
  `permission denied for table users`. The function, called inside an
  actor-bearing transaction, returns no row; called with the actor cleared, it
  returns the id and the stored hash for a known email (compared against the
  value written, equal) and no row for an unknown one.
- **Mutation checks** (each reverted, the catalog snapshot identical after).
  Re-granting table-wide `SELECT` fails `TestRLSReadWalk_UsersCredential`,
  `TestUsersReadGrant_Columns` and `TestLoginCredential_RouteNeedsNoTableRead`.
  Granting `EXECUTE` on the function to `PUBLIC` fails
  `TestUsersReadGrant_CredentialFunction` ("EXECUTE is also granted to
  PUBLIC"). Flipping the function's condition to `IS NOT NULL` fails
  `TestRLSReadWalk_CredentialHelper`,
  `TestUsersReadGrant_OnlyLoginCarriesCredential` and both login tests.
- **Real binary, real server.** The built binary served as `insideout_app` on a
  loopback port of its own (started and stopped by this worker; its log ends in
  `shutting down` and nothing listens on the port). `promote-admin` (owner pool,
  password on stdin) created a throwaway account. Login with the right password
  was 200 with a session; a wrong password and an unknown email were both 401
  with the same body, `{"error":"invalid email or password"}`. psql as
  `insideout_app` against the same database was refused on `SELECT
  password_hash`. `insideout setup --check`, given that server's API base, ended
  api ok (reachable, authentication required), db ok, github and llm skipped (no
  keys), exit 0.
- **Migration, twice, and from zero.** The runner applied
  `20261018200000_users_read_grants.sql` once and the second `migrate` reported
  nothing to do. The file was also re-applied by hand twice as `insideout_owner`;
  a snapshot of the `users` table ACL, every column ACL, and the function's body
  hash, ACL and config was identical across the three states (`password_hash`
  carries `INSERT` only, `email` and `username` `INSERT` and `SELECT`, every
  other column `SELECT`; the function ACL is the owner and `insideout_app`, its
  config `search_path=insideout, pg_temp`). A fresh scratch database migrated
  from zero (all 42 files) and the store, api and cmd suites passed there; it was
  dropped afterwards.
- **Backups are unaffected.** `scripts/backup-db.sh` dumps with
  `DATABASE_OWNER_URL`, which keeps `SELECT` on every column, so a dump still
  carries the credential it is meant to restore.

## Known ceilings, recorded on purpose

- **The door is soft.** The condition is a convention the app role can satisfy:
  it can `set_config('app.user_id', '', true)` and call the function with any
  email it can read (a co-member's `email` is one of the readable columns). So
  against an attacker who already runs arbitrary SQL as the app role, this does
  not hide the hash. What it does remove: every ordinary way for the hash to
  escape, a table read, `SELECT *`, a join, a `RETURNING`, an ORM-style row
  mapping, a dump taken through the pool. Exactly one named path to it is left
  to audit, `GetUserByEmail`. A hard boundary needs a second role that alone
  holds `EXECUTE` and a second pool for login; that is a role and a deploy
  change, not made for a defence-in-depth finding.
- **Sibling credential columns are unchanged.** `sessions.token_hash`,
  `auth_tokens.token_hash` and `invitations.code_hash` stay readable by the app
  role: those tables are backend-only auth plumbing without RLS by design, and
  hold SHA-256 of high-entropy single-use or rotating opaque values, not a slow
  hash of a human-chosen password. Not raised as findings.
- **The grant can be undone by a later migration.** A migration that grants
  `SELECT` on `insideout.users` (or on all tables of the schema) to
  `insideout_app` re-opens the column. This file's `DO` assertion guards only its
  own run; `TestUsersReadGrant_Columns` is what catches a later one, in CI.
- **A new `users` column needs a grant.** Until a migration grants `SELECT` on
  it, the same test is red, and a production reader of it would fail with
  permission denied.
- **Deploy-time effect: the previous image does not survive this migration.**
  The previous image selects `password_hash` in every user read, so once the
  migration has run, its `GetUserByID`, `UpdateProfile` and `CreateUser` fail
  with permission denied. Migrations apply at startup of the new image, so a
  rolling deploy has a window in which an old instance still serves; with one
  instance there is none. Rolling back the image alone leaves it failing. To
  restore the previous image, as the owner:
  `GRANT SELECT (password_hash) ON insideout.users TO insideout_app;`, which
  reopens the finding until this image is redeployed.
- **The hosted role was not inspected from here.** The `DO` assertion fails the
  migration loudly if the hosted `insideout_app` inherits `SELECT` on the column
  from another role or from `PUBLIC`; check that before the deploy rather than
  at it.

## Left for the integrator

Documentation, comment and index edits only; no code is waiting.

INTEGRATOR TODO: `docs/architecture/database-and-rls.md` (users column grants
and the definer function): after the `users` columns allow-list (lines 149 to
171), add that `insideout_app` reads every `users` column except `password_hash`
(`REVOKE`, then `GRANT SELECT (...)` in `20261018200000_users_read_grants.sql`),
that `insideout._credential_for_email(text)` is the one path to the credential
(definer, `search_path = insideout, pg_temp`, `EXECUTE` for `insideout_app`
only, a row only while `app.user_id` is unset), that a new `users` column needs
its own `GRANT SELECT` and that `users_read_grant_test.go` fails until it has
one, and the soft-door ceiling above. In the system-context exceptions paragraph
(lines 73 to 81), say that `GetUserByEmail` reaches `users` through that
function; the sentence that it runs without `app.user_id` stays true and is now
load-bearing.

INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md`: the
rv4-DATA-03 row (line 159) `PARTIAL` to `FIXED` with red `ea1d675` and fix
`fdd0a56` (use the cherry-picked SHAs), its note replaced by "read half closed
by a column-level `SELECT` allow-list and a definer function; the function's
soft door is a recorded ceiling"; R2-02 (line 175) closed with a link to this
fragment, in the style of the other closed wave-2 residue. `docs/issues/README.md`
likewise.

INTEGRATOR TODO: indexes: the `w3-users-read-grant` row in this folder's
`index.md` (the cluster-fragments table, and the wave-3 row that still reads
"not started"); `docs/changelogs/README.md` and `docs/changelogs/index/2026-10.md`;
the plan board `docs/plans/README.md` and `docs/plans/2026-10-18-multirole-4.md`
with its `.track.yaml`, for the cluster's state.

INTEGRATOR TODO (optional): `docs/usage/deployment.md`, a deploy note that this
release revokes `SELECT` on `insideout.users.password_hash` from the app role
and that the previous image cannot read users after it (the rollback statement is
under the ceilings above). `docs/usage/backup-restore.md` (line 46) needs no
edit: the dump runs as the owner.

## Verification

Run against a private PostgreSQL 18 cluster on a port of its own (roles
`insideout_owner` and `insideout_app`, the same two the project uses), with the
one live server on a loopback port of its own.

- `go build ./...` and `go vet ./...` clean; `gofmt -l internal cmd` prints
  nothing.
- `go test ./internal/store/... ./internal/api/... ./cmd/... -count=1 -v`: 354
  top-level tests passed, 0 failed, 1 skipped (`TestReadSecretTerminal`, which
  needs a terminal, is not database-gated and is skipped on the base too).
- `go test ./internal/store/... ./internal/api/... -race -count=1`: both ok.
- The whole server suite once at the end, `go test ./... -count=1`: 27 packages
  ok, 3 without test files, 0 failed.
- The setup role probes (`TestProbeRolesReadsTheCatalog`, `TestDecideRoles` and
  the rest of `cli_setup_roles*_test.go`) pass in the `./cmd/...` run.
- Every `scripts/tests/check_*.py` (9) exits 0 and prints only its own `ok`
  line; `sh scripts/check-redaction.sh` prints nothing; `sh
  scripts/check-docs-recorded.sh` ok; no Go file this branch touches is over 350
  lines (the largest is `rls_write_walk_test.go` at 265).
