# 2026-10-04 — Remediation Waves 3–4 (board promises, idea privacy, verification spine)

**Status:** Finished 2026-10-04
**Source:** [`docs/review/2026-10-04-multirole/40-remediation-plan.md`](../review/2026-10-04-multirole/40-remediation-plan.md)
Waves 3 and 4 (the release's last two). Register IDs refer to
[`01-findings-register.md`](../review/2026-10-04-multirole/01-findings-register.md).
Waves 0–1 landed in `ecfe4d2`, Wave 2 in `417e45a`. Changelog:
[`docs/changelogs/2026-10-04-remediation-wave-3-4.md`](../changelogs/2026-10-04-remediation-wave-3-4.md).

**Plan review:** none run separately — the multirole review (10 reviewers +
skeptic pass over the register) already reviewed this exact scope; this plan
only executes its rows.

## Decisions (recorded for 3.4 / USER-05, the one product call)

Ideas are **private to their author by default**, per PRODUCT.md ("Capture
remains one-step and private to its author by default") and the README's
"每个成员都有一个零门槛的想法收集箱 / every member gets a … idea inbox".
Implemented at the enforcement layer, not just the query: the `ideas_select`
RLS policy becomes author-or-admin (admins keep their existing drop
authority), `ListIdeas` filters to the author, `GetIdeaForMember` returns
404 for non-authors, and the ideas wall states the rule. An explicit share
action stays out of scope until the product asks for it.

The quiet-project threshold (3.1 / FEAT-01) is a constant: an `active` or
`planning` project with no update in 14 days (falls back to `created_at`
when a project has no updates) is `flagged`. `done`/`paused`/`archived`
projects are never flagged — quiet is expected there.

## Checklist (smallest end-to-end first)

### Wave 3 — truth in advertising

- [x] 3.1 FEAT-01 — `projectView.flagged` computed from
      `LatestUpdateAt ?? CreatedAt` vs the 14-day threshold; unit test for
      the flag rule.
- [x] 3.4 USER-05/FEAT-04 — idea privacy per the decision above: failing
      store test first, then the RLS migration
      (`20261004…_ideas_private_by_default.sql`), `ListIdeas` author
      filter, `GetIdeaForMember` author-or-admin, ideas-wall hint string.
- [x] 3.2 FEAT-02 — board cards render the quiet badge, latest-update
      kind/content line, and a relative timestamp (`latestUpdateAt` joins
      the client model); pull-to-refresh on the board. Widget test.
- [x] 3.3 FEAT-03 — export page: markdown downloads a real `.md` file on
      web; print opens the print HTML in a new tab (download fallback).
- [x] 3.5 FEAT-05 — roadmap default view renders `GET /roadmap/progress`
      (server truth: deadlined Now, ≤3 justified Next, needsDeadline,
      doneCount; pressure pills). Bands and canvas stay as secondary
      modes. Widget test.

### Wave 4 — verification spine

- [x] 4.4 TEST-05 — `agentcontext_test.go` log-only branch becomes
      `t.Errorf`.
- [x] 4.3 TEST-04 — `server/internal/api/conversations_test.go`: DB-gated
      httptest POST against a stubbed coach asserting incrementally
      flushed SSE frames (frame 1 observable before frame 2 is written).
- [x] 4.5 CQ-02 — `feature_wiring_test.dart` demoted to a named lint-tier
      backstop; the behavioral coverage lives in the widget tests added
      by 3.2/3.5 (and existing `versions_page_test`).
- [x] 4.6 CQ-01 — dated modularity issues for `coach.go`, `anthropic.go`,
      `models.dart` (`client.dart` already has
      [`2026-09-18-flutter-apiclient-line-budget.md`](../issues/2026-09-18-flutter-apiclient-line-budget.md)).
- [x] 4.1 + 4.2 PM-05/TEST-02 — `.github/workflows/ci.yml`: Go
      build/vet/test with a `postgres:18` service (roles + migrations
      applied; `DATABASE_URL` exported so the store suite runs, 0 SKIP),
      Flutter analyze + test, and the env suites — on pull_request and
      push to `main`.

## Verification

- [x] `go build ./... && go vet ./... && go test ./...` green with
      `DATABASE_URL` set (0 SKIP in `internal/store`)
- [x] `cd client && flutter analyze && flutter test` green
- [x] `cd scripts && python3 -m unittest test_env_catalog test_env_writes` green
- [x] Local live stack: board walk shows badge + latest update + relative
      time; export downloads a file; ideas of a second member are
      invisible to the first; roadmap defaults to the progress view
- [x] CI green on the commit that lands the workflow
- [x] changelog + register statuses + board + HANDOFF updated; deployed
      (`railway up --service server`, `--service app`, restart `app`) and
      domain-verified

## Operator actions carried in (not agent-executable)

- Rotate the Resend mail key if this machine's session transcripts can
  leave it (a shell probe printed the key into a local transcript on
  2026-10-04; nothing was committed or transmitted).
- The release gate's signed-in hosted page load (PM-02) still needs a
  member login on the deployed domain.

*Superseded 2026-10-11: both operator items closed — the Resend key was
rotated 2026-10-05 and re-keyed 2026-10-11 with the full mailed funnel
live ([mail-truth changelog](../changelogs/2026-10-11-owner-deploy-and-mail-truth.md)),
and the hosted signed-in exercise ran 2026-10-11 (PM-02 Fixed in the
[disposition ledger](../review/2026-10-10-disposition-ledger/04-multirole-3.md)).*
