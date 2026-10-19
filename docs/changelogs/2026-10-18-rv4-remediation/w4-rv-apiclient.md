# Wave 4, the Go API client: renewal trouble, Guide's 401, the final-401 advice, interrupted renewals, Logout — review-4 remediation

Branch `rv4/w4-rv-apiclient`, `582a4bd..cd98147` (ten commits: five red/fix
pairs), then this fragment. Cluster `w4-rv-apiclient` of the whole-change
review of `git diff d53035a..rv4/integration`. Findings are cited as
`rv4-<ID>` because review IDs restart per review; `FR` is that review's prefix.
Eight rows, all `FIXED`, in five roots. The Logout test ran against a private
PostgreSQL (`needs_db`); its port was torn down. No address, host, secret or
token is spelled in this fragment.

| Finding | Root | Red | Fix |
| --- | --- | --- | --- |
| `rv4-FR-F151`, `rv4-FR-F164` | A. a refused renewal (429, 503, any 5xx) reads as an ended session | `e98b3bb4` | `cd14d802` |
| `rv4-FR-F169`, `rv4-FR-F165`, `rv4-FR-F157` | B. `Guide` has a request path of its own with a bare 401 | `5e6c7038` | `6bea7860` |
| `rv4-FR-F152` | C. the final 401 can only say `insideout login` | `545bb6d0` | `36f5fe75` |
| `rv4-FR-F66` | D. an interrupt drops a renewal in flight | `1a5f51c2` | `2811d4bf` |
| `rv4-FR-F70` (client half, row `rv4-FR-F70-api`) | E. no way to end the server session behind a login | `4bb60e46` | `cd981475` |

Findings that share a root share its red test. Each finding is still its own
row in the report, and the test names say which roots they cover.

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
  `SetRenewal` in either order and a rejected renewal (the rejection now clears
  `Reload` and `Save` in place instead of dropping the whole `Renewal`, which
  used to take the hint with it). A `Renewal` without both hooks is a carrier of
  the hint and is never armed. `AgentScoped` returns a new client and does not
  copy the hint; set it on the client it returns.
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
  stolen laptop) kept a refresh token that renews for weeks. `Logout`
  (`logout.go`) posts the refresh token to `POST /auth/logout`, which revokes
  the session behind it (no bearer; the server answers 200 for any token and
  clears cookies). A client with no refresh token sends nothing and returns nil.
  An unanswered request (offline, DNS, timeout) returns an error wrapping the
  exported `ErrLogoutUnreachable`; a refusal names its status; a cancellation
  wraps `context.Canceled` and is not reported as an outage. The access token
  already issued is stateless and lives out its short TTL; only the refresh
  token, the long-lived credential, is revoked. The client keeps its credentials;
  the caller discards them.

## Evidence

Red commits touch tests only (checked with `git show --stat`); the F152 red and
the F70 red name a method the client does not have yet through an interface
assertion (`rvApiClientHas`), so the package still compiles and the rest of its
tests run at the red commit. Fix commits carry the sources and any tests that
cannot be red (guards, the registry rows). Outputs are from `go test -race`.

| Root | Red (tests at the red commit, sources as at the base) | Green (HEAD) |
| --- | --- | --- |
| A | `TestRenewalTroubleIsNotAVerdictOnTheSession/429` and `/503` at `e98b3bb4`: `call 0: err = apiclient: GET /me: unauthorized: the session expired or was revoked; run \`insideout login\` to sign in again (...), want the failed renewal named with its status, not a verdict on the session` | both subtests `PASS`; the same test also asserts the client tried the refresh twice and saved nothing |
| B | `TestUnauthorizedNamesTheFix` at `5e6c7038`, signed in and signed out: `err = apiclient: GET guide: 401 Unauthorized: {"error":"invalid or expired session"}, want ErrUnauthorized naming insideout login and INSIDEOUT_TOKEN` | `PASS` |
| C | `TestRvApiClientUnauthorizedHintIsPerClient` and `TestRvApiClientHintSurvivesARejectedRenewal` at `545bb6d0`, all four subtests: `*Client has no SetUnauthorizedHint` | all `PASS`; `TestSweepCoversEveryClientMethod` and `TestHostileValuesStayOneSegmentOrOneQueryValue/SetUnauthorizedHint` `PASS` |
| D | `TestRvApiClientCancelledRenewalKeepsTheRotatedPair` at `1a5f51c2`: `stored session = {access:old refresh:ref-1 saved:0}, want the pair the server rotated to (new/ref-2) saved once` (the stub asserts it did rotate). `TestRvApiClientInterruptedRenewalStillSavesTheRotatedPair` in `cmd/insideout`, a child process hit with SIGINT while its renewal is in flight, both subtests: `saved config after the interrupt = {... Token:acc-old RefreshToken:ref-old}, want the pair the server rotated to {... Token:acc-new RefreshToken:ref-new}` | all `PASS` (0.3 s and 0.5 s); `TestCancelledContextAbortsARenewal`, `TestCtrlCAbortsTheRequestInFlight`, `TestCtrlCAtAPromptEndsTheCommand` and the renewal tests in `cmd/insideout` stay green |
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
  was spent for a caller that had gone`). The detachment itself is the red
  commit's whole content.
- **E.** nothing sent: the real-API test fails (`after Logout the refresh token
  still renews: HTTP 200, want 401`) and so does every case of the request test.
  The `context.Canceled` exclusion removed: the cancelled case fails
  (`could not reach the server: ...: context canceled`).

## Design notes

- **The drain is a ceiling, not a guarantee.** An answer slower than
  `renewalDrain` after the cancellation, or a second signal (the CLI keeps "a
  second Ctrl-C ends the process at once"), still loses the pair. A Ctrl-C taken
  during a renewal can now take up to 750 ms longer to end the command, only
  then. The way to remove the ceiling is a two-step rotation on the server
  (confirm the new pair before the old token dies); that is a server design
  change, not taken here. The sentence is on `renewalDrain`.
- **The drain is read before the goroutine starts.** The cancel callback runs on
  a goroutine of its own; reading the package variable there raced with a test
  that changes it (the race detector said so). `postRefresh` copies it first.
- **Why not abort and retry.** The renewal could be re-sent after an abort, but
  the first request may already have rotated the token; the second would then
  present a spent token and, after the reuse grace, end the whole token family.
  Waiting for the answer in flight is the only safe course.
- **`WaitForRenewal` polls `renewing.TryLock()`** every 10 ms. The renewal holds
  that lock from before its POST until after its save, so "unlocked" means "no
  pair is half-written". It adds no state.
- **The escape sweep.** `escape_test.go` (not this cluster's) lists every
  exported `Client` method and fails on one it does not know. `SetUnauthorizedHint`
  and `Logout` register their own rows from an `init()` in
  `rv4_review_apiclient_test.go`, so that file stays untouched and the registry
  keeps working. The `Logout` row uses a copy of the sweep's client with the
  hostile value as the refresh token (it travels in the body, never the URL).
- **`rv4_review_apiclient_test.go` is 337 lines**, under the cap; the child
  process tests are in `server/cmd/insideout/rv4_review_interrupt_test.go`.

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

For `w4-copy-sweep` (the CLI half of F70):

- `insideout logout` should call `c.Logout()` with the saved refresh token (the
  client from `clientconfig` is armed, `RefreshToken()` returns it), then delete
  the local file. On an error wrapping `apiclient.ErrLogoutUnreachable`, say the
  session could not be revoked on the server and delete the file anyway; on any
  other error name it. A cancelled `Logout` is an interrupt, not an outage.

INTEGRATOR TODO items (shared indexes and files outside this cluster):

- INTEGRATOR TODO: `server/internal/apiclient/agent_token.go`: optional, make
  `AgentScoped` copy the login client's hint onto the client it returns (it
  needs `c.renew.hint`); until then the MCP cluster sets it on the result.
- INTEGRATOR TODO: `server/internal/apiclient/escape_test.go`: optional, fold
  the `SetUnauthorizedHint` and `Logout` rows (registered from
  `rv4_review_apiclient_test.go`) into the registry and its `covered` map.
- INTEGRATOR TODO: `docs/architecture/cli-mcp-and-client-config.md`: the
  renewal paragraph (`renewalGrace`, around line 96) gains the drain: a call
  cancelled while a renewal is in flight waits up to 750 ms for the answer and
  saves it, and the CLI's forced exit waits for it (`WaitForRenewal`); a 429/5xx
  from `/auth/refresh` is reported as trouble, not as an ended session; `Logout`
  revokes the server session.
- INTEGRATOR TODO: `docs/usage/` (the CLI reference, if it describes `logout`):
  once `w4-copy-sweep` lands, `logout` also revokes the session on the server.
- INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md`: the eight
  rows become `FIXED` (F70 with the CLI half pending on `w4-copy-sweep`).
- INTEGRATOR TODO: `docs/plans/2026-10-18-multirole-4.md` and
  `docs/plans/2026-10-18-multirole-4.track.yaml`: the `w4-rv-apiclient` step and
  its history.
- INTEGRATOR TODO: this folder's `index.md` (a row for this fragment),
  `docs/changelogs/README.md` and `docs/changelogs/index/2026-10.md`.
- Optional learning for the records agent: a request that mutates server state
  as it is read (a one-time token rotation) cannot be cancelled like a read; the
  client either waits for the answer or loses the state.
