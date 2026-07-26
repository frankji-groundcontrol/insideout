# Issues

Concrete, dated problem records: bugs with root causes and fixes, structural
debt, deferred splits, and open questions that need a decision. Compact
issues are single dated files (`YYYY-MM-DD-title.md`); issues that accumulate
evidence get a dated folder. English only.

Bug records keep their historical `BUG-NNN` identity in the filename and
prose (they are cross-referenced by number throughout `docs/`); new bug
records continue the numbering (next: BUG-014).

## Bug records

- [BUG-001 — happy-dom localStorage](2026-07-20-bug-001-happydom-localstorage.md)
  — test-infra localStorage capability check broke under happy-dom.
- [BUG-002 — BaseInput SSR hydration](2026-07-20-bug-002-baseinput-ssr-hydration.md)
  — `Math.random()` default id mismatched between server and client render.
- [BUG-003 — Tailwind dynamic class interpolation](2026-07-20-bug-003-tailwind-dynamic-class-interpolation.md)
  — template-literal class names are invisible to Tailwind's scanner.
- [BUG-004 — Compose nested interpolation](2026-07-20-bug-004-compose-nested-interpolation.md)
  — docker-compose can't nest `${...}` inside `:-` defaults; `DATABASE_URL`
  became explicitly required.
- [BUG-005 — go.mod toolchain mismatch](2026-07-20-bug-005-gomod-toolchain-mismatch.md)
  — `go mod init` inherited the dev machine's toolchain version, breaking
  Docker builds.
- [BUG-006 — pnpm ignored build scripts](2026-07-20-bug-006-pnpm-ignored-build-scripts.md)
  — `pnpm-workspace.yaml` (holding the build allow-list) was missing from
  the Dockerfile's manifest COPY.
- [BUG-007 — RLS against real Postgres](2026-07-20-bug-007-rls-against-real-postgres.md)
  — five distinct bugs (schema bootstrap privileges, policy self-recursion,
  SECURITY DEFINER under FORCE RLS, row-locking vs cross-table policies,
  simple-protocol jsonb encoding) found only by running against a live
  database.
- [BUG-008 — shared-instance DB provisioning](2026-07-20-bug-008-shared-instance-db-provisioning.md)
  — dedicated-instance assumptions (role ownership, connection paths,
  pooler modes) broke against the real shared-instance target.
- [BUG-009 — langchaingo removed](2026-07-21-bug-009-langchaingo-removed.md)
  — the unmaintained library hid real answers behind extended-thinking
  content blocks; replaced with a direct Anthropic client.
- [BUG-010 — SSE Flusher swallowed by middleware](2026-07-21-bug-010-sse-flusher-swallowed-by-logging-middleware.md)
  — interface-embedding in the logging middleware dropped `http.Flusher`,
  silently breaking every streaming response.
- [BUG-011 — locale SSR hydration mismatch](2026-07-23-bug-011-locale-ssr-hydration-mismatch.md)
  — SSR always renders `zh-CN` (no `localStorage` on the server) while the
  client hydrates to the saved `en-US`, so every localized page logs a
  hydration text mismatch. Fix: read the locale from a cookie during SSR.
- [BUG-012 — project list NULL latest-update scan](2026-07-23-bug-012-project-list-null-latest-update-scan.md)
  — `ListProjectsForWorkspace`'s `LEFT JOIN LATERAL` yields `NULL` `lu.*`
  columns for projects with no updates; scanning them into value
  `string`/`time.Time` 500'd the workspace board. Fix: pointer fields.
- [BUG-013 — GitHub-sync repo link is a relative URL](2026-07-25-bug-013-github-sync-relative-repo-link.md)
  — the sync card binds the stored `owner/repo` straight into the anchor
  `href`, so an `owner/repo` link (the placeholder invites it) resolves
  relative to the app origin and 404s. Fix: normalize to
  `https://github.com/…` for display only.

## Other open issues

- [2026-07-26 — Roadmap canvas: adversarial-verify follow-ups (Workstream D)](2026-07-26-roadmap-canvas-adversarial-followups.md)
  — from the T9 adversarial pass: the popover-trapped-under-sibling defect is
  **fixed** (one-line `.rm-card:focus-within` lift; it could silently reparent a
  node); two pre-existing low items remain open — no keyboard directional pan and
  two pre-existing motions not reduced-motion-gated — each with a bounded fix prompt.
- [2026-07-26 — Canvas fit-to-view zooms wide roadmaps out too small](2026-07-26-canvas-fit-too-small-on-wide-trees.md)
  — `fitTo` clamps a wide tidy-tree to ~0.35–0.5, so the first impression of a
  large roadmap is tiny. Enhancement, not a defect: add a readable-min fit clamp
  (~0.6) with root-anchoring, keeping the user zoom floor untouched.
- [2026-07-25 — Build-from-PRD runs the LLM before the conflict guard](2026-07-25-build-from-prd-runs-llm-before-conflict-guard.md)
  — `handleBuildFromPrd` calls `PlanMVP` (LLM) before `ReplaceRoadmapTree`
  returns the non-empty 409, so a first build on a populated roadmap pays the
  call then discards it. Correctness is safe (in-tx advisory-locked guard is
  authoritative); deferred as a wasted-cost/latency fix.
- [2026-07-25 — Canvas failure feedback is a banner, not the specced toast](2026-07-25-canvas-failure-feedback-is-a-banner-not-the-specced-toast.md)
  — collab A3's stacked `useToast` + delete descendant-count confirm shipped
  as a simpler in-canvas error banner + static confirm; corrects the collab
  plan's T4 line. No mutation fails silently, so the intent is met.
- [2026-07-23 — Backend pre-commit review findings](2026-07-23-backend-precommit-review-findings.md)
  — deferred hardening/correctness follow-ups from the adversarial review before
  the first public backend commit (2 medium concurrency/data-integrity bugs, 6
  low items). No push-blockers.
- [2026-07-26 — Backend optimization review, deferred findings](2026-07-26-backend-optimization-deferred.md)
  — the items NOT fixed in the 2026-07-26 optimization pass, headed by a HIGH
  6-digit invite-code brute-force (cross-tenant membership), plus a ConvertIdea
  double-create race, auth rate-limiting, a PRD CAS race, and six low SQL/error
  items — each with a bounded fix prompt.
- [2026-07-21 — Tracked coding-tool scratch directories](2026-07-21-tracked-tool-scratch-dirs.md)
  — `.sisyphus/`, `.trae/`, `review/` are committed but are JuanLeMe-era tool
  scratch; untracking them needs a user decision.
