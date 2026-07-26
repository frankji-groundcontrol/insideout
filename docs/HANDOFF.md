# HANDOFF

Living handoff for the next agent (Claude, Codex, human) taking over. Update
this on every checkpoint commit so a fresh agent can resume without re-reading
history. Keep it short and current.

## Current state

The InsideOut rewrite (Go backend + PostgreSQL/RLS + Nuxt 4 SSR frontend +
direct-Anthropic PRD Coach) is implemented and verified against the real
database and a real LLM endpoint. See
[docs/changelogs/2026-07-20-go-rewrite-and-rls-cutover/](changelogs/2026-07-20-go-rewrite-and-rls-cutover/index.md)
for what changed and how it was verified, and
[docs/architecture/](architecture/index.md) for the current system shape. The
old `juanleme` schema is gone (data migrated, then dropped).

On 2026-07-22 the product pivoted from "a PRD tool" to **idea → reality**:
branched-tree roadmaps on projects, GitHub progress sync, AI "build the MVP"
(PRD → generated roadmap + node expand). See
[docs/changelogs/2026-07-22-idea-to-reality/](changelogs/2026-07-22-idea-to-reality/index.md)
and the [plan](plans/2026-07-22-idea-to-reality.md).

On 2026-07-23 that day's Prisma cinematic theme detour was **reverted** to the
committed **国风留白 / Ink & Seal** world (celadon, sumi-ink, vermilion seal,
Noto Serif SC display; light + dark via `prefers-color-scheme`), and the public
landing was rebuilt as **「The Assembly」** — an ink build-instruction narrative.
Three a11y fixes landed from an adversarial critique (synchronous reduced-motion
gate, `BaseButton to` prop so CTAs aren't `<a><button>`, status-info chip
contrast). See
[docs/changelogs/2026-07-23-ink-seal-landing/](changelogs/2026-07-23-ink-seal-landing/index.md),
[PRODUCT.md](../PRODUCT.md) / [DESIGN.md](../DESIGN.md), and
[docs/design-system/CHANGELOG.md](design-system/CHANGELOG.md) `0.4.0`.

Also on 2026-07-23 the **whole authed frontend** was designed + implemented to
the Ink & Seal standard per the
[frontend-pages plan](plans/2026-07-23-frontend-pages.md): shared chrome
(`PageHeader` breadcrumbs, `BaseModal`, `BaseEmptyState`, restyled `NavBar` +
`error.vue`), workspace settings/members, PRD revisions reader, and the PRD
coach as a **~440px detachable right sidebar** with stage stepper, fact ledger,
and clickable suggested-prompt cards inline. Two follow-up directives landed the
same day: coach messages render real **markdown** (`utils/markdown.ts` +
`common/MarkdownBody.vue`, marked + dompurify, SSR-safe), and copy was reframed
from "build/code" to **idea-shaping + roadmap-definition** (the app writes no
code). See
[docs/changelogs/2026-07-23-coach-markdown-and-positioning.md](changelogs/2026-07-23-coach-markdown-and-positioning.md)
and [BUG-012](issues/2026-07-23-bug-012-project-list-null-latest-update-scan.md).

## In flight / next steps

- Collaborative-canvas plan is **complete** (2026-07-26): all workstreams A–D
  and tasks T1–T10 landed. The last, **T9 / Workstream D** (card glide
  transitions + neutral sibling bands + full-route minimap), shipped with a
  load-bearing routing fix (a `pages/x/[id].vue` without `<NuxtPage />` was
  silently swallowing the canvas route — see the
  [learning note](learning/2026-07-26-nuxt-dynamic-route-parent-shadowing.md))
  and a one-line popover-stacking fix an adversarial pass surfaced (a trapped
  Add button could silently reparent a node). Live-verified light/dark ×
  embedded/full, 52/52 + typecheck. Plan:
  [docs/plans/2026-07-24-roadmap-canvas-collab.md](plans/2026-07-24-roadmap-canvas-collab.md);
  changelog:
  [docs/changelogs/2026-07-26-roadmap-canvas-workstream-d.md](changelogs/2026-07-26-roadmap-canvas-workstream-d.md).
- Open follow-ups live in [docs/TODO.md](TODO.md)
  ("Known Limitations": avatar upload placeholder, theme/locale in
  localStorage, plain-textarea PRD editors, GitHub sync is owner/admin +
  public-only, no roadmap drag-reorder UI) and [docs/issues/](issues/README.md).

## Key context / gotchas

- All work must be verified against a real database — see
  [docs/practices/](practices/README.md) and the RLS gotcha list in
  [docs/architecture/database-and-rls.md](architecture/database-and-rls.md).
- `DATABASE_URL` may point at a shared multi-tenant instance; never write
  outside the `insideout` schema.
- **Raw `psql` shows zero rows until you set the RLS user.** `FORCE ROW LEVEL
  SECURITY` hides every table unless the session runs
  `SELECT set_config('app.user_id', '<uuid>', false)` first (the store does
  this per-request via `withUserContext`). An empty result is usually a
  missing context, not missing data. Also strip any `?pgbouncer=…` query
  string from `DATABASE_URL` before handing it to `psql`.
- **`nuxt dev` can serve a bare HTTP 426 "Upgrade Required" to IPv4 clients.**
  Root-caused 2026-07-26 (Nuxt 4.4.8 / Vite 7.3.5, Node v25.6.0): with no
  `HOST` set the app binds its HTTP server to IPv6 `[::1]` *and* a separate
  IPv4 `*:port` socket that is an upgrade-only (HMR/WebSocket) listener — so
  `127.0.0.1` and `localhost` (resolves to IPv4 here) hit the 426 socket,
  while `http://[::1]:PORT` serves the app (200, `x-powered-by: Nuxt`). It
  is an environmental bind split, **not** app code and **not** a Node version
  bug (a bare `node` HTTP server on the same port returns 200). Fix: run the
  dev server with `HOST=127.0.0.1` (one IPv4 socket serves the app), or
  reach it via `http://[::1]:PORT`. See the
  [learning note](learning/2026-07-26-nuxt-dev-ipv6-426.md).
- Bug records live in [docs/issues/](issues/README.md) as dated English
  files keeping their `BUG-NNN` identity (the former bilingual
  `docs/en/BUGS/` + `docs/cn/BUGS/` pair was retired on 2026-07-21 per user
  direction — docs are English-only now).

## How to verify

```
cd server && go build ./... && go vet ./... && go test ./...
# with a real DATABASE_URL in ../.env:
set -a && source ../.env && set +a && go test ./internal/store/... -run TestAuthz -v
cd ../app && pnpm test && npx nuxi typecheck && pnpm build
docker compose build
```
