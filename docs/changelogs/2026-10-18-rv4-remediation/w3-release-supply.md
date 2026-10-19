# Wave 3, release supply chain — review-4 remediation

Branch `rv4/w3-release-supply`, base `8c74218`. Round 0 was `8c74218..1b4520e`
(18 commits). Repair round 1 adds two revert commits (`a71af2e`, `1ebae8f`) and
rewrites this fragment: 21 commits in all. Findings are cited as `rv4-<ID>`
elsewhere because review IDs restart per review. Row 3.4 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md):
PUB-03, PUB-04, PUB-07, SEC-07, PUB-09.

**Scope.** rv4-PM-04 (the `insideout version` stamp in `scripts/install.sh` and
the server image) and rv4-PM-05 (the `docs/usage/releasing.md` rewrite) are **not
claims of this cluster**. Round 0 did them by editing three files this cluster
does not own, and the repair handed those edits back. They move to the wave-4
cluster `w4-release-version`, which owns `docs/usage/releasing.md`,
`scripts/install.sh`, `server/Dockerfile` and `scripts/bootstrap.sh` (see
[Moved to wave 4](#moved-to-wave-4)).

**Nothing here has run on an Actions runner.** Every claim about a workflow, the
attestation, Dependabot or the Ubuntu steps is **pending first CI run after
push**. What was run: each guard at the red and at the fix commit (from a clean
`git archive` of that commit), a YAML parse of every workflow and
`dependabot.yml`, `actionlint` 1.7.12 on all five workflows (clean), the real
build-and-tar step of `release.yml` for all four targets in a scratch tree, the
skill's install blocks in a sandbox, and the pinned `govulncheck` and `gitleaks`
locally.

## Repair round 1: scope

The verifier accepted the technical work and failed one check, scope: round 0
edited `docs/usage/releasing.md`, `scripts/install.sh` and `server/Dockerfile`,
none of which is in `files_owned`. The wave-3 carry-forward text had asked for
those edits; the ownership list wins. Two revert commits hand the files back,
each paired with the guard assertions that depended on the file, so each revert
is green on its own (all ten guards pass at `a71af2e` and at `1ebae8f`):

- `a71af2e` `revert(rv4-PM-05)`: restores `docs/usage/releasing.md` to the base,
  drops the page assertions from `check_release_gate.py`, and repoints the one
  comment that named the page's owner section (the `security.yml` header) at this
  fragment, which now carries that section.
- `1ebae8f` `revert(rv4-PM-04)`: restores `scripts/install.sh` and
  `server/Dockerfile` to the base and drops their assertions from the guard.

| Check | Result |
| --- | --- |
| `git diff 8c74218 HEAD -- docs/usage/releasing.md scripts/install.sh server/Dockerfile` | empty: the three files are byte-identical to the base |
| `git diff 4b749aa HEAD -- scripts/tests/check_release_gate.py` | empty: the guard is the one the PUB-03 fix shipped, every `release.yml` assertion intact |
| Differential mutation run: 13 one-line mutations of `release.yml` and an unmutated control (add `--clobber`; drop `--verify-tag`, the ancestor check, the `status=success` filter, the attestation step, `actions: read`, `id-token: write`, `attestations: write`, `fetch-depth: 0` and `-race`; make `go test` and the gate step fail open; unstamp the `./cmd/insideout` build) | each mutation exits 1 under the repaired guard with the same `release.yml:` findings as the round-0 guard (`1b4520e`); the control exits 0. The round-0 guard exits 1 on the control only because it also reads the three restored files, so only the `release.yml:` lines are compared |
| `git diff --name-only 8c74218..HEAD` | 14 paths, all in `files_owned` |

The 14 paths: the five workflows, `.github/dependabot.yml`, `.gitignore`,
`THIRD_PARTY_NOTICES.md`, four guards under `scripts/tests/`, the install skill's
`references/modules.md`, and this fragment.

## Moved to wave 4

`w4-release-version` owns `docs/usage/releasing.md`, `scripts/install.sh`,
`server/Dockerfile` and `scripts/bootstrap.sh`. It takes over:

- **rv4-PM-04, `insideout version` stamped by every build recipe.** Round-0 work,
  still in the branch history: red `14d9377`, fix `7d832e9`. `install.sh` passed
  `-X main.version=$(git describe --tags --always --dirty)` (`dev` without git);
  the Dockerfile took `ARG VERSION=dev` **after** `FROM` (the context is
  `server/`, which has no `.git`, and an `ARG` before `FROM` is empty inside the
  stage). Round-0 hand runs: `install.sh` from a checkout printed
  `insideout v0.1.0-176-g14d9377-dirty`, from a git-less `git archive` export
  `insideout dev`; the Dockerfile `RUN` line printed `insideout v9.9.9-test` with
  `VERSION=v9.9.9-test` and `insideout dev` with the default. The image build
  itself is exercised by the `server image` job in `ci.yml`. Left for the same
  cluster: the optional `args: VERSION: ${INSIDEOUT_VERSION:-dev}` in
  `docker-compose.yml`, the comment in `server/cmd/insideout/version.go` that
  names `scripts/install.sh` as unstamped (true again at the base), and the
  owner question about a `VERSION` variable on the hosted service.
- **rv4-PM-05, the runbook.** Red `7718517`, fix `a015ba0`: tag rule, cutting a
  release, the pipeline in order, version policy, verifying a download,
  rollback, supply-chain controls and owner decisions, with a guard that keeps
  the page true to the workflow. Use `a015ba0` as the lead; the page must then be
  revised to the final `release.yml` and `bootstrap.sh`.
- **The residues of rv4-PUB-03 and rv4-SEC-07 in `scripts/bootstrap.sh`.** It
  still verifies only the same-origin `checksums.txt`, accepts any
  `INSIDEOUT_RELEASE_BASE`, and does not run `gh attestation verify`. No installer
  checks the attestation this cluster publishes until that lands, which is why
  those two rows are `PARTIAL`.

To restore the round-0 work, cherry-pick `14d9377 7d832e9 7718517 a015ba0` in
that order onto the integration head that holds this cluster. Checked in a
scratch clone: the base plus the 14 commits that are not among the six
(`14d9377 7d832e9 7718517 a015ba0 a71af2e 1ebae8f`), then those four originals,
has exactly the tree of `1b4520e`, the round-0 head before this fragment was
rewritten.

## What changed

- **A gated, attested, never-overwritten release** (PUB-03; red `f63eb35`, fix
  `4b749aa`). `release.yml` gains a first step that fails closed unless the
  tagged commit is reachable from the default branch **and** the `ci` workflow
  has a successful run for exactly that SHA. Then `go test ./... -race`, the
  build, an `actions/attest-build-provenance` step over every file in `dist/`,
  and `gh release create --verify-tag --generate-notes` (gh creates a draft,
  uploads, then publishes; an existing release for the tag is an error). The
  `--clobber` upload is gone. The job asks for `actions: read`,
  `contents: write`, `id-token: write`, `attestations: write`; the checkout
  fetches full history.
- **LICENSE, third-party notices and the MPL source offer in every archive**
  (PUB-04; red `5e3d071`, fix `555dcc3`). New `THIRD_PARTY_NOTICES.md`: the
  section 3.2 source offer (it tells the reader to check out the tag
  `insideout version` prints; a release archive prints its tag, stamped by
  `release.yml`), the Go standard library, the 37 modules linked into
  `insideout` and `insideout-mcp` (a guard fails when `server/go.mod` requires
  one the file does not name, bar four that no target links), and the license
  texts. `tar` packs `LICENSE THIRD_PARTY_NOTICES.md` next to the two binaries.
- **Token narrowing, Dependabot, vulnerability and secret scanning, key-file
  ignores** (PUB-07; red `3fb217f`, fix `2660df1`, line limit `fb0e1cf`).
  `permissions: contents: read` at the top of `ci.yml` and `ubuntu-install.yml`;
  `smoke.yml` had it, and `release.yml` had `contents: write` at the top, now
  `read` there and elevated in the job. `.github/dependabot.yml`: weekly `gomod`
  (`/server`), `pub` (`/client`) and `github-actions` (`/`) with a 7-day
  cooldown. New `security.yml`: `govulncheck` v1.9.0 and `gitleaks` v8.30.1, each
  installed by `go install` at an exact release, so no third-party Action joins
  the supply chain this work protects; gitleaks scans the commits a push or pull
  request adds, redacted. `.gitignore` gains `*.pem`, `*.key`, `*.p12`.
- **Every Action pinned by full commit SHA** (SEC-07; red `6179214`, fix
  `8325f47`). All 28 references (ci 16, release 3, security 4, smoke 4,
  ubuntu-install 1) are `@<40 hex> # vX.Y.Z`. The 23 that were tag-pinned were
  resolved read-only with `git ls-remote`, peeled commit for annotated tags;
  seven distinct pins, each re-checked against the tag.
- **The toolchain archives the install skill unpacks are verified** (PUB-09; red
  `b03efe4`, fix `0e2c121`). Module 3 pins Go 1.27.2 with two sha256 values
  (amd64, arm64) and Module 6 pins Flutter 3.44.0 for macOS arm64 (the Intel and
  Ubuntu values sit in comments). The check runs before anything is removed or
  unpacked, and a mismatch stops the block. A second pair (red `c32ceb3`, fix
  `cd13201`) closed a hole in the first fix: the Flutter `PATH` export sat on
  its own line after the `&&` chain, so a wrong hash printed `FAILED`, unpacked
  nothing, **exited 0 and still appended the export to the shell rc file**. The
  export is now the last link of the chain, and the guard requires an `&&` chain
  to be the last statement of its block. The Ubuntu CI check derives the Go
  floor from `server/go.mod` (`cd server && GOTOOLCHAIN=local go list -m`)
  instead of a hand-written release range, and a new CI step compares the
  Flutter pins to the vendor release feeds with `jq` rather than downloading a
  1.5 GB archive.

## Evidence

Each row is the guard run in a clean tree of the commit named.

| Row | Red (exit 1) | Green (exit 0) |
| --- | --- | --- |
| PUB-03 | `f63eb35`, `check_release_gate.py`, nine failures: no ancestor gate, `--clobber`, no `--verify-tag`, no attestation, four missing job permissions, shallow checkout | `4b749aa`, and unchanged at the head: `release gate ok: gated on a green ci run, go test -race, no clobber, attested, version-stamped` |
| PUB-04 | `5e3d071`, `check_supply_chain.py`: the tar packs no LICENSE, no `THIRD_PARTY_NOTICES.md`, and the file does not exist | `555dcc3`: `supply chain ok: pub04` |
| PUB-07 | `3fb217f`: no top-level `permissions:` in `ci.yml` and `ubuntu-install.yml`, no `dependabot.yml`, no `security.yml`, no `*.pem`, `*.key`, `*.p12` in `.gitignore` (seven failures) | `2660df1`: `supply chain ok: pub04, pub07` |
| SEC-07 | `6179214`: 23 `uses:` not SHA-pinned (ci 16, release 2, smoke 4, ubuntu-install 1) | `8325f47`: `supply chain ok: pub04, pub07, sec07` |
| PUB-09 | `b03efe4`: the Go and Flutter blocks unpack with no `sha256sum -c` / `shasum -a 256 -c`; `check_go_floor.py` flags the `go1\.(...)` range at `ubuntu-install.yml:67` | `0e2c121`: `supply chain ok: pub04, pub07, sec07, pub09`; `go floor ok: golang:1.26 >= go.mod's go 1.26.0, prose in sync, installer pins go1.27.2, CI derives the floor`; `ci gates ok: 9 checks` |
| PUB-09 (Flutter chain) | `c32ceb3`: `a command follows the && chain that checks flutter_macos_arm64_3.44.0-stable.zip: a mismatch stops the chain but that command still runs` | `cd13201`: `supply chain ok: pub04, pub07, sec07, pub09`; the real block, download stubbed: wrong pin exit 1, nothing unpacked, no rc line |

The SEC-07 red commit message says "23 of 25 Action references". The tree had
**28**: 23 tag-pinned and 5 already SHA-pinned (the attestation step and the four
in `security.yml`). The count of 23 is right; history is not rewritten.

Hand runs, with the shell Actions uses where it applies:

- **Build and tar**, re-run at the repaired head `1ebae8f`. The `release.yml`
  step, extracted and run in a clean `git archive` export with
  `GITHUB_REF_NAME=v9.9.9-test`: four archives, each listing `insideout`,
  `insideout-mcp`, `LICENSE`, `THIRD_PARTY_NOTICES.md`; `shasum -a 256 -c
  checksums.txt` is OK for all four; the extracted darwin-arm64 binary prints
  `insideout v9.9.9-test`; the notices file in the archive opens its section 3.2
  source offer.
- **Gate.** The step's own script, extracted from `release.yml`, run with
  `bash -e` in a scratch clone and a stand-in `gh`, seven scenarios: on `main`
  with a green run (exit 0); on `main` with no successful run, with an API error,
  with `null`, with an empty answer (exit 1 each); a tag on a branch-only commit
  with a green run (exit 1, "is not on main"); an annotated tag on `main` with a
  green run (exit 0). Only the two real passes pass; nothing fails open.
- **gitleaks.** The range-selection script, same method, five event shapes: pull
  request `origin/main..HEAD`; push with a usable `before` `before..sha`; push
  with all zeros, push with an unknown `before`, and a manual run all
  `HEAD~1..HEAD`. A scheduled run skips the job by its `if`. The pinned binary
  (v8.30.1, built by `go install`) over `8c74218..1ebae8f` with the job's flags:
  `20 commits scanned ... no leaks found`.
- **Hashes.** All five pins were hashed from the downloaded archives, and today
  each matches the vendor feed: the two Go values in `go.dev/dl/?mode=json`, the
  three Flutter values in `releases_macos.json` and `releases_linux.json`.
- **Go block.** Its head (arch case, a real download of the go1.27.2 archive,
  `sha256sum -c`) ran in a sandbox with `set -euo pipefail`, a marker `echo` in
  place of the removal and unpack line. With a one-digit-wrong pin: `FAILED`,
  `go.tgz does not match the pinned sha256: not installing`, exit 1, marker not
  reached. With the right pin: `OK`, marker reached. **The line that removes
  `/usr/local/go` and unpacks was not run on this macOS host**: a safety check of
  the tool blocked a script containing it, and it was not worked around. It is
  unchanged from the base apart from coming after the check; the Ubuntu lane
  runs it.
- **Flutter block.** The real block from `modules.md`, only the `curl` pair
  swapped for a copy of a small archive and `HOME` pointed at a scratch
  directory, run with `bash` and with `zsh`. Right pin: exit 0, unpacked, `PATH`
  line written. Wrong pin: `FAILED`, exit 1, nothing unpacked, no rc line. The
  previous version of the block with the wrong pin: exit 0, nothing unpacked, rc
  line written.
- **Mutations.** The guards were mutation-tested: an unpinned `uses:`, a missing
  `--verify-tag`, an unchecked archive, a `&&`-less check, a command left after
  the Flutter chain. Each flips red; the `|| exit 1` form with commands after it
  stays green, as it should.

## Design decisions

- **A green `ci` run on the tag SHA, not a PostgreSQL service in the job**
  (R1-05, carry-forward (b)). The `DATABASE_URL`-gated suites need the two-role
  setup `ci.yml` builds once; a copy in `release.yml` would drift. A green `ci`
  run also carries the client tests, the install smoke, the redaction guard and
  the contract guards. Cost: `actions: read`, and a tag pushed minutes after a
  merge fails the gate until `ci` is green. **Pending first CI run.**
- **`|| { ...; exit 1; }` for the Go check, one closed `&&` chain for Flutter.**
  Under `set -e` a failed command inside an `&&` list does not stop the script,
  so `sha256sum -c - && sudo rm -rf ... && sudo tar` followed by more lines would
  carry on to the next line. The Go block has more lines after the check, so it
  exits. The Flutter block is a single chain that ends with the `PATH` export,
  with nothing after it, so a mismatch is the exit status of the block. The
  first version left the export outside the chain (`c32ceb3`, `cd13201`).
- **`go install` at a pinned release, not an Action,** for govulncheck and
  gitleaks. gitleaks' module path is `github.com/zricethezav/gitleaks/v8`.
  govulncheck is built with the toolchain it then analyses; an older one refuses
  a newer module.
- **gitleaks scans the added range, not the history.** A full-history run reports
  7 findings (see owner decisions); a whole-history gate would be red forever.
- **`ci.yml` is 349 of 350 lines.** The permissions block cost three. R1-01 said
  four spare lines; one is left.

## Known issues and ceilings

- **Until wave 4 lands, the base behavior is back.** `insideout version` from
  `scripts/install.sh` or from the server image prints `dev`; release archives
  are stamped by `release.yml` and guarded. No guard covers the `install.sh` and
  Dockerfile stamp for now. `docs/usage/releasing.md` is the base page and is
  **stale against `release.yml`**: it says a tag on a commit without a green `ci`
  run can still publish (step 1 and "Before you tag"), that the archive holds
  only the two binaries (step 2), that an existing release is overwritten with
  `--clobber` (step 3), and it lists the overwrite, the rollback rule and a
  stronger gate as undecided ("Not decided yet").
- **The first `security` run is red.** `govulncheck` under go1.26.0 (what
  `go-version-file: server/go.mod` installs) reports **29** affecting
  vulnerabilities, measured 2026-10-19 against the live vulnerability database:
  28 in the standard library, fixed in go1.26.1 through go1.26.9, and
  `golang.org/x/text` GO-2026-6629, fixed in v0.41.0. With go1.27.2 only the
  `x/text` one remains. The release binaries are built with the same go1.26.0
  toolchain. The fix is in `server/go.mod`, which this cluster does not own
  (below). The job is not muted.
- **Full-history `gitleaks`: 7 findings, all `generic-api-key`.** Five are in
  three files still in the tree (two test fixtures and a review ledger; two of
  the three report twice), two are in files deleted since. The range scan does
  not see them.
- **Docker base images** are pinned by tag, not digest, and `dependabot.yml` has
  no `docker` entry.
- **The Homebrew installer line in Module 2** is `curl | bash` from the vendor;
  it is not an archive and was left alone.
- **The Go block's `exit 1`** ends the shell it runs in: right for a script or an
  agent's command, but it closes an interactive terminal it is pasted into. The
  block is Ubuntu-only and the install skill is run by an agent.
- **The pins are literals.** Bumping Go or Flutter is a lockstep edit of
  `modules.md`, `ubuntu-install.yml` and the guards; the Flutter half is checked
  against the vendor feeds on every `ubuntu-install` run.
- **The pre-commit redaction check still only warns** (`config/git-hooks/` is not
  owned). What blocks is the `ci` redaction guard and `gitleaks`.

## Owner decisions (`DEFERRED_OWNER`)

Written down here so nobody assumes an answer; the base `releasing.md` does not
carry them. None is a workflow edit an agent can make.

- **Version policy.** What raises MINOR versus PATCH while the major is 0, what
  makes `v1.0.0`, and whether a tag with a suffix (`v0.2.0-rc.1`) publishes as a
  pre-release. Today every tag publishes as a normal release and takes `latest`;
  a pre-release lane is a `--prerelease` condition in `release.yml`. Proposed
  default, not decided: while the major is 0, a change that breaks a documented
  command, flag, config key or API contract raises MINOR and anything else raises
  PATCH.
- **Immutable releases.** Make a published release's tag and assets unchangeable?
  Repository Settings > General > Releases > "Enable release immutability".
- **Who may tag.** Restrict creating, updating and deleting `v*` tags to the
  maintainers? Settings > Rules > Rulesets > New ruleset > New tag ruleset,
  target `v*`, restrict creations, updates and deletions.
- **Required checks.** Which `ci` and `security` jobs must be green to merge to
  `main`? `security` cannot be required until `govulncheck` is green. Settings >
  Rules > Rulesets > New ruleset > New branch ruleset, target `main`, "Require
  status checks to pass".
- **GitHub secret and dependency features.** Turn on secret scanning, push
  protection, Dependabot alerts and Dependabot security updates? Settings >
  Advanced Security (older UI: Code security and analysis).
- **Old history (rv4-PUB-05).** The 7 findings above. Allow-list the fixtures in
  a `.gitleaksignore`, and rotate or disable the key behind the deleted-file
  ones? History is never rewritten by an agent. The `security` workflow scans only
  the commits a push or pull request adds, so these do not turn it red.
- **Hosted version.** Should the hosted image report a version rather than `dev`?
  Railway service `server` > Variables, add `VERSION` (it takes effect once
  `w4-release-version` puts the `ARG` back in the Dockerfile).

## Integrator to-do

- INTEGRATOR TODO (pre-gate): `server/go.mod`: add a `toolchain` line of go1.26.9
  or newer and bump `golang.org/x/text` to v0.41.0, then `go mod tidy`. Until
  then the first `security` run is red and the release binaries carry the
  vulnerable standard library.
- INTEGRATOR TODO: `docs/usage/releasing.md` is stale against `release.yml` until
  `w4-release-version` rewrites it (the stale claims are listed above).
  `docs/HANDOFF.md` and the docs-truth record should say the page does not
  describe the current workflow, and nobody should cite it as if it did.
- INTEGRATOR TODO: take the branch commits in order. The six commits
  `14d9377 7d832e9 7718517 a015ba0 a71af2e 1ebae8f` net to zero for `install.sh`,
  the Dockerfile, `releasing.md` and the guard, so skipping all six is possible
  (checked in a scratch clone: the tree then differs from the branch head only in
  that one hunk), but then re-apply the `security.yml` comment hunk of `a71af2e`,
  one comment line replaced by two.
- INTEGRATOR TODO (wave 4, `w4-release-version`): `scripts/bootstrap.sh`: enforce
  https for `INSIDEOUT_RELEASE_BASE` and run `gh attestation verify` when `gh` is
  present (closes PUB-03 and SEC-07).
- INTEGRATOR TODO: `docs/changelogs/README.md`, `docs/changelogs/index/2026-10.md`
  and `docs/changelogs/2026-10-18-rv4-remediation/index.md`: index this fragment.
- INTEGRATOR TODO: `docs/plans/README.md`, `docs/HANDOFF.md`: row 3.4 state
  (PUB-04, PUB-07 and PUB-09 fixed; PUB-03 and SEC-07 partial until `bootstrap.sh`
  verifies the attestation), a wave-4 row `w4-release-version` (PM-04, PM-05 and
  the `bootstrap.sh` residues), and the first-CI-run follow-up (a `v*` tag run
  proves the gate and the attestation; a `security` run proves the scans).
- INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md` and
  `docs/issues/README.md`: R1-04, and R1-05 and R1-16 as far as they are page
  text and the bump rule, stay open and move to `w4-release-version` (the R1-05
  decision, a green `ci` run on the tag SHA, is implemented in `release.yml`);
  R1-01 now has 1 spare line, not 4; new open items: the `go.mod` bump, the 7
  historical findings, `bootstrap.sh`, Docker image digests; R2-07 (the
  `DefaultBase` stamp) is still open.
- INTEGRATOR TODO: `docs/learning/README.md`: a lesson on three traps met here,
  a failed command inside an `&&` list does not stop a `set -e` script, a Docker
  `ARG` before `FROM` is empty in the stage, and `-X` ignores a symbol that does
  not exist.
