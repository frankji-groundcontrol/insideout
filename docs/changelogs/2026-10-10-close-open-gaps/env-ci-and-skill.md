# CI gaps, the env-catalog parity test, and the Ubuntu install skill

Five things that were claimed or assumed but never actually executed.

## The env-suites CI step ran only one of its batteries

`scripts/test_env_catalog.py` and `scripts/test_env_writes.py` are plain-assert
scripts that end in `raise SystemExit(run(globals()))`. The CI step loaded them
through `python3 -m unittest`, so importing the first one ended the whole
process with that script's status: the second battery and the key-store audits
behind it never ran, and the step stayed green. The step (and the two live
docs that copied the command, `docs/usage/local-development.md` and
`skills/insideout-setup/SKILL.md`) now run each battery as a script, then
`unittest discover` over `tests/keys`. Lesson:
[script-style tests are skipped by unittest](../../learning/2026-10-10-script-style-tests-are-skipped-by-unittest.md).

## One schema, two readers: a parity test

`server/internal/setup/catalog` (Go) and `scripts/env_catalog.py` (Python) both
read `.env.example`
([issue](../../issues/2026-10-07-env-catalog-go-python-duplication.md)). The
decision is to keep both readers and make drift fail CI: `parity_test.go` and
`parity_cases_test.go` push one skeleton and one live `.env` through both and
diff the per-variable results (declarations in file order: name, required,
example, default in effect, help text, secret; live state as value and status).
The test needs `python3` and fails without it, since CI treats a skip as a
failure. It runs in the existing `server (go + postgres)` job. It documents its
own scope: Python's `splitlines()`/`strip()` also split on lone CR, VT, FF,
NEL, U+2028/9 and strip Unicode spaces, which the Go side does not, and no rule
in the repo says which is right, so neither was changed.

## The Android job: cache and one retry

`23f6401` failed the android job on a transient Gradle POM fetch while the
other three jobs passed. The job now caches Gradle and retries the APK build
once (with a `::warning::`), so a flaky network is visible but not red. Green
on `0606bdb` and `09c06fe`.

## The install skill's Ubuntu commands run on real Ubuntu

`skills/insideout-setup` had only ever been executed on macOS
([issue](../../issues/2026-10-06-install-skill-ubuntu-unverified.md)).
`.github/workflows/ubuntu-install.yml` now runs the skill's own Ubuntu command
blocks on an `ubuntu-24.04` VM, extracted from `references/modules.md` by
`scripts/tests/skill_blocks.py` so the doc cannot drift from what CI proves. It
runs on pushes and pull requests that touch the skill, `scripts/install.sh`,
`scripts/default-api.sh`, the extractor, or the workflow.

It found two defects before it went green:

1. **The skill's Go block could leave a stale `go` winning.** It appended
   `/usr/local/go/bin` to `PATH`, so an older `go` already present (apt's
   `golang-go` 1.22, which the same section warns about, or a preinstalled
   toolchain) won and `go version` printed the stale one. Fixed in the doc:
   the directory goes first (`0606bdb`).
2. **The workflow asserted the wrong install branch.** `install.sh` skips its
   rc-file edit when `~/.local/bin` is already on `PATH`, which the runner's is.
   The install step now runs under a clean `PATH`, checks the `.bashrc` line,
   and proves a new interactive shell resolves both binaries (`41a5eac`).

Green run (`09c06fe`, Ubuntu 24.04.5 LTS x86_64): Module 1 base tools (git
2.55, curl 8.5, jq 1.7, Python 3.12, OpenSSL 3.0); Module 3 Go 1.27.1 from the
tarball; Module 4 `docker.io` 29.1.3 and `docker-compose-v2` 2.40.3 have
candidates; Module 5 PostgreSQL 16.15 cluster online; Module 6 the pinned
Flutter 3.44.0 archive answers HTTP 200; Step 3A `install.sh` builds both
binaries in about 28 s, appends the PATH line, `insideout setup --hosted`
prints the hosted path, and `whoami` against a closed port reports
`connection refused`.

Deliberately not run on the VM: installing Docker (the runner already ships
Docker CE and `docker.io` beside it would break the daemon), installing Flutter
(a ~700 MB archive; the job checks the URL), and `setup --local`, which the
roleless-cluster and bundled walks in
[setup-roles-and-bundled-owner.md](setup-roles-and-bundled-owner.md) cover.

## The backup runbook

`docs/usage/backup-restore.md` told operators to dump with the app-role DSN,
which `pg_dump` cannot read through row-level security. Walked end to end on
PostgreSQL 18 (dump with the owner DSN, 244 KB; restore into a bare database;
`migrate` a no-op; 23 tables; store and api suites green with no skips) and
corrected to `DATABASE_OWNER_URL`, which `scripts/backup-db.sh` already read.

## Operator notes

None for the CI changes. The skill's Go block change only affects people who
already have an older `go` on `PATH`.
