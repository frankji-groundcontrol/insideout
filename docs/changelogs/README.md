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
