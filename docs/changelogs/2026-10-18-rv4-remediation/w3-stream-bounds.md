# Wave 3, stream bounds — review-4 remediation

Branch `rv4/w3-stream-bounds`, `8c74218..b3dd27e` (fourteen commits: seven red/fix
pairs), then this fragment. Findings are cited as `rv4-<ID>` because review IDs
restart per review. Row 3.2 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md).
There is no migration, no new route and no new environment variable. No address,
host, secret or one-time code is spelled in this fragment.

## Design spike

Row 3.2 gave a direction, not a design, so each half was settled by reading the
code it touches and taking the smallest thing that holds. Stream caps are a pair
of counters under one mutex, taken after the membership check and before the
first header, so a refusal is a plain JSON 429 and a refused attempt never
registers presence. There is no global cap (the total is bounded by users times
the per-user cap) and no limiter on the `?client=` key space, because every
registered key belongs to a live stream and so the per-user cap already bounds
it. The cursor limit is a fixed one-second window per user, not a token bucket:
the canvas never bursts and only a script reaches it. The request deadline is
one middleware around the mux that looks up the matched pattern and skips seven
exempt ones (the two passive streams and the five routes that wait on the
model), rather than a wrapper on each handler. The membership and token recheck
rides the ten-second heartbeat that already wakes each stream, and fails open
when the database errors, because an outage must not log everyone out and
`/healthz` is the signal for that. The singleton guard retries on its own
hijacked connection every second for two minutes, which covers the overlap of a
rolling deploy, and reports the unchanged ERROR only when the window ends. The
web presence and progress streams reuse the coach's existing refresh-and-reopen
helper through the same raw fetch. The section cap is 10,000 characters, below
the review's 20,000 example: eight sections at the cap are 80,000 characters of
context per turn, and the same number at write and in the prompts means the
coach reads exactly what was saved.

## What changed

- **Stream and cursor caps** (rv4-REL-01, red `15e776b`, fix `566345e`).
  `stream_caps.go`: at most 20 open passive streams (presence and roadmap
  progress) per user and 100 per project, refused with `429` code
  `APP_STREAM_LIMIT`; closing a tab gives the slot back. Cursor posts are limited
  to 25 a second per user (five times the canvas's own 200 ms throttle), refused
  with `429` code `APP_THROTTLE` and `Retry-After: 1` before any database read.
  Red: "one member opened 150 streams and none was refused", "9 members opened 135
  streams on one project and none was refused", "300 cursor posts in one instant
  were all accepted", "one member registered 75 presence sessions with distinct
  client ids, want at most 50". Tests: `stream_caps_test.go` (real PostgreSQL).
- **Deadlines** (rv4-REL-02, red `696e35d`, fix `d7edb99`). `/healthz` pings the
  database under a 2 s deadline and answers 503 `database unreachable` when it
  cannot; every other route runs under a 25 s request deadline
  (`request_deadline.go`), except the seven exempt patterns. A route-table test
  fails when an exempt pattern names no registered route. Red: healthz "gave no
  answer within 6s while the database was stalled" (the test took 21 s); "a plain
  route (confirm stage) ran with no request deadline". Green: healthz in 2.01 s.
  The stall is a TCP proxy in front of the test database (`db_stall_proxy_test.go`)
  that stops forwarding bytes, because `statement_timeout` cannot fire on a
  frozen server.
- **Membership recheck** (rv4-REL-03, red `36a2381`, fix `8be6554`). On every
  heartbeat each stream asks `GetProjectForMember` again; a member removed from
  the workspace loses both streams and their presence entry at the next tick.
  Red: "the removed member's presence stream is still open 3s after the removal".
- **Singleton retry** (rv4-REL-04, red `f077c2a`, fix `ef083f1`). Only the retry
  half was in scope. A second process now logs a WARN with the retry settings,
  retries every second for two minutes on a per-attempt hijacked connection, logs
  an INFO when it takes over, and logs the unchanged ERROR text only when the
  window expires. Red: "the second process logged an ERROR in its first moments"
  and "8s after the incumbent exited nobody holds the singleton lock". The old
  test `TestSingletonGuardErrorsWhenASecondServerRuns` was adapted to the
  shortened window; its claim (an ERROR for a lasting second replica) is kept.
- **Stream lifetime** (rv4-SEC-08, red `d09558a`, fix `b57435c`). The same
  heartbeat check verifies the stream's own access token, so a stream ends when
  the session that opened it expires; the per-user cap half of the finding is
  rv4-REL-01. Red: "the presence stream is still open 6s after its 3s access
  token expired". The server closes cleanly, and the client reconnects (below).
- **Web stream refresh** (rv4-FULL-04, red `f0d2763`, fix `37b9612`).
  `client_stream.dart`: the web presence and progress streams opened a raw fetch
  that threw on any status of 400 or more, so a 401 ended them for good. They now
  go through `coachSseWithRefresh` (one single-flight refresh, one reopen, tokens
  cleared on a second 401), with a test seam `fetchAttemptForTests`. The red
  commit carries the behaviour-neutral seam so the failure is behavioural: six of
  ten tests fail ("stream failed", refresh count 0 not 1, tokens not cleared).
  Tests: `sse_refresh_streams_test.dart`.
- **PRD section cap** (rv4-COMP-04, red `9574d06`, fix `b3dd27e`).
  `PATCH /prds/{id}` refuses a save with a section over 10,000 characters (runes,
  not bytes) with `400` `section <key> is too long (max 10000 characters)`; the
  whole save is refused and nothing is stored. `formatSectionsForPrompt`, which
  the coach, critic and roadmap planner all use, cuts a section at the same
  number and appends `...[truncated: N more characters]`. Red: "a 10001-character
  section: status 200, want 400"; "a 200000-character section rendered 200131
  characters, want at most 10500"; coach, critic and planner in both locales grew
  190,000 characters. Mutating either cap by one fails the tests on that side.

## Constants

| Constant | Value | Where |
| --- | --- | --- |
| `maxStreamsPerUser` | 20 | `stream_caps.go` |
| `maxStreamsPerProject` | 100 | `stream_caps.go` |
| `maxCursorPostsPerSecond` | 25 | `stream_caps.go` |
| `healthzPingTimeout` | 2 s | `healthz.go` |
| `requestDeadline` | 25 s | `request_deadline.go` |
| `streamRecheckTimeout` | 5 s | `stream_recheck.go` |
| `streamHeartbeat` (unchanged) | 10 s | `server.go` |
| `singletonRetryEvery` / `singletonRetryWindow` | 1 s / 2 min | `single_replica.go` |
| `maxPrdSectionRunes`, `maxPromptSectionRunes` | 10,000 each | `prd_section_cap.go`, `agent/prompts.go` |

## Live check

Built binaries on a private throwaway PostgreSQL cluster, bound to loopback only.

| Check | Observed |
| --- | --- |
| 30 progress streams from one user at once | 20 accepted, 10 refused with `APP_STREAM_LIMIT` |
| 6 users, 20 presence streams each, distinct `?client=` ids, one project | 100 accepted, 20 refused |
| 80 cursor posts from one user in 0.04 s | 25 accepted, 55 refused with `Retry-After: 1`; a second user unaffected |
| A removed member's two open streams | both closed 8.6 s after the removal; reopening answers 404 |
| A stream opened with a 12 s access token | closed 20.1 s after opening (the first heartbeat after expiry) |
| The private cluster frozen with SIGSTOP (its own processes only) | `/healthz` 503 in 2.00 s; a plain route 500 `internal error` in 25.00 s; both answered 200 at once after SIGCONT |
| A second server started beside the first, then the first stopped | WARN with the retry settings and no ERROR; INFO "took over the singleton lock" 0.16 s after the first logged "shutting down" |
| PRD save | 10,000 characters (30,032 bytes) 200; 10,001 characters 400 naming the section; 200,000 characters plus a valid section in one save 400, nothing stored |

## Limits and what is not covered

- Each open stream now costs one membership read per heartbeat, at most users
  times 20 reads every ten seconds. A removed member or expired token is caught
  at the next tick, so up to ten seconds of stale access remains by design; the
  streams carry only presence names and roadmap progress. A database error or the
  5 s recheck timeout fails open.
- There is no global stream cap, only the per-user and per-project ones. The
  cursor limiter keeps one small entry per user who ever posted a cursor
  (`ponytail:` note in `stream_caps.go`).
- Streams now end when the access token expires (15 minutes by default). The
  non-web client already refreshed through dio; the web client depends on the
  rv4-FULL-04 fix in this same branch, so the two must land together.
- A plain route that hits the 25 s deadline answers 500 `internal error`, the
  handlers' generic mapping, not 503.
- SSE 401 refresh is covered by a Dart test with the repository's scripted fake
  transport on the VM, not by a browser; the Chrome-run suite under
  `client/test/web` was neither extended nor run.
- Not exercised against real services: a real coach turn with a clipped section
  (the cut is tested on the three prompt builders with stub streamers), more than
  one replica, the hosted deploy, the project cap from many client addresses (six
  local users), and the Flutter web build.
- rv4-COMP-04 is closed for the user-driven path. Outside this cluster's files,
  `agent/tools.go` writes `update_prd_section` text without the cap and `get_prd`
  returns sections whole; a row saved above the cap before this change stays
  stored whole and the coach sees its first 10,000 characters and the marker.
  Fix prompt: in `agent/tools.go` refuse an `update_prd_section` over 10,000
  characters with a message telling the model to shorten it, and clip `get_prd`
  sections with the same marker.
- The PRD desk's section editors have no length limit, and a save sends every
  section, so one over-long section refuses the whole save through the guard's
  SnackBar with the message above. Fix prompt: in
  `client/lib/features/prd/prd_desk.dart` give the section editors a
  `maxLength` of 10,000 with a counter.

## Integrator's to-do

- `docs/architecture/backend.md` and `docs/usage/deployment.md`: the cap
  constants, the healthz deadline and the request deadline (table above).
- `docs/usage/deployment.md`, around lines 150 to 160: the singleton passage says
  a second process logs the ERROR "on its first moments of life". Now it logs a
  WARN, retries for two minutes, takes over with an INFO, and logs the ERROR only
  if the window expires; the alert advice stands but should say "within two
  minutes".
- `docs/usage/environment.md`: none, there is no new variable.
- `docs/changelogs/2026-10-18-rv4-remediation/index.md`, `docs/changelogs/README.md`
  and `docs/changelogs/index/2026-10.md`: list this fragment and the wave 3 row.
- `docs/plans/README.md` and `docs/issues/2026-10-18-review-4-open-findings.md`:
  row 3.2 state, and the two fix prompts above as open items.

## Verification

Run on the head of the branch against a private PostgreSQL cluster.

- `go build ./...` and `go vet` on the api, presence, progresshub and agent
  packages: clean. `gofmt -l`: clean. No Go or Dart file the branch touches is
  over 350 lines (the largest is `prds.go` at 347).
- `go test -count=1` on api, presence and agent: pass (progresshub has no tests);
  `-race` on api, presence and progresshub: pass.
- `flutter pub get --enforce-lockfile`, `flutter analyze --no-fatal-infos` (14
  infos, none new: the one in `client_stream.dart` sits on a parameter the base
  already had), `flutter test` 493 tests and `TZ=Asia/Shanghai flutter test` 493
  tests: pass.
- The whole server suite ran once: 26 packages passed and `internal/store`
  failed one test, `TestSignupModeSetting`, because the live check had left
  `signup_mode` set to open on the private test database. With the seeded value
  restored the store package passes; nothing under `internal/store` changed on
  this branch.
- `scripts/tests/check_*.py` print the same success lines as the integration
  branch and exit 0; `scripts/check-redaction.sh` and
  `scripts/check-docs-recorded.sh --worktree` exit 0.
- Each red commit fails for the behaviour in its message and each fix commit turns
  it green. Each fix was also mutated once and restored, and its new tests went
  red: REL-01 (the three caps lifted), REL-02 (the healthz timeout dropped, then
  the request-deadline wiring removed), REL-03 (the recheck taken out of both
  handlers), REL-04 (the retry window set to zero), SEC-08 (the token check
  skipped), FULL-04 (the refresh stubbed to fail) and COMP-04 (each cap moved by
  one, on the api side and on the prompt side).
