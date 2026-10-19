# Review-4 remediation, wave 1: the front door, the gates and correctness

The wave-1 section of the [folder index](index.md), moved here unchanged so
each wave record stays under the line budget. Three edits were made in the
move: R1-03 and R1-09 are marked closed in wave 2 (see [wave 2](wave-2.md)),
and the `cli_setup.go` line citation was refreshed after wave 2 moved the line.

## Wave 1

Nine clusters, plan rows 1.1 to 1.18. The integrator cherry-picked 74 commits
with `-x` onto the wave-0 records commit `1dba3fe` (`f30001c` through `bfd5094`, no
conflicts, no baseline failure) and applied one coordinator fix-up (`a079404`,
below). The records follow-ups are `7091e14`; this section and the per-ID state
are the commit titled `docs(rv4): wave 1 dispositions`. Eight verifiers
accepted in round 1; `w1-misc-correctness` was sent back once (its
`go mod tidy` gate needed a `go.sum` change the cluster spec had forbidden) and
accepted in round 2. SHAs are on `rv4/integration`; the fragments cite the
worker-branch SHAs, and each integration commit is patch-identical to its
worker commit (its body names the original in the `cherry-pick -x` line).

Outcome lists (34 findings): `FIXED` 30; `PARTIAL` 4 (rv4-PM-05, rv4-TEST-08,
rv4-SEC-06, rv4-USER-09); `NOT_REAL` none; `BLOCKED` none. `DEFERRED_OWNER` is
the owner half of two findings: rv4-PM-05 (the version policy, the rollback
rule, a stronger release gate) and rv4-COMP-02 (date-typed deadlines,
viewer-zone timestamps), recorded as R1-05, R1-16 and R1-15.

### Clusters

| Cluster | Rows | What landed | Verifier | Fragment |
| --- | --- | --- | --- | --- |
| `w1-cli-front-door` | 1.1, 1.2, 1.5 | `promote-admin` creates the first admin (password from stdin); `help`, `-h`, `--help`, `version`, `--version` and unknown verbs answer before `config.Load()` (exit 0, 0, 2); `release.yml` runs `go test ./... -race` first and stamps the tag into `main.version` | round 1 | [w1-cli-front-door.md](w1-cli-front-door.md) |
| `w1-ci-gates` | 1.4, 1.6, 1.11, CI half of 1.15 | weekly shuffled scheduled run; in-progress cancel for pull requests only; every `scripts/tests/check_*.py` guard in the `env-suites` job (`python3 -I`, `fetch-tags: true`); Ubuntu install lane accepts Go 1.26 and newer; analyzer baseline 14; a `go mod tidy -diff` step | round 1 | [w1-ci-gates.md](w1-ci-gates.md) |
| `w1-clock-and-runes` | 1.3, 1.17 | `Server.now()` in the project, roadmap node and progress views plus a guard against raw `time.Now()` there; one `tooLong` helper counting characters on the 18 capped fields | round 1 | [w1-clock-and-runes.md](w1-clock-and-runes.md) |
| `w1-auth-atomic` | 1.9, 1.10, cooldown half of 1.13 | a login-code guess is charged and compared in one atomic `UPDATE`; create-invitation returns `mailSent` and the dialog says when nothing was e-mailed; the resend cooldown keys on `clientIP` | round 1 | [w1-auth-atomic.md](w1-auth-atomic.md) |
| `w1-apiclient-escape` | apiclient half of 1.13 | every apiclient path segment and query value is escaped (39 rows under a completeness test) | round 1 | [w1-apiclient-escape.md](w1-apiclient-escape.md) |
| `w1-github-repo` | 1.14 | dot-only repo segments and foreign hosts refused; the canonical repo URL is stored | round 1 | [w1-github-repo.md](w1-github-repo.md) |
| `w1-client-polish` | 1.16, dialog half of 1.12 | the Edit-project description field is labelled as a description; the picked deadline is sent as noon UTC of its calendar day | round 1 | [w1-client-polish.md](w1-client-polish.md) |
| `w1-ops-docs` | 1.7, 1.8, 1.18, bootstrap half of 1.12 | the README `prd update` example parses; compose `stop_grace_period: 35s`; the prebuilt installer stops offering `setup --local`; the backup dump is written 0600 and documented | round 1 | [w1-ops-docs.md](w1-ops-docs.md) |
| `w1-misc-correctness` | 1.15 without its CI step | `charmbracelet/huh` marked direct and `go.sum` tidied; `items` declared in the propose tool schema; a stable `openQuestions` order; an empty owner DSN fails `env.sh check` on the bundled PostgreSQL; the timeline cursor survives a deleted row | repair, then round 2 | [w1-misc-correctness.md](w1-misc-correctness.md) |

### Dispositions

Status is the engineering status. Every `FIXED` has a red test that failed
before its fix and passes after it (the verifiers replayed each pair at the
worker SHAs; the integrator ran the whole suite at the head). A CI-config
property can only be checked locally, because `actionlint` is not installed on
the author's machines: those rows say **pending first CI run after push**
(R1-17). The R1 IDs are the rows of the wave-1 residue table in the
[open-findings record](../../issues/2026-10-18-review-4-open-findings.md#wave-1-open-residue-integrator-2026-10-18);
the one-line dispositions are in the
[dispositions record](../../issues/2026-10-18-review-4-dispositions.md#wave-1-dispositions-integrator-2026-10-18).

| Finding | Cluster | Row | Status | Red | Fix | Left open |
| --- | --- | --- | --- | --- | --- | --- |
| rv4-USER-01 | cli-front-door | 1.1 | FIXED | `f30001c` | `c913e8e` | The password minimum counts bytes (R1-11) |
| rv4-USER-02, rv4-FULL-03 | cli-front-door | 1.2 | FIXED | `e0908b3` | `abc19a0` | none |
| rv4-PM-04 | cli-front-door | 1.2 | FIXED | `e0908b3`, `0baafaf` | `abc19a0`, `b7db11d` | The container image and `scripts/install.sh` build unstamped and print `dev` (R1-04, owner) |
| rv4-TEST-01 | cli-front-door | 1.5 | FIXED | `0baafaf` | `b7db11d` | The release runner has no PostgreSQL service, so the `DATABASE_URL`-gated suites skip there (R1-05, owner); pending first CI run (R1-17) |
| rv4-PM-05 | cli-front-door | 1.5 | PARTIAL | `0baafaf` | `b7db11d` | The test gate is done with TEST-01 and [releasing](../../usage/releasing.md) is written (`7091e14`); the version policy, the rollback rule and a stronger gate are `DEFERRED_OWNER` (R1-05, R1-16) |
| rv4-PM-02 | ci-gates | 1.4 | FIXED | `8b9cd0b` | `57fa71e`, `5f52381` | The HANDOFF half landed in `7091e14` (the worker's `PARTIAL` was only that records edit); the schedule is pending first CI run (R1-17) |
| rv4-TEST-08 | ci-gates | 1.4 | PARTIAL | `735ba99`, `4609a63` | `adf2177`, `cc534e7` | The concurrency half is done; the client image job did not fit in `ci.yml` (R1-01, [ci-yml-split](../../issues/2026-10-18-ci-yml-split.md)) |
| rv4-PM-01, rv4-CLOSE-01, rv4-TEST-05, rv4-PUB-08 | ci-gates | 1.6 | FIXED | `9112dc6` | `352dbc2`, `b410302` | The guard loop is pending first CI run (R1-17); `check_ops_hardening.py` does not scan root `SKILL.md` (R1-02) |
| rv4-PM-12 | ci-gates | 1.11 | FIXED | `1f3c74a` | `b2aa936` | The Go regex in `ubuntu-install.yml` is pending first CI run (R1-17) |
| rv4-PM-07 | ci-gates | 1.11 | FIXED | `3ad4022` | `acf57c0` | The register's "0 analyzer infos" does not reproduce: 14 are measured. The red test pins the real defect, a ratchet looser than the baseline (it allowed 16); `INFO_BASELINE` is now 14 |
| rv4-PM-10 | ci-gates, misc-correctness | 1.15 | FIXED | `c47e3ec`, `622bf58`; CI step `4c55a55` | `5b642bd`, `7951036`; CI step `3996134` | `ci-gates` found its tidy step would fail on an untidy `go.sum`; `misc-correctness` marked `huh` direct and tidied `go.sum`, so both halves landed. The step is pending first CI run (R1-17) |
| rv4-L1 | clock-and-runes | 1.3 | FIXED | `01d64c3` | `0a6b5c3` | `flagged` on detail responses is judged from `createdAt` alone, unlike the board (R1-10) |
| rv4-ARCH-04 | clock-and-runes | 1.3 | FIXED | `14dc369` | `6172113` (clock in `0a6b5c3`) | none |
| rv4-QUAL-05 | clock-and-runes | 1.3 | FIXED | `524716a` | `6172113` | none |
| rv4-COMP-01 | clock-and-runes | 1.17 | FIXED | `0129059` | `e129f02` | Byte caps outside the sweep: the coach message (10,000 bytes) and `firstLine` (R1-12) |
| rv4-SEC-01 | auth-atomic | 1.9 | FIXED | `b4edce1` | `4df79d6` | none |
| rv4-USER-07 | auth-atomic | 1.10 | FIXED | `bee99f7` | `a7f9519` | The `invite_create` MCP tool description does not name `mailSent` (R1-03, wave 2 row 2.5; closed in wave 2 by `f9e9894`) |
| rv4-QUAL-02 | auth-atomic | 1.13 | FIXED | `8765fed` | `5cf5bc2` | none |
| rv4-QUAL-03 | apiclient-escape | 1.13 | FIXED | `e45229e` | `868e49f` | `Client.Cursor` passes a nil reply target (R1-13, observation) |
| rv4-SEC-06 | github-repo | 1.14 | PARTIAL | `e646d8e` | `cbd9fe8` | The code half is done (dot segments, host, canonical URL); the per-installation token, the case-insensitive lookup with its migration and `FetchGuideFile` dot names go to `w3-github-evidence` (R1-08) |
| rv4-USER-11 | client-polish | 1.12 | FIXED | `b17641a` | `bed39fb` | none |
| rv4-COMP-02 | client-polish | 1.16 | FIXED | `5a47205` | `137d6f5` | CI runs in UTC and cannot see this bug until the Shanghai-zone step lands (R1-01); date-typed deadlines and viewer-zone timestamps are `DEFERRED_OWNER` (R1-15) |
| rv4-USER-06 | ops-docs | 1.7 | FIXED | `a875428` | `7b6f4c0` | Root `SKILL.md` corrected in `7091e14`; the guard does not scan it (R1-02) |
| rv4-REL-05 | ops-docs | 1.8 | FIXED | `092f7f1` | `3a8defb` | `docker compose config -q` has not run anywhere (R1-17) |
| rv4-USER-09 | ops-docs | 1.12 | PARTIAL | `8161369` | `6f32133` | The installer banner is done; `README.md:86` (R1-06) and the `setup --hosted` hint at `cli_setup.go:148` (R1-07) still advertise the command, both wave 3 |
| rv4-COMP-07 | ops-docs | 1.18 | FIXED | `ef63477` | `074d4e0` | `sh scripts/backup-db.sh` fails under dash (R1-14); dumps taken earlier keep their mode (operator notes) |
| rv4-AIDEV-10 | misc-correctness | 1.15 | FIXED | `4d04a6d` | `9da2086` | none |
| rv4-QUAL-01 | misc-correctness | 1.15 | FIXED | `9deac14` | `d40c64b` | none |
| rv4-FULL-06 | misc-correctness | 1.15 | FIXED | `4fe72b2` | `ded3836` | none |
| rv4-DATA-05 | misc-correctness | 1.15 | FIXED | `174c764` | `d671860` | none |

Incidental: `promote-admin` now trims and lowercases the address, the
promote-admin half of the Low finding rv4-DATA-10.
`TestPromoteAdminNormalisesTheAddress` arrived with the fix and has no red of
its own, so DATA-10 is not claimed here; it stays with the hygiene batch.

### Gates (integrator, at `a079404`)

On a fresh private PostgreSQL that was torn down afterwards:

- `gofmt -l .` prints nothing; `go build ./...`, `go vet ./...` and
  `go mod tidy -diff` (in `server/`) are clean.
- `go test ./... -race -count=1`: 27 packages ok, 0 failures; the only skip is
  `TestReadSecretTerminal` (it needs a pty). `TestProjectQuietFlagged` and its
  threshold-boundary rows pass, so the wave-0 baseline failure is gone.
- Flutter: `pub get --enforce-lockfile` ok (`pubspec.lock` unchanged); `analyze`
  14 infos (equals `INFO_BASELINE` 14); 466 tests pass, also under
  `TZ=Asia/Shanghai`.
- All nine `scripts/tests/check_*.py` guards exit 0; `sh scripts/check-redaction.sh`
  and `sh scripts/check-docs-recorded.sh` exit 0.
- The plan's wave-1 check: `env -i insideout --help`, `-h` and `version` exit 0
  with empty stderr; a build with `-ldflags -X main.version=...` reports the
  value; `go test` appears once in `release.yml`; the 67-character title is
  covered by the `w1-clock-and-runes` tests in the full suite.
- **Pending first CI run after push** (never run on a runner; `actionlint` is
  not installed): the `ci.yml` schedule, shuffle, concurrency groups, guard
  loop, `fetch-tags` and tidy step; the `release.yml` race-test step; the
  `ubuntu-install.yml` Go regex; `docker compose config -q` (R1-17).

After the two records commits the build, vet, `gofmt`, `go mod tidy -diff`, the
guards and the link check of the touched files were run again and are green;
the only Go edit in them is a comment.

### Coordinator note (2026-10-18)

The integrator already applied this fix-up in `a079404`; do not redo it:

```json
[{"item":"w1-misc-correctness r1 backend-layout patch","path":"docs/architecture/backend.md","what":"added the modtidy/ Layout line between mcpserver/ and presence/ so check_docs_truth.py stays green"}]
```

### Follow-ups closed in `7091e14`

Residue from the worker hand-overs that is only a document, a comment or an
index edit (29 files). No product code, test, script, workflow or migration
changed; each item was read back and the guards pass.

| Closed | Where | Finding |
| --- | --- | --- |
| HANDOFF cites the last green `ci` run (`9f0b1c7`, 2026-10-11), says the `d53035a` run is red on the server job, and says waves 0 and 1 are integrated | `docs/HANDOFF.md` | rv4-PM-02 |
| The review-3 row stops freezing gate counts | `docs/plans/README.md` | rv4-PM-07 |
| `promote-admin` creates the account when absent | `docs/usage/installation.md` (twice), `docs/usage/local-development.md`, `docs/SETENV.md`, `docs/architecture/backend.md`, `skills/insideout-setup/SKILL.md` | rv4-USER-01 |
| `help` and `version` rows, `mailSent`, `promote-admin`, the accepted `repo set` forms | `docs/usage/cli-mcp-and-guidance.md` | rv4-USER-02, rv4-USER-07 |
| The `mailSent` sentence; `prd update --section goals=… <id>` (flags before the id) | root `SKILL.md` | rv4-USER-07, rv4-USER-06 |
| A comment on `CreateInvitation` naming `mailSent` (comment only) | `server/internal/apiclient/auth.go` | rv4-USER-07 |
| An empty `DATABASE_OWNER_URL` fails `env.sh check` on the bundled PostgreSQL | `docs/usage/environment.md`, `docs/usage/keys/database/database-owner-dsn.md` | rv4-FULL-06 |
| The deadline picker sends noon UTC; the `s.now()` clock and the `tooLong` helper | `docs/architecture/product-subsystems.md`, `docs/architecture/backend.md` | rv4-COMP-02, rv4-L1, rv4-COMP-01 |
| "Fixed in" lines and the shipped shape | `docs/learning/2026-10-18-a-wall-clock-test-needs-a-scheduled-run.md` | rv4-L1 |
| New: the release process as it is, with the open decisions; `ci.yml` at 346 of 350 lines | [releasing](../../usage/releasing.md), [ci-yml-split](../../issues/2026-10-18-ci-yml-split.md) | rv4-PM-05, rv4-TEST-08 |

Also: the analyzer reports 14 infos after the client change, equal to
`INFO_BASELINE`, so no edit; the nine fragments each gained a "Closed" paragraph;
and the rest became the residue table R1-01 to R1-17 in the open-findings
record, each with an owner and a one-line fix prompt (product code, tests,
executing scripts and workflows are outside a records pass).

### Carry forward into waves 2 and 3

1. **The first CI run (R1-17).** After the branch is pushed, read the first
   `ci` run and the first tag run job by job and record each green or red in
   the open-findings record, together with the wave-0 item (the redaction guard
   under Ubuntu `sh` and GNU `grep`).
2. **`ci.yml` is at 346 of 350 lines.** Any further CI edit starts with
   [the split](../../issues/2026-10-18-ci-yml-split.md), which also unblocks R1-01.
3. **Residue with a later owner.** R1-03 (row 2.5; closed in wave 2 by
   `f9e9894`), R1-09 (row 2.3; closed by `bfe019e`), R1-06 and R1-07 (row 3.8),
   R1-08 (row 3.10), R1-12 (rows 3.1 and 3.10), R1-13 (row 3.9): a worker on
   those rows reads its R1 line first.
4. **Guards stay runnable under `python3 -I`.** A `check_*.py` added to
   `scripts/tests/` joins the CI loop on its own; one that needs a third-party
   module or the network breaks the job.
5. **`releasing.md` and row 3.4.** The page says the workflow overwrites the
   assets of an existing tag with `--clobber`; row 3.4 stops that, and the page
   changes in the same row.
6. **HANDOFF and the plans board.** The "Everything engineering is closed"
   paragraph of `docs/HANDOFF.md` and the docs-order row of `docs/plans/README.md`
   ("zero broken doc links") are unchanged; `w3-docs-truth` (row 3.6) refreshes
   them. Wave-0 items 1 to 3 and 5 to 7 still hold.

### Operator notes

No migration and no new dependency version (`go.mod` marks `huh` direct and
`go.sum` gains 18 hashes of its test-only dependencies). What a running install
sees:

- `insideout promote-admin <email>` creates the account when it is absent and
  reads a new password from stdin (never from a second argument); an existing
  account is promoted and its password is not touched. Without
  `DATABASE_OWNER_URL` the promotion cannot apply, and the command says so.
- `insideout help` and `insideout version` answer without any configuration.
- The compose `server` gets `stop_grace_period: 35s` at its next stop or recreate.
- New backup dumps are written 0600 in a 0700 directory. Earlier dumps keep
  their mode: run `chmod 600 backups/*.sql` and `chmod 700 backups` once on any
  machine that has them.
- The create-invitation response gains `mailSent`; `false` means nothing was
  e-mailed, and the dialog shows the link to copy.
- `repo_url` values stored before the fix keep their old form; the
  case-insensitive lookup and the migration that canonicalises them are R1-08.
- A tag now runs `go test ./... -race` before any build, so a red commit no
  longer publishes (not yet run on a runner).
- `scripts/tests/check_release_gate.py` arrived as mode 100644 (it was 100755);
  the guards run through `python3`, so nothing changes.

Nothing is pushed, merged or deployed.
