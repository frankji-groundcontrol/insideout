# Review 4 open findings — the ones that block a stranger or a release

Open, recorded 2026-10-18. [Multirole review 4](../review/2026-10-18-multirole-4/README.md)
read `main` @ `9f0b1c7` and filed 145 findings (0 Critical, 10 High, 88 Medium,
47 Low; nine Highs skeptic-confirmed, one added by the critic). The
[register](../review/2026-10-18-multirole-4/01-findings-register.md) holds every
row and the
[remediation plan](../review/2026-10-18-multirole-4/40-remediation-plan.md)
is the ordered fix list; this record tracks the nine Highs, the red test on
`main`, and the owner decisions, each linked to its register ID and plan row.

Review IDs restart per review, so cite these as `rv4-USER-01` outside the
review folder (review 3 also has a `USER-01`).

Status 2026-10-19: waves 0 to 2 of the plan (rows 0.1 to 2.6) are integrated
on the local branch `rv4/integration` (base `d53035a`; wave 0 head `97d7529`,
wave 1 integration head `a079404`, wave 2 integration head `e6bc637`, then the
records commits; nothing pushed) and recorded in the
[remediation changelog](../changelogs/2026-10-18-rv4-remediation/index.md).
All ten Highs have a fix on the branch: wave 1 closed three (rv4-USER-01,
rv4-TEST-01, rv4-COMP-02) and wave 2 the other seven; the web-feed half of
rv4-FEAT-02 stays `PARTIAL` (R2-01). Wave 3 has not started. The Disposition
column is filled by each wave's integrator with `FIXED` (red and fix commits),
`NOT_REAL`, `DEFERRED_OWNER`, `PARTIAL` or `BLOCKED`, plus the reason.

## Highs and the red test (priority order)

| Register ID | Sev | Priority | Owner | Plan row | Done when | Disposition |
|---|---|---|---|---|---|---|
| rv4-USER-01 | High | P0 | engineering | 1.1 | On a fresh invitation-only database, `promote-admin` creates or promotes the first admin and admin settings answer 200; the red test is in the suite. | `FIXED` (wave 1): red `f30001c`, fix `c913e8e`. |
| rv4-TEST-01 | High | P0 | engineering | 1.5 | `release.yml` runs `go test ./... -race` (or requires a green `ci` run for the tag SHA) before it publishes. | `FIXED` (wave 1): red `0baafaf`, fix `b7db11d`. The release runner has no PostgreSQL service (R1-05) and the step is pending first CI run after push (R1-17). rv4-PM-05 stays `PARTIAL` (Owner decisions). |
| rv4-L1 (ARCH-04, QUAL-05, PM-02) | Medium | P0 | engineering | 1.3, 1.4 | `TestProjectQuietFlagged` passes on any date; `ci.yml` has a weekly schedule; HANDOFF cites a dated SHA. | `FIXED` (wave 1): L1 red `01d64c3`, fix `0a6b5c3`; ARCH-04 red `14dc369`, fix `6172113`; QUAL-05 red `524716a`, fix `6172113`; PM-02 red `8b9cd0b`, fix `57fa71e`, and the dated SHA (`9f0b1c7`) in HANDOFF since `7091e14`. The schedule is pending first CI run (R1-17). The wave-0 baseline failure is gone (gate note below). |
| rv4-FULL-02, rv4-USER-04 | High | P1 | engineering | 2.2 | A CLI login older than the access TTL still runs `whoami`; plain `login` persists its token. | `FIXED` (wave 2): FULL-02 red `ef1af83`, fix `96748fc`; USER-04 red `37da942`, `a5a790f` (the concurrent-renewal race), fix `12b6a00`, `90bab89`. The renewal wait is a poll, not a lock: R2-12. |
| rv4-ARCH-01 | High | P1 | engineering | 2.1 | Each verb honours `--api` over config; the MCP binary reads the config `api_base`; one resolver remains. | `FIXED` (wave 2): red `38aafed`, fix `c5af0e9`, chore `8899c73`. `release.yml` does not pass the `-X` link flag yet: R2-07. |
| rv4-FEAT-01, rv4-AIDEV-04 | High | P1 | engineering | 2.5 | An agent-scoped token is refused at `commit` and `proposal_decide` server-side; the MCP profile omits `signup_mode`, `invite_*` and deletes. | `FIXED` (wave 2): FEAT-01 red `62e68f4`, fix `3a22813`, design `9b3cdb9`; AIDEV-04 red `a308076`, `c4087ec`, fix `5e4968e`, `8def9d6`. `DEFERRED_OWNER`: whether an agent token may ever decide a proposal (Owner decisions); the strict default shipped. |
| rv4-FEAT-02 | High | P1 | engineering | 2.3 | Accepting a proposal with apply twice leaves one set of roadmap nodes; apply errors surface as 500. | `FIXED` (wave 2, server half): red `f624efb`, fix `a512e0a`, guard `70d3136`. The web feed still re-creates each accepted node after the server did: FEAT-02-CLIENT-APPLY, `PARTIAL` (R2-01). |
| rv4-FULL-01 | High | P1 | engineering | 2.4 | Web Expand with AI creates children; a real-shape contract test pins the response. | `FIXED` (wave 2): red `2046fb5`, fix `29c0956`. The expand route takes no position lock and gates on membership only: R2-04. |
| rv4-COMP-02, rv4-COMP-01 | High, Medium | P1 | engineering | 1.16, 1.17 | A deadline picked as 2026-10-18 still reads 2026-10-18 under `TZ=Asia/Shanghai`; a 67-character Chinese title is accepted on every field that has a character limit. | `FIXED` (wave 1): COMP-02 red `5a47205`, fix `137d6f5` (CI runs in UTC; the Shanghai-zone step is R1-01); COMP-01 red `0129059`, fix `e129f02`. |

## Wave 1 dispositions (integrator, 2026-10-18)

One row for each of the 34 findings wave 1 touched: 30 `FIXED`, 4 `PARTIAL`
(rv4-PM-05, rv4-TEST-08, rv4-SEC-06, rv4-USER-09), no `NOT_REAL`, no `BLOCKED`;
the owner halves of rv4-PM-05 and rv4-COMP-02 are `DEFERRED_OWNER`. SHAs are on
`rv4/integration` (the cluster fragments cite the worker-branch SHAs). A
CI-config property can only be checked locally here (`actionlint` is not
installed), so those rows end with "pending first CI run after push" (R1-17). Run
state and the clusters are in the
[wave-1 record](../changelogs/2026-10-18-rv4-remediation/wave-1.md);
what each cluster left open is the residue table below.

| Register ID | Row | Disposition | Red | Fix | Reason or what is left |
|---|---|---|---|---|---|
| rv4-USER-01 | 1.1 | `FIXED` | `f30001c` | `c913e8e` | `promote-admin` creates the account when it is absent (password from stdin, never argv). The password minimum counts bytes: R1-11. |
| rv4-USER-02 | 1.2 | `FIXED` | `e0908b3` | `abc19a0` | `help`, `-h`, `--help`, `version`, `--version` exit 0 and an unknown verb exits 2, all before `config.Load()`. |
| rv4-FULL-03 | 1.2 | `FIXED` | `e0908b3` | `abc19a0` | Same commits as USER-02 (the verb table and its drift test). |
| rv4-PM-04 | 1.2 | `FIXED` | `e0908b3`, `0baafaf` | `abc19a0`, `b7db11d` | `insideout version` prints the tag that `release.yml` stamps into `main.version`. The container image and `scripts/install.sh` print `dev`: R1-04. |
| rv4-TEST-01 | 1.5 | `FIXED` | `0baafaf` | `b7db11d` | `release.yml` runs `go test ./... -race -count=1` before any build or upload. The release runner has no PostgreSQL service: R1-05. Pending first CI run: R1-17. |
| rv4-PM-05 | 1.5 | `PARTIAL` | `0baafaf` | `b7db11d` | The test gate is done with TEST-01 and [releasing](../usage/releasing.md) records the process (`7091e14`). `DEFERRED_OWNER`: the version policy, the rollback rule, a stronger gate (R1-05, R1-16). |
| rv4-PM-02 | 1.4 | `FIXED` | `8b9cd0b` | `57fa71e`, `5f52381` | A weekly `schedule` runs `go test ./... -race -shuffle=on`; HANDOFF cites `9f0b1c7` (2026-10-11) since `7091e14`, which closed the clause the worker had reported `PARTIAL`. Pending first CI run: R1-17. |
| rv4-TEST-08 | 1.4 | `PARTIAL` | `735ba99`, `4609a63` | `adf2177`, `cc534e7` | In-progress runs are cancelled for pull requests only; every other run has a group of its own. The client image job did not fit (`ci.yml` 346 of 350 lines): R1-01, [ci-yml-split](2026-10-18-ci-yml-split.md). |
| rv4-PM-01 | 1.6 | `FIXED` | `9112dc6` | `352dbc2`, `b410302` | The `env-suites` job runs every `scripts/tests/check_*.py` with `python3 -I`. Pending first CI run: R1-17. |
| rv4-CLOSE-01 | 1.6 | `FIXED` | `9112dc6` | `352dbc2`, `b410302` | Same step as PM-01 (the guard loop). |
| rv4-TEST-05 | 1.6 | `FIXED` | `9112dc6` | `352dbc2`, `b410302` | Same step as PM-01 (the guard loop). |
| rv4-PUB-08 | 1.6 | `FIXED` | `9112dc6` | `352dbc2`, `b410302` | Same step as PM-01 (the guard loop). |
| rv4-PM-12 | 1.11 | `FIXED` | `1f3c74a` | `b2aa936` | The Ubuntu install lane's Go check accepts 1.26 and newer, as `go.mod` does. Pending first CI run: R1-17. |
| rv4-PM-07 | 1.11 | `FIXED` | `3ad4022` | `acf57c0` | The register's "0 analyzer infos" does not reproduce (14 measured, three independent measurements). The real defect was a ratchet that allowed 16; `INFO_BASELINE` is now 14. |
| rv4-PM-10 | 1.15 | `FIXED` | `c47e3ec`, `622bf58`, `4c55a55` | `5b642bd`, `7951036`, `3996134` | `huh` is direct in `go.mod`, `go.sum` is tidy and the server job fails on an untidy module (`ci-gates` reported its half `PARTIAL` until `misc-correctness` tidied `go.sum`). The step is pending first CI run: R1-17. |
| rv4-L1 | 1.3 | `FIXED` | `01d64c3` | `0a6b5c3` | The project views read `Server.now()`, so `TestProjectQuietFlagged` passes on any date. `flagged` on detail responses is judged from `createdAt` alone: R1-10. |
| rv4-ARCH-04 | 1.3 | `FIXED` | `14dc369` | `6172113` | A guard fails on a raw `time.Now()` in the `internal/api` view builders (the clock itself is `0a6b5c3`). |
| rv4-QUAL-05 | 1.3 | `FIXED` | `524716a` | `6172113` | The roadmap node and progress views read the server clock. |
| rv4-COMP-01 | 1.17 | `FIXED` | `0129059` | `e129f02` | One `tooLong` helper counts characters on the 18 capped fields; 67 Chinese characters pass. Byte caps outside the sweep (the coach message, `firstLine`): R1-12. |
| rv4-SEC-01 | 1.9 | `FIXED` | `b4edce1` | `4df79d6` | A login-code guess is charged and compared in one atomic `UPDATE`, so a concurrent burst is capped at 5. |
| rv4-USER-07 | 1.10 | `FIXED` | `bee99f7` | `a7f9519` | Create-invitation returns `mailSent` and the dialog says when nothing was e-mailed. The `invite_create` MCP description names it since wave 2 (R1-03, `f9e9894`). |
| rv4-QUAL-02 | 1.13 | `FIXED` | `8765fed` | `5cf5bc2` | The resend cooldown keys on `clientIP`, not a drifted copy, so a trailing-empty `X-Forwarded-For` no longer puts every client in one bucket. |
| rv4-QUAL-03 | 1.13 | `FIXED` | `e45229e` | `868e49f` | Every apiclient path segment and query value is escaped (39 rows under a completeness test). `Client.Cursor` passes a nil reply target: R1-13. |
| rv4-SEC-06 | 1.14 | `PARTIAL` | `e646d8e` | `cbd9fe8` | The code half is fixed: dot-only repo segments and foreign hosts are refused and the canonical URL is stored. The token half (a per-installation token, a case-insensitive lookup with a canonicalising migration, `FetchGuideFile` dot names) goes to `w3-github-evidence`, row 3.10: R1-08. |
| rv4-USER-11 | 1.12 | `FIXED` | `b17641a` | `bed39fb` | The Edit-project description field is labelled as a description. |
| rv4-COMP-02 | 1.16 | `FIXED` | `5a47205` | `137d6f5` | The picked deadline is sent as noon UTC of its calendar day, so 2026-10-18 reads 2026-10-18 under `TZ=Asia/Shanghai`. CI runs in UTC and cannot see the bug until the Shanghai-zone step lands: R1-01. Date-typed deadlines and viewer-zone timestamps are `DEFERRED_OWNER`: R1-15. |
| rv4-USER-06 | 1.7 | `FIXED` | `a875428` | `7b6f4c0` | The README `prd update` example puts the id after the flags; root `SKILL.md` was corrected in `7091e14`. The guard does not scan `SKILL.md`: R1-02. |
| rv4-REL-05 | 1.8 | `FIXED` | `092f7f1` | `3a8defb` | The compose `server` has `stop_grace_period: 35s`, explained in the deployment architecture. `docker compose config -q` has not run anywhere: R1-17. |
| rv4-USER-09 | 1.12 | `PARTIAL` | `8161369` | `6f32133` | The installer's closing message no longer offers `setup --local`. `README.md:86` (R1-06) and the `setup --hosted` hint at `cli_setup.go:148` (R1-07) still do; both are wave 3. |
| rv4-COMP-07 | 1.18 | `FIXED` | `ef63477` | `074d4e0` | The backup dump is written 0600 (umask and chmod) and the usage doc says what it holds. `sh scripts/backup-db.sh` fails under dash: R1-14. |
| rv4-AIDEV-10 | 1.15 | `FIXED` | `4d04a6d` | `9da2086` | The `propose` tool schema declares `items`. |
| rv4-QUAL-01 | 1.15 | `FIXED` | `9deac14` | `d40c64b` | One blank-section helper serves readiness and the coach context; the `openQuestions` order is stable. |
| rv4-FULL-06 | 1.15 | `FIXED` | `4fe72b2` | `ded3836` | An empty `DATABASE_OWNER_URL` fails `env.sh check` on the bundled PostgreSQL. |
| rv4-DATA-05 | 1.15 | `FIXED` | `174c764` | `d671860` | The timeline cursor is a `(created_at, id)` position, so a deleted cursor row no longer empties the next page. |

Incidental: `promote-admin` now trims and lowercases the address, the
promote-admin half of the Low finding rv4-DATA-10. Its test arrived with the fix
and has no red of its own, so DATA-10 is not claimed; it stays with the
hygiene batch.

## Wave 1 open residue (integrator, 2026-10-18)

What wave 1 left open on purpose: product code, tests, executing scripts and
workflows (outside a records pass), items a later wave owns, and owner calls.
Nothing here is false after wave 1; each item can only say more or do more.
Each has an ID, an owner and a one-line fix prompt; R1-03 and R1-09 were closed
in wave 2 and say so. The cluster fragments in the
[remediation changelog](../changelogs/2026-10-18-rv4-remediation/index.md)
cite these IDs.

| ID | Left open | Owner | Fix prompt |
|---|---|---|---|
| R1-01 | `ci.yml` has 4 spare lines (346 of 350), so two CI additions did not land: the client image job (rv4-TEST-08, second half) and `TZ=Asia/Shanghai flutter test test/roadmap_deadline_tz_test.dart` (rv4-COMP-02; on a UTC runner that test cannot fail). | The next worker to touch `ci.yml`, after the split in [ci-yml-split](2026-10-18-ci-yml-split.md). | Split `ci.yml` into reusable workflows red-first, then add both items with `check_ci_gates.py` checks. |
| R1-02 | `check_ops_hardening.py` reads `README.md` only for the `prd update` flag order; the root `SKILL.md` had the same defect (corrected in the wave-1 follow-up) and nothing guards it. | Any worker that edits the guards. | Red-first: make the USER-06 check scan `SKILL.md` too, then mutate the file back to id-first and watch it fail. |
| R1-03 | Closed in wave 2 (row 2.5): the `invite_create` description in `server/internal/mcpserver/product_tools.go` names `mailSent` (false means nothing was e-mailed) and an `mcpserver` test pins it. | Done: red `0b2e9fd`, fix `f9e9894`. | None. |
| R1-04 | Only `release.yml` stamps `main.version`; `server/Dockerfile` (its `go build` line) and `scripts/install.sh` (lines 29-30) build unstamped, so those binaries print `dev`. | Unowned; owner call on whether `dev` is right for source builds. | Pass `-ldflags "-X main.version=..."` in both (a build arg; `git describe`), or record that `dev` is intended; extend `check_release_gate.py` either way. |
| R1-05 | A release gate stronger than the test step (see Owner decisions). | Owner. | After the decision: a PostgreSQL service in the release job, or a required green `ci` run on the tagged SHA (`actions: read`). |
| R1-06 | `README.md:86` still lists `insideout setup --local --postgres bundled` under "Once installed" (rv4-USER-09, README half), and lines 182 and 186 do not say `promote-admin` creates the account. | `w3-docs-truth`, row 3.8 (the README hunks). | In the row's one README commit: drop or qualify line 86 for prebuilt users; add the create-when-absent sentence. |
| R1-07 | `server/cmd/insideout/cli_setup.go:148`: `setup --hosted` prints "A server on this machine: insideout setup --local ...", which exits 2 for a prebuilt-binary user (no checkout). | `w3-first-run-copy`, row 3.8. | Red-first test on the `--hosted` output; print the hint only when a checkout is found, or say a checkout is needed. |
| R1-08 | rv4-SEC-06, token half: (a) `fetchCommitPage` aims one server-wide `GITHUB_TOKEN` at every linked repo; (b) `_projects_by_repo` is an exact, case-sensitive match and rows stored before the fix are not canonical; (c) `FetchGuideFile` accepts dot-only names. | `w3-github-evidence`, row 3.10 ((b) together with COMP-05). | Per-installation token in `fetchCommitPage` (the webhook path already uses `s.ghTokens`); case-insensitive lookup plus a migration that canonicalises `repo_url`; refuse dot-only names in `FetchGuideFile`. |
| R1-09 | Closed in wave 2 (row 2.3): `handleAgentPropose` validates the items before its first write, so a refused request (bad title, bad action, an 11th item) leaves no `agent_proposal` on the timeline; a live check confirmed 400 and 0 rows. | Done: red `6d8e957`, fix `bfe019e`. | None. |
| R1-10 | `flagged` on the create, detail, update and repo-link responses is judged from `createdAt` alone (the view is built from `store.ProjectWithLatest{Project: *p}`), while the board judges on the latest update: an old active project with a fresh update reads `flagged: true` on its detail and `false` on the board. | Unowned; file as a finding if wanted. | Red-first route test (old project, fresh update, detail against board); load the latest update into the view in those handlers. |
| R1-11 | The password minimum counts bytes (`len(...) < 8` in `auth.go:74`, `auth_email_flows.go:155` and `promote_admin.go:98`), so three Chinese characters pass "at least 8 characters". | Unowned; a policy call. | Decide bytes or characters; if characters, count runes in all three places with one red test each. |
| R1-12 | `conversations.go` caps a coach message at 10,000 bytes (about 3,333 Chinese characters); `firstLine` in `github_webhook_evidence.go` cuts at 120 bytes and can split a multi-byte character. | Row 3.1 (`conversations.go`); row 3.10 (`firstLine`). | Count runes for the message cap; cut `firstLine` on a rune boundary; one red test each. |
| R1-13 | `Client.Cursor` in `server/internal/apiclient/roadmap.go` passes `nil` as the reply target, so a JSON reply would make it error after the request went out; the route answers 204 today, so it does not bite. | Unowned; row 3.9 touches `apiclient`. | Red-first router test where the cursor route answers 200 with a body; pass a discard target. |
| R1-14 | `scripts/backup-db.sh` has a bash shebang and `set -o pipefail`, so `sh scripts/backup-db.sh` fails under dash (Ubuntu's `sh`); the documented form is `./scripts/backup-db.sh`. | Unowned; low. | Keep the documented form, or make the script POSIX; add a guard that no doc says `sh scripts/backup-db.sh`. |
| R1-15 | Two product questions from `w1-client-polish`: whether the API should take a date-typed deadline (`yyyy-MM-dd`; the PATCH handler parses RFC3339 only) and whether timestamps should show in the viewer's zone (`isoDate` is also `timeAgo`'s date fallback). | Owner (see Owner decisions). | None until decided. |
| R1-16 | `docs/usage/releasing.md` lists the version policy and the rollback as "Not decided yet". | Owner (see Owner decisions). | Write the decision into the page and, if it changes the workflow, into plan row 3.4. |
| R1-17 | Every CI-config property of wave 1 was verified locally only (`actionlint` is not installed): the `release.yml` race-test step (no PostgreSQL service on that runner), the weekly schedule and shuffle (PM-02), the concurrency groups (TEST-08), the guard loop and `fetch-tags` (PM-01, CLOSE-01, TEST-05, PUB-08), the Go regex in `ubuntu-install.yml` (PM-12), the `go mod tidy -diff` step (PM-10) and `INFO_BASELINE` 14 (PM-07). | The first push. | Pending first CI run after push: read the first `ci` run and the first tag run job by job and record each green or red here. |

## Wave 2 dispositions (integrator, 2026-10-19)

One row for each of the 23 rows wave 2 touched: 21 `FIXED`, 2 `PARTIAL`
(`FEAT-02-CLIENT-APPLY`, rv4-DATA-03), no `NOT_REAL`, no `BLOCKED`; the owner
calls and operator actions are `DEFERRED_OWNER` in the last table. SHAs are on
`rv4/integration` (the cluster fragments cite the worker-branch SHAs). No row is
a CI-config property, so none waits for the first CI run. R1-03 and R1-09 are
the closed wave-1 residue rows; `FEAT-02-CLIENT-APPLY` is the worker's id for the
web feed's own re-apply, not a register ID. The gates, the live check and the
carry-forward are in the
[wave-2 record](../changelogs/2026-10-18-rv4-remediation/wave-2.md).

| Register ID | Row | Disposition | Red | Fix | Reason or what is left |
|---|---|---|---|---|---|
| rv4-ARCH-01 | 2.1 | `FIXED` | `38aafed` | `c5af0e9`, chore `8899c73` | One `clientconfig.Resolve(flagAPI)` serves every verb, every `--api` flag and `insideout-mcp`: 37 verb forms ignored `--api` and 7 went to the hosted default (the red message says 38). `scripts/setup.sh:53` still teaches the `INSIDEOUT_TOKEN` export: R2-06. |
| rv4-ARCH-02 | 2.1 | `FIXED` | `3976879` | `ca1c06a` | A saved token goes only to the base it was saved for, and an unreadable config is an error naming the file. A request that cannot start reads `Get "/me": ...`: R2-13. |
| rv4-ARCH-05 | 2.1 | `FIXED` | `acdc088` | `1df2a47`, chore `3645f0d` | `DefaultBase` is a `var`, so `-ldflags -X` retargets it. `release.yml` does not pass the flag yet: R2-07. |
| rv4-SEC-05 | 2.1, 2.5 | `FIXED` | `f312ab6`, `a308076` | `6aa5760`, `5e4968e` | `PARTIAL` in its own cluster (the MCP process fixes its base once); `FIXED` here because rv4-AIDEV-04 closed the admin-tools half in the same wave. |
| rv4-FULL-02 | 2.2 | `FIXED` | `ef1af83` | `96748fc` | Setup keeps the refresh token for every sign-in method. |
| rv4-USER-04 | 2.2 | `FIXED` | `37da942`, `a5a790f` | `12b6a00`, `90bab89` | A saved session renews once on a 401; a loser of a concurrent renewal takes the winner's pair. The wait is a poll: R2-12. An MCP `auth_*` session cannot renew: R2-10. `Client.Guide` prints a plain 401: R2-11. |
| rv4-USER-03 | 2.2 | `FIXED` | `8bed85d` | `5111740` | Plain `login` and `register` write the 0600 config and print its path on stderr. |
| rv4-USER-05 | 2.2 | `FIXED` | `719b780` | `76b55a2` | `setup --check` asks as the signed-in user and fails a rejected login. An unreachable server stays a `skip`, on purpose: the check also runs offline right after install. |
| rv4-FULL-05 | 2.2 | `FIXED` | `f3d51af` | `1d9d1ff` | The web session ends only on a 401 or 403 refresh, not on a dropped connection or a 5xx. |
| rv4-FEAT-02 | 2.3 | `FIXED` | `f624efb` | `a512e0a`, guard `70d3136` | Server half: an item applies once, in the transaction of the decision (`proposal_items.applied_at`), and a failed apply is a 500 with no decision. Three store functions have no production caller: R2-14. |
| FEAT-02-CLIENT-APPLY | 2.3, 2.4 | `PARTIAL` | staged patch | not applied | The web feed (`project_updates_feed.dart`) re-creates every accepted `add_node` item after the server did. The worker's patch was outside the cluster's files and the integrator takes no product code: R2-01. |
| rv4-FEAT-09 | 2.3 | `FIXED` | `3891f08` | `f8c9959` | The Decision Log comment says the row is upserted, not immutable. Older migrations keep their historical comments. |
| rv4-LEAD-PROPOSE-VALIDATE | 2.3 | `FIXED` | `6d8e957` | `bfe019e` | This is R1-09: propose validates before its first write. |
| rv4-FULL-01 | 2.4 | `FIXED` | `2046fb5` | `29c0956` | Web "Break down with AI" drafts, shows the draft and confirms it; Go and Dart contract tests share fixtures. The expand route takes no position lock and gates on membership only: R2-04. |
| rv4-FEAT-01 | 2.5 | `FIXED` | `62e68f4` | `3a22813`, design `9b3cdb9` | A signed `scope` claim and one fail-closed `agentRoutePolicy`: an agent token is refused at the proposal decision, Commit, the admin routes and deletes (403 `agent_scope_denied`). `DEFERRED_OWNER`: whether an agent token may ever decide a proposal. Audit rows still carry the user id: R2-05. Accepted limits: R2-15. |
| rv4-AIDEV-04 | 2.5 | `FIXED` | `a308076`, `c4087ec` | `5e4968e`, `8def9d6` | A JWT login's registry is 40 tools, not 52: twelve are withheld (`signup_mode`, `invite_*`, deletes). A process with no credential keeps 52 until its first sign-in: R2-15. |
| rv4-AIDEV-01 | 2.5 | `FIXED` | `883daa1` | `6e6622f` | Coach and MCP results come back labelled and inside `<prd_data>`. `agent/prompt_fence.go` still holds its own copy of the fence rule: R2-08. |
| rv4-AIDEV-08 | 2.5 | `FIXED` | `53ddb09` | `7aaf8ad` | Context and evidence carry an `untrusted` block. The CLI `context` and `evidence` verbs print the API JSON unfenced: R2-09. |
| R1-03 | 2.5 | `FIXED` | `0b2e9fd` | `f9e9894` | The `invite_create` description names `mailSent` (residue row of wave 1). |
| rv4-DATA-01 | 2.6 | `FIXED` | `91098a7` | `f696db4` | A membership `INSERT` needs a reason: an `admin` row only for the workspace's creator, a `member` row only with the join code. A workspace admin can still re-point a seat: R2-03. |
| rv4-DATA-02 | 2.6 | `FIXED` | `91098a7` | `f108e64` | A write-side RLS walk runs every write as `insideout_app` (the twin of the read walk). It shares its red commit with DATA-01 and DATA-03; one of its tests is green on the base on purpose, as a guard. |
| rv4-DATA-03 | 2.6 | `PARTIAL` | `91098a7` | `f108e64` | Write half done: `users` `INSERT` is cut by column grants and `UPDATE` is pinned by a trigger. Read half: a co-member's `password_hash` is readable to anything that holds the app role (no API response carries it): R2-02. |
| rv4-DATA-04 | 2.6 | `FIXED` | `ad7ce63` | `e9d8a79` | `serve` refuses a `BYPASSRLS` or `SUPERUSER` pool role before it migrates and prints no DSN. `SUPERUSER` is covered by a table test only; the hosted role is an operator check (Owner decisions). |

## Wave 2 open residue (integrator, 2026-10-19)

What wave 2 left open on purpose: product code, tests, executing scripts and
workflows (outside a records pass), items a later wave owns, and owner calls.
Nothing here is false after wave 2; each item can only say more or do more. Each
has an ID, an owner and a one-line fix prompt; the cluster fragments and the
[wave-2 record](../changelogs/2026-10-18-rv4-remediation/wave-2.md) cite these
IDs. R2-01 and R2-02 are new wave-3 clusters (`w3-client-apply`,
`w3-users-read-grant`); R2-06 and R2-08 ride on existing wave-3 rows.

| ID | Left open | Owner | Fix prompt |
|---|---|---|---|
| R2-01 | `FEAT-02-CLIENT-APPLY`: `client/lib/features/project/project_updates_feed.dart` calls `_applyAcceptedItems` after `decideProposal`, so an accept through the web feed builds each `add_node` item a second time after the server applied it. | `w3-client-apply` (new, the 12th wave-3 cluster). | Red-first: rename 'accepting a proposal with one item adds that roadmap node' in `client/test/project_page_test.dart` to 'accepting a proposal leaves applying its items to the server' (the fake records `(id, accept, apply)` and expects `[(up1, true, true)]` and no client-side node); then delete `_applyAcceptedItems` and its call so `_decide` is one `decideProposal` write, and drop the `api/client.dart` import only it used. The worker checked this in a scratch copy: 474 client tests, format and analyze clean. |
| R2-02 | rv4-DATA-03 read half: `users_select` plus a table-wide `SELECT` let anything that holds the app role read a co-member's `password_hash` (`userColumns` in `store/users.go` selects it on every user read; no API response carries it). | `w3-users-read-grant` (new, the 13th wave-3 cluster, after `w3-auth-network`). | Add a `SECURITY DEFINER` `insideout._credential_for_email(email)` returning `(id, password_hash)`; `REVOKE SELECT` on `insideout.users` from `insideout_app` and re-grant every column but `password_hash`; drop `password_hash` from `userColumns`; make the login in `api/auth.go` (around lines 167 and 177) use the helper; extend the write walk with a `SELECT (password_hash)` as a co-member that must be refused (about 25 lines and one test). |
| R2-03 | `workspace_memberships_update` is `USING` only (the default check repeats it), and nothing pins `user_id`, so a workspace admin can rewrite a membership's `user_id` and re-point a seat; the only application `UPDATE` sets `role` (`store/memberships.go`). A lead, not a confirmed finding. | Unowned; the final review's security lens. | Red-first case in `rls_write_walk_test.go`; then a `BEFORE UPDATE` trigger that pins `user_id` and `workspace_id`, like `users_pin_columns`. |
| R2-04 | `ExpandRoadmapNode` (`store/roadmap_plan.go`) computes `MAX(position)+1` without the per-project advisory lock that `createRoadmapNodeTx` takes, and gates on `requireMember` where the other roadmap writes use `requirePlanWriter` ([BUG-020](2026-10-06-bug-020-workspace-member-writes-the-plan.md)). Read from the code, not exercised live. | Unowned; a final-review lead (row 2.4 touched the route). | Red-first concurrent-confirm test (two confirms must land on distinct positions); take the lock as `createRoadmapNodeTx` does; decide member or plan writer and record it. |
| R2-05 | Per-row actor attribution: an agent token acts under the user's id, so audit and update rows carry the user and only the log line gains `scope=agent`. | Unowned; an owner call (a schema change). | If wanted: one column on the rows an agent can write, set from the verified scope claim in one place, with a migration and a red test; otherwise record that the log line is the attribution. |
| R2-06 | `LEAD-SETUP-BLURB`: `scripts/setup.sh` (`hosted_blurb`, the export on line 53) still teaches `export INSIDEOUT_TOKEN="$(insideout login you@example.com)"`, although `login` now saves the session and an `INSIDEOUT_TOKEN` in the environment hides it. | `w3-first-run-copy`, row 3.8. | Say `insideout login you@example.com` then `insideout whoami`; pin it in the first-run copy test. |
| R2-07 | `.github/workflows/release.yml` builds both binaries (`go build` at lines 49 and 52) without `-X ...apiclient.DefaultBase=`, so release binaries keep the compiled-in hosted default. | The owner names the repository variable (Owner decisions); then the next worker to touch `release.yml`. | Pass `-X` to both builds from that variable, skipped when it is empty, failing when it is set but not an https URL ending `/api/v1`; extend `check_release_gate.py`. Pending first CI run after push (R1-17). |
| R2-08 | `LEAD-FENCE-DEDUP`: `server/internal/agent/prompt_fence.go` (`fenceData`, line 19) still holds its own copy of the fence rule that `agentcontext.FenceData` implements; `TestFenceDataAgreesWithTheMCPFence` keeps the two equal. | `w3-coach-trust`, row 3.1 (an optional row). | Make `fenceData` delegate to `agentcontext.FenceData`; the parity test becomes a plain guard. |
| R2-09 | The CLI `context` and `evidence` verbs print the API JSON, including its `untrusted` block, without the `<prd_data>` fence; the fence covers the MCP results and the coach prompt. | Unowned; an owner call. | If the CLI should be covered: print the user-written fields through the same fence, one test per verb. Otherwise record the CLI as human-facing. |
| R2-10 | `mcpserver/auth_tools.go`: the `auth_*` tools hand the access token to the process session (`setSessionToken(c.Token())`) but not the refresh token, so an in-process sign-in lives one access TTL (15 minutes) and cannot renew. | Unowned; an optional lead. | Red-first with a short access TTL (the live check used 3 s); arm `SetRenewal` for the in-process session from the sign-in's refresh token. |
| R2-11 | `Client.Guide` (`apiclient/roadmap.go`, line 151) builds its own request: it renews through the transport, but a final 401 prints the plain status, not the shared wording of `unauthorized` (which names the saved login and an `INSIDEOUT_TOKEN` override). | Unowned; an optional lead. | Return `c.unauthorized("GET guide")` on a 401, as `get` does; one test. |
| R2-12 | The renewal wait is a 500 ms poll (`renewalGrace` in `apiclient/refresh.go`, marked `ponytail:`), not a cross-process lock: each loser sends one refused refresh, which counts against the per-IP auth throttle (30 a minute). `Client.renew` and `token` are also written without a lock (the CLI is sequential). | Unowned; revisit if the server ever revokes a refresh family on reuse, or a burst still loses. | Serialize with a lock file beside the config, as the `ponytail:` comment says; red-first with N parallel processes against a 3 s access TTL. |
| R2-13 | Client-credentials nits: (a) an unreadable-config request reads `Get "/me": ...` (the transport error is wrapped by `net/http`); (b) the ARCH-01 red message says "38 verb forms" where the exact count is 37 plus 7 saved-base verbs (history is not rewritten); (c) one saved login per machine, with no profiles. | Unowned; cosmetic. | (a) Unwrap the `*url.Error` where `Failing`'s error is printed, with one message assertion; (b) none; (c) a product call, not a defect. |
| R2-14 | `AddProposalItems`, `ApplyProposalItemsForUpdate` and `ApplyProposalItems` (`store/proposal_items.go`) have no production caller: the routes use `AddProposal` and `DecideAndApplyProposal`. Only store tests reach them. | Unowned; low. | Delete them and move their claim tests onto `DecideAndApplyProposal`, or comment them as a test seam. |
| R2-15 | Accepted limits of the agent scope (listed in the [design](../architecture/agent-token-scope.md)): the scope defends an agent confined to MCP tools, not one that can read the saved config or environment; a process with no credential keeps the 52-tool registry until its first sign-in; a cached agent token outlives a server-secret rotation or a clock skew by up to one token life; the server must deploy before the MCP binary; fencing is a label, not a sandbox. | Owner (accepted). | None; revisit if the threat model grows. |

## Gate note (wave 0, coordinator, 2026-10-18)

The Go test gate was red on exactly one test, `server/internal/api`
`TestProjectQuietFlagged` (2 subtests: "active with fresh update is not
flagged" and "active recently created with no updates is not flagged"). It
fails identically at the wave base `d53035a` and at the integration head
`97d7529` (the coordinator re-ran it under `env -i`, 2026-10-18).
`w0-privacy` changes no Go file, so wave 0 is recorded as green apart from
that proven baseline failure. Plan row 1.3 (wave 1, cluster
`w1-clock-and-runes`) fixes it; from wave 1 on the full suite must be fully
green. Source: `coordinator-w0.json` in the remediation reports directory
(outside the repository); the wave record is in the
[wave-0 record](../changelogs/2026-10-18-rv4-remediation/wave-0.md).

## Gate note (wave 1, integrator, 2026-10-18)

At `a079404` (74 cherry-picked commits and one fix-up, no conflicts) every gate
was green with no baseline failure: `gofmt`, `go build`, `go vet` and
`go mod tidy -diff` clean; `go test ./... -race -count=1` 27 packages ok, the
only skip `TestReadSecretTerminal`, and `TestProjectQuietFlagged` passing, so
the wave-0 failure above is fixed by row 1.3; Flutter `analyze` 14 infos (equal
to `INFO_BASELINE`) and 466 tests, also under `TZ=Asia/Shanghai`; all nine
`scripts/tests/check_*.py` guards, `sh scripts/check-redaction.sh` and
`sh scripts/check-docs-recorded.sh` exit 0. The CI-config properties are
pending first CI run after push (R1-17). The full list is in the
[wave-1 record](../changelogs/2026-10-18-rv4-remediation/wave-1.md#gates-integrator-at-a079404).

**Coordinator note (do not redo).** The integrator already applied this
fix-up in `a079404`:

```json
[{"item":"w1-misc-correctness r1 backend-layout patch","path":"docs/architecture/backend.md","what":"added the modtidy/ Layout line between mcpserver/ and presence/ so check_docs_truth.py stays green"}]
```

## Gate note (wave 2, integrator, 2026-10-19)

At `e6bc637` (55 cherry-picked commits, no conflicts, no fix-ups, 148 above
`d53035a`) every gate was green with no baseline failure, on a fresh private
PostgreSQL 18 that was torn down afterwards: `gofmt`, `go build`, `go vet` and
`go mod tidy -diff` clean; the migrations apply from empty, including
`20261018120000` and `20261018140000`, and a second `migrate` does nothing;
`go test ./... -race -count=1` every package ok (1,563 `PASS` lines, the only
skip `TestReadSecretTerminal`, no data race); Flutter `pub get` with
`--enforce-lockfile` ok, `analyze` 14 infos (equal to `INFO_BASELINE`) and 483
tests, also under `TZ=Asia/Shanghai`; all nine `scripts/tests/check_*.py` guards,
`sh scripts/check-redaction.sh` and `sh scripts/check-docs-recorded.sh` exit 0.
No workflow file changed, so R1-17 stands as it was. A live check of seven
behaviours on real binaries and a real database is in the
[wave-2 record](../changelogs/2026-10-18-rv4-remediation/wave-2.md#live-check-integrator).

## Owner decisions (no engineering action until decided)

Dispositions of the first six rows are wave 0's, the next two are wave 1's and
the last four are wave 2's; SHAs are on `rv4/integration`, and the privacy halves
are written out in [the owner record](2026-10-18-rv4-owner-privacy-decisions.md).

| Register ID | Decision | Default until decided | Disposition |
|---|---|---|---|
| rv4-PUB-01, rv4-PM-11 | Rotate the hosted admin address after scrubbing it from five docs. | Scrub the docs (plan row 0.1); keep the address. | Scrub `FIXED`: PUB-01 red `c9216d7`, fix `7df9dfe`; PM-11 red `950f0ee`, fix `98f7c15`; the stems, the owner mail domain and the quoted finding left the handoff, the plans board and the review report in `3d65d50` (grep-verified: the guard cannot see truncated stems). Rotation `DEFERRED_OWNER`. |
| rv4-PUB-05 | Rotate or disable the legacy Supabase publishable key and confirm the Data API exposes nothing. | Delete `.gitignore:72` (plan row 0.2); key untouched. | `DEFERRED_OWNER`. The code half is `FIXED`: red `8b80c54`, fix `23b2617` (`.env.*` ignores the legacy env file again). The dashboard check and the key are the owner's. |
| rv4-PUB-06 | Accept the institutional author identities in history, or rewrite history (force-push of `main`). | Accept; do not rewrite. | `DEFERRED_OWNER`. Nothing to change by default; the decision is filed in `67da7b2`. The release tag's tagger address falls under this row. |
| rv4-PUB-10 | Ship `hosts/` as an example tree and ignore the real one. | Scrub paths to placeholders only. | Scrub `FIXED`: red `8b421b7`, fix `22f0665`; the OS user left two older docs and the setup skill's host sentence matches in `3d65d50`. The example tree is `DEFERRED_OWNER`. |
| rv4-PUB-12 | Whether the deployed host also becomes a placeholder in dated records (the provider ids already did). | Keep the host: the shipped default, public by design. | Ids `FIXED`: red `848ad0c`, fix `c2b5caa`, pinned to three records by a test (no tree-wide UUID rule, on purpose). The host is the owner's call. |
| rv4-PUB-02, rv4-DATA-06 | Privacy notice, retention and erasure posture for the hosted service. | Write `PRIVACY.md` first (plan row 3.5). | Open, wave 3. |
| rv4-PM-05 (with rv4-TEST-01) | The version policy, the rollback rule, and whether the release gate becomes stronger than the test step (a PostgreSQL service in the release job, or a required green `ci` run on the tagged SHA). | The test step is the floor; [releasing](../usage/releasing.md) records the process as it is and lists these three as open. | `DEFERRED_OWNER` (R1-05, R1-16). The test gate is `FIXED` and the page exists. |
| rv4-COMP-02 (follow-on) | Whether deadlines become date-typed in the API, and whether timestamps show in the viewer's zone. | The picker sends noon UTC of the picked day; labels read the UTC date. | `DEFERRED_OWNER` (R1-15). Not scheduled. |
| rv4-FEAT-01 (follow-on) | Whether an agent token may ever decide a proposal, for example in a single-user workspace where the owner is also the author, or whether a decision always needs an interactive human session. | Strict: an agent token never decides (403 `agent_scope_denied`). | `DEFERRED_OWNER`. Relaxing it is one `agentRoutePolicy` entry for the decision route plus the handler's own check in `agent_routes.go`; R2-15 lists the other accepted limits. |
| rv4-ARCH-05 (follow-on) | The name of the repository variable that feeds `-X ...apiclient.DefaultBase` in `release.yml`, and whether release binaries are retargeted at all. | The compiled-in hosted default stays; the workflow reads no variable. | `DEFERRED_OWNER` (R2-07). A source build can already retarget with `-ldflags`. |
| rv4-DATA-04 | Confirm that the hosted `DATABASE_URL` is the `insideout_app` role and `DATABASE_OWNER_URL` the owner, before the first deploy that contains wave 2. | `serve` exits 1 before it migrates on a pool role with `BYPASSRLS` or `SUPERUSER`; the hosted environment was not inspected from here. | Operator action, `DEFERRED_OWNER`. Rollback is redeploying the previous image; `migrate`, `seed` and `promote-admin` never reach the check. |
| rv4-AIDEV-04 | Deploy the server before the MCP binary. | A new `insideout-mcp` against a server without `POST /api/v1/agent/token` fails every tool with a hint to update the server. | Operator action, `DEFERRED_OWNER`. Tokens issued before the change carry no `scope` claim and stay full sessions, so nobody signs in again. |

## What closes this record

Rows 0.1 to 2.6 of the plan are done and their checks pass, or each remaining
row has its own dated record. Rows 0.1 to 2.6 are recorded (waves 0 to 2); the
owner halves of 0.1 to 0.4 stay open in
[the owner record](2026-10-18-rv4-owner-privacy-decisions.md), and the wave-1
and wave-2 residue stays open above until each R1 and R2 item is done or decided
by the owner (R1-03 and R1-09 are done).
The Medium and Low rows stay in the register and the plan; they do not get
issue files unless a wave opens one.
