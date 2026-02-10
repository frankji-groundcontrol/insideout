# Learnings & Conventions

## 2026-02-10 Task: Initial Analysis
- Package manager: pnpm (pnpm-lock.yaml exists)
- Pinia 3.x installed (not 2.x)
- Path alias: `@/` → `src/` in tsconfig.app.json
- Build command: `vue-tsc -b && vite build`
- WorkshopDetailView has dual-script-block bug (line 1-38 + line 40-42)
- mockApi has auth + workshop methods, missing: node status update, content save/load, AI chat
- All comments in Chinese (中文注释)
- tsconfig.app.json has `erasableSyntaxOnly: true` — no enums, use `as const` objects instead
- tsconfig.app.json has `noUnusedLocals: true` and `noUnusedParameters: true` — strict

## 2026-02-10 Task: Infrastructure Setup (Task 0)
- pnpm requires `corepack enable pnpm` on this machine (not globally installed, uses corepack)
- Shell init required: `export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"` before pnpm/node
- Vitest v4.0.18 installed, uses `mergeConfig` from `vitest/config` to extend vite config
- happy-dom environment works well for Vue component testing
- Tiptap v2.27.2 installed (@tiptap/vue-3, starter-kit, pm, extension-placeholder)
- vue-i18n@9.14.5 installed (deprecated but still latest v9 line)
- Pre-existing build errors found and fixed:
  - RoadmapSidebar.vue: unused `computed` import
  - DashboardView.vue: unused `router` variable from `useRouter()`
  - mock/index.ts: unused `workshopId` param (prefixed with `_`)
- Service interfaces designed in `app/src/types/services.ts` covering: IAuthService, IWorkshopService, IEditorService, IAiService, IExportService
- New types added to `app/src/types/index.ts`: EditorDraft, AiMessage, AiConversation, ExportConfig, UserSubmission
- WorkshopDetailView dual-script-block bug fixed: merged `computed` import into first script block, removed second script block

## 2026-02-10 Task: Dark Mode + i18n + Emoji Cleanup (Task 1)
- Tailwind dark mode enabled with `darkMode: 'class'` in `app/tailwind.config.js`.
- i18n setup created in `app/src/i18n/index.ts` with locale persistence key `juanleme-lang` and default `zh-CN`.
- Locale files `app/src/i18n/locales/zh-CN.ts` and `app/src/i18n/locales/en-US.ts` now centralize UI text by page/component domains.
- Added `ThemeToggle.vue` (`juanleme-theme`) and `LangToggle.vue` (`juanleme-lang`) with `data-testid` hooks for Vitest.
- Emoji scan on macOS cannot use `grep -P`; Python Unicode-range scan was reliable for verification.
- One leftover emoji existed in `app/src/components/HelloWorld.vue`; replaced with Heroicon (`SwatchIcon`) to satisfy zero-emoji rule across components/views.

## 2026-02-10 Task: Auth Module (Task 2)
- Pinia 3.x composition API: `defineStore('name', () => { ... })` works perfectly with `createPinia()` + `setActivePinia()` in tests
- `email.split('@')[0]` returns `string | undefined` under strict TS — need `?? email` fallback
- `vue-tsc` build is the true type-checker; Biome LSP shows false "unused" warnings for `<script setup>` variables used only in template
- localStorage keys: `juanleme-token`, `juanleme-user` for session persistence
- Router guard pattern: check `to.meta.public` for public routes, redirect unauthenticated to `/login`, redirect authenticated away from `/login` to `/dashboard`
- `useUserStore()` works inside `router.beforeEach()` because Pinia is registered before router in main.ts
- Test timing: 500ms mock delay in login means tests take ~2.5s for 5 login-related tests
- NavBar: `BaseButton` doesn't support `as` prop for polymorphic rendering — use native `RouterLink` with matching styles instead

## 2026-02-10 Task: Roadmap Store + RoadmapItem (Task 4)
- 先做 RED：`workshop.test.ts` 先落地后再建 `workshop.ts`，能明确锁定状态机行为（locked 不可选、pending 选中后进 in_progress）。
- `services.workshop` 可直接在 Vitest 中用 `vi.spyOn(...).mockResolvedValue(...)` 打桩，不需要额外 DI 层。
- `completeNode` 若要满足验收“next node -> pending”，不要直接复用 `selectNode(nextId)`，否则会把 pending 立刻推进到 in_progress。
- Roadmap 节点组件拆分后，Sidebar 只保留容器职责（标题、滚动区、事件转发），更利于 Task 7 接入 store。
- Heroicons 24/outline 中没有 `CircleIcon`，用 `StopCircleIcon as CircleIcon` 做 pending 的圆形描边替代。

## 2026-02-10 Task: Tiptap Editor + editorStore (Task 5)
- `editorStore` 新增防抖自动保存：`setContent()` 每次更新递增 `draftRevision`，800ms 后仅保存最新 revision，避免过期写入。
- `flush()` 用于节点切换/卸载时立即保存，并清理防抖计时器，防止重复写入。
- TDD 中使用 `vi.hoisted()` 声明 `vi.mock` 依赖函数，避免 mock 提升导致的 TDZ 报错。
- `TaskEditor.vue` 采用单向同步：运行期只做 Tiptap -> Pinia，只有首次加载草稿时从 store 回填编辑器。
- 插入队列采用 `watch(insertQueue.length)` + `dequeueInsert()` 循环消费，保证 AI 侧批量插入顺序稳定。
- `EditorToolbar.vue` 独立抽离后，按钮 active/disabled 状态直接依赖 `editor.isActive()` 与 `editor.can().chain()`。

## 2026-02-10 Task: AI Sidebar (Task 6)
- `AiSidebar.vue` 使用本地会话状态（`messages`、`isThinking`、`adoptedIds`），并在 `nodeId` 变化时重置，避免跨节点串话。
- AI 回复直接调用 `services.ai.reply(nodeId, message)`，用 `requestToken` 防止节点切换后的异步回包污染当前对话。
- 采纳流程只通过 `editorStore.enqueueInsert(text)` 进行，确保 AI 面板与编辑器解耦。
- 通过 `watch(messages.length + isThinking)` + `nextTick` 自动滚动到底部，保证新消息与“思考中”状态可见。
- 新增 `ai.*` 国际化文案统一管理标题、输入占位、思考态、采纳按钮和空状态。

## 2026-02-10 Task: WorkshopDetail 3-Panel + Mobile Tabs (Task 7)
- `WorkshopDetailView.vue` 改为桌面三栏（路线图/编辑器/AI）与移动端三标签切换，且保留初始加载 spinner。
- 节点切换流程固定为：`editorStore.flush()` -> `workshopStore.selectNode(nodeId)` -> watcher 中 `editorStore.loadDraft(draftKey)`。
- 草稿 key 统一使用 `draft:${userId}:${workshopId}:${nodeId}:v1`，当用户未登录时回退到 `guest` 防止 key 为空。
- 移动端编辑器用 `<KeepAlive>` + `v-show`，避免切 tab 销毁 Tiptap 实例导致编辑态丢失。
- 节点完成按钮放在编辑区头部，先 flush 再 complete；若 store 未自动切换，视图层兜底选中下一个 `pending` 节点。
- `workshop.tabs.*` 文案需在 `zh-CN` 与 `en-US` 同步补齐，避免移动端标签出现缺失键。

## 2026-02-10 Task: Integration Verification (Task 10)
- 端到端手工自动化链路已验证：登录 -> Dashboard -> Workshop -> 编辑器输入 -> AI 回复与采纳 -> 提交节点 -> 导出页 -> Markdown 下载 -> 个人资料保存。
- Playwright 下载事件可直接通过 `page.waitForEvent('download')` 验证，建议保留该模式作为后续导出回归测试模板。
- Dark Mode 校验应使用“先探测再切换”的策略：先检查 `document.documentElement.classList.contains('dark')`，避免因初始状态不确定导致误判。
- 移动端布局验证中，`scrollWidth > innerWidth` 是快速发现横向溢出的有效指标；本项目关键页面均通过。
- i18n 切换已覆盖导航品牌文案往返（`卷了么` <-> `JuanLeMe`），可作为全局语言状态可用性的最小验收信号。

## 2026-02-10 Task: Modularity Refactor Pass
- `WorkshopDetailView.vue` 已降为薄壳路由入口，核心逻辑迁移到 `features/workshop/detail/WorkshopDetailPage.vue`，避免视图层继续膨胀。
- 工作坊详情拆分了 3 个 composable：`useWorkshopViewport`（断点状态）、`useWorkshopDraftKey`（草稿 key 规则）、`useWorkshopDetailSession`（节点切换/完成/加载编排）。
- `AiSidebar.vue` 的会话状态机与异步请求逻辑抽离到 `useAiConversation`，组件层只保留渲染与交互绑定，后续可继续拆消息列表与输入框子组件。
- 在 `noUnusedLocals` 严格模式下，模板 `ref` 变量可能被 TS 误判未使用；可通过脚本内显式 `void listEl` 规避。

## 2026-02-10 Task: Oracle Follow-up Slices
- `useWorkshopViewport` 已补充浏览器环境保护（`window.matchMedia` 延迟到 mounted 并带 guard），避免非浏览器上下文崩溃。
- 提取 `WorkshopActionBar.vue` 统一桌面/移动端“导出 + 提交”操作区，减少 `WorkshopDetailPage.vue` 模板重复块。
- 变更后再次通过全量验证：`pnpm --filter app test -- --run`（43/43）与 `pnpm --filter app run build`（成功）。

## 2026-02-10 Task: User Profile Page
- 资料页表单用 `watch(user, { immediate: true })` 回填，能兼容首次渲染和后续 store 变更。
- 关键词输入用逗号分隔字符串，提交时再 `split/map/trim/filter` 转成 `keywords: string[]` 持久化。
- 头像上传仅做本地预览时，`URL.createObjectURL(file)` 就够用，不需要接入真实上传链路。
- `updateProfile(payload: Partial<UserProfile>)` 放在 `userStore` 内统一合并并写回 `juanleme-user`，视图层只负责传字段。
- Vitest 里把 Pinia 与 i18n 一起挂载，可直接覆盖“回显、保存、必填校验、store 持久化”四个核心用例。

## 2026-02-10 Task: Export Feature
- 导出工具建议放在 `app/src/utils/export.ts`，保持纯函数 `generateMarkdown(workshop, nodes)`，页面层只负责取数与触发下载/打印。
- `downloadMarkdown` 在 happy-dom 下可通过替换 `URL.createObjectURL/revokeObjectURL` 与拦截 `document.createElement("a")` 做可重复单测。
- 导出预览页若使用 `<script setup>`，Biome 会对模板内使用的变量误报未使用；沿用项目既有 `biome-ignore-all` 注释可避免噪音。
- 打印导出优先使用 `@media print` + `.no-print`，并在打印时强制黑白、去阴影、A4 页边距，能兼容深色主题页面。
- 路由按现有守卫规则仅需新增非 `meta.public` 路由即可自动受鉴权保护（`/workshop/:id/export`）。
