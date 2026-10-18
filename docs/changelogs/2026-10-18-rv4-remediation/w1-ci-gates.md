# Wave 1, CI gates — review-4 remediation

Branch `rv4/w1-ci-gates`, `1dba3fe..3e70023` (16 commits: seven red/fix pairs
and two guard-hardening commits), then this fragment. The worktree and branch
already existed at the base SHA and were reused. Findings are cited as
`rv4-<ID>` because review IDs restart per review. Rows 1.4, 1.6, 1.11 and the
CI half of 1.15 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md).
No value, host or address is spelled in this fragment.

**Nothing here has run on an Actions runner.** Every claim about a workflow is
**pending first CI run after push**. What was run: the pinned `python3 -I`
guard, a YAML parse, the step scripts by hand with the shell Actions uses,
`go mod tidy -diff` and `flutter analyze` by hand. `actionlint`, `shellcheck`
and `yamllint` are not installed here.

## What changed

- **A weekly shuffled run** (rv4-PM-02; red `e04afa3`, fix `b18e021`, hardened
  `17c3ee2`). `ci.yml` gains `schedule: cron "41 3 * * 1"` (Mondays 03:41 UTC,
  off the :00 and :30 marks where scheduled runs queue). The server job's one
  test step takes `SHUFFLE: ${{ github.event_name == 'schedule' &&
  '-shuffle=on' || '-shuffle=off' }}` and passes `"$SHUFFLE"` to `go test ./...
  -race -count=1 -v`. One suite run, not two, so the 30-minute job timeout is
  not doubled. `-shuffle=on` prints the seed, so a red scheduled run replays
  with `-shuffle=<seed>`. Pushes and pull requests keep the fixed order.
- **Only a pull request's older run is cancelled** (rv4-TEST-08; red `215c01a`,
  fix `2d7878e`). `cancel-in-progress` is `${{ github.event_name ==
  'pull_request' }}`, as the finding asks.
- **A push to main also gets a group of its own** (rv4-TEST-08, second pair;
  red `b0be9bf`, fix `3e70023`). The finding's fix is not enough alone: GitHub
  replaces a run that is waiting in a concurrency group when another is queued
  behind the running one, with `cancel-in-progress` on or off. Pushes A, B, C
  in quick succession on `ci-<ref>` lose B. The group is now `ci-<ref>` for a
  pull request (one per PR, so its older run is still cancelled) and
  `ci-<run id>` for everything else. Cost: quick pushes run side by side; each
  run owns its services and database.
- **The contract guards run in CI** (rv4-PM-01, CLOSE-01, TEST-05, PUB-08: one
  shared pair; red `60c1a65`, fix `b17a849`, hardened `3a76089`). Four roles
  found one gap. The last `env-suites` step loops over
  `scripts/tests/check_*.py` with `python3 -I`, runs every guard and then
  fails if any did, so one run names every red guard. `working-directory: .`
  because the job defaults to `scripts/`, where the glob matches nothing. It is
  a glob: the tree has seven guards (six, plus `check_ci_gates.py`), not the
  five the plan names, and the next guard needs no edit. The same job's
  checkout now sets `fetch-tags: true`: `check_docs_truth.py` asks git for
  `v*` tags and a default shallow checkout has none, so CI would judge a
  different premise than a clone (today both pass; they diverge the day the
  README leads with the release one-liner).
- **The Ubuntu lane accepts Go 1.26 and newer** (rv4-PM-12; red `2aa80a5`, fix
  `d4a5c2e`). The verify grep accepted 1.25 while `server/go.mod` says
  `go 1.26.0`; it is `go1\.(2[6-9]|[3-9][0-9])` now and the step is named
  "Go 1.26+". The skill's Module 3 block installs Go 1.27.1, which matches.
  The finding also suggests adding the workflow to `check_go_floor.py`'s file
  list; that file is outside this cluster, so the same demand lives in
  `check_ci_gates.py`, which reads the floor from `server/go.mod`.
- **The analyzer ratchet is the measured 14** (rv4-PM-07; red `d4510a1`, fix
  `fd0f12b`): `INFO_BASELINE` goes from `"16"` to `"14"`. See the next section
  for why not 0.
- **CI fails on an untidy module** (rv4-PM-10, CI half; red `405b868`, fix
  `6ee8289`). A server-job step `go mod tidy -diff` runs straight after
  `actions/setup-go`, in `server/`. It prints the diff and exits 1 when
  `go mod tidy` would rewrite `go.mod` or `go.sum`; it writes nothing.

## The register's "0 analyzer infos" does not reproduce

rv4-PM-07 says the tree has 0 infos and the ratchet should drop to 0. Three
independent measurements say 14:

- `flutter analyze --no-fatal-infos` here (Flutter 3.44.8, a clean worktree,
  `pub get --enforce-lockfile`): 14 `info •` lines, 14 issues found.
- The last green `ci` run (head `9f0b1c7`, 2026-10-11): `analyzer infos: 14
  (baseline 16)` on the pinned Flutter 3.44.0.
- The `ci` run on `d53035a` (2026-10-18), whose client job passed: the same
  line. Its only red job is the server job, `TestProjectQuietFlagged`, which
  another cluster owns.

A baseline of 0 would turn the client job red at once, so the baseline is 14
and the plan's "0" is recorded here as not reproduced. The prose gate counts in
`docs/plans/README.md` ("analyze 14") were right about this number.

## Design notes

- **A guard that reads text, with the premise from the tree.** The workflows
  are read as text by a stdlib-only `check_ci_gates.py`; no YAML parser is
  needed and a runner may not ship one. The Go floor is read from
  `server/go.mod` and the Ubuntu verify pattern is evaluated against
  `go1.0` to `go1.99` to find the oldest version it accepts, so a floor bump
  fails the guard until the lane follows it. The guard set for tag handling is
  found by scanning `scripts/tests/`.
- **One check runs code.** The `check_*.py` loop is extracted from `ci.yml` and
  run with `bash -e` (what an Actions step gets when no shell is named)
  against a throwaway tree: all guards must run, from the step's effective
  working directory, and a failing guard must fail the step. A gate that
  cannot fail is the bug this row closes, and text matching cannot prove that.
- **`INFO_CEILING = 14` is a ceiling, not an equality.** Lowering
  `INFO_BASELINE` as infos are fixed passes; raising it past the measured count
  fails. Raise the constant only on purpose, with a toolchain bump.
- **Env-based shuffle, not a second suite run.** A second `go test` would
  double the job's runtime. The guard requires the flag to reach the command
  (an earlier version only found the words, and dropping `"$SHUFFLE"` left it
  green; `17c3ee2` closes that).
- **The `dir`-marked rows are not in this cluster.** Rows 1.4, 1.6, 1.11 and
  1.15 are plain diffs, so no design spike was needed.

## Known ceilings

- The guard pins the text of the workflow. The group expression, the schedule,
  `fetch-tags` and every `${{ }}` are evaluated only by Actions.
- The weekly run uses the default branch's latest commit and is the only run
  with `-shuffle=on`; a shuffled failure may be an existing order dependence
  (a finding, not a regression) and the seed is in its log.
- `ci.yml` is 346 lines of the 350 allowed. The client image job (below) does
  not fit without splitting the file first.
- `check_go_floor.py` stays unwired to the workflows it cannot see; the
  cross-check lives in `check_ci_gates.py`.

## Partial and not done

- **rv4-TEST-08, client image job.** The second half of the finding (add a
  `docker build client/` job) is not added: this cluster was told to leave it,
  and a job needs about nine lines, which takes `ci.yml` past 350. Split the
  file first (a reusable workflow for `android` and `env-suites` is the natural
  seam), file the dated modularity issue, then add
  `docker build -t insideout-client-ci client/` beside `server-image`.
- **rv4-PM-10, the module itself.** `go mod tidy -diff` exits 1 on the base
  tree: `go.mod` wants `charmbracelet/huh` in the direct block (1 line moves)
  and `go.sum` wants 18 more lines, more than the finding says. That is a full
  `go mod tidy`, owned by `w1-misc-correctness`. Until it lands, the new step
  is red. On a scratch copy after `go mod tidy` it exits 0.

## Left for the integrator

- `docs/HANDOFF.md`: replace "the head run is green" with a dated SHA (the
  second half of row 1.4; `w3-docs-truth` owns the HANDOFF claims). The last
  green `ci` run is `9f0b1c7` (2026-10-11); the run on `d53035a` is red on the
  server job until `w1-clock-and-runes` lands.
- Any prose that freezes the analyzer count (`docs/HANDOFF.md`,
  `docs/plans/README.md`): "see the last green `ci` run" instead of a number
  (rv4-PM-07's second half, a docs row).
- Re-measure after all of wave 1's client changes land
  (`w1-client-polish` edits client files): `cd client && flutter analyze
  --no-fatal-infos | grep -c -E '^ +info • '`. If it fell, lower
  `INFO_BASELINE`; `INFO_CEILING` in the guard follows on purpose.
- Before the first push, `cd server && go mod tidy -diff` must print nothing
  (needs `w1-misc-correctness`), or the new server step fails the run.
- `docs/changelogs/README.md` and `docs/changelogs/index/2026-10.md`: list this
  fragment; the folder's `index.md` should link it next to `w0-privacy.md`.
- The client image job above, and the modularity issue for `ci.yml`.

## Verification

All checks live in `scripts/tests/check_ci_gates.py` (nine checks, one per
demand above). Every commit was replayed in a scratch checkout from
`git archive`; the guard exits 1 at each red commit and 0 at each fix commit
(the two hardening commits exit 0). The red and green outputs are kept as
`w1-ci-gates-red-<ID>.txt` and `w1-ci-gates-green-<ID>.txt` under
`/private/tmp/insideout-rv4/reports/` (PM-01, CLOSE-01, TEST-05 and PUB-08
share content; the second TEST-08 pair is `…-TEST-08-group.txt`).

Mutation checks a verifier can run on `ci.yml` or `ubuntu-install.yml`, each
turning the guard red (16 mutations plus an unmutated control, all as
expected; the table is `w1-ci-gates-mutations-final.txt`): remove the
`schedule` block, set its minute to 0, change `'schedule'` to `'push'`, drop
`"$SHUFFLE"`, set `cancel-in-progress` to `true`, restore the group
`ci-${{ github.ref }}`, drop `working-directory: .`, make the loop record no
failure, make the step `exit 0`, narrow the glob, drop `fetch-tags`, restore
`2[5-9]` or "Go 1.25+", restore `"16"`, remove the tidy step or add `|| true`.

Gates run in this worktree: `python3 -I scripts/tests/check_ci_gates.py` (9
checks); every `scripts/tests/check_*.py` (all 7 pass, also in a shallow clone
with no tags and in one with the tag); PyYAML parse of both workflows;
`sh scripts/check-redaction.sh`; `sh scripts/check-docs-recorded.sh`;
`ci.yml` 346 lines; `flutter analyze --no-fatal-infos` 14 infos; `go mod tidy
-diff` exit 1 on the base (expected) and 0 on a tidied copy. No Go or Dart file
changed, so no Go or Flutter suite was run for this cluster; the integrator's
full gates cover them.
