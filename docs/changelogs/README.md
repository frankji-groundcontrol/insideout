# Changelogs

Dated records of what changed in this repository, written for maintainers.

## Convention

- **Ordinary change**: one dated file, `YYYY-MM-DD-short-slug.md`.
- **Large change** (multi-phase, multiple subsystems): a dated folder,
  `YYYY-MM-DD-short-slug/`, with an `index.md` linking focused child files
  (e.g. `summary.md`, `verification.md`, `migration-notes.md`).

Each record states what changed, how it was verified, and what an operator of
an existing deployment needs to do. Cite sources (plans, bug book entries,
code) with relative links.

## Records

- [2026-08-20 — Owner/app roles rolled out to the shared Supabase instance](2026-08-20-owner-app-roles-shared-instance.md)
  — scoped ownership transfer + grants migration via admin session;
  FORCE RLS restored; Railway deployed with `DATABASE_OWNER_URL`
  (`railway up`, not stale-image `redeploy`); local `.env` repaired.
- [2026-08-19 — insideout_owner + insideout_app](2026-08-19-owner-app-roles.md)
  — two NOSUPERUSER roles; SECURITY DEFINER owned by owner, not superuser.
- [2026-08-19 — Restore Ink & Seal on Flutter](2026-08-19-restore-ink-seal.md)
  — Material 3 visual lock reversed; Flutter ThemeData + AuthDoor chrome
  use recovered celadon/ink/seal tokens.
- [2026-08-18 — Delete leftover Nuxt `app/`](2026-08-18-delete-nuxt-app.md)
  — Nuxt tree removed; compose is postgres + server; env catalog has
  no frontend dotenv contract.
- [2026-08-18 — Railway `app` serves Flutter web, not Nuxt](2026-08-18-flutter-web-host.md)
  — `client/` nginx + Flutter web on the public app domain; same-origin
  `/api/` proxy to `server`; `app/` source kept.
- [2026-08-18 — Railway server uses shared Supabase, dedicated Postgres removed](2026-08-18-railway-supabase.md)
  — production `DATABASE_URL` is `insideout_app` on the session pooler
  (5432); Railway plugin Postgres deleted after a live write check.
- [2026-08-18 — LLM env names and messages/responses schema](2026-08-18-llm-env.md)
  — `INSIDEOUT_LLM_BASE_URL` / `API_KEY` / `MODEL` / `SCHEMA` replace
  `ANTHROPIC_*` and `AI_MODEL`; chat URL is `{base}/messages` or
  `{base}/responses` with no inserted `/v1`.
- [2026-08-17 — Flutter client started (web + iOS + Android)](2026-08-17-flutter-client.md)
  — `client/` Flutter app against the existing Go API; login/register
  return Bearer tokens; refresh/logout accept a JSON refresh token;
  Railway `server` is now publicly reachable for native clients.
- [2026-08-17 — First Railway deploy + build-time API proxy](2026-08-17-railway-deploy.md)
  — public app at `https://app-production-591e.up.railway.app`; server
  honors `PORT` when `INSIDEOUT_ADDR` is unset; Nuxt Dockerfile bakes
  `NUXT_API_INTERNAL_BASE` so the same-origin API proxy does not target
  localhost.
- [2026-08-13 — Task board and handoff responsibility correction](2026-08-13-task-board-and-handoff.md)
  — makes `docs/plans/README.md` the authoritative concurrent task board and
  reduces `docs/HANDOFF.md` to one human-readable resume path with worktree
  warnings; docs only.
- [2026-08-13 — Canonical product-experience baseline](2026-08-13-product-experience-baseline.md)
  — consolidates the completed product interview into `PRODUCT.md`: a
  version-first Coach, audience projections, human Commit/Branch/Diff/Merge,
  one role-aware and deadline-bound Roadmap Graph, Git evidence, and shared
  Web/CLI/MCP/Agent context. Docs only; target experience is kept distinct
  from current implementation.
- [2026-07-30 — Env catalog + TUI, contract-scoped propagation, honest schema](2026-07-30-env-catalog-propagate.md)
  — `env.sh edit` (curses catalog of every variable and its state, secrets
  masked both ways) and `env.sh propagate` (generate `app/.env` scoped to the
  component's contract and stamped with the root file's checksum); `check`
  becomes component- and staleness-aware and `dev.sh` gates on it, so a stale
  copy blocks that launch instead of silently disagreeing with the root.
  `.env.example` corrected — nine optional variables were marked required and
  two consumed variables were absent — and `env.sh` split into three files,
  closing the ~350-line budget issue.
- [2026-07-28 — env.sh: interactive .env setup + validation; dev.sh preflights it](2026-07-28-envsh.md)
  — new `scripts/env.sh` (`init` = interactive tty-only setup that never
  prints values; `check` = non-interactive required-key validation) and
  `scripts/dev.sh` now runs `check` before launching anything, aborting with
  the named missing keys. Companion to the same-day SETENV.md guide.
- [2026-07-28 — Hands-on .env key-set guide](2026-07-28-setenv-guide.md)
  — new `docs/SETENV.md` walkthrough (create → choose each key → prove it)
  as the hands-on companion to `docs/usage/environment.md`'s variable
  reference; cross-links wired through the doc map, agent routers,
  environment.md, local-development.md, and the `.env.example` header.
- [2026-07-28 — dev.sh moves to the repo root with `-C <dir>`](2026-07-28-devsh-root.md)
  — the server-only env-export wrapper becomes one root `scripts/dev.sh -C
  server|app <cmd>` for any consumer (and grows the frontend propagation path
  for a non-default `NUXT_API_INTERNAL_BASE`); doc commands repointed
  root-relative, fixing bare `go run` invocations that never exported `.env`.
  No behavior change.
- [2026-07-27 — Canonical environment configuration guide](2026-07-27-env-guide.md)
  — new `docs/usage/environment.md` is the single source of truth for every
  environment variable (grouped by consumer, with required/default/meaning,
  the `.env`→process bridges, database setups, recipes, and fail-fast
  troubleshooting); the duplicated var tables in local-development.md collapse
  into links, and README/CLAUDE.md/deployment.md repoint at it. Companion to
  the same-day environment hygiene pass. Docs only, no behavior change.
- [2026-07-27 — Environment hygiene pass](2026-07-27-env-hygiene.md)
  — deleted the Supabase-era `app/.env.example` fossil, regrouped the root
  `.env.example` by consumer (Go backend / AI provider / docker-compose
  only — fixing `SERVER_PORT`/`APP_PORT` misfiled under Go/frontend
  headings), and added `server/scripts/dev.sh` so `./scripts/dev.sh <cmd>`
  replaces the hand-typed `set -a && source ../.env` incantation in CLAUDE.md,
  local-development.md, and HANDOFF.md. No behavior change.
- [2026-07-27 — Auth pages become one shared prompted floating modal](2026-07-27-auth-door-modal.md)
  — `/login` and `/register` are now thin pages rendering a shared `AuthDoor`
  shell: a paper panel over a dimmed scrim with real dialog semantics
  (focus trap, Escape, backdrop close → `/`, scroll lock, focus restore)
  extracted from `BaseModal` into `composables/useDialogA11y.ts` and reused by
  `BaseModal` unchanged; reduced-motion-gated seal stamp + surfacing motion;
  register loses its fake bordered-glyph placeholder for the real 印 seal.
  Frontend-only; routes, i18n, and the auth flow are unchanged.
- [2026-07-27 — New docs/design-qa/ record surface](2026-07-27-design-qa-surface.md)
  — a records surface for verbatim user design-QA feedback on page appearance
  and the frontend (each comment quoted exactly, with its resolution and files
  touched), plus a standing rule added identically to AGENTS.md and CLAUDE.md
  to record such comments there. Seeded with the auth-door QA thread.
- [2026-07-27 — Login page refined into the Ink & Seal door](2026-07-27-login-page-refinement.md)
  — the real 印 baiwen seal (was a fake bordered letter), the serif wordmark,
  and a paper card that surfaces like a modal, animated with the landing's
  reduced-motion-aware stamp + rise. Notes the pre-existing product-wide
  motion hydration warning (shared composable, not a regression).
- [2026-07-27 — PuHuiTi fonts: reference, don't redistribute](2026-07-27-puhuiti-reference-only.md)
  — removed the 4 self-hosted PuHuiTi WOFF2 binaries and purged them from git
  history (the font is proprietary "free to use", not redistributable — an
  open-source-style grant it does not carry). The font is now named in the sans
  stack as a reference only; rendering falls back to Noto Sans SC / system CJK.
  Honest attribution added to the README; self-host claims corrected in
  DESIGN.md / PRODUCT.md.
- [2026-07-27 — Repository structure cleanup](2026-07-27-repo-structure-cleanup.md)
  — deleted 12 unreferenced root verification screenshots, gitignored `.claude/`,
  and (user-approved) untracked the JuanLeMe-era `.sisyphus/` / `.trae/` /
  `review/` tool scratch and purged it from git history (`git filter-repo` +
  force-push). Closes the tracked-tool-scratch-dirs issue.
- [2026-07-27 — Live end-to-end smoke test for all five surfaces](2026-07-27-live-smoke-test.md)
  — `server/scripts/smoke.sh`: one rerun-safe script that boots the server and
  drives the PRD coach SSE, AI roadmap, GitHub sync, project-updates timeline,
  and cross-tenant authz over real HTTP against the real DB (curl + jq, no
  mocks), exiting non-zero on any failure. Closes task #75; 48/48 green twice.
- [2026-07-27 — Cross-surface security hardening pass](2026-07-27-security-hardening-pass.md)
  — 22 workflow findings (F1–F22) + 4 code-traced items (R1–R4) fixed at root
  cause across the PRD coach, AI roadmap, GitHub sync, project-updates
  timeline, and authz: invite code raised to 128 bits, four concurrency races
  closed (ConvertIdea, EnsureProject, PRD-section CAS, revision snapshot),
  DecodeJSON rejects trailing bytes, upstream error detail kept off the wire.
  Resolves six 2026-07-26 deferred items; real-DB tests throughout.
- [2026-07-26 — Backend optimization pass](2026-07-26-backend-optimization-pass.md)
  — five audit findings fixed at root cause: request-body cap + server
  timeouts, the dispatcher conversation-lock TOCTOU, the roadmap move/create
  cycle race, the coach success-path detached-context persist, and unchecked
  SSE frame-index assertions.
- [2026-07-26 — Roadmap canvas: Workstream D (transitions + sibling bands + minimap) — collab T9](2026-07-26-roadmap-canvas-workstream-d.md)
  — closes the collaborative-canvas plan. Cards glide (keyed wrappers + a
  reduced-motion-gated `transform` transition), parallel tracks get a neutral
  sibling band, and the full route gains a pointer-only minimap — plus a
  load-bearing routing fix (a `pages/x/[id].vue` without `<NuxtPage />` was
  silently swallowing the canvas route) and a one-line popover stacking fix the
  adversarial pass surfaced (a trapped Add button could silently reparent a
  node). Live-verified light/dark × embedded/full; 52/52 tests + clean typecheck.
- [2026-07-26 — Roadmap canvas: B3 attribution (collab T6)](2026-07-26-roadmap-canvas-collab-attribution.md)
  — roadmap nodes gain provenance: `created_by`/`updated_by`
  (`uuid → users ON DELETE SET NULL`), resolved to display names via a
  `LEFT JOIN`, surfaced as a neutral last-editor initial + "created by X ·
  edited by Y" tooltip on each card; the insert policy now checks
  `created_by = current_user_id()`. Live-verified against this repo as
  subject: GitHub sync persisted 20 commits (idempotent re-sync `added:0`)
  and attribution columns round-tripped through the real DB, and the card
  visual (neutral roundel + tooltip) is now confirmed in light and dark.
  The earlier browser pass was blocked not by Node 25 but by an IPv4/IPv6
  bind split in the Nuxt dev server (fixed with `HOST=127.0.0.1`).
- [2026-07-26 — Roadmap canvas: review mode + edge re-semantics (collab T7–T8)](2026-07-26-roadmap-canvas-collab-review-and-edges.md)
  — frontend-only collab lanes: a truly read-only review/present mode
  (`v-if` + `:disabled`, `?review=1` deep-link, drag disabled) with a quiet
  chip, i18n-aware card freshness, a prospective drag edge, and neutral
  hairline edges with 7-day hot-branch emphasis (2.5px in dark). Live-verified
  light/dark × 中文/EN; 44/44 tests + clean typecheck.
- [2026-07-26 — Roadmap node actions get explicit `aria-label`s](2026-07-26-roadmap-node-action-aria-labels.md)
  — the five per-node canvas action buttons (status-cycle, Break down with AI,
  Add sub-task, Edit, Delete) relied on `title` alone for their accessible name;
  they now carry an explicit `:aria-label` mirroring `:title`, matching the
  toolbar buttons.
- [2026-07-26 — Backend optimization pass](2026-07-26-backend-optimization-pass.md)
  — five confirmed audit findings fixed at root cause: a 1 MiB request-body
  cap + server timeouts (unauth DoS), the dispatcher conversation-lock TOCTOU,
  the roadmap move/create per-project advisory-lock cycle race, the coach
  success-path detached-context persist, and checked SSE frame-index
  assertions. Deferred findings (incl. a HIGH invite-code brute-force) tracked
  in the [bug book](../issues/2026-07-26-backend-optimization-deferred.md).
- [2026-07-25 — Roadmap canvas: single-player hardening pass](2026-07-25-roadmap-canvas-hardening.md)
  — eight adversarial-review fixes close out the pan/zoom roadmap canvas
  (toolbar-aware fit, load-error state, pointercancel, mid-stream rate-limit
  mapping, popover flip, keyboard/touch action reveal, focus restore, mutation
  error surfacing); the single-player canvas is now implemented + hardened.
- [2026-07-24 — AI-generated baiwen seals for landing + navbar](2026-07-24-seal-image-generation.md)
  — replaced the landing's hand-drawn `bg-seal` chops and the text-only navbar
  brand with six codex-generated baiwen seal impressions (落墨/成文/分枝/盖印,
  印, 内外), margin-knocked-out to transparency, normalized to the
  `--color-seal` token, and served as 320px WebP.
- [2026-07-23 — WOFF2 fonts + scratch cleanup](2026-07-23-woff2-fonts-and-scratch-cleanup.md)
  — converted the 4 self-hosted PuHuiTi weights from TTF (~32 MB) to WOFF2
  (~16 MB, full CJK coverage kept) and deleted the Prisma-detour scratch
  screenshots/mock.
- [2026-07-23 — Bring the frontend rewrite + infra under version control](2026-07-23-frontend-version-control.md)
  — tracked the Nuxt 4 SSR frontend rewrite and deployment infra
  (docker-compose + Dockerfiles), removed the superseded `supabase/` backend,
  and self-hosted the PuHuiTi fonts. Completes the cutover: the whole new
  stack is now in git.
- [2026-07-23 — Coach markdown rendering + idea-shaping positioning](2026-07-23-coach-markdown-and-positioning.md)
  — coach messages render real markdown (marked + dompurify, SSR-safe) via a
  token-styled `MarkdownBody`, and copy was reframed from "build/code" to
  idea-shaping + roadmap-definition. Also records BUG-012 (workspace-board
  NULL-scan 500).
- [2026-07-23 — Ink & Seal reconciliation + landing rethink](2026-07-23-ink-seal-landing/index.md)
  — reverted the Prisma cinematic detour back to the committed Ink & Seal world,
  and rebuilt the public landing as 「The Assembly」 (ink build-instructions;
  three a11y fixes from an adversarial critique).
- [2026-07-22 — Idea → Reality](2026-07-22-idea-to-reality/index.md)
  — branched-tree roadmap on projects, GitHub progress sync, AI "build the
  MVP" (PRD → generated roadmap), and a full UI re-theme to the Prisma
  cinematic reference.
- [2026-07-21 — Documentation reorganization](2026-07-21-docs-reorganization.md)
  — clean-repo-org layout: routers, architecture/usage/learning/practices
  surfaces, bug book merged into `docs/issues/` (English-only), legacy
  retired to `docs/history/`, guardrail installed.
- [2026-07-20 — Go rewrite and RLS cutover](2026-07-20-go-rewrite-and-rls-cutover/index.md)
  — full backend rewrite to Go, frontend swap to the Go API, JWT+RLS
  defense-in-depth, and the juanleme → insideout data cutover.
