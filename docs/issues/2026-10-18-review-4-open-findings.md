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

Status 2026-10-18: wave 0 of the plan (rows 0.1 to 0.4) is integrated on the
local branch `rv4/integration` (base `d53035a`, head `97d7529`, then the
records commit) and recorded in the
[remediation changelog](../changelogs/2026-10-18-rv4-remediation/index.md).
Waves 1 to 3 have not started. The Disposition column is filled by each wave's
integrator with `FIXED` (red and fix commits), `NOT_REAL`, `DEFERRED_OWNER`,
`PARTIAL` or `BLOCKED`, plus the reason.

## Highs and the red test (priority order)

| Register ID | Sev | Priority | Owner | Plan row | Done when | Disposition |
|---|---|---|---|---|---|---|
| rv4-USER-01 | High | P0 | engineering | 1.1 | On a fresh invitation-only database, `promote-admin` creates or promotes the first admin and admin settings answer 200; the red test is in the suite. | Open, wave 1. |
| rv4-TEST-01 | High | P0 | engineering | 1.5 | `release.yml` runs `go test ./... -race` (or requires a green `ci` run for the tag SHA) before it publishes. | Open, wave 1. |
| rv4-L1 (ARCH-04, QUAL-05, PM-02) | Medium | P0 | engineering | 1.3, 1.4 | `TestProjectQuietFlagged` passes on any date; `ci.yml` has a weekly schedule; HANDOFF cites a dated SHA. | Open, wave 1. The test is red on the wave-0 base too (gate note below). |
| rv4-FULL-02, rv4-USER-04 | High | P1 | engineering | 2.2 | A CLI login older than the access TTL still runs `whoami`; plain `login` persists its token. | Open, wave 2. |
| rv4-ARCH-01 | High | P1 | engineering | 2.1 | Each verb honours `--api` over config; the MCP binary reads the config `api_base`; one resolver remains. | Open, wave 2. |
| rv4-FEAT-01, rv4-AIDEV-04 | High | P1 | engineering | 2.5 | An agent-scoped token is refused at `commit` and `proposal_decide` server-side; the MCP profile omits `signup_mode`, `invite_*` and deletes. | Open, wave 2. |
| rv4-FEAT-02 | High | P1 | engineering | 2.3 | Accepting a proposal with apply twice leaves one set of roadmap nodes; apply errors surface as 500. | Open, wave 2. |
| rv4-FULL-01 | High | P1 | engineering | 2.4 | Web Expand with AI creates children; a real-shape contract test pins the response. | Open, wave 2. |
| rv4-COMP-02, rv4-COMP-01 | High, Medium | P1 | engineering | 1.16, 1.17 | A deadline picked as 2026-10-18 still reads 2026-10-18 under `TZ=Asia/Shanghai`; a 67-character Chinese title is accepted on every field that has a character limit. | Open, wave 1. |

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
row has its own dated record. Rows 0.1 to 0.4 are recorded (wave 0); their
owner halves stay open in [the owner record](2026-10-18-rv4-owner-privacy-decisions.md).
The Medium and Low rows stay in the register and the plan; they do not get
issue files unless a wave opens one.
