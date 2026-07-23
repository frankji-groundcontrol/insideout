# BUG-007: Five distinct bugs found only by running RLS + migrations against a real PostgreSQL database

**Found**: 2026-07-20, during the InsideOut rewrite (P1–P4), implementing JWT + RLS defense-in-depth (`db/migrations/20260720150000_row_level_security.sql` onward) and then actually running the authorization-checklist integration tests (`internal/store/authz_test.go`) against the real, shared-instance `insideout` schema for the first time. None of these were, or could have been, caught by unit tests or `go vet`/`go build` — they only exist as real Postgres runtime behavior.

## 1. Migration bootstrap unconditionally ran `CREATE SCHEMA IF NOT EXISTS`

**Symptom**: `migrate: bootstrap: ERROR: permission denied for database postgres`.

**Root cause**: `CREATE SCHEMA` — even a redundant `IF NOT EXISTS` against a schema the role already owns — requires `CREATE` privilege on the *database*, checked before existence is even considered. A schema-scoped role (this shared instance's provisioning model — see BUG-008 in the same investigation) has no such privilege, only the dedicated-instance model (where the role owns the whole database) does.

**Fix**: `internal/store/migrate.go`'s `Migrate()` now checks `information_schema.schemata` first and only attempts `CREATE SCHEMA` when it's actually missing.

## 2. RLS policy self-referencing its own table: "infinite recursion detected in policy"

**Symptom**: `ERROR: infinite recursion detected in policy for relation "workspace_memberships"` (SQLSTATE 42P17), on the very first `CREATE ROLE`-adjacent query.

**Root cause**: `workspace_memberships_select`'s policy needs to answer "is the current user also a member of this row's workspace" — which requires querying `workspace_memberships` from within its own policy. Postgres detects that self-reference as a structural cycle and refuses to plan it at all.

**Fix attempt that didn't fully work**: wrapping the check in a `SECURITY DEFINER` function (`insideout._is_member`). This broke the *static* recursion detection, but once real rows existed, it caused a *different*, worse failure (see #3) — `SECURITY DEFINER` doesn't bypass `FORCE ROW LEVEL SECURITY` when the function owner is the same role as the table owner (`insideout_app`, in both cases).

**Actual fix**: `ALTER TABLE insideout.workspace_memberships NO FORCE ROW LEVEL SECURITY`. Since `insideout_app` is the only role that will ever connect (single-role model), Postgres's normal (non-forced) owner-bypass is safe here, and Go's `requireMember`/`requireAdminMember` (`internal/store/memberships.go`) already fully enforce this table's rules transactionally. The policies stay defined for documentation and in case a lower-privileged role is ever added.

## 3. "stack depth limit exceeded" once real rows existed

**Symptom**: `ERROR: stack depth limit exceeded` (SQLSTATE 54001) on `JoinWorkspace`'s duplicate-join check, only once the membership table actually had rows to evaluate.

**Root cause**: the `SECURITY DEFINER` function from #2's first attempt still executed as `insideout_app`, so its own internal query against `workspace_memberships` re-triggered the same policy, which called the function again — genuine unbounded runtime recursion, not just a static planning-time cycle. An empty table never actually reaches this code path, which is why the first (wrong) fix looked like it worked.

**Fix**: same as #2 — un-forcing RLS on `workspace_memberships` removes the recursive call path entirely.

## 4. `FOR UPDATE`/`FOR KEY SHARE` silently returns zero rows when the table's RLS policy references another table

**Symptom**: `requireCreatorOrAdmin`'s `SELECT ... FOR KEY SHARE OF w` returned 0 rows for a real, visible member — no error, just silent wrong behavior. Isolated by testing the identical query with and without the locking clause.

**Root cause**: this is a genuine PostgreSQL limitation, not a bug in the policy logic — confirmed by testing a `SECURITY DEFINER`-wrapped version of the same check, which *also* failed identically under `FOR KEY SHARE`. Postgres's `EvalPlanQual` re-check during row locking does not correctly re-evaluate a cross-table RLS policy reference.

**Fix**: removed the explicit row lock from all 7 affected functions across `workspaces.go`, `projects.go`, `project_updates.go`, `ideas.go`, `prds.go`, `prd_revisions.go` — the check and the write still happen in the same transaction (the property that actually matters for the TOCTOU rule in `01-database.md` §5), just without the extra lock, which never protected against the most likely race anyway (a role change touches `workspace_memberships`, a table these locks never touched). `// ponytail:` comments mark the narrower TOCTOU window and the upgrade path (embed the same condition directly in the mutating statement's `WHERE` clause) if this race ever matters in practice.

## 5. `jsonb` parameter encoding fails under the simple query protocol (pgbouncer/Supavisor transaction mode)

**Symptom**: two different errors depending on the Go type — `unable to encode map[string]string{...} into text format for unknown type (OID 0): cannot find encode plan` for a raw `map[string]string`, and `invalid input syntax for type json` for a `[]byte`/`json.RawMessage` value even after adding an explicit `::jsonb` cast.

**Root cause**: under `pgx.QueryExecModeSimpleProtocol` (required for pgbouncer transaction pooling — see BUG-005-adjacent `pool.go` change), pgx has no server round-trip to learn a parameter's target column type. Arbitrary Go map types have no default encoder at all (first error). A `[]byte` value *does* have a default encoder — but it's `bytea` (hex-escaped), and an explicit `::jsonb` cast on that hex-escaped text tries to parse the *escaped hex string itself* as JSON, not the underlying bytes (second error).

**Fix**: `json.Marshal(...)` explicitly in Go, then bind the result as a Go `string` (not `[]byte`) with an explicit `::jsonb` cast in the SQL text. A `string` is embedded as an unescaped text literal under simple protocol, which the cast can actually parse. A `jsonParam(json.RawMessage) any` helper in `internal/store/agent_messages.go` handles the nullable cases (nil stays nil/SQL NULL, non-nil becomes a string) for `tool_calls`/`meta`/`request_messages`/`response_payload`.

## Bonus: two pre-existing, RLS-unrelated bugs surfaced by the same test run

- **Alias collision** (SQLSTATE 42702, "column reference is ambiguous"): an RLS policy's own subquery alias (e.g. `m` for `workspace_memberships`) collided with an *application* query that happened to alias the same table the same way (`GetProjectForMember` joins `workspace_memberships m` too — Postgres merges the policy expression textually into the same query scope). Fixed by moving every cross-table policy check into `SECURITY DEFINER` helper functions (`_is_member`, `_is_admin`, `_project_workspace`, `_prd_workspace`, `_prd_author`, `_shares_workspace`, `_conversation_owner`), which introduce no alias into the outer query at all. A related mistake in the first attempt — passing a *bare*, unqualified column name (`workspace_id`) as a function argument — was itself ambiguous whenever the outer query joined another table with the same column name; every argument now uses the fully table-qualified form (`insideout.projects.workspace_id`), which Postgres correctly rebinds regardless of the outer query's aliasing.
- **`SELECT p.`+columns only qualifies the first column**: `GetProjectForMember`, `GetIdeaForMember`, and `GetPrdForMember` (and `ListProjectsForWorkspace`) built their SELECT list as `"SELECT p." + projectColumns` — Go string concatenation only prefixes the *first* column (`p.id`); every other column in the shared `*Columns` constant stayed unqualified. This was a real, dormant ambiguity bug (each of these queries joins another table sharing a column name) since the functions were first written — nothing before this session's real-database integration tests ever executed them against a schema where the ambiguity could surface. Fixed with a `qualifyColumns(alias, cols string) string` helper in `internal/store/pool.go` that prefixes every column.

## Why it matters

Every one of these is invisible to `go build`, `go vet`, and pure-Go unit tests — they are real Postgres/pgx runtime behavior, several of them (#2–#4) specific, non-obvious PostgreSQL RLS limitations that don't show up in documentation examples. This is the strongest argument in this codebase for never marking database-dependent work "done" without running it against a real database — see `docs/testing/item-32-testing.md`'s "no mocks" rule.
