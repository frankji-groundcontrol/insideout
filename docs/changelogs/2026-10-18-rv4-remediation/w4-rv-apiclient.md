# Wave 4, the Go API client: renewal trouble, Guide's 401, the final-401 advice, interrupted renewals, Logout — review-4 remediation

Branch `rv4/w4-rv-apiclient`, `582a4bd..cd98147` (ten commits: five red/fix
pairs), a test-margin commit (`234f4724`, see the design notes), a repair round
after the verifier's first pass (`61281ffc..001f2f99`, five commits: a test move
and two red/fix pairs, see "Repair round 1"), and this fragment. Cluster
`w4-rv-apiclient` of the whole-change
review of `git diff d53035a..rv4/integration`. Findings are cited as
`rv4-<ID>` because review IDs restart per review; `FR` is that review's prefix.
Eight rows, all `FIXED`, in five roots; the repair round reopened roots C (F152)
and E (F70). The Logout tests run against a private PostgreSQL (`needs_db`);
its port was torn down. No address, host, secret or token is spelled in this
fragment.

| Finding | Root | Red | Fix |
| --- | --- | --- | --- |
| `rv4-FR-F151`, `rv4-FR-F164` | A. a refused renewal (429, 503, any 5xx) reads as an ended session | `e98b3bb4` | `cd14d802` |
| `rv4-FR-F169`, `rv4-FR-F165`, `rv4-FR-F157` | B. `Guide` has a request path of its own with a bare 401 | `5e6c7038` | `6bea7860` |
| `rv4-FR-F152` | C. the final 401 can only say `insideout login` | `545bb6d0` | `36f5fe75` |
| `rv4-FR-F66` | D. an interrupt drops a renewal in flight | `1a5f51c2` | `2811d4bf` |
| `rv4-FR-F70` (client half, row `rv4-FR-F70-api`) | E. no way to end the server session behind a login | `4bb60e46` | `cd981475` |

Findings that share a root share its red test. Each finding is still its own
row in the report, and the test names say which roots they cover. The repair
round adds a second pair to C (`566b4021` red, `001f2f99` fix) and to E
(`22a845ae` red, `34210174` fix, after the move `61281ffc`).

## What changed

- **A renewal the server declined is not a verdict on the session** (F151,
  F164). `renewSession` (`server/internal/apiclient/refresh.go`) answered only
  200, 401 and 403 and returned "the refusal stands" for everything else, so the
  per-IP throttle on `/auth/refresh` (429), a deploy in progress (503) and any
  5xx made the caller report the original 401: "the session expired or was
  revoked; run `insideout login`". A user whose session was fine was told to
  sign in again. The 401/403 case now ends with its own return and any other
  status falls through to an error that names `POST /auth/refresh`, the status
  and "try again; run `insideout login` only if this persists". It is not
  `ErrUnauthorized`, the stored session is untouched, and the client stays armed
  (the next call tries again), as it already did for an unreachable server.
- **`Guide` takes the shared raw-GET path** (F169, F165, F157). `Guide`
  (`roadmap.go`) built its own GET and turned any status of 300 or more into a
  bare `apiclient: GET guide: 401 Unauthorized: {...}`: no `ErrUnauthorized`, no
  advice, and a second place that had to be kept in step with the shared
  request path. It is now `getBytes(apiPath("projects", id, "guide"))`, the same
  call `PrdExport` makes, so it gets the context and deadline, the renewal and
  the final 401 text like every other raw read. `roadmap.go` lost its own
  response handling and three imports.
- **A client can replace the final 401's advice** (F152).
  `Client.SetUnauthorizedHint(hint)` swaps the advice a final 401 gives *this
  client* for `hint` (`""` restores the default text, byte for byte). The
  default says `insideout login`, which cannot cure a client whose credential is
  not the saved login: the MCP server's in-process sign-in. The hint survives
  `SetRenewal` in either order and a rejected renewal (at the base the
  rejection dropped the whole `Renewal`, and the hint with it; it now sets a
  `rejected` flag on it, and the hooks and the hint stay: see "Repair round 1").
  A `Renewal` without both hooks is a carrier of the hint and is never armed.
  `AgentScoped` returns a new client and does not copy the hint; set it on the
  client it returns.
- **`SetRenewal` with caller-supplied hooks** (F152, second half). It already
  works: apiclient does no file I/O, only the hooks do (the CLI's are in
  `clientconfig`). `refresh_test.go` runs every renewal test on in-memory hooks
  (`file`, a struct holding the pair), so an in-process login that keeps its own
  refresh token can arm its client with closures over a variable. No code was
  needed; the doc comment on `SetRenewal` now says so. This is what
  `w4-rv-mcpserver` builds on (see the handover below).
- **An interrupt no longer drops a renewal in flight** (F66). The server rotates
  the refresh token as it reads `POST /auth/refresh`. `cliCtx` was bound to that
  request, so a Ctrl-C aborted it before the answer, and the forced exit after
  `interruptGrace` could end the process mid-renewal; either way the saved login
  held a spent token and the next command said to sign in again, while
  `runInterruptible`'s comment promised exactly that could not happen.
  `postRefresh` now runs the request on a context that keeps the caller's
  deadline but not its cancellation, and gives the answer in flight
  `renewalDrain` (750 ms) after the caller is cancelled before it drops the
  request. A caller already gone before the request is sent sends nothing, so
  no token is spent for nobody. `apiclient.WaitForRenewal()` returns once no
  renewal is in flight (bounded at `2*renewalDrain`), and `runInterruptible`
  calls it before the forced `os.Exit(130)`. Its comment now states what holds
  and the ceiling.
- **`Client.Logout()` ends the server session** (F70, client half).
  `insideout logout` only deleted the saved file, so a copy of it (a backup, a
  stolen laptop) kept a refresh token that renews for up to 30 days by default
  (`INSIDEOUT_REFRESH_TTL`, `720h`). `Logout`
  (`logout.go`) posts the refresh token to `POST /auth/logout`, which revokes
  the session behind it (no bearer; the server answers 200 for any token and
  clears cookies). A client with no refresh token sends nothing and returns nil.
  An unanswered request (offline, DNS, timeout) returns an error wrapping the
  exported `ErrLogoutUnreachable`; a refusal names its status; a cancellation
  wraps `context.Canceled` and is not reported as an outage. The access token
  already issued is stateless and lives out its short TTL; only the refresh
  token, the long-lived credential, is revoked. A nil return leaves the client
  empty (no access token, no refresh token), so it cannot present a revoked
  login again; any error leaves both in place, so a retry presents the same
  refresh token (see "Repair round 1").

## Evidence

Red commits touch tests only (checked with `git show --stat`); the F152 red and
the F70 red name a method the client does not have yet through an interface
assertion (`rvApiClientHas`), so the package still compiles and the rest of its
tests run at the red commit. Fix commits carry the sources and any tests that
cannot be red (guards, the registry rows). Outputs are from `go test -race`.
This table and the mutation list are the first pass; the repair round's own
evidence is under "Repair round 1".

| Root | Red (tests at the red commit, sources as at the base) | Green (HEAD) |
| --- | --- | --- |
| A | `TestRenewalTroubleIsNotAVerdictOnTheSession/429` and `/503` at `e98b3bb4`: `` call 0: err = apiclient: GET /me: unauthorized: the session expired or was revoked; run `insideout login` to sign in again (...), want the failed renewal named with its status, not a verdict on the session `` | both subtests `PASS`; the same test also asserts the client tried the refresh twice and saved nothing |
| B | `TestUnauthorizedNamesTheFix` at `5e6c7038`, signed in and signed out: `err = apiclient: GET guide: 401 Unauthorized: {"error":"invalid or expired session"}, want ErrUnauthorized naming insideout login and INSIDEOUT_TOKEN` | `PASS` |
| C | `TestRvApiClientUnauthorizedHintIsPerClient` and `TestRvApiClientHintSurvivesARejectedRenewal` at `545bb6d0`, all four subtests: `*Client has no SetUnauthorizedHint` | all `PASS`; `TestSweepCoversEveryClientMethod` and `TestHostileValuesStayOneSegmentOrOneQueryValue/SetUnauthorizedHint` `PASS` |
| D | `TestRvApiClientCancelledRenewalKeepsTheRotatedPair` at `1a5f51c2`: `stored session = {access:old refresh:ref-1 saved:0}, want the pair the server rotated to (new/ref-2) saved once` (the stub asserts it did rotate). `TestRvApiClientInterruptedRenewalStillSavesTheRotatedPair` in `cmd/insideout`, a child process hit with SIGINT while its renewal is in flight, both subtests: `saved config after the interrupt = {... Token:acc-old RefreshToken:ref-old}, want the pair the server rotated to {... Token:acc-new RefreshToken:ref-new}` | all `PASS` (0.3 s and 0.4 s); `TestCancelledContextAbortsARenewal`, `TestCtrlCAbortsTheRequestInFlight`, `TestCtrlCAtAPromptEndsTheCommand` and the renewal tests in `cmd/insideout` stay green |
| E | `TestRvApiClientLogoutRevokesTheServerSession` at `4bb60e46`, real handlers over real PostgreSQL (two sessions of one user): `*Client has no Logout` | `PASS`: after `Logout` the leaving session's refresh token answers 401 and the control session's answers 200; `TestRvApiClientLogoutRequestAndFailures` `PASS` |

Mutation checks (the hunk put back or the key condition flipped, the tests of
HEAD kept, then restored; the tree is clean after each):

- **A.** the fall-through replaced by `return "", nil`: both subtests of
  `TestRenewalTroubleIsNotAVerdictOnTheSession` fail.
- **B.** `roadmap.go` as at the base: `TestUnauthorizedNamesTheFix` fails for
  signed in and signed out.
- **C.** the hint branch of `unauthorized` removed: all four subtests fail. The
  rejection put back to `c.renew = nil`: `TestRvApiClientHintSurvivesARejectedRenewal`
  fails in both orders.
- **D.** the `WaitForRenewal` call removed from `runInterruptible`: only the
  "grace runs out first" subtest fails, with the old pair on disk. The
  pre-send check removed from `postRefresh`:
  `TestRvApiClientCancelledBeforeTheRenewalSendsNothing` fails (`a refresh token
  was spent for a caller that had gone`). The detachment itself needs no
  mutation: the base behavior is what the red commit shows (the pair is lost).
- **E.** nothing sent: the real-API test fails (`after Logout the refresh token
  still renews: HTTP 200, want 401`) and so does every case of the request test.
  The `context.Canceled` exclusion removed: the cancelled case fails
  (`could not reach the server: ...: context canceled`).

## Repair round 1

The verifier accepted seven rows and required three repairs, in roots E and C.
Each is a red commit (tests only, run to failure) and a separate fix.

**1 and 2. `Logout` leaves the client empty, and a 401 is "already signed out"
(F70).** `Logout` returned nil with the access and refresh tokens still on the
client, so a caller that went on using it presented a login the server had just
revoked. And a 401 from `POST /auth/logout` was an error, though the handler is
public and idempotent (it answers 200 for any token): a 401 can only come from
something in front of the server, and "this credential is no longer valid" is
what logout wanted anyway.

- Fix `34210174` (`logout.go`, `auth.go`): a nil return sets `c.token` and
  `c.refresh` to `""`, also for a client that had no refresh token (it sends
  nothing). `authPost` now carries a refusal's status in `authBody.status` (the
  error text is unchanged) and `Logout` takes 401 as nil, without a retry.
  Every other failure (another status, no answer, a cancellation) keeps both
  tokens: a second `Logout` on an emptied client would send nothing and report
  a revocation that never happened.
- `61281ffc` moved the Logout tests to `rv4_review_logout_test.go`, unchanged
  (see the design notes). The real-API test now reads the refresh token before
  the call, since the client is empty after it, and still checks the server: the
  spent token answers 401, another session of the same user 200.
- Residual: if a gateway swallowed the request, the revocation never reached
  the server and the refresh token lives until its TTL. The client cannot tell,
  and a retry would meet the same gateway.

**3. A call queued behind a rejected renewal ends with the refusal (F152).**
Renewals are serialized on a process-wide lock. When the server refused the
refresh token for good, the winner disarmed the client by clearing `Reload` and
`Save`; calls that had already seen their own 401 were queued on the lock and
called the nil hooks next (a nil-pointer panic), and the unlocked `armed()` read
in `RoundTrip` raced with that write. Re-checking `armed()` once the lock is
held removes the panic but not the race (M6 below: the test passes without
`-race` and fails with it), because the hooks are still written while other
calls read them.

- Fix `001f2f99` (`refresh.go`): the hooks never change after `SetRenewal`.
  `Renewal.rejected` records the refusal and is read and written only under the
  renewal lock; a call that takes the lock and finds it set returns "the refusal
  stands", as the first call did. The hint stays. After a rejection `armed()`
  stays true, so a later final 401 takes the lock once more (it waits only if a
  renewal is in flight) and returns at once.
- `TestRvApiClientCallsQueuedBehindARejectedRenewalEndWithTheRefusal`
  (`rv4_review_concurrent_test.go`): four goroutines call `Whoami` on one armed
  client; a barrier makes the stub refuse all four together, and it refuses the
  one refresh with 401. Each call must end `ErrUnauthorized` telling to sign in,
  the dead token is offered once, nothing is saved.

| Red (tests at the red commit, sources as at `dbd2d89c`) | Green (HEAD) |
| --- | --- |
| `22a845ae`, `TestRvApiClientLogoutRequestAndFailures`: `answered 200: the client holds access "access" and refresh "ref-1", want neither`; the same for no refresh token; `answered 401: err = POST /auth/logout: 401 Unauthorized: , want nil (already signed out)`; `answered 401: the client holds access "access" and refresh "ref-1", want neither` | `PASS`, with `TestRvApiClientLogoutRevokesTheServerSession` (real PostgreSQL) and the `Logout` rows of the hostile-value and registry sweeps |
| `566b4021`, `TestRvApiClientCallsQueuedBehindARejectedRenewalEndWithTheRefusal`: `call 1: err = panic: runtime error: invalid memory address or nil pointer dereference, want the refusal (ErrUnauthorized telling to sign in again)` (calls 2 and 3 too), `WARNING: DATA RACE`, `race detected during execution of test` | `PASS` (0.08 s); `-race -count=30` stable; the whole package `ok` under `-race` |

Mutation checks against the HEAD tests (each restored, tree clean after each):

- **F70.** M1 `Logout` not emptying the client: fails at the 200, no-refresh and
  401 cases. M3 emptying it on failure too: fails at the 503, no-answer and
  cancelled cases (`want both kept for a retry`). M2 a 401 returned as an error
  again: `answered 401: err = POST /auth/logout: 401 Unauthorized: , want nil`.
  M8 nothing sent: `after Logout the refresh token still renews: HTTP 200, want
  401` and the request test. M9 the `context.Canceled` exclusion pointed
  elsewhere: `cancelled: err = could not reach the server: ...: context canceled`.
- **F152.** M4 the `rejected` check disabled: `the dead refresh token was
  offered 4 times, want once`. M5 the rejection clearing the hooks again: three
  of the four calls panic on a nil hook and the race detector fires. M6 the
  narrower repair (hooks cleared, `armed()` re-checked under the lock): `race
  detected during execution of test` with `-race`, `ok` without. M7 the rejection
  dropping the whole `Renewal`: `TestRvApiClientHintSurvivesARejectedRenewal`
  fails in both orders.

## Design notes

- **The drain is a ceiling, not a guarantee.** An answer slower than
  `renewalDrain` after the cancellation, or a second signal (the CLI keeps its
  rule that a second signal ends the process at once), still loses the pair. A Ctrl-C taken
  during a renewal can now take up to 750 ms longer to end the command, only
  then. The way to remove the ceiling is a two-step rotation on the server
  (confirm the new pair before the old token dies); that is a server design
  change, not taken here. The sentence is on `renewalDrain`.
- **The drain is read before the goroutine starts.** The cancel callback runs on
  a goroutine of its own; reading the package variable there raced with a test
  that changes it (the race detector said so). `postRefresh` copies it first.
- **Why not abort and retry.** The renewal could be re-sent after an abort, but
  the first request may already have rotated the token and its answer is gone;
  a re-send presents the spent token, which the server refuses (and, past the
  reuse grace, treats as reuse and ends the whole token family). Waiting for
  the answer in flight is the only safe course.
- **The timing margin of the CLI test.** The stub of
  `TestRvApiClientInterruptedRenewalStillSavesTheRotatedPair` answers 300 ms
  and 400 ms after the request, against the 750 ms drain and a 100 ms
  `interruptGrace` in the second case. `234f4724` lowered the second answer
  from 500 ms to 400 ms: the margin before the answer could miss the drain on
  a loaded machine is 350 ms, and the answer still lands after the grace, so
  the forced-exit path stays covered (the `WaitForRenewal` mutation still
  turns exactly that case red).
- **`WaitForRenewal` polls `renewing.TryLock()`** every 10 ms. The renewal holds
  that lock from before its POST until after its save, so "unlocked" means "no
  pair is half-written". It adds no state.
- **The escape sweep.** `escape_test.go` (not this cluster's) lists every
  exported `Client` method and fails on one it does not know. `SetUnauthorizedHint`
  and `Logout` register their own rows from an `init()` each, in
  `rv4_review_apiclient_test.go` and `rv4_review_logout_test.go`, so the
  registry file stays untouched and keeps working. The `Logout` row uses a copy
  of the sweep's client with the hostile value as the refresh token (it travels
  in the body, never the URL).
- **Where the tests live.** Adding the repair assertions to
  `rv4_review_apiclient_test.go` (337 lines) would have taken it past the
  350-line cap, so the Logout tests moved, unchanged, to `rv4_review_logout_test.go` (189
  lines) in a commit of their own (`61281ffc`) before anything was added. The
  concurrency test is in `rv4_review_concurrent_test.go` (83 lines); the
  apiclient file is 195. The child process tests are in
  `server/cmd/insideout/rv4_review_interrupt_test.go`.
- **A `Client` serves one caller at a time.** The rejection path is now safe for
  overlapping calls, but the rest is not: on the success path a renewal writes
  `c.token` and `c.refresh` under the lock while `do` reads `c.token` without
  it, and a burst of N expired calls on one client renews N times in turn (each
  later call finds the pair the winner stored equal to what the shared client
  now holds). Those are valid rotations, only wasteful. Callers that overlap
  serialize per client or give each call its own client (see the handover).

## Handover and the integrator's list

For `w4-rv-mcpserver` (it builds on C, D and E):

- Keep the refresh token the sign-in returned instead of dropping it
  (`auth_tools.go` calls `setSessionToken(c.Token())`), arm the in-process
  client with `SetRenewal(refresh, apiclient.Renewal{Reload: ..., Save: ...})`
  over in-memory hooks, and call `SetUnauthorizedHint("...call the auth_login
  tool again...")` on it. If `AgentScoped` mints a client for the call, set the
  hint on the client it returns.
- A tool call that is cancelled mid-renewal now keeps the rotated pair (D), so
  the in-memory hooks see `Save` even when the call returns `context.Canceled`.
- Overlapping tool calls on one client end cleanly when the refresh token is
  rejected for good (every call gets the refusal and the hint), but a burst of
  expired calls still renews once per call and shares unlocked token fields (see
  the design notes). Serialize tool calls per client, or give each concurrent
  call its own client armed with the same hooks and hint.
- A refusal stays on the client until the next `SetRenewal` (a new `Login` does
  not clear it, as at the base, where it dropped the whole `Renewal`): re-arm
  the client after every new sign-in.

For `w4-copy-sweep` (the CLI half of F70):

- `insideout logout` should call `c.Logout()` with the saved refresh token (the
  client from `clientconfig` is armed, `RefreshToken()` returns it), then delete
  the local file. On an error wrapping `apiclient.ErrLogoutUnreachable`, say the
  session could not be revoked on the server and delete the file anyway; on any
  other error name it. A cancelled `Logout` is an interrupt, not an outage.
- A nil return means the server session is gone or was never there (a 401 counts
  as already signed out) and the client is empty: read `RefreshToken()` before
  the call if it is needed after. On an error the client keeps its tokens.

INTEGRATOR TODO items (shared indexes and files outside this cluster):

- INTEGRATOR TODO: `server/internal/apiclient/agent_token.go`: optional, make
  `AgentScoped` copy the login client's hint onto the client it returns (it
  needs `c.renew.hint`); until then the MCP cluster sets it on the result.
- INTEGRATOR TODO: `server/internal/apiclient/escape_test.go`: optional, fold
  the `SetUnauthorizedHint` and `Logout` rows (registered from
  `rv4_review_apiclient_test.go` and `rv4_review_logout_test.go`) into the
  registry and its `covered` map.
- INTEGRATOR TODO: `docs/architecture/cli-mcp-and-client-config.md`: the
  renewal paragraph (`renewalGrace`, around line 96) gains the drain: a call
  cancelled while a renewal is in flight waits up to 750 ms for the answer and
  saves it, and the CLI's forced exit waits for it (`WaitForRenewal`); a 429/5xx
  from `/auth/refresh` is reported as trouble, not as an ended session; a
  refresh token the server refused for good is remembered by the client, and
  calls queued behind that refusal end with it; `Logout` revokes the server
  session, takes a 401 as already signed out and empties the client.
- INTEGRATOR TODO: `docs/usage/` (the CLI reference, if it describes `logout`):
  once `w4-copy-sweep` lands, `logout` also revokes the session on the server.
- INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md`: the eight
  rows become `FIXED` (F70 with the CLI half pending on `w4-copy-sweep`), plus
  two ceilings: a 401 from `/auth/logout` may mean a gateway swallowed the
  revocation, and one `Client` still serves one caller at a time.
- INTEGRATOR TODO: `docs/plans/2026-10-18-multirole-4.md` and
  `docs/plans/2026-10-18-multirole-4.track.yaml`: the `w4-rv-apiclient` step and
  its history.
- INTEGRATOR TODO: this folder's `index.md` (a row for this fragment),
  `docs/changelogs/README.md` and `docs/changelogs/index/2026-10.md`.
- Optional learning for the records agent: a request that mutates server state
  as it is read (a one-time token rotation) cannot be cancelled like a read; the
  client either waits for the answer or loses the state.
