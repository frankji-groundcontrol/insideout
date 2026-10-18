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

Status 2026-10-18: waves 0 and 1 of the plan (rows 0.1 to 1.18) are integrated
on the local branch `rv4/integration` (base `d53035a`; wave 0 head `97d7529`,
wave 1 integration head `a079404`, then the records commits; nothing pushed)
and recorded in the
[remediation changelog](../changelogs/2026-10-18-rv4-remediation/index.md).
Of the ten Highs wave 1 closed three (rv4-USER-01, rv4-TEST-01, rv4-COMP-02);
the other seven are wave 2. Waves 2 and 3 have not started. The Disposition
column is filled by each wave's integrator with `FIXED` (red and fix commits),
`NOT_REAL`, `DEFERRED_OWNER`, `PARTIAL` or `BLOCKED`, plus the reason.

## Highs and the red test (priority order)

| Register ID | Sev | Priority | Owner | Plan row | Done when | Disposition |
|---|---|---|---|---|---|---|
| rv4-USER-01 | High | P0 | engineering | 1.1 | On a fresh invitation-only database, `promote-admin` creates or promotes the first admin and admin settings answer 200; the red test is in the suite. | `FIXED` (wave 1): red `f30001c`, fix `c913e8e`. |
| rv4-TEST-01 | High | P0 | engineering | 1.5 | `release.yml` runs `go test ./... -race` (or requires a green `ci` run for the tag SHA) before it publishes. | `FIXED` (wave 1): red `0baafaf`, fix `b7db11d`. The release runner has no PostgreSQL service (R1-05) and the step is pending first CI run after push (R1-17). rv4-PM-05 stays `PARTIAL` (Owner decisions). |
| rv4-L1 (ARCH-04, QUAL-05, PM-02) | Medium | P0 | engineering | 1.3, 1.4 | `TestProjectQuietFlagged` passes on any date; `ci.yml` has a weekly schedule; HANDOFF cites a dated SHA. | `FIXED` (wave 1): L1 red `01d64c3`, fix `0a6b5c3`; ARCH-04 red `14dc369`, fix `6172113`; QUAL-05 red `524716a`, fix `6172113`; PM-02 red `8b9cd0b`, fix `57fa71e`, and the dated SHA (`9f0b1c7`) in HANDOFF since `7091e14`. The schedule is pending first CI run (R1-17). The wave-0 baseline failure is gone (gate note below). |
| rv4-FULL-02, rv4-USER-04 | High | P1 | engineering | 2.2 | A CLI login older than the access TTL still runs `whoami`; plain `login` persists its token. | Open, wave 2. |
| rv4-ARCH-01 | High | P1 | engineering | 2.1 | Each verb honours `--api` over config; the MCP binary reads the config `api_base`; one resolver remains. | Open, wave 2. |
| rv4-FEAT-01, rv4-AIDEV-04 | High | P1 | engineering | 2.5 | An agent-scoped token is refused at `commit` and `proposal_decide` server-side; the MCP profile omits `signup_mode`, `invite_*` and deletes. | Open, wave 2. |
| rv4-FEAT-02 | High | P1 | engineering | 2.3 | Accepting a proposal with apply twice leaves one set of roadmap nodes; apply errors surface as 500. | Open, wave 2. |
| rv4-FULL-01 | High | P1 | engineering | 2.4 | Web Expand with AI creates children; a real-shape contract test pins the response. | Open, wave 2. |
| rv4-COMP-02, rv4-COMP-01 | High, Medium | P1 | engineering | 1.16, 1.17 | A deadline picked as 2026-10-18 still reads 2026-10-18 under `TZ=Asia/Shanghai`; a 67-character Chinese title is accepted on every field that has a character limit. | `FIXED` (wave 1): COMP-02 red `5a47205`, fix `137d6f5` (CI runs in UTC; the Shanghai-zone step is R1-01); COMP-01 red `0129059`, fix `e129f02`. |

## Wave 1 dispositions (integrator, 2026-10-18)

One row for each of the 34 findings wave 1 touched: 30 `FIXED`, 4 `PARTIAL`
(rv4-PM-05, rv4-TEST-08, rv4-SEC-06, rv4-USER-09), no `NOT_REAL`, no `BLOCKED`;
the owner halves of rv4-PM-05 and rv4-COMP-02 are `DEFERRED_OWNER`. SHAs are on
`rv4/integration` (the cluster fragments cite the worker-branch SHAs). A
CI-config property can only be checked locally here (`actionlint` is not
installed), so those rows end with "pending first CI run after push" (R1-17). Run
state and the clusters are in the
[remediation changelog](../changelogs/2026-10-18-rv4-remediation/index.md#wave-1);
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
| rv4-USER-07 | 1.10 | `FIXED` | `bee99f7` | `a7f9519` | Create-invitation returns `mailSent` and the dialog says when nothing was e-mailed. The `invite_create` MCP description does not name it yet: R1-03. |
| rv4-QUAL-02 | 1.13 | `FIXED` | `8765fed` | `5cf5bc2` | The resend cooldown keys on `clientIP`, not a drifted copy, so a trailing-empty `X-Forwarded-For` no longer puts every client in one bucket. |
| rv4-QUAL-03 | 1.13 | `FIXED` | `e45229e` | `868e49f` | Every apiclient path segment and query value is escaped (39 rows under a completeness test). `Client.Cursor` passes a nil reply target: R1-13. |
| rv4-SEC-06 | 1.14 | `PARTIAL` | `e646d8e` | `cbd9fe8` | The code half is fixed: dot-only repo segments and foreign hosts are refused and the canonical URL is stored. The token half (a per-installation token, a case-insensitive lookup with a canonicalising migration, `FetchGuideFile` dot names) goes to `w3-github-evidence`, row 3.10: R1-08. |
| rv4-USER-11 | 1.12 | `FIXED` | `b17641a` | `bed39fb` | The Edit-project description field is labelled as a description. |
| rv4-COMP-02 | 1.16 | `FIXED` | `5a47205` | `137d6f5` | The picked deadline is sent as noon UTC of its calendar day, so 2026-10-18 reads 2026-10-18 under `TZ=Asia/Shanghai`. CI runs in UTC and cannot see the bug until the Shanghai-zone step lands: R1-01. Date-typed deadlines and viewer-zone timestamps are `DEFERRED_OWNER`: R1-15. |
| rv4-USER-06 | 1.7 | `FIXED` | `a875428` | `7b6f4c0` | The README `prd update` example puts the id after the flags; root `SKILL.md` was corrected in `7091e14`. The guard does not scan `SKILL.md`: R1-02. |
| rv4-REL-05 | 1.8 | `FIXED` | `092f7f1` | `3a8defb` | The compose `server` has `stop_grace_period: 35s`, explained in the deployment architecture. `docker compose config -q` has not run anywhere: R1-17. |
| rv4-USER-09 | 1.12 | `PARTIAL` | `8161369` | `6f32133` | The installer's closing message no longer offers `setup --local`. `README.md:86` (R1-06) and the `setup --hosted` hint at `cli_setup.go:168` (R1-07) still do; both are wave 3. |
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
Each has an ID, an owner and a one-line fix prompt. The cluster fragments in
the [remediation changelog](../changelogs/2026-10-18-rv4-remediation/index.md)
cite these IDs.

| ID | Left open | Owner | Fix prompt |
|---|---|---|---|
| R1-01 | `ci.yml` has 4 spare lines (346 of 350), so two CI additions did not land: the client image job (rv4-TEST-08, second half) and `TZ=Asia/Shanghai flutter test test/roadmap_deadline_tz_test.dart` (rv4-COMP-02; on a UTC runner that test cannot fail). | The next worker to touch `ci.yml`, after the split in [ci-yml-split](2026-10-18-ci-yml-split.md). | Split `ci.yml` into reusable workflows red-first, then add both items with `check_ci_gates.py` checks. |
| R1-02 | `check_ops_hardening.py` reads `README.md` only for the `prd update` flag order; the root `SKILL.md` had the same defect (corrected in the wave-1 follow-up) and nothing guards it. | Any worker that edits the guards. | Red-first: make the USER-06 check scan `SKILL.md` too, then mutate the file back to id-first and watch it fail. |
| R1-03 | The `invite_create` tool description in `server/internal/mcpserver/product_tools.go` ("response contains the one-time code and signup URL") does not mention `mailSent`. | Wave 2, row 2.5 (same file). | Name `mailSent` in the description (false means nothing was e-mailed) and pin it in an `mcpserver` test. |
| R1-04 | Only `release.yml` stamps `main.version`; `server/Dockerfile` (its `go build` line) and `scripts/install.sh` (lines 29-30) build unstamped, so those binaries print `dev`. | Unowned; owner call on whether `dev` is right for source builds. | Pass `-ldflags "-X main.version=..."` in both (a build arg; `git describe`), or record that `dev` is intended; extend `check_release_gate.py` either way. |
| R1-05 | A release gate stronger than the test step (see Owner decisions). | Owner. | After the decision: a PostgreSQL service in the release job, or a required green `ci` run on the tagged SHA (`actions: read`). |
| R1-06 | `README.md:86` still lists `insideout setup --local --postgres bundled` under "Once installed" (rv4-USER-09, README half), and lines 182 and 186 do not say `promote-admin` creates the account. | `w3-docs-truth`, row 3.8 (the README hunks). | In the row's one README commit: drop or qualify line 86 for prebuilt users; add the create-when-absent sentence. |
| R1-07 | `server/cmd/insideout/cli_setup.go:168`: `setup --hosted` prints "A server on this machine: insideout setup --local ...", which exits 2 for a prebuilt-binary user (no checkout). | `w3-first-run-copy`, row 3.8. | Red-first test on the `--hosted` output; print the hint only when a checkout is found, or say a checkout is needed. |
| R1-08 | rv4-SEC-06, token half: (a) `fetchCommitPage` aims one server-wide `GITHUB_TOKEN` at every linked repo; (b) `_projects_by_repo` is an exact, case-sensitive match and rows stored before the fix are not canonical; (c) `FetchGuideFile` accepts dot-only names. | `w3-github-evidence`, row 3.10 ((b) together with COMP-05). | Per-installation token in `fetchCommitPage` (the webhook path already uses `s.ghTokens`); case-insensitive lookup plus a migration that canonicalises `repo_url`; refuse dot-only names in `FetchGuideFile`. |
| R1-09 | `handleAgentPropose` writes the proposal update before it validates the items, so a refused request (bad title, bad action, an 11th item) leaves an `agent_proposal` on the timeline. | `w2-proposal-apply`, row 2.3 (owns `agent_routes.go`). | Red-first: a refused propose leaves no update; validate, then write. |
| R1-10 | `flagged` on the create, detail, update and repo-link responses is judged from `createdAt` alone (the view is built from `store.ProjectWithLatest{Project: *p}`), while the board judges on the latest update: an old active project with a fresh update reads `flagged: true` on its detail and `false` on the board. | Unowned; file as a finding if wanted. | Red-first route test (old project, fresh update, detail against board); load the latest update into the view in those handlers. |
| R1-11 | The password minimum counts bytes (`len(...) < 8` in `auth.go:74`, `auth_email_flows.go:155` and `promote_admin.go:98`), so three Chinese characters pass "at least 8 characters". | Unowned; a policy call. | Decide bytes or characters; if characters, count runes in all three places with one red test each. |
| R1-12 | `conversations.go` caps a coach message at 10,000 bytes (about 3,333 Chinese characters); `firstLine` in `github_webhook_evidence.go` cuts at 120 bytes and can split a multi-byte character. | Row 3.1 (`conversations.go`); row 3.10 (`firstLine`). | Count runes for the message cap; cut `firstLine` on a rune boundary; one red test each. |
| R1-13 | `Client.Cursor` in `server/internal/apiclient/roadmap.go` passes `nil` as the reply target, so a JSON reply would make it error after the request went out; the route answers 204 today, so it does not bite. | Unowned; row 3.9 touches `apiclient`. | Red-first router test where the cursor route answers 200 with a body; pass a discard target. |
| R1-14 | `scripts/backup-db.sh` has a bash shebang and `set -o pipefail`, so `sh scripts/backup-db.sh` fails under dash (Ubuntu's `sh`); the documented form is `./scripts/backup-db.sh`. | Unowned; low. | Keep the documented form, or make the script POSIX; add a guard that no doc says `sh scripts/backup-db.sh`. |
| R1-15 | Two product questions from `w1-client-polish`: whether the API should take a date-typed deadline (`yyyy-MM-dd`; the PATCH handler parses RFC3339 only) and whether timestamps should show in the viewer's zone (`isoDate` is also `timeAgo`'s date fallback). | Owner (see Owner decisions). | None until decided. |
| R1-16 | `docs/usage/releasing.md` lists the version policy and the rollback as "Not decided yet". | Owner (see Owner decisions). | Write the decision into the page and, if it changes the workflow, into plan row 3.4. |
| R1-17 | Every CI-config property of wave 1 was verified locally only (`actionlint` is not installed): the `release.yml` race-test step (no PostgreSQL service on that runner), the weekly schedule and shuffle (PM-02), the concurrency groups (TEST-08), the guard loop and `fetch-tags` (PM-01, CLOSE-01, TEST-05, PUB-08), the Go regex in `ubuntu-install.yml` (PM-12), the `go mod tidy -diff` step (PM-10) and `INFO_BASELINE` 14 (PM-07). | The first push. | Pending first CI run after push: read the first `ci` run and the first tag run job by job and record each green or red here. |

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
[changelog index](../changelogs/2026-10-18-rv4-remediation/index.md).

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
[changelog index](../changelogs/2026-10-18-rv4-remediation/index.md#gates-integrator-at-a079404).

**Coordinator note (do not redo).** The integrator already applied this
fix-up in `a079404`:

```json
[{"item":"w1-misc-correctness r1 backend-layout patch","path":"docs/architecture/backend.md","what":"added the modtidy/ Layout line between mcpserver/ and presence/ so check_docs_truth.py stays green"}]
```

## Owner decisions (no engineering action until decided)

Dispositions of the first six rows are wave 0's and the last two are wave 1's;
SHAs are on `rv4/integration`, and the privacy halves are written out in
[the owner record](2026-10-18-rv4-owner-privacy-decisions.md).

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

## What closes this record

Rows 0.1 to 2.6 of the plan are done and their checks pass, or each remaining
row has its own dated record. Rows 0.1 to 1.18 are recorded (waves 0 and 1);
the owner halves of 0.1 to 0.4 stay open in
[the owner record](2026-10-18-rv4-owner-privacy-decisions.md), and the wave-1
residue stays open above until each R1 item is done or decided by the owner.
The Medium and Low rows stay in the register and the plan; they do not get
issue files unless a wave opens one.
