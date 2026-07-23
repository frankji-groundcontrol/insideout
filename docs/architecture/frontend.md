# Frontend

Nuxt 4 Universal SSR app under `app/` (`srcDir: 'src'`), package-managed with
pnpm. Pinia for state, vue-i18n for zh-CN (default) + en-US, Tailwind CSS with
the "Ink & Seal" (国风留白) semantic token system.

## Transport: same-origin only

The browser and Nuxt SSR both hit same-origin `/api/v1/**`; a Nitro
`routeRules` proxy in `app/nuxt.config.ts` forwards those to the Go server
(`NUXT_API_INTERNAL_BASE`, default `http://127.0.0.1:8080/api/v1`). This keeps
the httpOnly auth cookies first-party — no CORS, no `SameSite` pitfalls, and
SSR requests carry the same cookies. There is no other backend path: the mock
service mode and the Supabase adapters from the JuanLeMe era were deleted
outright (the frontend's only service implementation is `src/services/api/`).

## Layout

```
app/src/
  pages/          route components: landing, login/register, dashboard,
                  workspace/[id] (project board), workspace/[id]/ideas,
                  projects/[id], prd/[id] (PRD workspace + coach chat),
                  prd/[id]/export, profile
  components/     common/ (BaseButton, BaseInput, BaseCard, BaseBadge,
                  PrdStatusBadge, LangToggle, ThemeToggle), layout/ (NavBar,
                  AppFooter)
  services/       api/ — one adapter per backend resource (auth, workspace,
                  project, idea, prd, coach, export); http.ts holds apiFetch
                  and the 429/503 error mapping; registry.ts wires the bundle
  stores/         user.ts — session state hydrated from the cookie session
  composables/    useCoachStream.ts (SSE-driven coach chat state), useTimeAgo
  middleware/     auth.global.ts — runs on server and client
  i18n/           locales/zh-CN.ts, en-US.ts + a key-parity test
  assets/         tokens.css — the Ink & Seal CSS custom properties
  types/          domain types + service interfaces
```

## Design tokens

`src/assets/tokens.css` defines semantic CSS custom properties (light + dark
under a `.dark` class); `tailwind.config.js` maps them to utilities
(`bg-surface-*`, `text-fg-*`, `border-stroke-*`, `bg-btn`/`text-btn-fg`,
`bg-status-*-bg/-fg`, `text-seal`, `rounded-control/card/pill/hero`).
Components use only these semantic utilities — no raw Tailwind palette colors
and no `dark:` variant sprawl. Two gotchas already hit and recorded: Tailwind
cannot see dynamically interpolated class strings
([BUG-003](../issues/2026-07-20-bug-003-tailwind-dynamic-class-interpolation.md)), and
token CSS must be loaded via `nuxt.config.ts` `css:` before Tailwind's layers.

## Coach chat

`src/composables/useCoachStream.ts` consumes the SSE contract described in
[the agent doc](prd-coach-agent.md): it maintains messages, streaming text,
PRD-section refresh triggers, stage changes, and the rate-limit countdown
(the preserved `APP_THROTTLE`/`CIRCUIT_OPEN` error shapes). History loads
from `GET /api/v1/conversations/{id}/messages`.

## Known limitations

Tracked in [`docs/TODO.md`](../TODO.md): avatar upload is a local-preview
placeholder; theme/locale persistence still uses `localStorage` (not cookies,
so SSR first paint can flash); PRD section editors are plain textareas.
