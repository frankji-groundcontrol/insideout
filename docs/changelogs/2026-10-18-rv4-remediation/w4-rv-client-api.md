# Wave 4, the client API layer: repo-link copy, stream refresh, one decision per proposal — review-4 remediation

Branch `rv4/w4-rv-client-api`, `582a4bd..784291a` (eight commits: four red/fix
pairs), then this fragment. Cluster `w4-rv-client-api` of the whole-change
review of `git diff d53035a..rv4/integration`. Findings are cited as
`rv4-<ID>` because review IDs restart per review; `FR` is that review's prefix.
Eight rows, all `FIXED`, in three independent roots. No database was used
(`needs_db` is false), and no browser was opened (see the ceilings). No
address, host, secret or one-time code is spelled in this fragment.

| Finding | Root | Red | Fix |
| --- | --- | --- | --- |
| `rv4-FR-F03`, `rv4-FR-F09`, `rv4-FR-F16` | A. the repo-link codes have no copy | `bfdfb54` | `e9f5956` |
| `rv4-FR-F01` | B. a refresh that merely failed ends the live streams | `16252f5` | `b3050aee` |
| `rv4-FR-F112` (client half) | B. a refresh another tab won clears the winner's pair | `a8edde07` | `99020680` |
| `rv4-FR-F02`, `rv4-FR-F08`, `rv4-FR-F15` (client half) | C. Accept and Reject can go out twice | `959101d7` | `784291a2` |

Findings that share a root share its red test. Each finding is still its own
row in the report, and the test names say which findings they cover.

## What changed

- **A refused repo link says what to do** (F03, F09, F16). `localizedError`
  (`client/lib/api/errors_l10n.dart`) mapped an `ApiException` by HTTP status
  alone, so the server's three repo-link codes (`server/internal/api/github.go`)
  read as status copy: `GITHUB_REPO_NOT_VISIBLE` and `GITHUB_REPO_URL_INVALID`
  (both 400) as "bad request", `GITHUB_REPO_ALREADY_LINKED` (409) as "Changed
  elsewhere — refresh and try again", which never helps. A `_codeKeys` map is
  now read after the throttle check and before the status switch, and
  `error.githubRepoNotVisible`, `error.githubRepoUrlInvalid` and
  `error.githubRepoAlreadyLinked` exist in `en_shell.dart` and `zh_shell.dart`.
  The key names follow the camel-case of `error.badRequest`, not the dotted
  names one finding floated. The English advice for a repo GitHub does not show
  the server is "install the InsideOut GitHub App on the repository, then link
  it again". A code the map does not know falls through to the status copy, so
  every other error reads as before.
- **A refresh that merely failed no longer ends presence and progress** (F01).
  FULL-05 keeps the session when a refresh fails for a reason that says nothing
  about the token (a dropped connection, a 5xx), but `coachSseWithRefresh`
  (`client/lib/api/sse.dart`) then threw `ApiException` with status 401, which
  the roadmap page's `_retrying` loop takes for "signed out" and stops on, so
  live presence and progress stayed dead for a user who was still signed in.
  When the session is kept the stream now throws the refresh error's own status
  (a 429 stays a 429) or, when it has none, 503: the loop backs off and reopens.
  401 is reserved for the two paths that clear the session (a refused refresh,
  a retry that is still 401).
- **A refresh another tab won is not the end of the session** (F112, client
  half). One-time refresh tokens mean two tabs that expire together race, and
  the server answers the loser 401 (inside its 10 s reuse grace it only spares
  the session family). The loser took that for a dead session and cleared the
  shared token store, wiping the pair the winner had just written. `refresh()`
  (`client/lib/api/client_core.dart`) now looks in the store after a 401 or 403:
  if a refresh token other than the one it sent appears within 500 ms (polled
  every 25 ms), another tab rotated it, so `refresh()` returns normally and
  every caller re-reads the store (the interceptor's retry and the stream
  reopen both do, so both carry the winner's bearer). Nothing new appears, so
  the refusal stands and the session clears as before. This mirrors the Go
  client's `renewalGrace` path in `server/internal/apiclient/refresh.go`. The
  server keeps answering 401; no other status was asked for.
- **A proposal is decided once** (F02, F08, F15, client half).
  `ProjectUpdatesFeed` is now stateful (`project_updates_feed.dart`). It keeps
  the proposals whose decision is on the wire (with which button was pressed)
  and those whose decision has landed. While a decision is pending, the pressed
  button shows `busy` and the other is off; once it has landed both stay off,
  even while the list the feed holds does not show the decision yet. `_decide`
  also returns early for either set, because a second tap can arrive before the
  frame that disables the buttons. A refused decision (`guardWrite` returned
  false) puts both buttons back. Other proposals are unaffected, and a proposal
  that already carries a decision still shows the note menu instead of buttons.

## Other callers and strings

- `coachSseWithRefresh` has two callers, both in `client_stream.dart`: the GET
  streams (presence, progress), whose consumer is the roadmap retry loop, and
  the coach send. The roadmap loop is the only place in `client/lib` that reads
  a stream's 401, 403 or 404 as final (`roadmap_page.dart`, unchanged). A coach
  send whose refresh merely failed now reports 503 or 429 copy ("Server
  hiccup", "Too many attempts") where it said the session expired, which is
  the truer thing to say to a user who is still signed in.
- `refresh()` is reached from four places: the 401 interceptor, the two stream
  openers (through the single-flight wrapper) and `SessionState.hydrate`. All
  four get the adopt-and-carry-on behavior, which is what a second tab loading
  while the first rotates needs. `hydrate` already kept the session on a failure
  that was not a 401 or 403 and still clears on one.
- The 409 that stage decline and confirm answer (`conversations.go`,
  `answerStage`) carries no code, so there is nothing to switch on. It keeps
  the generic conflict copy, which is accurate there (the stage moved on). The
  `rv4-FR-F03` text mentions it; no change is needed for it.
- `guardWrite` is the route for the repo card (`project_github_card.dart`), so
  the new copy appears as "Write failed: <advice>" with no change to the card.
- The three new keys are read by `localizedError`, so `l10n_unread_keys_test`
  and the en/zh parity test in `l10n_test.dart` stay green. The Chinese strings
  were written beside the English and follow the tone of the neighbouring keys.
  They have had no native review.

## Evidence

Red commits touch tests only; fix commits touch sources only (checked with
`git show --stat` for all eight). The runs below are `flutter test` on the
named files.

| Root | Red (tests at the red commit, sources as at the base) | Green (HEAD, `784291a`) |
| --- | --- | --- |
| A | `errors_l10n_test.dart` + `project_page_test.dart` at `bfdfb54`: `+11 -10`. In `errors_l10n_test.dart` two tests fail per code (`has its own words in both languages`, `reads as its own advice, not as a bare 400`/`409`) plus `a private repo is told to install the GitHub App, then link again`; in `project_page_test.dart` the three `a repo save refused with <code> shows its advice` widget tests fail. Example: `Expected: 'error.githubRepoNotVisible'  Actual: 'error.badRequest'`. The guard `the same status without a repo-link code keeps the status copy` passes at the base | same two files `+21: All tests passed!` |
| B, F01 | `api_refresh_clearing_test.dart` + `rv4_review_client_api_test.dart` at `16252f5`: `+9 -6`. Unit: a network error, a 503 and an answer with no status each `threw ApiException:<stream failed>` where a status `not (401 or 403 or 404)` was expected, and a refresh refused for rate lost its 429. Page: the dropped-connection and 503 cases stop after one presence open (`Expected: <2>  Actual: <1>`, reason `the loop took a kept session for a signed-out one`) | `api_refresh_clearing_test.dart` `+13`; `rv4_review_client_api_test.dart` `+8` (two of them the page tests) |
| B, F112 | `rv4_refresh_rotation_test.dart` at `a8edde07`: `+1 -4`. The 401 case, the 403 case and the late winner (120 ms) each throw the server's `invalid or expired session` out of `me()` with the winner's pair already, or soon, in the store; the presence stream dies with `stream failed` instead of reopening with the winner's bearer. The fifth test, no rotation so the session clears, passes at the base: it guards the other side | `+5: All tests passed!` |
| C | `rv4_review_client_api_test.dart` at `959101d7`: `+3 -3`. Two taps: `Expected: an object with length of <1>` with two `(accept: true, id: up1)` calls, reason `each decision adds a history note every member reads`. Accept pending: the Accept button is not `busy` (`Expected: true  Actual: <false>`). Decision landed: `Expected: null  Actual: <Closure: () => void>`, the button is still live. Three guards pass at the base (a refused decision puts the buttons back, other proposals stay live, a decided proposal has no buttons) | `+8: All tests passed!` (two of them the F01 page tests) |

Mutation checks on the fixes (the source file of each root put back as it was
at the base, the tests of HEAD kept, then restored; the tree is clean):

- **A.** `errors_l10n.dart` reverted: `+15 -6`. The three `reads as its own
  advice` tests and the three repo-save widget tests fail. The `own words` tests
  and the English-advice test stay green, since they read the tables, which this
  mutation leaves.
- **B, F01.** `sse.dart` reverted: `+15 -6`. The four unit tests and both page
  tests fail.
- **B, F112.** `client_core.dart` reverted: `+1 -4`, the same four tests as the
  red run; the no-rotation guard stays green.
- **C.** `project_updates_feed.dart` reverted: `+5 -3`, the same three tests as
  the red run; the three guards stay green.

## Design notes

- **The status of a kept session.** The refresh error's own status when it has
  one, else 503. That keeps a 429 readable as "throttled" and never produces a
  401, 403 or 404, the three the roadmap loop stops on. A 404 from the refresh
  endpoint itself is passed through and would stop the loop, as a 404 from the
  stream open already did at the base; it is not special-cased.
- **Adopting is returning.** The token store is the shared state (the secure
  store writes the access token first and the refresh token second, so a
  changed refresh token means the new access token is already there), and every
  caller re-reads it on each request. So `refresh()` needs no new return value:
  success without writing is the adoption. The poll counts steps instead of
  reading the clock, so it behaves under fake time.
- **The grace is a private constant** (`_rotationGrace`, 500 ms), not a
  constructor parameter, because nothing needs to inject it. The server's
  `refreshReuseGrace` (10 s) covers it with a wide margin, as it covers the Go
  client's.
- **A landed decision locks for the feed's lifetime.** `_reload` on the page
  returns at once and the list can be stale for a moment, so the lock does not
  wait for the list. If the reload then fails, the buttons stay off until the
  page leaves the tree; the alternative is the re-tap the findings describe.
- `roadmap_page.dart`, `client_auth.dart` and `token_store.dart` were among the
  cluster's files and needed no change; `roadmap_page.dart` (337 lines) has not
  grown. The new tests went into new files: the F01 page tests and the feed
  tests share `rv4_review_client_api_test.dart` (328 lines), and the rotation
  tests have `rv4_refresh_rotation_test.dart` (187) because the first has no
  room left under the 350-line cap.

## Gates

Run in the worktree; the cluster is Dart only.

| Command | Result |
| --- | --- |
| `(cd client && flutter pub get --enforce-lockfile)` | exit 0; `client/pubspec.lock` unchanged |
| `flutter analyze --no-fatal-infos` | exit 0; 14 infos, all in `lib/api/*`: `requests.dart` 7, `client_prd.dart` 2, `client_stream.dart` 1, and the four `prefer_initializing_formals` on the untouched constructor lines 17 to 20 of `client_core.dart`. The same 14 the base reports; none on a line of this cluster |
| `flutter test` | 565 passed |
| `TZ=Asia/Shanghai flutter test` | 565 passed |
| `dart format --output=none --set-exit-if-changed lib test` | exit 0 |
| `gofmt -l server/internal server/cmd` | empty |
| 350-line gate on the changed `*.go` and `*.dart` files | nothing over (`sse.dart` 326, `rv4_review_client_api_test.dart` 328, `project_page_test.dart` 317, `project_updates_feed.dart` 188, `client_core.dart` 150) |
| `scripts/tests/check_*.py` (14 scripts) | all exit 0 |
| `sh scripts/check-redaction.sh`, `sh scripts/check-docs-recorded.sh` | both exit 0 |

The extra 500 ms is paid only by a refresh the server refuses and no other tab
answered, so the existing tests that expect a refused refresh to clear the
session are about half a second slower each; the suite still runs in about 43
seconds.

## Known ceilings, recorded on purpose

- **The server half of F02, F08 and F15 is not here.** A second or opposite
  decision still rewrites the stored decision and adds another timeline note
  when it reaches the server by another route (a second tab, a script). That is
  `w4-rv-store-db`'s. This cluster stops the web feed from sending it twice.
- **The server still answers the loser of a rotation with 401 and clears its
  cookies** (R3-11 in the [open findings](../../issues/2026-10-18-review-4-open-findings.md)).
  The Flutter client keeps its pair in the token store, sends the refresh token
  in the request body and the access token as a Bearer header, so that clear
  does not touch it. A client that leaned on the cookies would still be hit.
  The finding's alternative (return the winner's pair, or a non-401, inside the
  grace) was not taken: the cluster note says the server keeps its 401.
- **A genuinely dead session signs out up to 500 ms later** than before, since
  the refused refresh now waits out the poll before it clears.
- **No browser run.** The tests drive the real dio stack over a loopback
  server in the Dart VM, the roadmap page and the feed in widget tests with
  the repository's fake `ApiClient`, and the raw-fetch streams through
  `fetchAttemptForTests`. `client/test/web` was not extended and no Flutter web
  build was opened (R3-07 still stands). Two real tabs sharing one
  `localStorage` have not been exercised; the tests stand the store in with a
  shared object.
- **Kept-session wording on the coach send** changed from "session expired"
  (401) to the server-hiccup or throttled copy. No test pinned the old wording.

## Left for the integrator

No code is waiting. The documentation lines are not this cluster's files:

- INTEGRATOR TODO: `docs/architecture/frontend.md` (the Coach chat paragraph at
  lines 73 to 78, or a new Transport paragraph): a stream open that meets a 401
  refreshes once and reopens; a refused refresh clears the session and ends the
  stream with 401; one that merely failed keeps the session and throws a status
  the roadmap loop retries (its own, else 503); a refresh that lost a rotation
  to another tab adopts the stored pair within 500 ms (rv4-FR-F01, F112).
- INTEGRATOR TODO: `docs/architecture/cli-mcp-and-client-config.md` (the
  `renewalGrace` sentence at line 96): the web client has the same 500 ms wait
  (`_rotationGrace` in `client/lib/api/client_core.dart`).
- INTEGRATOR TODO: `docs/architecture/backend.md` (the `refreshReuseGrace`
  sentence at line 82): a loser inside the grace still gets 401 with cleared
  cookies; both first-party clients now adopt the winner's stored pair instead of
  ending the session. The cookie clear itself (R3-11) is still open.
- INTEGRATOR TODO: `docs/architecture/product-subsystems.md` (lines 50 to 53):
  after "creates no node of its own", the feed sends one decision per proposal;
  the buttons are busy or off while it is pending and after it lands (rv4-FR-F02,
  F08, F15). At lines 96 to 101: the three repo-link codes have English and
  Chinese copy in the web client.
- INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md`: R3-11 and
  R2-12 now have a client half (the Dart client mirrors the Go wait and adopts
  the winner's pair); the register rows for the eight findings become `FIXED`.
- INTEGRATOR TODO: `docs/plans/2026-10-18-multirole-4.md` and
  `docs/plans/2026-10-18-multirole-4.track.yaml`: the `w4-rv-client-api` step
  and its history.
- INTEGRATOR TODO: this folder's `index.md` (a row for this fragment),
  `docs/changelogs/README.md` and `docs/changelogs/index/2026-10.md`.
- Optional learning for the records agent: a failed refresh has three outcomes
  (the session is over, another tab already rotated it, the attempt merely
  failed), and every consumer of the status that follows has to agree which of
  them it is looking at; F01 and F112 were one disagreement seen from two
  sides.
