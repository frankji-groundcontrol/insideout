# JuanLeMe (卷了么) Phase 1 Continuation — Complete Mock Mode

## TL;DR

> **Quick Summary**: Complete all remaining Phase 1 (Mock Mode) features for the JuanLeMe workshop learning platform. This includes infrastructure upgrades (Vitest, Tiptap, vue-i18n, dark mode), service layer abstraction, auth flow, rich text editor with AI sidebar, roadmap node logic, profile page, and export. Every existing component also gets emoji→icon cleanup, dark mode support, and i18n text extraction.
>
> **Deliverables**:
> - Test infrastructure (Vitest + vue-test-utils + happy-dom)
> - Dark mode toggle + Tailwind dark mode across all components
> - i18n setup (vue-i18n) with EN/CN locale files
> - Emoji→icon migration on all existing components
> - Service interface contracts + mock adapters + service registry
> - Auth flow: userStore, mock login/logout, route guards
> - Tiptap rich text editor with toolbar, Markdown shortcuts, autosave
> - AI chatbot sidebar with mock responses + "adopt" insert flow
> - Roadmap node status machine (locked/pending/in-progress/completed)
> - User profile page with avatar preview + form editing
> - Export: Markdown download + Print CSS PDF
>
> **Estimated Effort**: XL (5-7 days of agent work)
> **Parallel Execution**: YES — 4 waves
> **Critical Path**: Task 0 → Task 1 → Task 2 → Task 3 → Task 5 → Task 7 → Task 9

---

## Context

### Original Request
Continue implementing JuanLeMe (卷了么), a Vue 3 workshop learning platform. Complete all remaining Phase 1 Mock Mode features so the app is fully demo-able with fake data.

### Interview Summary
**Key Discussions**:
- User wants full Rule 25 compliance: replace emojis with @heroicons, add dark mode, add i18n
- Test strategy: Vitest + TDD (RED-GREEN-REFACTOR)
- Build order: Infrastructure → Auth → Editor+AI → Roadmap → Profile → Export (Oracle-recommended)
- Service layer: Interface contracts + mock adapters, swappable to Supabase later

**Research Findings**:
- Oracle: Use JSON as Tiptap primary storage format, 800ms debounce autosave to localStorage
- Oracle: Use Markdown download + Print CSS for PDF export (avoid html2canvas+jspdf)
- Oracle: Service registry via VITE_API_MODE env variable, no heavy DI framework
- Oracle: editorStore.enqueueInsert() pattern for AI sidebar → editor communication
- Oracle: KeepAlive for mobile tab switching to preserve editor instance
- Metis: No Pinia stores exist yet — all state is local ref(), must build from scratch
- Metis: WorkshopDetailView has dual-script-block bug, must fix
- Metis: mockApi missing methods for node status, content save/load, AI chat
- Metis: LoginPage redirects to `/` not `/dashboard`, NavBar has own `isLoggedIn` ref not synced to store
- Metis: Pinia 3.x installed (not 2.x as docs say), use as-is
- Metis: Package manager is pnpm (pnpm-lock.yaml exists)

### Metis Review
**Identified Gaps** (addressed):
- Emoji vs icons discrepancy: User chose to fix all emojis → icons
- Dark mode missing: User chose to implement in Phase 1
- i18n missing: User chose to implement in Phase 1
- No Vitest installed: Added as explicit Step 0
- No Tiptap packages: Added as explicit Step 0
- Missing types (EditorDraft, AiMessage, etc.): Added to type extension task
- Missing mockApi methods: Added to service layer task
- WorkshopDetailView bug: Added to infrastructure fix task
- No RegisterPage needed: Mock login accepts any credentials
- Navigation guards: Included in auth task

---

## Work Objectives

### Core Objective
Transform JuanLeMe from a partially-built UI shell into a fully interactive, demo-ready Mock Mode application where users can experience the complete journey: login → browse workshops → follow roadmap → edit with AI assistance → export results.

### Concrete Deliverables
- `app/vitest.config.ts` + test infrastructure
- `app/src/i18n/` directory with EN/CN locale files
- Tailwind dark mode config + `ThemeToggle` component
- All existing components updated: emojis→icons, dark mode classes, i18n text
- `app/src/types/services.ts` — service interface contracts
- `app/src/services/registry.ts` — service registry
- Updated `app/src/services/mock/` with all required mock methods
- `app/src/stores/user.ts`, `workshop.ts`, `editor.ts` — Pinia stores
- `app/src/views/auth/LoginPage.vue` — updated with store integration
- `app/src/views/workshop/TaskEditor.vue` — Tiptap editor
- `app/src/components/workshop/AiSidebar.vue` — AI chat mock
- `app/src/components/workshop/RoadmapItem.vue` — node status component
- Updated `app/src/views/workshop/WorkshopDetailView.vue` — 3-panel + mobile tabs
- `app/src/views/profile/UserProfile.vue` — profile page
- `app/src/views/workshop/ExportPreview.vue` — export page
- `app/src/utils/export.ts` — export logic + print CSS

### Definition of Done
- [ ] `pnpm --filter app test -- --run` → all tests pass
- [ ] `pnpm --filter app run build` → exit code 0, no TS errors
- [ ] Full user journey works in browser: login → dashboard → workshop → edit → AI chat → export
- [ ] Dark mode toggle switches all pages correctly
- [ ] Language toggle switches all text EN↔CN
- [ ] No emojis remain in component source code (only @heroicons)
- [ ] Mobile responsive: all pages usable at 375px width

### Must Have
- Mock auth flow (any credentials work, session persists via localStorage)
- Tiptap editor with basic formatting toolbar
- AI sidebar with mock responses + adopt/insert
- Autosave to localStorage with 800ms debounce
- Roadmap node status progression
- Markdown export download
- Print-friendly CSS for PDF via window.print()
- Dark mode with toggle
- i18n EN/CN

### Must NOT Have (Guardrails)
- No separate RegisterPage.vue — mock login accepts any credentials
- No social login implementation (buttons visual only)
- No "forgot password" flow
- No real API calls or Supabase integration
- No discussion/chat area (keep "Coming Soon" placeholder)
- No workshop creation form (mock toast "Coming in Phase 2")
- No image upload service — avatar is URL.createObjectURL() preview only
- No CRDT/Yjs real-time collaboration
- No html2canvas or jspdf — Markdown + Print CSS only
- No more than 8 mock AI response templates
- Do not modify existing component visual designs unless fixing bugs
- No new npm dependencies beyond: Vitest ecosystem, Tiptap ecosystem, vue-i18n
- Single file must not exceed 500 lines

---

## Verification Strategy

> **UNIVERSAL RULE: ZERO HUMAN INTERVENTION**
>
> ALL tasks MUST be verifiable WITHOUT any human action.
> ALL verification is executed by the agent using tools (Playwright, bash, etc.).

### Test Decision
- **Infrastructure exists**: NO (will be set up in Task 0)
- **Automated tests**: YES (TDD — RED-GREEN-REFACTOR)
- **Framework**: Vitest + @vue/test-utils + happy-dom

### TDD Workflow (Per Feature Task)

Each TODO follows RED-GREEN-REFACTOR:

**Task Structure:**
1. **RED**: Write failing test first
   - Test file: `app/src/{module}/__tests__/{name}.test.ts`
   - Test command: `pnpm --filter app test -- --run {file}`
   - Expected: FAIL (test exists, implementation doesn't)
2. **GREEN**: Implement minimum code to pass
   - Command: `pnpm --filter app test -- --run {file}`
   - Expected: PASS
3. **REFACTOR**: Clean up while keeping green
   - Command: `pnpm --filter app test -- --run {file}`
   - Expected: PASS (still)

### Agent-Executed QA Scenarios

| Type | Tool | How Agent Verifies |
|------|------|-------------------|
| **Frontend/UI** | Playwright (playwright skill) | Navigate, interact, assert DOM, screenshot |
| **Build** | Bash | `pnpm --filter app run build` → exit 0 |
| **Unit Tests** | Bash | `pnpm --filter app test -- --run` → all pass |

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately):
├── Task 0: Infrastructure & Dependencies (Vitest, Tiptap, vue-i18n, types, service contracts)
└── (nothing else — everything depends on this)

Wave 2 (After Wave 1):
├── Task 1: Dark Mode + i18n Setup + Emoji Cleanup (existing components)
├── Task 2: Auth Module (userStore, login flow, route guards)
└── NOTE: Both tasks modify NavBar.vue. To avoid merge conflicts:
    - Task 1 should handle NavBar emoji/icon swap + dark mode + i18n text + ThemeToggle + LangToggle placement
    - Task 2 should handle NavBar auth state (replace local isLoggedIn ref with store, add logout button, display username)
    - SEQUENCING: Run Task 1 FIRST, then Task 2 second (not truly parallel) — or ensure Task 2 agent reads NavBar AFTER Task 1 commits

Wave 3 (After Wave 2):
├── Task 3: Service Layer & Mock API Extension
├── Task 4: workshopStore + Roadmap Node Status Logic
└── Task 5: Tiptap Editor + editorStore + Autosave
    (Task 3 must complete before 4 and 5; 4 and 5 can run in parallel)

Wave 4 (After Wave 3):
├── Task 6: AI Chatbot Sidebar + Adopt Flow
├── Task 7: WorkshopDetailView 3-Panel Refactor + Mobile Tabs
└── (Task 6 depends on Task 5 editor; Task 7 depends on Tasks 4, 5, 6)

Wave 5 (After Wave 4):
├── Task 8: Profile Page
├── Task 9: Export Feature
└── (8 and 9 are independent, both depend on Waves 1-4 being complete)

Wave 6 (Final):
└── Task 10: Integration Test & Full Journey Verification

Critical Path: 0 → 2 → 3 → 5 → 6 → 7 → 9 → 10
```

### Dependency Matrix

| Task | Depends On | Blocks | Can Parallelize With |
|------|------------|--------|---------------------|
| 0 | None | 1, 2, 3, 4, 5, 6, 7, 8, 9, 10 | None (first) |
| 1 | 0 | 7 | 2 |
| 2 | 0 | 3 | 1 |
| 3 | 2 | 4, 5 | None |
| 4 | 3 | 7 | 5 |
| 5 | 3 | 6, 7 | 4 |
| 6 | 5 | 7 | None |
| 7 | 1, 4, 5, 6 | 10 | None |
| 8 | 0, 2 | 10 | 9 |
| 9 | 0, 3, 5 | 10 | 8 |
| 10 | ALL | None | None (final) |

### Agent Dispatch Summary

| Wave | Tasks | Recommended Agents |
|------|-------|-------------------|
| 1 | 0 | category="unspecified-high" (infra setup) |
| 2 | 1, 2 | visual-engineering (dark/i18n), quick (auth) |
| 3 | 3, 4, 5 | unspecified-high (services), visual-engineering (roadmap, editor) |
| 4 | 6, 7 | visual-engineering (AI sidebar, layout refactor) |
| 5 | 8, 9 | visual-engineering (profile, export) |
| 6 | 10 | deep (integration verification) |

---

## TODOs

- [ ] 0. Infrastructure & Dependencies Setup

  **What to do**:
  - Install Vitest + @vue/test-utils + happy-dom as dev dependencies via pnpm
  - Create `app/vitest.config.ts` with happy-dom environment, path aliases matching `tsconfig.json`
  - Create example test `app/src/__tests__/example.test.ts` to verify setup works
  - Install Tiptap packages: `@tiptap/vue-3`, `@tiptap/starter-kit`, `@tiptap/pm`, `@tiptap/extension-placeholder`
  - Install vue-i18n: `vue-i18n@9`
  - Add `"test"` script to `app/package.json`: `"vitest --run"`
  - Extend `app/src/types/index.ts` with new types: `EditorDraft`, `AiMessage`, `AiConversation`, `ExportConfig`, `UserSubmission`, `ServiceRegistry`
  - Create `app/src/types/services.ts` — define TS interfaces: `IAuthService`, `IWorkshopService`, `IEditorService`, `IAiService`, `IExportService`
  - Fix `app/src/views/workshop/WorkshopDetailView.vue` dual-script-block bug — consolidate `computed` import into the single `<script setup>` block
  - Verify: `pnpm --filter app test -- --run` passes example test
  - Verify: `pnpm --filter app run build` succeeds with no TS errors

  **Must NOT do**:
  - Do not implement any features yet — this is infra only
  - Do not modify visual appearance of any component
  - Do not install unnecessary packages

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Infrastructure setup spanning multiple config files, type definitions, and dependency management
  - **Skills**: [`frontend-ui-ux`]
    - `frontend-ui-ux`: Vue 3 + Vite config expertise for Vitest/Tiptap integration

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 1 (solo)
  - **Blocks**: ALL subsequent tasks
  - **Blocked By**: None (start immediately)

  **References**:

  **Pattern References**:
  - `app/tsconfig.json` — Path alias configuration (`@/` → `src/`) that vitest.config.ts must mirror
  - `app/tsconfig.app.json` — Compiler options for Vue 3 setup
  - `app/vite.config.ts` — Current Vite config that vitest extends
  - `app/package.json` — Current dependencies and scripts structure

  **API/Type References**:
  - `app/src/types/index.ts` — Existing type definitions (User, Workshop, RoadmapNode, etc.) to extend
  - `app/src/services/mock/index.ts` — Current mockApi interface to understand what service contracts need

  **Documentation References**:
  - `docs/PLAN.md:154-196` — Database schema defining data shapes for type definitions
  - `docs/PLAN.md:26-44` — Tech stack versions

  **External References**:
  - Vitest Vue setup: https://vitest.dev/guide/
  - @vue/test-utils: https://test-utils.vuejs.org/
  - Tiptap Vue 3: https://tiptap.dev/docs/editor/getting-started/install/vue3
  - vue-i18n@9: https://vue-i18n.intlify.dev/guide/installation.html

  **WHY Each Reference Matters**:
  - `tsconfig.json` path aliases must be mirrored in vitest.config.ts or tests can't resolve `@/` imports
  - `vite.config.ts` is the base config that vitest.config extends
  - Existing `types/index.ts` must be extended (not replaced) to preserve existing type usage
  - `services/mock/index.ts` shows the current API surface that service interfaces must cover

  **Acceptance Criteria**:

  **TDD (Red-Green-Refactor):**
  - [ ] `app/vitest.config.ts` created with happy-dom environment
  - [ ] `app/src/__tests__/example.test.ts` created with passing test
  - [ ] `pnpm --filter app test -- --run` → PASS (1 test, 0 failures)

  **Agent-Executed QA Scenarios:**

  ```
  Scenario: Vitest runs and passes
    Tool: Bash
    Preconditions: Dependencies installed
    Steps:
      1. pnpm --filter app test -- --run
      2. Assert: exit code 0
      3. Assert: stdout contains "1 passed"
    Expected Result: Test infrastructure works
    Evidence: Terminal output captured

  Scenario: Build succeeds with new types
    Tool: Bash
    Preconditions: Types extended, packages installed
    Steps:
      1. pnpm --filter app run build
      2. Assert: exit code 0
      3. Assert: no "error TS" in output
    Expected Result: TypeScript compiles cleanly
    Evidence: Build output captured

  Scenario: WorkshopDetailView bug fixed
    Tool: Bash
    Preconditions: Bug fix applied
    Steps:
      1. grep -c "<script " app/src/views/workshop/WorkshopDetailView.vue
      2. Assert: result is "1" (only one script block)
    Expected Result: Single script setup block
    Evidence: grep output captured
  ```

  **Evidence to Capture:**
  - [ ] `pnpm --filter app test -- --run` output
  - [ ] `pnpm --filter app run build` output
  - [ ] grep verification of WorkshopDetailView fix

  **Commit**: YES
  - Message: `chore(infra): add Vitest, Tiptap, vue-i18n deps and service type contracts`
  - Files: `app/package.json`, `app/vitest.config.ts`, `app/src/types/index.ts`, `app/src/types/services.ts`, `app/src/__tests__/example.test.ts`, `app/src/views/workshop/WorkshopDetailView.vue`
  - Pre-commit: `pnpm --filter app test -- --run && pnpm --filter app run build`

---

- [ ] 1. Dark Mode + i18n Setup + Emoji Cleanup

  **What to do**:
  - Configure Tailwind dark mode: add `darkMode: 'class'` to `app/tailwind.config.js`
  - Create `app/src/components/common/ThemeToggle.vue` — toggle button that adds/removes `dark` class on `<html>`, persists preference to localStorage
  - Create `app/src/i18n/index.ts` — vue-i18n setup with `createI18n()`, default locale CN
  - Create `app/src/i18n/locales/zh-CN.ts` — Chinese translations for ALL existing hardcoded text
  - Create `app/src/i18n/locales/en-US.ts` — English translations
  - Create `app/src/components/common/LangToggle.vue` — language switch button
  - Register i18n plugin in `app/src/main.ts`
  - Update ALL existing components to replace emojis with `@heroicons/vue` icons:
    - `NavBar.vue`: Replace 🌯 with brand icon, add ThemeToggle + LangToggle
    - `DashboardView.vue`: Replace 👋📂👑 with Heroicons (HandRaisedIcon, FolderIcon, CrownIcon etc.)
    - `WorkshopCard.vue`: Replace 👥🟢🔴⚪️ with Heroicons (UsersIcon, status dot/badge)
    - `RoadmapSidebar.vue`: Replace 🗺️📝📤 with Heroicons
    - `TaskDetail.vue`: Replace emojis with Heroicons
    - `LoginPage.vue`: Replace any emojis
    - `AppFooter.vue`: Clean up
  - Add `dark:` Tailwind variants to ALL existing components for dark mode support:
    - Background colors: `bg-white dark:bg-gray-900`, `bg-gray-50 dark:bg-gray-800`
    - Text colors: `text-gray-900 dark:text-gray-100`, `text-gray-600 dark:text-gray-400`
    - Border colors: `border-gray-200 dark:border-gray-700`
    - Card/surface: `bg-white dark:bg-gray-800`
    - Input fields: `bg-white dark:bg-gray-700` with `text-gray-900 dark:text-gray-100`
  - Extract ALL hardcoded Chinese text from existing components into i18n locale keys
  - Write tests for ThemeToggle (toggles class) and LangToggle (switches locale)

  **Must NOT do**:
  - Do not redesign any component layout — only swap emojis→icons, add dark: classes, extract i18n text
  - Do not add complex theme system beyond Tailwind dark mode class strategy
  - Do not add more than EN and CN locales

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: Heavy UI/styling work across all components, dark mode design, icon selection
  - **Skills**: [`frontend-ui-ux`]
    - `frontend-ui-ux`: Expert at TailwindCSS dark mode patterns, icon systems, UI consistency

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Task 2)
  - **Blocks**: Task 7 (WorkshopDetailView refactor)
  - **Blocked By**: Task 0

  **References**:

  **Pattern References**:
  - `app/src/components/layout/NavBar.vue` — Current emoji usage + component structure pattern to follow
  - `app/src/views/dashboard/DashboardView.vue` — Current emoji usage in dashboard
  - `app/src/components/workshop/WorkshopCard.vue` — Card styling pattern + emoji status badges
  - `app/src/components/workshop/RoadmapSidebar.vue` — Sidebar with emoji icons
  - `app/src/components/workshop/TaskDetail.vue` — Task display with emojis
  - `app/src/views/auth/LoginPage.vue` — Full-screen split layout as dark mode reference
  - `app/src/components/common/BaseButton.vue` — Base component pattern to follow for ThemeToggle/LangToggle
  - `app/src/main.ts` — Plugin registration pattern (createPinia, createRouter → add createI18n)

  **External References**:
  - Tailwind dark mode: https://tailwindcss.com/docs/dark-mode
  - vue-i18n composition API: https://vue-i18n.intlify.dev/guide/advanced/composition.html
  - Heroicons catalog: https://heroicons.com/

  **WHY Each Reference Matters**:
  - Each existing component file is needed to identify all emoji instances and hardcoded Chinese text
  - BaseButton.vue shows the component conventions (script setup + defineProps) for new toggle components
  - main.ts shows how plugins are registered (add i18n the same way Pinia and Router are added)

  **Acceptance Criteria**:

  **TDD:**
  - [ ] Test: ThemeToggle click adds `dark` class to document.documentElement
  - [ ] Test: ThemeToggle persists preference to localStorage
  - [ ] Test: LangToggle switches i18n locale between 'zh-CN' and 'en-US'
  - [ ] `pnpm --filter app test -- --run src/components/__tests__/ThemeToggle.test.ts` → PASS
  - [ ] `pnpm --filter app test -- --run src/components/__tests__/LangToggle.test.ts` → PASS

  **Agent-Executed QA Scenarios:**

  ```
  Scenario: Dark mode toggle works on dashboard
    Tool: Playwright (playwright skill)
    Preconditions: Dev server running on localhost:5173
    Steps:
      1. Navigate to: http://localhost:5173/
      2. Assert: html element does NOT have class "dark"
      3. Click: [data-testid="theme-toggle"] button
      4. Assert: html element HAS class "dark"
      5. Assert: body background color is dark (rgb close to gray-900)
      6. Screenshot: .sisyphus/evidence/task-1-dark-mode-on.png
      7. Click: [data-testid="theme-toggle"] again
      8. Assert: html element does NOT have class "dark"
      9. Screenshot: .sisyphus/evidence/task-1-dark-mode-off.png
    Expected Result: Dark mode toggles correctly with visible style changes
    Evidence: .sisyphus/evidence/task-1-dark-mode-on.png, task-1-dark-mode-off.png

  Scenario: Language toggle switches all text
    Tool: Playwright (playwright skill)
    Preconditions: Dev server running
    Steps:
      1. Navigate to: http://localhost:5173/
      2. Assert: page contains Chinese text (e.g., "工作坊" or similar)
      3. Click: [data-testid="lang-toggle"] button
      4. Assert: Chinese text replaced with English equivalents
      5. Screenshot: .sisyphus/evidence/task-1-lang-en.png
      6. Click: [data-testid="lang-toggle"] again
      7. Assert: English text replaced back to Chinese
    Expected Result: All UI text switches between EN and CN
    Evidence: .sisyphus/evidence/task-1-lang-en.png

  Scenario: No emojis remain in source
    Tool: Bash
    Preconditions: Emoji cleanup complete
    Steps:
      1. grep -rP "[\x{1F300}-\x{1F9FF}]" app/src/components/ app/src/views/ --include="*.vue" || true
      2. Assert: no matches found (exit code 1 = no matches)
    Expected Result: Zero emoji characters in Vue source files
    Evidence: grep output captured

  Scenario: Build succeeds with dark mode + i18n
    Tool: Bash
    Steps:
      1. pnpm --filter app run build
      2. Assert: exit code 0
    Expected Result: Clean build
    Evidence: Build output
  ```

  **Commit**: YES
  - Message: `feat(ui): add dark mode, i18n (EN/CN), replace emojis with Heroicons`
  - Files: `app/tailwind.config.js`, `app/src/i18n/**`, `app/src/components/common/ThemeToggle.vue`, `app/src/components/common/LangToggle.vue`, `app/src/main.ts`, all updated component files
  - Pre-commit: `pnpm --filter app test -- --run && pnpm --filter app run build`

---

- [ ] 2. Auth Module (userStore + Login Flow + Route Guards)

  **What to do**:
  - **RED**: Write tests first for userStore:
    - `userStore.login(email, password)` → sets `isAuthenticated = true`, stores user profile, saves token to localStorage
    - `userStore.logout()` → clears user, removes token, `isAuthenticated = false`
    - `userStore.initFromStorage()` → restores session from localStorage on app load
    - Mock login accepts ANY email/password combo (mock mode)
  - **GREEN**: Create `app/src/stores/user.ts` — Pinia store with:
    - State: `user: UserProfile | null`, `isAuthenticated: boolean`, `token: string | null`
    - Actions: `login(email, password)`, `logout()`, `initFromStorage()`
    - Login creates a mock user profile from the email, generates a fake token, saves to localStorage
  - Implement router navigation guards in `app/src/router/index.ts`:
    - `router.beforeEach`: if route is not `/login` and not authenticated → redirect to `/login`
    - If route is `/login` and already authenticated → redirect to `/dashboard`
  - Update `app/src/views/auth/LoginPage.vue`:
    - Replace `setTimeout` mock with `userStore.login()` call
    - Redirect to `/dashboard` on success (not `/`)
    - Show error toast on failure (even mock mode — for demo purposes)
    - Remove social login button click handlers (keep visual only)
    - Keep "Forgot password?" as non-functional link
  - Update `app/src/components/layout/NavBar.vue`:
    - Replace local `isLoggedIn` ref with `userStore.isAuthenticated`
    - Display `userStore.user.username` in the avatar area
    - Add logout button that calls `userStore.logout()`
  - Update `app/src/App.vue` to call `userStore.initFromStorage()` on mount
  - Fix route config: ensure `/dashboard` route exists and points to DashboardView

  **Must NOT do**:
  - No RegisterPage.vue — login page mock accepts any credentials
  - No social login implementation
  - No forgot password flow
  - No real authentication — everything is mock localStorage

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Auth mock is straightforward store + route guard logic, well-defined scope
  - **Skills**: [`frontend-ui-ux`]
    - `frontend-ui-ux`: Vue Router guards, Pinia store patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Task 1)
  - **Blocks**: Task 3 (service layer uses userStore.user.id for mock data)
  - **Blocked By**: Task 0

  **References**:

  **Pattern References**:
  - `app/src/views/auth/LoginPage.vue` — Current login implementation to refactor (setTimeout on line ~85-95)
  - `app/src/components/layout/NavBar.vue:7` — Local `isLoggedIn` ref to replace with store
  - `app/src/router/index.ts` — Current route config to add guards and fix paths
  - `app/src/main.ts` — App initialization where `initFromStorage()` should be called
  - `app/src/App.vue` — Root component for initialization lifecycle

  **API/Type References**:
  - `app/src/types/index.ts:UserProfile` — User type shape for store state
  - `app/src/types/services.ts:IAuthService` — (created in Task 0) Auth service interface

  **WHY Each Reference Matters**:
  - LoginPage.vue has the existing mock login code that needs replacing with store-based flow
  - NavBar.vue has a local `isLoggedIn` ref that must be migrated to store consumption
  - router/index.ts needs navigation guards added to the existing route definitions

  **Acceptance Criteria**:

  **TDD:**
  - [ ] Test: `userStore.login('test@test.com', 'any')` → `isAuthenticated === true`
  - [ ] Test: `userStore.login()` stores user profile with username derived from email
  - [ ] Test: `userStore.logout()` → `isAuthenticated === false`, user is null
  - [ ] Test: `userStore.initFromStorage()` restores session from localStorage
  - [ ] Test: router guard redirects unauthenticated user to /login
  - [ ] `pnpm --filter app test -- --run src/stores/__tests__/user.test.ts` → PASS

  **Agent-Executed QA Scenarios:**

  ```
  Scenario: Login with any credentials succeeds
    Tool: Playwright (playwright skill)
    Preconditions: Dev server running on localhost:5173
    Steps:
      1. Navigate to: http://localhost:5173/login
      2. Wait for: input[name="email"] visible (timeout: 5s)
      3. Fill: input[name="email"] → "demo@juanleme.com"
      4. Fill: input[name="password"] → "anything123"
      5. Click: button[type="submit"]
      6. Wait for: navigation to /dashboard (timeout: 5s)
      7. Assert: URL contains "/dashboard"
      8. Assert: NavBar shows username text
      9. Screenshot: .sisyphus/evidence/task-2-login-success.png
    Expected Result: Redirected to dashboard with user info in NavBar
    Evidence: .sisyphus/evidence/task-2-login-success.png

  Scenario: Unauthenticated access redirects to login
    Tool: Playwright (playwright skill)
    Steps:
      1. Clear localStorage (execute JS: localStorage.clear())
      2. Navigate to: http://localhost:5173/dashboard
      3. Wait for: URL change (timeout: 5s)
      4. Assert: URL contains "/login"
      5. Screenshot: .sisyphus/evidence/task-2-auth-guard.png
    Expected Result: Redirected to login page
    Evidence: .sisyphus/evidence/task-2-auth-guard.png

  Scenario: Session persists across page refresh
    Tool: Playwright (playwright skill)
    Steps:
      1. Login with any credentials (from Scenario 1)
      2. Reload page
      3. Assert: URL still /dashboard (not redirected to /login)
      4. Assert: NavBar still shows username
    Expected Result: Session survives refresh
    Evidence: .sisyphus/evidence/task-2-session-persist.png

  Scenario: Logout clears session
    Tool: Playwright (playwright skill)
    Steps:
      1. Login first
      2. Click: logout button in NavBar
      3. Wait for: navigation to /login (timeout: 5s)
      4. Assert: URL contains "/login"
      5. Navigate to: /dashboard
      6. Assert: redirected back to /login
    Expected Result: Fully logged out, can't access protected routes
    Evidence: .sisyphus/evidence/task-2-logout.png
  ```

  **Commit**: YES
  - Message: `feat(auth): add userStore, mock login/logout, route guards`
  - Files: `app/src/stores/user.ts`, `app/src/stores/__tests__/user.test.ts`, `app/src/router/index.ts`, `app/src/views/auth/LoginPage.vue`, `app/src/components/layout/NavBar.vue`, `app/src/App.vue`
  - Pre-commit: `pnpm --filter app test -- --run && pnpm --filter app run build`

---

- [ ] 3. Service Layer & Mock API Extension

  **What to do**:
  - Create `app/src/services/registry.ts` — service registry that reads `VITE_API_MODE` env var and returns mock or real service implementations
  - Update `app/.env` (or create `app/.env.development`) with `VITE_API_MODE=mock`
  - Implement all mock service adapters conforming to interfaces from `types/services.ts`:
    - `app/src/services/mock/authService.ts` — implements IAuthService
    - `app/src/services/mock/workshopService.ts` — implements IWorkshopService (includes: getWorkshops, getWorkshop, getRoadmap, updateNodeStatus, getSubmission, saveSubmission)
    - `app/src/services/mock/editorService.ts` — implements IEditorService (loadDraft, saveDraft from localStorage)
    - `app/src/services/mock/aiService.ts` — implements IAiService (reply with keyword matching + preset templates)
    - `app/src/services/mock/exportService.ts` — implements IExportService (generateMarkdown, generatePrintHTML)
  - Update `app/src/services/mock/data.ts` — add mock data for: user submissions (draft content per node), AI conversation templates (5-8 templates with keywords)
  - Update `app/src/services/mock/index.ts` — re-export through registry pattern
  - Write TDD tests for each mock service (basic contract verification)
  - Update existing components that import `mockApi` directly to use `services.workshop` / `services.auth` etc. via registry

  **Must NOT do**:
  - No Supabase implementation — only mock adapters
  - No more than 8 AI response templates
  - Do not change existing mock data structure — extend it

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Service abstraction layer design requires careful interface implementation and testing
  - **Skills**: [`frontend-ui-ux`]
    - `frontend-ui-ux`: Vue 3 service patterns, TypeScript interface implementations

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3 (sequential, after Wave 2)
  - **Blocks**: Tasks 4, 5
  - **Blocked By**: Task 2 (needs userStore for user ID in mock services)

  **References**:

  **Pattern References**:
  - `app/src/services/mock/index.ts` — Current mockApi that needs to be wrapped/extended
  - `app/src/services/mock/data.ts` — Current mock data to extend with submissions + AI templates
  - `app/src/views/dashboard/DashboardView.vue:27-40` — Current direct mockApi usage pattern to migrate

  **API/Type References**:
  - `app/src/types/services.ts` — (created in Task 0) All service interfaces to implement
  - `app/src/types/index.ts` — Data types that services operate on

  **Documentation References**:
  - `docs/PLAN.md:154-196` — Database schema showing the full data relationships

  **WHY Each Reference Matters**:
  - `services/mock/index.ts` is the existing API surface that the new registry must be backward-compatible with
  - `services/mock/data.ts` has the mock datasets that need new fields (submissions, AI templates)
  - DashboardView shows how existing code calls mockApi — all such call sites need migration to registry

  **Acceptance Criteria**:

  **TDD:**
  - [ ] Test: `services.workshop.getWorkshops()` returns array of workshops
  - [ ] Test: `services.workshop.saveSubmission(nodeId, content)` persists and `getSubmission()` retrieves
  - [ ] Test: `services.editor.saveDraft(key, doc)` → `loadDraft(key)` round-trips correctly
  - [ ] Test: `services.ai.reply("项目灵感")` returns a non-empty string after delay
  - [ ] Test: `services.export.generateMarkdown(workshopId)` returns valid markdown string
  - [ ] `pnpm --filter app test -- --run src/services/__tests__/` → PASS

  **Agent-Executed QA Scenarios:**

  ```
  Scenario: Service registry resolves mock services
    Tool: Bash
    Steps:
      1. pnpm --filter app test -- --run src/services/__tests__/registry.test.ts
      2. Assert: exit code 0
      3. Assert: output contains "PASS"
    Expected Result: All mock services correctly registered
    Evidence: Test output

  Scenario: Dashboard still works with service registry
    Tool: Playwright (playwright skill)
    Preconditions: Dev server running, user logged in
    Steps:
      1. Navigate to: http://localhost:5173/dashboard
      2. Wait for: workshop cards visible (timeout: 5s)
      3. Assert: at least 3 workshop cards displayed
      4. Screenshot: .sisyphus/evidence/task-3-dashboard-works.png
    Expected Result: Dashboard loads normally with registry-based services
    Evidence: .sisyphus/evidence/task-3-dashboard-works.png
  ```

  **Commit**: YES
  - Message: `refactor(services): add service registry pattern with mock adapters`
  - Files: `app/src/services/registry.ts`, `app/src/services/mock/*.ts`, `app/.env.development`, updated view files
  - Pre-commit: `pnpm --filter app test -- --run && pnpm --filter app run build`

---

- [ ] 4. workshopStore + Roadmap Node Status Logic

  **What to do**:
  - **RED**: Write tests for workshopStore:
    - `workshopStore.loadWorkshop(id)` loads workshop + roadmap via service
    - `workshopStore.selectNode(nodeId)` sets current active node
    - `workshopStore.completeNode(nodeId)` transitions node status: in_progress → completed, unlocks next node
    - Node status state machine: `locked → pending → in_progress → completed`
  - **GREEN**: Create `app/src/stores/workshop.ts` — Pinia store with:
    - State: `currentWorkshop`, `roadmapNodes`, `currentNodeId`, `loading`
    - Actions: `loadWorkshop(id)`, `selectNode(nodeId)`, `completeNode(nodeId)`, `getNodeStatus(nodeId)`
    - Node status logic: first node starts as `pending`, rest as `locked`. When user selects a `pending` node, it becomes `in_progress`. Submitting makes it `completed` and next node becomes `pending`.
  - Create or update `app/src/components/workshop/RoadmapItem.vue`:
    - Visual status indicators using Heroicons + color:
      - `locked`: gray, LockClosedIcon
      - `pending`: blue outline, CircleIcon
      - `in_progress`: indigo solid, PlayIcon
      - `completed`: green, CheckCircleIcon
    - Click handler: only pending/in_progress nodes are clickable
    - Dark mode support
    - i18n text
  - Update `app/src/components/workshop/RoadmapSidebar.vue` to use RoadmapItem components + workshopStore

  **Must NOT do**:
  - No complex node dependency graphs — simple linear progression only
  - No drag-and-drop reordering (that's organizer mode, Phase 2)
  - No animations beyond simple transitions

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: State machine logic + visual status indicators + responsive component design
  - **Skills**: [`frontend-ui-ux`]
    - `frontend-ui-ux`: Vue component design, Tailwind status styling, Pinia store patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Task 5)
  - **Blocks**: Task 7 (WorkshopDetailView refactor)
  - **Blocked By**: Task 3 (needs service layer for data loading)

  **References**:

  **Pattern References**:
  - `app/src/components/workshop/RoadmapSidebar.vue` — Existing sidebar to update with new RoadmapItem components
  - `app/src/components/workshop/WorkshopCard.vue` — Status badge styling pattern (green/yellow/red) to follow
  - `app/src/services/mock/data.ts` — Mock roadmap node data structure

  **API/Type References**:
  - `app/src/types/index.ts:RoadmapNode` — Node type with status field
  - `app/src/types/services.ts:IWorkshopService` — Service methods for node operations

  **Documentation References**:
  - `docs/PLAN.md:76-81` — Roadmap view interaction specification
  - `docs/PLAN.md:186-191` — roadmap_nodes schema

  **WHY Each Reference Matters**:
  - RoadmapSidebar is the container that will host the new RoadmapItem components
  - WorkshopCard shows the existing color/badge conventions for status indicators
  - PLAN.md:76-81 defines the exact UX behavior (participant vs organizer, click behavior)

  **Acceptance Criteria**:

  **TDD:**
  - [ ] Test: `workshopStore.loadWorkshop(id)` populates currentWorkshop and roadmapNodes
  - [ ] Test: First node starts as `pending`, rest as `locked`
  - [ ] Test: `selectNode(pendingNodeId)` → status becomes `in_progress`
  - [ ] Test: `completeNode(id)` → status `completed`, next node → `pending`
  - [ ] Test: Cannot select a `locked` node (no state change)
  - [ ] `pnpm --filter app test -- --run src/stores/__tests__/workshop.test.ts` → PASS

  **Agent-Executed QA Scenarios:**

  ```
  Scenario: Roadmap shows correct node status indicators
    Tool: Playwright (playwright skill)
    Preconditions: Dev server running, user logged in, navigated to a workshop
    Steps:
      1. Navigate to: http://localhost:5173/workshop/{mockWorkshopId}
      2. Wait for: roadmap sidebar visible (timeout: 5s)
      3. Assert: first node has status indicator (blue/indigo = pending/in_progress)
      4. Assert: subsequent nodes have gray/locked indicator
      5. Screenshot: .sisyphus/evidence/task-4-roadmap-status.png
    Expected Result: Visual status indicators match node state
    Evidence: .sisyphus/evidence/task-4-roadmap-status.png

  Scenario: Clicking a locked node does nothing
    Tool: Playwright (playwright skill)
    Steps:
      1. Find a locked (gray) node in the roadmap
      2. Click it
      3. Assert: no navigation or state change (right panel unchanged)
    Expected Result: Locked nodes are non-interactive
    Evidence: .sisyphus/evidence/task-4-locked-node.png
  ```

  **Commit**: YES
  - Message: `feat(roadmap): add workshopStore with node status state machine and RoadmapItem component`
  - Files: `app/src/stores/workshop.ts`, `app/src/stores/__tests__/workshop.test.ts`, `app/src/components/workshop/RoadmapItem.vue`, `app/src/components/workshop/RoadmapSidebar.vue`
  - Pre-commit: `pnpm --filter app test -- --run && pnpm --filter app run build`

---

- [ ] 5. Tiptap Rich Text Editor + editorStore + Autosave

  **What to do**:
  - **RED**: Write tests for editorStore:
    - `editorStore.setContent(jsonContent)` updates state and triggers debounced save
    - `editorStore.loadDraft(key)` restores content from service
    - `editorStore.flush()` immediately saves pending changes (called on node switch)
    - `editorStore.enqueueInsert(text)` queues text for AI adoption
    - `editorStore.dequeueInsert()` returns and removes queued text
    - `editorStore.isDirty` computed tracks unsaved changes
  - **GREEN**: Create `app/src/stores/editor.ts` — Pinia store with:
    - State: `content: JSONContent | null`, `isDirty: boolean`, `insertQueue: string[]`, `currentDraftKey: string`
    - Actions: `setContent()`, `loadDraft(key)`, `saveDraft()`, `flush()`, `enqueueInsert(text)`, `dequeueInsert()`
    - Autosave: 800ms debounce after `setContent()` calls `services.editor.saveDraft()`
    - Draft key format: `draft:${userId}:${workshopId}:${nodeId}:v1`
    - **IMPORTANT (Oracle review)**: Add `draftRevision: number` to state. Increment on every `setContent()`. When debounced save fires, compare stored revision with current — skip save if stale (prevents race conditions with rapid node switching).
    - **Sync Direction**: ONE-WAY only — Tiptap is runtime source-of-truth, Pinia is snapshot cache. Never write back from store to editor (except on initial load/node switch).
  - Create `app/src/views/workshop/TaskEditor.vue`:
    - Integrate `@tiptap/vue-3` with `StarterKit` extension
    - Custom toolbar component with buttons: Bold, Italic, Heading (1-3), Bullet List, Ordered List, Code Block, Blockquote, Horizontal Rule, Undo, Redo
    - Markdown shortcuts enabled via StarterKit (type `#` for H1, `*` for list, etc.)
    - `@tiptap/extension-placeholder` for empty state hint text
    - Editor `onUpdate` → `editorStore.setContent(editor.getJSON())`
    - On mount: load draft via `editorStore.loadDraft(key)` → set editor content
    - Watch for `editorStore.insertQueue` changes → dequeue and insert at cursor position
    - Dark mode styling for editor content area
    - i18n for toolbar tooltips
    - Responsive: full width on mobile, panel width on desktop
  - Create `app/src/components/workshop/EditorToolbar.vue` — extracted toolbar component for reuse

  **Must NOT do**:
  - No image upload in editor (Phase 2)
  - No collaborative editing (no Yjs/CRDT)
  - No custom Tiptap extensions beyond StarterKit + Placeholder
  - Do not exceed 500 lines per file — split toolbar into separate component

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: Complex UI component with rich text editing, toolbar design, responsive layout
  - **Skills**: [`frontend-ui-ux`]
    - `frontend-ui-ux`: Tiptap integration expertise, editor UX patterns, Tailwind styling

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Task 4)
  - **Blocks**: Task 6 (AI sidebar), Task 7 (layout refactor), Task 9 (export)
  - **Blocked By**: Task 3 (needs service layer for draft save/load)

  **References**:

  **Pattern References**:
  - `app/src/views/workshop/WorkshopDetailView.vue` — Container that will host the editor (right panel)
  - `app/src/components/workshop/TaskDetail.vue` — Current task display with textarea to be replaced by Tiptap
  - `app/src/components/common/BaseButton.vue` — Button component pattern for toolbar buttons

  **API/Type References**:
  - `app/src/types/index.ts:EditorDraft` — (created in Task 0) Draft type definition
  - `app/src/types/services.ts:IEditorService` — (created in Task 0) Editor service interface

  **External References**:
  - Tiptap Vue 3 setup: https://tiptap.dev/docs/editor/getting-started/install/vue3
  - Tiptap StarterKit: https://tiptap.dev/docs/editor/extensions/functionality/starterkit
  - Tiptap JSON content: https://tiptap.dev/docs/editor/guide/output#option-1-json

  **WHY Each Reference Matters**:
  - WorkshopDetailView is where the editor lives — understand its layout constraints
  - TaskDetail.vue has the textarea that Tiptap replaces — understand the current submit flow
  - Tiptap docs for correct Vue 3 integration with `useEditor()` composable

  **Acceptance Criteria**:

  **TDD:**
  - [ ] Test: `editorStore.setContent(json)` updates content state
  - [ ] Test: `editorStore.setContent()` triggers saveDraft after 800ms (use fake timers)
  - [ ] Test: `editorStore.flush()` saves immediately without waiting for debounce
  - [ ] Test: `editorStore.loadDraft(key)` restores content from service
  - [ ] Test: `editorStore.enqueueInsert("hello")` → `dequeueInsert()` returns "hello"
  - [ ] `pnpm --filter app test -- --run src/stores/__tests__/editor.test.ts` → PASS

  **Agent-Executed QA Scenarios:**

  ```
  Scenario: Tiptap editor loads and accepts input
    Tool: Playwright (playwright skill)
    Preconditions: Dev server running, user logged in, navigated to workshop detail
    Steps:
      1. Navigate to workshop detail page with active node
      2. Wait for: .ProseMirror editor visible (timeout: 5s)
      3. Click into editor area
      4. Type: "This is a test paragraph"
      5. Assert: editor contains typed text
      6. Screenshot: .sisyphus/evidence/task-5-editor-typing.png
    Expected Result: Editor accepts and displays text input
    Evidence: .sisyphus/evidence/task-5-editor-typing.png

  Scenario: Toolbar formatting works
    Tool: Playwright (playwright skill)
    Steps:
      1. Select text in editor
      2. Click: Bold toolbar button
      3. Assert: selected text wrapped in <strong> (or visual bold)
      4. Click: Heading 1 toolbar button
      5. Assert: line becomes an h1
      6. Screenshot: .sisyphus/evidence/task-5-toolbar.png
    Expected Result: Toolbar buttons apply formatting
    Evidence: .sisyphus/evidence/task-5-toolbar.png

  Scenario: Markdown shortcuts work
    Tool: Playwright (playwright skill)
    Steps:
      1. Click at beginning of new line in editor
      2. Type: "# Heading One" + Enter
      3. Assert: text rendered as h1 heading
      4. Type: "- list item" + Enter
      5. Assert: bullet list created
    Expected Result: Markdown shortcuts auto-format
    Evidence: .sisyphus/evidence/task-5-markdown.png

  Scenario: Autosave persists content
    Tool: Playwright (playwright skill)
    Steps:
      1. Type text in editor
      2. Wait 1 second (for debounce to fire)
      3. Execute JS: check localStorage for draft key matching pattern "draft:*:v1"
      4. Assert: localStorage entry exists and contains typed text in JSON
      5. Reload page
      6. Wait for editor to load
      7. Assert: editor contains previously typed text
    Expected Result: Content auto-saved and restored
    Evidence: .sisyphus/evidence/task-5-autosave.png
  ```

  **Commit**: YES
  - Message: `feat(editor): integrate Tiptap rich text editor with autosave and editorStore`
  - Files: `app/src/stores/editor.ts`, `app/src/stores/__tests__/editor.test.ts`, `app/src/views/workshop/TaskEditor.vue`, `app/src/components/workshop/EditorToolbar.vue`
  - Pre-commit: `pnpm --filter app test -- --run && pnpm --filter app run build`

---

- [ ] 6. AI Chatbot Sidebar + Adopt Flow

  **What to do**:
  - Create `app/src/components/workshop/AiSidebar.vue`:
    - Chat-style UI: message list (user messages left-aligned, AI right-aligned), input field at bottom
    - User types message → mock AI responds after 500-1500ms random delay
    - AI responses come from keyword-matching mock service (5-8 templates)
    - Each AI response has an "采纳" (Adopt) button
    - Click "采纳" → calls `editorStore.enqueueInsert(messageText)` → text appears in editor at cursor
    - Loading state: typing indicator ("AI 正在思考..." / "AI is thinking...")
    - Conversation history stored in component local state (per-node, resets on node switch)
    - Dark mode styling
    - i18n text
    - Responsive: on mobile this is a tab, on desktop a side panel
  - Create mock AI response templates in `app/src/services/mock/aiService.ts`:
    - Template 1: Keywords ["灵感", "想法", "inspiration"] → creative brainstorming response
    - Template 2: Keywords ["产品", "功能", "feature"] → product feature suggestions
    - Template 3: Keywords ["设计", "UI", "界面"] → design recommendations
    - Template 4: Keywords ["技术", "代码", "code"] → technical guidance
    - Template 5: Keywords ["文案", "copy", "写"] → copywriting help
    - Template 6: Default fallback → general encouragement
  - Write tests for AI service mock and adopt flow

  **Must NOT do**:
  - No real AI/LLM calls
  - No more than 8 response templates
  - No conversation persistence across sessions
  - No complex NLP or keyword classification
  - Do not import editor component directly — communicate via editorStore only

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: Chat UI design, message bubbles, loading states, responsive layout
  - **Skills**: [`frontend-ui-ux`]
    - `frontend-ui-ux`: Chat UI patterns, animation/transitions, component communication

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (after editor exists)
  - **Blocks**: Task 7 (WorkshopDetailView refactor)
  - **Blocked By**: Task 5 (needs editorStore.enqueueInsert for adopt flow)

  **References**:

  **Pattern References**:
  - `app/src/stores/editor.ts` — (created in Task 5) enqueueInsert/dequeueInsert API for adopt flow
  - `app/src/components/common/BaseInput.vue` — Input component pattern for chat input field
  - `app/src/components/common/BaseButton.vue` — Button pattern for "adopt" button

  **API/Type References**:
  - `app/src/types/index.ts:AiMessage, AiConversation` — (created in Task 0) Message types
  - `app/src/types/services.ts:IAiService` — AI service interface with `reply()` method

  **Documentation References**:
  - `docs/PLAN.md:85-91` — AI chatbot interaction specification
  - `docs/INIT.md:40` — "在回答过程中与接入大模型的chatbot交互，获得书写灵感"

  **WHY Each Reference Matters**:
  - editorStore is the communication bridge — AI sidebar pushes, editor dequeues
  - PLAN.md:85-91 defines the exact UX: sidebar provides suggestions, one-click insert into editor

  **Acceptance Criteria**:

  **TDD:**
  - [ ] Test: `services.ai.reply("产品灵感")` returns string containing product-related content
  - [ ] Test: `services.ai.reply("random gibberish")` returns fallback response
  - [ ] Test: AI response delay is between 500-1500ms (mock timer)
  - [ ] `pnpm --filter app test -- --run src/services/__tests__/aiService.test.ts` → PASS

  **Agent-Executed QA Scenarios:**

  ```
  Scenario: AI responds to user message
    Tool: Playwright (playwright skill)
    Preconditions: Dev server running, user in workshop with editor visible
    Steps:
      1. Navigate to AI sidebar (click AI tab on mobile, or see side panel on desktop)
      2. Wait for: chat input field visible
      3. Type in chat input: "给我一些产品灵感"
      4. Click send button or press Enter
      5. Wait for: AI response message appears (timeout: 3s)
      6. Assert: response message is non-empty
      7. Assert: "采纳" button visible on AI message
      8. Screenshot: .sisyphus/evidence/task-6-ai-response.png
    Expected Result: AI responds with relevant content
    Evidence: .sisyphus/evidence/task-6-ai-response.png

  Scenario: Adopt inserts text into editor
    Tool: Playwright (playwright skill)
    Steps:
      1. After AI response appears (from previous scenario)
      2. Click: "采纳" button on AI response
      3. Switch to editor tab/panel
      4. Assert: editor now contains the AI response text
      5. Screenshot: .sisyphus/evidence/task-6-adopt-insert.png
    Expected Result: AI text inserted into editor
    Evidence: .sisyphus/evidence/task-6-adopt-insert.png

  Scenario: Loading indicator during AI response
    Tool: Playwright (playwright skill)
    Steps:
      1. Send a message in AI chat
      2. Immediately assert: loading/typing indicator visible
      3. Wait for response (timeout: 3s)
      4. Assert: loading indicator gone, response visible
    Expected Result: Typing indicator shown during AI "thinking"
    Evidence: .sisyphus/evidence/task-6-loading.png
  ```

  **Commit**: YES
  - Message: `feat(ai): add AI chatbot sidebar with mock responses and adopt-to-editor flow`
  - Files: `app/src/components/workshop/AiSidebar.vue`, `app/src/services/mock/aiService.ts`, tests
  - Pre-commit: `pnpm --filter app test -- --run && pnpm --filter app run build`

---

- [ ] 7. WorkshopDetailView 3-Panel Refactor + Mobile Tabs

  **What to do**:
  - Refactor `app/src/views/workshop/WorkshopDetailView.vue`:
    - **Desktop (md+)**: 3-column layout
      - Left: RoadmapSidebar (w-64 or w-72)
      - Center: TaskEditor (flex-1, main content area)
      - Right: AiSidebar (w-80)
    - **Mobile (<md)**: Tab-based navigation
      - 3 tabs at top: "路线图" / "编辑器" / "AI 助手"
      - Only active tab content visible
      - Use `<KeepAlive>` around editor tab to preserve Tiptap instance state
      - **IMPORTANT (Oracle review)**: Do NOT destroy Tiptap editor on `deactivated` lifecycle. Only destroy on `unmounted`. Re-focus editor on `activated` to restore editing UX. If KeepAlive still causes ProseMirror issues, fallback to CSS `display:none/block` for panel switching instead of component swap.
    - Wire up workshopStore: load workshop on mount, select first pending node
    - Wire up node selection: clicking RoadmapItem → loads content into editor
    - Wire up node completion: editor submit button → `workshopStore.completeNode()` → next node activates
    - **Important**: Flush editorStore on node switch (`editorStore.flush()` before loading new content)
    - Dark mode support for layout containers
    - i18n for tab labels
    - Remove old TaskDetail component usage (replaced by TaskEditor)
  - Update `app/src/router/index.ts` if needed for workshop route params

  **Must NOT do**:
  - Do not add drag-and-drop for roadmap reordering
  - Do not add organizer "edit mode"
  - Do not add animations beyond basic tab transitions

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: Complex responsive layout refactor, 3-panel design, mobile tab switching
  - **Skills**: [`frontend-ui-ux`]
    - `frontend-ui-ux`: Responsive layout design, Vue KeepAlive patterns, panel composition

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (after all panel components exist)
  - **Blocks**: Task 10 (integration test)
  - **Blocked By**: Tasks 1 (dark mode), 4 (roadmap), 5 (editor), 6 (AI sidebar)

  **References**:

  **Pattern References**:
  - `app/src/views/workshop/WorkshopDetailView.vue` — Current 2-panel layout to refactor
  - `app/src/components/workshop/RoadmapSidebar.vue` — Left panel (updated in Task 4)
  - `app/src/views/workshop/TaskEditor.vue` — Center panel (created in Task 5)
  - `app/src/components/workshop/AiSidebar.vue` — Right panel (created in Task 6)

  **API/Type References**:
  - `app/src/stores/workshop.ts` — Workshop store for data loading and node selection
  - `app/src/stores/editor.ts` — Editor store for flush on node switch

  **Documentation References**:
  - `docs/PLAN.md:85-91` — Task editor layout specification
  - `docs/PLAN.md:200-209` — Responsive breakpoints (md: 768px, lg: 1024px)
  - `docs/design/01_dashboard_wireframe.md:72-82` — Layout rules (full-width, no max-w constraints)

  **WHY Each Reference Matters**:
  - WorkshopDetailView.vue is THE file being refactored — must understand its current structure deeply
  - Responsive breakpoints from PLAN.md define exactly when to switch between 3-panel and tabs
  - Layout rules ensure no max-width constraints on this functional page

  **Acceptance Criteria**:

  **Agent-Executed QA Scenarios:**

  ```
  Scenario: Desktop 3-panel layout
    Tool: Playwright (playwright skill)
    Preconditions: Dev server running, viewport 1280x800
    Steps:
      1. Login and navigate to a workshop
      2. Wait for: page loaded with all 3 panels visible
      3. Assert: left panel (roadmap) visible with node list
      4. Assert: center panel (editor) visible with Tiptap editor
      5. Assert: right panel (AI sidebar) visible with chat input
      6. Screenshot: .sisyphus/evidence/task-7-desktop-3panel.png
    Expected Result: All 3 panels visible side by side
    Evidence: .sisyphus/evidence/task-7-desktop-3panel.png

  Scenario: Mobile tab switching
    Tool: Playwright (playwright skill)
    Preconditions: Dev server running, viewport 375x812 (iPhone)
    Steps:
      1. Login and navigate to a workshop
      2. Assert: tab bar visible with 3 tabs
      3. Assert: only one panel content visible at a time
      4. Click: "编辑器" tab
      5. Assert: editor panel visible, roadmap hidden
      6. Type in editor: "Mobile test text"
      7. Click: "AI 助手" tab
      8. Click: "编辑器" tab again
      9. Assert: "Mobile test text" still present (KeepAlive preserved state)
      10. Screenshot: .sisyphus/evidence/task-7-mobile-tabs.png
    Expected Result: Tab switching works, editor state preserved
    Evidence: .sisyphus/evidence/task-7-mobile-tabs.png

  Scenario: Node switching flushes editor and loads new content
    Tool: Playwright (playwright skill)
    Steps:
      1. Type content in editor for Node 1
      2. Click Node 2 in roadmap (if pending)
      3. Assert: editor content changes to Node 2's content (or empty if new)
      4. Click back to Node 1
      5. Assert: Node 1's typed content is restored (from autosave)
    Expected Result: Content correctly saved and loaded per node
    Evidence: .sisyphus/evidence/task-7-node-switch.png
  ```

  **Commit**: YES
  - Message: `feat(layout): refactor WorkshopDetail to 3-panel desktop + tab mobile layout`
  - Files: `app/src/views/workshop/WorkshopDetailView.vue`, `app/src/router/index.ts`
  - Pre-commit: `pnpm --filter app test -- --run && pnpm --filter app run build`

---

- [ ] 8. Profile Page

  **What to do**:
  - Create `app/src/views/profile/UserProfile.vue`:
    - Display current user info from userStore (username, email, bio, keywords)
    - Form fields: username (text input), bio (textarea), keywords (tag input or comma-separated)
    - Avatar section: current avatar display + file input for "upload" (mock: `URL.createObjectURL()` preview only)
    - Save button: updates userStore (mock — no backend persistence)
    - Toast/feedback on save
    - Dark mode support
    - i18n text
    - Responsive: single-column on mobile, centered card on desktop
  - Add `/profile` route to `app/src/router/index.ts`
  - Add profile link in NavBar (user avatar/name dropdown or direct link)
  - Write TDD tests for profile form validation (username required, bio max length)

  **Must NOT do**:
  - No real image upload — URL.createObjectURL() only
  - No image cropping or size validation
  - No email change flow
  - No password change flow

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: Form design, avatar preview, responsive card layout
  - **Skills**: [`frontend-ui-ux`]
    - `frontend-ui-ux`: Form UX patterns, file input styling, responsive design

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Task 9)
  - **Blocks**: Task 10
  - **Blocked By**: Tasks 0, 2 (needs userStore)

  **References**:

  **Pattern References**:
  - `app/src/views/auth/LoginPage.vue` — Form layout pattern (centered card, input fields)
  - `app/src/components/common/BaseInput.vue` — Input component to use for form fields
  - `app/src/components/common/BaseButton.vue` — Save button

  **API/Type References**:
  - `app/src/types/index.ts:UserProfile` — User profile type with fields to display/edit
  - `app/src/stores/user.ts` — userStore with user state to read/update

  **Documentation References**:
  - `docs/PLAN.md:97-101` — Profile page specification
  - `docs/INIT.md:36` — "可以编辑个人资料，昵称，头像，个人简介，关键词等"

  **WHY Each Reference Matters**:
  - LoginPage.vue shows the form card design pattern to follow for consistent UI
  - INIT.md specifies exact fields: 昵称(username), 头像(avatar), 个人简介(bio), 关键词(keywords)

  **Acceptance Criteria**:

  **TDD:**
  - [ ] Test: Profile form displays userStore data
  - [ ] Test: Saving updates userStore state
  - [ ] Test: Username is required (validation error if empty)
  - [ ] `pnpm --filter app test -- --run src/views/__tests__/UserProfile.test.ts` → PASS

  **Agent-Executed QA Scenarios:**

  ```
  Scenario: Profile page displays user info
    Tool: Playwright (playwright skill)
    Preconditions: Dev server running, user logged in
    Steps:
      1. Navigate to: http://localhost:5173/profile
      2. Wait for: profile form visible (timeout: 5s)
      3. Assert: username field contains current user's name
      4. Assert: email displayed (read-only)
      5. Screenshot: .sisyphus/evidence/task-8-profile.png
    Expected Result: Profile form pre-populated
    Evidence: .sisyphus/evidence/task-8-profile.png

  Scenario: Avatar preview works
    Tool: Playwright (playwright skill)
    Steps:
      1. On profile page, click avatar upload area
      2. Upload a test image file
      3. Assert: avatar preview updates to show uploaded image
    Expected Result: Avatar preview displays selected file
    Evidence: .sisyphus/evidence/task-8-avatar.png

  Scenario: Save updates NavBar
    Tool: Playwright (playwright skill)
    Steps:
      1. Change username to "TestNewName"
      2. Click save button
      3. Assert: success feedback visible
      4. Check NavBar: username text updated to "TestNewName"
    Expected Result: Profile changes reflected in NavBar
    Evidence: .sisyphus/evidence/task-8-save.png
  ```

  **Commit**: YES
  - Message: `feat(profile): add user profile page with avatar preview and form editing`
  - Files: `app/src/views/profile/UserProfile.vue`, `app/src/router/index.ts`, NavBar update
  - Pre-commit: `pnpm --filter app test -- --run && pnpm --filter app run build`

---

- [ ] 9. Export Feature (Markdown + Print CSS PDF)

  **What to do**:
  - Create `app/src/utils/export.ts`:
    - `generateMarkdown(workshopId)`: Collects all node submissions from workshopStore, formats as Markdown document with headings per node
    - `downloadMarkdown(content, filename)`: Creates blob + download link
    - `printPreview()`: Opens window.print() with print-friendly CSS
  - Create `app/src/views/workshop/ExportPreview.vue`:
    - Shows compiled document preview (all node content in order)
    - Nodes with no content: show as "[未完成]" / "[Not completed]" placeholder
    - Two action buttons: "下载 Markdown" and "导出 PDF (打印)"
    - Markdown download: triggers file download
    - PDF: applies print CSS + calls `window.print()`
    - Dark mode support
    - i18n text
    - Responsive layout
  - Create `app/src/assets/print.css`:
    - Print-specific styles: hide NavBar, sidebar, buttons
    - Clean typography for printed content
    - Page margins, font sizes optimized for A4
    - **IMPORTANT (Oracle review)**: Force print-safe styles in `@media print` — always `bg-white`, `text-black`, disable dark mode inheritance, disable decorative effects. Call `window.print()` only after `nextTick` + one frame so DOM/styles settle.
  - Add export route or modal trigger from WorkshopDetailView
  - Write tests for Markdown generation logic

  **Must NOT do**:
  - No html2canvas or jspdf
  - No server-side PDF generation
  - No custom PDF cover page or page numbering
  - No branded PDF template

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: Print CSS design, document preview layout, export UX
  - **Skills**: [`frontend-ui-ux`]
    - `frontend-ui-ux`: Print media queries, document layout, download UX patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Task 8)
  - **Blocks**: Task 10
  - **Blocked By**: Tasks 0, 3 (needs service layer), 5 (needs editor content/store)

  **References**:

  **Pattern References**:
  - `app/src/stores/workshop.ts` — (created in Task 4) Workshop data for export compilation
  - `app/src/stores/editor.ts` — (created in Task 5) Node content access

  **API/Type References**:
  - `app/src/types/index.ts:ExportConfig` — Export configuration type
  - `app/src/types/services.ts:IExportService` — Export service interface

  **Documentation References**:
  - `docs/PLAN.md:92-95` — Export page specification
  - `docs/INIT.md:42` — "所有内容汇总，形成一个整体文档，导出txt, md, 或者pdf"

  **WHY Each Reference Matters**:
  - workshopStore has the node data and submissions to compile into export document
  - INIT.md confirms export is per-workshop: ALL node content combined into one document

  **Acceptance Criteria**:

  **TDD:**
  - [ ] Test: `generateMarkdown(workshopId)` returns valid markdown string with all node headings
  - [ ] Test: Empty nodes shown as "[未完成]" in export
  - [ ] Test: Markdown includes workshop title as H1
  - [ ] `pnpm --filter app test -- --run src/utils/__tests__/export.test.ts` → PASS

  **Agent-Executed QA Scenarios:**

  ```
  Scenario: Export preview shows all node content
    Tool: Playwright (playwright skill)
    Preconditions: Dev server running, user has content in at least 2 nodes
    Steps:
      1. Navigate to export page/modal
      2. Wait for: preview content visible
      3. Assert: workshop title displayed as heading
      4. Assert: node titles visible as sub-headings
      5. Assert: user-entered content visible under each node
      6. Screenshot: .sisyphus/evidence/task-9-preview.png
    Expected Result: Complete document preview
    Evidence: .sisyphus/evidence/task-9-preview.png

  Scenario: Markdown download works
    Tool: Playwright (playwright skill)
    Steps:
      1. On export page, click "下载 Markdown" button
      2. Wait for: download triggered (check downloads)
      3. Assert: .md file downloaded with correct content
    Expected Result: Markdown file downloaded
    Evidence: Download confirmation

  Scenario: Print produces clean output
    Tool: Playwright (playwright skill)
    Steps:
      1. On export page, trigger print preview (evaluate JS: window.print simulation)
      2. Assert: @media print CSS hides navigation elements
      3. Take print-mode screenshot if possible
    Expected Result: Print-friendly layout
    Evidence: .sisyphus/evidence/task-9-print.png
  ```

  **Commit**: YES
  - Message: `feat(export): add Markdown download and Print CSS PDF export`
  - Files: `app/src/utils/export.ts`, `app/src/views/workshop/ExportPreview.vue`, `app/src/assets/print.css`
  - Pre-commit: `pnpm --filter app test -- --run && pnpm --filter app run build`

---

- [ ] 10. Integration Test & Full Journey Verification

  **What to do**:
  - Run complete test suite: `pnpm --filter app test -- --run` → all tests pass
  - Run build: `pnpm --filter app run build` → exit 0
  - Execute full user journey via Playwright:
    1. Login → Dashboard → Select workshop → Roadmap → Edit → AI chat → Adopt → Submit → Export
  - Verify dark mode works across ALL pages
  - Verify i18n switches ALL text on ALL pages
  - Verify mobile responsiveness (375px viewport) on ALL pages
  - Fix any broken interactions discovered during integration
  - Update `docs/TODO.md` with completion status for all Phase 1 tasks

  **Must NOT do**:
  - No new features — bug fixes and polish only
  - Do not start Phase 2 work

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Thorough integration testing requires deep investigation of edge cases
  - **Skills**: [`playwright`, `frontend-ui-ux`]
    - `playwright`: Browser automation for full journey testing
    - `frontend-ui-ux`: Visual verification of UI consistency

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 6 (final, solo)
  - **Blocks**: None (final task)
  - **Blocked By**: ALL previous tasks

  **References**:

  **Pattern References**:
  - ALL components and views created/updated in Tasks 0-9

  **Documentation References**:
  - `docs/TODO.md` — Checklist to update with completion status
  - `docs/PLAN.md:273-278` — Test and acceptance standards

  **Acceptance Criteria**:

  **Agent-Executed QA Scenarios:**

  ```
  Scenario: Full user journey (happy path)
    Tool: Playwright (playwright skill)
    Preconditions: Dev server running, clean localStorage
    Steps:
      1. Navigate to: http://localhost:5173/login
      2. Login with: "demo@juanleme.com" / "password"
      3. Assert: redirected to /dashboard
      4. Assert: workshop cards visible
      5. Click first workshop card
      6. Assert: WorkshopDetail loads with 3 panels (desktop) or tabs (mobile)
      7. Assert: first roadmap node is pending/active
      8. Click first node in roadmap
      9. Assert: editor loads for that node
      10. Type: "我的产品灵感是做一个宠物社交App"
      11. Wait 1s for autosave
      12. Open AI sidebar, type: "给我一些产品灵感"
      13. Wait for AI response
      14. Click "采纳" on response
      15. Assert: AI text appears in editor
      16. Click submit/complete button
      17. Assert: node status changes to completed, next node unlocks
      18. Navigate to export page
      19. Assert: submitted content visible in preview
      20. Click "下载 Markdown"
      21. Assert: file downloads
      22. Screenshot: .sisyphus/evidence/task-10-journey-complete.png
    Expected Result: Complete user journey works end-to-end
    Evidence: .sisyphus/evidence/task-10-journey-complete.png

  Scenario: Dark mode across all pages
    Tool: Playwright (playwright skill)
    Steps:
      1. Enable dark mode via toggle
      2. Navigate through: login → dashboard → workshop → profile → export
      3. Assert each page: background is dark, text is light, no unreadable elements
      4. Screenshot each page in dark mode
    Expected Result: Dark mode consistent across all pages
    Evidence: .sisyphus/evidence/task-10-dark-*.png

  Scenario: Mobile responsive across all pages
    Tool: Playwright (playwright skill)
    Preconditions: Viewport 375x812
    Steps:
      1. Navigate through all pages
      2. Assert: no horizontal scroll
      3. Assert: all interactive elements accessible (min 44x44px touch targets)
      4. Assert: WorkshopDetail uses tab layout, not 3-panel
      5. Screenshot each page
    Expected Result: All pages usable at mobile width
    Evidence: .sisyphus/evidence/task-10-mobile-*.png

  Scenario: All tests pass and build succeeds
    Tool: Bash
    Steps:
      1. pnpm --filter app test -- --run
      2. Assert: exit code 0, all tests pass
      3. pnpm --filter app run build
      4. Assert: exit code 0
    Expected Result: Clean test suite and build
    Evidence: Test + build output
  ```

  **Commit**: YES
  - Message: `test(integration): verify full Phase 1 user journey and fix integration issues`
  - Files: `docs/TODO.md`, any bug fix files
  - Pre-commit: `pnpm --filter app test -- --run && pnpm --filter app run build`

---

## Commit Strategy

| After Task | Message | Key Files | Verification |
|------------|---------|-----------|--------------|
| 0 | `chore(infra): add Vitest, Tiptap, vue-i18n deps and service type contracts` | package.json, vitest.config.ts, types/ | test + build |
| 1 | `feat(ui): add dark mode, i18n (EN/CN), replace emojis with Heroicons` | tailwind.config.js, i18n/, all components | test + build |
| 2 | `feat(auth): add userStore, mock login/logout, route guards` | stores/user.ts, router, LoginPage, NavBar | test + build |
| 3 | `refactor(services): add service registry pattern with mock adapters` | services/registry.ts, services/mock/ | test + build |
| 4 | `feat(roadmap): add workshopStore with node status state machine` | stores/workshop.ts, RoadmapItem.vue | test + build |
| 5 | `feat(editor): integrate Tiptap rich text editor with autosave` | stores/editor.ts, TaskEditor.vue | test + build |
| 6 | `feat(ai): add AI chatbot sidebar with mock responses and adopt flow` | AiSidebar.vue, aiService.ts | test + build |
| 7 | `feat(layout): refactor WorkshopDetail to 3-panel desktop + tab mobile` | WorkshopDetailView.vue | test + build |
| 8 | `feat(profile): add user profile page with avatar preview` | UserProfile.vue, router | test + build |
| 9 | `feat(export): add Markdown download and Print CSS PDF export` | ExportPreview.vue, export.ts, print.css | test + build |
| 10 | `test(integration): verify full Phase 1 user journey` | TODO.md, bug fixes | test + build |

---

## Success Criteria

### Verification Commands
```bash
pnpm --filter app test -- --run    # Expected: all tests pass
pnpm --filter app run build        # Expected: exit 0, no TS errors
pnpm --filter app run dev          # Expected: app starts on localhost:5173
```

### Final Checklist
- [ ] All "Must Have" features present and working
- [ ] All "Must NOT Have" items absent
- [ ] All Vitest tests pass
- [ ] Full user journey works: login → dashboard → workshop → edit → AI → export
- [ ] Dark mode toggle works on all pages
- [ ] i18n EN/CN switch works on all pages
- [ ] No emoji characters in source code (only Heroicons)
- [ ] All pages responsive at 375px mobile width
- [ ] No console errors in browser during normal usage
- [ ] `docs/TODO.md` updated with Phase 1 completion status
