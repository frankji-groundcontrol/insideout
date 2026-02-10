# JuanLeMe Phase 2 Backend (Supabase-First)

## TL;DR

> **Quick Summary**: Implement Phase 2 backend in Supabase using schema `juanleme` (9 tables), strict least-privilege RLS with SECURITY DEFINER helpers, publishable-key-only client access, dual-client Edge Functions for AI/export boundaries, and bundled adapter cutover.
>
> **Deliverables**:
> - Supabase schema `juanleme` with 9 tables + audit fields + migrations + RLS policies
> - Supabase adapters wired into existing service registry contracts
> - Store-to-registry wiring fix (pre-cutover parity)
> - Edge Functions (`ai-generate`, `export-document`) with dual-client pattern + idempotency
> - Export storage under `workshops` bucket / `juanleme/` folder with workspace-scoped paths
> - Bundled adapter cutover with rollback safety
> - End-to-end verification gates with strict TDD per slice
>
> **Estimated Effort**: Large
> **Parallel Execution**: YES (4 waves)
> **Critical Path**: Schema bootstrap -> Domain tables -> RLS -> Adapters -> Store parity -> Bundled cutover -> Final gate
> **Total Tasks**: 10

---

## ⚠️ Execution Constraints (NON-NEGOTIABLE)

> **READ THIS BEFORE EVERY TASK. These apply to ALL tasks without exception.**

### 1. Use Supabase MCP Tools — ALWAYS
All database operations MUST go through the Supabase MCP tools available in the environment:
- **DDL / Migrations**: `SupabaseApplyMigration_tool` (name + query params)
- **DML / Queries / Verification**: `SupabaseExecuteSql_tool` (query param)
- **Security Audit**: `SupabaseGetAdvisors_tool` (type="security" or "performance")
- **Schema Inspection**: `SupabaseListTables_tool` (schemas=["juanleme"])
- **Edge Functions**: `SupabaseDeployEdgeFunction_tool`, `SupabaseGetEdgeFunction_tool`, `SupabaseListEdgeFunctions_tool`
- **Type Generation**: `SupabaseGenerateTypescriptTypes_tool`
- **Project Config**: `SupabaseGetProjectUrl_tool`, `SupabaseGetPublishableKeys_tool`

**FORBIDDEN**: Raw psql, direct DB connections, or any non-MCP database access.

### 4. Helper Views & RPCs — Use When RLS Gets Complex
- When RLS on raw tables creates friction (cross-table joins, computed fields, multi-step mutations), use **helper views** and **RPC functions** instead of fighting RLS
- **Views** (`juanleme.v_*`): Use `SECURITY INVOKER` so underlying RLS still applies. Ideal for complex reads (e.g., roadmap nodes with computed per-user status)
- **RPCs** (`juanleme.*`): Use `SECURITY DEFINER` with explicit `SET search_path`. Ideal for multi-step writes (e.g., join workshop, complete node). Called via `supabase.rpc()`
- **Direct table access**: Only for simple single-table CRUD where RLS is straightforward
- This is standard Supabase serverless practice — no traditional backend needed

### 2. Schema `juanleme` — ALWAYS
- Every table, function, policy, and type MUST live under the `juanleme` schema
- Every SQL statement MUST use fully-qualified names: `juanleme.profiles`, `juanleme.workspaces`, etc.
- **NEVER** create app tables in `public` schema
- **NEVER** write unqualified table names in migrations (e.g., `CREATE TABLE profiles` ← WRONG)
- Correct: `CREATE TABLE juanleme.profiles (...)`
- The very first migration MUST include `CREATE SCHEMA IF NOT EXISTS juanleme;`

### 3. Publishable Key Only — NEVER Service Role
- Client-side Supabase client MUST use the publishable (anon) key only
- Use `SupabaseGetPublishableKeys_tool` to retrieve the correct key
- **NEVER** use service-role key in any app runtime code, adapter, or client config
- Edge Functions may use `Deno.env.get('SUPABASE_SERVICE_ROLE_KEY')` internally for admin operations, but this key MUST NEVER appear in frontend code or be sent to the client

---

## Context

### Original Request
User asked whether backend had started and requested backend brainstorming/plan.

### Interview Summary
**Key Decisions**:
- Backend direction: Supabase-first
- Tenant model: workspace-centric
- Priorities: auth/session, core persistence, AI backend path, admin/roles, export persistence
- Permissions: strict least-privilege
- Test strategy: strict TDD
- AI provider contract: OpenAI-compatible interface
- Export retention: 30 days

**Hard Constraints**:
- Use schema `juanleme` (do not use `public` for app tables)
- Use publishable key for client-side integration
- Never use service-role key in app runtime flows

### Metis Review (Applied)
**Guardrails incorporated**:
- Deny-by-default RLS and table-level policy matrix by role
- Explicit schema qualification (`juanleme.*`) in migrations and SQL
- Freeze MVP scope (no billing/realtime/analytics/multi-region)
- Edge Functions validate membership from JWT-backed DB checks (not client claims)
- Per-slice executable acceptance gates (including negative security tests)

### Oracle Review (Applied)
**Structural gaps identified and resolved**:
1. **Missing roadmap table**: Added `juanleme.workshop_nodes` for roadmap structure (node order, title, description) — without this, `getRoadmap()` and node-level progress won't map cleanly
2. **Missing key constraints**: Added unique `(workspace_id, code)` for join codes, unique `(workspace_id, user_id)` for memberships, indexes on all RLS predicate columns
3. **RLS search_path footgun**: Helper functions must be `SECURITY DEFINER STABLE` with explicit `SET search_path = pg_catalog, juanleme`; must `GRANT USAGE ON SCHEMA juanleme` to `authenticated`
4. **Edge Function dual-client**: Use JWT-scoped client for authz checks + optional service-role client only for privileged side effects; always derive workspace lineage from resource IDs in DB
5. **Storage security gap**: Task 7 must include workspace-scoped paths under existing `workshops` bucket / `juanleme/` folder, signed URL issuance for exports
6. **Store/service disconnect**: `app/src/stores/user.ts` bypasses service registry (implements mock auth directly) — must be fixed before cutover or adapter switch won't affect real login
7. **Cutover bundling**: Gate by bundles (auth+profile, then workshop+editor, then ai/export) with contract tests in both modes; current `registry.ts` only supports global `mock` toggle
8. **Audit fields**: Add `created_at`/`updated_at` consistently across all tables
9. **Idempotency keys**: Add to Edge Function requests to prevent duplicate `ai_runs`/`export_jobs` on retries

---

## Work Objectives

### Core Objective
Replace mock backend behavior with a secure Supabase backend while preserving current frontend contracts and user-visible behavior.

### Concrete Deliverables
- Supabase migrations creating `juanleme` schema with 9 MVP tables (incl. `workshop_nodes`) + audit fields
- RLS policies with `SECURITY DEFINER` helpers for `admin`, `member`, and owner-based actions
- Supabase adapters for auth/workshop/editor/ai/export service interfaces
- Store-to-registry wiring fix (all stores use service registry, no direct mock imports)
- `ai-generate` and `export-document` Edge Functions with dual-client pattern + idempotency
- Export artifacts in `workshops` bucket under `juanleme/exports/{workspace_id}/` path
- Bundled adapter cutover with validation
- Contract parity and security verification suite

### Definition of Done
- [ ] All required tables exist under `juanleme` and none required under `public`
- [ ] RLS enabled and enforced on all app tables
- [ ] Client flows work using publishable key + user JWT only
- [ ] Full app tests/build pass after adapter cutover

### Must Have
- Schema-qualified SQL (`juanleme.*`)
- Strict least-privilege policies
- Slice-by-slice rollout with rollback safety

### Must NOT Have (Guardrails)
- No service-role usage in app runtime
- No phase-2 scope creep (billing/realtime/analytics/admin console)
- No breaking of existing `services/*` interface contracts

---

## Verification Strategy (MANDATORY)

> **UNIVERSAL RULE: ZERO HUMAN INTERVENTION**
> All acceptance criteria must be agent-executable (commands, SQL checks, API calls, Playwright/curl/tmux).

### Test Decision
- **Infrastructure exists**: YES (`vitest`)
- **Automated tests**: Strict TDD (Iron Law: NO PRODUCTION CODE WITHOUT A FAILING TEST FIRST)
- **Frameworks**: `vitest` (frontend contract tests), Supabase SQL verification (`SupabaseExecuteSql_tool`), Edge function HTTP checks (`curl`)

### TDD Pattern per Slice
1. **RED**: Write failing test FIRST — verify it fails for the expected reason (feature missing, not typo)
2. **VERIFY RED**: Capture failing output — proves the test actually tests something
3. **GREEN**: Write minimal code/migration/policy to pass
4. **VERIFY GREEN**: All tests pass, output pristine
5. **REFACTOR**: Clean up, keep tests green

### TDD Strategy by Task Type (Oracle-Guided)

> SQL migrations, RLS policies, views, RPCs, and Edge Functions are ALL production code. TDD applies to all of them.

| Task Type | RED Step | GREEN Step | Test Tool |
|-----------|----------|------------|-----------|
| **Schema/Tables (Tasks 1-2)** | SQL assertion: assert table/schema EXISTS → fails pre-migration | Apply migration via `SupabaseApplyMigration_tool` | `SupabaseExecuteSql_tool` |
| **RLS/Views/RPCs (Task 3)** | SQL behavior test: SET LOCAL ROLE + JWT claims, assert cross-workspace denied → fails pre-policy | Apply RLS migration | `SupabaseExecuteSql_tool` |
| **Adapters (Tasks 4-5)** | Vitest: mock Supabase client, assert contract shape → fails pre-implementation | Implement adapter TypeScript | `vitest` (unit) + real backend (integration) |
| **Edge Functions (Tasks 6-7)** | Layer 1: vitest unit test on pure handler logic (authz, idempotency) → fails. Layer 2: curl HTTP test → fails post-deploy | Layer 1: implement handler. Layer 2: deploy via `SupabaseDeployEdgeFunction_tool` | `vitest` (unit) + `curl` (integration) |
| **Store parity (Task 8)** | Vitest: assert stores call registry services → fails (currently bypasses) | Refactor store to use registry | `vitest` |
| **Cutover (Task 9)** | Vitest: assert bundle validation rejects mixed state → fails | Implement bundle registry | `vitest` |
| **Final gate (Task 10)** | Playwright E2E + security matrix | Full stack verification | `playwright` + `SupabaseExecuteSql_tool` + `curl` |

### Test Layers

| Layer | Scope | Tool | When |
|-------|-------|------|------|
| **SQL assertions** | Schema exists, FK integrity, RLS enforced, views/RPCs return correct data | `SupabaseExecuteSql_tool` | Tasks 1-3, 10 |
| **Vitest unit** | Adapter contract parity, store behavior, Edge Function handler logic | `vitest` | Tasks 4-9 |
| **Real-backend integration** | Adapter → live Supabase (critical flows: joinWorkshop, getRoadmap, completeNode, profile denial) | `vitest` + real Supabase | Tasks 4-5, 10 |
| **HTTP contract** | Edge Function auth/response shape/DB side effects | `curl` | Tasks 6-7 |
| **Playwright E2E** | Full user flow with real backend | `playwright` | Task 10 only |

### Oracle Consultation Triggers During Execution

> Consult Oracle at these specific points (not ceremony — real value):

| After Task | Consult Oracle For | Why |
|------------|-------------------|-----|
| Task 2 | Schema/index/constraint sanity review | Catch structural issues before RLS builds on top |
| Task 3 | RLS/view/RPC threat model check | Security-critical — one missed policy = data leak |
| Before Task 6 deploy | Dual-client + lineage validation review | Edge Function security boundary review |
| Before Task 7 deploy | Storage policy + path parsing review | Shared bucket isolation verification |
| Before Task 9 | Bundle-mode invariants review | Cutover correctness before switching live adapters |

> Skip Oracle on: straightforward store refactors (Task 8), simple adapter CRUD, unless a test exposes contract ambiguity.

### Mandatory Security Assertions
- RLS enabled on all `juanleme` tables
- Cross-workspace access denied
- Unauthenticated function requests rejected
- Export retention logic enforces 30-day policy
- Storage path isolation verified for `workshops/juanleme/` prefix

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation, sequential):
  Task 1 -> Task 2 -> Task 3

Wave 2 (Domain build, partial parallel after Task 3):
  Task 4 and Task 5 (parallel)

Wave 2.5 (Boundaries + store fix, parallel after Tasks 4,5):
  Task 6, Task 7, Task 8 (parallel where dependencies allow)

Wave 3 (Cutover, sequential):
  Task 9 -> Task 10

Critical Path: 1 -> 2 -> 3 -> 4/5 -> 8 -> 9 -> 10
```

### Dependency Matrix

| Task | Description | Depends On | Blocks | Can Parallelize With |
|------|-------------|------------|--------|----------------------|
| 1 | Schema bootstrap | None | 2,3,4,5 | None |
| 2 | Domain tables | 1 | 3,4,5,6,7,8 | None |
| 3 | RLS policies | 2 | 4,5,6,7,8 | None |
| 4 | Auth adapters | 3 | 8,9 | 5 |
| 5 | Workshop adapters | 3 | 8,9 | 4 |
| 6 | AI Edge Function | 4,5 | 9,10 | 7, 8 |
| 7 | Export Edge Function | 5 | 9,10 | 6, 8 |
| 8 | Store parity fix | 4,5 | 9,10 | 6, 7 |
| 9 | Registry cutover | 6,7,8 | 10 | None |
| 10 | Final gate | 9 | None | None |

---

## TODOs

- [ ] 1. Supabase Bootstrap and Schema Guardrails

  **What to do**:
  - Use `SupabaseApplyMigration_tool` to create initial migration establishing schema `juanleme`
  - Use `SupabaseGetPublishableKeys_tool` to retrieve publishable key for client config
  - Use `SupabaseGetProjectUrl_tool` to retrieve project URL for client config
  - Add base extensions/index helpers only if needed
  - Ensure all migration DDL references `juanleme.*` (fully-qualified)
  - Add test that fails if app tables appear in `public`
  - Configure Supabase client in app to use publishable key ONLY

  **Must NOT do**:
  - No app-table DDL in `public` — everything under `juanleme.*`
  - No service-role key in frontend runtime or env config
  - No raw psql — use Supabase MCP tools exclusively

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`git-master`]

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 1
  - **Blocks**: 2,3,4,5
  - **Blocked By**: None

  **References**:
  - `app/src/services/registry.ts` — current adapter switching point
  - `app/src/types/services.ts` — contract surface that must remain stable
  - `docs/TODO.md` — phase tracking expectations

  **Acceptance Criteria**:
  - [ ] `SupabaseApplyMigration_tool` creates `juanleme` schema successfully
  - [ ] `SupabaseExecuteSql_tool` confirms schema exists and no app tables in `public`
  - [ ] `SupabaseGetPublishableKeys_tool` returns key; key is configured in app client
  - [ ] `SupabaseListTables_tool(schemas=["juanleme"])` shows schema exists
  - [ ] RED->GREEN test artifact exists for schema isolation check

  **Agent-Executed QA Scenarios**:
  ```text
  Scenario: Schema isolation baseline
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. SELECT table_name FROM information_schema.tables WHERE table_schema = 'juanleme'
      2. SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('profiles','workspaces','workspace_memberships','projects','workshop_nodes','documents','document_revisions','ai_runs','export_jobs')
      3. Assert first query returns expected tables, second returns zero rows
    Expected Result: isolation check passes

  Scenario: Negative guard — public schema pollution
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('profiles','workspaces','workspace_memberships','projects','workshop_nodes','documents','document_revisions','ai_runs','export_jobs')
      2. Assert count = 0
    Expected Result: no public leakage

  Scenario: Publishable key retrieval
    Tool: SupabaseGetPublishableKeys_tool
    Steps:
      1. Call tool, retrieve key where disabled is false
      2. Verify key format starts with expected prefix
    Expected Result: valid publishable key available
  ```

---

- [ ] 2. Core Domain Tables (Workspace-Centric)

  **What to do**:
  - Use `SupabaseApplyMigration_tool` to add tables under `juanleme`: `juanleme.profiles`, `juanleme.workspaces`, `juanleme.workspace_memberships`, `juanleme.projects`, `juanleme.workshop_nodes`, `juanleme.documents`, `juanleme.document_revisions`, `juanleme.ai_runs`, `juanleme.export_jobs`
  - All table names MUST be fully-qualified with `juanleme.` prefix
  - **`juanleme.workshop_nodes`** (Oracle-required): roadmap node definition table mapping to frontend `RoadmapNode` type (`app/src/types/index.ts:25`):
    - `id UUID PK`, `workspace_id UUID FK → juanleme.workspaces(id)`, `title TEXT`, `description TEXT`, `order INT`
    - Note: frontend type uses `workshop_id` as the field name (`RoadmapNode.workshop_id`), but the FK targets `juanleme.workspaces` table. The DB column should be `workspace_id` to match the FK target table name. The Supabase adapter maps `workspace_id` ↔ frontend `workshop_id` at the adapter layer.
    - `UNIQUE (workspace_id, "order")` — no duplicate ordering within a workspace/workshop
    - This table stores **node definitions only** (title, description, order). Per-user progress (`status`, `content`) is derived from `juanleme.documents`/`juanleme.document_revisions` linked to user+node
    - Frontend `RoadmapNode.status` ('locked'|'pending'|'in_progress'|'completed') is computed at query time from document state, not stored on the node definition
    - No `node_type` column needed — frontend types don't use it
  - Add PK/FK/indexes and ownership columns (`workspace_id`, `created_by`)
  - Add status columns for async jobs (`pending/running/succeeded/failed`)
  - Add `created_at TIMESTAMPTZ DEFAULT now()` and `updated_at TIMESTAMPTZ DEFAULT now()` on ALL tables (Oracle-recommended audit fields)
  - Add `updated_at` trigger function in `juanleme` schema to auto-set on UPDATE
  - **Key constraints (Oracle-required)**:
    - UNIQUE `(code)` on `juanleme.workspaces` for join codes (globally unique 6-digit invite codes per `Workshop.code` in `app/src/types/index.ts:17`)
    - UNIQUE `(workspace_id, user_id)` on `juanleme.workspace_memberships` to prevent duplicate memberships
    - UNIQUE `(workspace_id, "order")` on `juanleme.workshop_nodes` to prevent duplicate ordering within a workspace
    - Indexes on ALL RLS predicate columns: `workspace_id`, `created_by`, membership lookup fields (`user_id`, `project_id`, `workshop_id`)
  - Verify with `SupabaseListTables_tool(schemas=["juanleme"])`

  **Must NOT do**:
  - No polymorphic over-modeling
  - No non-MVP domains
  - No unqualified table names — always `juanleme.*`
  - No raw psql — use Supabase MCP tools exclusively

  **Recommended Agent Profile**:
  - **Category**: `ultrabrain`
  - **Skills**: [`git-master`]

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 1
  - **Blocks**: 3,4,5,6,7,8
  - **Blocked By**: 1

  **References**:
  - `app/src/types/index.ts` — existing domain shape from frontend
  - `app/src/services/mock/data.ts` — current data semantics to preserve

  **Acceptance Criteria**:
  - [ ] `SupabaseApplyMigration_tool` applies migration successfully
  - [ ] `SupabaseListTables_tool(schemas=["juanleme"])` shows all 9 tables (profiles, workspaces, workspace_memberships, projects, workshop_nodes, documents, document_revisions, ai_runs, export_jobs)
  - [ ] `SupabaseExecuteSql_tool` validates FK integrity and required non-null constraints
  - [ ] Unique constraints verified: `(code)` on workspaces, `(workspace_id, user_id)` on memberships, `(workspace_id, "order")` on workshop_nodes
  - [ ] All tables have `created_at` and `updated_at` columns
  - [ ] `updated_at` trigger fires on UPDATE
  - [ ] RED->GREEN test artifacts exist

  **Agent-Executed QA Scenarios**:
  ```text
  Scenario: Relational integrity (full chain)
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. INSERT test profile into juanleme.profiles
      2. INSERT test workspace into juanleme.workspaces
      3. INSERT membership into juanleme.workspace_memberships linking both
      4. INSERT project into juanleme.projects under that workspace
      5. INSERT workshop_node into juanleme.workshop_nodes under that workspace (workspace_id FK)
      6. INSERT document into juanleme.documents linked to workshop_node
      7. SELECT with JOIN across full chain — assert all rows linked correctly
    Expected Result: chain integrity passes

  Scenario: Negative FK case
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. INSERT INTO juanleme.projects with non-existent workspace_id
      2. Assert error contains 'foreign key' or 'violates'
    Expected Result: invalid write rejected

  Scenario: Unique constraint enforcement
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. INSERT duplicate membership (same workspace_id + user_id)
      2. Assert unique violation error
      3. INSERT workspace with duplicate code (same code as existing workspace)
      4. Assert unique violation error
      5. INSERT duplicate workshop_node (same workspace_id + order)
      6. Assert unique violation error
    Expected Result: all duplicates rejected

  Scenario: Audit fields auto-populate
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. INSERT row into juanleme.profiles
      2. Assert created_at IS NOT NULL and updated_at IS NOT NULL
      3. UPDATE same row
      4. Assert updated_at > created_at
    Expected Result: audit timestamps work
  ```

---

- [ ] 3. RLS Role Helpers, Views, RPCs, and Policy Matrix

  **What to do**:
  - Use `SupabaseApplyMigration_tool` to enable RLS on all `juanleme` app tables (all 9 tables)
  - **Schema exposure (Oracle-critical)**: `GRANT USAGE ON SCHEMA juanleme TO authenticated, anon;` and grant appropriate table-level privileges to `authenticated` role. Ensure `juanleme` is added to Supabase exposed schemas (via Supabase dashboard or migration)
  - Implement helper function(s) under `juanleme` schema for effective role checks:
    - Functions MUST be `SECURITY DEFINER STABLE` with explicit `SET search_path = pg_catalog, juanleme`
    - Example: `juanleme.get_user_workspace_role(p_workspace_id UUID) RETURNS TEXT` — checks `juanleme.workspace_memberships` for `auth.uid()`
  - Add policy matrix for `admin`, `member`, and owner-based writes
  - Encode deny-by-default semantics
  - All function/policy SQL must use `juanleme.*` fully-qualified names

  - **Helper Views & RPC Strategy (user-specified — use when RLS gets complex)**:
    > When direct RLS on raw tables becomes awkward (cross-table joins, computed fields, complex authorization), prefer **helper views** and **RPC functions** over fighting RLS policies. This is standard Supabase practice.

    **Helper Views** — for complex reads that span multiple tables:
    - `juanleme.v_roadmap_with_status` — joins `workshop_nodes` + `documents` to compute per-user `RoadmapNode.status` ('locked'|'pending'|'in_progress'|'completed') at query time. The frontend `RoadmapNode` type expects `status` but it's derived, not stored on the node. This view solves that cleanly.
    - `juanleme.v_workshop_summary` — joins `workspaces` + `workspace_memberships` to return workshop list with `member_count` and `is_joined` for the current user (matches `Workshop` type fields)
    - Views should use `SECURITY INVOKER` (default in Supabase) so RLS on underlying tables still applies
    - Additional helper views as needed when RLS on raw tables creates friction

    **RPC Functions** — for complex writes, multi-step operations, or authorization-sensitive mutations:
    - `juanleme.join_workshop(p_code TEXT)` — validates invite code, checks membership doesn't exist, inserts membership. Called via `supabase.rpc('join_workshop', { code })`
    - `juanleme.complete_node(p_node_id UUID, p_content JSONB)` — validates user is member of the workshop, creates/updates document + revision, returns updated status. Single RPC call instead of multiple client-side queries
    - `juanleme.get_workshop_roadmap(p_workshop_id UUID)` — returns roadmap nodes with computed user status in one call
    - RPCs MUST be `SECURITY DEFINER` with explicit `SET search_path = pg_catalog, juanleme` and validate `auth.uid()` internally
    - Use RPCs whenever a client operation would need multiple queries that are hard to secure individually via RLS

    **Decision guide for implementer**:
    | Situation | Use |
    |-----------|-----|
    | Simple CRUD on single table | Direct table access + RLS policies |
    | Read spanning multiple tables with computed fields | Helper view (`juanleme.v_*`) |
    | Write needing multi-step logic or cross-table validation | RPC function (`juanleme.*_rpc` or descriptive name) |
    | RLS policy becoming complex/fragile | Refactor to view or RPC |

  **Must NOT do**:
  - No broad `USING (true)` shortcuts
  - No client-claim trust without DB membership check
  - No policies referencing `public` schema tables
  - No helper functions without explicit `SET search_path` (security footgun)
  - No `SECURITY DEFINER` views — use `SECURITY INVOKER` for views (let underlying RLS apply)
  - No raw psql — use Supabase MCP tools exclusively

  **Recommended Agent Profile**:
  - **Category**: `ultrabrain`
  - **Skills**: [`git-master`]

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 1
  - **Blocks**: 4,5,6,7,8
  - **Blocked By**: 2

  **References**:
  - `app/src/router/index.ts` — current auth-protected route assumptions
  - `app/src/stores/user.ts` — auth/session state behavior to preserve

  **Acceptance Criteria**:
  - [ ] `SupabaseApplyMigration_tool` applies RLS migration successfully
  - [ ] `SupabaseExecuteSql_tool`: RLS enabled on all 9 tables (`relrowsecurity = true`)
  - [ ] `GRANT USAGE ON SCHEMA juanleme TO authenticated` applied successfully
  - [ ] Helper functions created with `SECURITY DEFINER STABLE` and explicit `SET search_path`
  - [ ] Helper views created with `SECURITY INVOKER` — `v_roadmap_with_status`, `v_workshop_summary` queryable
  - [ ] RPC functions created and callable via `supabase.rpc()` — `join_workshop`, `complete_node`, `get_workshop_roadmap`
  - [ ] Policy tests for allow/deny per role/table pass via `SupabaseExecuteSql_tool`
  - [ ] Cross-workspace unauthorized read/write tests fail
  - [ ] RPC functions reject unauthorized callers (non-member trying to complete_node in another workshop)
  - [ ] `SupabaseGetAdvisors_tool(type="security")` shows no critical RLS gaps

  **Agent-Executed QA Scenarios**:
  ```text
  Scenario: Member isolation
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. SET LOCAL role to authenticated; set jwt claim to user in workspace A
      2. SELECT * FROM juanleme.documents WHERE project_id belongs to workspace B
      3. Assert zero rows returned
      4. Attempt UPDATE on juanleme.documents row owned by another user
      5. Assert zero rows affected
    Expected Result: no access / operation denied

  Scenario: Admin permissions
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. SET LOCAL role to authenticated; set jwt claim to admin of workspace A
      2. UPDATE juanleme.workspace_memberships to change a member's role
      3. Assert 1 row affected
    Expected Result: admin governance works

  Scenario: RLS enabled verification
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. SELECT c.relname, c.relrowsecurity FROM pg_class c JOIN pg_namespace n ON c.relnamespace = n.oid WHERE n.nspname = 'juanleme' AND c.relkind = 'r'
      2. Assert ALL rows have relrowsecurity = true
    Expected Result: no table without RLS

  Scenario: Helper view — roadmap with computed status
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. Seed workshop_nodes + documents for a user (one completed, one in_progress, one pending, one locked)
      2. SET LOCAL role to authenticated; set jwt claim to that user
      3. SELECT * FROM juanleme.v_roadmap_with_status WHERE workshop_id = '{test_workshop_id}'
      4. Assert returned rows include computed status matching expected per-node state
      5. Query as different user — assert their status reflects their own document state (not first user's)
    Expected Result: view returns per-user computed status correctly

  Scenario: RPC — join_workshop
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. SET LOCAL role to authenticated; set jwt claim to user not yet in workshop
      2. SELECT juanleme.join_workshop('888888') — valid invite code
      3. Assert membership row created in juanleme.workspace_memberships
      4. Call again with same code — assert error (duplicate membership)
      5. Call with invalid code 'XXXXXX' — assert error (code not found)
    Expected Result: join flow works, idempotent rejection, invalid code rejected

  Scenario: RPC — complete_node unauthorized
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. SET LOCAL role to authenticated; set jwt claim to user NOT in workshop
      2. SELECT juanleme.complete_node('{node_id}', '{"content": "test"}')
      3. Assert error (not a member)
    Expected Result: non-members cannot complete nodes
  ```

---

- [ ] 4. Auth + Profile + Membership Adapters

  **What to do**:
  - Add Supabase auth/profile adapter implementing existing `IAuthService`
  - Add membership bootstrap logic for workspace onboarding
  - Keep mock mode intact behind config switch

  **Must NOT do**:
  - No contract-breaking return shape changes
  - No service-role key usage

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`git-master`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with 5)
  - **Blocks**: 8,9
  - **Blocked By**: 3

  **References**:
  - `app/src/types/services.ts:IAuthService`
  - `app/src/services/mock/authService.ts`
  - `app/src/stores/user.ts`

  **Acceptance Criteria**:
  - [ ] Contract tests pass for login/getCurrentUser/updateProfile/logout parity
  - [ ] Session hydration works on page reload with publishable-key flow

  **Agent-Executed QA Scenarios**:
  ```text
  Scenario: Auth adapter parity
    Tool: bash (vitest)
    Steps:
      1. Run adapter contract tests
      2. Assert outputs match existing service contract
    Expected Result: parity maintained

  Scenario: Unauthorized profile update
    Tool: curl / supabase_execute_sql
    Steps:
      1. Attempt update for another user profile
      2. Assert denied by RLS
    Expected Result: rejected
  ```

---

- [ ] 5. Workshop/Document Persistence Adapters

  **What to do**:
  - Implement Supabase adapters for workshop/roadmap/submission persistence
  - Preserve existing interfaces (`IWorkshopService`, `IEditorService` where applicable)
  - **Use helper views and RPCs from Task 3 as primary data access layer**:
    - `getWorkshops()` → query `juanleme.v_workshop_summary` (returns member_count + is_joined computed)
    - `getRoadmap()` → call `supabase.rpc('get_workshop_roadmap', { workshop_id })` (returns nodes with per-user computed status)
    - `joinWorkshop()` → call `supabase.rpc('join_workshop', { code })`
    - `completeNode()` / `saveSubmission()` → call `supabase.rpc('complete_node', { node_id, content })`
    - Simple CRUD (get single workshop, get document) → direct table queries with RLS
  - Add revision writes on meaningful content updates (handled inside `complete_node` RPC)

  **Must NOT do**:
  - No UI-level API shape changes
  - No eager schema redesign during adapter work
  - No fighting RLS on complex cross-table queries — use views/RPCs instead

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`git-master`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with 4)
  - **Blocks**: 6,7,8,9
  - **Blocked By**: 3

  **References**:
  - `app/src/types/services.ts:IWorkshopService` — contract to implement
  - `app/src/types/index.ts:RoadmapNode` — status is computed, not stored (line 30)
  - `app/src/types/index.ts:Workshop` — `member_count` + `is_joined` are computed (lines 21-22)
  - `app/src/stores/workshop.ts` — consumer of workshop service
  - `app/src/stores/editor.ts` — consumer of editor service
  - `app/src/services/mock/workshopService.ts` — mock behavior to match
  - DB views/RPCs from Task 3: `juanleme.v_workshop_summary`, `juanleme.v_roadmap_with_status`, `juanleme.get_workshop_roadmap()`, `juanleme.join_workshop()`, `juanleme.complete_node()`

  **Acceptance Criteria**:
  - [ ] `getWorkshops()` returns data from `v_workshop_summary` with correct `member_count` and `is_joined`
  - [ ] `getRoadmap()` returns nodes with per-user computed `status` via RPC
  - [ ] `completeNode()` persists content + creates revision via RPC
  - [ ] `joinWorkshop()` creates membership via RPC, rejects invalid codes
  - [ ] Roadmap status transitions persist correctly across page reloads
  - [ ] Draft/submission retrieval matches previous mock behavior
  - [ ] Contract parity tests pass

  **Agent-Executed QA Scenarios**:
  ```text
  Scenario: Persist and restore workshop flow via views/RPCs
    Tool: bash (vitest + SupabaseExecuteSql_tool)
    Steps:
      1. Call complete_node RPC to save submission
      2. Query v_roadmap_with_status for the workshop
      3. Assert node status changed to 'completed' and content persisted
      4. Query juanleme.document_revisions — assert revision row created
    Expected Result: persistent behavior matches mock contract, status computed correctly

  Scenario: Workshop list with computed fields
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. Query juanleme.v_workshop_summary as authenticated user
      2. Assert member_count reflects actual membership count
      3. Assert is_joined is true for workshops user has joined, false otherwise
    Expected Result: computed fields match Workshop type contract

  Scenario: Cross-workspace data access denial
    Tool: SupabaseExecuteSql_tool
    Steps:
      1. Query another workspace's document as non-member
      2. Assert zero rows/permission denied
      3. Call complete_node RPC for a node in a workshop user is not a member of
      4. Assert error (unauthorized)
    Expected Result: isolation enforced at both table and RPC level
  ```

---

- [ ] 6. AI Edge Function (`ai-generate`) + `ai_runs`

  **What to do**:
  - Use `SupabaseDeployEdgeFunction_tool` to deploy `ai-generate` Edge Function
  - Implement AI generation via OpenAI-compatible contract
  - **Dual-client pattern (Oracle-required)**:
    - JWT-scoped Supabase client for authorization checks (verify membership via `juanleme.workspace_memberships`)
    - Service-role client ONLY for privileged side effects (writing `juanleme.ai_runs` lifecycle records)
  - **Lineage validation (Oracle-required)**: Always derive workspace from resource IDs (node/document/project) via DB joins — never trust client-supplied workspace ID or role claims alone
  - Track lifecycle in `juanleme.ai_runs` (fully-qualified)
  - Add timeout/retry-safe behavior and failure statuses
  - **Idempotency keys (Oracle-recommended)**: Accept client-supplied idempotency key in request to prevent duplicate `ai_runs` on retries

  **Must NOT do**:
  - No provider keys exposed client-side
  - No function trust in client-supplied workspace role claims
  - No service-role key in frontend code — only inside Edge Function runtime
  - No workspace lineage from client claims alone — always verify via DB

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`git-master`]

  **Parallelization**:
  - **Can Run In Parallel**: LIMITED
  - **Parallel Group**: Wave 2/3 bridge
  - **Blocks**: 8,9
  - **Blocked By**: 4,5

  **References**:
  - `app/src/types/services.ts:IAiService`
  - `app/src/components/workshop/AiSidebar.vue`
  - `app/src/features/workshop/ai/composables/useAiConversation.ts`

  **Acceptance Criteria**:
  - [ ] Unauthorized requests return 401/403
  - [ ] Authorized request writes `ai_runs` record and returns expected shape
  - [ ] Failure path writes failed status with diagnostic message

  **Agent-Executed QA Scenarios**:
  ```text
  Scenario: Authorized AI request
    Tool: curl
    Steps:
      1. Call function with valid JWT + workspace context
      2. Assert 200 and response payload shape
      3. Query ai_runs row exists with succeeded/failed terminal status
    Expected Result: lifecycle persisted

  Scenario: Unauthorized AI request
    Tool: curl
    Steps:
      1. Call function without JWT
      2. Assert 401/403
    Expected Result: request blocked
  ```

---

- [ ] 7. Export Persistence + Edge Function (`export-document`)

  **What to do**:
  - Use `SupabaseDeployEdgeFunction_tool` to deploy `export-document` Edge Function
  - Add `juanleme.export_jobs` workflow with status transitions (fully-qualified)
  - **Dual-client pattern (Oracle-required)**: JWT-scoped client for authz, service-role for privileged writes
  - **Lineage validation**: Derive workspace from document/project chain in DB, not client claims
  - **Idempotency keys (Oracle-recommended)**: Accept idempotency key to prevent duplicate `export_jobs`
  - **Storage (user-specified)**:
    - Use existing `workshops` bucket — do NOT create a new bucket
    - All export artifacts stored under `juanleme/` folder prefix within `workshops` bucket
    - Path scoping by workspace: `juanleme/exports/{workspace_id}/{export_job_id}/...`
    - Signed URL issuance for download (time-limited, workspace-scoped)
    - **Storage RLS policies on `storage.objects`** (Oracle round 2 — folder prefix alone is NOT isolation):
      - Add policy: `SELECT` on `storage.objects` WHERE `bucket_id = 'workshops'` AND `name LIKE 'juanleme/exports/%'` AND user is member of the workspace parsed from path segment
      - Add policy: `INSERT` restricted to Edge Function (service-role) only — clients cannot upload directly
      - Add policy: `DELETE` restricted to cleanup job (service-role) only
      - Membership check uses `juanleme.workspace_memberships` joined on workspace_id extracted from storage path
  - Implement export function producing artifact metadata + storage path
  - Enforce 30-day retention cleanup policy:
    - Cleanup job deletes both `juanleme.export_jobs` records AND storage artifacts older than 30 days

  **Must NOT do**:
  - No indefinite storage default
  - No bypass of workspace access checks
  - No creating new storage buckets — use existing `workshops` bucket with `juanleme/` folder
  - No storing artifacts outside `juanleme/` prefix in the `workshops` bucket
  - No service-role key in frontend code — only inside Edge Function runtime

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`git-master`]

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 6 where dependencies allow)
  - **Parallel Group**: Wave 3
  - **Blocks**: 8,9
  - **Blocked By**: 5

  **References**:
  - `app/src/types/services.ts:IExportService`
  - `app/src/utils/export.ts`
  - `app/src/views/workshop/ExportPreview.vue`

  **Acceptance Criteria**:
  - [ ] Export request produces `export_jobs` row and result metadata
  - [ ] Access denied for non-member requests
  - [ ] Cleanup test verifies >30-day artifacts are removed/invalidated

  **Agent-Executed QA Scenarios**:
  ```text
  Scenario: Export happy path
    Tool: curl + supabase_execute_sql
    Steps:
      1. Trigger export with valid JWT
      2. Assert status transitions to succeeded
      3. Verify artifact metadata exists
    Expected Result: export flow works

  Scenario: Retention enforcement
    Tool: supabase_execute_sql
    Steps:
      1. Seed old export record >30 days
      2. Run cleanup path
      3. Assert old record/artifact unavailable
    Expected Result: retention policy enforced
  ```

---

- [ ] 8. Store Integration Parity (Oracle-Required Pre-Cutover Fix)

  **What to do**:
  - **Critical fix**: `app/src/stores/user.ts` currently implements mock auth directly (bypasses service registry). Refactor it to call `services.auth.*` methods from the registry — without this, adapter cutover won't affect real login/session.
  - Ensure `app/src/stores/workshop.ts` completion/submission flows persist via `services.workshop.*` methods
  - Ensure `app/src/features/workshop/detail/composables/useWorkshopDetailSession.ts` node switching/load/complete flows call service methods (not direct mock data)
  - Verify all stores consume services exclusively through registry — no direct mock imports

  **Must NOT do**:
  - No breaking changes to store public API (components still call same store actions)
  - No service contract changes
  - No UI changes

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`git-master`]

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 6, 7) — **Oracle guard**: only if Task 8 stays limited to registry wiring in stores. If Task 8 needs to change flows that depend on Edge Function response shapes, gate it behind Tasks 6/7.
  - **Parallel Group**: Wave 2/3 bridge
  - **Blocks**: 9, 10
  - **Blocked By**: 4, 5

  **References**:
  - `app/src/stores/user.ts` — currently bypasses registry (Oracle-identified risk)
  - `app/src/stores/workshop.ts` — completion/submission flow
  - `app/src/stores/editor.ts` — content persistence flow
  - `app/src/features/workshop/detail/composables/useWorkshopDetailSession.ts` — orchestration layer
  - `app/src/services/registry.ts` — the switching point all stores must use
  - `app/src/types/services.ts` — contracts stores must call through

  **Acceptance Criteria**:
  - [ ] `app/src/stores/user.ts` no longer imports from `services/mock/*` directly — uses registry only
  - [ ] All store actions route through `services.*` from registry
  - [ ] Existing tests still pass: `pnpm --filter app test -- --run`
  - [ ] Build passes: `pnpm --filter app run build`

  **Agent-Executed QA Scenarios**:
  ```text
  Scenario: Store-service wiring verification
    Tool: bash (grep + vitest)
    Steps:
      1. Grep for direct mock imports in stores/ and features/ composables
      2. Assert zero direct mock service imports (only registry imports)
      3. Run full test suite
      4. Assert all tests pass
    Expected Result: stores fully wired through registry

  Scenario: Mock mode still works after refactor
    Tool: bash (vitest)
    Steps:
      1. Set VITE_API_MODE=mock
      2. Run tests covering login, workshop load, editor save
      3. Assert all pass
    Expected Result: mock behavior preserved through registry indirection
  ```

  **Commit**: YES
  - Message: `refactor(stores): wire all stores through service registry for adapter cutover`
  - Files: `app/src/stores/user.ts`, `app/src/stores/workshop.ts`, `app/src/stores/editor.ts`, possibly composables
  - Pre-commit: `pnpm --filter app test -- --run`

---

- [ ] 9. Service Registry Cutover (Mock -> Supabase)

  **What to do**:
  - Keep existing interfaces untouched
  - **Bundled cutover (Oracle-required)** — NOT per-service arbitrary mixing:
    - Bundle 1: `auth + profile` (switch together)
    - Bundle 2: `workshop + editor` (switch together)
    - Bundle 3: `ai + export` (switch together)
  - Introduce env-based adapter switch per BUNDLE (not per individual service)
  - **Compatibility checks (Oracle-required)**: Add explicit runtime validation that all services within a bundle are using the same adapter mode — reject mixed states
  - Current `app/src/services/registry.ts` supports only global `mock` toggle — extend to support per-bundle switching with validation
  - Keep fallback rollback path to mock mode per bundle

  **Must NOT do**:
  - No big-bang switch (all at once)
  - No arbitrary per-service mixing within a bundle
  - No mixed contract payload shapes

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`git-master`]

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3
  - **Blocks**: 10
  - **Blocked By**: 4,5,6,7,8

  **References**:
  - `app/src/services/registry.ts` — currently only supports global `mock` toggle
  - `app/src/services/mock/*.ts`
  - `app/src/types/services.ts`

  **Acceptance Criteria**:
  - [ ] Bundle-based adapter selection works per environment variable
  - [ ] Mixed state within a bundle is rejected at startup
  - [ ] Contract parity tests pass for all service interfaces in both modes
  - [ ] Rollback to mock mode per bundle works

  **Agent-Executed QA Scenarios**:
  ```text
  Scenario: Bundled adapter parity check
    Tool: bash (vitest)
    Steps:
      1. Run contract tests with all bundles in mock mode
      2. Switch bundle 1 (auth+profile) to supabase mode, keep others mock
      3. Run contract tests — assert auth+profile use supabase, others use mock
      4. Switch all bundles to supabase mode
      5. Run full contract test suite
    Expected Result: equivalent contracts in all configurations

  Scenario: Mixed state rejection
    Tool: bash (vitest)
    Steps:
      1. Configure auth=supabase but profile=mock (within same bundle)
      2. Assert startup throws configuration error
    Expected Result: mixed bundle state rejected
  ```

  **Commit**: YES
  - Message: `refactor(services): enable bundled mock-to-supabase adapter cutover with validation`
  - Files: `app/src/services/registry.ts`, possibly new `app/src/services/supabase/*.ts`
  - Pre-commit: `pnpm --filter app test -- --run`

---

- [ ] 10. Final Security + Integration Gate

  **What to do**:
  - Run full test suite + build
  - Run policy/security matrix checks
  - Run end-to-end flow verification with real backend adapters
  - Run Supabase advisors (security + performance)
  - Update docs with final Phase 2 MVP completion notes

  **Must NOT do**:
  - No merge if any deny-path check fails
  - No skipping advisor review

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`playwright`, `git-master`]

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Final
  - **Blocks**: None
  - **Blocked By**: 9

  **References**:
  - `docs/TODO.md`
  - `app/src/router/index.ts`
  - `app/src/views/*`

  **Acceptance Criteria**:
  - [ ] `pnpm --filter app test -- --run` passes
  - [ ] `pnpm --filter app run build` passes
  - [ ] Security deny-path checks pass
  - [ ] `SupabaseGetAdvisors_tool(type="security")` reviewed and addressed
  - [ ] `SupabaseGetAdvisors_tool(type="performance")` reviewed and addressed

  **Agent-Executed QA Scenarios**:
  ```text
  Scenario: End-to-end with Supabase adapters
    Tool: Playwright
    Steps:
      1. Login via Supabase auth
      2. Open workspace/project/workshop flow
      3. Edit content, run AI suggestion, export
      4. Reload and verify persisted state from juanleme.* tables
    Expected Result: full path works with real backend

  Scenario: Security gate
    Tool: SupabaseExecuteSql_tool + curl
    Steps:
      1. Run deny matrix checks for non-member and cross-workspace cases on all juanleme.* tables
      2. Probe edge functions without JWT
      3. Verify `workshops` bucket `juanleme/exports/` path access denied for non-members
    Expected Result: all unauthorized actions blocked
  ```

---

## Commit Strategy

| After Task | Message | Verification |
|------------|---------|--------------|
| 1 | `chore(db): bootstrap juanleme schema and migration guardrails` | SQL checks + tests |
| 2 | `feat(db): add workspace-centric domain tables with audit fields` | FK + unique + audit tests |
| 3 | `feat(security): add RLS policy matrix with SECURITY DEFINER helpers` | deny/allow + search_path tests |
| 4 | `feat(auth): add supabase auth/profile adapter with membership bootstrap` | contract tests |
| 5 | `feat(data): add supabase persistence adapters for workshop/editor` | contract + integration |
| 6 | `feat(ai): add ai-generate edge function with dual-client and idempotency` | function auth + lifecycle |
| 7 | `feat(export): add export edge function with storage security and retention` | export + storage + retention tests |
| 8 | `refactor(stores): wire all stores through service registry for adapter cutover` | test suite passes |
| 9 | `refactor(services): enable bundled mock-to-supabase adapter cutover` | parity suite + bundle validation |
| 10 | `test(integration): enforce phase2 security and e2e verification gates` | full suite + advisors |

---

## Success Criteria

### Verification Commands
```bash
pnpm --filter app test -- --run
pnpm --filter app run build
```

### Supabase Verification Gates (MCP Tools)
- `SupabaseListTables_tool(schemas=["juanleme"])`: all 9 tables present, zero app tables in public
- `SupabaseExecuteSql_tool`: schema + RLS + policy matrix assertions
- `SupabaseGetAdvisors_tool(type="security")`: no unaddressed critical issues
- `SupabaseGetAdvisors_tool(type="performance")`: no unaddressed critical issues
- `SupabaseGetPublishableKeys_tool`: publishable key configured (never service-role)

### Final Checklist
- [ ] All 9 app tables exist under `juanleme` schema only (including `workshop_nodes`)
- [ ] Publishable-key flow operational; service-role unused in runtime (only in Edge Functions)
- [ ] Strict least-privilege RLS with `SECURITY DEFINER` helpers and explicit `search_path`
- [ ] All stores wired through service registry (no direct mock imports)
- [ ] AI and export Edge Functions use dual-client pattern with DB lineage validation
- [ ] Export artifacts stored in `workshops` bucket under `juanleme/` folder with workspace-scoped paths
- [ ] 30-day retention enforced (both DB records and storage artifacts)
- [ ] Bundled mock-to-supabase cutover is reversible and contract-safe
- [ ] `created_at`/`updated_at` audit fields on all tables
