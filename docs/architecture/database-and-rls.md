# Database and RLS

## Schema ownership models

InsideOut uses **two NOSUPERUSER roles**. The go-rewrite D2 single-role
shortcut (`insideout_app` owns everything and connects at runtime) is
reversed — `SECURITY DEFINER` helpers must not run as a superuser, and they
must not be owned by the runtime login either (that re-triggered FORCE RLS
inside `_is_member`; see
[BUG-007](../issues/2026-07-20-bug-007-rls-against-real-postgres.md)).

| Role | Superuser | Login | Purpose |
| --- | --- | --- | --- |
| bootstrap (`postgres` image user) | yes | init only | `CREATE ROLE`; never owns `insideout` objects |
| `insideout_owner` | **no** (`BYPASSRLS`) | migrate (`DATABASE_OWNER_URL`) | Owns the `insideout` schema, tables, and DEFINER functions |
| `insideout_app` | no (nor `BYPASSRLS`) | runtime (`DATABASE_URL`) | DML only; subject to RLS; the server refuses to serve on a role that is not |

Two PostgreSQL topologies, same roles and the same migration set:

1. **Dedicated instance** (bundled `docker-compose` `postgres`): init
   creates both roles. The database owner is `insideout_owner`.
2. **Shared instance** (multi-tenant managed Postgres): provision both
   roles with [`server/db/provision_roles.sql`](../../server/db/provision_roles.sql)
   as the instance bootstrap superuser, then `REASSIGN OWNED` into
   `insideout_owner`. Migrations never touch `public`.

The Go server **connects as `insideout_app`** and refuses to start on a role
with `BYPASSRLS` or superuser (see
[Membership and user writes](#membership-and-user-writes)). DDL/migrate uses
`DATABASE_OWNER_URL` (`insideout_owner`). If a superuser is used to apply
migrations, the runner `SET LOCAL ROLE insideout_owner` first so DEFINER
functions are never superuser-owned.

## Migrations

`server/db/migrations/*.sql` are timestamp-prefixed, embedded into the binary
via `//go:embed` (`server/db/embed.go`), and applied by the server's own
runner (`server/internal/store/migrate.go`) — no external migration tool.
Applied filenames are tracked in `insideout.schema_migrations`; each file runs
in its own transaction, in filename order.

Every foreign key of the schema has an index on its referencing columns
(migration `20261018210000_index_foreign_keys.sql`, rv4-DATA-07). PostgreSQL
indexes only the referenced side, so without one a cascade scans the child table
once per deleted parent row. The migration finds the unindexed keys in the
catalog instead of listing them, and `store/fk_index_test.go` reads the catalog
after `migrate` and fails when a new foreign key arrives without a supporting
index (its exemption map is empty; an entry needs a written reason).

The bootstrap step only creates the `insideout` schema when it is actually
missing (checked via `information_schema.schemata`) — `CREATE SCHEMA`, even a
redundant `IF NOT EXISTS`, requires database-level `CREATE` privilege, which
a schema-scoped role (model 2 above) does not have even for its own already-
owned schema. See
[BUG-007](../issues/2026-07-20-bug-007-rls-against-real-postgres.md#1-migration-bootstrap-unconditionally-ran-create-schema-if-not-exists).

## JWT + RLS defense-in-depth

Every store function that touches an RLS-protected table runs inside
`Store.withUserContext(ctx, actorID, fn)` (`server/internal/store/pool.go`):
it opens a transaction, sets `app.user_id` via `SELECT
set_config('app.user_id', $1, true)` from the JWT-validated caller's ID, runs
`fn`, and commits. RLS policies (`server/db/migrations/20260720150000_row_level_security.sql`
onward) read that value via a small `insideout.current_user_id()` SQL
function and enforce the same authorization checklist the Go app layer
already checks — a database-level backstop against app-layer bugs. `insideout_app` is
not the table owner, so RLS applies to the runtime login. DEFINER
helpers run as `insideout_owner` (NOSUPERUSER, `BYPASSRLS`) so membership
lookups do not recurse under FORCE RLS.

16 tables carry RLS, every one of them `FORCE`d (`workspace_memberships` too,
since the 2026-08-19 role split): `agent_conversations`, `agent_messages`,
`ai_runs`, `ideas`, `prd_commits`, `prd_revisions`, `prds`, `project_updates`,
`projects`, `proposal_decisions`, `proposal_items`, `roadmap_evidence`,
`roadmap_nodes`, `users`, `workspace_memberships` and `workspaces`.
`sessions`, `auth_tokens`, `invitations`, `settings`, `ai_run_events`,
`ai_circuit_breaker` and the migration ledger `schema_migrations` are
intentionally left without RLS: backend-only auth plumbing, deployment
settings, system telemetry and bookkeeping with no per-end-user access story.

Two system-context exceptions are load-bearing and deliberate, and they are
the only RLS tables that trust an unset `app.user_id`
(`current_user_id() IS NULL`), since `insideout_app` is never called by
anything but this repo's own backend. `users`: `CreateUser` (registration) and
`GetUserByEmail` (login) run **without** `app.user_id` set — there is no
authenticated identity yet at that point in the flow; login reads the stored
credential only through `insideout._credential_for_email`, which answers in that
same unset-context state and nowhere else (the users read grant, below).
`ai_runs`: the stale-run
reaper (`ReapStaleAIRuns`) is a background sweep with no acting user
(migration `20260721160000`). Joining a workspace by its workspace code has a
similar, narrower exception: `JoinWorkspace` sets a second session variable,
`app.join_code`, so the one-time "look up a workspace I'm not a member of yet,
by the code I was given" read can pass RLS without a blanket bypass (see the
`workspaces_select` policy).

`server/internal/store/rls_walk_test.go` (`TestRLSWalk`) holds this boundary
mechanically rather than by memory: it reads the RLS tables from `pg_class`,
requires each to be `FORCE`d and listed in its `rlsTables` map, and checks that
a member, a user in no workspace, and a session with no `app.user_id` see
exactly what is described above. A new RLS table, or a new policy (on any
command) that trusts an unset user, fails it until a reviewer decides which way
it goes.

That walk only reads, and a policy that lets the wrong `INSERT` or `UPDATE`
through is invisible to its `SELECT` counts. Its write twin,
`server/internal/store/rls_write_walk_test.go` (`TestRLSWriteWalk_*`,
rv4-DATA-02), runs as `insideout_app` in transactions that are always rolled
back: the membership `INSERT` matrix (a bare self-insert as admin is the first
case), an `UPDATE` of every `users` column (the list comes from the catalog, so
a new column fails the walk until it is classified) and an `INSERT` that sets
each privileged one, and an `UPDATE` and a `DELETE` on every RLS table as an
outsider and as an actor-less session, which must change no row, with a
member's run as the control that statements can change rows at all.

Authorship is not membership (SEC-11). The author branches of `ideas_select`,
`ideas_update` and `prds_insert` also require
`insideout._is_member(<table>.workspace_id, current_user_id())` (migration
`20261010120000`), and so do their four siblings `prds_update`,
`project_updates_update`, `project_updates_delete` and `prd_revisions_insert`
(migration `20261010130000`, which reaches the workspace through
`_project_workspace` and `_prd_workspace` where the row has no
`workspace_id`). A member removed from a workspace therefore loses read, edit,
drop and convert over the ideas they wrote there, cannot insert a PRD into it,
and can no longer edit or snapshot their PRDs or edit or delete their project
updates. The admin branches need no change, since `_is_admin` implies
membership.

The Go store checks the same: `requireMember` in `UpdateIdea`, `DropIdea` and
`ConvertIdea` (and the membership predicate of `GetIdeaForMember`), and ahead
of the author-or-admin rule in `requirePrdAuthorOrAdmin` (so `UpdateSections`
and `EnsureConversationForPrd`), `UpdatePrdStatus`, `CreateRevision`,
`CreatePrdCommit`, `UpdateProjectUpdate` and `DeleteProjectUpdate`. Through
`insideout_app` a removed member therefore gets `ErrNotFound` (RLS hides the
row, exactly as for a stranger); the Go checks alone answer `ErrForbidden`.
Both layers are pinned separately: the policies, one bare statement at a time
under the runtime role, by `server/internal/store/ideas_exmember_test.go` and
`authored_exmember_test.go`; the Go layer, by running the same store on the
`BYPASSRLS` owner pool, by the `GoLayerDenies` tests in the first and in
`authored_exmember_golayer_test.go`.

## Membership and user writes

Migration `20261018140000` closes the two write paths the policies left open
(rv4-DATA-01, rv4-DATA-03): any signed-in session could insert itself as
`admin` of a workspace whose id it knew, and one statement could make an
account an admin.

**A membership row needs a reason.** `workspace_memberships_insert` used to
check only `user_id = current_user_id()`. It now also requires
`insideout._may_insert_membership(workspace_id, current_user_id(), role)`: an
`admin` row only for the workspace's creator (what `createWorkspaceTx` writes
right after the workspace itself), a `member` row only while the workspace is
active and `app.join_code` holds its code (what `JoinWorkspace` sets, as
`workspaces_select` already reads it). The helper is `SECURITY DEFINER` so that
reading `workspaces` from inside a `workspace_memberships` policy cannot recurse
through `workspaces_select`. A caller therefore cannot add anyone but
themselves, cannot self-promote, and a join code buys `member` and nothing more.

**`users` columns are an allow-list.** `insideout_app` held table-wide `INSERT`
and `UPDATE` on `insideout.users` while `users_insert` is `WITH CHECK (true)`
and `users_update` only pins the row to its owner, so a session could register
as `system_role = 'admin'` or rewrite its own `role`, `system_role`, `email`,
`email_verified_at` or `meta`.

- `INSERT` is cut down by column grants to `(email, password_hash, username)`,
  all that `CreateUser` writes. Any other column, one added later included, is
  refused with SQLSTATE 42501 before a row exists. The table-level `REVOKE`
  comes first, because a table-level privilege masks column grants.
- `UPDATE` is limited by the `users_pin_columns` trigger
  (`insideout._pin_user_columns`), not by column grants. A column privilege is
  checked before any row is read, so `promote-admin` run without
  `DATABASE_OWNER_URL` (an `UPDATE` of someone else's `system_role` on the app
  pool) would stop answering "no row", which the CLI turns into its "set
  `DATABASE_OWNER_URL`" hint, and fail with "permission denied" instead. The
  trigger only sees rows RLS already let through. It compares `NEW` with `OLD`
  after removing the columns a profile edit really changes (`username`,
  `avatar_url`, `bio`, `keywords`, `password_hash`; `updated_at` belongs to
  `set_updated_at`) and raises 42501 on any difference, so a column added later
  stays pinned until someone adds it to that list. A role RLS does not apply to
  (`BYPASSRLS` or superuser: `insideout_owner`, which runs migrations and
  `promote-admin`) is trusted with every column, the same line RLS draws.

**`users.password_hash` is not readable by the app role** (rv4-DATA-03, read
half; migration `20261018200000_users_read_grants.sql`). `users_select` shows a
user every co-member's row, and `insideout_app` once held table-wide `SELECT`,
so a co-member's stored credential was one `SELECT password_hash` away for a
read bug (`SELECT *`, a join, a `RETURNING`, a dump). The migration revokes the
table-wide `SELECT`, grants every column except `password_hash` by name, and
ends in a `DO` assertion that fails if the app role can still read the column
through an inherited role or `PUBLIC`. Login, the one flow that verifies a
password, reads it through `insideout._credential_for_email(email)`: a
`SECURITY DEFINER` function owned by `insideout_owner`, `search_path` pinned,
`EXECUTE` for the app role only, which returns a row only while `app.user_id` is
unset. Inside a request's own transaction it returns nothing, so a handler that
forgets this cannot read a co-member's hash by passing an email. `UPDATE` stays
table-wide (pinned by `users_pin_columns`; `SET password_hash = $2 WHERE id = $1`
needs only `UPDATE` on the column and `SELECT` on `id`). Ceilings: the
unset-context condition is a convention the app role could satisfy with
arbitrary SQL (a hard boundary needs a second role and pool); the sibling hash
columns on `sessions`, `auth_tokens` and `invitations` are unchanged; a later
blanket `GRANT` on `users` is caught only by the CI guard
(`users_read_grant_test.go`); a new `users` column needs its own `GRANT SELECT`;
and the previous image fails user reads once this migration is applied (the
rollback statement is in the
[w3-users-read-grant record](../changelogs/2026-10-18-rv4-remediation/w3-users-read-grant.md)).

**The server refuses to serve on a role that skips RLS** (rv4-DATA-04).
`BYPASSRLS` and superuser ignore every policy above, `FORCE ROW LEVEL SECURITY`
included. `runServe` calls `Store.CheckServeRole`
(`server/internal/store/serve_role.go`) first thing, before the startup
migrations: it reads `rolsuper` and `rolbypassrls` for the pool's own role from
`pg_roles` and exits 1 with a plain error that names the attribute and
`DATABASE_URL`. The error never carries the role name or the DSN, since it
reaches CI and hosted logs. Only the request pool is checked: `migrate`, `seed`
and `promote-admin` do not go through `runServe`, and the migrate pool is meant
to be `insideout_owner`. Every documented setup (compose, `scripts/setup.sh`,
`.env.example`, CI) already serves as `insideout_app`; the dashboard `postgres`
user of a managed instance is the case this catches.

**Proposal items are immutable apart from one stamp** (rv4-FEAT-02, migration
`20261018120000`). `proposal_items.applied_at` (NULL = not applied, which is
every row that existed before) is what makes accepting a proposal with `apply`
idempotent: the apply claims an item with `UPDATE ... WHERE applied_at IS NULL`
in the transaction that creates its node, so a repeat, a race or a rolled-back
apply cannot build it twice. The app role's table-wide `UPDATE`, which the
default privileges of migration `20260819190000` had granted, is revoked and
re-granted on that one column, and the policy `item_applier_update` lets only
the project owner or a workspace admin stamp. `DELETE` stays granted by the
default privileges, but no `DELETE` policy exists, so RLS denies it.

## Known PostgreSQL/RLS gotchas

Found only by running real migrations and real queries against a live
database — see [BUG-007](../issues/2026-07-20-bug-007-rls-against-real-postgres.md)
for the full investigation. In short, in case a future change to RLS
policies needs to route around any of these again:

- A policy that queries its own table from within its own `USING`/`WITH
  CHECK` expression triggers "infinite recursion detected in policy" — a
  static, structural-cycle check, not a runtime one.
- Wrapping the self-check in a `SECURITY DEFINER` function does **not**
  bypass FORCE RLS if the function's owner is the same as the table owner
  and that owner is also the connecting role. The 2026-08-19 split owns
  DEFINER helpers as `insideout_owner` (`BYPASSRLS`, not superuser) while
  `insideout_app` connects at runtime, so membership FORCE RLS is restored.
- A historical workaround (`NO FORCE` on `workspace_memberships`, migration
  `20260720153000`) applied only under the single-role model; do not re-copy
  it. Go's `requireMember` / `requireAdminMember` remain in
  `internal/store/memberships.go`.
- `SELECT ... FOR UPDATE` / `FOR KEY SHARE` on a table whose RLS policy
  references another table silently returns **zero rows** — not an error —
  because Postgres's `EvalPlanQual` row-locking re-check does not correctly
  re-evaluate a cross-table policy reference. Fixed by removing the explicit
  row lock from the seven affected functions (`requireCreatorOrAdmin` and
  friends); the check and the write still happen in the same transaction,
  which is the property that actually matters for the TOCTOU rule.
- An RLS policy's own subquery alias can collide with an *application*
  query's alias for the same table (Postgres merges the policy expression
  textually into the query it protects). Fixed by moving every cross-table
  policy check into small `SECURITY DEFINER` helper functions
  (`insideout._is_member`, `_is_admin`, `_project_workspace`,
  `_prd_workspace`, `_prd_author`, `_shares_workspace`,
  `_conversation_owner`, `_may_insert_membership`) — a function call
  introduces no alias into the outer query at all. A related,
  easy-to-repeat mistake: a **bare**,
  unqualified column name passed as that function's argument is *still*
  ambiguous if the outer query joins another table with the same column
  name (the argument is resolved in the outer query's scope, not the
  function's) — every argument must be the fully table-qualified form
  (`insideout.projects.workspace_id`, not `workspace_id`).
- Column privileges and row security answer at different moments. A column
  privilege is checked before any row is read, so narrowing `UPDATE` with a
  column `GRANT` turns "RLS hid the row, 0 rows changed" into "permission
  denied"; a `BEFORE UPDATE` trigger only sees the rows RLS let through. Also,
  `GRANT INSERT (cols)` is masked by a table-level `INSERT`, so `REVOKE` the
  table-level privilege first (a table-level `REVOKE` clears column grants too,
  which is what makes the migration re-runnable).
- `jsonb` columns bound from Go `map`/`[]byte` values fail under
  `pgx.QueryExecModeSimpleProtocol` (required for PgBouncer/Supavisor
  transaction-pooling connections — see [deployment](deployment.md)): Go
  `map` types have no default encoder at all, and `[]byte` defaults to
  `bytea` (hex-escaped) rather than plain text, which an `::jsonb` cast
  cannot parse. Fixed by `json.Marshal`-ing to a Go `string` explicitly and
  casting in SQL (`$N::jsonb`).

## Authorization checklist

The full per-resource rule table (who can read/write what, and the deny
paths each rule implies) lives in
[`docs/plans/2026-07-20-go-rewrite/01-database.md`](../plans/2026-07-20-go-rewrite/01-database.md#5-authorization-checklist--授权清单),
and is exercised end-to-end by `server/internal/store/authz_test.go` against
a real database.
