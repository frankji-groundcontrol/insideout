# 04 — Frontend: API Swap + Modernization / 前端：API 切换与现代化

> Part of the [InsideOut rewrite plan](README.md). / [InsideOut 重写计划](README.md)的一部分。

## 1. Service Layer Swap / 服务层切换

The existing registry seam (`src/services/registry.ts` + `useServices()`) is exactly the right cut point: pages/stores never import supabase directly. Plan:
现有注册表接缝（`src/services/registry.ts` + `useServices()`）正是理想切换点：页面/store 从不直接引入 supabase。计划：

- Add an **`api` mode** whose adapters call `/api/v1` with `$fetch`/`useFetch`. All calls are **same-origin**: a Nitro route rule proxies `/api/v1/**` to the Go server ([02 §4](02-backend-go.md)), so the httpOnly auth cookies are first-party and flow automatically in the browser; SSR data fetches use `useRequestFetch()` so the incoming request's cookies are forwarded through the proxy. No CORS, no `SameSite` pitfalls. `api` becomes the default; ~~**`mock` mode stays**~~ **[superseded during implementation, per user decision: mock mode was removed entirely — the registry ships a single real API-backed bundle; the offline dev path is the server-side template-reply coach instead]**.
  新增 **`api` 模式**，适配器用 `$fetch`/`useFetch` 调 `/api/v1`。所有调用均为**同源**：Nitro 路由规则将 `/api/v1/**` 代理到 Go 服务（[02 §4](02-backend-go.md)），因此 httpOnly 认证 cookie 属第一方、在浏览器中自动携带；SSR 数据获取使用 `useRequestFetch()`，将入站请求的 cookie 经代理转发。无 CORS、无 `SameSite` 陷阱。`api` 成为默认；~~**`mock` 模式保留**~~ **【实现期已被取代（用户决定）：mock 模式已彻底移除——注册表只提供单一的真实 API 实现；离线开发路径改由服务端模板回复教练承担】**。
- The interface set changes with the pivot: keep `IAuthService`; replace `IWorkshopService`/`IEditorService` with `IWorkspaceService`, `IProjectService`, `IIdeaService`, `IPrdService`; replace `IAiService` with `ICoachService` (SSE streaming); `IExportService` becomes one download call. Mock implementations updated to match.
  接口集随产品转向调整：保留 `IAuthService`；以 `IWorkspaceService`、`IProjectService`、`IIdeaService`、`IPrdService` 取代 `IWorkshopService`/`IEditorService`；以 `ICoachService`（SSE 流式）取代 `IAiService`；`IExportService` 收敛为一个下载调用。mock 实现同步更新。
- After cutover: **delete** `src/services/supabase/`, `src/lib/supabase.ts`, the `@supabase/supabase-js` dependency, and their mocked unit tests. Dead surface (unused `updateNodeStatus`/`getSubmission`/etc.) is not re-created in the new interfaces.
  切换后：**删除** `src/services/supabase/`、`src/lib/supabase.ts`、`@supabase/supabase-js` 依赖及其 mock 单测。僵尸接口不在新接口集中重建。
- SSE client: `EventSource` can't POST, so `ICoachService.send()` uses `fetch` + `ReadableStream` parsing in a `useCoachStream()` composable — reusing the existing rate-limit countdown logic from `useAiConversation.ts` (the 429/503 error contract is preserved server-side for exactly this reason).
  SSE 客户端：`EventSource` 不能 POST，故 `ICoachService.send()` 用 `fetch` + `ReadableStream` 解析，封装为 `useCoachStream()`——复用 `useAiConversation.ts` 现有的限流倒计时逻辑（服务端保留 429/503 契约正为此）。

## 2. Real Auth in the App / 应用内的真实认证

- Login/register pages call the Go auth endpoints; the fabricated `session_${Date.now()}` token, `juanleme-token` localStorage key, and `plugins/auth-init.client.ts` are removed. The user store hydrates from `GET /api/v1/me`.
  登录/注册页调用 Go 认证接口；伪造 token、`juanleme-token` localStorage 键和 `auth-init.client.ts` 移除。用户 store 从 `GET /api/v1/me` 水合。
- With httpOnly-cookie sessions, `middleware/auth.global.ts` finally runs on the **server** too (drop the `import.meta.server` early-return): it awaits the me-fetch and redirects unauthenticated visitors before HTML is sent — no more logged-out flash.
  有了 httpOnly cookie 会话，`auth.global.ts` 终于也能在**服务端**运行（去掉 `import.meta.server` 提前返回）：HTML 发出前完成用户获取与未登录重定向——不再有「未登录闪烁」。
- Data fetching moves from `onMounted` to `useAsyncData` page by page, making SSR real instead of nominal.
  数据获取逐页从 `onMounted` 迁到 `useAsyncData`，让 SSR 名副其实。

## 3. Information Architecture / 信息架构

| Route / 路由 | Page / 页面 |
|---|---|
| `/` | Real landing: product intro, three-pillar value props, CTA — replaces the HelloWorld showcase / 真实落地页：产品介绍、三大支柱价值、行动号召——替换 HelloWorld 展示页 |
| `/login`, `/register` | Real forms; the decorative fake social/register buttons go away / 真实表单；装饰性假社交/注册按钮删除 |
| `/dashboard` | My workspaces (joined / managed) — **wires the create & join buttons that are currently `console.log` stubs** / 我的工作区（参与/管理）——**接通目前还是 `console.log` 桩的创建与加入按钮** |
| `/workspace/[id]` | **Project board** (the group-leader view): table or card grid of projects — owner, status, latest update, staleness indicator (e.g., "no update in 14 days"); filters by status/owner / **项目看板**(组长视图)：项目表格或卡片——负责人、状态、最新动态、滞后提示（如「14 天无更新」）；按状态/负责人筛选 |
| `/workspace/[id]/ideas` | **Idea inbox**: one-line quick-capture composer on top, idea list with status chips; convert-to-PRD action / **想法收集箱**：顶部一行快速记录，下方带状态标签的列表；转化为 PRD 操作 |
| `/projects/[id]` | Project detail: updates timeline (progress/blocker/note composer), linked PRD / 项目详情：动态时间线（进度/阻塞/备注发布器）、关联 PRD |
| `/prd/[id]` | **PRD workspace** — the flagship screen: left = 8 structured sections (markdown editing, autosave, live-refresh on agent `prd_updated` events), right = coach chat with streaming; top bar = title, `PrdStatusBadge`, revision snapshot, submit-for-review / **PRD 工作台**——旗舰页面：左侧 8 个结构化章节（markdown 编辑、自动保存、随 Agent `prd_updated` 事件实时刷新），右侧流式教练对话；顶栏含标题、`PrdStatusBadge`、版本快照、提交评审 |
| `/prd/[id]/export` | Print-preview + markdown download (calls the Go export endpoint) / 打印预览 + markdown 下载（调用 Go 导出接口） |
| `/profile` | Kept; avatar upload stays a deferred item (currently a fake local preview) / 保留；头像上传仍延后（现为假的本地预览） |

Approve/reject lives on the PRD workspace itself (admins see the action on `reviewing` PRDs) plus a "needs review" filter on the board — no separate review-queue page for v1. `// ponytail: add a dedicated review queue page when review volume demands it`
通过/驳回就放在 PRD 工作台上（管理员在 `reviewing` 状态可见操作），看板加一个「待评审」筛选——v1 不做独立评审队列页。

Retired with D10: `/workshop/[id]` roadmap 3-pane flow and its stores/composables. ~~The Tiptap editor, AI sidebar patterns, draft-autosave logic, and mobile tab-switcher all **survive** — repurposed into the PRD workspace.~~ **[Superseded during implementation: Tiptap did not survive — all `@tiptap/*` deps were removed; PRD sections ship as plain markdown textareas (a known limitation in docs/TODO.md). The AI-sidebar interaction pattern lives on as the coach chat.]**
随 D10 退役：`/workshop/[id]` 路线图三栏流程及其 store/composable。~~Tiptap 编辑器、AI 侧栏模式、草稿自动保存逻辑、移动端标签切换器全部**存活**——改造进 PRD 工作台。~~ **【实现期已被取代：Tiptap 未存活——全部 `@tiptap/*` 依赖已移除；PRD 章节以纯 markdown 文本域交付（docs/TODO.md 已知限制）。AI 侧栏交互模式以教练对话的形式延续。】**

## 4. Prettify: Ink & Seal, For Real / 美化：让「国风留白」真正落地

The 国风留白 / Ink & Seal token layer is designed, mapped in Tailwind — and **never loaded**. Modernization is mostly *activating what exists*:
「国风留白 / Ink & Seal」令牌层已设计、已映射进 Tailwind——却**从未加载**。现代化的主体是*激活既有资产*：

1. **Wire the tokens**: add `~/assets/tokens.css` to the `nuxt.config.ts` css array; move `body` typography onto the token font stacks (Inter + Noto Sans SC; Noto Serif SC for display headings).
   **接线令牌**：把 `~/assets/tokens.css` 加入 `nuxt.config.ts` css 数组；`body` 排版切到令牌字体栈（Inter + Noto Sans SC；标题用思源宋体）。
2. **Codemod every component** off raw `indigo-*`/`gray-*` + scattered `dark:` utilities onto the semantic set (`bg-surface-*`, `border-stroke-*`, `text-fg-*`, `bg-btn text-btn-fg`, status ramps). Dark mode then flows from the `.dark` token flip alone. Exit criterion for P6: **zero raw palette utilities remain**.
   **逐组件替换**：清除裸 `indigo-*`/`gray-*` 与散落的 `dark:`，改用语义类。暗色模式仅靠 `.dark` 令牌翻转生效。P6 退出标准：**不残留任何裸调色板工具类**。
3. **Base component build-out**: `BaseCard`, `BaseBadge` (+`PrdStatusBadge` with 评审中/通过 seal-wash statuses the design system already specified), `BaseModal`, `BaseToast`, `icons.ts`. The design-system changelog claims these landed, but they exist only in the sibling juanleme repo — **port from there if accessible, else rebuild to the changelog's spec**.
   **基础组件补齐**：`BaseCard`、`BaseBadge`（含设计系统早已规划的 `PrdStatusBadge` 印泥晕染状态）、`BaseModal`、`BaseToast`、`icons.ts`。设计系统 changelog 声称已落地，实际只存在于姊妹 juanleme 仓库——**可访问则移植，否则按 changelog 规格重建**。
4. **Fix the orphans**: `AiSidebar.vue` is styled against an undefined third vocabulary (`--jlm-*` vars, `border-border-default`, `shadow-brutal-sm` — all resolving to nothing). It gets rewritten on the semantic tokens as part of becoming the coach chat panel.
   **修复孤儿样式**：`AiSidebar.vue` 用的是不存在的第三套词汇（`--jlm-*`、`border-border-default`、`shadow-brutal-sm`——全部落空）。改造为教练对话面板时基于语义令牌重写。
5. **Brand hygiene**: real favicon + logo replacing `/vite.svg`, title `InsideOut` (Q3 decided) instead of `app`, BaseButton primary switched to ink (`bg-btn text-btn-fg`) with vermilion strictly as seal/accent — per the design-system's stated intent.
   **品牌卫生**：真实 favicon 与 logo 取代 `/vite.svg`，标题由 `app` 改为 `InsideOut`（Q3 已定），BaseButton 主按钮切为墨色（`bg-btn text-btn-fg`），朱砂只作印章/点缀——符合设计系统既定意图。
6. **Design rules honored on every new screen** (the documented 铁律): functional pages full-width (no `max-w-7xl` containers), auto-fit card grids, 44px minimum touch targets, mobile-first, no hover-only interactions, light+dark audited page by page.
   **新页面全部遵守既有铁律**：功能页满宽（禁 `max-w-7xl` 容器）、auto-fit 卡片网格、44px 最小触控目标、移动优先、无仅悬停交互、亮暗两态逐页审查。
7. **Modern-feel polish**: 150–200ms micro-transitions on interactive states (respecting `prefers-reduced-motion`), token-driven radius/shadow consistency, designed empty states with CTAs (dashboard, ideas, board), skeleton loaders kept, toast feedback on all mutations, visible focus rings.
   **现代感打磨**：交互态 150–200ms 微过渡（尊重 `prefers-reduced-motion`）、令牌驱动的圆角/阴影一致性、带 CTA 的空状态设计（工作台、想法、看板）、保留骨架屏、所有写操作有 toast 反馈、可见焦点环。

## 5. i18n & Theme Persistence / 国际化与主题持久化

Move locale + theme from localStorage to **cookies** so SSR renders the user's actual language and theme — killing the zh-CN/light-mode hydration flash. Prefer migrating to `@nuxtjs/i18n` (cookie strategy built in); if the migration fights us, the minimal fix is cookie-reading in our existing vue-i18n plugin. zh-CN stays default, en-US kept, and a locale key-parity guard test is **added** (none exists today).
将语言与主题从 localStorage 移到 **cookie**，让 SSR 按用户真实语言与主题渲染——消灭 zh-CN/亮色水合闪烁。优先迁移到 `@nuxtjs/i18n`（内置 cookie 策略）；若迁移阻力大，最小方案是让现有 vue-i18n 插件读 cookie。zh-CN 仍为默认，en-US 保留，并**新增**语言键齐全性守卫测试（现无此测试）。

Naming cleanup rides along (Q3 decided): `juanleme-theme`/`juanleme-lang` keys → `insideout-*` cookies; the `app` page title → `InsideOut` (§4.5); i18n copy pass for the new product language.
命名清理顺路完成（Q3 已定）：`juanleme-theme`/`juanleme-lang` 键改为 `insideout-*` cookie；页面标题由 `app` 改为 `InsideOut`（§4.5）；为新产品语言重写 i18n 文案。

## 6. Frontend Testing / 前端测试

- Keep vitest for stores and composables (`useCoachStream` SSE parsing against a real local Go server in test mode), plus the new i18n parity guard from §5.
  vitest 继续覆盖 store 与 composable（`useCoachStream` 的 SSE 解析对着测试模式下的真实本地 Go 服务），外加 §5 新增的 i18n 齐全性守卫。
- The old supabase-adapter unit tests (which mocked the client) are deleted with the adapters; their replacement is the Go backend's own HTTP-level integration tests plus one end-to-end happy path: register → create workspace → record idea → convert → one coach exchange → snapshot revision → submit review — run against the compose stack.
  旧的 supabase 适配层单测（mock 客户端）随适配层删除；替代物是 Go 后端自身的 HTTP 集成测试，外加一条端到端主路径：注册 → 建工作区 → 记想法 → 转化 → 一次教练往返 → 存版本 → 提交评审——在 compose 栈上运行。
