# Wave 3, test hardening — review-4 remediation

Branch `rv4/w3-test-hardening`, base `6383b87` (the cluster base with the
`w3-context-deadlines` and `w3-auth-network` work already merged), eleven
commits `0f86405..e1d492c`: four red/fix pairs (TEST-02, TEST-06, DATA-07,
CLOSE-08), two test-only commits (TEST-03, TEST-04) and one comment commit
(TEST-07), then this fragment. Row 3.7 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md)
(a design row). Findings are cited as `rv4-<ID>` because review IDs restart per
review. No host, address, secret or token is spelled here; every database below
was a private throwaway cluster on loopback, and its scratch databases were
dropped.

## Design spike (row 3.7 is `dir`)

Five choices were settled before any code, each against the obvious
alternative. **The order comes from the database.** `go test ./...` starts every
package's test binary as its own process, so nothing inside one (`t.Parallel`, a
mutex) can order two tests of different packages; what they share is the
database, so it orders them: a session-level advisory lock held on a connection
taken out of the pool, the way the server's single-replica guard holds its own.
`go test -p 1` lost (it serializes the whole suite for two global rows and does
nothing for a developer running two packages by hand); a transaction lock lost
(the store runs its statements on the pool's own connections, so no transaction
spans a test); an injectable seam for the row in production code lost (it
changes the product for a test). **One lock per global row, not one big
lock**: `signup_mode` and `breaker`, so unrelated tests still run in parallel,
under a two-int key that cannot collide with the server's own bigint keys.
**A lock only counts if skipping it is caught.** `testlock.AssertWaits` holds
the lock itself, starts the test binary again to run one named test, and
requires the child to park on the lock (seen in `pg_locks` by a per-child
application name) instead of running to the end. That is each lock's red test:
the unfixed tests finish at once. The test names in each `AssertWaits` call are
the registry a new test that touches the row is added to. **Strict mode is
configuration only.** `INSIDEOUT_TEST_STRICT` makes a pool whose configured host
is not `localhost`, a loopback address or a unix socket fail the test; it never
connects and never prints the host. It is opt-in, so CI and the shared-instance
workflows behave as before. **Indexes come from the catalog.** The migration
finds every foreign key of schema `insideout` without a supporting index and
indexes it, and the test runs the same question, so a key a later migration
adds without an index fails the suite instead of waiting for a review.
**The contract test uses the real server and nothing else**: the API's own
handler on a loopback listener over the real database, the real clients pointed
at it, no route re-implemented.

## Dispositions

| ID | Status | Red | Fix | Note |
| --- | --- | --- | --- | --- |
| rv4-TEST-02 | `FIXED` | `0f86405` | `8c05a9f` | the `signup_mode` touchers take a cross-process lock; new `internal/testlock` |
| rv4-TEST-03 | `FIXED` | measured, 63.0% | `d101b88` | `internal/api` 63.0% to 72.3%; found a defect not in the review (below) |
| rv4-TEST-04 | `PARTIAL` | six mutations | `1fae42a` | 35 of 52 tools and 18 CLI verbs against the real API; no static route-literal check yet |
| rv4-TEST-06 | `PARTIAL` | `e52cc45` | `9589e6a` | the helper and strict mode only; no `dev.sh` flag, the 31 preambles are untouched |
| rv4-DATA-07 | `FIXED` | `ff42053` | `30f644b` | 18 of 41 foreign keys had no index, now 0 |
| rv4-CLOSE-08 | `PARTIAL` | `844386e` | `f6048ab` | serialized with a lock; a row of its own per test needs a store change |
| rv4-TEST-07 | `DEFERRED_OWNER` | | `e1d492c` (comment) | which migration the hosted database has applied |
| rv4-CLOSE-09 | `DEFERRED_OWNER` | | `e1d492c` (comment) | the same fact |

## What changed

- **Tests that flip or read `signup_mode` wait for one another across
  processes** (rv4-TEST-02, red `0f86405`, fix `8c05a9f`). Store's
  `TestSignupModeSetting` sets the row to `open` while the API's invitation and
  agent-token tests need `invitation_only`; they live in different packages, so
  `go test ./...` ran them at the same time. The new package
  `server/internal/testlock` has `Acquire`, `SignupMode`, `AssertWaits` and the
  `SignupModeLock` and `BreakerLock` names. `SignupMode(t, pool)` is called
  before a test reads or saves the row, so the restore the test registers runs
  while the lock is still held. Five call sites take it: store's
  `TestSignupModeSetting`, and the API's admin read/write test, agent-token test,
  auth-config test and the mail world of the invitation funnel (which serves
  `TestInvitationRoundTrip`). Two `...WaitForTheCrossProcessLock` tests pin those
  five tests by name. The API's write test now writes back the mode it already
  has instead of flipping it, still proven by `updated_at` moving, so a crash
  leaves the mode alone.
  The `apiclient`, `mcpserver` and `cmd/insideout` tests that mention the mode
  (`auth_test.go`, `escape_test.go`, `admin_tools_test.go`,
  `agent_profile_helpers_test.go`, `cli_setup_test.go`) talk to `httptest`
  stubs, never to the row, so they are unchanged.
- **`INSIDEOUT_TEST_STRICT` refuses a database that is not on this machine**
  (rv4-TEST-06, red `e52cc45`, fix `9589e6a`). `testlock.RequireLocalHost(t,
  pool)` reads the pool's configuration, so it never connects, and the failure
  leaves the host out so a log never carries it. `Acquire`, the new coverage
  tests and the contract test call it. Empty, `0`, `false`, `no` and `off` mean
  off; anything else means on. The red commit is a compile failure (the helper
  did not exist), which is the honest red for a new helper.
- **Every foreign key of schema `insideout` has a supporting index**
  (rv4-DATA-07, red `ff42053`, fix `30f644b`). PostgreSQL indexes the
  referenced side of a foreign key and never the referencing side, so deleting a
  workspace scanned `ai_runs` and `agent_conversations` once per cascaded row.
  `20261018210000_index_foreign_keys.sql` indexes the 18 keys that had none
  (indexes 57 to 75) with `CREATE INDEX IF NOT EXISTS`, found in the catalog
  and named `table_column_idx`; it adds no constraint, so no data can make it
  fail, and a second run finds nothing to do. `fk_index_test.go` lists the
  unindexed keys from the catalog after migrate (the red names the 18) and pins
  the columns the review called out.
- **The breaker-row tests wait for one another across processes** (rv4-CLOSE-08,
  red `844386e`, fix `f6048ab`). The agent package's wiring tests open and fill
  the one production `anthropic` breaker row while store's
  `TestCircuitFunctions_TakeARowID`, in a sibling process, asserts it is not
  open. `resetBreaker` and that store test now take `BreakerLock`; the lock is
  taken first so the restore runs while it is still held. This is a serialization
  and not the cure the finding asks for; see the ceilings.
- **Never-executed handlers now run against the real database** (rv4-TEST-03,
  `d101b88`). Nine new files, `coverage_*_test.go` (each under 350 lines) over
  the `routeWorld`: the session lifecycle over HTTP (logout, refresh by cookie or
  body, protected-route credentials, `healthz` including a closed pool),
  workspaces (create, join once, edit, delete by the creator alone), members,
  ideas (inbox, create, edit, drop, convert), projects and project updates, PRDs
  (read, save, revisions, status lifecycle, conversation) and the export
  formats. Each test asserts the role split the handler documents: what a
  member, an admin, the creator and an outsider get, with RLS-hidden rows
  answering 404.
- **One test drives the CLI and the MCP tools through the real API** (rv4-TEST-04,
  `1fae42a`; `contract_real_api_test.go` and `contract_real_api_mcp_test.go`
  in `cmd/insideout`). It mounts `api.NewServer(...).Handler()` on a loopback
  listener over the real database, with no coach and no planner (the LLM seams
  are not under test), and signs one user in with `apiclient.Login`, saved the
  way `insideout login` saves it. Phase 1 runs 14 CLI verbs as child processes
  (the test binary re-executed as `main`), each reading that login; phase 2
  calls MCP tools in-process through the SDK's JSON-RPC (reads, writes, the
  agent family on a token traded at `POST /agent/token`, the draft family
  including `build_confirm` and `expand_confirm`), each reading what the CLI
  wrote and writing what the CLI then reads; phase 3 runs an MCP process that
  signs itself in (`auth_login`, `whoami`, `signup_mode`, `auth_logout`); phase
  4 runs the admin verbs (`invite`, `invites`, `invite revoke`) as a system
  admin promoted through the owner pool. In all it calls 35 of the 52 MCP tools
  and 18 CLI verbs. A proposal made by the agent tool is accepted and applied by
  the CLI, which is the one place both halves of that flow meet.

## Measured

TEST-02 and CLOSE-08, four concurrent test processes against one database:

| Run | Base | After |
| --- | --- | --- |
| store `TestSignupModeSetting` x150 beside the API tests | 41 of 150 failed | 0 |
| API `TestInvitationRoundTrip` x60, same run | 2 of 60 failed | 0 |
| API `TestAnAgentTokenCannotReachTheAdminRoutes` x60, a second base run | 2 of 60 failed ("the agent token wrote the signup mode") | 0 |
| store `TestCircuitFunctions_TakeARowID` x400 beside the agent tests x400 | 67 of 400 failed | 0 |

At the base the race also left the shared row on `open`, after which the store
test failed every run until it was reset by hand. After the fix the row ends as
it began.

TEST-03, `go test ./internal/api/... -coverprofile`, total statements:

| | Before | After |
| --- | --- | --- |
| `internal/api` | 63.0% (the review measured 46.5% on an older tree) | 72.3% |

The movement by handler: `handleLogout`, `handleHealthz` and
`projectUpdateResponse` 0% to 100%; `handleExportPrd` 20% to 91%;
`handleEditProjectUpdate` 29% to 88%; `handleAddProjectUpdate` 26% to 82%;
`handleDeleteProjectUpdate` 31% to 81%; `handleUpdateMemberRole` 35% to 89%;
`handleRemoveMember` 37% to 84%; `handleListMembers` 17% to 65%;
`handleJoinWorkspace` 19% to 77%; `handleListIdeas` 20% to 70%;
`handleListRevisions` 20% to 70%; `handleUpdatePrdStatus` 46% to 89%;
`handleConvertIdea` 48% to 87%; `handleUpdateProject` 63% to 97%. Hand-run
mutation check, never committed: 17 deliberate production mutants, 15 caught.
The two survivors are equivalent, not holes: the store-level guards for dropping
an idea and deleting a workspace are shadowed by RLS, which answers first.

TEST-04, six one-line production mutations, each run against the existing
client suites (`apiclient`, `mcpserver` and the other `cmd/insideout` tests) and
against the new test:

| Mutation | Existing client suites | The contract test |
| --- | --- | --- |
| `GET /me` renamed to `/profile` | red only through the old real-API refresh test that already calls `/me` | red: `whoami` gets 405 |
| the workspace JSON field `myRole` renamed | green | red: `workspaces create` has no role |
| `POST /agent/checkpoint` renamed | green | red: the tool gets 404 |
| `POST .../build/confirm` renamed | green | red: `build_confirm` gets 404 |
| readiness marked human-only in the agent policy | green | red: the tool gets 403 `agent_scope_denied` |
| `signupMode` renamed in `/auth/config` | green | red: the tool returns the empty string |

Five of six mutations leave every Go suite but the new one green, which is the
finding. The contract test alone covers 50.7% of `apiclient`, 69.6% of
`mcpserver` and 26.8% of `internal/api`, in process; the CLI child processes
are not counted. It passes with `-race` (21 s; 0.45 s without) and with
`-race -count=3`.

TEST-07 and CLOSE-09, the upgrade lane run on a scratch database: at the current
cutoff it is green (31 of 42 migrations replayed, 11 pending ones meet the
seeded data, this branch's index migration among them). With the cutoff bumped
to `20261011140000`, the production state `docs/HANDOFF.md` records, it fails:
`no workspace keeps a sub-32 invite code ... got '1', want '0'`, because the seed
and that assertion are aimed at `20261010110000`, which would then be behind the
cutoff.

Evidence files, one red and one green per row, are under
`/private/tmp/insideout-rv4/reports/w3-test-hardening-evidence/`.

## Found on the way

- **A project owner who is not a workspace admin cannot hand the project over,
  and the API answers 500.** Not in the review's list; found by TEST-03. `PATCH
  /api/v1/projects/{pid}` lets the owner edit, but when the edit changes or
  clears `ownerId` the database refuses the new row (`new row violates row-level
  security policy for table "projects"`, SQLSTATE 42501), because
  `projects_update` has a `USING` clause and no `WITH CHECK`, and the handler
  answers `500 {"error":"internal error"}`. Pinned, not fixed, by
  `TestProjectOwnerWhoIsNotAnAdminCannotGiveTheProjectAway_KnownDefect`
  (whoever fixes it must revisit that test). Fix, outside this cluster's files:
  a new migration adding `WITH CHECK` to `projects_update` (a product decision:
  may an owner give a project away?), or `ErrForbidden` from
  `store.UpdateProject`.
- **Observation only:** `RemoveMember` and `UpdateMemberRole` have no last-admin
  guard, so a workspace can be left with no admin. The new member tests do not
  pin it.

## Known ceilings, recorded on purpose

- **CLOSE-08 is serialized, not cured.** The finding asks each test to have a
  breaker row of its own. `Store.CheckCircuit` and `Store.RecordCircuitResult`
  (`internal/store/agent_messages.go`, around lines 315 to 327) are hardwired to
  the default row, and the `Coach` (`internal/agent/coach.go`, holding the
  concrete `*store.Store`; `gate.go:32` calls them) reaches the breaker only
  through them. The SQL functions already take a row id (migration
  `20261011100000`); the two Store methods need to pass one. Until then the
  readers of the shared row outside this
  cluster's files take no lock: API tests that build a real `agent.New(...)`
  coach, and the agent package's `gate_test.go`, `terminal_errors_test.go` and
  `coach_deadline_test.go`. `coach_fail_test.go`, owned here, needed no edit.
- **TEST-06 is the helper, not the guard everywhere.** Nothing sets
  `INSIDEOUT_TEST_STRICT` yet (`scripts/dev.sh` is not owned), and the 31 copied
  `DATABASE_URL` skip preambles still stand, so a test that does not call
  `Acquire` or `RequireLocalHost` still runs against any database. The check is
  a host check, not the marker the review suggested (a `*_test` database name):
  a hostname that resolves to loopback is refused, because the check never does
  a DNS lookup.
- **A killed test binary leaves its flip behind.** The advisory lock frees when
  the session ends, but a row a test changed is restored in `t.Cleanup`, which a
  SIGKILL or a timeout panic skips. The next `TestSignupModeSetting` then fails
  its "seeded `invitation_only`" assertion until the row is reset by hand.
- **TEST-04 is a contract test, not the route-literal check.** 17 of the 52 MCP
  tools are not called. Ten are withheld from a signed-in agent process by
  design (`commit`, `proposal_decide`, `idea_delete`, `roadmap_delete`,
  `invite_create`, `invite_list`, `invite_revoke`, `auth_register`,
  `auth_forgot_password`, `auth_reset_password`; the CLI half covers the
  invitations and the decision on a proposal). Four need delivered mail
  (`auth_login_code_request`, `auth_login_code_verify`, `auth_magic_link_request`,
  `auth_magic_link_consume`). Two bind or pull a GitHub repository (`repo_set`,
  `sync`), and `workspace_join` needs a second account. The review's static
  check, that every path literal in `apiclient` resolves to a registered route,
  is also not written. No drift beyond the six mutations was found.
- **TEST-03 left some code below 50%**: the GitHub sync and webhook paths, the
  project guide, the conversation message list and the roadmap-draft loader.
  They need the provider, the GitHub seam or a stream. The review's last
  suggestion, to publish `go test -cover` in CI as a non-gating number, is a CI
  change and not made here.
- **TEST-07 and CLOSE-09 wait on an owner fact.** `DEPLOYED_THROUGH` stays at
  `20261006120000`; `scripts/tests/check_upgrade_cutoff.py` is not written.

## Owner question

Which migration has the hosted database actually applied? `docs/HANDOFF.md`
records the deploy of 2026-10-11 as migrated through `20261011140000`, but only
the database knows: run `SELECT max(filename) FROM insideout.schema_migrations`
against it (the filename must exist under `server/db/migrations/` and sit behind
the newest file there, or the lane has nothing to replay). With the answer,
three edits follow in one change: set `DEPLOYED_THROUGH`; re-aim
`server/scripts/upgrade-seed.sql` (not owned here) and the assertions in
`upgrade-replay.sh` at the migrations after the cutoff, so the lane still has an
old shape to meet; and add the cutoff to the release checklist (CLOSE-09), with
`scripts/tests/check_upgrade_cutoff.py` guarding that it stays behind HEAD.

## Left for the integrator

INTEGRATOR TODO (pre-gate): `docs/architecture/backend.md`: add the Layout line
for `server/internal/testlock` (cross-process advisory locks and the strict-mode
host guard that the `DATABASE_URL`-gated tests share; imported by tests only).
`scripts/tests/check_docs_truth.py` prints "backend.md's Layout tree omits
internal/testlock/" until it lands; it is the only gate line that fails here.

INTEGRATOR TODO: `docs/usage/environment.md`: `INSIDEOUT_TEST_STRICT | no |
unset | anything but 0, false, no or off makes the DATABASE_URL-gated tests
refuse a database that is not on this machine (localhost, a loopback address or
a unix socket); read by internal/testlock; CI leaves it unset`.

INTEGRATOR TODO: `docs/practices/` (the real-database testing page, and its
README line if a page is added): a test that reads or writes a global row takes
the named lock (`testlock.SignupMode`, `testlock.Acquire(t, pool,
testlock.BreakerLock)`) first, and is added to that package's `AssertWaits`
list.

INTEGRATOR TODO: `docs/issues/`: file the defects and residues above, one entry
each with its fix hook: the `projects_update` missing `WITH CHECK` (500 on an
owner's hand-over), the missing last-admin guard, the CLOSE-08 breaker row id,
the TEST-06 `dev.sh` flag and preambles, the TEST-04 residue (17 tools, the
route-literal check), the non-gating coverage number in CI. The issue-book index
lines are the integrator's.

INTEGRATOR TODO: indexes: the `w3-test-hardening` row in this folder's
`index.md`; `docs/changelogs/README.md` and `docs/changelogs/index/2026-10.md`;
the open-findings record and `docs/issues/README.md` with the dispositions above
(TEST-02, DATA-07 `FIXED`; TEST-03 `FIXED`; TEST-04, TEST-06, CLOSE-08
`PARTIAL`; TEST-07, CLOSE-09 `DEFERRED_OWNER`, with the owner question above).

INTEGRATOR TODO: keep `m.mail.Wait()` in `lastMail`
(`server/internal/api/auth_email_funnel_test.go`) when merging; the
`w3-auth-network` cluster added it and this cluster edited the same file around
it (the file merged cleanly on this branch).

## Verification

All database-backed runs used a private PostgreSQL cluster on loopback, torn
down at the end; a skipped `DATABASE_URL` test counts as red. Nothing was
pushed, merged or deployed, and no server was started (the in-process loopback
listeners of the tests only).

| Gate | Result |
| --- | --- |
| `gofmt -l internal cmd` | empty |
| `go build ./... && go vet ./...` | clean |
| `go test ./... -race -count=1`, the whole suite, once, last | every package ok; `cmd/insideout` 251 s, `internal/store` 22 s, `internal/api` 15 s, `internal/agent` 8 s, `internal/testlock` 2.6 s |
| `go test ./internal/store/... ./internal/api/... -count=3 -p 4` | ok (store 55 s, api 14 s) |
| `insideout migrate`, again on the migrated database | `nothing to do, already up to date`, exit 0, three times; the first application of the index migration was checked on a fresh chain and in the upgrade lane (31 replayed, 11 pending met the seeded data) |
| `go test ./internal/api/... -coverprofile`, `go tool cover -func` | 63.0% before, 72.3% after |
| files over 350 lines, `.go` and `.dart` changed since the merge base with `rv4/integration` | none (67 files) |
| `scripts/tests/check_*.py` | eight pass; `check_docs_truth.py` prints only `backend.md's Layout tree omits internal/testlock/`, the pre-gate item above |
| `scripts/check-redaction.sh`, `scripts/check-docs-recorded.sh` | silent, exit 0 |
