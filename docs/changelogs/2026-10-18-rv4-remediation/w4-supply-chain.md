# Wave 4, supply-chain guard — review-4 remediation

Branch `rv4/w4-supply-chain`, base `582a4bd`, seven commits (six below, plus the
commit that adds this fragment). One finding, **rv4-PUB-07**: the
`security.yml` guard in `scripts/tests/check_supply_chain.py` could be muted
without going red, and the `server/go.mod` toolchain and `golang.org/x/text`
floor that makes `govulncheck` green was held by nothing. Findings are cited as
`rv4-<ID>` because review IDs restart per review. The change is that guard and
this fragment; `.github/workflows/security.yml`, `server/go.mod` and
`server/go.sum` are untouched (the guard did not need a shape the workflow
lacked, and the go.mod half was fixed by the integrator's `3a1c4ac`, so it has
no red/fix pair here). No database was started.

**Nothing here has run on an Actions runner.** Every claim about what
`security.yml` does on a runner, about which Go a runner selects, and about the
shipped binaries is **pending first CI run after push**. What was run: the guard
at every commit (from a clean `git archive` of that commit), the guard against
muted copies of the real workflow and against the pre-fix `go.mod`, the pinned
`govulncheck` v1.9.0 on this base, and a local emulation of the runner's
toolchain flow (below).

## Commits

- `a5d3035` red: `pub07_mutations` added to the guard; `|| true` after
  govulncheck and gitleaks replaced by `echo` leave it green.
- `af8cbbb` red: five more rows (`|| true` after the install, gitleaks
  `--exit-code 0`, `continue-on-error`, `if: false`, govulncheck from the
  repository root).
- `6d38647` fix: exact scanner lines, the mute bans, the `if:` allowlist and the
  `working-directory: server` shape. All seven rows green.
- `fd356cd` red: `exit 0` after gitleaks, and `set +e` in place of
  `set -euo pipefail`.
- `d2df9d4` fix: `set +e`, `exit 0` and `trap` banned. Both rows green.
- `62452ea` test: `pub07_floor` (toolchain and x/text floors) and its two go.mod
  mutation rows.

## Row 1: a muted security.yml turns the guard red

**Red, on the real file.** The base guard on a copy of the tree with one line of
`security.yml` muted stays green; the guard at `62452ea` goes red on the same
file.

- `run: govulncheck ./...` changed to `run: govulncheck ./... || true`: base
  guard prints `supply chain ok: pub04, pub07, sec07, pub09`, exit 0.
- the `gitleaks git ...` line prefixed with `echo`: same output, exit 0.
- same two files, guard at `62452ea`, exit 1: ``the one `govulncheck` command
  must be exactly `govulncheck ./...`, found ['govulncheck ./... || true']`` and
  ``the one `gitleaks` command must be exactly `gitleaks git --redact --verbose
  --no-banner --exit-code 1 --log-opts="$range" .`, found none``.

**Red, at the red commits** (the guard's own mutation proof, from a clean
`git archive`, exit 1 each):

- `a5d3035`: the guard stays green on a copy with `|| true` after govulncheck;
  the guard stays green on a copy with gitleaks replaced by echo, flags kept.
- `af8cbbb`: all seven rows report "the guard stays green on a copy with ...".
- `fd356cd`: the guard stays green on a copy with `exit 0` after gitleaks; the
  guard stays green on a copy with `set +e` instead of `set -euo pipefail`.

**Green.** `6d38647`, `d2df9d4` and `62452ea` print `supply chain ok: ...`, exit
0. The real `security.yml` yields no finding.

**What the guard pins** (`security_workflow`, the proof is `pub07_mutations`):

1. Each scanner is one exact command line (`SCANS`): `govulncheck ./...` and
   `gitleaks git --redact --verbose --no-banner --exit-code 1
   --log-opts="$range" .`. A flag, pipe, redirect, `echo` in front or an
   `||` list changes the line and is red. Gitleaks runs as a command with `git`,
   `--redact` and the commit range.
2. govulncheck runs from `server/`, where `go.mod` is: `working-directory:
   server` directly above `run: govulncheck ./...`.
3. No step carries an `||` list or `continue-on-error`, and none carries
   `set +e`, `exit 0` or `trap` (`MUTES`).
4. The only `if:` in the file is the secrets job's `github.event_name !=
   'schedule'`; a skipped scan is a muted scan.

**The proof lives in the guard.** `pub07_mutations()` rewrites the real files the
way a hurried edit would and requires the guard to go red on each; a rewrite
that stops applying is itself a failure, so the proof cannot pass vacuously. It
runs from `main()`, in the `env-suites` job with the other guards, with no new
file. Each row and the rule that catches it, measured on the real workflow:

1. `|| true` after govulncheck: exact line, `working-directory` shape, `||` ban.
2. `|| true` after the govulncheck install: `||` ban.
3. gitleaks replaced by `echo`, flags kept: exact line ("found none").
4. gitleaks `--exit-code 0`: exact line.
5. `continue-on-error: true` on the first job: `continue-on-error` ban.
6. `if: false` on the first job: `if:` allowlist.
7. govulncheck run from the repository root: `working-directory` shape.
8. `exit 0` after gitleaks: `exit 0` ban.
9. `set +e` for `set -euo pipefail`: `set +e` ban.

## Row 2: the go.mod floors are held by the guard

`pub07_floor` requires the newer of the `go` and `toolchain` lines of
`server/go.mod` to reach go1.26.9 (the go command runs the newer of the two) and
`golang.org/x/text` to be v0.41.0 or newer. The constants (`TOOLCHAIN_FLOOR`,
`XTEXT_FLOOR`) sit next to the advisory ids they close, with the sentence that
**govulncheck itself is the authority on what is vulnerable and the constants are
bumped on purpose when a security run goes red**. They only make an edit that
drops a floor fail offline, in a second.

Advisories closed (govulncheck v1.9.0 under go1.26.3 on `4060d64`, the parent of
`3a1c4ac`: 19 affecting; w3 recorded 29 under go1.26.0):

- standard library, fixed in go1.26.9: GO-2026-6603, 6605, 6607, 6608, 6610,
  6611, 6612, 6613, 6617;
- standard library, fixed earlier: GO-2026-6088, 6089, 6090, 5972, 6218 and 5026
  (go1.26.6), GO-2026-5856 (go1.26.5), GO-2026-5039 and 5037 (go1.26.4);
- `golang.org/x/text` v0.40.0: GO-2026-6629, fixed in v0.41.0.

**Mutation proof against the pre-fix go.mod.** `4060d64:server/go.mod` has
`go 1.26.0`, no `toolchain` line and `x/text v0.40.0`.

- `pub07_floor` on it: red, two findings: ``neither `go` nor `toolchain` reaches
  go1.26.9`` and ``golang.org/x/text v0.40.0 is below v0.41.0 (GO-2026-6629)``.
- `pub07_floor` on the head `go.mod`: green.
- Whole guard on a scratch tree with that `go.mod` swapped in: the guard at
  `d2df9d4` (no floor check) prints `supply chain ok: pub04, pub07,
  pub07_mutations, sec07, pub09`, exit 0; the guard at `62452ea` exits 1 with the
  two findings above.
- The two go.mod rows in `MUTATIONS` (toolchain line dropped, x/text back at
  v0.40.0) each go red under `pub07_floor`.

**Boundaries** (ad hoc, not committed): `toolchain go1.26.8` red; `go1.26.9`,
`go1.26.10` (compared as numbers, not text), `go1.27.0` and `go 1.26.9` with no
toolchain line green. x/text `v0.40.9` red; `v0.41.0`, `v0.42.0` and a
pseudo-version `v0.41.1-0.2026...` green. The `require` line removed is red with
"no longer requires golang.org/x/text: drop XTEXT_FLOOR", the same way
`NOT_LINKED` forces a clean-up in `pub04`.

## Which Go a runner uses (establishing the setup-go behaviour)

- `actions/setup-go` v5.6.0 (`40f1582b...`, the pin in every workflow) does
  **not** honour the `toolchain` line. `parseGoVersionFile` reads a `go.mod` with
  one regex, `/^go (\d+(\.\d+)*)/m`, so `go-version-file: server/go.mod`
  installs go1.26.0. v6.0.0 (`44694675...`) does read `^toolchain go(...)` unless
  `GOTOOLCHAIN=local`. Both read from the action's `src/installer.ts` at those
  SHAs; the v5.6.0 file was fetched twice with identical SHA-256.
- What makes the `toolchain` line count on a runner is the go command itself:
  `GOTOOLCHAIN` defaults to `auto`, so the first `go` command run inside
  `server/` downloads go1.26.9 (module proxy, cached) and uses it. A `go` command
  at the repository root, which has no `go.mod`, stays on the installed go1.26.0.
- `security.yml` therefore installs govulncheck at the root with go1.26.0 and
  scans from `server/` with go1.26.9. That is why the guard pins
  `working-directory: server`: the module and its `toolchain` line are there, and
  from the root govulncheck exits 1 (`no go.mod file`), which invites a mute as
  the quick fix. `release.yml` and the
  server jobs of `ci.yml` default their steps to `working-directory: server`, and
  `smoke.yml` sets it on the step that runs `go`, so they switch the same way.
  The Docker image and the Ubuntu install check run with `GOTOOLCHAIN=local` and
  ignore the `toolchain` line (they enforce the `go` line, see
  `check_go_floor.py`).
- I changed no `go-version-file` and no workflow.

**Local emulation of that flow** (`GOTOOLCHAIN=go1.26.0+auto`, cached
toolchains): at the repository root `go version` is go1.26.0 and the pinned
govulncheck builds with it (`go version -m` shows go1.26.0); inside `server/`
`go version` is go1.26.9; `govulncheck ./...` exits 0 with "Your code is
affected by 0 vulnerabilities". With the default toolchain (local go1.26.3)
the scan's JSON config reports `go_version: go1.26.9`, `scan_level: symbol`.
This is an emulation on this machine, not a runner: pending first CI run after
push. When the first tag run finishes, `go version -m` on an archive's
`insideout` should print go1.26.9.

## govulncheck on this base (v1.9.0, kept output)

```
=== Symbol Results ===

No vulnerabilities found.

Your code is affected by 0 vulnerabilities.
This scan also found 0 vulnerabilities in packages you import and 4
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
```

Exit 0. The four are informational and all in `golang.org/x/crypto` v0.54.0:
GO-2026-6355 and 6354 (fixed in v0.56.0), GO-2026-6303 (v0.55.0) and
GO-2026-5932 (no fix yet). They do not fail the job. If a call path ever reaches
one, the scan goes red; a Dependabot `gomod` bump to v0.56.0 clears three.

## Known ceilings

- The guard pins the **text** of `security.yml`. It cannot prove runner behaviour
  (the Go selection, the range a push produces, the exit status); the first CI
  run is the proof.
- Not caught: the `range=` assignments (a range that selects nothing, such as
  `HEAD..HEAD`), deleting `set -euo pipefail` and then appending a command,
  shell functions or `PATH` shadowing named `gitleaks`, a `branches:` filter on
  the triggers, and a commit that edits both the workflow and the guard. The
  range logic was exercised by hand in w3; review of a diff that touches both
  files is the control for the last one.
- The `if:` allowlist forbids every other conditional in `security.yml`; adding a
  legitimate one means editing the allowlist on purpose.
- The constants are floors, not proof. A go.mod that meets them can still be
  vulnerable to an advisory published tomorrow; the weekly `security` schedule
  and govulncheck decide, and the constants are bumped when it goes red.
- The shipped image's standard library is whatever `golang:1.26-alpine` holds at
  build time (the official images run `GOTOOLCHAIN=local`); the floor is not
  enforced there. `server/Dockerfile` belongs to `w4-release-version`.

## Gates

Run on the final tree, all exit 0:

- `(cd server && go build ./... && go vet ./...)`; `gofmt -l internal cmd` is
  empty (no Go file changed).
- `check_supply_chain.py`, `check_go_floor.py` and `check_release_gate.py`, and
  all fourteen `scripts/tests/check_*.py` guards.
- `govulncheck` v1.9.0 `./...` from `server/`: "Your code is affected by 0
  vulnerabilities" (output above).
- `sh scripts/check-redaction.sh` and `sh scripts/check-docs-recorded.sh`.
- The 350-line check: no `.go` or `.dart` file changed; the guard is 348 lines
  and this fragment is under 250.

## Integrator to-do

- INTEGRATOR TODO: add this fragment to
  `docs/changelogs/2026-10-18-rv4-remediation/index.md` (a wave-4 row for
  `w4-supply-chain`) and to the month index; record rv4-PUB-07 in the
  open-findings file: guard half `FIXED` (red `a5d3035`, `af8cbbb`, `fd356cd`;
  fix `6d38647`, `d2df9d4`; floor check `62452ea`), go.mod half `FIXED` by
  `3a1c4ac`. The CI-config property stays "pending first CI run after push".
- INTEGRATOR TODO: prose that says `security` is red or cannot be required until
  `govulncheck` is green is stale once this lands. At this base the scan reports
  0 affecting vulnerabilities. The sentences are in `w3-release-supply.md`
  (lines 237, 280 and 298; a dated record, leave it) and, if
  `docs/usage/releasing.md` or the owner-decision text in the open-findings
  record carries the same claim when `w4-release-version` lands, reword it: the
  `security` check can be required after it has run green once on GitHub.
- INTEGRATOR TODO: the `security.yml` header comment ("Built by the toolchain it
  then analyses", "runs with the toolchain server/go.mod selects") overstates the
  patch level: the scanner binary is built by the runner's go1.26.0 and only the
  analysis runs on go1.26.9. Same minor, so nothing breaks. Left as is because
  the guard needed no change to the workflow; reword it in a later pass.
- INTEGRATOR TODO: `docs/usage/local-development.md` already tells contributors
  that the first `go` command in `server/` downloads the toolchain
  (`da93d78`); nothing to add. If Dependabot proposes `actions/setup-go` v6, it
  would make `go-version-file` install go1.26.9 directly; SEC-07 still requires
  the SHA pin and `# vX.Y.Z` comment, and `GOTOOLCHAIN=local` jobs would ignore
  the `toolchain` line again.
