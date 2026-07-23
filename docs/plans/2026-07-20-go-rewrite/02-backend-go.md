# 02 — Go Backend / Go 后端

> Part of the [InsideOut rewrite plan](README.md). / [InsideOut 重写计划](README.md)的一部分。

## 1. Stack & Layout / 技术栈与目录

**Stack (D5)**: Go 1.25+, stdlib `net/http` with 1.22+ method-pattern routing (no web framework), `jackc/pgx/v5` (pool; no ORM), `golang-jwt/jwt/v5`, `golang.org/x/crypto/argon2` (+`bcrypt` only if we verify imported GoTrue hashes), `log/slog`. sqlc is deliberately skipped for now — plain pgx queries in a small store layer; adopt sqlc if/when the query count makes hand-written scanning painful.
**技术栈（D5）**：Go 1.25+、标准库 `net/http`（1.22+ 方法模式路由，不用框架）、`jackc/pgx/v5`（连接池；不用 ORM）、`golang-jwt/jwt/v5`、`x/crypto/argon2`（若校验导入的 GoTrue 哈希再加 `bcrypt`）、`log/slog`。sqlc 有意暂缓——先用小型 store 层手写 pgx 查询；当查询数量让手写扫描变痛苦时再引入。

```text
server/
├── cmd/insideout/main.go     # config, pool, routes, graceful shutdown / 配置、连接池、路由、优雅退出
├── internal/
│   ├── api/                  # handlers per domain + server.go (route registration) + middleware (auth, CORS, logging, recover)
│   ├── auth/                 # argon2id hashing, JWT mint/verify, refresh rotation
│   ├── store/                # pgx queries per domain: users, workspaces, projects, ideas, prds, conversations
│   ├── agent/                # PRD coach — see 03-agents.md / PRD 教练——见 03-agents.md
│   ├── export/               # markdown + print-HTML rendering
│   └── config/               # env parsing, fail-fast validation
├── db/migrations/            # SQL files — see 01-database.md §3 / SQL 迁移文件
└── go.mod
```

The 350-line-per-file rule applies to Go files; the per-domain split above is sized for that.
每文件 350 行的规则适用于 Go 文件；上述按领域拆分即为此设计。

**Config (env)** / **配置（环境变量）**: `INSIDEOUT_ADDR`, `DATABASE_URL` (as `insideout_app` — any PostgreSQL 14+ instance: a user-provided remote one during development, or the compose `postgres:17` in production), `INSIDEOUT_JWT_SECRET`, `INSIDEOUT_ACCESS_TTL` (default 15m), `INSIDEOUT_REFRESH_TTL` (default 720h), `ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, `AI_MODEL`. This replaces Supabase Vault's `get_ai_config()`. We keep its graceful degradation: **missing AI config → canned template reply, not an error** (it doubles as offline dev mode).
取代 Supabase Vault 的 `get_ai_config()`。保留其优雅降级：**AI 配置缺失 → 固定模板回复而非报错**（兼作离线开发模式）。

**Embedded migration runner / 内嵌迁移执行器**: no external migration tool and no MCP dependency — `internal/store/migrate.go` reads `db/migrations/*.sql` (embedded via `//go:embed`), tracks applied filenames in a `insideout.schema_migrations(filename text primary key, applied_at timestamptz)` table (bootstrapped with `CREATE SCHEMA IF NOT EXISTS insideout` + `CREATE TABLE IF NOT EXISTS ... schema_migrations` before anything else), and applies unapplied files in filename order inside a transaction each. Invoked via `go run ./cmd/insideout migrate` (a CLI subcommand, not an HTTP route) against whatever `DATABASE_URL` is configured. This is what "single role, `insideout_app`" in [01 §2](01-database.md) actually runs as.
无外部迁移工具、也不依赖任何 MCP——`internal/store/migrate.go` 通过 `//go:embed` 读取 `db/migrations/*.sql`，用 `insideout.schema_migrations(filename text primary key, applied_at timestamptz)` 表记录已应用的文件名（首次运行前自举 `CREATE SCHEMA IF NOT EXISTS insideout` + 该表），按文件名顺序在各自事务中应用未应用的文件。通过 `go run ./cmd/insideout migrate` 命令（CLI 子命令，非 HTTP 路由）对配置的 `DATABASE_URL` 执行。这正是 [01 §2](01-database.md) 中「单一角色 `insideout_app`」实际运行迁移的方式。

## 2. Auth / 认证

Current "auth" is half-fake (OTP call with no verify step + fabricated localStorage token), so this is a greenfield design with boring, standard choices (D3):
现状认证是半假的（无校验步骤的 OTP 调用 + 伪造的 localStorage token），因此这是全新设计，采用保守标准方案（D3）：

- **Register/login**: email + password; argon2id (64 MiB, t=1, p=4). Optionally verify imported GoTrue bcrypt hashes on first login and re-hash to argon2id (only relevant if real users exist).
  **注册/登录**：邮箱+密码；argon2id（64 MiB，t=1，p=4）。可选：首次登录校验导入的 GoTrue bcrypt 哈希并重哈希为 argon2id（仅当存在真实用户时有意义）。
- **Sessions**: access = 15-min JWT (HS256, claims: `sub`, `exp`, `iat`) in an httpOnly `SameSite=Lax` `Secure` cookie; refresh = opaque random token, sha256-stored in `insideout.sessions`, 30-day TTL, **rotated on every refresh** (old row gets `revoked_at`). No token-family reuse detection for v1. `// ponytail: rotation only; add family reuse-detection if this ever guards real money`
  **会话**：access 为 15 分钟 JWT（HS256），放 httpOnly `SameSite=Lax` `Secure` cookie；refresh 为不透明随机令牌，sha256 存入 `insideout.sessions`，30 天有效，**每次刷新轮换**（旧行写入 `revoked_at`）。v1 不做令牌家族重用检测。
- Cookies unlock **SSR auth** — the Nuxt middleware finally works on the server ([04 §2](04-frontend.md)). The API also accepts `Authorization: Bearer` for tests and CLI use.
  Cookie 顺带解锁 **SSR 认证**——Nuxt 中间件终于能在服务端生效（[04 §2](04-frontend.md)）。API 同时接受 `Authorization: Bearer`，便于测试与命令行。
- Middleware resolves the user once per request and injects it into `context`; handlers never parse tokens themselves.
  中间件每请求解析一次用户并注入 `context`；handler 绝不自行解析令牌。

## 3. API Surface / API 面

All under `/api/v1`. JSON bodies are **camelCase** (matches the frontend boundary convention). Auth column: 🔓 public, 🔑 authenticated, 👤 member of the resource's workspace, 🛡 workspace admin.
统一挂在 `/api/v1` 下。JSON 一律 **camelCase**（契合前端边界约定）。认证列：🔓 公开、🔑 已登录、👤 资源所属工作区成员、🛡 工作区管理员。

| Method + Path | Purpose / 用途 | Auth |
|---|---|---|
| POST `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` | Session lifecycle / 会话生命周期 | 🔓 |
| GET / PATCH `/me` | Own profile (username, bio, keywords, avatarUrl) / 本人资料 | 🔑 |
| GET / POST `/workspaces` | List mine; create (creator becomes admin, 6-digit code generated **with collision retry** — fixes the old no-retry bug) / 列出我的；创建（创建者成为管理员，6 位邀请码**带碰撞重试**——修复旧无重试 bug） | 🔑 |
| POST `/workspaces/join` `{code}` | Join an active workspace / 凭码加入活跃工作区 | 🔑 |
| GET / PATCH / DELETE `/workspaces/{id}` | Detail (incl. memberCount) / manage / 详情（含成员数）/ 管理 | 👤 / 🛡 / creator |
| GET `/workspaces/{id}/members` · PATCH / DELETE `/workspaces/{id}/members/{userId}` | List; role change; remove (or self-leave) / 列表；改角色；移除（或自退） | 👤 / 🛡 |
| GET / POST `/workspaces/{id}/projects` | Board list (each project with latest update + staleness) ; create / 看板列表（含最新动态与滞后度）；创建 | 👤 |
| GET / PATCH / DELETE `/projects/{id}` | Detail with updates timeline / manage / 详情含动态时间线 / 管理 | 👤 / owner-or-🛡 / 🛡 |
| POST `/projects/{id}/updates` · PATCH / DELETE `/updates/{id}` | Add progress / blocker / note; edit or remove one / 添加进度/阻塞/备注；编辑或删除 | 👤 / author-or-🛡 |
| GET / POST `/workspaces/{id}/ideas` | Idea inbox; quick capture / 想法收集箱；快速记录 | 👤 |
| GET / PATCH / DELETE `/ideas/{id}` | Edit; drop / 编辑；放弃 | author / author-or-🛡 |
| POST `/ideas/{id}/convert` | Idea → PRD + coach conversation, idea marked `converted` / 想法 → PRD + 教练对话，想法置为 `converted` | author |
| GET / PATCH `/prds/{id}` | Read; edit title/sections (autosave) / 读取；编辑标题/章节（自动保存） | 👤 / author-or-🛡 |
| GET / POST `/prds/{id}/revisions` | History; snapshot current sections as next revision / 历史；将当前章节快照为下一版本 | 👤 / author |
| POST `/prds/{id}/status` `{status}` | draft→reviewing (author); reviewing→approved / rejected (🛡); rejected→draft (author — iterate and resubmit) / 状态流转：草稿→评审（作者）；评审→通过/驳回（🛡）；驳回→草稿（作者——修改后重新提交） | author / 🛡 |
| GET `/prds/{id}/export?format=markdown|print` | On-demand render, direct download (D8: no object storage, no job table) / 按需渲染直接下载（D8：无对象存储、无任务表） | 👤 |
| GET `/prds/{id}/conversation` | Latest coach conversation for this PRD — lets the PRD workspace resume coaching on revisit, not just right after conversion / 该 PRD 最新的教练对话——让 PRD 工作台在再次访问时能续接教练对话，而不仅限于转化后那一次 | 👤 |
| GET `/conversations/{id}` · GET `/conversations/{id}/messages` | Coach conversation + history (server-side now) / 教练对话与历史（改为服务端存储） | owner |
| POST `/conversations/{id}/messages` | Send message → **SSE stream** reply ([03 §4](03-agents.md)) / 发消息 → **SSE 流式**回复 | owner |
| GET `/healthz` | Liveness + DB ping / 存活与数据库探测 | 🔓 |

**Error contract / 错误契约**: `{ "error": string, "code"?: string, ... }`. The AI throttling shapes are preserved **verbatim** so the existing frontend countdown logic keeps working: 429 `code:"APP_THROTTLE"` with `retry_after_seconds`/`current_count`/`max_requests`; 503 `code:"CIRCUIT_OPEN"` with `retry_after_seconds`/`circuit_state`; 503 `code:"ANTHROPIC_RATE_LIMIT"` with `retry_after_seconds`. (Yes, these extras are snake_case while bodies are camelCase — inherited; unifying is optional cleanup **after** cutover, never during.)
AI 限流响应**逐字保留**，确保前端倒计时逻辑不改即用：429 `APP_THROTTLE`、503 `CIRCUIT_OPEN`、503 `ANTHROPIC_RATE_LIMIT` 及其字段。（这些附加字段是 snake_case 而正文是 camelCase——历史遗留；统一命名是切换**之后**的可选清理，切换期间绝不动。）

**Dropped, not ported / 明确不移植**: the export job model (`export_jobs`, idempotency keys, signed URLs, the `html`→`print` CHECK-constraint retry hack, escaped-markdown-in-`<pre>` "HTML") — export becomes a synchronous render-and-download. The dead service surface the frontend never called is not blindly re-created either; the table above is the real contract.
**明确不移植**：导出任务模型（`export_jobs`、幂等键、签名 URL、`html`→`print` 的 CHECK 约束重试黑科技、转义 markdown 塞 `<pre>` 的伪 HTML）——导出改为同步渲染下载。前端从未调用的僵尸服务面也不盲目重建；上表即真实契约。

## 4. Cross-Cutting / 横切关注点

- **DB access**: one `pgxpool` per process; every multi-step write is a transaction; membership re-check inside the transaction per [01 §5](01-database.md). Store functions take `context.Context` first, return domain structs.
  **数据库访问**：每进程一个 `pgxpool`；所有多步写入走事务；事务内复查成员资格（见 [01 §5](01-database.md)）。store 函数首参 `context.Context`，返回领域结构体。
- **Timeouts**: `ReadHeaderTimeout` 5s; default per-request context timeout 10s; agent/SSE routes 120s. Graceful shutdown drains in-flight SSE streams.
  **超时**：`ReadHeaderTimeout` 5 秒；默认请求上下文 10 秒；Agent/SSE 路由 120 秒。优雅退出时排空进行中的 SSE 流。
- **Transport (no CORS)**: the Nuxt app proxies `/api/v1/**` to the Go server via a Nitro route rule, so browser and SSR calls are **same-origin** and the httpOnly cookies just work — cross-origin CORS-with-credentials (and its SameSite pitfalls) is avoided entirely. The old `Access-Control-Allow-Origin: *` dies with the edge functions. A permissive-CORS dev flag exists only for hitting the API directly with curl/tests.
  **传输（无 CORS）**：Nuxt 经 Nitro 路由规则将 `/api/v1/**` 代理到 Go 服务，浏览器与 SSR 调用均为**同源**，httpOnly cookie 天然生效——完全避开跨域 CORS 携带凭据（及其 SameSite 陷阱）。旧的 `*` 通配随边缘函数一起消亡。仅保留供 curl/测试直连 API 的宽松 CORS 开发开关。
- **Logging**: `slog` JSON, request id middleware, one line per request (method, path, status, duration, userId if any). No bodies logged.
  **日志**：`slog` JSON、请求 id 中间件、每请求一行（方法、路径、状态、耗时、用户 id）。不记录请求体。
- **Validation**: at the trust boundary in handlers — UUID path params, enum values, length limits (message ≤ 10,000 chars carried over), email format. 400 for malformed input, 422 never (keep the existing 400 convention).
  **校验**：在 handler 信任边界完成——UUID 路径参数、枚举值、长度上限（消息 ≤ 10,000 字继承）、邮箱格式。畸形输入返回 400，不用 422（沿用现有 400 约定）。

## 5. Testing / 测试

Per our testing rules: **no mocks, real dependencies, all tests must pass**.
按测试规则：**不用 mock、依赖真实、全部通过才算完成**。

1. **Integration tests against real PostgreSQL** (`TEST_DATABASE_URL`): each package's tests run inside a transaction rolled back per test, or against a scratch schema created/dropped per run. Covers every row of the authorization checklist — **including every deny path** (403s are where rewrites regress).
   **对真实 PostgreSQL 的集成测试**：每条测试在回滚事务或一次性 scratch schema 中运行。覆盖授权清单每一行——**含所有拒绝路径**（403 正是重写最易回归之处）。
2. **HTTP-level tests** via `httptest` against the real router + real DB: auth lifecycle (register→login→refresh rotation→logout→refresh reuse rejected), workspace join by code, PRD status transitions, rate-limit 429 shape.
   **HTTP 层测试**：`httptest` 打真实路由+真实库：认证生命周期、邀请码加入、PRD 状态流转、限流 429 形状。
3. **One real-AI smoke test**, skipped unless `ANTHROPIC_AUTH_TOKEN` is set: sends one coach message, asserts a non-empty streamed reply and a persisted `ai_runs` row. Keeps CI honest without burning tokens on every run.
   **一条真实 AI 冒烟测试**，仅在设置 `ANTHROPIC_AUTH_TOKEN` 时运行：发一条教练消息，断言非空流式回复与已落库的 `ai_runs` 行。既守住真实性又不在每次 CI 烧 token。
4. Bugs found along the way get dated `docs/issues/` records (English-only since 2026-07-21; the former bilingual `docs/en|cn/BUGS/` pair was merged into `docs/issues/`).
   过程中发现的 bug 写入按日期命名的 `docs/issues/` 记录（2026-07-21 起仅英文；原双语 `docs/en|cn/BUGS/` 已并入 `docs/issues/`）。

## 6. Delivery / 交付形态

`docker-compose.yml` at repo root: `postgres:17` + `server` (multi-stage Go build, distroless) + `app` (Nuxt, fronting the API via its Nitro proxy) — the self-hosted default. During this build the developer instead points `DATABASE_URL` at a separately provisioned remote PostgreSQL instance; the embedded migration runner behaves identically either way, so the compose file stays the shipped artifact without being the only supported path. `.env.example` with every variable documented, real `.env` git-ignored and never read by tooling. At P7, the retired TypeScript backend (`supabase/functions/` and remaining `supabase/` config) is deleted from the repo.
仓库根部 `docker-compose.yml`：`postgres:17` + `server`（多阶段 Go 构建，distroless）+ `app`（Nuxt，经其 Nitro 代理承接 API）——自托管默认形态。本次构建中开发者改为将 `DATABASE_URL` 指向单独配置的远程 PostgreSQL 实例；内嵌迁移执行器对两者行为一致，因此 compose 文件仍是交付产物，但不是唯一支持路径。`.env.example` 注释齐全，真实 `.env` 忽略入库且工具不读取。P7 阶段从仓库删除弃用的 TypeScript 后端（`supabase/functions/` 及其余 `supabase/` 配置）。
