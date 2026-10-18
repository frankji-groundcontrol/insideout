# Wave 1, clock and runes — review-4 remediation

Branch `rv4/w1-clock-and-runes`, `1dba3fe..7994d28` (7 commits: four red tests
and three fixes), then this fragment. Findings are cited as `rv4-<ID>` because
review IDs restart per review. Rows 1.3 and 1.17 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md);
the rules are in the
[execution contract](../../review/2026-10-18-multirole-4/41-remediation-contract.md).
Every change is in `server/internal/api`: no migration, no client change and
no change to any response shape.

## What changed

- **The project views judge "quiet" against the server's clock** (rv4-L1, red
  `868ebe7`, fix `a86147c`). `TestProjectQuietFlagged` pins its own date
  (2026-10-04) but asserted through `projectResponse`, which read the wall
  clock, so it went red once the real date passed 2026-10-16 with no commit
  changing. `Server` now owns a clock (below) and `projectResponse(p, now)`
  takes the instant. The five callers (board list, create, detail, update, and
  the repo link in `github.go`) pass `s.now()`; the list takes it once, so every
  card in one response is judged against one instant. The red commit adds three
  boundary rows (exactly 14 days is not quiet, 14 days and a second is). On the
  base the existing test already failed in two rows; with the new rows four fail.
- **The roadmap views do too** (rv4-QUAL-05, red `99c25fa`, fix `34d7415`).
  `roadmapNodeResponse(n, now)` feeds `roadmaptime.Pressure`, and
  `roadmapProgress` feeds `roadmaptime.Assemble` from `s.now()`. Callers: the
  list (once per response), create, update, move, the validate response, the
  expand-confirm loop and the Progress view. The red tests pin the clock 30
  days, 7 days and 48 hours before a fixed deadline and one second after it,
  and read the pressure back from the PATCH, list, move, validate and Progress
  responses. On the base every response said `overdue`, because that deadline
  is in the past on the real clock.
- **A guard keeps the view builders off the wall clock** (rv4-ARCH-04, red
  `b3545a9`, fix `34d7415`; the clock itself arrives in `a86147c`).
  `view_clock_guard_test.go` parses every non-test file of the package and
  fails on `time.Now`, `time.Since` or `time.Until`, called or handed over as a
  value, under whatever name the file gives the `time` import. The red commit
  reports three hits (`projects.go`, `roadmap.go`, `roadmap_progress.go`);
  `a86147c` clears the first and `34d7415` the other two, so the guard is green
  from `34d7415`.
- **Field limits count characters** (rv4-COMP-01, red `d257713`, fix `7994d28`).
  The messages said "max 200 characters" and the checks were `len(s) > 200`,
  which counts bytes: 67 Chinese characters are 201 bytes and were refused.
  `util.go` gains `tooLong(s, n)`, which counts runes, and all 18 character caps
  call it: the titles of ideas, projects, workspaces, PRDs and roadmap nodes
  (create and update, 200), the PRD commit name (200), the validation
  observation detail, source URL and confirm reason (2000), the agent
  checkpoint and proposal summaries (2000), the proposal item title (200) and
  the two usernames (60). The red test drives each field over its real route
  with limit/3+1 characters and exactly limit characters of three-byte text
  (both must be accepted) and with limit+1 (must be refused with that field's
  own message): 18 fields, 54 subtests, and the 36 accept subtests fail on the
  base.

## The clock

- `Server.clock` is a `func() time.Time` and `Server.now()` the accessor (a
  field and a method cannot share a name, so the plan's `Server.now` is the
  method). A nil clock reads the wall clock, so `NewServer` keeps its
  signature, every `&Server{...}` literal in the tests still works, and
  production sets nothing. Only tests set it, through `routeWorld.pinClock` in
  `view_clock_test.go`.
- Builders take the instant as a parameter rather than a server, so a unit test
  passes its own date (`projects_test.go` does) and a route test pins the
  server.
- The board route test derives its instants from the project's own `createdAt`
  (14 days after is not quiet, one second more is), so it holds on any real
  date and needs no hard-coded one. The roadmap tests move the clock around a
  fixed deadline.
- The rule itself is untouched: `quietAfter`, `projectQuietFlagged` and
  `roadmaptime.Pressure` are as they were, and no tolerance was widened.

## The guard and its ceilings

- The allowlist is keyed by file name with the reason as the value:
  `admin.go`, `auth.go` and `auth_email_flows.go` (expiry stamps written to the
  database), `auth_cooldown.go` and `auth_throttle.go` (the default clock of
  the cooldown and the throttle; their tests inject their own),
  `middleware.go` (times a request for the access log) and `server.go` (the
  production default that `Server.now` falls back to).
- It is per file, not per function: a view builder added to one of those seven
  files could read the clock unseen. Keep view builders out of them. A
  per-function rule needs type information and is not worth it for this list.
- A stale entry (a listed file that no longer reads the clock) is not an error.
  It costs one missed catch in a file nobody is editing; delete the entry when
  you next touch the file.
- Two layers on purpose. Putting `time.Now()` back into a handler fails the
  guard and the route test; a handler that hands over the zero time instead of
  the clock fails only the route test.

## The character count

- `tooLong` is `utf8.RuneCountInString(s) > n`, the count `project_updates.go`
  already used for its 5000-character note cap. It counts code points, not
  grapheme clusters, so an emoji with a skin-tone modifier is two characters.
  That matches the existing cap and needs no dependency.
- Trimming, the messages and the limits are unchanged; only the unit of the
  comparison changed. No column in the schema has a length constraint, so there
  is no byte limit underneath that a longer value could now reach.
- Register is the one route that does not answer 2xx to an accepted value:
  signup is invitation-only by default, so it answers 403 after the length check
  has passed. For that field the test requires only that the length message is
  absent.
- `TestTooLongCountsRunes` (the helper's own table) ships with the fix, not
  with the red commit: it calls `tooLong`, so at the red commit it would be a
  compile error, and a compile error is not a red test. The route test compiles
  on the base and fails on behavior.

## Observations, not fixed

Found while reading; each is outside the two rows or outside the owned files.
They are candidates for the integrator, not part of the dispositions.

- The create, detail, update and repo-link responses build the view from
  `store.ProjectWithLatest{Project: *p}`, which carries no latest update, so
  `flagged` is judged on `createdAt` alone while the board list judges on the
  latest update. An active project created more than 14 days ago with a fresh
  update reads `flagged: true` on its detail response and `false` on the board.
  The new route test uses a project without updates, so it does not pin this.
- `handleAgentPropose` writes the proposal update before it validates the
  items, so a request refused for an item title, an item action or an 11th item
  leaves an `agent_proposal` update on the timeline.
- The password minimum (`auth.go` and `auth_email_flows.go`) is
  `len(req.Password) < 8` under the message "at least 8 characters". It counts
  bytes, so three Chinese characters satisfy it. It is the opposite direction
  from this finding (a weaker rule, not a refused value) and a policy call, so
  it was left to the auth cluster.
- `conversations.go` caps a coach message at 10,000 bytes (about 3,333 Chinese
  characters); the cluster notes leave it alone. `firstLine` in
  `github_webhook_evidence.go` cuts at 120 bytes and can split a multi-byte
  character.

## Left for the integrator

- Add this fragment to the wave-1 section of this folder's `index.md` with the
  dispositions below. SHAs are worker-branch SHAs; each integration commit's
  `cherry-pick -x` line names the original.
- `docs/issues/README.md` and the open-findings record: ledger rows for L1,
  ARCH-04, QUAL-05 and COMP-01 (all FIXED), and the four observations above if
  the integrator wants them as findings.
- `docs/architecture/backend.md` has no sentence on view builders or the clock.
  If it should: "Response builders take the instant as a parameter and handlers
  pass `s.now()`; `view_clock_guard_test.go` enforces it. Request limits count
  characters through `tooLong`."
- `docs/learning/2026-10-18-a-wall-clock-test-needs-a-scheduled-run.md` names
  `Server.now func() time.Time`; the shipped shape is the `clock` field plus the
  `now()` method. A "fixed in" line pointing here would close it.
- Ordering: `w1-github-repo` runs after this cluster and edits `github.go`;
  this cluster changed one line there (the `projectResponse` call). Until
  integration, other api clusters run with `-skip TestProjectQuietFlagged`; at
  integration the test is green. The guard allows the three files
  `w1-auth-atomic` edits (`admin.go`, `auth_cooldown.go`,
  `auth_email_flows.go`) but not `github.go`: a raw `time.Now()` there fails the
  guard.

| Finding | Row | Status | Red | Fix |
| --- | --- | --- | --- | --- |
| rv4-L1 | 1.3 | FIXED | `868ebe7` | `a86147c` |
| rv4-ARCH-04 | 1.3 | FIXED | `b3545a9` | `34d7415` (clock in `a86147c`) |
| rv4-QUAL-05 | 1.3 | FIXED | `99c25fa` | `34d7415` |
| rv4-COMP-01 | 1.17 | FIXED | `d257713` | `7994d28` |

## Verification

Each red commit was replayed from a plain export of its tree against a private
PostgreSQL 18 (the route tests run the real handlers under row-level security,
and a skip counts as red); each fails, and its fix turns it green:

| Commit | Test | Result |
| --- | --- | --- |
| `1dba3fe` (base) | `TestProjectQuietFlagged` | fails, 2 rows |
| `868ebe7` | `TestProjectQuietFlagged` | fails, 4 rows |
| `b3545a9` | `TestViewBuildersDoNotReadTheWallClock` | fails, 3 hits |
| `a86147c` | quiet tests, board route test | pass |
| `99c25fa` | roadmap node and Progress pressure tests | fail, 7 subtests |
| `34d7415` | roadmap tests, guard, quiet test | pass |
| `d257713` | `TestFieldLimitsCountCharactersNotBytes` | fails, 36 subtests |
| `7994d28` | rune tests | pass |

The outputs are kept as `w1-clock-and-runes-red-<ID>.txt` and
`w1-clock-and-runes-green-<ID>.txt` under `/private/tmp/insideout-rv4/reports/`.
Mutation checks a verifier can run, each restored from git: make `tooLong`
count bytes (both rune tests fail); read `time.Now()` in the board list
handler (guard and board route test fail); pass the zero time there (route test
only); put `time.Now()` back in `roadmap.go` (guard and the node pressure test
fail) or in `roadmap_progress.go` (guard and the Progress test fail).

Gates: `gofmt -l` is empty; `go build ./...` and `go vet ./internal/api/...`;
`go test ./internal/api/... -count=1` (127 tests, no failure, no skip);
`-race` over `Quiet|Clock|Runes|Limit`; `go test ./... -race -count=1` over the
whole server (26 packages ok); `prds.go` is 343 lines; every touched file is
at most 350 lines; the six `scripts/tests/check_*.py` guards,
`sh scripts/check-redaction.sh` and `sh scripts/check-docs-recorded.sh`.
