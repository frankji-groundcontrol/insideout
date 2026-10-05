# 2026-10-05 — Planning-flow remediation

**Status:** Finished 2026-10-05 — waves 1–2 plus the gap-closing phase
(all 40 slices delivered or recovered). **Source:** the multirole review of the planning
flow (run `dwfrun-473205b2`, scores and register in its artifacts) —
its 40-slice fix/harden/improve/add list is this plan's checklist.

## Checklist

- [x] Fix wave (11): audience crash + render, coach stage shown + confirm
      gate, one Now definition, build requires commit + approved PRD,
      planner errors honestly, template branches from PRD sections,
      commit screen shows diff/gaps/note.
- [x] Harden wave (8): draft-then-confirm (build + split), legal status
      transitions, no invented percentage, fresh tree visible on
      Progress, readiness must_clarify_now + gaps persisted, e2e
      happy-path test, wiring test taps real handlers.
- [x] Improve wave (14): progress cards (status/open/action/evidence),
      Done lists leaves, parents on board, Progress default + hierarchy,
      desk next-step, blocking gaps first, visible audience button,
      critic findings listed, dock unfinished step, workspace board
      stage, Evidence dock leaf evidence.
- [x] Add wave partial (3): progress SSE (live board), proposal applies
      items, build lands on roadmap.
- [x] Suites green after two coordinator repairs (nil-store hook guard;
      RoadmapBands progress wiring) — go all pass, flutter 89/89.
- [x] Records: changelog + index + this plan + board row.
- [x] **Gap-closing phase** — audit found 6 more workflow-reported-done
      slices that never landed (8, 9, 12, 23, 25, 26); all recovered.
      Migration `20261005120000_roadmap_acceptance_owner.sql` adds
      `acceptance` + `owner_user_id`; legal status transitions enforced;
      fresh tree seeds one in_progress leaf with a deadline; Done lists
      its leaves; childless cards show no bar, parents show their done
      share; the workspace board and the Evidence screen render counts
      and leaf evidence. Failing tests written first; every DB test
      actually executed against the disposable compose stack this time.
- [x] Gap-phase verification: disposable-stack suites green (Go all,
      Flutter 94/94); deploy + domain verify below.
- [x] Close: this update + changelog + board + track.

## Decisions

- Migration slices were deliberately held out of the workflow run: the
  local `DATABASE_URL` is the shared production Supabase, and unattended
  subagents were not going to touch it. The gap phase applies the
  migration deliberately, tested on the disposable compose stack first.
- Slice counts: 36 done / 4 specified (the workflow's own summary said
  37 + 4; one slice was double-counted).
- Coach stage confirm is client-side on the existing event stream; no
  new wire protocol.
- Build gating: commit **and** approved status required
  (`TestBuildFromPrdRequiresCommit`, `TestBuildFromPrdRequiresApproved`).

## Verification log

- 2026-10-05: `go test` (agent/api/roadmaptime/store) all pass;
  `flutter test` 89/89 — after the two post-run repairs recorded in the
  changelog.

- 2026-10-05 (gap phase): the f8f16eb suites were green only because
  DB-gated tests skipped without DATABASE_URL; first real run on the
  disposable stack caught the ListRoadmap scan arity, the planner/e2e
  disagreement (template now titles branches "key — first clause of
  body"), and the never-executed e2e path. Lesson recorded in learning.
