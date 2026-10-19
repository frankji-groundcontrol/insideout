# Review 4 dispositions — the closed rows, wave by wave

The per-ID dispositions of waves 1 to 3 of the
[review-4 remediation](../changelogs/2026-10-18-rv4-remediation/index.md), moved out of the
[open-findings record](2026-10-18-review-4-open-findings.md) so that both stay
inside the 350-line budget. That record keeps the Highs, the open residue (R1,
R2, R3), the gate notes, the coordinator notes and the owner decisions; this one
keeps what is closed. Wave 0's dispositions are the first six rows of the owner
decisions there. Cite IDs as `rv4-USER-01` outside the review folder (review IDs
restart per review). A disposition is `FIXED` (red and fix commits), `NOT_REAL`,
`DEFERRED_OWNER`, `PARTIAL` or `BLOCKED`, plus the reason.

## Wave 1 dispositions (integrator, 2026-10-18)

One row for each of the 34 findings wave 1 touched: 30 `FIXED`, 4 `PARTIAL`
(rv4-PM-05, rv4-TEST-08, rv4-SEC-06, rv4-USER-09), no `NOT_REAL`, no `BLOCKED`;
the owner halves of rv4-PM-05 and rv4-COMP-02 are `DEFERRED_OWNER`. SHAs are on
`rv4/integration` (the cluster fragments cite the worker-branch SHAs). A
CI-config property can only be checked locally here (`actionlint` is not
installed), so those rows end with "pending first CI run after push" (R1-17). Run
state and the clusters are in the
[wave-1 record](../changelogs/2026-10-18-rv4-remediation/wave-1.md);
what each cluster left open is the
[residue table](2026-10-18-review-4-open-findings.md#wave-1-open-residue-integrator-2026-10-18). A "Wave 3:" tail marks a row that wave 3
changed; the counts are as wave 1 left them.

| Register ID | Row | Disposition | Red | Fix | Reason or what is left |
|---|---|---|---|---|---|
| rv4-USER-01 | 1.1 | `FIXED` | `f30001c` | `c913e8e` | `promote-admin` creates the account when it is absent (password from stdin, never argv). The password minimum counts bytes: R1-11. |
| rv4-USER-02 | 1.2 | `FIXED` | `e0908b3` | `abc19a0` | `help`, `-h`, `--help`, `version`, `--version` exit 0 and an unknown verb exits 2, all before `config.Load()`. |
| rv4-FULL-03 | 1.2 | `FIXED` | `e0908b3` | `abc19a0` | Same commits as USER-02 (the verb table and its drift test). |
| rv4-PM-04 | 1.2 | `FIXED` | `e0908b3`, `0baafaf` | `abc19a0`, `b7db11d` | `insideout version` prints the tag that `release.yml` stamps into `main.version`. The container image and `scripts/install.sh` print `dev`: R1-04. Wave 3: stamping `install.sh` and the Dockerfile was written (red `0de150c`, fix `5320251`) and reverted (`c201f92`) because neither file was the cluster's; it moves to the proposed `w4-release-version` (R1-04, R3-18). |
| rv4-TEST-01 | 1.5 | `FIXED` | `0baafaf` | `b7db11d` | `release.yml` runs `go test ./... -race -count=1` before any build or upload. The release runner has no PostgreSQL service: R1-05. Pending first CI run: R1-17. |
| rv4-PM-05 | 1.5 | `PARTIAL` | `0baafaf` | `b7db11d` | The test gate is done with TEST-01 and [releasing](../usage/releasing.md) records the process (`7091e14`). `DEFERRED_OWNER`: the version policy, the rollback rule, a stronger gate (R1-05, R1-16). Wave 3: the gate half landed with rv4-PUB-03 (pending first CI run); the `releasing.md` rewrite was reverted (`12bc405`) and the policy and rollback stay `DEFERRED_OWNER` (R3-18). |
| rv4-PM-02 | 1.4 | `FIXED` | `8b9cd0b` | `57fa71e`, `5f52381` | A weekly `schedule` runs `go test ./... -race -shuffle=on`; HANDOFF cites `9f0b1c7` (2026-10-11) since `7091e14`, which closed the clause the worker had reported `PARTIAL`. Pending first CI run: R1-17. |
| rv4-TEST-08 | 1.4 | `PARTIAL` | `735ba99`, `4609a63` | `adf2177`, `cc534e7` | In-progress runs are cancelled for pull requests only; every other run has a group of its own. The client image job did not fit (`ci.yml` 346 of 350 lines): R1-01, [ci-yml-split](2026-10-18-ci-yml-split.md). Wave 3: `ci.yml` is now 349 of 350 lines (R3-20). |
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
| rv4-COMP-01 | 1.17 | `FIXED` | `0129059` | `e129f02` | One `tooLong` helper counts characters on the 18 capped fields; 67 Chinese characters pass. Byte caps outside the sweep (the coach message, `firstLine`): R1-12. Wave 3: `firstLine` cuts on a character boundary (`1bcbc99`, `e9ead0e`); the coach message cap stays R1-12. |
| rv4-SEC-01 | 1.9 | `FIXED` | `b4edce1` | `4df79d6` | A login-code guess is charged and compared in one atomic `UPDATE`, so a concurrent burst is capped at 5. |
| rv4-USER-07 | 1.10 | `FIXED` | `bee99f7` | `a7f9519` | Create-invitation returns `mailSent` and the dialog says when nothing was e-mailed. The `invite_create` MCP description names it since wave 2 (R1-03, `f9e9894`). |
| rv4-QUAL-02 | 1.13 | `FIXED` | `8765fed` | `5cf5bc2` | The resend cooldown keys on `clientIP`, not a drifted copy, so a trailing-empty `X-Forwarded-For` no longer puts every client in one bucket. |
| rv4-QUAL-03 | 1.13 | `FIXED` | `e45229e` | `868e49f` | Every apiclient path segment and query value is escaped (39 rows under a completeness test). `Client.Cursor` passes a nil reply target: R1-13. |
| rv4-SEC-06 | 1.14 | `PARTIAL` | `e646d8e` | `cbd9fe8` | The code half is fixed: dot-only repo segments and foreign hosts are refused and the canonical URL is stored. The token half (a per-installation token, a case-insensitive lookup with a canonicalising migration, `FetchGuideFile` dot names) goes to `w3-github-evidence`, row 3.10: R1-08. Wave 3: `FIXED` with `1aebc35`, `e7356b8`, the FEAT-04 installation token and the COMP-05 canonical lookup; R1-08 is closed. |
| rv4-USER-11 | 1.12 | `FIXED` | `b17641a` | `bed39fb` | The Edit-project description field is labelled as a description. |
| rv4-COMP-02 | 1.16 | `FIXED` | `5a47205` | `137d6f5` | The picked deadline is sent as noon UTC of its calendar day, so 2026-10-18 reads 2026-10-18 under `TZ=Asia/Shanghai`. CI runs in UTC and cannot see the bug until the Shanghai-zone step lands: R1-01. Date-typed deadlines and viewer-zone timestamps are `DEFERRED_OWNER`: R1-15. |
| rv4-USER-06 | 1.7 | `FIXED` | `a875428` | `7b6f4c0` | The README `prd update` example puts the id after the flags; root `SKILL.md` was corrected in `7091e14`. The guard does not scan `SKILL.md`: R1-02. |
| rv4-REL-05 | 1.8 | `FIXED` | `092f7f1` | `3a8defb` | The compose `server` has `stop_grace_period: 35s`, explained in the deployment architecture. `docker compose config -q` has not run anywhere: R1-17. |
| rv4-USER-09 | 1.12 | `PARTIAL` | `8161369` | `6f32133` | The installer's closing message no longer offers `setup --local`. `README.md:86` (R1-06) and the `setup --hosted` hint at `cli_setup.go:148` (R1-07) still do; both are wave 3. Wave 3: the README half is closed (R1-06, `583839d`, `9299dc6`); the `--hosted` hint stays open as R1-07 (R3-46). |
| rv4-COMP-07 | 1.18 | `FIXED` | `ef63477` | `074d4e0` | The backup dump is written 0600 (umask and chmod) and the usage doc says what it holds. `sh scripts/backup-db.sh` fails under dash: R1-14. |
| rv4-AIDEV-10 | 1.15 | `FIXED` | `4d04a6d` | `9da2086` | The `propose` tool schema declares `items`. |
| rv4-QUAL-01 | 1.15 | `FIXED` | `9deac14` | `d40c64b` | One blank-section helper serves readiness and the coach context; the `openQuestions` order is stable. |
| rv4-FULL-06 | 1.15 | `FIXED` | `4fe72b2` | `ded3836` | An empty `DATABASE_OWNER_URL` fails `env.sh check` on the bundled PostgreSQL. |
| rv4-DATA-05 | 1.15 | `FIXED` | `174c764` | `d671860` | The timeline cursor is a `(created_at, id)` position, so a deleted cursor row no longer empties the next page. |

Incidental: `promote-admin` now trims and lowercases the address, the
promote-admin half of the Low finding rv4-DATA-10. Its test arrived with the fix
and has no red of its own, so DATA-10 is not claimed; it stays with the
hygiene batch.

## Wave 2 dispositions (integrator, 2026-10-19)

One row for each of the 23 rows wave 2 touched: 21 `FIXED`, 2 `PARTIAL`
(`FEAT-02-CLIENT-APPLY`, rv4-DATA-03; wave 3 closed both), no `NOT_REAL`, no
`BLOCKED`; the owner
calls and operator actions are `DEFERRED_OWNER` in the
[owner decisions](2026-10-18-review-4-open-findings.md#owner-decisions-no-engineering-action-until-decided). SHAs are on
`rv4/integration` (the cluster fragments cite the worker-branch SHAs). No row is
a CI-config property, so none waits for the first CI run. R1-03 and R1-09 are
the closed wave-1 residue rows; `FEAT-02-CLIENT-APPLY` is the worker's id for the
web feed's own re-apply, not a register ID. The gates, the live check and the
carry-forward are in the
[wave-2 record](../changelogs/2026-10-18-rv4-remediation/wave-2.md).

| Register ID | Row | Disposition | Red | Fix | Reason or what is left |
|---|---|---|---|---|---|
| rv4-ARCH-01 | 2.1 | `FIXED` | `38aafed` | `c5af0e9`, chore `8899c73` | One `clientconfig.Resolve(flagAPI)` serves every verb, every `--api` flag and `insideout-mcp`: 37 verb forms ignored `--api` and 7 went to the hosted default (the red message says 38). `scripts/setup.sh:53` still teaches the `INSIDEOUT_TOKEN` export: R2-06. Wave 3: R2-06 is closed (`e6ab582`, `7ba574e`). |
| rv4-ARCH-02 | 2.1 | `FIXED` | `3976879` | `ca1c06a` | A saved token goes only to the base it was saved for, and an unreadable config is an error naming the file. A request that cannot start reads `Get "/me": ...`: R2-13. |
| rv4-ARCH-05 | 2.1 | `FIXED` | `acdc088` | `1df2a47`, chore `3645f0d` | `DefaultBase` is a `var`, so `-ldflags -X` retargets it. `release.yml` does not pass the flag yet: R2-07. |
| rv4-SEC-05 | 2.1, 2.5 | `FIXED` | `f312ab6`, `a308076` | `6aa5760`, `5e4968e` | `PARTIAL` in its own cluster (the MCP process fixes its base once); `FIXED` here because rv4-AIDEV-04 closed the admin-tools half in the same wave. |
| rv4-FULL-02 | 2.2 | `FIXED` | `ef1af83` | `96748fc` | Setup keeps the refresh token for every sign-in method. |
| rv4-USER-04 | 2.2 | `FIXED` | `37da942`, `a5a790f` | `12b6a00`, `90bab89` | A saved session renews once on a 401; a loser of a concurrent renewal takes the winner's pair. The wait is a poll: R2-12. An MCP `auth_*` session cannot renew: R2-10. `Client.Guide` prints a plain 401: R2-11. |
| rv4-USER-03 | 2.2 | `FIXED` | `8bed85d` | `5111740` | Plain `login` and `register` write the 0600 config and print its path on stderr. |
| rv4-USER-05 | 2.2 | `FIXED` | `719b780` | `76b55a2` | `setup --check` asks as the signed-in user and fails a rejected login. An unreachable server stays a `skip`, on purpose: the check also runs offline right after install. |
| rv4-FULL-05 | 2.2 | `FIXED` | `f3d51af` | `1d9d1ff` | The web session ends only on a 401 or 403 refresh, not on a dropped connection or a 5xx. |
| rv4-FEAT-02 | 2.3 | `FIXED` | `f624efb` | `a512e0a`, guard `70d3136` | Server half: an item applies once, in the transaction of the decision (`proposal_items.applied_at`), and a failed apply is a 500 with no decision. Three store functions have no production caller: R2-14. Wave 3: the web half is closed (`FEAT-02-CLIENT-APPLY`). |
| FEAT-02-CLIENT-APPLY | 2.3, 2.4 | `PARTIAL` | staged patch | not applied | The web feed (`project_updates_feed.dart`) re-creates every accepted `add_node` item after the server did. The worker's patch was outside the cluster's files and the integrator takes no product code: R2-01. Wave 3: `FIXED`, red `6096862`, fix `0d33077`; R2-01 is closed (leftovers R3-33). |
| rv4-FEAT-09 | 2.3 | `FIXED` | `3891f08` | `f8c9959` | The Decision Log comment says the row is upserted, not immutable. Older migrations keep their historical comments. |
| rv4-LEAD-PROPOSE-VALIDATE | 2.3 | `FIXED` | `6d8e957` | `bfe019e` | This is R1-09: propose validates before its first write. |
| rv4-FULL-01 | 2.4 | `FIXED` | `2046fb5` | `29c0956` | Web "Break down with AI" drafts, shows the draft and confirms it; Go and Dart contract tests share fixtures. The expand route takes no position lock and gates on membership only: R2-04. |
| rv4-FEAT-01 | 2.5 | `FIXED` | `62e68f4` | `3a22813`, design `9b3cdb9` | A signed `scope` claim and one fail-closed `agentRoutePolicy`: an agent token is refused at the proposal decision, Commit, the admin routes and deletes (403 `agent_scope_denied`). `DEFERRED_OWNER`: whether an agent token may ever decide a proposal. Audit rows still carry the user id: R2-05. Accepted limits: R2-15. |
| rv4-AIDEV-04 | 2.5 | `FIXED` | `a308076`, `c4087ec` | `5e4968e`, `8def9d6` | A JWT login's registry is 40 tools, not 52: twelve are withheld (`signup_mode`, `invite_*`, deletes). A process with no credential keeps 52 until its first sign-in: R2-15. |
| rv4-AIDEV-01 | 2.5 | `FIXED` | `883daa1` | `6e6622f` | Coach and MCP results come back labelled and inside `<prd_data>`. `agent/prompt_fence.go` still holds its own copy of the fence rule: R2-08. Wave 3: R2-08 is closed (`f893a32`, `3f7ba17`). |
| rv4-AIDEV-08 | 2.5 | `FIXED` | `53ddb09` | `7aaf8ad` | Context and evidence carry an `untrusted` block. The CLI `context` and `evidence` verbs print the API JSON unfenced: R2-09. |
| R1-03 | 2.5 | `FIXED` | `0b2e9fd` | `f9e9894` | The `invite_create` description names `mailSent` (residue row of wave 1). |
| rv4-DATA-01 | 2.6 | `FIXED` | `91098a7` | `f696db4` | A membership `INSERT` needs a reason: an `admin` row only for the workspace's creator, a `member` row only with the join code. A workspace admin can still re-point a seat: R2-03. |
| rv4-DATA-02 | 2.6 | `FIXED` | `91098a7` | `f108e64` | A write-side RLS walk runs every write as `insideout_app` (the twin of the read walk). It shares its red commit with DATA-01 and DATA-03; one of its tests is green on the base on purpose, as a guard. |
| rv4-DATA-03 | 2.6 | `PARTIAL` | `91098a7` | `f108e64` | Write half done: `users` `INSERT` is cut by column grants and `UPDATE` is pinned by a trigger. Read half: a co-member's `password_hash` is readable to anything that holds the app role (no API response carries it): R2-02. Wave 3: `FIXED`, red `8f57d77`, fix `c57073b` (migration `20261018200000_users_read_grants.sql`); R2-02 is closed (ceilings R3-34). |
| rv4-DATA-04 | 2.6 | `FIXED` | `ad7ce63` | `e9d8a79` | `serve` refuses a `BYPASSRLS` or `SUPERUSER` pool role before it migrates and prints no DSN. `SUPERUSER` is covered by a table test only; the hosted role is an operator check (Owner decisions). |

## Wave 3 dispositions (integrator, 2026-10-19)

One row for each of the 64 dispositions wave 3 recorded: 48 `FIXED`, 13 `PARTIAL`
(rv4-AIDEV-07, rv4-SEC-04, rv4-PUB-03, rv4-SEC-07, rv4-PUB-02, rv4-PM-09,
rv4-TEST-04, rv4-TEST-06, rv4-CLOSE-08, rv4-FEAT-05, rv4-USER-12, rv4-QUAL-04,
rv4-COMP-05), 3 `DEFERRED_OWNER` (rv4-DATA-06, rv4-TEST-07, rv4-CLOSE-09), no
`NOT_REAL`, no `BLOCKED`. Three more rows carry an owner half beside finished
engineering (rv4-AIDEV-02, rv4-COMP-05, `COMP-08-DECISION`); those are in the
[owner decisions](2026-10-18-review-4-open-findings.md#owner-decisions-no-engineering-action-until-decided). SHAs are on `rv4/integration`
and each integration commit is patch-identical to its worker commit (the
fragments cite the worker-branch SHAs). Every `FIXED` has a red test that failed
before its fix, except rv4-TEST-03 (a coverage task) and rv4-CLOSE-10 (the
integrator's pre-gate patch); six rows say `-` in the Red column: those two,
rv4-TEST-04 (test-only commits) and the three `DEFERRED_OWNER` rows. A row that
rests on a workflow file or a CI-config property is **pending first CI run after
push** (R1-17); that is not repeated in every cell. The R3 IDs are the rows of the
[wave-3 residue table](2026-10-18-review-4-open-findings.md#wave-3-open-residue-integrator-2026-10-19). The cluster of
each row and the judgement calls are in
[wave-3-dispositions.md](../changelogs/2026-10-18-rv4-remediation/wave-3-dispositions.md); the gates, the lane
logs and the operator notes are in the [wave-3 record](../changelogs/2026-10-18-rv4-remediation/wave-3.md).

| Register ID | Row | Disposition | Red | Fix | Reason or what is left |
|---|---|---|---|---|---|
| rv4-AIDEV-02 | 3.1 | `FIXED` | `493ed84` | `7628169` | The finalize gate is kept on purpose: a skipped critic still reaches finalize, but the stream, the view and the finalize card now say no review ran. `DEFERRED_OWNER`: whether to refuse while a skip is set, or only on contention, is one condition plus its pinning test (owner decisions) |
| rv4-AIDEV-03 | 3.1 | `FIXED` | `da550d4` | `2a7a66b` | none |
| rv4-AIDEV-05 | 3.1 | `FIXED` | `36dba2d` | `facd204` | Deployment risk: a gateway that never sends `message_stop` or `response.completed` now fails every turn, and no real provider was driven (R3-02). The Responses stub in `llm_factory_test.go` lacks `response.completed` and waits about 2 s on a retry (R3-03) |
| rv4-AIDEV-06 | 3.1 | `FIXED` | `8214cba` | `3a0947e` | A dropped parallel call is silent; the Responses forced-tool critic path is unit-tested only (R3-04) |
| rv4-AIDEV-07 | 3.1 | `PARTIAL` | `9a22b75` | `8b3a9b7` | Conversation completion still hangs on the `NO OPEN QUESTIONS` marker line, not on server state; the quote floor (12 characters) and the text-support floor (0.2) are tunables calibrated on 39 hand-written pairs (R3-01) |
| rv4-CLOSE-03 | 3.1 | `FIXED` | `ff94752` | `d0ab02d` | New route `POST /api/v1/conversations/{id}/stage/decline` (human only). A second tab keeps its card and gets a 409; an agent refusal is covered by the route sweep only (R3-04) |
| rv4-LEAD-FENCE-DEDUP (R2-08) | 3.1 | `FIXED` | `f893a32` | `3f7ba17` | none; R2-08 is closed (`agent/prompt_fence.go` now delegates to `agentcontext.FenceData`) |
| rv4-REL-01 | 3.2 | `FIXED` | `66e3434` | `0da4148` | Accepted limits (R3-08): no instance-wide stream cap, one limiter entry per user who ever posted a cursor. A refusal is 429 `APP_STREAM_LIMIT` (20 per user, 100 per project); cursor posts are 25 per second per user |
| rv4-REL-02 | 3.2 | `FIXED` | `618859c` | `87d8ce2` | A plain route that reaches the 25 s deadline answers 500, not 503; seven streaming or long-running routes are exempt (R3-08). `healthz` answers 503 after 2 s |
| rv4-REL-03 | 3.2 | `FIXED` | `227d3bf` | `769366d` | Membership is rechecked on the 10 s heartbeat with a 5 s bound and fails open on a database error, so a removed member can read for up to 10 s (R3-08) |
| rv4-REL-04 | 3.2 | `FIXED` | `2eb8d3d` | `3326cec` | none. The second instance now logs a WARN, retries every 1 s for 2 minutes and logs an INFO on takeover; the ERROR comes only after the window (operator notes) |
| rv4-SEC-08 | 3.2 | `FIXED` | `0c6ae16` | `49f6f25` | A stream now ends at access-token expiry (15 minutes), so it is usable only together with FULL-04, which reopens it (R3-07) |
| rv4-FULL-04 | 3.2 | `FIXED` | `b98b11e` | `32f29f8` | Covered by a Dart VM test with a scripted transport; no browser drives a mid-stream 401 (R3-07) |
| rv4-COMP-04 | 3.2 | `FIXED` | `ac40f8c` | `0aaac09` | `update_prd_section` through the agent has no cap and `get_prd` returns sections whole (R3-05); the web editors have no `maxLength` and rows stored above the cap stay whole (R3-06). The API refuses a section over 10,000 characters with a 400 |
| rv4-SEC-02 | 3.3 | `FIXED` | `7545e09` | `bc42da5`; guard repair `46f252b` | `docker compose config` was not run, docker is not installed here (R3-12). Hosted must set `INSIDEOUT_TRUSTED_PROXY_HOPS=1` before it deploys (R3-13) |
| rv4-SEC-03 | 3.3 | `FIXED` | `aa317b3` | `6ec3d8e`; typo `aacd86a` | Register still answers 409 for a taken address, on purpose; the owner may read SEC-03 as PARTIAL (owner decisions). The mail queue is in memory: a queued mail is lost on exit and a full queue drops the newest (R3-10) |
| rv4-REL-07 | 3.3 | `FIXED` | `aa317b3` | `6ec3d8e` | Same commits as SEC-03. Measured: a known address answered after about 202 ms against 0.9 ms for an unknown one; with the queue the gap is at most 0.06 ms. Same queue limits (R3-10) |
| rv4-SEC-04 | 3.3 | `PARTIAL` | `a73f232` | `48a1896` | A spent refresh token now ends the whole session family after a 10 s grace. `iss`/`aud` claims and a logout denylist in `auth/jwt.go` are open, so an access JWT outlives a family kill by up to 15 minutes (R3-09); a refused refresh still clears the auth cookies, so a browser tab that loses a rotation race can clear the cookie the winner just set (R3-11) |
| rv4-PUB-03 | 3.4 | `PARTIAL` | `4e1f4b1` | `b242e09` | Pending first CI run after push: the gate on a green `ci` run for the exact SHA, the attestation and the no-overwrite publish. `bootstrap.sh` does not yet require https for `INSIDEOUT_RELEASE_BASE` or verify the attestation (R3-14). The version policy and rollback section (PM-05) moved to the proposed `w4-release-version` (R3-18) |
| rv4-PUB-04 | 3.4 | `FIXED` | `bd2bcdd` | `a30f09e` | Packaging was run by hand: four tarballs holding `insideout`, `insideout-mcp`, `LICENSE` and `THIRD_PARTY_NOTICES.md`, the version stamped, four checksums. The workflow run itself is pending first CI run after push |
| rv4-PUB-07 | 3.4 | `FIXED` | `347920b` | `d2620e9`; `331a5f3` | Pending first CI run after push. Seven historical gitleaks findings need an allow-list or a rotation owner (R3-16); the pre-commit redaction hook only warns (R3-17); `ci.yml` is 349 of 350 lines (R3-20, R1-01) |
| rv4-SEC-07 | 3.4 | `PARTIAL` | `4191157` | `47e748c` | Every Action reference is pinned by full commit SHA (23 of 25 were tags); pending first CI run after push. Docker base images are not pinned by digest and dependabot has no docker ecosystem (R3-15); the pins are literals, so a Go or Flutter bump is a lockstep edit (R3-19); `bootstrap.sh` (R3-14) |
| rv4-PUB-09 | 3.4 | `FIXED` | `00c488b`, `1d5bf61` | `d07d8b3`, `2bd00ba` | Go and Flutter archives are checksum-verified before they are unpacked, a wrong Flutter hash stops the whole block, and the CI Go floor comes from `go.mod`; pending first CI run after push. The pins are literals (R3-19) |
| rv4-PUB-02 | 3.5 | `PARTIAL` | `e1f9494` | `b80d01a`, `a4374fe`, `384fb9e` | `PRIVACY.md` is published and linked from the sign-in and sign-up door, but the web link copies the notice address instead of opening a tab; the opener is the proposed `w4-privacy-opener` (R3-21). No guard ties the processor list to the variables (R3-23) |
| rv4-DATA-06 | 3.5 | `DEFERRED_OWNER` | `-` | `c4823d9` (decision record) | Five owner questions in the retention and erasure record: the retention window, the erasure rule, self-service delete, export before erasure, and who is `<privacy-contact>` and how fast they answer. The reaper cannot purge conversations as built (R3-22) |
| rv4-COMP-08 | 3.6 | `FIXED` | `95e0817` | `c96ff59`; `2c116c8` | Every `PARTIAL` promise row has a dated issue and the docs say PostgreSQL 18. Folded `COMP-08-DECISION`: should `in_progress` to `done` need linked evidence or a short reason, enforced in `UpdateRoadmapNode`? An owner question (owner decisions) |
| rv4-FULL-09 | 3.6 | `FIXED` | `3996fea` | `228c41a` | Compose has no pass-through for `INSIDEOUT_AI_TURN_DEADLINE`, so its reference row says compose does not interpolate it (R3-35) |
| rv4-USER-08 | 3.6 | `FIXED` | `80dde55` | `201a8ed` | The client banner was not touched by any cluster; `client/lib/l10n/en_prd.dart` already says replies are echoes, not AI answers (R3-50) |
| rv4-FEAT-08 | 3.6 | `FIXED` | `dba0e13` | `d147465` | none |
| rv4-ARCH-06 | 3.6 | `FIXED` | `ce648d1` | `56eb57e` | New `docs/architecture/cli-mcp-and-client-config.md`; `backend.md` corrected. ARCH-05's point that a fork's binaries aim at the upstream hosted API stays an owner follow-on (R3-38) |
| rv4-PM-03 | 3.6 | `FIXED` | `72d696f` | `1ca978b` | none; HANDOFF and the board state nine open product-promise slices |
| rv4-PM-09 | 3.6 | `PARTIAL` | `3356c20` | `4318627` | The stale Flutter 3.47.0 warning is gone and guarded. Archiving finished rows of the plans board is not done, because the guards read specific board rows and need an edit first (R3-36) |
| rv4-CLOSE-04 | 3.6 | `FIXED` | `b027c5a` | `0970300`; `745f60a` | Ten docs-only ledger rows relabelled Recorded and the counts stated. The ledger guard knows root-level documents by extension only, and the promise register still has DOCS-ONLY rows without a dated issue (R3-37) |
| rv4-CLOSE-04-BOARD | 3.6 | `FIXED` | `80a69cc` | `4d7ea78` | Repair row from round 1: the plans board and the review-3 README restate the recomputed counts. The guard's restated arm is a ceiling (at most 69), not equality with 63 |
| rv4-PM-06 | 3.6 | `FIXED` | `b6bc872` | `156f32d`; pre-gate `4060d64` | The worker reported `PARTIAL`: 42 broken links sat in dated records it did not own. The integrator's pre-gate patch repaired them and set the dated ceilings to 0. Over-budget docs stay on their modularity issue (R3-38) |
| rv4-CLOSE-10 | 3.6 | `FIXED` | `-` | pre-gate `4060d64` | No red commit: the wrong sentence ("existing rows default to en") sat in an unowned file and the migration says `DEFAULT 'zh'`. The patch also recomputed the rv3 index header and the rv3 track outcome |
| rv4-USER-09 (R1-06) | 3.6 | `FIXED` | `583839d` | `9299dc6` | README half: `setup --local` needs a checkout and `promote-admin` creates the account. The `--hosted` blurb in `cli_setup.go` and the "invite code" word in `cli.go` stay R1-07 (R3-46) |
| rv4-PM-08 | 3.6, 3.8 | `FIXED` | `f662ac5`, `7449a58` | `04877ae`, `544c5c8` | One combined disposition: the README half (docs-truth) and the installers and bootstrap messages (first-run-copy) both state macOS and Linux and lead with the prebuilt one-liner. The `--hosted` blurb is R1-07 (R3-46); bootstrap https and attestation are PUB-03 (R3-14) |
| rv4-TEST-02 | 3.7 | `FIXED` | `7855dfc` | `12bbefd` | A test binary that is killed leaves `signup_mode` changed until the value is put back (R3-45) |
| rv4-TEST-03 | 3.7 | `FIXED` | `-` | `9afc391` | No red commit: a coverage task. API coverage 63.0% to 72.3%; 15 of 17 mutations caught and 2 equivalent. Packages still under 50%: GitHub sync and webhook, project guide, conversation messages, roadmap-draft loader (R3-42). Pinned, not fixed: an owner who is not a workspace admin cannot give a project away (API 500, `projects_update` has USING and no WITH CHECK) and nothing guards the last admin (R3-43) |
| rv4-TEST-04 | 3.7 | `PARTIAL` | `-` | `96362a2` | The CLI and the MCP tools are driven through the real API, but 17 of 52 tools are not called and the route-literal check was not written (R3-40) |
| rv4-TEST-06 | 3.7 | `PARTIAL` | `c82a3b9` | `855d621` | `INSIDEOUT_TEST_STRICT` refuses a database that is not on this machine, but nothing sets it yet (`scripts/dev.sh`, CI); 31 test files keep their own `DATABASE_URL` preamble; the check is a host check, not a `*_test` name marker (R3-39) |
| rv4-DATA-07 | 3.7 | `FIXED` | `2b90c56` | `5f2dda2` | none. 18 of 41 foreign keys had no supporting index and now every one has one (migration `20261018210000_index_foreign_keys.sql`) |
| rv4-CLOSE-08 | 3.7 | `PARTIAL` | `677af98` | `c7c792d` | The breaker-row tests are serialized, not cured: `CheckCircuit` and `RecordCircuitResult` are hardwired to the default breaker row, and three agent test files read the row without the lock (R3-41) |
| rv4-TEST-07 | 3.7 | `DEFERRED_OWNER` | `-` | `5e281d1` | A comment only. The replay cutoff stays at `20261006120000` until the owner reads `SELECT max(filename) FROM insideout.schema_migrations` on the hosted database (R3-44) |
| rv4-CLOSE-09 | 3.7 | `DEFERRED_OWNER` | `-` | `5e281d1` | The same owner question. `check_upgrade_cutoff.py` was not written, and moving the cutoff alone turns the upgrade lane red (R3-44) |
| rv4-CLOSE-02 | 3.8 | `FIXED` | `16ec858` | `4088b72`, `c40199a` | `idea convert --locale` takes `en` or `zh` in the CLI and the MCP tool (one variadic `IdeaConvert`); the default comes from `LC_ALL`, `LC_MESSAGES`, `LANG`, else `en`, and is a tunable (R3-49) |
| rv4-FEAT-05 | 3.8 | `PARTIAL` | `16ec858` | `4088b72`, `c40199a` | Shares its commits with CLOSE-02. The export headings in `server/internal/export/render.go` are still English (R3-47) |
| rv4-USER-10 | 3.8 | `FIXED` | `59d6f42` | `46a1a92` | The workspace code and the account invitation have different names. Three client tests outside the cluster changed the word they look for (R3-50). An invitation that can carry a workspace is an API and schema slice (R3-48) |
| rv4-USER-12 | 3.8 | `PARTIAL` | `173c429` | `386fe18` | A closed server's register 403 says registration is closed. The contact route is `DEFERRED_OWNER`; the invitation field is still shown when the mode is closed (R3-48) |
| rv4-LEAD-SETUP-BLURB (R2-06) | 3.8 | `FIXED` | `e6ab582` | `7ba574e` | New guard `scripts/tests/check_first_run_copy.py`. The `--hosted` blurb in `cli_setup.go` and "invite code" in `cli.go` are R1-07 (R3-46) |
| rv4-QUAL-04 | 3.9 | `PARTIAL` | `5bf3bd9` | `4ead866`; test `98f6b3a` | The caller's context now reaches `apiclient`, the MCP handlers and the CLI (exit 130 on Ctrl-C after a 400 ms grace); the CLI half was unproven until the round-1 repair `98f6b3a`, a test only. `runHealthCheck` in `cli_setup_check.go` still builds its own 60 s context and its sign-in check has none (R3-24) |
| rv4-QUAL-06 | 3.9 | `FIXED` | `ac02f14` | `76e4556` | none |
| rv4-REL-06 | 3.9 | `FIXED` | `cdb5fa4` | `707a01b` | Quick calls get 20 s and AI calls 5 minutes. A cut call reads the raw "context deadline exceeded" (R3-26); the AI handlers never read the request body, so a client that leaves is not seen (R3-25) |
| rv4-FULL-07 | 3.9 | `FIXED` | `ae5030d` | `587c0e3` | A node expand now gets the model's deadline too. Same body-read gap (R3-25) |
| rv4-FEAT-04 | 3.10 | `FIXED` | `d87a489` | `8f83c0f` | Commit sync reads with the repo's App installation token and the server no longer reads `GITHUB_TOKEN` for it. `setup --check`, two scripts and a skip message still ask for that key (R3-27); the web client has no text for the new error codes (R3-28) |
| rv4-FEAT-07 | 3.10 | `FIXED` | `f70410a` | `e50d882` | none. Only a successful `deployment_status` is release evidence, so the GitHub App must deliver that event (operator notes) |
| rv4-COMP-05 | 3.10 | `PARTIAL` | `994a71d`, `4eb6ea8` | `27fede5`; pins `ffd55db`, `649844b` | A repo binds to one project, and only once GitHub shows it to the server (migration `20261018180000_github_repo_binding.sql`). `DEFERRED_OWNER`: what proves the linker controls the repo (owner decisions). Renamed or transferred repos, and duplicate bindings that already exist (R3-29) |
| rv4-SEC-06 | 3.10 | `FIXED` | `1aebc35` | `e7356b8` | `FetchGuideFile` refuses dot-only names and a stored refused repo link answers 400. The web client shows generic text for the new codes and for a 403 read as a rate limit (R3-28) |
| R1-12 (firstLine half) | 3.10 | `FIXED` | `1bcbc99` | `e9ead0e` | A commit subject or PR title is cut between characters, at 120. The message-cap half (`conversations.go`, 10,000 bytes) stays open as R1-12 |
| rv4-COMP-03 | 3.11 | `FIXED` | `2e4ac83`, `1859c1a`, `c83ab86`, `525d089` | `b11e200`, `0e8092c`, `b0b260c`; chore `666c3f8` | CanvasKit and a font mirror (872 files, 23 MB) are served from this origin and the CSP names no host. Pending first CI run after push: the web build flag and the nginx checker. `loadWebFonts()` waits up to 10 s for CSS faces CanvasKit ignores (R3-30); nothing compares the mirror to the SDK's font table and nginx sets no long cache for `/fonts/` (R3-31); no hosted build was loaded in a browser (R3-32) |
| rv4-COMP-06 | 3.11 | `FIXED` | `b99500a` | `d7f78f3` | Manifest level only: no Android SDK here, so no APK was built, and a `dart:io` socket may not consult the cleartext config (not proven); a Dart https guard in `session.dart` is not written (R3-32) |
| FEAT-02-CLIENT-APPLY (R2-01) | 2.3, 2.4 | `FIXED` | `6096862` | `0d33077` | The feed's own apply is gone; the server applies once. Leftovers: an unused `projectId`, the unread `ProposalItem` and `ProjectUpdate.items`, the "N nodes added" key, and a failed apply is a 500 (R3-33) |
| rv4-DATA-03 read half (R2-02) | 2.6 | `FIXED` | `8f57d77` | `c57073b` | Migration `20261018200000_users_read_grants.sql` takes `users.password_hash` out of the app role's reads; the rollback `GRANT` is in the fragment. The hosted role was not inspected, and the migration's `DO` assertion fails if `PUBLIC` or an inherited role can still read the column. Ceilings (R3-34) |

Not counted, and not fixed: rv4-PM-04 (red `0de150c`, fix `5320251`) and
rv4-PM-05 (red `7ac78ce`, fix `3b54d25`). The release-supply worker stamped the
version in `install.sh` and the Dockerfile and rewrote `docs/usage/releasing.md`,
none of them files it owned; the verifier returned the cluster and both were
reverted (`c201f92`, `12bc405`). They move to the proposed wave-4
`w4-release-version`, with R1-04, R1-05 and R1-16 (R3-18).
