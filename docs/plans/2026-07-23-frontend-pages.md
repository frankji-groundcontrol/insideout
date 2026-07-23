# 2026-07-23 — Comprehensive frontend: page structure + design

Live checklist. Drives the work; check items off as they land. Ink & Seal /
Assembly world, whole-product scope, bilingual EN/zh-CN. Modes: public pages =
**Persuade**, authed pages = **Operate** (scanability + calm, brand in details).

## Direction (confirmed with the user 2026-07-23)

- "Design the pages structure and implement and design frontend. Do it
  comprehensively." → design the IA for the whole app, then bring **every** page
  to the Assembly/Ink & Seal standard (structure + design, not just theme).
- **Coach directive:** "the coach should be a sizable detachable sidebar with
  conversation and prompted card in conversation." → the PRD coach becomes a
  first-class **dockable right sidebar** (sizable ~440px, collapses so the editor
  reclaims full width, persistent reopen affordance, overlay drawer on mobile),
  holding the live conversation **plus clickable suggested-prompt cards rendered
  inline in the thread**.

## Product map → gaps (from the Understand pass)

The API already supports more than the UI surfaces:

- **No wayfinding** — NavBar brand → `/`, no `/dashboard` link, no breadcrumbs.
- **Workspace members & settings** — `listMembers` / `updateMemberRole` /
  `removeMember` / `update` / `remove` exist; **zero UI**.
- **PRD revisions** — `listRevisions` exists; the "Snapshot revision" button
  writes into a void (no reader UI).
- Everything is themed; only the landing is *designed*.

## Target IA (page structure)

Two realms. Public (Persuade): `/` landing (done), `/login`, `/register`,
`error.vue`. Authed (Operate, behind `auth.global`): everything else.

- Global chrome (NEW): `PageHeader` (breadcrumb + serif title + actions slot),
  `BaseModal`, `BaseEmptyState`, `CoachPanel`; `NavBar` brand→/dashboard + user
  menu; `error.vue` restyle + i18n.
- `/dashboard` — greeting, workspace cards, create/join via modal.
- `/profile` — PageHeader, BaseInput fields, avatar (preview-only).
- `/workspace/[id]` — board; PageHeader(Dashboard / workspace), project cards,
  New Project modal; actions: Ideas, Settings.
- `/workspace/[id]/ideas` — inbox; capture card, idea rows, modal drop-confirm.
- `/workspace/[id]/settings` (NEW) — General (rename/desc/invite code),
  Members (list / role / remove), Danger zone (delete). Admin-gated mutations.
- `/projects/[id]` — PageHeader(Dashboard / workspace / project); **roadmap as
  the primary stage** (key feature) + activity rail (GitHub sync, post update,
  feed). 404 handling; project edit/delete (admin).
- `/prd/[id]` — PageHeader; editor + **detachable CoachPanel**; grouped action
  bar; revisions link.
- `/prd/[id]/revisions` (NEW) — revision list + read-only snapshot view.
- `/prd/[id]/export` — PageHeader, styled markdown, copy-to-clipboard,
  download/print.

## Phase 1 — foundation (shared, done first; all i18n keys added here)

- [x] i18n: add every new key to `en-US.ts` + `zh-CN.ts` (breadcrumbs, settings,
      members, revisions, coach panel/suggestions, empty states, error page).
- [x] `components/common/BaseModal.vue` — accessible dialog (focus trap, Esc,
      backdrop, `aria-labelledby`, return focus), token-based.
- [x] `components/common/BaseEmptyState.vue` — icon slot + title + hint + action.
- [x] `components/layout/PageHeader.vue` — breadcrumb trail + serif title +
      subtitle + `#actions` slot (+ `#title` slot for editable titles).
- [x] `components/layout/NavBar.vue` — brand → `/dashboard` when authed; user
      menu (Profile, Logout); keep Lang/Theme toggles.
- [x] `error.vue` — Ink & Seal restyle, i18n, 404 vs 500 copy.

## Phase 2 — the coach (user directive)

- [x] `components/prd/CoachPanel.vue` — self-contained: resolves the PRD's
      conversation, wires `useCoachStream`, emits `prd-updated`.
- [x] Sizable (~440px) docked right sidebar; collapse → editor full width;
      persistent reopen rail/button; overlay drawer + backdrop < lg; open state
      persisted.
- [x] Header: title + stage stepper (clarify→draft→critique→finalize) + detach.
- [x] Fact ledger (collapsible) + conversation thread + streaming + rate-limit.
- [x] **Suggested-prompt cards** keyed by stage, inline in the thread; click →
      send. i18n.
- [x] Rewire `prd/[id]/index.vue` to host `CoachPanel`; editor flexes.

## Phase 3 — authed pages

- [x] `dashboard.vue`
- [x] `profile.vue`
- [x] `workspace/[id]/index.vue` (board)
- [x] `workspace/[id]/ideas.vue`
- [x] `workspace/[id]/settings.vue` (NEW: general + members + danger)
- [x] `projects/[id].vue` (roadmap-primary two-column)
- [x] `prd/[id]/index.vue` (editor + CoachPanel host)
- [x] `prd/[id]/revisions.vue` (NEW)
- [x] `prd/[id]/export.vue`

## Phase 4 — public auth pages + verify

- [x] `login.vue` / `register.vue` — Ink & Seal door (brand, serif, calm).
- [x] `nuxi typecheck` + `pnpm test` + `pnpm build` green.
- [x] impeccable `detect.mjs` anti-pattern scan; adversarial critique workflow.
- [x] Browser-render each page light + dark (authed pages need backend + DB;
      real verification, no mocks).

## Phase 5 — markdown rendering + positioning (user directive, 2026-07-23)

- [x] Coach messages render real markdown (not raw `**`/`*`): `utils/markdown.ts`
      (marked + dompurify, SSR-safe) + `common/MarkdownBody.vue`, wired into
      `CoachPanel` history + streaming bubble.
- [x] Reframe coding-implying copy to idea-shaping / roadmap-definition (the app
      does not write code): 5 strings per locale — hero "Shape your idea…",
      "Start shaping" CTAs, PRD button "Draft the roadmap" (zh 规划路线图),
      coach finalize "hand off to the team".
- [x] Verify: typecheck/tests/build green, detector clean, browser light + dark.
      Recorded in `changelogs/2026-07-23-coach-markdown-and-positioning.md` +
      BUG-012.

## Deferred (note, don't build)

- Roadmap drag-reorder (`roadmap.move` API exists; HANDOFF known limitation).
- Idea inline edit; project-update edit/delete (APIs exist) — light, follow-up.
- Avatar real upload (HANDOFF known limitation — preview-only).
- Server-driven coach suggestions (belongs to the 2026-07-21 agent-harness
  plan); this phase ships client-curated stage-keyed cards.
