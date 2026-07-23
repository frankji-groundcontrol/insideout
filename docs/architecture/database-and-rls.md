# Database and RLS

## Schema ownership models

InsideOut supports two PostgreSQL provisioning models with the same
migration set, unmodified:

1. **Dedicated instance** (the bundled `docker-compose` `postgres` service):
   `insideout_app` owns the whole database, and therefore owns `public`
   within it by Postgres default.
2. **Shared instance** (a multi-tenant managed Postgres project, e.g. a
   Supabase project also hosting unrelated tenants' schemas): `insideout_app`
   is scoped to own only the `insideout` schema. It is never granted
   anything on `public` beyond the harmless default `USAGE` — "never write to
   `public`" is enforced by the migrations simply never targeting it, not by
   an unreliable `REVOKE` (see the migration comment in
   `server/db/migrations/20260720135749_schema_and_lockdown.sql` for why a
   `REVOKE CREATE ON SCHEMA public FROM PUBLIC` step would be a no-op on
   model 1 and a cross-tenant risk on model 2).

Either way, `insideout_app` is the **only** role the Go server ever connects
as — there is no separate admin/runtime role split (see the decision record
in [`docs/plans/2026-07-20-go-rewrite/README.md`](../plans/2026-07-20-go-rewrite/README.md),
D2).

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
already checks — a database-level backstop against app-layer bugs, not a
boundary against `insideout_app` itself (it is the only role that ever
connects).

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
  bypass this if the function's owner is the same role as the table's owner
  (true here, since `insideout_app` owns everything) — `FORCE ROW LEVEL
  SECURITY` still applies inside the function, and the recursion becomes a
  genuine runtime stack overflow once real rows exist instead of a
  compile-time-visible cycle.
- The actual fix for `workspace_memberships` specifically: `ALTER TABLE
  insideout.workspace_memberships NO FORCE ROW LEVEL SECURITY`. Since
  `insideout_app` is the sole connecting role, Postgres's normal
  (non-forced) owner-bypass is safe, and Go's `requireMember`/
  `requireAdminMember` (`server/internal/store/memberships.go`) already fully
  enforce this table's rules transactionally.
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
