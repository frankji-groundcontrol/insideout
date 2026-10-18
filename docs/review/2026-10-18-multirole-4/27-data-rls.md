# 27 · Data and RLS

**Lens:** migrations, schema confinement (L18), RLS policies and FORCE, SECURITY DEFINER functions, grants and ownership, retention and deletion, query shape and pagination, migration runner and CI replay, runtime DSN/role handling  
**Score:** **7 / 10**  
**Bottom line:** Schema confinement, role split, FORCE RLS on 16 tables and DEFINER hygiene are strong and proven on a private PG18.4 cluster. The write side of the RLS backstop is weak: any caller can self-insert as admin into any workspace, and users can self-escalate system_role. TestRLSWalk is SELECT-only, so none of this is red-lined. Retention and erasure are also one-sided.

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

## Skeptic verdicts on this lens's Highs

| ID | Filed | Verdict | Final | Reason |
|---|---|---|---|---|
| DATA-01 | High | DOWNGRADED | Medium | **[Correction]** The mechanism is real and I reproduced it. On a private PG 18.4 cluster (insideout_app NOBYPASSRLS, all 38 migrations applied) the live catalog still holds the first-migration policy: pg_policies shows workspace_memberships_insert with_check = (user_id = insideout.current_user_id()). Later migrations only rewrote the select, … |

---

## Data, migrations and RLS (data-rls)

### Scope

I read all 38 migrations in `server/db/migrations`, `provision_roles.sql`, `server/internal/store` (pool, migrate, migrate_role, sessions, auth_tokens, invitations, the list and write queries), `docs/architecture/database-and-rls.md`, the CI roleless and upgrade-replay jobs, and `internal/config`. I then stood up a private Postgres 18.4 cluster with CI-parity roles (`insideout_owner` BYPASSRLS, `insideout_app` NOBYPASSRLS), applied every migration through the server's own runner, and attacked it as `insideout_app` with `app.user_id` set to a second tenant. The cluster is stopped and the worktree is untouched. Claims marked "ran" were reproduced in psql on that cluster.

### Scores

| Aspect | Score | Why |
| --- | --- | --- |
| Schema confinement (L18) | 9.0 | No migration or query touches `public`, `extensions`, `CREATE EXTENSION`, or `ALTER DATABASE`. Ran: as the app role, `CREATE TABLE public.x`, `CREATE SCHEMA`, `CREATE EXTENSION`, and `CREATE TABLE/FUNCTION insideout.*` were all denied. No test pins it. |
| Read-side RLS | 9.0 | 16 tables are ENABLE+FORCE, ownership is split, and DEFINER helpers pin `search_path` and reference fully qualified tables. Ran: a non-member sees 0 rows. |
| Write-side RLS | 5.5 | `workspace_memberships_insert` lets any caller self-join as admin, and `users_update` is not column-scoped. |
| Migrations | 7.5 | Each file runs in a transaction under an advisory lock, as `insideout_owner`, with a CI upgrade replay. There is no checksum, no down story, and one data-rewrite UPDATE. |
| Retention and deletion | 5.0 | Auth rows and `ai_runs` are purged. Conversations and messages are never purged, and no user can be deleted. |
| Query shape | 7.0 | No N+1 found, and the lateral latest-update join is indexed. Several lists are unbounded and one cascading FK is unindexed. |
| **Overall** | **7.0** | Strong foundations and an honest doc. The write side of the backstop is thinner than the doc says. |

### What is good

- **Role split is enforced, not just documented.** `20260819190000_owner_app_grants.sql` raises if `insideout_owner` is a superuser or lacks BYPASSRLS, and `migrate_role.go` `prepareMigrateConn` does `SET LOCAL ROLE insideout_owner` for superuser callers. Ran: all tables and functions are owned by `insideout_owner`, and app holds no `CREATE` on the schema.
- **DEFINER hygiene is good.** All 8 DEFINER functions are owned by the non-superuser owner and pin `search_path = insideout, pg_catalog`. The bodies use only schema-qualified names, and the policies call them with table-qualified args, which closes the alias-collision class (BUG-007).
- **`TestRLSWalk` reads RLS tables from `pg_class`** (`rls_walk_test.go:12`) and forces new tables to be classified. It also fails on any new `current_user_id() IS NULL` clause.
- **The migration runner is careful.** It takes a transaction-scoped advisory lock (`migrate_role.go`), runs one transaction per file, and has a roleless-cluster CI job plus `upgrade-replay` in `ci.yml`.
- **Token consume paths are single-statement and atomic** (`auth_tokens.go` `ConsumeAuthToken`, `VerifyLoginCode`, `sessions.go` `RotateSession` with the affected-rows guard). Refresh reuse fails closed.

### Findings, worst first

**DATA-01 (High) Any caller can join any workspace as admin at the DB layer.**
`server/db/migrations/20260720150000_row_level_security.sql:109`: `FOR INSERT WITH CHECK (user_id = insideout.current_user_id());` Nothing constrains `role` or requires the join code. This policy is still the live one; no later migration replaces it. The `app.join_code` gate exists only on the `workspaces` SELECT policy and in Go. Ran: with `app.user_id` = a stranger, a bare `INSERT INTO workspace_memberships (workspace_id, user_id, role) VALUES (<victim ws>, <me>, 'admin')` succeeded. The stranger then saw the workspace, listed its members, and ran `UPDATE workspaces SET title=...` successfully. Because `users_select` shows co-members (`_shares_workspace`), the same session could also read `password_hash` of every member. Any future app-layer bug that lets a caller choose a workspace id, or a removed member who still knows the id, becomes a full takeover with no DB backstop. This contradicts the doc's "enforce the same authorization checklist" claim. Fix (about 15 lines): a new migration replacing the policy with `user_id = me AND ((role='member' AND <workspace joinable by app.join_code>) OR <workspace.creator_id = me>)`, using a DEFINER helper to avoid alias bugs. diff_size: about 15 lines.

**DATA-02 (Medium) `TestRLSWalk` is read-only, so write-side holes are invisible.**
`server/internal/store/rls_walk_test.go:12`: `// SEC-04 / SEC-13: the RLS boundary is checked, not remembered.` The walk only counts SELECT visibility per actor. No test attempts a bare INSERT/UPDATE across tenants (no test file has a direct `INSERT INTO insideout.workspace_memberships`), which is why DATA-01 and DATA-03 survived four review rounds. The doc (`database-and-rls.md:85`) says the walk holds the boundary "mechanically". Direction: add a write walk. First check: run the DATA-01 INSERT inside `withUserContext` for a non-member and expect `42501`.

**DATA-03 (Medium) `users_update` is not column-scoped, and co-members can read `password_hash`.**
`server/db/migrations/20260720150000_row_level_security.sql:47`: `USING (id = insideout.current_user_id())`. The grant is table-wide (`information_schema.role_table_grants` shows UPDATE and SELECT on all columns). Ran: as a normal user, `UPDATE insideout.users SET system_role='admin', role='admin', email_verified_at=now() WHERE id=<me>` returned `UPDATE 1`. `system_role` gates the admin UI (`api/admin.go`) and invitation minting. Today only `SetSystemRole` on the owner pool writes it, so this is defense-in-depth only. Fix: revoke table UPDATE and SELECT on `users` from `insideout_app`, then grant column-level UPDATE on the profile and password columns, and SELECT on non-secret columns. A BEFORE UPDATE trigger pinning `system_role`, `role` and `email_verified_at` is the smaller alternative. diff_size: about 10 lines.

**DATA-04 (Medium) The server never checks that its runtime role is subject to RLS.**
`server/cmd/insideout/main.go:51`: `st, err := store.Open(ctx, cfg.DatabaseURL)`. The only `rolbypassrls` checks are in tests and in `setup roles`. If an operator points `DATABASE_URL` at the owner DSN (the old single-role habit the doc still discusses), every FORCE RLS policy is silently bypassed and the server boots. Direction: after `Open`, query `rolbypassrls OR rolsuper` for `current_user` and refuse to serve unless an explicit dev override is set. First check: run `serve` against the owner DSN on a scratch cluster and confirm it starts.

**DATA-05 (Medium) A deleted-cursor row silently truncates the timeline.**
`server/internal/store/project_updates.go:85`: `AND ($2::uuid IS NULL OR (pu.created_at, pu.id) < (`. The cursor is resolved by a subselect on the row id. Ran: with three updates, paging with the middle one as cursor returned the older row, but after the cursor row was deleted (authors may delete via `project_updates_delete`) the same query returned zero rows. A client mid-scroll loses all older history with no error. Fix: make the cursor the pair `(created_at, id)` encoded in the token instead of resolving it in SQL. diff_size: about 15 lines.

**DATA-06 (Medium) Retention and erasure are one-sided; no user can ever be deleted.**
`server/db/migrations/20260720135754_agent.sql:4`: `user_id uuid NOT NULL REFERENCES insideout.users(id),`. Eleven user FKs are NO ACTION (`prds.author_id`, `ideas.author_id`, `ai_runs.user_id`, `agent_conversations.user_id`, and others), and no store function deletes a user. `agent_messages.go:261` purges `ai_runs` after 30 days, but `agent_conversations` and `agent_messages` (private coaching chats) and `ai_run_events.request_messages` are not purged on any schedule. The repo is public now. A real person asking to delete their account has no path, and no issue in `docs/issues` records it. Direction: record the policy decision, then add an erasure routine (anonymise authored rows, delete conversations) run on the owner pool.

**DATA-07 (Medium) A cascading FK read path has no index.**
`server/db/migrations/20260822130000_proposal_items.sql:7`: `update_id uuid NOT NULL REFERENCES insideout.project_updates(id) ON DELETE CASCADE,`. Ran: with 50k rows, `WHERE update_id = ...` (the `ListProposalItems` query, plus the RLS subquery) is a Seq Scan. The same applies to the cascade targets `ai_runs.workspace_id` and `agent_conversations.workspace_id`, and `roadmap_nodes.parent_id` (`roadmap_move.go:46`: `JOIN descendants d ON n.parent_id = d.id`, where the only index leads with `project_id`). Fix: `CREATE INDEX ON insideout.proposal_items (update_id, position)` plus indexes on the other three FK columns, in one migration. diff_size: about 6 lines.

**DATA-08 (Low) Cross-tenant FKs are accepted at the DB layer.**
`server/db/migrations/20260720135753_ideas_prds.sql:26`: `project_id uuid REFERENCES insideout.projects(id) ON DELETE SET NULL,`. Ran: a member of workspace B inserted a PRD in B whose `project_id` is a project in workspace A (FK checks bypass RLS). Go sets `project_id` only from its own project (`roadmap_plan.go:249`), so there is no live exploit; it is an integrity gap and a UUID-existence oracle. Fix: composite FKs `(workspace_id, project_id)` or a CHECK trigger.

**DATA-09 (Low) Decorative grants and a broad app grant.**
`server/db/migrations/20260821130000_roadmap_evidence.sql:45`: `GRANT SELECT, INSERT ON insideout.roadmap_evidence TO insideout_app;` The default privileges set in `20260819190000` already give app UPDATE and DELETE, so the "append-only" intent (also `prd_commits`, `proposal_items`, `proposal_decisions`) holds only because no RLS policy exists. Ran: `information_schema.role_table_grants` shows all four DML privileges on every table, including the `schema_migrations` ledger, `sessions` and `settings`. The app role can also `ALTER ROLE insideout_app SET search_path` (ran: succeeded) and can call the DEFINER oracles (`_is_member`, `_project_workspace`) with arbitrary ids. Fix: revoke DML on `schema_migrations`, and either drop the dead GRANT lines or revoke UPDATE and DELETE on the append-only tables.

**DATA-10 (Low) Operator subcommand does not case-fold email.**
`server/cmd/insideout/main.go:88`: `if err := st.SetSystemRole(ctx, email, "admin"); err != nil {`. The API lowercases (`auth.go:66`) and the DB column is a plain case-sensitive `UNIQUE`, so `promote-admin Name@Example.com` returns not-found for a stored lowercase address. Fix: `strings.ToLower(strings.TrimSpace(...))` there, and optionally a `CHECK (email = lower(email))`.

**DATA-11 (Low) Applied migrations are not checksummed, and there is no stated rollback story.**
`server/internal/store/migrate.go`: `applied[name] = true`. The ledger stores only a filename, so editing an already-applied file passes silently, and a fresh install then differs from production. Backup and restore is documented, but forward-only is stated nowhere. Fix: store a sha256 in `schema_migrations` and warn on mismatch, plus one sentence in `docs/usage/` that migrations are forward-only.

### Lead verification

| Lead | Result | Evidence |
| --- | --- | --- |
| L1 siblings | Refuted for my lens | Hard-coded 2026 dates in `store/*_test.go` appear only as stored values (`roadmap_validation_test.go:43,66`, never compared to `now()`). `upgrade-seed.sql` has no dates. The other fixed-date tests (`agentcontext`, `roadmaptime`, `auth_cooldown`, `presence`) inject a clock. No new sibling. |
| L2 | Extended | CI never runs the DB-gated suite on a schedule, so nothing would catch a retention or migration regression on a cron either. |
| L3 | Confirmed | Full migration set applied cleanly on a fresh Postgres 18.4 cluster. |
| L15 | Confirmed | `docker-compose.yml` binds `127.0.0.1` and uses dev defaults. |
| L16 | Not a data issue | Not examined. |
| L17 | Not run | Out of lens. |
| L18 | Confirmed, then extended | No test or check pins schema confinement. A scan of every migration found zero `public`, extension or database-level DDL. Ran as app: DDL outside `insideout` and any `CREATE` inside it are denied. The one gap is `ALTER ROLE ... SET` on itself (DATA-09). |

### Recommendations

1. Replace `workspace_memberships_insert` so a non-creator can only insert `role='member'` with a valid join code (DATA-01), and add the red test first.
2. Add a write-side RLS walk beside `TestRLSWalk` that attempts the bare INSERT, UPDATE and DELETE statements cross-tenant per table (DATA-02).
3. Lock the `users` row columns with column grants or a trigger, and stop co-members reading `password_hash` (DATA-03).
4. Make `serve` refuse to start on a BYPASSRLS or superuser runtime role (DATA-04).
5. Add the missing FK indexes in one migration (DATA-07).
6. Decide the erasure and retention policy, record it as an issue, then implement the purge for conversations (DATA-06).
7. Change the update-timeline cursor to carry `(created_at, id)` (DATA-05).
8. Store migration checksums and drop the dead GRANT lines (DATA-09, DATA-11).

### What this role would do first

I would write the membership self-admin test, because that statement is the single shortest path from "a caller who knows a UUID" to "admin of someone else's workspace" and nothing red-lines it today. It fails on main right now. Fixing it needs one small migration with a DEFINER join helper, and the same test file then becomes the home for the other write-side checks. Everything else here is hardening of a design that is otherwise sound: the schema stays inside `insideout`, the role split holds, and the helper functions are clean.

The one schema or policy invariant to turn into an automated test first: **a session with `app.user_id` set to a user who is not the creator of a workspace and has not set `app.join_code` to its code cannot INSERT any `workspace_memberships` row for that workspace, and can never insert one with `role='admin'`.**
