# 2026-10-05 — Planning-flow remediation (waves 1–2 of the review of it)

**Scope:** the 40-slice fix/harden/improve/add that followed the multirole
review of the planning flow (dynamic-workflow run `dwfrun-473205b2`). 36
slices landed; the 4 that need new database columns are specified in the
plan and close next. Plan:
[`docs/plans/2026-10-05-planning-flow-remediation/`](../plans/2026-10-05-planning-flow-remediation/2026-10-05-planning-flow-remediation.md).

## What changed

**Fix (11).** The audience page no longer throws on a missing projection
and renders end to end; the coach panel shows the current stage (labels
existed unused in both l10n tables) and a stage change waits for the
person to confirm; Bands and Progress share one definition of Now (the
server's, via `RoadmapBands(progress:)`); building a roadmap requires a
committed **and** approved PRD (the home copy's signature claim is now
true); a failed or unconfigured planner returns an error instead of a
generic tree, and the offline template branches from the PRD's own
sections; the commit screen renders the section diff, carried gaps, and
decision note instead of three counts.

**Harden (8).** Build and expand are draft-then-confirm (`roadmap_draft`
+ confirm, nothing written until confirmed); node status moves only
through legal transitions (pending → in_progress → done, done reopens);
the node card stops inventing 55%; a freshly built tree is visible on
Progress (one in-progress leaf with a deadline); readiness returns
`must_clarify_now` for empty sections and the commit persists its gaps;
one Postgres-backed test walks PRD → approval → build → deadline → done
→ progress (`TestPrdThroughApprovalToProgress`); the wiring test taps
the real build/review actions instead of grepping source.

**Improve (14).** Progress cards show status, open the node, carry the
legal status action, and render their evidence; Done lists the finished
leaves instead of a count; parents appear on the board; Progress is the
default roadmap view and carries the hierarchy; the PRD desk names one
next step; blocking audience gaps sort first; the audience action is a
visible button; the coach panel renders the critic's findings as a list;
the dock marks the unfinished step; the workspace board shows each
project's stage and counts; the Evidence dock renders leaf evidence.

**Add (3 of 7).** Progress updates while work runs
([own changelog](2026-10-05-roadmap-progress-stream.md)); accepting a
proposal applies its roadmap items; a build lands on the roadmap, not
the evidence page.

**Specified, not applied (4).** Acceptance sentence (16), owner (17),
implementation context carrying acceptance/deadline/pressure (36), and
the progress card naming its owner (39) — all need new columns; the
local database is the shared production instance, so they wait for the
gap-closing phase with their own migration.

## Verification

- `cd server && go test ./internal/agent/ ./internal/api/ ./internal/roadmaptime/ ./internal/store/` — all pass.
- `cd client && flutter test` — 89 pass.
- Honest repairs after the workflow's own final run reported both suites
  red: `NewServer` registered the roadmap-change hook on a nil store
  (panic in the auth-cooldown test; now nil-guarded), and the new bands
  test passed a `progress` argument the widget never accepted — the
  slice's actual change (server definition of Now) was missing and was
  completed by hand. Both suites green after the two repairs.
- The wave reviewers accepted whole waves at a time, which is how the
  bands gap slipped; per-slice acceptance was not independent.

## Operator

- No migration in this commit. Deploy both services (`server`, then
  restart `app` so nginx re-resolves).
- The four schema slices land next with one migration; nothing to do
  until then.

## Gap-closing phase (same day)

The four schema slices plus six the workflow had reported done but never
landed (8 legal transitions, 9 fresh tree visible, 12 done list, 23 honest
progress bar, 25 workspace counts, 26 leaf evidence) — found by auditing
every reported test name against the tree.

- **Migration** `20261005120000_roadmap_acceptance_owner.sql`:
  `roadmap_nodes.acceptance text NOT NULL DEFAULT ''` and
  `roadmap_nodes.owner_user_id uuid → users ON DELETE SET NULL`. Applied to
  the disposable compose stack first; the deployment applies it on boot.
- **Store:** acceptance/owner on the node, read and write through
  `UpdateRoadmapNode` (owner must be a workspace member); status moves only
  through the legal machine (`ErrInvalidTransition`); `ReplaceRoadmapTree`
  seeds the first leaf in_progress with a 7-day deadline.
- **Payload + context:** progress items and Done list entries carry
  acceptance and owner; the implementation agent context carries
  acceptance, deadline, pressure, and owner.
- **Client:** the edit dialog writes the acceptance sentence and picks the
  owner from workspace members; cards and progress items render both; Done
  lists its leaves; childless cards show no progress bar and parents show
  their done share; the workspace board shows each project's Now/Done
  counts; the Evidence screen lists done leaves with their evidence rows.

**Verification.** Failing tests first; for the first time the DB-gated
suites actually executed — on the disposable compose stack
(`docker compose --env-file` with throwaway credentials, never the shared
production DSN). `go test ./...` green with both role URLs;
`flutter test` 94/94. The first real run caught a scan-arity break in
ListRoadmap and a planner/e2e disagreement (template branches now titled
"key — first clause of the body") — both fixed here. Lesson: a green suite
without `DATABASE_URL` set says nothing about the DB path; the earlier
"f8f16eb suites green" was skip-green.

**Operator.** Deploy `server` (the migration applies on boot), then restart
`app`. Nothing else to configure.
