# Wave 2, RLS write-side hardening — review-4 remediation

Branch `rv4/w2-rls`, `4b2a1db..4af1df7` (five commits: one red commit for the
write walk that is also the red for rv4-DATA-01 and rv4-DATA-03, the two RLS
fixes, and one red/fix pair for the serve-role check), then this fragment.
Findings are cited as `rv4-<ID>` because review IDs restart per review. Row 2.6
of the [remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md).
The one migration slot this cluster was given, `20261018140000`, is used and
nothing else touches `server/db/migrations/`. No address, host, secret or
one-time code is spelled in this fragment.

## What changed

- **A membership row needs a reason** (rv4-DATA-01, red `92a0230`, fix
  `06e8b78`). `workspace_memberships_insert` checked only
  `user_id = current_user_id()`, so any signed-in session could insert itself as
  `admin` of any workspace whose id it knew (ids reach members in URLs and
  exports), and an admin reads and writes every row of the workspace. The policy
  now also calls `insideout._may_insert_membership(workspace_id, user_id, role)`:
  an `admin` row only for the workspace's creator (what `createWorkspaceTx`
  writes right after the workspace), a `member` row only while the workspace is
  active and `app.join_code` holds its code (what `JoinWorkspace` sets, and what
  `workspaces_select` already reads). A join code buys `member` and nothing
  more, and nobody can add anyone but themselves.
- **The write side of RLS is walked** (rv4-DATA-02, red `92a0230`; all three
  tests green at `571a5d8`). `server/internal/store/rls_write_walk_test.go` is
  the write twin of `TestRLSWalk`, which only counts `SELECT` visibility and so
  could not see either hole above. It runs every statement as `insideout_app`,
  as a named actor, in a transaction that is always rolled back. Three tests:
  `TestRLSWriteWalk_MembershipInsert` (eight cases: seven that must be refused
  with SQLSTATE 42501, starting with a bare self-insert as admin, and the one
  legitimate self-insert as member with the right code),
  `TestRLSWriteWalk_UserColumns` (an `UPDATE` of every `users` column, the list
  read from the catalog so a new column fails the walk until it is classified;
  an `INSERT` that sets each privileged column; and the actor-less and
  other-user `UPDATE` that must still answer "0 rows" with no error), and
  `TestRLSWriteWalk_NoRowChanges` (an `UPDATE` and a `DELETE` on every RLS table
  from the catalog as an outsider and as an actor-less session, with a member's
  run as the control that statements can change rows at all). The third test is
  green on the base schema on purpose: it pins the boundary that already holds.
- **`insideout_app` can no longer create or rewrite privileged `users`
  columns** (rv4-DATA-03, write half; red `92a0230`, fix `571a5d8`). The role
  held table-wide `INSERT` and `UPDATE` on `insideout.users` while `users_insert`
  is `WITH CHECK (true)` and `users_update` only pins the row to its owner, so one
  statement could register an admin or promote an account. `INSERT` is now cut
  down by column grants to `(email, password_hash, username)`, all that
  `CreateUser` writes. `UPDATE` is pinned by a `BEFORE UPDATE` trigger that lets
  a role RLS applies to change only `username`, `avatar_url`, `bio`, `keywords`
  and `password_hash` (and `updated_at`, which `set_updated_at` owns). The read
  half of the finding is not done; see the ceilings below.
- **The server refuses to serve on a pool role that skips RLS** (rv4-DATA-04,
  red `98c5ba8`, fix `4af1df7`). `Store.CheckServeRole`
  (`server/internal/store/serve_role.go`) reads `rolsuper` and `rolbypassrls`
  for the pool's own role from `pg_roles`, and `runServe` calls it before the
  startup migrations. On a hit the process logs one plain error, exits 1 and
  never listens. The error names the attribute (`BYPASSRLS`, `SUPERUSER` or
  both) and `DATABASE_URL`, tells the operator to point it at the
  `insideout_app` role and keep the privileged role in `DATABASE_OWNER_URL`, and
  carries neither the role name nor any part of the DSN. `migrate`, `seed` and
  `promote-admin` never reach `runServe`, so the owner role keeps working for
  them.

## Design notes (row 2.6)

**The membership check is a DEFINER helper.** The insert policy has to read
`workspaces` (creator, status, code). Doing that inline from a
`workspace_memberships` policy recurses through `workspaces_select`, which reads
memberships (the BUG-007 family). The helper is `SECURITY DEFINER`, owned by
`insideout_owner`, with `search_path = insideout, pg_catalog` pinned and its
column arguments qualified with the table name, like `_is_member` and
`_is_admin`. It stays `STABLE` and returns a single `EXISTS`.

**`INSERT` is column grants; `UPDATE` is a trigger.** Both are the finding's own
options. A column grant for `UPDATE` was tried first and lost for one reason: a
column privilege is checked before any row is read. `promote-admin` run without
`DATABASE_OWNER_URL` is an `UPDATE` of someone else's `system_role` on the app
pool; RLS answers "0 rows" and the CLI turns that into its "set
`DATABASE_OWNER_URL`" hint (`TestPromoteAdminWithoutOwnerPoolSaysSoAndRecovers`).
Measured in a rolled-back transaction on the private cluster, the same
statement under the column-grant design failed with "permission denied for table
users", and under the shipped design answers `UPDATE 0`. A `BEFORE UPDATE`
trigger only sees rows RLS already let through, so that contract holds. For
`INSERT` there is no such contract (nothing reads a row first), so the column
grant is the smaller tool, and a new column is refused by default.

- The trigger compares `to_jsonb(NEW) - editable` with `to_jsonb(OLD) -
  editable`, so the list names what may change and everything else, a column
  added later included, has to come out unchanged. Fail-closed on purpose.
  Measured: with a throwaway column added in a rolled-back transaction, an
  `UPDATE` of it as `insideout_app` raised the trigger's 42501 and an `INSERT`
  that set it was refused ("permission denied for table users").
- A role RLS does not apply to (`BYPASSRLS` or superuser) is trusted with every
  column. That is the line RLS draws, and it keeps `insideout_owner` (migrations,
  `promote-admin`, `SetSystemRole`) unchanged. The trigger looks the role up in
  `pg_roles` by `current_user`.
- The `REVOKE` comes before the `GRANT`: a table-level privilege masks column
  grants, and a table-level `REVOKE` also clears column grants made earlier, so
  re-running the file ends in the same state.

**The refusal sits before the startup migrations, and there is no override.**
Putting it first means a refused boot also applies nothing. Only the request
pool is checked: `DATABASE_OWNER_URL` (`insideout_owner`) is meant to bypass RLS.
No `INSIDEOUT_ALLOW_PRIVILEGED_SERVE` flag was added. The row asked for a
refusal and the owner-decision condition ("a documented workflow genuinely needs
a privileged serve role") does not hold: `docker-compose.yml`'s defaults,
`scripts/setup.sh`, `.env.example`, CI and the usage docs all serve as
`insideout_app`, and `docs/usage/environment.md` already calls `DATABASE_URL`
"the runtime role `insideout_app`". The `migrate`-only fallback ("Empty =
migrate uses `DATABASE_URL`, which must already be that owner") still works for
the one-shot `migrate` command. So no DEFERRED_OWNER row.

## Evidence

All runs are against a real PostgreSQL 18 cluster on a port of its own,
migrated from this worktree, with DSNs that exist only for the run. A skipped
database test counts as red; none skipped in the scoped runs.

| Finding | Red | Green |
| --- | --- | --- |
| rv4-DATA-01 | at `92a0230`, `TestRLSWriteWalk_MembershipInsert` fails five cases on the base schema, e.g. `rls_write_walk_test.go:109: bare self-insert as admin: want a refusal (SQLSTATE 42501), got <nil>`; the same line for a bare self-insert as member, a wrong join code, the right join code asking for admin, and an admin of another workspace | at `06e8b78` it passes; the legitimate self-insert as member with the right code stays allowed; `TestAuthz_WorkspaceMembershipLifecycle` and `TestRLSWalk` still pass |
| rv4-DATA-02 | at `92a0230`, `--- FAIL: TestRLSWriteWalk_MembershipInsert` and `--- FAIL: TestRLSWriteWalk_UserColumns`; `TestRLSWriteWalk_NoRowChanges` passes (a regression guard, not a red) | at `571a5d8` all three pass, with `TestRLSWalk`, `TestAuthz_*` and `TestSetSystemRole` |
| rv4-DATA-03 | at `92a0230`, `TestRLSWriteWalk_UserColumns` fails twelve cases: an `UPDATE` of own `email`, `role`, `created_at`, `system_role`, `email_verified_at` and `meta` (line 179) and an `INSERT` that sets `id`, `role`, `system_role`, `email_verified_at`, `meta` or `created_at` (line 197) all return `<nil>` where a 42501 refusal is wanted. Still red at `06e8b78` | at `571a5d8` it passes, `CreateUser`, `UpdateProfile` and `UpdatePasswordHash` unchanged, `TestPromoteAdminWithoutOwnerPoolSaysSoAndRecovers` still passes |
| rv4-DATA-04 | at `98c5ba8`, `TestServeRefusesAPoolRoleThatBypassesRLS` fails after 20.01s: `serve was still running 20s after start on a pool role with BYPASSRLS: it must refuse to start` | at `4af1df7` it passes in 0.02s (exit 1, output names `BYPASSRLS` and `DATABASE_URL`, holds no DSN, password or host); the three store tests `TestServeRoleAcceptsTheAppRole`, `TestServeRoleRefusesABypassRLSPool` and `TestServeRoleNamesEveryUnsafeAttribute` pass |

Outside the Go suite, on the same cluster:

- **psql as `insideout_app`** (the role the server connects as; a transaction per
  attack, each rolled back). Denied: a bare self-insert as admin of another
  user's workspace, a member insert with a wrong join code, an admin insert
  holding the right code, adding someone else as member with the right code, the
  creator adding someone else as admin (all "new row violates row-level security
  policy", SQLSTATE 42501); an `INSERT` that sets `system_role` ("permission
  denied for table users"); an `UPDATE` of own `system_role`, `email_verified_at`
  or `email` (the trigger's 42501). Still answering "0 rows, no error": an
  `UPDATE` of someone else's `system_role`, and the same with no actor at all.
  Allowed: the creator's admin row written with its workspace, `JoinWorkspace`'s
  member row with the code set, `UpdateProfile` and `UpdatePasswordHash`.
- **Real binary, real server.** The final binary served as `insideout_app` on a
  loopback port of its own: `promote-admin` (owner pool, password on stdin)
  created the first admin, who logged in and opened signup; two ordinary users
  registered; the attacker got 403 on admin settings. The admin created a
  workspace through the API (201, creator role `admin`); the attacker's join with
  a wrong code was 404; the joiner's join with the right code was 200 and a
  second join was 409; `PATCH /me` updated username and bio and left
  `systemRole` at `member`. Then psql as `insideout_app` against that real
  workspace and that real attacker id: the admin-membership `INSERT` and the
  `system_role` `UPDATE` both failed with 42501, and the attacker was still 403
  on admin settings and 404 on the workspace.
- **Refusal, live.** The same binary with `DATABASE_URL` set to the owner DSN
  exited 1 at once with the single error line (`refusing to serve` ... the role
  in `DATABASE_URL` has BYPASSRLS ...), left no listener on its port, and its
  output held no part of the DSN (checked for the DSN, the password,
  host:port and the role name). The base binary given the same DSN served
  `/healthz` 200 on a `BYPASSRLS` role, which is the finding reproduced.
- **Migration, twice.** A fresh private database migrated from this worktree
  (every file applied); a second `migrate` applied nothing ("already up to
  date"). The new file was also re-applied by hand twice as `insideout_owner`;
  a catalog snapshot (policies, column and table grants, triggers, function
  body hashes) was identical across the three states.

## Known ceilings, recorded on purpose

- **rv4-DATA-03 read half is open** (the row is PARTIAL). `users_select` lets a
  row through to anyone who shares a workspace and `insideout_app` holds
  table-wide `SELECT`, so a co-member's `password_hash` is readable by anything
  that holds the app role. No API response carries it (`PasswordHash` never
  leaves the store), so this is defense in depth against a compromised app role
  or a future injection, as the finding says of the write half. Closing it needs
  files outside this cluster: `internal/store/users.go` (`userColumns` loads
  `password_hash` for every user query) and `internal/api/auth.go` (lines 167 and
  177 are the only readers of `PasswordHash`, in login). Follow-up proposal,
  about 25 lines and one test: a `SECURITY DEFINER` helper
  `insideout._credential_for_email(email)` returning `(id, password_hash)`;
  `REVOKE SELECT ON insideout.users FROM insideout_app` and re-grant every column
  but `password_hash`; drop `password_hash` from `userColumns`; have
  `GetUserByEmail`'s login caller use the helper; extend the write walk with a
  `SELECT (password_hash)` as a co-member that must be refused.
- **A seat can still be re-pointed by its workspace's admin.** Outside the four
  findings, so unchanged: `workspace_memberships_update` has only a `USING`
  clause, which PostgreSQL also applies as the check on the new row, so an admin
  can rewrite a membership's `user_id` and move a seat to another account without
  that account's join code. The only application `UPDATE` of the table sets
  `role` (`memberships.go:99`). A pin on `user_id` and `workspace_id`, a trigger
  like `users_pin_columns`, would close it. Not done here: it is a new finding,
  and the admin already controls the seat and the workspace.
- **The serve check is a boot check on attributes.** It reads the pool's
  `current_user` once at startup. A role altered to `BYPASSRLS` later is noticed
  at the next restart, not while running; a role that reaches a privileged role
  by `SET ROLE` is outside what it sees (the server never issues one); table
  ownership and grants are not inspected, because `FORCE ROW LEVEL SECURITY` and
  the walks cover those.
- **Deploy-time effect.** A deployment whose `DATABASE_URL` is a privileged role
  (the managed dashboard `postgres` user, for instance) exits 1 at boot with
  this image, before migrating. The hosted environment was not inspected from
  here. Before the first deploy of this wave, confirm `DATABASE_URL` is the
  `insideout_app` role and `DATABASE_OWNER_URL` the owner; rollback is
  redeploying the previous image.
- **`SUPERUSER` is covered by a pure table test only.** A superuser role cannot
  be made on the shared database without a `CREATE ROLE` outside the `insideout`
  schema, so `servePrivilegeError` is tested directly for `SUPERUSER`, `BYPASSRLS`
  and both, and `BYPASSRLS` is tested end to end with the owner pool.

## Left for the integrator

Documentation, comment and index edits only; no code is waiting except the
DATA-03 read half above.

- `docs/usage/environment.md` (the fail-fast list at line 235, and line 96 whose
  owner-fallback sentence stays true for `migrate`) and `docs/SETENV.md` (line
  302, the same list): one sentence that `insideout` (serve) now exits 1 when the
  `DATABASE_URL` role has `BYPASSRLS` or is a superuser. It is raised after the
  pool opens, not by `config.Load`, so say so.
- `.env.example` (the comment above `#DATABASE_OWNER_URL=`: "Empty = migrate uses
  DATABASE_URL (only valid if that role already is insideout_owner)"): add that
  serve refuses that role. `docs/usage/deployment.md` (line 44 table): the hosted
  `DATABASE_URL` must be the app role.
- `docs/architecture/backend.md` line 74 ("the only path to admin is the
  `insideout promote-admin <email>` operator subcommand") is now also enforced by
  the database; no edit is needed for it to stay true.
- `docs/architecture/index.md`: no edit (its description of the database doc is
  generic). No new Go package.
- The open-findings record and `docs/issues/README.md`: per-ID dispositions
  rv4-DATA-01, rv4-DATA-02 and rv4-DATA-04 `FIXED`; rv4-DATA-03 `PARTIAL` with
  the read half and its follow-up above, and the re-pointable-seat observation
  as a new low item; this folder's `index.md` wave-2 row; the changelog index
  files.

## Verification

Run against a private PostgreSQL 18 cluster on a port of its own (roles
`insideout_owner` and `insideout_app`, the same two the project uses), with the
server on loopback ports of their own.

- `go test ./internal/store/... -run 'RLS|ServeRole' -count=1 -v`: seven
  top-level tests pass (`TestRLSWalk`, three `TestRLSWriteWalk_*`, three
  `TestServeRole*`), 0 failed, 0 skipped.
- `go test ./internal/store/... ./cmd/insideout/... -race -count=1 -v`: 149
  passed, 0 failed, 1 skipped (`TestReadSecretTerminal`, which needs a terminal
  and is skipped on the base too).
- The whole server suite once at the end, `go test ./... -count=1`: 27 packages
  ok, 3 without test files, 0 failed.
- `gofmt -l .` prints nothing; `go build ./...` and `go vet ./...` clean.
- Every `scripts/tests/check_*.py` (9) exits 0; `sh scripts/check-redaction.sh`
  prints nothing; `sh scripts/check-docs-recorded.sh` ok; every touched file is
  at most 350 lines (the largest is `main.go` at 259).

One hazard hit on the way and worth knowing: the live proof opened signup
through the admin API, and `signup_mode` is persisted shared state, so a later
`TestSignupModeSetting` run failed with `seeded signup_mode = "open", want
invitation_only (fail closed)` until the setting was put back. That test only
restores a value it read. Reset the setting after any live proof that opens
signup on a database the suite also uses.
