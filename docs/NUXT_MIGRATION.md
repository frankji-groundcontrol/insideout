# Nuxt 4 Universal SSR Migration Spec (Canonical)

Status: **Authoritative.** Every generation agent follows this verbatim. Do not re-derive decisions.

## 0. Scope, Locations, and Global Rules

- **Target:** Nuxt 4, Universal SSR (`ssr: true`).
- **Project root (Nuxt rootDir):** `/Users/frankji/Projects/juanleme/app` — the existing Vite app is **replaced in place**. `pnpm-lock.yaml` already lives here; this is the package root.
- **Package manager:** pnpm.
- **Source layout decision — KEEP `src/` AND set `srcDir: "src"`.** The current code uses the `@/` alias mapped to `./src/*` pervasively (verified across `@/components`, `@/stores`, `@/services`, `@/types`, `@/utils`, `@/i18n`, `@/lib`, `@/views`). To avoid touching hundreds of import statements, we set Nuxt `srcDir: "src"` so Nuxt's app dirs (`pages/`, `layouts/`, `middleware/`, `plugins/`, `components/`, `composables/`, `app.vue`, `error.vue`, `assets/`) live **inside `src/`**, and we keep an explicit `@` → `./src` alias. `server/`, `public/`, `modules/`, `nuxt.config.ts`, `tailwind.config.js`, `package.json` stay at the **rootDir** (`app/`).
  - This deviates from the prompt's `srcDir: "."` wording but achieves the same goal (no `app/app/` nesting) with **strictly less churn** — the `@/*` → `src/*` mapping that every file already uses is preserved unchanged. The prompt's intent ("keep source files where the `@` alias already points, avoid ugly nesting, document the choice") is satisfied. If a future agent insists on literal `srcDir: "."`, they must additionally set `serverDir`, `dir.public`, and rewrite the `@` alias to root — explicitly NOT done here.
  - Nuxt auto-detects `app`/`server` dirs only **outside** a `src` dir; because we use `srcDir: "src"` (not `src/app`), Nuxt's app directories sit directly under `src/` and are detected normally. Do **not** nest a second `app/` inside `src/`.

### SSR rules every agent MUST obey (non-negotiable)

1. **No browser globals during SSR setup.** Guard `localStorage`, `sessionStorage`, `window`, `document`, `navigator`, `URL.createObjectURL` behind `import.meta.client` or inside `onMounted` / event handlers. Never at module top-level or in `setup()` synchronous body.
2. **No `import.meta.env.VITE_*`.** Replace every read with `useRuntimeConfig()` (see §2 mapping). `import.meta.client` / `import.meta.server` are the SSR branch flags (these are fine).
3. **No eager Supabase client and no `getServices()` at module top-level.** Both are deferred into a Nuxt plugin / composable that runs in app context (see §7, §8). The line `export const services = getServices()` in `registry.ts` is **deleted**.
4. **`@/` alias must keep resolving to the Nuxt source root (`./src`).** `~` and `@` both point at `srcDir` (= `src`).
5. **Preserve behavior and Chinese comments/copy verbatim** wherever not structurally changed.

---

## 1. Exact `package.json`

Path: `/Users/frankji/Projects/juanleme/app/package.json`.

### Remove these (Nuxt provides them transitively; manual Vite/router tooling is obsolete)

- `vite`
- `@vitejs/plugin-vue`
- `vue-tsc`
- `vue-router` (Nuxt bundles vue-router; do not declare it directly)
- `@vue/tsconfig` (replaced by `.nuxt/tsconfig.json`)
- `@vue/eslint-config-typescript`, `@vue/eslint-config-prettier` may stay or be replaced by `@nuxt/eslint`; keep for now (out of critical path).

### Add these

| Package | Version | Why |
|---|---|---|
| `nuxt` | `^4.0.0` | Framework. |
| `@pinia/nuxt` | `^0.11.0` | Auto-registers Pinia; replaces `createPinia()` in `main.ts`. |
| `pinia` | keep `^3.0.4` | Peer of `@pinia/nuxt`. |
| `@nuxtjs/tailwindcss` | `^6.12.0` | Tailwind **v3**-compatible module line (7.x is Tailwind v4 — do NOT use). Wires PostCSS automatically. |
| `vue-i18n` | keep `^9.14.5` | Used by the **custom** i18n plugin (we do NOT adopt `@nuxtjs/i18n` — see §9). |
| `@nuxt/test-utils` | `^3.14.0` | Vitest + Nuxt env for component/view mount tests (see §10). |

Keep: `@supabase/supabase-js`, `@heroicons/vue`, all `@tiptap/*`, `vue`, `happy-dom`, `@vue/test-utils`, `vitest`, `typescript`, `@types/node`, `tailwindcss@^3.4.19`, `postcss`, `autoprefixer`, eslint stack, prettier.

> We do **NOT** add `@nuxtjs/supabase` (§8) or `@nuxtjs/i18n` (§9). Justification in those sections.

### Scripts (replace the entire `scripts` block)

```json
{
  "scripts": {
    "dev": "nuxt dev",
    "build": "nuxt build",
    "generate": "nuxt generate",
    "preview": "nuxt preview",
    "postinstall": "nuxt prepare",
    "test": "vitest --run"
  }
}
```

`postinstall: nuxt prepare` generates `.nuxt/` types so `tsconfig` extension and CI type-checks work. `test` is unchanged (`vitest --run`); type-check via `nuxt typecheck` if desired (optional, not required).

---

## 2. Exact `nuxt.config.ts`

Path: `/Users/frankji/Projects/juanleme/app/nuxt.config.ts` (create).

```ts
import { fileURLToPath } from 'node:url'

export default defineNuxtConfig({
  ssr: true,
  srcDir: 'src',
  compatibilityDate: '2025-06-01',

  modules: ['@pinia/nuxt', '@nuxtjs/tailwindcss'],

  // Global stylesheet (was imported in main.ts). @tailwind directives processed by the tailwind module.
  css: ['~/style.css'],

  // Keep the existing '@' alias resolving to the Nuxt source root (./src).
  // '~' already points at srcDir; we add '@' explicitly because the codebase uses '@/...'.
  alias: {
    '@': fileURLToPath(new URL('./src', import.meta.url)),
  },

  app: {
    head: {
      htmlAttrs: { lang: 'en' },
      title: 'app',
      meta: [
        { charset: 'UTF-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1.0' },
      ],
      link: [{ rel: 'icon', type: 'image/svg+xml', href: '/vite.svg' }],
    },
  },

  runtimeConfig: {
    // No private/server-only secrets in this app today. (Anthropic/Vault secrets live in the
    // Supabase Edge Function, not the Nuxt runtime.) Add server-only keys here later if needed.
    public: {
      apiMode: '',          // NUXT_PUBLIC_API_MODE         (was VITE_API_MODE)
      bundleAuth: '',       // NUXT_PUBLIC_BUNDLE_AUTH       (was VITE_BUNDLE_AUTH)
      bundleData: '',       // NUXT_PUBLIC_BUNDLE_DATA       (was VITE_BUNDLE_DATA)
      bundleAiExport: '',   // NUXT_PUBLIC_BUNDLE_AI_EXPORT  (was VITE_BUNDLE_AI_EXPORT)
      supabaseUrl: '',      // NUXT_PUBLIC_SUPABASE_URL      (was VITE_SUPABASE_URL)
      supabaseAnonKey: '',  // NUXT_PUBLIC_SUPABASE_ANON_KEY (was VITE_SUPABASE_ANON_KEY)
    },
  },

  // @nuxtjs/tailwindcss auto-detects content paths AND auto-wires PostCSS (tailwindcss + autoprefixer).
  // We keep an explicit tailwind.config.js (§ below) so darkMode:'class' and widened globs are pinned.

  typescript: {
    // Surface type errors but do not block dev server. Set true if CI should fail on type errors.
    typeCheck: false,
  },
})
```

### Why `runtimeConfig.public.*` defaults are empty strings

Nuxt overrides any `runtimeConfig` value from a matching `NUXT_*` env var at **runtime** (not build-time inlining). Empty-string defaults mean: when no env is set, `apiMode` is falsy → registry falls back to `'mock'` (preserving the current "no env → mock" default that tests rely on). Setting `NUXT_PUBLIC_API_MODE=supabase` (etc.) flips behavior at runtime.

### `.env` migration (also see §2b file plan)

Nuxt only auto-loads `.env` (not `.env.supabase`). Create `/Users/frankji/Projects/juanleme/app/.env` and `/Users/frankji/Projects/juanleme/app/.env.example`. **Full env name mapping:**

| Old (Vite) | New (Nuxt runtime env) | runtimeConfig path |
|---|---|---|
| `VITE_API_MODE` | `NUXT_PUBLIC_API_MODE` | `public.apiMode` |
| `VITE_BUNDLE_AUTH` | `NUXT_PUBLIC_BUNDLE_AUTH` | `public.bundleAuth` |
| `VITE_BUNDLE_DATA` | `NUXT_PUBLIC_BUNDLE_DATA` | `public.bundleData` |
| `VITE_BUNDLE_AI_EXPORT` | `NUXT_PUBLIC_BUNDLE_AI_EXPORT` | `public.bundleAiExport` |
| `VITE_SUPABASE_URL` | `NUXT_PUBLIC_SUPABASE_URL` | `public.supabaseUrl` |
| `VITE_SUPABASE_ANON_KEY` | `NUXT_PUBLIC_SUPABASE_ANON_KEY` | `public.supabaseAnonKey` |

`.env` contents (carry the existing values from `.env.supabase`; the anon key is a publishable key, so `public.*` exposure is acceptable):

```
NUXT_PUBLIC_API_MODE=supabase
NUXT_PUBLIC_SUPABASE_URL=https://kvvxenjebjjdsqpbdqvx.supabase.co
NUXT_PUBLIC_SUPABASE_ANON_KEY=sb_publishable_hDag7ZDipC0VShhvcEjY7w_w_BHPhhk
```

> Note: the bundle keys (`VITE_BUNDLE_*`) were never set in `.env.supabase`; they default to `apiMode`. Same behavior preserved: leave them unset in `.env`.

### Tailwind config + PostCSS

- Keep `@nuxtjs/tailwindcss@^6` in `modules`. It injects `@tailwind` directives **and** the PostCSS pipeline automatically.
- **Delete** `postcss.config.js` (the module owns PostCSS). If an agent prefers to keep manual PostCSS, that's mutually exclusive with the module — do not do both.
- Replace `tailwind.config.js` content globs (flat `src/` layout, **no `app/` prefix** because `srcDir: 'src'` and files live under `src/`):

```js
/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: [
    './src/components/**/*.{vue,js,ts,jsx,tsx}',
    './src/layouts/**/*.vue',
    './src/pages/**/*.vue',
    './src/plugins/**/*.{js,ts}',
    './src/features/**/*.{vue,js,ts}',
    './src/views/**/*.vue',           // remove once views/ is fully emptied into pages/components
    './src/app.vue',
    './src/error.vue',
  ],
  theme: { extend: {} },
  plugins: [],
}
```

The `@nuxtjs/tailwindcss` module auto-detects most of these, but we keep explicit globs to guarantee `src/features/**` and `darkMode:'class'` are never purged.

### Supabase + i18n module options

Not applicable — we use **custom plugins**, not `@nuxtjs/supabase` / `@nuxtjs/i18n`. Therefore there is **no `supabase:` block and no `i18n:` block** in `nuxt.config.ts`. The decision and justification (including "`redirect: false` vs custom guard"): **we keep the custom Pinia auth guard, so there is no `@nuxtjs/supabase` redirect to configure at all** — see §6 and §8.

---

## 3. TypeScript Config Strategy

- **`tsconfig.json`** (root, `app/tsconfig.json`) — replace the project-references solution with a thin file that extends the Nuxt-generated config:

```json
{
  "extends": "./.nuxt/tsconfig.json"
}
```

`.nuxt/tsconfig.json` is generated by `nuxt prepare` (run via `postinstall`) and already wires `@`/`~` → `srcDir`, `#imports`, `strict`, DOM libs, and Vue. Because we also set the explicit `alias` in `nuxt.config.ts`, the `@/*` paths resolve in both the IDE and the build.

- **Delete** `tsconfig.app.json` and `tsconfig.node.json` — both are superseded. `tsconfig.node.json` only scoped `vite.config.ts` (deleted). The `@/*` → `src/*` path mapping it carried is replaced by the Nuxt alias.
- **Vitest** types: `@nuxt/test-utils`/`vitest` globals come from the vitest config (§10); no extra tsconfig needed.

---

## 4. File Plan (every source file)

Base dir for all paths: `/Users/frankji/Projects/juanleme/app/`. After migration, Nuxt app dirs live under `src/`.

### 4a. Entry / config — delete & create

| From | To | Action |
|---|---|---|
| `src/main.ts` | — | **delete** (Nuxt owns createApp/mount; Pinia→module, router→file routing, i18n→plugin, style.css→nuxt.config.css) |
| `index.html` | — | **delete** (head migrated to `nuxt.config.ts app.head`) |
| `vite.config.ts` | — | **delete** (folded into nuxt.config; `@` alias → nuxt.config alias) |
| `vitest.config.ts` | `vitest.config.ts` | **transform** (see §10) |
| `tsconfig.json` | `tsconfig.json` | **transform** (extends `./.nuxt/tsconfig.json`) |
| `tsconfig.app.json` | — | **delete** |
| `tsconfig.node.json` | — | **delete** |
| `postcss.config.js` | — | **delete** (module owns PostCSS) |
| `tailwind.config.js` | `tailwind.config.js` | **transform** (widen globs, §2) |
| `eslint.config.js` | `eslint.config.js` | **transform** (add `.nuxt`,`.output` to ignores) |
| `.env.supabase` | `.env` + `.env.example` | **transform/rename** (rename keys per §2 table) |
| `package.json` | `package.json` | **transform** (§1) |
| — | `nuxt.config.ts` | **create** (§2) |
| — | `src/app.vue` | **create** (§5) |
| — | `src/error.vue` | **create** (§5) |
| `public/vite.svg` | `public/vite.svg` | keep (favicon; referenced by `app.head`) |
| `src/router/index.ts` | — | **delete** (replaced by file routing + `middleware/auth.global.ts`) |

### 4b. App shell / router → Nuxt

| From | To | Action |
|---|---|---|
| `src/App.vue` | `src/app.vue` + `src/layouts/default.vue` + `src/layouts/empty.vue` | **transform/split** (§5) |
| `src/router/index.ts` (guard) | `src/middleware/auth.global.ts` | **transform** (§6) |

### 4c. Views → Pages + Components (file-based routing)

Routes from `src/router/index.ts` map to Nuxt pages. `[id]` is the dynamic segment.

| Route | From (view) | To (Nuxt page) | Action | Page meta |
|---|---|---|---|---|
| `/` | `src/components/HelloWorld.vue` | `src/pages/index.vue` | **create** wrapper rendering `<HelloWorld/>` | `definePageMeta({ public: true })` |
| `/dashboard` | `src/views/dashboard/DashboardView.vue` | `src/pages/dashboard.vue` | **move + transform** | (auth required; no meta) |
| `/workshop/:id` | `src/views/workshop/WorkshopDetailView.vue` | `src/pages/workshop/[id]/index.vue` | **move + transform** | (auth required) |
| `/workshop/:id/export` | `src/views/workshop/ExportPreview.vue` | `src/pages/workshop/[id]/export.vue` | **move + transform** | (auth required) |
| `/profile` | `src/views/profile/UserProfile.vue` | `src/pages/profile.vue` | **move + transform** | (auth required) |
| `/login` | `src/views/auth/LoginPage.vue` | `src/pages/login.vue` | **move + transform** | `definePageMeta({ public: true, layout: 'empty' })` |

> `HelloWorld.vue` is a demo gallery (hardcoded zh-CN, no i18n). Keep it as a **component** and render it from `pages/index.vue`. Do NOT move `HelloWorld.vue` into `pages/`.

Non-routed feature/view files that move to `components/`:

| From | To | Action |
|---|---|---|
| `src/views/workshop/TaskEditor.vue` | `src/components/workshop/TaskEditor.vue` | **move + transform** (it is a child component, NOT a route; Tiptap host — wrap editor creation client-only, see §5/Tiptap) |
| `src/features/workshop/detail/WorkshopDetailPage.vue` | `src/features/workshop/detail/WorkshopDetailPage.vue` | **transform in place** (update `@/views/workshop/TaskEditor.vue` import → `@/components/workshop/TaskEditor.vue`; `useRoute().params.id`, `navigateTo`) |
| `src/features/workshop/detail/components/WorkshopActionBar.vue` | same | **keep** (alias-only; SSR-safe) |
| `src/features/workshop/detail/composables/*` | same | **keep + transform** (alias rewrites; `useWorkshopViewport` already guards `window`; `useWorkshopDetailSession` keeps `useRouter()`/`navigateTo`) |
| `src/features/workshop/ai/composables/useAiConversation.ts` | same | **transform** (fix `requireServices` import → `useServices()`, see §7) |

> `src/features/**` stays under `src/features/` (explicit imports, not Nuxt auto-import). Composables here are NOT moved into `src/composables/` to avoid auto-import name collisions; keep explicit `@/features/...` imports.

### 4d. Stores, services, lib, types, i18n, utils, assets

| From | To | Action |
|---|---|---|
| `src/stores/user.ts` | same | **transform** (guard all 4 localStorage touchpoints with `import.meta.client`; switch `services` access to `useServices()`, §7) |
| `src/stores/editor.ts` | same | **transform** (move `let debounceTimer` **inside** the `defineStore` setup callback to make it per-instance/per-request; switch `services` access to `useServices()`) |
| `src/stores/workshop.ts` | same | **transform** (alias-only + `useServices()` for `services` access) |
| `src/services/registry.ts` | same | **transform** (delete `export const services = getServices()`; make `resolveBundleModes` accept config; add `requireServices`/`getServicesAsync(modes)`; §7) |
| `src/lib/supabase.ts` | same | **transform** (`createSupabaseClient(url, key)` factory; remove module singleton + `import.meta.env`; §8) |
| `src/services/mock/*` | same | **transform** (guard each `localStorage` call with `import.meta.client`; `mock/index.ts` is dead `mockApi` — **delete**) |
| `src/services/supabase/*` | same | **transform** (replace `import { getSupabase } from '@/lib/supabase'` + `getSupabase()` calls with the injected client from the registry; `aiService.ts` guard `localStorage` for `ai-conv:`; keep `.schema('juanleme')` calls verbatim) |
| `src/types/index.ts` | same | **keep** (pure types + 2 Error subclasses; SSR-safe) |
| `src/types/services.ts` | same | **keep** (`@/types` alias still resolves) |
| `src/i18n/index.ts` | `src/i18n/index.ts` (reduced) + `src/plugins/i18n.ts` | **split** (§9) |
| `src/i18n/locales/zh-CN.ts` | same | **keep** (verbatim) |
| `src/i18n/locales/en-US.ts` | same | **keep** (verbatim) |
| `src/utils/export.ts` | same | **transform** (add `if (import.meta.client)` / early `return` guards in `downloadMarkdown` & `triggerPrint`; `generateMarkdown` unchanged) |
| `src/style.css` | `src/style.css` | **keep** (referenced via `nuxt.config css: ['~/style.css']`; do not import in any component) |
| `src/assets/print.css` | `src/assets/print.css` | **keep** (still imported in `pages/workshop/[id]/export.vue` via `@/assets/print.css`) |
| `src/assets/vue.svg` | — | **delete** (dead asset, zero references) |

### 4e. Components

All 11 `src/components/**` files **move/stay in place** under `src/components/` (Nuxt auto-imports them). Per-file transforms:

| File | Transform |
|---|---|
| `components/HelloWorld.vue` | alias/auto-import only (keep) |
| `components/common/BaseButton.vue` | keep (SSR-safe) |
| `components/common/BaseInput.vue` | replace `Math.random()` id default with Vue `useId()` (auto-imported in Nuxt) to avoid hydration-id mismatch |
| `components/common/LangToggle.vue` | keep `import { LANG_STORAGE_KEY } from '@/i18n'`; `localStorage.setItem` stays in click handler (client-only); `useI18n()` continues to work via the custom plugin |
| `components/common/ThemeToggle.vue` | keep `document`/`localStorage` access inside `onMounted`/handlers; add inline head theme-init script (§5) to avoid FOUC; guard with `import.meta.client` for defense |
| `components/layout/NavBar.vue` | `RouterLink` → `<NuxtLink>`; `useRouter().push('/login')` → `navigateTo('/login')`; auth-conditional UI shows logged-out on SSR until client hydrate (expected) |
| `components/layout/AppFooter.vue` | keep (`new Date().getFullYear()` benign New-Year-only mismatch) |
| `components/workshop/AiSidebar.vue` | alias rewrite for `@/features/workshop/ai/composables/useAiConversation` (resolves unchanged) |
| `components/workshop/EditorToolbar.vue` | keep (operates on injected `Editor`; SSR-render-safe) |
| `components/workshop/RoadmapItem.vue` | `@/types` alias keep |
| `components/workshop/RoadmapSidebar.vue` | keep (auto-import `RoadmapItem` or explicit `@/...`) |
| `components/workshop/TaskDetail.vue` | `@/types`, `BaseButton` alias/auto-import |
| `components/workshop/WorkshopCard.vue` | `import { RouterLink } from 'vue-router'` + `<RouterLink>` → `<NuxtLink>` (vue-router not a direct dep) |

> **Auto-import policy:** set `components: [{ path: '~/components', pathPrefix: false }]` in `nuxt.config.ts` so existing tags `<BaseButton/>`, `<RoadmapItem/>`, `<WorkshopCard/>` keep working without the path-prefixed names (`<CommonBaseButton/>` etc.). Add this to the `nuxt.config.ts` from §2:
>
> ```ts
> components: [{ path: '~/components', pathPrefix: false }],
> ```
>
> Explicit `import X from '@/components/...'` statements may remain (they still resolve) or be removed in favor of auto-import — either is acceptable; do not break templates.

### 4f. Tests

All `*.test.ts` stay co-located (same relative paths under `src/`). See §10 for the env-stub and alias details. Only `src/__tests__/schema-isolation.test.ts` needs body edits (env key rename). `mock/index.ts` deletion does not affect tests (registry uses the `create*` factories).

---

## 5. App Shell: `app.vue`, layouts, `error.vue`

### `src/app.vue` (create) — replaces `App.vue` mount shell

```vue
<script setup lang="ts">
// initFromStorage moved to plugins/auth-init.client.ts (runs before first paint, client-only).
</script>

<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>
```

- `<RouterView/>` → `<NuxtPage/>`; the NavBar/Footer wrapper and `isLayoutEmpty` computed are **removed** — the Nuxt layout system replaces them.
- The `onMounted(initFromStorage)` from `App.vue` moves into a client plugin (§7 `plugins/auth-init.client.ts`) so session is rehydrated before the global middleware evaluates auth on the client.

### `src/layouts/default.vue` (create) — the non-empty branch of `App.vue`

```vue
<script setup lang="ts"></script>

<template>
  <div class="min-h-screen flex flex-col bg-gray-50 text-gray-900 font-sans dark:bg-gray-900 dark:text-gray-100">
    <NavBar />
    <main class="flex-grow">
      <slot />
    </main>
    <AppFooter />
  </div>
</template>
```

### `src/layouts/empty.vue` (create) — the `meta.layout === 'empty'` branch (login)

```vue
<script setup lang="ts"></script>

<template>
  <div class="min-h-screen flex flex-col bg-gray-50 text-gray-900 font-sans dark:bg-gray-900 dark:text-gray-100">
    <main class="flex-grow">
      <slot />
    </main>
  </div>
</template>
```

- `pages/login.vue` selects it via `definePageMeta({ layout: 'empty' })`. All other pages get `default` automatically. This replaces the `v-if="!isLayoutEmpty"` logic 1:1.

### `src/error.vue` (create) — Nuxt error boundary (no equivalent existed; add a minimal one)

```vue
<script setup lang="ts">
import type { NuxtError } from '#app'
defineProps<{ error: NuxtError }>()
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center bg-gray-50 text-gray-900 dark:bg-gray-900 dark:text-gray-100">
    <h1 class="text-4xl font-bold">{{ error.statusCode }}</h1>
    <p class="mt-2">{{ error.statusMessage }}</p>
    <button class="mt-4 underline" @click="clearError({ redirect: '/' })">返回首页</button>
  </div>
</template>
```

### Theme FOUC fix (ThemeToggle) — inline head script

To prevent the light→dark flash and `<html>` class hydration mismatch, add an inline head script in `nuxt.config.ts app.head.script` that sets the `dark` class before paint, reading the same `juanleme-theme` key:

```ts
// inside app.head in nuxt.config.ts
script: [
  {
    innerHTML:
      "try{var t=localStorage.getItem('juanleme-theme');var d=t?t==='dark':matchMedia('(prefers-color-scheme:dark)').matches;document.documentElement.classList.toggle('dark',d)}catch(e){}",
    tagPosition: 'head',
  },
],
```

Set `app.head.script[0].tagPosition: 'head'` and Nuxt will not defer it. (Optional but recommended; ThemeToggle's `onMounted` still syncs its reactive `isDark`.) If an agent prefers, install `@nuxtjs/color-mode` instead — but the inline script is lower-risk and keeps the existing `juanleme-theme` key and component logic intact.

### Tiptap host (TaskEditor)

`components/workshop/TaskEditor.vue` creates a Tiptap `Editor` (client-only). Wrap the editor render area in `<ClientOnly>` **or** create the editor inside `onMounted`. Add `build: { transpile: ['@tiptap/vue-3'] }` to `nuxt.config.ts` only if Nuxt reports ESM/CJS interop errors for Tiptap (try without first).

---

## 6. Auth Guard → `src/middleware/auth.global.ts`

The `router.beforeEach` guard becomes a **global route middleware**. The decision (justified in §8): **we keep the custom Pinia/localStorage auth guard and do NOT use `@nuxtjs/supabase`'s redirect.** Because the session lives only in `localStorage` (client-only), the middleware **must short-circuit on the server**, otherwise SSR (where `isAuthenticated` is always `false`) would redirect every protected route to `/login`.

`src/middleware/auth.global.ts` (create):

```ts
export default defineNuxtRouteMiddleware((to) => {
  // 会话存于 localStorage，仅客户端可用；服务端跳过守卫，避免 SSR 把所有受保护路由重定向到 /login
  if (import.meta.server) return

  const userStore = useUserStore()

  // 已登录用户访问 /login → 跳转到工作台
  if (to.path === '/login' && userStore.isAuthenticated) {
    return navigateTo('/dashboard')
  }

  // 未认证用户访问非公开页面 → 跳转到登录页
  if (!to.meta.public && !userStore.isAuthenticated) {
    return navigateTo('/login')
  }
})
```

- `to.meta.public` is provided by `definePageMeta({ public: true })` on `pages/index.vue` and `pages/login.vue`.
- vue-router guards returned a path string; Nuxt middleware returns `navigateTo(path)`. 1:1 mapping of the two original `beforeEach` branches.
- `useUserStore` is auto-imported by `@pinia/nuxt` (it lives in `src/stores/user.ts`). Session rehydration before guard evaluation is guaranteed by `plugins/auth-init.client.ts` (§7) running before the first client navigation resolves.

> Follow-up (NOT required for this mechanical port): migrating the token to `useCookie('juanleme-token')` would enable true server-side auth and remove the client-only short-circuit. Out of scope; note only.

---

## 7. THE SERVICE ACCESS API (critical — all agents align on this)

See the `serviceAccessApi` field of the structured output for the verbatim contract. Summary here for completeness; the structured field is the single source of truth.

**Problem:** `registry.ts` ends with `export const services = getServices()` (runs at import, throws for any non-all-mock bundle), and `lib/supabase.ts` memoizes one `createClient()` (shared across SSR requests, localStorage-session oriented). Both illegal under SSR.

**Replacement, three parts:**

1. **`registry.ts` becomes pure** — no top-level side effects, no `import.meta.env`. `resolveBundleModes(env)` takes a plain config object; `getServicesAsync(modes, deps)` builds the registry (where `deps` carries the Supabase client factory output for the supabase bundle). Delete `export const services`. Add `requireServices()` (synchronous accessor that throws if the plugin has not provided the registry — fixes the pre-existing broken import in `useAiConversation.ts`).

2. **`plugins/services.ts`** (Nuxt plugin) reads `useRuntimeConfig().public`, resolves+validates modes, builds the SSR-safe Supabase client (per request, §8), calls `getServicesAsync(modes, { supabase })`, and `provide`s the registry as `$services`.

3. **`useServices()` composable** wraps `useNuxtApp().$services` and returns `{ auth, workshop, editor, ai, export }`. Stores/components/composables call `useServices()` **inside actions/handlers/setup**, never at module top-level.

---

## 8. Supabase Client

**Decision: custom Nuxt plugin building a per-request client from `useRuntimeConfig()`. Do NOT use `@nuxtjs/supabase`.**

Justification:
- `@nuxtjs/supabase` **auto-redirects unauthenticated users to `/login`** using its own cookie-based session — this directly conflicts with the app's custom Pinia/localStorage guard (§6) and the **mock-by-default** mode (the app must run with zero Supabase config). Disabling that (`redirect: false`) plus the module's required `url`/`key` at boot would make Supabase a hard dependency the mock path doesn't want.
- The app talks to a **non-public `juanleme` schema** via per-query `.schema('juanleme')` and to Edge Functions — it does not need the module's auth-cookie helpers.
- Keeping a thin custom client preserves the existing `getSupabase()` contract with minimal change and keeps mock mode dependency-free.

`src/lib/supabase.ts` (transform) — factory, no module singleton, no `import.meta.env`:

```ts
import { createClient, type SupabaseClient } from '@supabase/supabase-js'

export function createSupabaseClient(url: string, key: string): SupabaseClient {
  if (!url || !key) {
    throw new Error(
      'NUXT_PUBLIC_SUPABASE_URL and NUXT_PUBLIC_SUPABASE_ANON_KEY are required when using Supabase mode',
    )
  }
  return createClient(url, key)
}
```

- The client is created **inside `plugins/services.ts`** (runs per request on the server → no cross-request session leak). The supabase service adapters (`src/services/supabase/*`) receive this client (injected via the registry `deps`) instead of importing `getSupabase()`. Replace every `getSupabase()` call site with the injected client. `.schema('juanleme')` calls are unchanged.
- For the **all-mock default** path, the plugin does NOT build a Supabase client (skips it), so a missing URL/key never throws in mock mode.
- True cookie-based SSR auth (`@supabase/ssr createServerClient`) is a documented follow-up, not required; the current adapters are client-fetched and localStorage-backed.

> If a future agent wants Universal SSR data-fetching with cookie auth, the migration path is `@supabase/ssr` per-request clients (server: `createServerClient` bound to `useRequestEvent()`, client: `createBrowserClient`). Explicitly out of scope here.

---

## 9. i18n

**Decision: custom vue-i18n Nuxt plugin. Do NOT adopt `@nuxtjs/i18n`.**

Justification (lower-risk, faithful):
- `@nuxtjs/i18n` v9 (the Nuxt 4 line) **restructures the `i18n/` directory** (`restructureDir`, `langDir: 'locales'`, `i18n.config.ts`), **renames the locale `iso` → `language`**, and expects a specific file convention. Our locales are **plain TS objects** (`zh-CN.ts`, `en-US.ts`) loaded programmatically, and `LangToggle.vue` imports `LANG_STORAGE_KEY` from `@/i18n`. Adopting the module would force a directory/config rewrite, risk purging/relocating the verbatim Chinese copy, and break the `@/i18n` re-export contract.
- A custom plugin reuses `createI18n({ legacy:false, fallbackLocale:'zh-CN', globalInjection:true, messages })` **verbatim**, keeps `useI18n()` working in every component, and only fixes the one SSR blocker: the top-level `localStorage.getItem`.

**Split `src/i18n/index.ts` into two files:**

`src/i18n/index.ts` (reduced — constant + message re-exports only, **no `createI18n`, no localStorage**):

```ts
export { default as zhCN } from './locales/zh-CN'
export { default as enUS } from './locales/en-US'
export const LANG_STORAGE_KEY = 'juanleme-lang'
export const DEFAULT_LOCALE = 'zh-CN'
```

`src/plugins/i18n.ts` (create — runs on server AND client; locale resolved safely):

```ts
import { createI18n } from 'vue-i18n'
import { zhCN, enUS, LANG_STORAGE_KEY, DEFAULT_LOCALE } from '@/i18n'

export default defineNuxtPlugin((nuxtApp) => {
  // SSR 默认 zh-CN；客户端再读取 localStorage 中保存的语言，避免服务端 localStorage 未定义崩溃
  let locale: 'zh-CN' | 'en-US' = DEFAULT_LOCALE
  if (import.meta.client) {
    const saved = localStorage.getItem(LANG_STORAGE_KEY)
    if (saved === 'en-US' || saved === 'zh-CN') locale = saved
  }

  const i18n = createI18n({
    legacy: false,
    locale,
    fallbackLocale: 'zh-CN',
    globalInjection: true,
    messages: { 'zh-CN': zhCN, 'en-US': enUS },
  })

  nuxtApp.vueApp.use(i18n)
})
```

- `LangToggle.vue` keeps `import { LANG_STORAGE_KEY } from '@/i18n'` — still resolves (the constant moved but the export name and path are preserved).
- **Hydration note:** SSR renders with `zh-CN`; on the client the plugin may switch to a saved `en-US`. Because `globalInjection` + reactive `locale` apply after hydration, this is the same acceptable post-hydrate flip as the auth shell. If strict no-flip is required later, persist locale in `useCookie('juanleme-lang')` (read on both server and client) — follow-up, not required.
- Do **not** create `i18n.config.ts` and do **not** add an `i18n:` block to `nuxt.config.ts` (those are `@nuxtjs/i18n`-only).

---

## 10. Test Strategy Under Nuxt

`vitest.config.ts` (transform) — stop merging the deleted `vite.config.ts`; provide the `@`→`./src` alias explicitly so every `@/...` import and `vi.mock('@/...')` string literal keeps resolving:

```ts
import { defineVitestConfig } from '@nuxt/test-utils/config'
import { fileURLToPath } from 'node:url'

export default defineVitestConfig({
  test: {
    environment: 'happy-dom',
    globals: true,
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
})
```

- **Keep `environment: 'happy-dom'` and `globals: true` verbatim** — every store/component/util test depends on the DOM globals (`localStorage`, `document`, `URL`, `Blob`) and auto-imported `describe/it/expect`.
- **Keep the `@` → `./src` alias** — 5 supabase adapter tests use `vi.mock('@/lib/supabase')` string literals that must byte-match the SUT's own import specifier. This is why we keep `src/` and the `@`→`src` mapping (not `@`→root).
- **Pure-logic + mocked-module tests** (example, schema-isolation*, bundle-validation, supabase-*-adapter, useAiConversation, editor store) need no code change beyond alias resolution. They do not mount Nuxt; `@nuxt/test-utils` `defineVitestConfig` only supplies the resolver/env.
- **Component/view mount tests** (LangToggle, ThemeToggle, UserProfile) keep using `@vue/test-utils` `mount` with their own locally-created vue-i18n + pinia. They do NOT need `mountSuspended`. They rely only on `happy-dom`.

**Satisfying `import.meta.env` reads → runtimeConfig:**
- `registry.ts` and `lib/supabase.ts` no longer read `import.meta.env` (they take config as arguments, §7/§8), so importing them in tests no longer pulls env-reading code. `bundle-validation.test.ts` and `workshop.test.ts` import the now-pure registry cleanly.
- `registry.test.ts` relies on "no env → mock adapters." Preserve by calling `resolveBundleModes({})` (empty config → `apiMode` falsy → `'mock'`) inside the test, OR keep the test calling the all-mock path. Either way the registry default stays mock.
- `schema-isolation.test.ts` is the ONLY test that stubs env: change `vi.stubEnv('VITE_SUPABASE_URL', ...)` / `VITE_SUPABASE_ANON_KEY` to the new key names `NUXT_PUBLIC_SUPABASE_URL` / `NUXT_PUBLIC_SUPABASE_ANON_KEY`, OR (cleaner) refactor it to call `createSupabaseClient(url, key)` directly with literal args and drop the env stub. Recommended: refactor to direct-arg call since `lib/supabase.ts` no longer reads env.
- No `setupFiles` exist or are needed today.

---

## 11. Execution Order (recommended for generation agents)

1. `package.json`, `nuxt.config.ts`, `tailwind.config.js`, `tsconfig.json`, delete `vite.config.ts`/`tsconfig.app.json`/`tsconfig.node.json`/`postcss.config.js`/`index.html`/`main.ts`/`src/router/`, create `.env`/`.env.example`. Run `pnpm install` then `pnpm exec nuxt prepare`.
2. `src/lib/supabase.ts` factory + `src/services/registry.ts` purification + `src/plugins/services.ts` + `useServices()` composable (`src/composables/useServices.ts`) + add `requireServices`. (§7/§8 — the critical service layer; everything downstream depends on it.)
3. Stores (`user`/`editor`/`workshop`) — localStorage guards + per-request timer + `useServices()`.
4. Supabase + mock adapters — inject client, guard localStorage.
5. i18n split + plugin (§9); `src/plugins/auth-init.client.ts`.
6. App shell: `app.vue`, `layouts/default.vue`, `layouts/empty.vue`, `error.vue` (§5).
7. Pages from views (§4c) + `middleware/auth.global.ts` (§6).
8. Components (NuxtLink, useId, alias) (§4e).
9. Tests config + env-stub fix (§10). Run `pnpm test`.
10. `pnpm build` (SSR build must succeed with no top-level browser-global throws).

---

## Appendix A: Pre-existing bugs surfaced (fix during migration)

- **`requireServices` was never exported** from `registry.ts` but is imported by `useAiConversation.ts` (3 call sites) and its test. The migration **defines** `requireServices()` (§7) — this resolves the broken import.
- `src/services/mock/index.ts` (`mockApi`) is **dead code** (not referenced by the registry) — delete.
- `generateMarkdown` in `utils/export.ts` hardcodes `'[未完成]'` instead of the i18n `export.notCompleted` key — pre-existing, NOT an SSR issue, leave as-is (behavior preservation).
