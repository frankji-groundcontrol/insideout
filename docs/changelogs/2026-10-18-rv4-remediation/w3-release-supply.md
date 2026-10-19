# Wave 3, release supply chain — review-4 remediation

Branch `rv4/w3-release-supply`, base `8c74218`, `8c74218..cd13201` (17 commits:
eight red/fix pairs and one line-limit follow-up), then this fragment. Findings
are cited as `rv4-<ID>` elsewhere because review IDs restart per review. Row 3.4
of the [remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md)
(PUB-03, PUB-04, PUB-07, SEC-07, PUB-09), plus two carry-forwards from wave 1:
the residual of rv4-PM-04 (R1-04) and the second half of rv4-PM-05 (R1-05,
R1-16). The runbook is [releasing](../../usage/releasing.md).

**Nothing here has run on an Actions runner.** Every claim about a workflow, the
attestation, Dependabot or the Ubuntu steps is **pending first CI run after
push**. What was run: each guard at the red and at the fix commit (from a clean
`git archive` of that commit), a YAML parse of every workflow and
`dependabot.yml`, `actionlint` 1.7.12 on all five workflows (clean), the real
build-and-tar step of `release.yml` for all four targets in a scratch tree,
`install.sh` in a sandbox, the Dockerfile `RUN` line by hand, the skill's
install blocks in a sandbox, and the pinned `govulncheck` and `gitleaks` locally.
Docker and podman are not installed on this machine.

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
  `insideout version` prints, which is why the stamp below matters), the Go
  standard library, the 37 modules linked into `insideout` and `insideout-mcp`
  (a guard fails when `server/go.mod` requires one the file does not name, bar
  four that no target links), and the license texts. `tar` packs `LICENSE THIRD_PARTY_NOTICES.md` next to the two binaries.
- **Token narrowing, Dependabot, vulnerability and secret scanning, key-file
  ignores** (PUB-07; red `3fb217f`, fix `2660df1`, line limit `fb0e1cf`).
  `permissions: contents: read` at the top of `ci.yml` and `ubuntu-install.yml`;
  `smoke.yml` had it, and `release.yml` had `contents: write` at the top, now
  `read` there and elevated in the job. `.github/dependabot.yml`: weekly `gomod` (`/server`),
  `pub` (`/client`) and `github-actions` (`/`) with a 7-day cooldown. New
  `security.yml`: `govulncheck` v1.9.0 and `gitleaks` v8.30.1, each installed by
  `go install` at an exact release, so no third-party Action joins the supply
  chain this work protects; gitleaks scans the commits a push or pull request
  adds, redacted. `.gitignore` gains `*.pem`, `*.key`, `*.p12`.
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
- **`insideout version` is stamped by every build recipe** (rv4-PM-04 residual,
  R1-04; red `14d9377`, fix `7d832e9`). `scripts/install.sh` passes
  `-X main.version=$(git describe --tags --always --dirty)` (`dev` without git);
  `server/Dockerfile` takes `ARG VERSION=dev` after `FROM`. Outside `files_owned`
  by name, done because carry-forward (c) instructs it. `check_release_gate.py`
  now pins all three build paths.
- **The runbook describes the release as it is** (rv4-PM-05 second half; red
  `7718517`, fix `a015ba0`). `docs/usage/releasing.md`: tag rule, cutting a
  release, the pipeline in order, version policy, verifying a download,
  rollback, supply-chain controls, owner decisions. Same carry-forward note on
  `files_owned`. A guard keeps it true to the workflow.

## Evidence

Each row is the guard run in a clean tree of the commit named.

| Row | Red (exit 1) | Green (exit 0) |
| --- | --- | --- |
| PUB-03 | `f63eb35`, `check_release_gate.py`, nine failures: no ancestor gate, `--clobber`, no `--verify-tag`, no attestation, four missing job permissions, shallow checkout | `4b749aa`: `release gate ok: gated on a green ci run, go test -race, no clobber, attested, version-stamped` |
| PUB-04 | `5e3d071`, `check_supply_chain.py`: the tar packs no LICENSE, no `THIRD_PARTY_NOTICES.md`, and the file does not exist | `555dcc3`: `supply chain ok: pub04` |
| PUB-07 | `3fb217f`: no top-level `permissions:` in `ci.yml` and `ubuntu-install.yml`, no `dependabot.yml`, no `security.yml`, no `*.pem`, `*.key`, `*.p12` in `.gitignore` (seven failures) | `2660df1`: `supply chain ok: pub04, pub07` |
| SEC-07 | `6179214`: 23 `uses:` not SHA-pinned (ci 16, release 2, smoke 4, ubuntu-install 1) | `8325f47`: `supply chain ok: pub04, pub07, sec07` |
| PUB-09 | `b03efe4`: the Go and Flutter blocks unpack with no `sha256sum -c` / `shasum -a 256 -c`; `check_go_floor.py` flags the `go1\.(...)` range at `ubuntu-install.yml:67` | `0e2c121`: `supply chain ok: pub04, pub07, sec07, pub09`; `go floor ok: golang:1.26 >= go.mod's go 1.26.0, prose in sync, installer pins go1.27.2, CI derives the floor`; `ci gates ok: 9 checks` |
| PUB-09 (Flutter chain) | `c32ceb3`: `a command follows the && chain that checks flutter_macos_arm64_3.44.0-stable.zip: a mismatch stops the chain but that command still runs` | `cd13201`: `supply chain ok: pub04, pub07, sec07, pub09`; the real block, download stubbed: wrong pin exit 1, nothing unpacked, no rc line |
| PM-04 | `14d9377`: install.sh and the Dockerfile are not stamped and supply no value | `7d832e9`: `release gate ok: ... version-stamped (release.yml, install.sh, Dockerfile)` |
| PM-05 | `7718517`: no `## Version policy`, no `## Rollback`, no `--verify-tag`, no attestation, and the page tells the reader to use `--clobber` | `a015ba0`: `... runbook matches (releasing.md)` |

The SEC-07 red commit message says "23 of 25 Action references". The tree had
**28**: 23 tag-pinned and 5 already SHA-pinned (the attestation step and the four
in `security.yml`). The count of 23 is right; history is not rewritten.

Hand runs, with the shell Actions uses where it applies:

- **Build and tar.** The `release.yml` step, extracted and run at `fb0e1cf` with
  `GITHUB_REF_NAME=v9.9.9-test`: four archives, each listing `insideout`,
  `insideout-mcp`, `LICENSE`, `THIRD_PARTY_NOTICES.md` (`tar -tzf ... | grep -E
  "LICENSE|THIRD_PARTY_NOTICES"` prints both for all four); `shasum -a 256 -c
  checksums.txt` is OK; the extracted darwin-arm64 binary prints
  `insideout v9.9.9-test`.
- **Gate.** The step's own script, extracted from `release.yml`, run with
  `bash -e` in a scratch clone and a stand-in `gh`, seven scenarios: on `main`
  with a green run (exit 0); on `main` with no successful run, with an API error,
  with `null`, with an empty answer (exit 1 each); a tag on a branch-only commit
  with a green run (exit 1, "is not on main"); an annotated tag on `main` with a
  green run (exit 0). Only the two real passes pass; nothing fails open.
- **gitleaks.** The range-selection script, same method, five event shapes: pull
  request `origin/main..HEAD`; push with a usable `before` `before..sha`; push
  with all zeros, push with an unknown `before`, and a manual run all `HEAD~1..HEAD`.
  A scheduled run skips the job by its `if`. The pinned binary over this branch's
  15 commits, with the job's flags: `15 commits scanned ... no leaks found`.
- **Stamp.** `install.sh` from a checkout printed
  `insideout v0.1.0-176-g14d9377-dirty`; from a git-less `git archive` export
  `insideout dev`. The Dockerfile `RUN` line run directly printed
  `insideout v9.9.9-test` with `VERSION=v9.9.9-test` and `insideout dev` with the
  default. The image build itself is exercised by the `server image` job in
  `ci.yml` on the first run.
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
  `--verify-tag`, an `ARG` before `FROM`, an unchecked archive, a `&&`-less check,
  a command left after the Flutter chain, a wrapped "no `--clobber`" in the page.
  Each flips red; the wrapped one stays green, and so does the `|| exit 1` form
  with commands after it.

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
- **The Dockerfile `ARG` sits after `FROM`.** The context is `server/`, which has
  no `.git`, and an `ARG` before `FROM` is empty inside the stage.
- **`ci.yml` is 349 of 350 lines.** The permissions block cost three. R1-01 said
  four spare lines; one is left.

## Known issues and ceilings

- **The first `security` run is red.** `govulncheck` under go1.26.0 (what
  `go-version-file: server/go.mod` installs) reports **29** affecting
  vulnerabilities, measured 2026-10-19 against the live database: 28 in the
  standard library, fixed in go1.26.1 through go1.26.9, and `golang.org/x/text`
  GO-2026-6629, fixed in v0.41.0. With go1.27.2 only the `x/text` one remains.
  The release binaries are built with the same go1.26.0 toolchain. The fix is in
  `server/go.mod`, which this cluster does not own (below). The job is not muted.
- **Full-history `gitleaks`: 7 findings, all `generic-api-key`.** Five are in
  three files still in the tree (two test fixtures and a review ledger; two of
  them report twice), two are in files deleted since. The range scan does not
  see them.
- **`scripts/bootstrap.sh` is not touched** (not in `files_owned`). It still
  verifies only the same-origin `checksums.txt`, accepts any
  `INSIDEOUT_RELEASE_BASE`, and does not run `gh attestation verify`. This is why
  PUB-03 and SEC-07 are `PARTIAL`.
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
- **The hosted image prints `dev`** until a `VERSION` variable is set on the
  Railway `server` service; compose passes none.
- **`files_owned` tension.** `docs/usage/releasing.md`, `server/Dockerfile` and
  `scripts/install.sh` are not listed, and carry-forwards (a) and (c) instruct
  edits to them. They were edited and nothing else outside the list.

## Owner decisions (`DEFERRED_OWNER`)

Questions and click paths are in [releasing](../../usage/releasing.md#owner-decisions).
In short:

- Version policy: what raises MINOR versus PATCH, what makes `v1.0.0`, whether a
  suffixed tag publishes as a pre-release. The page carries a proposed default.
- Settings > General > Releases > "Enable release immutability".
- Settings > Rules > Rulesets > New ruleset > New tag ruleset, target `v*`.
- Settings > Rules > Rulesets > New ruleset > New branch ruleset, target `main`,
  "Require status checks to pass" (`security` only once `govulncheck` is green).
- Settings > Advanced Security (older UI: Code security and analysis): secret
  scanning, push protection, Dependabot alerts, Dependabot security updates.
- The 7 historical findings: allow-list the fixtures in a `.gitignore`-style
  `.gitleaksignore`, and rotate or disable the key behind the deleted-file ones
  (rv4-PUB-05). History is not rewritten by an agent.
- Whether the hosted image should report a version (Railway `server` > Variables
  > `VERSION`).

## Integrator to-do

- INTEGRATOR TODO (pre-gate): `server/go.mod`: add a `toolchain` line of go1.26.9
  or newer and bump `golang.org/x/text` to v0.41.0, then `go mod tidy`. Until
  then the first `security` run is red and the release binaries carry the
  vulnerable standard library.
- INTEGRATOR TODO: `server/cmd/insideout/version.go`: the comment says an
  unstamped build is "go run, scripts/install.sh"; `install.sh` and the
  Dockerfile now stamp, so only `go run`, `go build` and `go test` print `dev`.
- INTEGRATOR TODO: `docker-compose.yml`: optional `args: VERSION:
  ${INSIDEOUT_VERSION:-dev}` on the server build.
- INTEGRATOR TODO: `scripts/bootstrap.sh`: enforce https for
  `INSIDEOUT_RELEASE_BASE` and run `gh attestation verify` when `gh` is present
  (closes PUB-03 and SEC-07).
- INTEGRATOR TODO: `docs/usage/README.md`: the `releasing.md` row now covers the
  gate, test, build, attestation and publish, the version policy, verifying a
  download, rollback and the owner decisions.
- INTEGRATOR TODO: `docs/changelogs/README.md`, `docs/changelogs/index/2026-10.md`
  and `docs/changelogs/2026-10-18-rv4-remediation/index.md`: index this fragment.
- INTEGRATOR TODO: `docs/plans/README.md`, `docs/HANDOFF.md`: row 3.4 state and
  the first-CI-run follow-up (a `v*` tag run proves the gate and the attestation;
  a `security` run proves the scans).
- INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md` and
  `docs/issues/README.md`: R1-04 closed for `install.sh` and the Dockerfile; R1-05
  decided (a green `ci` run); R1-16 answered in the page except the bump rule;
  R1-01 now has 1 spare line, not 4; new open items: the `go.mod` bump, the 7
  historical findings, `bootstrap.sh`, Docker digests; R2-07 (the `DefaultBase`
  stamp) is still open.
- INTEGRATOR TODO: `docs/learning/README.md`: a lesson on three traps met here,
  a failed command inside an `&&` list does not stop a `set -e` script, a Docker
  `ARG` before `FROM` is empty in the stage, and `-X` ignores a symbol that does
  not exist.
