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
| `insideout_app` | no | runtime (`DATABASE_URL`) | DML only; subject to RLS |

Two PostgreSQL topologies, same roles and the same migration set:

1. **Dedicated instance** (bundled `docker-compose` `postgres`): init
   creates both roles. The database owner is `insideout_owner`.
2. **Shared instance** (multi-tenant managed Postgres): provision both
   roles with [`server/db/provision_roles.sql`](../../server/db/provision_roles.sql)
   as the instance bootstrap superuser, then `REASSIGN OWNED` into
   `insideout_owner`. Migrations never touch `public`.

The Go server **connects as `insideout_app`**. DDL/migrate uses
`DATABASE_OWNER_URL` (`insideout_owner`). If a superuser is used to apply
migrations, the runner `SET LOCAL ROLE insideout_owner` first so DEFINER
functions are never superuser-owned.

## Migrations

`server/db/migrations/*.sql` are timestamp-prefixed, embedded into the binary
via `//go:embed` (`server/db/embed.go`), and applied by the server's own
runner (`server/internal/store/migrate.go`) — no external migration tool.
Applied filenames are tracked in `insideout.schema_migrations`; each file runs
in its own transaction, in filename order.

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

11 tables carry RLS: `users`, `workspaces`, `projects`, `project_updates`,
`ideas`, `prds`, `prd_revisions`, `agent_conversations`, `agent_messages`,
`ai_runs`, and `workspace_memberships` (policies **defined** but not
**forced** — see below). `sessions`, `ai_run_events`, and `ai_circuit_breaker`
are intentionally left without RLS: pure auth plumbing or system telemetry
with no per-end-user access story.

Two pre-auth exceptions are load-bearing and deliberate: `CreateUser`
(registration) and `GetUserByEmail` (login) run **without** `app.user_id` set
— there is no authenticated identity yet at that point in the flow. The
`users` RLS policy treats `current_user_id() IS NULL` as a trusted system
context, since `insideout_app` is never called by anything but this repo's
own backend. Joining a workspace by invite code has a similar, narrower
exception: `JoinWorkspace` sets a second session variable, `app.join_code`,
so the one-time "look up a workspace I'm not a member of yet, by the code I
was given" read can pass RLS without a blanket bypass (see the
`workspaces_select` policy).

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
  `_conversation_owner`) — a function call introduces no alias into the
  outer query at all. A related, easy-to-repeat mistake: a **bare**,
  unqualified column name passed as that function's argument is *still*
  ambiguous if the outer query joins another table with the same column
  name (the argument is resolved in the outer query's scope, not the
  function's) — every argument must be the fully table-qualified form
  (`insideout.projects.workspace_id`, not `workspace_id`).
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
