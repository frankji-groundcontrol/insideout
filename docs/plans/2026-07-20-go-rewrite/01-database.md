# 01 — Database: Rename, Decouple, Extend / 数据库：重命名、解耦、扩展

> Part of the [InsideOut rewrite plan](README.md). / [InsideOut 重写计划](README.md)的一部分。
>
> **Implementation outcome (2026-07-21) / 实现结果**: the shipped migration set diverged from §2/§3 below in three ways, all recorded in the main plan's D1/D2/D4 and implemented as the 13 timestamped files under `server/db/migrations/`: (1) no in-place `juanleme` rename — the `insideout` schema was created fresh, real data was copied over, and `juanleme` was then dropped; (2) the `REVOKE CREATE ON SCHEMA public` lockdown migration was removed — "never write to `public`" is enforced by the migrations never targeting it (required for the shared-instance model, see [BUG-008](../../issues/2026-07-20-bug-008-shared-instance-db-provisioning.md)); (3) §5's "optional later hardening" RLS **was** subsequently implemented (migrations `20260720150000`+ and `withUserContext` in `server/internal/store/pool.go`) — the live database runs with RLS on. Current state: [docs/architecture/database-and-rls.md](../../architecture/database-and-rls.md).
> **实现结果（2026-07-21）**：实际交付的迁移集与下文 §2/§3 有三处出入，均记录于主计划 D1/D2/D4，落地为 `server/db/migrations/` 下的 13 个带时间戳文件：(1) 未做 `juanleme` 原地改名——`insideout` schema 全新创建、真实数据复制后删除 `juanleme`；(2) `REVOKE CREATE ON SCHEMA public` 锁定迁移被移除——「绝不写入 `public`」靠迁移从不指向它来保证（共用实例模型的要求，见 [BUG-008](../../issues/2026-07-20-bug-008-shared-instance-db-provisioning.md)）；(3) §5 的「可选后续加固」RLS **已**实现（迁移 `20260720150000` 起与 `server/internal/store/pool.go` 的 `withUserContext`）——线上数据库已启用 RLS。当前状态见 [docs/architecture/database-and-rls.md](../../architecture/database-and-rls.md)。

## 1. Strategy / 策略

Per the decided D1: **rename `juanleme` → `insideout` and evolve in place**. The rename keeps all existing rows. What does *not* survive a schema rename automatically — and must be recreated or dropped — is everything Supabase-specific:
按已定的 D1：**将 `juanleme` 重命名为 `insideout` 并原地演进**。重命名保留所有现有数据。但重命名**不会**自动适配的、必须重建或删除的，是所有 Supabase 特有物：

- Function bodies and `SET search_path = pg_catalog, juanleme` settings still say `juanleme` after a rename (they are stored as text) — the old RPCs would break at runtime. We drop nearly all of them anyway (the Go backend replaces PostgREST RPCs with direct SQL) and recreate the two keepers (rate limiter, circuit breaker).
  函数体和 `SET search_path = pg_catalog, juanleme` 在重命名后仍写着 `juanleme`（以文本存储）——旧 RPC 运行时会报错。反正我们几乎全部删除（Go 后端用直接 SQL 取代 PostgREST RPC），只重建两个保留项（限流器、熔断器）。
- RLS policies reference `auth.uid()`, which only works under Supabase's JWT machinery. Authorization moves to the Go app layer (D4); policies are dropped, RLS disabled. The *rules they encoded* are preserved as the checklist in §5.
  RLS 策略依赖只在 Supabase JWT 机制下有效的 `auth.uid()`。授权移至 Go 应用层（D4）；删除策略、关闭 RLS。策略所承载的*规则*以 §5 清单形式保留。
- The PostgREST RPC bridge (7 `export_*` wrappers + `ai_*` service helpers) is dropped wherever it lives. Note: on the **live** database it was already moved out of `public` into `juanleme` (PostgREST there exposes `juanleme` directly); `public.export_*` only exists if replaying the repo's stale migration files. Either way, after migration #3 nothing of ours exists in `public`.
  PostgREST RPC 桥（7 个 `export_*` 包装 + `ai_*` 服务助手）无论在哪都删除。注意：**线上**数据库已把桥从 `public` 移入 `juanleme`（那里的 PostgREST 直接暴露 `juanleme`）；只有重放仓库过期迁移文件时 `public.export_*` 才存在。无论哪种情况，迁移 #3 之后 `public` 里不再有我们的任何对象。
- `profiles.id → auth.users(id)` FK, grants to `anon`/`authenticated`/`service_role`, Vault-based `get_ai_config()`, and `storage.objects` policies are all removed. AI secrets move to Go env vars; export artifacts stop using object storage (D8).
  指向 `auth.users` 的外键、对 `anon`/`authenticated`/`service_role` 的授权、基于 Vault 的 `get_ai_config()`、`storage.objects` 策略全部移除。AI 密钥改由 Go 环境变量提供；导出物不再用对象存储（D8）。

**Baseline & instance (important, found during plan verification)**: the repo's migration files are a *stale subset* of the live database. On the live instance the RPC bridge was already moved from `public` into `juanleme`, an `ai_run_events` telemetry table and `ai_record_run_event_service` function exist with no corresponding repo file, and several applied migrations are missing from the repo entirely. Moreover the live Supabase instance is **shared with unrelated projects**, so destructive migrations and a global `public` lockdown must not run there. Consequences:
**基线与实例（重要，计划核查时发现）**：仓库迁移文件只是线上数据库的*过期子集*。线上实例的 RPC 桥已从 `public` 移入 `juanleme`，存在无对应仓库文件的 `ai_run_events` 遥测表与 `ai_record_run_event_service` 函数，且多条已应用迁移在仓库中缺失。此外线上 Supabase 实例**与无关项目共用**，破坏性迁移和全局 `public` 锁定绝不能在那里执行。因此：

1. **Step 0 (default, not contingency)**: `pg_dump --schema=juanleme` from the live instance + a copy of the needed `auth.users` columns (id, email, encrypted_password, created_at), restored into the **new dedicated PostgreSQL instance**. Migrations #1+ run only there. The live instance stays untouched until P5 cutover.
   **第 0 步（默认路径而非备选）**：从线上实例 `pg_dump --schema=juanleme`，加上所需 `auth.users` 列（id、email、encrypted_password、created_at），恢复到**新专用 PostgreSQL 实例**。迁移 #1 起只在专用实例执行。线上实例保持不动直至 P5 切换。
2. The drop-lists below are written against the **live** schema (the dump), not the repo files; final DDL in P1 starts from a fresh catalog inspection of the restored dump.
   下方删除清单以**线上** schema（转储）为准而非仓库文件；P1 的最终 DDL 从恢复后转储的目录实查出发。

## 2. Roles & Permissions / 角色与权限

**Simplified (D2 revised during implementation)**: a **single role, `insideout_app`**, for everything — it owns the `insideout` schema, applies migrations, and is the runtime role the Go server connects as. The two-role (`insideout_admin` + `insideout_app`) split from the original draft added `ALTER DEFAULT PRIVILEGES`/`FOR ROLE` bookkeeping that only matters when different roles create vs. use objects; since our own Go server is what applies migrations (no external migration tool, no Supabase MCP in the loop for this — the target is whatever remote PostgreSQL URL the user provides), one role that owns everything it creates needs no default-privilege dance at all.
**简化（实现期修订 D2）**：**单一角色 `insideout_app`** 承担一切——拥有 `insideout` schema、执行迁移、也是 Go 服务连接的运行时角色。原草案的双角色（`insideout_admin` + `insideout_app`）拆分带来了 `ALTER DEFAULT PRIVILEGES`/`FOR ROLE` 的额外簿记，那只在「建对象的角色」与「用对象的角色」不同时才有意义；既然是我们自己的 Go 服务在执行迁移（不经外部迁移工具、这次也不经 Supabase MCP——目标是用户提供的任意远程 PostgreSQL URL），一个拥有自己所建一切对象的角色完全不需要默认权限那套机制。

Provisioning is whatever the user's remote PostgreSQL provider requires to end up with: a dedicated database, and one role/user — call it `insideout_app` — that **owns** that database (so it also owns `public` within it by Postgres default, letting migration #1 lock `public` down without any separate admin step). `DATABASE_URL` in `.env` is that role's connection string; the Go server's own embedded migration runner ([02 §1](02-backend-go.md)) is the *only* thing that ever runs DDL, no separate migration tool or MCP required for a non-Supabase remote instance.
配置方式取决于用户的远程 PostgreSQL 服务商，但目标一致：一个专用数据库，加一个角色/用户——就叫 `insideout_app`——**拥有**该数据库（因此按 Postgres 默认规则它也拥有库内的 `public`，使迁移 #1 无需额外管理员步骤即可锁定 `public`）。`.env` 里的 `DATABASE_URL` 就是这个角色的连接串；Go 服务自带的内嵌迁移执行器（[02 §1](02-backend-go.md)）是**唯一**执行 DDL 的东西，非 Supabase 的远程实例不需要额外迁移工具或 MCP。

```sql
-- Migration #2, run as insideout_app (already owns this database & its public schema):
-- 迁移 #2，以 insideout_app 执行（已拥有该数据库及其 public schema）：
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
```

Nothing else is needed — every later `CREATE TABLE`/`CREATE FUNCTION` in `insideout` is owned by `insideout_app` automatically, since it's the role that creates them.
无需其他操作——之后在 `insideout` 里创建的每张表/函数都自动归 `insideout_app` 所有，因为正是它在创建。

Verification (P1 acceptance): connect as `insideout_app` and assert it **can** run DML in `insideout` and **cannot** `CREATE TABLE` in `public` (confirms the revoke took; `insideout_app` owning `public` means it *could* re-grant itself, so this is a regression guard, not a security boundary against that role — the boundary that matters is that no *other*, lower-privileged role can create in `public`).
验证（P1 验收）：以 `insideout_app` 连接，断言它**能**在 `insideout` 执行 DML、**不能**在 `public` 建表（确认 revoke 生效；由于 `insideout_app` 拥有 `public`，它理论上可以给自己重新授权，故此项是回归防护而非针对该角色本身的安全边界——真正的边界是不让任何*其他*、权限更低的角色能在 `public` 建表）。

## 3. Migration Sequence / 迁移顺序

SQL files live in `server/db/migrations/` (timestamp-prefixed), applied in order by the Go server's own embedded migration runner ([02 §1](02-backend-go.md)) against whichever `DATABASE_URL` (remote PostgreSQL, user-provided) is configured — no external migration tool, no MCP in this loop. One migration = one concern:
SQL 文件放在 `server/db/migrations/`（时间戳前缀），由 Go 服务自带的迁移执行器（[02 §1](02-backend-go.md)）按顺序应用于配置的 `DATABASE_URL`（远程 PostgreSQL，用户提供）——不经外部迁移工具，也不经 MCP。一条迁移只做一件事：

1. `rename_schema` — `ALTER SCHEMA juanleme RENAME TO insideout;`
2. `create_app_role_lock_public` — §2 above / 见上文 §2
3. `drop_supabase_coupling` — drop all RLS policies + disable RLS; drop **all** old RPCs and views per live catalog inspection (~22 functions incl. the `export_*` wrappers, `ai_*` service helpers, `ai_record_run_event_service`, `get_ai_config`, and the vault reader — plus `public.export_*` if present from a repo replay); drop `profiles→auth.users` FK; revoke `anon`/`authenticated`/`service_role` grants / 删除全部 RLS 策略并关闭 RLS；按线上目录实查删除**全部**旧 RPC 与视图（约 22 个函数，含 `export_*` 包装、`ai_*` 服务助手、`ai_record_run_event_service`、`get_ai_config`、vault 读取器——若为仓库重放还含 `public.export_*`）；删除指向 `auth.users` 的外键；清理三个 PostgREST 角色的授权
4. `users_own_auth` — evolve `profiles` into `users`: rename table; add `password_hash text`, `email_verified_at timestamptz`, `meta jsonb NOT NULL DEFAULT '{}'`; backfill `email` (and optionally bcrypt `encrypted_password`, which Go can verify) from `auth.users` if present; add `sessions` table / 将 `profiles` 演进为 `users`：改表名；加 `password_hash`、`email_verified_at`、`meta`；如存在 `auth.users` 则回填 `email`（可选回填 bcrypt 的 `encrypted_password`，Go 可校验）；新增 `sessions` 表
5. `extend_projects` — new columns on existing `projects` (§4) / 在现有 `projects` 上加列（§4）
6. `create_domain_tables` — `project_updates`, `ideas`, `prds`, `prd_revisions`, `agent_conversations`, `agent_messages` (§4); then `ALTER TABLE ai_runs ADD conversation_id ..., DROP COLUMN node_id` (its FK targets soon-to-be-dropped `workshop_nodes`); finally `ALTER TABLE ideas ADD FOREIGN KEY (prd_id) REFERENCES prds(id) ON DELETE SET NULL` (added last, after `prds` exists; app-level rule: deleting a PRD reverts the linked idea to `refining`) / 新建六张领域表（§4）；随后 `ALTER TABLE ai_runs ADD conversation_id ..., DROP COLUMN node_id`（其外键指向即将删除的 `workshop_nodes`）；最后补 `ideas.prd_id → prds(id) ON DELETE SET NULL` 外键（在 `prds` 建成后添加；应用层规则：删除 PRD 时将关联想法回退为 `refining`）
7. `port_rate_limit_circuit` — recreate `ai_check_rate_limit(p_user_id)` and the circuit-breaker functions in `insideout` with `SET search_path = pg_catalog, insideout`; keep the advisory-lock sliding window (10/min, 60/hr) and 5-fail-open / 2-success-close semantics verbatim; since `insideout_app` both defines and calls these functions, no cross-role `GRANT EXECUTE` is needed / 在 `insideout` 中重建限流与熔断函数，`search_path` 指向新 schema；逐字保留 advisory-lock 滑动窗口（10/分、60/时）与 5 败开启/2 成关闭语义；因 `insideout_app` 既定义又调用这些函数，无需跨角色 `GRANT EXECUTE`
8. `retire_legacy` (after P5 cutover) — **archive, then drop** (decided, Q1): first `pg_dump --table` the four retired tables (`workshop_nodes`, `documents`, `document_revisions`, `export_jobs`) to a compressed archive kept outside the repo, then drop them / （P5 切换后）**先归档再删除**（已定，Q1）：先用 `pg_dump --table` 将四张弃用表导出为仓库之外保存的压缩归档，再执行删除

Additionally, a dev-only seed command (`go run ./cmd/insideout seed`, never a migration) provides a sample workspace with projects, ideas, and a PRD for local development. It's Go, not a raw `.sql` file, because it needs to create a real, loggable-in demo user — hashing the password through the exact same `internal/auth` argon2id path the server uses, rather than duplicating hashing logic in SQL.
另有仅供开发的种子命令（`go run ./cmd/insideout seed`，绝不做成迁移），提供含项目、想法与一份 PRD 的示例工作区。它用 Go 而非纯 `.sql` 文件实现，因为需要建一个真正可登录的演示用户——密码要经与服务端完全相同的 `internal/auth` argon2id 路径哈希，而非在 SQL 里重复一套哈希逻辑。

## 4. Target Tables / 目标表结构

Kept from the rename (columns unchanged unless noted): `users` (ex-`profiles`), `workspaces` (invite `code`, `status`), `workspace_memberships` (`admin`|`member`, UNIQUE(workspace, user)), `projects`, `ai_runs`, `ai_circuit_breaker` (note: TEXT PK, single row `'anthropic'`), and `ai_run_events` — the live-only telemetry table (model, token counts, latency, payloads) is **kept** and becomes the agent's usage telemetry. Domain tables keep UUID PKs, `created_at`/`updated_at` + the existing `set_updated_at()` trigger, and gain `meta jsonb NOT NULL DEFAULT '{}'` where missing (our extendability convention).
经重命名保留（未注明即列不变）：`users`（原 `profiles`）、`workspaces`（邀请码 `code`、`status`）、`workspace_memberships`（`admin`|`member`，UNIQUE(workspace, user)）、`projects`、`ai_runs`、`ai_circuit_breaker`（注意：TEXT 主键，单行 `'anthropic'`），以及 `ai_run_events`——仅存在于线上的遥测表（模型、token 数、延迟、载荷）**保留**，成为 Agent 的用量遥测。领域表保留 UUID 主键、`created_at`/`updated_at` 与现有 `set_updated_at()` 触发器，缺 `meta jsonb` 的补上（可扩展性约定）。

New / extended (condensed DDL sketch — final DDL written in P1):
新增/扩展（精简 DDL 草图——最终 DDL 在 P1 编写）：

```sql
-- sessions: refresh-token store / 刷新令牌存储
CREATE TABLE insideout.sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES insideout.users(id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE,          -- sha256 of refresh token / 刷新令牌的 sha256
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- projects: extend for tracking / 扩展以支持跟踪
ALTER TABLE insideout.projects
  ADD COLUMN owner_id uuid REFERENCES insideout.users(id),   -- the person maintaining it / 维护者
  ADD COLUMN status text NOT NULL DEFAULT 'active'
    CHECK (status IN ('planning','active','paused','done','archived')),
  ADD COLUMN meta jsonb NOT NULL DEFAULT '{}';

CREATE TABLE insideout.project_updates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id uuid NOT NULL REFERENCES insideout.projects(id) ON DELETE CASCADE,
  author_id uuid NOT NULL REFERENCES insideout.users(id),
  kind text NOT NULL CHECK (kind IN ('progress','blocker','note')),
  content text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE insideout.ideas (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES insideout.workspaces(id) ON DELETE CASCADE,
  author_id uuid NOT NULL REFERENCES insideout.users(id),
  title text NOT NULL,
  content text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'inbox'
    CHECK (status IN ('inbox','refining','converted','dropped')),
  prd_id uuid,                               -- FK to prds added at end of migration #6 (prds exists by then) / 外键在迁移 #6 末尾补加（届时 prds 已建）
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  meta jsonb NOT NULL DEFAULT '{}'
);

CREATE TABLE insideout.prds (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES insideout.workspaces(id) ON DELETE CASCADE,
  idea_id uuid REFERENCES insideout.ideas(id) ON DELETE SET NULL,
  project_id uuid REFERENCES insideout.projects(id) ON DELETE SET NULL,
  author_id uuid NOT NULL REFERENCES insideout.users(id),
  title text NOT NULL,
  sections jsonb NOT NULL DEFAULT '{}',      -- keyed markdown sections, see 03-agents.md §2 / 按键存 markdown 章节
  status text NOT NULL DEFAULT 'draft'
    CHECK (status IN ('draft','reviewing','approved','rejected')),
  current_revision int NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  meta jsonb NOT NULL DEFAULT '{}'
);

CREATE TABLE insideout.prd_revisions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  prd_id uuid NOT NULL REFERENCES insideout.prds(id) ON DELETE CASCADE,
  revision int NOT NULL,
  sections jsonb NOT NULL,
  created_by uuid NOT NULL REFERENCES insideout.users(id),
  note text,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (prd_id, revision)
);

CREATE TABLE insideout.agent_conversations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES insideout.workspaces(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES insideout.users(id),
  prd_id uuid NOT NULL REFERENCES insideout.prds(id) ON DELETE CASCADE,
  stage text NOT NULL DEFAULT 'clarify'
    CHECK (stage IN ('clarify','draft','critique','finalize')),
  status text NOT NULL DEFAULT 'active'
    CHECK (status IN ('active','completed','abandoned')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  meta jsonb NOT NULL DEFAULT '{}'           -- rolling summary, stage notes / 滚动摘要、阶段笔记
);

CREATE TABLE insideout.agent_messages (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id uuid NOT NULL REFERENCES insideout.agent_conversations(id) ON DELETE CASCADE,
  role text NOT NULL CHECK (role IN ('user','assistant','tool')),
  content text NOT NULL DEFAULT '',
  tool_calls jsonb,                          -- assistant tool-call payloads / assistant 的工具调用
  tool_call_id text,                         -- for role='tool' results / role='tool' 的结果关联
  tokens int,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ON insideout.agent_messages (conversation_id, created_at);
```

`ai_runs` gains `conversation_id uuid REFERENCES insideout.agent_conversations(id) ON DELETE SET NULL`, loses `node_id` (whose FK targets the retiring `workshop_nodes`), and remains the rate-limit counting source (one row per user message to the agent).
`ai_runs` 增加 `conversation_id` 外键、移除 `node_id`（其外键指向待退役的 `workshop_nodes`），并继续作为限流计数来源（用户发给 Agent 的每条消息一行）。

Agent conversation history moves **server-side** (currently localStorage-only) — that is what `agent_conversations`/`agent_messages` provide.
Agent 对话历史移到**服务端**（现状仅 localStorage）——即 `agent_conversations`/`agent_messages` 的用途。

## 5. Authorization Checklist / 授权清单

The RLS matrix being dropped encodes hard-won rules. The Go layer must enforce each of these, and each gets an integration test (deny path included):
被删除的 RLS 矩阵承载了来之不易的规则。Go 层必须逐条执行，且每条都配集成测试（含拒绝路径）：

| Resource / 资源 | Rule / 规则 |
|---|---|
| users | Read: self, or co-member of a shared workspace. Update: self only, and **never** own `role` (admin flag) — role changes are admin-only. / 读：本人或同工作区成员。改：仅本人，且**永不**可改自己的 `role`——角色变更仅管理员。 |
| workspaces | Read: members only. Update: workspace admin or creator. Delete: creator only. / 读：仅成员。改：管理员或创建者。删：仅创建者。 |
| memberships | Join **only** via invite code against an `active` workspace (duplicate join = 409). Role change: admin. Remove: admin, or self-leave. / 加入**仅**凭邀请码且工作区为 `active`（重复加入 409）。改角色：管理员。移除：管理员或本人退出。 |
| projects | Read: members. Create: any member (owner defaults to creator). Update: project owner or workspace admin. Delete: admin. / 读：成员。建：任意成员（负责人默认为创建者）。改：项目负责人或工作区管理员。删：管理员。 |
| project_updates | Read: members. Create: any member. Edit/delete: author or admin. / 读：成员。建：任意成员。改/删：作者或管理员。 |
| ideas | Read: members. Create: any member. Update: author. Drop/delete: author or admin. / 读：成员。建：任意成员。改：作者。放弃/删除：作者或管理员。 |
| prds | Read: members. Edit sections: author (or admin). Submit for review: author. Approve/reject: workspace admin (not the author acting alone). / 读：成员。改章节：作者（或管理员）。提交评审：作者。通过/驳回：工作区管理员（不得作者自审）。 |
| conversations / messages | Owner (`user_id`) only, both read and write. / 仅归属用户可读写。 |
| ai_runs | Read own; writes only by the server itself. / 读自己的；写入仅服务端内部。 |

Two structural lessons carried over from the hardening migrations / 从加固迁移继承的两条结构性经验：

1. **Ex-member lockout**: every mutation re-checks *current* membership inside the same transaction as the write (the TOCTOU fix), taking `FOR KEY SHARE` on the membership row in multi-statement transactions.
   **移除成员锁定**：每次写操作都在同一事务内复查*当前*成员资格（TOCTOU 修复），多语句事务对成员行加 `FOR KEY SHARE`。
2. **Revision serialization**: `prd_revisions.revision = MAX+1` under `FOR UPDATE` on the PRD row (same pattern `complete_node` used for `document_revisions`).
   **版本号串行化**：在 PRD 行 `FOR UPDATE` 下取 `MAX+1`（沿用 `complete_node` 对 `document_revisions` 的做法）。

~~Optional later hardening (not in scope now)~~ — **implemented on 2026-07-20 per explicit user direction**: RLS with policies reading `current_setting('app.user_id')`, set per-transaction by the Go store layer, is live (migrations `20260720150000` onward). See [docs/architecture/database-and-rls.md](../../architecture/database-and-rls.md) and [BUG-007](../../issues/2026-07-20-bug-007-rls-against-real-postgres.md) for the real-Postgres gotchas hit along the way.
~~可选的后续加固（暂不做）~~——**已于 2026-07-20 按用户明确指示实现**：基于 `current_setting('app.user_id')`、由 Go 存储层按事务设置的 RLS 已上线（迁移 `20260720150000` 起）。见 [docs/architecture/database-and-rls.md](../../architecture/database-and-rls.md) 与 [BUG-007](../../issues/2026-07-20-bug-007-rls-against-real-postgres.md)。

## 6. What We Deliberately Keep in SQL / 有意保留在 SQL 里的部分

- **Rate limiter** (`ai_check_rate_limit(p_user_id)`): per-user sliding window counted from `ai_runs`, serialized with `pg_advisory_xact_lock` — correct across multiple Go instances, unlike in-process counters. Known quirk kept: failed runs count toward the limit (anti-abuse). Known bug **fixed in the Go port**: a provider 429 must mark the run `failed`, not strand it `pending` (the old edge function leaked pending runs and blocked their idempotency keys).
  **限流器**：基于 `ai_runs` 的按用户滑动窗口，`pg_advisory_xact_lock` 串行化——多 Go 实例下依然正确，进程内计数器做不到。保留的怪癖：失败的调用也计数（防滥用）。Go 移植中**修复**的旧 bug：供应商 429 时必须将 run 置为 `failed`，不能永远悬在 `pending`（旧边缘函数会泄漏 pending 并卡死其幂等键）。
- **Circuit breaker** (`ai_circuit_breaker` + check/record functions): DB-persisted so state is shared across instances; 30s cooldown → half-open, 2 successes close, 5 consecutive failures open — unchanged.
  **熔断器**：状态存库、多实例共享；30 秒冷却进入半开、2 次成功关闭、连续 5 次失败开启——语义不变。
