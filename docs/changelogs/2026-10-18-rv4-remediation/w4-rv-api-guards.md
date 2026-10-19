# Wave 4, API guards — review-4 remediation

Branch `rv4/w4-rv-api-guards`, base `582a4bd` (the integration tip with wave 3
recorded), eight commits `fd72911..4f27145`: three red/fix pairs (F129, F135
with F140, and the F195 group), one test-only commit (F137) and one mirror probe
(F65), then this fragment. This is a repair-wave cluster, so the findings are
cited as `rv4-FR-<n>`, as the contract requires (`rv4-<ID>`). No host,
address, secret or token is spelled here; every database below was a private
throwaway PostgreSQL 18 cluster on loopback, and it was torn down afterwards.

## Dispositions

| ID | Status | Red | Fix | Note |
| --- | --- | --- | --- | --- |
| rv4-FR-F129 | `FIXED` | `fd72911` | `5fc09a2` | the build and expand confirms leave `deadlineExempt`; both are bounded by the 25 s request deadline |
| rv4-FR-F135 | `FIXED` | `c472cd3` | `8d58d2a` | the coach message cap counts characters, like every other text field |
| rv4-FR-F140 | `FIXED` | `c472cd3` | `8d58d2a` | the same line, the same fix; the grep-style guard the finding suggests was not added |
| rv4-FR-F137 | `FIXED` | mutation, no red commit | `edaf546` | the stream-cap tests pin the real ceilings (20, 100, post 26); test-strength row, see Evidence |
| rv4-FR-F195 | `PARTIAL` | `65e6339` | `caef0b6` | the named helpers refuse a remote database in strict mode; other gated tests do not yet |
| rv4-FR-F201 | `PARTIAL` | `65e6339` | `caef0b6` | the same; a database-name rule was not added (policy) |
| rv4-FR-F65 | `PARTIAL` | `65e6339`, `4f27145` | `caef0b6` | the same; `docs/usage/environment.md:191` still overstates what is covered (not an owned file) |

## What changed

- **The build and expand confirms run under the request deadline** (rv4-FR-F129,
  red `fd72911`, fix `5fc09a2`). `POST /prds/{pid}/build/confirm` and
  `POST /roadmap/{nid}/expand/confirm` only write the tree the caller already
  holds, yet they sat in `deadlineExempt` beside the model-calling routes, so a
  database that stopped answering held them until the caller gave up. The two
  patterns are gone from the map in `request_deadline.go`; five remain (the two
  passive streams, the coach message POST, `build` and `expand`) and the comment
  now says that only model calls belong there. The existing
  `request_deadline_routes_test.go` still fails when a pattern stops naming a
  registered route. The new `TestRvGuardsConfirmRoutesAreBoundedWhenTheDatabaseStalls`
  runs the real API over real PostgreSQL behind the `stallProxy` relay of the
  `/healthz` test (a server-side `statement_timeout` cannot fire on a statement
  that is never delivered, so only the request deadline can end the wait), with
  the deadline set to 300 ms and the client giving up after 4 s. A plain route
  is the control: it proves the stall is real and that a bounded route answers.
- **The coach message cap counts characters** (rv4-FR-F135 and rv4-FR-F140, red
  `c472cd3`, fix `8d58d2a`). `handleSendConversationMessage` compared
  `len(req.Content)`, bytes, with 10,000 while every other text field goes
  through `tooLong`, runes, so 3,334 characters of three bytes (10,002 bytes)
  were refused as "message too long" although the limit says 10,000 characters.
  The fix is the one line `tooLong(req.Content, 10_000)` in `conversations.go`.
  The cap table behind `TestFieldLimitsCountCharactersNotBytes`
  (`limits_runes_test.go`) gets a `coach message content` row with its three
  cases: more bytes than the limit and fewer characters, exactly the limit in
  characters, and one character over. The route world wires no coach, so an
  accepted message answers 503 after the check; the row says so with `ok=false`,
  as the register row does for its 403. The conversation comes from the helper
  `rvGuardsConversation`.
- **The stream-cap tests pin the real ceilings** (rv4-FR-F137, `edaf546`, tests
  only). The per-user test accepted any ceiling from 10 to 50, the project test
  any up to 134, the closing test any from 1 to 99, and the presence assertion
  (`own > 50`) could not fail on its own because every presence key belongs to
  an accepted stream that the first assertion already bounds. The tests now
  assert the numbers `w3-stream-bounds` records, as literals in
  `stream_caps_test.go` (`rvGuardsStreamsPerUser = 20`,
  `rvGuardsStreamsPerProject = 100`, `rvGuardsCursorPostsPerSec = 25`) and
  never as the constants under test, so a ceiling edited in `stream_caps.go`
  breaks a test until the docs and the pins are edited with it. The first
  refused cursor post must be post 26, and one member must hold exactly one
  presence session per open presence stream. There is no production change.
- **`INSIDEOUT_TEST_STRICT` now reaches the shared test helpers** (rv4-FR-F195,
  rv4-FR-F201 and rv4-FR-F65, red `65e6339`, fix `caef0b6`, mirror probe
  `4f27145`). The guard in `testlock` only ran in tests that called `Acquire`,
  `SignupMode` or `RequireLocalHost`; the shared doors that open the store never
  did, so strict mode against a remote `DATABASE_URL` wrote to that database.
  - `testlock.RequireLocalURL(t, dsn)` is the guard for a DSN no pool exists for
    yet. It has to run before the open, because `store.Open` pings and a check
    on the pool it returns has already reached the host. It shares one check
    (`requireLocal`) with `RequireLocalHost`; it never prints the host; a DSN
    that does not parse passes, because the open that follows reports it before
    it connects. Hosts are the configured one and every fallback, so a remote
    fallback behind a local first host is refused.
  - Each helper calls it, with its name and signature unchanged: store
    `newTestStore` (and `writeWalkStore`, which delegates to it), agent
    `newTestStore`, cmd `dbEnv` (both the app and the owner DSN) and cmd
    `openTestStore`. `cli_refresh_real_test.go` now opens through
    `openTestStore` instead of its own `store.Open`.
  - `testlock.AssertRefuses(t, tests...)` is the proof that a guard is there. It
    re-runs the test binary once per named test as a child with strict mode on
    and a `DATABASE_URL` and `DATABASE_OWNER_URL` on an unresolvable `.invalid`
    host (RFC 6761), and passes only when the child fails with the guard's own
    message and without the host. A child that reaches the open (the host
    lookup fails) or that passes fails the parent. The parent tests name one
    probe per helper: store (`newTestStore`, `writeWalkStore`), agent
    (`newTestStore`) and cmd (two real tests that go through `dbEnv` and
    `openTestStore`, plus one probe for the app DSN and one for the owner DSN,
    so deleting either line of `dbEnv` turns the parent red).

## Evidence

The databases were private throwaway clusters; the shared instance was not
touched. Full outputs are kept outside the repository.

**rv4-FR-F129**, the confirms against a stalled database, red then green:

| Subtest | Red (`fd72911`) | Green (`5fc09a2`) |
| --- | --- | --- |
| control: a plain route | pass, 0.31 s | pass, 0.30 s |
| `build/confirm` | fail, the client gave up at 4.00 s | pass, 0.30 s |
| `expand/confirm` | fail, the client gave up at 4.00 s | pass, 0.30 s |

**rv4-FR-F135 and rv4-FR-F140**, red on the unfixed handler (`c472cd3`), green
after `8d58d2a`:

| Case | Red | Green |
| --- | --- | --- |
| 3,334 characters (10,002 bytes) | 400, "refused as too long (limit 10000)" | accepted |
| 10,000 characters (30,000 bytes), the limit | 400, "refused as too long (limit 10000)" | accepted |
| 10,001 characters | 400 | 400 |

**rv4-FR-F137**, a test-strength row. The pins pass on the real code by design,
so no red commit can exist; the red evidence is a mutation of the code under
test, run against the old tests (`stream_caps_test.go` as of `8d58d2a`) and
against the new ones. The recipe is one edit, one run and one restore per row
(BSD `sed`; `-run TestStreamCaps` over the `internal/api` package):

```sh
sed -i '' -E 's/(maxStreamsPerUser +=) 20/\1 49/'        server/internal/api/stream_caps.go
sed -i '' -E 's/(maxStreamsPerProject +=) 100/\1 130/'   server/internal/api/stream_caps.go
sed -i '' -E 's/(maxCursorPostsPerSecond +=) 25/\1 250/' server/internal/api/stream_caps.go
sed -i '' -E 's/return userID.String\(\) \+ ":" \+ client/return userID.String()/' \
  server/internal/api/presence_routes.go
```

| Mutation | Old tests | New tests |
| --- | --- | --- |
| `maxStreamsPerUser = 49` | pass | fail: "one member holds 49 of 150 streams, want exactly the per-user ceiling of 20" and "opened 49 streams before the first refusal" |
| `maxStreamsPerProject = 130` | pass | fail: "the project holds 130 streams, want exactly the per-project ceiling of 100" |
| `maxCursorPostsPerSecond = 250` | pass | fail: "post 251 was the first refused, want post 26" |
| `presenceKey` drops the client id | pass | fail: "one member has 1 presence sessions for 10 open presence streams, want one each" |

The new tests are green on the real code (10 presence and stream-cap tests).

**rv4-FR-F195, rv4-FR-F201 and rv4-FR-F65**, strict mode against an
unresolvable remote DSN, run per package:

| Probe | Red (`65e6339`) | Green (`caef0b6`) |
| --- | --- | --- |
| store `newTestStore` | child reaches the host: "lookup ... no such host" | refused with the guard's message |
| store `writeWalkStore` | the same | refused (by delegation) |
| agent `newTestStore` | the same | refused |
| cmd `TestPromoteAdminCreatesTheFirstAdmin` (`dbEnv`, `openTestStore`) | the same | refused |
| cmd `TestSavedLoginOutlivesItsAccessTokenTwice` | the same | refused |
| cmd owner-DSN probe | passes against the remote DSN: nothing refuses | refused |

Strict mode against the local database is the control: the real tests of the
three packages (including `TestAuthz_*`, `TestRLSWriteWalk_*`,
`TestStageGate_*`, `TestPromoteAdminCreatesTheFirstAdmin` and
`TestSavedLoginOutlivesItsAccessTokenTwice`) still run and pass.

Seven mutations, one guard line removed or bypassed at a time, each caught by
exactly the subtests that should catch it and by nothing else (the last one
by all four packages):

| Mutation | Subtests that went red |
| --- | --- |
| store `newTestStore` loses its guard call | its probe and the `writeWalkStore` probe |
| `writeWalkStore` opens the store itself | the `writeWalkStore` probe only |
| agent `newTestStore` loses its guard call | its probe |
| `dbEnv` loses the app-DSN guard | the app-DSN probe only |
| `dbEnv` loses the owner-DSN guard | the owner-DSN probe only |
| `openTestStore` loses its guard call | the saved-login refresh test only |
| `RequireLocalURL` does nothing | the three remote rows of its own table, the guarded-versus-unguarded test, and every helper probe |

The mirror probe `4f27145` exists because the first cut of the owner-DSN probe
kept the app DSN on this machine, so the app-DSN line in `dbEnv` could be
deleted without any parent turning red (the real promote-admin test is refused
by whichever of the three guards in its path comes first); the mirror probe
keeps the owner DSN local and closes that.

## Known ceilings and what was not taken

- **The guard reads the configured host.** A remote database reached through an
  SSH tunnel or a local proxy on `localhost` looks local to it; no setting of the
  DSN can show otherwise, and `RequireLocalHost` always behaved so.
- **Not every gated test is covered.** The audit, which ran each whole package
  under strict mode with a remote DSN and counted the top-level tests that were
  refused by the guard against the ones that reached the host anyway, found:

  | Package | Refused by the guard | Reached the host |
  | --- | --- | --- |
  | `internal/store` | 110 | 2 |
  | `internal/agent` | 57 | 0 |
  | `cmd/insideout` | 10 | 7 |
  | `internal/api` | 0 | 156 |

  The api package (`newRouteWorld`, `newStreamFixture`, `newGateFixture` and
  per-file preambles) is not covered at all, and neither is this cluster's own
  `TestRvGuardsConfirmRoutesAreBoundedWhenTheDatabaseStalls`, which seeds rows
  through `newGateFixture`. Those sites are outside the files this cluster owns
  and are listed below, which is why F195, F201 and F65 are `PARTIAL`.
- **F201's database-name rule is not added.** Refusing a database by name would
  refuse the working setups: the local and CI workflows both use the default
  database name, so any name a rule demanded would fail them. It is a policy
  choice for the owner, not a check a worker can pick.
- **No grep guard for F140.** A source scan for `len(req.<field>)` against a
  limit would flag every slice length too. The table row pins this field, and a
  new text field joins the table when it joins the API.
- **F137 has no red commit and no mutation harness in the repository.** The
  recipe above is manual, because a harness that edits production constants
  would be a test that can leave the tree broken; the pins themselves are the
  lasting guard.
- **`writeWalkStore` is covered by delegation**, not by a line of its own. The
  probe and the second mutation above pin that: if it ever opens the store
  itself the parent test goes red.

## Left for the integrator

None of these files is owned by this cluster.

- INTEGRATOR TODO: `docs/usage/environment.md:191`: the `INSIDEOUT_TEST_STRICT`
  row says the `DATABASE_URL`-gated Go tests refuse a database on another
  machine; name what is covered (store `newTestStore` and `writeWalkStore`, agent
  `newTestStore`, cmd `dbEnv` and `openTestStore`, and the tests that call
  `testlock.Acquire`, `SignupMode` or `RequireLocalHost`) and say the rest are
  not, until the sites below are guarded.
- INTEGRATOR TODO: `docs/architecture/backend.md:67`: still says "seven exempt
  patterns (... build and expand with their confirms)"; there are five now (the
  two passive streams, the coach message POST, `build`, `expand`) and the two
  confirms run under the 25 s deadline.
- INTEGRATOR TODO: `docs/usage/deployment.md:138-140`: still lists "the four
  roadmap-draft routes (`build`, `build/confirm`, `expand`, `expand/confirm`)"
  as exempt; they are two now (`build`, `expand`).
- INTEGRATOR TODO: `server/internal/store/serve_role_test.go:30`: opens
  `DATABASE_OWNER_URL` with no guard; add `testlock.RequireLocalURL(t, ownerURL)`
  before the open.
- INTEGRATOR TODO: `server/internal/store/circuit_row_id_test.go:32-37`: opens
  the store and only then takes the lock that guards; call `RequireLocalURL`
  before the open (or open through `newTestStore`).
- INTEGRATOR TODO: `server/cmd/insideout/`: add `testlock.RequireLocalURL(t,
  <dsn>)` before the open (or open through `openTestStore`) in
  `cli_idea_locale_test.go:124`, `cli_setup_roles_test.go:352`,
  `contract_real_api_test.go:66` (the open precedes its `SignupMode` guard at
  `:71`), `promote_admin_log_test.go:25`, `seed_test.go:24` and
  `shutdown_test.go:46` (and the owner DSN it reads at `:51`).
- INTEGRATOR TODO: `server/internal/api/`: guard the three shared fixtures
  (`newRouteWorld`, `newStreamFixture`, `newGateFixture`) the way the store,
  agent and cmd helpers now are, and add the api package to an
  `AssertRefuses` parent so deleting the call turns a test red. The audit
  counted 156 api tests that reach the host under strict mode.

## Verification

All Go gates ran against a private throwaway PostgreSQL 18 cluster on loopback
with the roles `insideout_owner` (BYPASSRLS) and `insideout_app` (NOBYPASSRLS);
a skipped `DATABASE_URL`-gated test counted as a failure.

- `(cd server && go build ./... && go vet ./...)`: clean.
- `go test ./internal/api/... ./internal/store/... ./internal/agent/...
  ./internal/testlock/... ./cmd/insideout/... -race -count=1`: all five packages
  `ok` (api 45 s, store 27 s, agent 14 s, testlock 5 s, `cmd/insideout` 260 s).
- `gofmt -l internal cmd`: empty. No Go file changed by this cluster is over 350
  lines (`testlock.go` is the largest, at 324).
- `scripts/tests/check_*.py`, `scripts/check-redaction.sh` and
  `scripts/check-docs-recorded.sh`: see the worker report for the run after this
  commit.
- The full server suite runs once after this commit, as the last gate; its
  result is in the worker report, not here.
- No live server was started: nothing in this cluster changed what a running
  server does beyond the two confirms leaving the exempt list, and those are
  exercised through the real handler over a real database by the F129 test.
