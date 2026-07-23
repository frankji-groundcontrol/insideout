# Issues

Concrete, dated problem records: bugs with root causes and fixes, structural
debt, deferred splits, and open questions that need a decision. Compact
issues are single dated files (`YYYY-MM-DD-title.md`); issues that accumulate
evidence get a dated folder. English only.

Bug records keep their historical `BUG-NNN` identity in the filename and
prose (they are cross-referenced by number throughout `docs/`); new bug
records continue the numbering (next: BUG-012).

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

## Other open issues

- [2026-07-23 — Backend pre-commit review findings](2026-07-23-backend-precommit-review-findings.md)
  — deferred hardening/correctness follow-ups from the adversarial review before
  the first public backend commit (2 medium concurrency/data-integrity bugs, 6
  low items). No push-blockers.
- [2026-07-21 — Tracked coding-tool scratch directories](2026-07-21-tracked-tool-scratch-dirs.md)
  — `.sisyphus/`, `.trae/`, `review/` are committed but are JuanLeMe-era tool
  scratch; untracking them needs a user decision.
