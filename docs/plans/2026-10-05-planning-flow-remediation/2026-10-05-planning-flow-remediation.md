# 2026-10-05 — Planning-flow remediation

**Status:** In flight 2026-10-05 — waves 1–2 landed (36 of 40 slices);
gap-closing phase open. **Source:** the multirole review of the planning
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
- [ ] **Gap-closing phase (4 specified slices)** — one migration adding
      `roadmap_nodes.acceptance` (text, default '') and
      `roadmap_nodes.owner_user_id` (uuid → users), then: API write/read
      through the existing node-update path; agent context carries
      acceptance + deadline + pressure; progress payload + card render
      both; failing tests first against the disposable local stack.
- [ ] Gap-phase verification: disposable-stack store tests green, both
      suites green, deploy both services, verify via the domain.
- [ ] Close: changelog for the gap phase, board row → Finished, track
      completed.

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
