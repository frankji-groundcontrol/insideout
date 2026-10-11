# 2026-10-04 — Remediation Waves 3–4 (board promises, idea privacy, verification spine)

Final two waves of the
[multirole-review remediation](../review/2026-10-04-multirole/40-remediation-plan.md)
(register: [01-findings-register.md](../review/2026-10-04-multirole/01-findings-register.md));
plan of record [docs/plans/2026-10-04-remediation-wave-3-4.md](../plans/2026-10-04-remediation-wave-3-4.md).
Waves 0–1 landed in
[2026-10-04-remediation-wave-0-1.md](2026-10-04-remediation-wave-0-1.md),
Wave 2 in [2026-10-04-remediation-wave-2.md](2026-10-04-remediation-wave-2.md).
This closes the five-wave plan.

## What changed

### The board keeps its promises (FEAT-01, FEAT-02)

- Quiet flag (server): `projectView.flagged` on every project response —
  an `active`/`planning` project whose latest update (or creation, when it
  has none) is more than 14 days old is flagged; `done`/`paused`/`archived`
  are never flagged. Rule pinned by table tests
  (`server/internal/api/projects_test.go`, boundary at exactly 14 days).
- Board cards (client): each card now renders the quiet badge, the
  latest-update line (kind label, one-line content, relative timestamp —
  `latestUpdateAt` joined the `Project` model), and the board gained
  pull-to-refresh (nothing refreshed before). Widget test:
  `client/test/workspace_page_test.dart`.

### Idea privacy: private to the author by default (USER-05, FEAT-04)

Product decision (recorded in the plan):
[PRODUCT.md](../../PRODUCT.md)'s "private to its
author by default" and the README's personal-inbox promise win — no
workspace-shared inbox.

- Migration `20261004120000_ideas_private_by_default.sql`: the
  `ideas_select` RLS policy becomes author-or-admin (admins keep read
  access matching their existing drop authority; write policies
  unchanged).
- Store: `ListIdeas` filters to the author; `GetIdeaForMember` returns
  `ErrNotFound` for fellow members (existence not disclosed).
- Client: the ideas wall states the rule (`idea.privateHint`, en+zh).
- Behavior pinned at both layers by
  `server/internal/store/ideas_privacy_test.go`: fellow member cannot
  list/fetch, author and admin can, the RLS policy itself hides the row
  from a member's actor-context SELECT, and admin drop still works.
- Note for CLI/MCP users: `idea list` now returns only the calling
  user's ideas (same store path).

### Export downloads a real file (FEAT-03)

*Corrected 2026-10-11: one ID, two findings — FEAT-03 here is the
review-1 register's export-download finding; the same-day review-2 register
uses FEAT-03 for the agent-proposals web decision loop
([record](2026-10-04-remediation-review-2.md)).*

- New `client/lib/api/download_stub.dart` + `download_web.dart`
  (conditional import, the `sse_transport` pattern): markdown export
  saves a real `.md` download; the print format opens the print HTML in
  a new tab (browser print-to-PDF one keystroke away) with a `.html`
  download fallback. Non-web keeps the inline preview.

### Roadmap defaults to the Progress view (FEAT-05)

- New `client/lib/features/project/roadmap_progress_view.dart`: renders
  `GET /roadmap/progress` — deadlined Now with pressure pills, ≤3
  justified Next, needsDeadline, doneCount. The view toggle becomes
  Progress (default) | Bands | Canvas; Bands/Canvas stay for structure
  planning. Widget tests: `client/test/roadmap_page_test.dart` (server
  buckets, pressure, done count) and the updated
  `roadmap_canvas_test.dart` (toggle flow through all three views).

### Verification spine (PM-05, TEST-02, TEST-04, TEST-05, CQ-01, CQ-02)

- `.github/workflows/ci.yml` (new): on pull_request + push to `main` —
  - **server** job: `postgres:18` service, the two-role bootstrap
    mirroring `docker/postgres-init`, migrations applied via
    `insideout migrate`, then build/vet/test with `DATABASE_URL` set —
    and a skipped store test is a **red build** (TEST-02: the DB-gated
    suite must run, not skip).
  - **client** job: Flutter 3.47.0 `pub get` + `analyze
    --no-fatal-infos` (the 21 pre-existing infos are the accepted
    baseline; warnings/errors stay fatal) + `flutter test`.

    *Corrected 2026-10-11: the pin was 3.44.0 — the same-day review-2
    remediation pinned CI to the Dockerfile's Flutter 3.44.0
    ([record](2026-10-04-remediation-review-2.md)), and the repo has stayed
    on 3.44.x.*
  - **env-suites** job: `python3 -m unittest test_env_catalog
    test_env_writes`.

    *Superseded 2026-10-11: CI has since been minimized to a single head
    pipeline (owner-directed, repo public — see the
    [owner-deploy record](2026-10-11-owner-deploy-and-mail-truth.md)).*
- `server/internal/api/conversations_test.go` (new, DB-gated): POST a
  conversation message through the real handler stack against a stubbed
  coach — asserts `text/event-stream`, and that frame 1
  (`message_start`) is readable **while the coach is still blocked
  before frame 2** (buffering anywhere in the middleware chain would
  hang the test), then delta/message_end; a stranger's valid token gets
  404. This is the coach SSE endpoint's first HTTP-level coverage
  (TEST-04).
- `agentcontext_test.go`: the log-only open-questions branch is now
  `t.Errorf` — an empty openQuestions (the blank-section failure mode)
  fails the test instead of logging (TEST-05).
- `feature_wiring_test.dart` demoted to a named lint-tier backstop;
  behavioral coverage moved to the widget tests above plus the existing
  `versions_page_test` (CQ-02).
- CQ-01: dated modularity issues filed for `coach.go` (467),
  `anthropic.go` (408), and `models.dart` (478) —
  [coach.go](../issues/2026-10-04-coach-go-line-budget.md),
  [anthropic.go](../issues/2026-10-04-anthropic-go-line-budget.md),
  [models.dart](../issues/2026-10-04-models-dart-line-budget.md);
  `client.dart` already had
  [2026-09-18-flutter-apiclient-line-budget.md](../issues/2026-09-18-flutter-apiclient-line-budget.md).

## Verification

- `go build ./... && go vet ./...` clean; `go test ./...` all packages
  `ok` with `DATABASE_URL` set — store suite ran with **0 SKIP**.
- `flutter analyze --no-fatal-infos` clean; `flutter test` 65/65.
- `scripts`: `python3 -m unittest test_env_catalog test_env_writes`
  47/47; `./scripts/env.sh check` 5 ok / 0 fail (local `.env` now sets
  `INSIDEOUT_PUBLIC_URL`, satisfying wave 2.9's mail-key gate).
- Live stack walk (disposable compose stack, throwaway volume, signup
  mode flipped open via SQL): two users; board shows `flagged` after
  backdating one project 20 days and `latestUpdate*` after posting an
  update; user B's idea list is empty while user A (author) and the
  workspace admin see the idea, and user B's direct fetch 404s;
  `roadmap/progress` returns the deadline-truth shape; torn down with
  `down -v`.
- CI green (first run exposed a race in the new SSE test itself — a
  nonblocking "flushed-before-read" check could fire before the stub's
  channel close and wedge `httptest.Server.Close`; the check was racy
  by design and removed, the stub now always released via
  `t.Cleanup`, `-race` clean, and the CI go-test timeout raised to
  20 m for the DB-gated suite).

## Operator notes

- Deploy: both services changed — `railway up --service server`, then
  restart `app` (nginx resolves the server's internal address at
  startup). Migration `20261004…` applies automatically on server boot.
- The migration is policy-only (ideas SELECT) — old server + new policy
  already behaves correctly, so deploy order is safe.
- Rotate the Resend mail key if local session transcripts can leave the
  machine (a shell probe printed it into a local transcript on
  2026-10-04; nothing was committed or transmitted).
