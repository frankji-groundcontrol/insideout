# 23 · Testing

**Lens:** dimension  
**Score:** **6.6 / 10**  
**Bottom line:** The suite is real-DB, honest and mostly deterministic (-shuffle=on -count=2 only fails L1, and the skip gate matches real output), but the lanes have seams. The release is ungated while main is red, the schedule is missing, a global signup_mode race fails under contention, and the CLI/MCP-to-API seam and the refresh/logout path have no automated guard.

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

## Skeptic verdicts on this lens's Highs

| ID | Filed | Verdict | Final | Reason |
|---|---|---|---|---|
| TEST-01 | High | CONFIRMED | High | release.yml is a single job whose steps are checkout, setup-go, 'Build, tar, and checksum the four targets', and 'Publish the GitHub release'. It has no `go test`, no `needs:`, no workflow_run, and no check that ci passed for the tag SHA. Its trigger is `push: tags: ['v*']` (lines 4-5), and a tag push starts it in parallel with … |

---

## Scope

Testing and CI, read at main @ 9f0b1c7 in the clean worktree. I read the Go test tree (185 `_test.go` files), `client/test` (133 files), all `ci.yml` jobs, `smoke.yml`, `release.yml`, `server/scripts/smoke.sh`, `upgrade-replay.sh`, and the guard scripts under `scripts/tests/`. I ran the following on a private PG 18.4 cluster with CI-parity roles:

- the full server suite with `-shuffle=on -count=2 -v` (about 1,800 pass lines);
- per-package and per-handler coverage;
- a stress run of two suspect tests concurrently;
- the five `scripts/tests/check_*.py` guards;
- a live `register`, `refresh`, replay, `logout` walk against a locally built server.

I also scanned every Go test with an AST pass for tests that contain no `t.Fatal`/`t.Error` call. The only hits (`agent_routes_test.go` and `prompt_fence_test.go`) assert through helpers such as `expect` and `assertFenced`, so they are false positives. No network was used outside localhost, and the cluster is stopped.

## Scores

| Aspect | Score | Why |
|---|---|---|
| Real-DB discipline and no-mocks conformance | 8.5 | Every DB-backed test hits real Postgres under the NOBYPASSRLS role. Stand-ins exist only for external services (LLM, GitHub, mail) and clocks. |
| Determinism, ordering, wall-clock | 6.5 | The wall-clock bomb is a single instance (L1). `-shuffle=on -count=2` is otherwise clean. A cross-package race on one global row is real (TEST-02). |
| CI design | 6.5 | The jobs are thorough and the skip gate works. There is no schedule, `release.yml` is ungated, and five guard scripts are unwired. |
| Coverage of the three headline surfaces | 5.5 | The `api` handler package is at 46.5%. The MCP, CLI and `apiclient` layers are tested only against canned servers. |
| Client tests | 7.5 | 458 tests. The route-contract test is good. Transport is faked everywhere and no field-level contract exists. |
| **Overall** | **6.6** | A strong, honest suite with real seams between its lanes. |

## What is good

- **Real database, real roles.** The `ci.yml` server job creates `insideout_owner` (BYPASSRLS) and `insideout_app` (NOBYPASSRLS) and runs `go test ./... -race -count=1 -v`. Tests that need the owner role skip loudly when it is missing (for example `store/migrate_lock_test.go:20`).
- **The skip gate matches real output.** `ci.yml:90-91` greps `--- SKIP: ` with an anchored exception for `TestReadSecretTerminal `. I saw exactly that line in real `-v` output and no other skip in the full run.
- **Roleless lane fails closed.** `cmd/insideout/cli_setup_roles_roleless_cluster_test.go:36` uses a default URL and `t.Fatal`s on any problem rather than skipping. It also refuses a cluster that already has the roles.
- **Careful authz probes.** `store/authored_exmember_test.go` runs bare `UPDATE`/`DELETE` statements through a rolled-back probe with control assertions first, so the zeros it checks can only come from policies.
- **Static route inventories.** `api/routes_require_auth_test.go:37` parses the AST for registered routes. `client/test/api_contract_test.dart` ties client paths to server routes. `scripts/tests/check_docs_truth.py` derives its premises from the tree.
- **The upgrade lane.** `upgrade-replay.sh` plus a pinned old-shape seed (`server/scripts/upgrade-seed.sql`) is a rare, real check that migrations meet production-shaped rows.

## Findings (worst first)

**TEST-01 (High). `release.yml` publishes on any `v*` tag with no test gate, and main is red today.**
`release.yml:5: tags: ['v*']`. The workflow builds, tars, checksums and `gh release create`s without a `go test`, without `needs:` on `ci`, and without checking the tagged commit's status. `TestProjectQuietFlagged` already fails on main. Tagging today would ship an unsigned release from a red tree. Fix: first step of the release job runs `go test ./... -race`, or gate on a successful `ci` run for the tag SHA (about 6 lines).

**TEST-02 (Medium). Two suites race on the global `signup_mode` row. I reproduced the failures.**
- `store/invitations_test.go:39: if mode != `"invitation_only"` {` asserts the global row, then `invitations_test.go:42` flips it to `"open"`.
- `api/auth_email_funnel_test.go:190: if prior != "invitation_only" {` reads and writes the same row from another package.
- `go test ./...` runs packages as parallel processes.
- Running the store test (`-count=150`) beside `TestInvitationRoundTrip` (`-count=60`) failed 10 of 150 and 3 of 60 times. The store test saw `"open"` instead of `invitation_only`; the funnel test saw its invitation never marked used.
- The windows are tiny, which is why 855 CI passes missed it. Under a loaded runner or `-race` it is a real flake, and a restore ordering can leave a shared DB in `open`.
Fix: take a Postgres advisory lock around the global mutation in a shared test helper, or keep each package's signup-mode assertions on its own database or schema.

**TEST-03 (Medium). Handlers behind the headline CRUD are never run by the Go suite, and the session lifecycle is untested over HTTP.**
- Coverage run of `internal/api`: 46.5% overall.
- 0% for `handleListProjects`, `handleCreateProject`, `handleGetProject`, `handleDeleteProject`, the ideas CRUD (`ideas.go:34`-`168`), the project-update handlers (`project_updates.go:60`-`136`), `handleUpdateMemberRole`, `handleExportPrd`, `handleRefresh` (`auth.go:194`) and `handleLogout` (`auth.go:248`).
- Neither `smoke.sh` nor any Go test calls `/auth/refresh` or `/auth/logout`. Only the token extractor is unit-tested (`auth_session_test.go:12`).
- I walked it by hand against a built server: register, refresh 200, replay 401, logout 200, refresh after logout 401. So today it is correct but unguarded.
Fix: add one `routeWorld` test for refresh rotation, replay refusal and logout, plus one projects/ideas happy-path table. Report `go test -cover` in CI as a non-gating number.

**TEST-04 (Medium). MCP (52 tools), CLI and `apiclient` are tested only against canned `httptest` servers, with no route or shape contract.**
- `mcpserver/product_tools_test.go:14: func TestWorkspaceJoinTool` and its siblings hand-write the API reply. 38 of the 52 registered tool names never appear in any test (`agent_context`, `projects`, `roadmap_add`, `prd_update`, `sync` and others).
- Coverage: `mcpserver` 60.5%, `apiclient` 40.7%, `cmd/insideout` 37.1%.
- The Flutter side has a path-level contract test (`client/test/api_contract_test.dart:54: const serverApiDir`). The Go side has none. A rename in `api/*.go` breaks the CLI and MCP with a green build. A crude comparison I ran found no drift today, only the missing guard.
Fix: reuse `registeredRoutes` (`routes_require_auth_test.go:37`) in an `apiclient`/`mcpserver` test that resolves every path literal to a registered route. Add one table test that calls every tool against a stub and asserts it reaches a registered method and path.

**TEST-05 (Medium). Five guard scripts exist and pass but run in no CI job.**
- Never wired: `scripts/tests/check_docs_truth.py`, `check_go_floor.py`, `check_product_promise_gaps.py` and `check_contribution_surface.py`. `check_smoke_wiring.py` is only named in a `smoke.yml` comment.
- I ran all five on a clean tree and all pass, so nothing is red today.
- `check_go_floor.py` is the exact guard for the Go-floor bug (L11, BUG-005). A drift would be invisible.
Fix: add one step to the `env-suites` job in `ci.yml` that runs `for f in scripts/tests/check_*.py; do python3 -I $f; done` (3 lines).

**TEST-06 (Medium). The documented integration command has no guard on its target database.**
- CLAUDE.md and `local-development.md` run the suite through `scripts/dev.sh`, which exports `.env`'s `DATABASE_URL`. The same file says that URL may be a shared instance.
- The suite creates unbounded rows and has no cleanup (`store/authz_test.go:20: url := os.Getenv("DATABASE_URL")`).
- It flips a global setting (`invitations_test.go:42`).
- It assumes a pristine seed (`invitations_test.go:39`). Run against a live deployment that fails, or briefly opens signup there.
Fix: refuse unless `DATABASE_URL` carries an explicit marker, such as a database name matching `*_test` or `INSIDEOUT_TEST_DB=1`, in one shared helper. The 31 copied `t.Skip` preambles can then collapse into it.

**TEST-07 (Low). The upgrade-replay pin is stale.**
`server/scripts/upgrade-replay.sh:33: DEPLOYED_THROUGH=20261006120000`, dated 2026-10-08, with the comment "Bump it when production catches up". HANDOFF records the seven pending migrations as applied on 2026-10-11, so production is now through `20261011140000`. The lane replays seven migrations that production already has and does not rehearse a true next deploy. Fix: bump the constant and extend the seed.

**TEST-08 (Low). `ci.yml` cancels in-progress main runs and builds no client image.**
`ci.yml:10: cancel-in-progress: true` applies to pushes to main too, so a quick follow-up push cancels the previous commit's only verdict. CLAUDE.md lists `docker compose build` as a key command, but `ci.yml` builds only `server/` (`server-image`); `client/Dockerfile` and `client/nginx.conf` are covered only by the python nginx checks. Fix: scope cancel-in-progress to PR refs, and add a `docker build client/` job.

**TEST-09 (Low). The login-timing test is a statistical wall-clock assertion.**
`api/auth_login_timing_test.go` asserts `medUnknown < medKnown/2` on a median of three samples. It is loose and documented, but it runs under `-race` beside parallel packages. If it ever flakes, retry with more samples rather than loosening it.

## Lead verification

| Lead | Result | Evidence |
|---|---|---|
| L1 siblings | **Extended: no sibling bombs.** | Every other fixed date in a `_test.go` is passed in as `now` (`roadmaptime`, `agentcontext`, `auth_cooldown_test.go:26`, `presence`) or is relative to `time.Now()`. `projects.go:84` is the only production `time.Now()` fed by a fixed-date test. Client `DateTime(2026...)` values are static inputs. No shuffle failure beyond L1. |
| L2 | **Confirmed, extended.** | Beyond no schedule: `release.yml` has no gate (TEST-01), and the nightly (`smoke.yml`) runs `smoke.sh`, not `go test`. |
| L3 | **Confirmed.** | My `-shuffle=on -count=2` run shows only L1 failing twice and the `TestReadSecretTerminal` skip twice. |
| L4 | **Not re-run.** | Not my lens. |
| L12 | **Confirmed.** | Budget is 400 in `line_budget_test.go`, against 350 in CLAUDE.md (a policy mismatch, not a test lie). |
| L16 | **Extended.** | Only the server image is built in CI, not the client image (TEST-08). |
| L18 | **Not in scope.** | I found no test or guard for the "never write outside `insideout`" rule in the test tree. |
| Skip gate vs real output | **Confirmed sound.** | Real line is `--- SKIP: TestReadSecretTerminal (0.00s)`. The anchored exception matches it, and a new skip would trip the gate. |

## Recommendations

1. Gate `release.yml` on tests, or on a green `ci` run for the tag SHA (TEST-01).
2. Add a scheduled `ci.yml` run (nightly `go test ./... -race -shuffle=on`) so a time bomb surfaces in a day rather than a week (L1/L2).
3. Serialize the shared `signup_mode` row, for example with an advisory lock in a test helper (TEST-02).
4. Add `check_*.py` to the `env-suites` job (TEST-05).
5. Add the refresh/logout and projects/ideas happy-path HTTP tests, and publish `go test -cover` (TEST-03).
6. Add a Go route-contract test for `apiclient` and MCP using `registeredRoutes`, plus a one-call-per-tool table (TEST-04).
7. Add a test-database marker guard in one shared helper (TEST-06).
8. Bump `DEPLOYED_THROUGH`, scope `cancel-in-progress`, add a client image build (TEST-07/08).

## What this role would do first

Fix the release gate and the schedule, because those two changes decide whether the next regression ships silently. One workflow edit adds `go test` before the publish step, and a `schedule:` trigger with `-shuffle=on` on `ci.yml` would have exposed L1 on the day the clock crossed. Then I would close the cross-package `signup_mode` race with a small advisory-lock helper, since it is the only real flake I could reproduce. Last, I would add the missing seams: the refresh/logout HTTP test and the Go-side route-contract test for the CLI and MCP.

## The three tests that would have caught real bugs

1. `TestProjectQuietFlagged` (`api/projects_test.go:46`). It already exists and fails on main. It would have caught the pinned-clock bug if CI ever ran on a clock rather than only on push.
2. A concurrent `signup_mode` test (a stress pair like the one I ran). It would have caught the cross-package race before a runner did.
3. A Go route-contract test for `apiclient` and the MCP tools, together with an HTTP refresh/logout test. Today the CLI/MCP-to-API seam and the session lifecycle have no automated guard.
