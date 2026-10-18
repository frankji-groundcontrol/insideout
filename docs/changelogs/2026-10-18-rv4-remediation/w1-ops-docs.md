# Wave 1, ops docs — review-4 remediation

Branch `rv4/w1-ops-docs`, base `1dba3fe`, eight commits (four red/fix pairs)
and then this fragment. Findings are cited as `rv4-<ID>` because review IDs
restart per review. Rows 1.7, 1.8, the bootstrap half of 1.12 and 1.18 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md).
No Go or Dart file changed. No secret, DSN or address is spelled here.

## What changed

- **The README `prd update` example parses** (rv4-USER-06, red `c586b21`, fix
  `9a942cf`). Go's `flag` package stops at the first positional, so
  `prd update <prd-id> --section ...` never parsed the flag and printed only the
  usage line. The example now reads `prd update --section goals="Ship the MVP"
  <prd-id>`, the order of the CLI's own usage line and of
  `docs/usage/cli-mcp-and-guidance.md`. One line of `README.md`.
- **The compose server gets time to drain** (rv4-REL-05, red `f6b8241`, fix
  `fd49e78`). `runServe` grants `Shutdown` 30 s (a 20 s drain plus the coach's
  10 s terminal writes); Docker's 10 s default stop timeout SIGKILLed the
  process in the middle of it. `stop_grace_period: 35s` on the `server` service
  only (the window plus a 5 s margin), with a comment that names the reason.
  `docs/architecture/deployment.md` explains it and says any other runtime
  needs an equivalent.
- **The prebuilt installer no longer offers `setup --local`** (rv4-USER-09,
  red `d35fb01`, fix `73b22f9`). The installer ships two binaries and no
  repository checkout, and that mode exits 2 without one. The line is gone from
  the closing message of `scripts/bootstrap.sh`; the interactive, `--hosted`
  and `--check` lines stay.
- **The backup dump is owner-only** (rv4-COMP-07, red `fc6489a`, fix `4320c2f`).
  `scripts/backup-db.sh` now sets `umask 077` at the top and runs
  `chmod 600 "$file"` after `pg_dump`. Both are needed: the umask covers a new
  file and a new output directory (0700), while `pg_dump` into an existing
  0644 file keeps that mode, so only the chmod narrows a leftover; a failed
  chmod aborts under `set -e`. `docs/usage/backup-restore.md` now names what
  the dump holds (password hashes, refresh-token hashes, emailed-token hashes,
  invitation-code hashes, and the workspace join codes in the clear) and its
  0600 mode.

## The guard

`scripts/tests/check_ops_hardening.py`, stdlib only, run as
`python3 -I scripts/tests/check_ops_hardening.py`. Four checks read file text,
so it needs no Docker, database or network. Each derives its premise from the
tree first and the demand disappears with the premise (the style of
`check_docs_truth.py`): the CLI's own `prd update` usage line (USER-06), the
`shutdownCtx` timeout in `server/cmd/insideout/main.go` (REL-05), the CLI's
"no repository checkout found" refusal (USER-09), and the `*_hash` columns in
the migrations (COMP-07). It exits 1 naming every offending property and 2
when a premise cannot be derived. It matches `check_*.py`, so the loop that
`w1-ci-gates` adds for row 1.6 picks it up.

Mutation sweep of the guard: 24 variants, each on a scratch copy of the tree
beside the unmutated tree, every one returning the expected exit code.

- REL-05: the value at 20 s fails, and so does 30 s (equal to the window is not
  enough); `1m` and a quoted `"35s"` pass; commented out, malformed, set only
  on the `postgres` service, and the deployment-doc mention removed all fail.
- COMP-07: the umask removed, only commented, set to 022 or moved after the
  dump; the chmod removed, only commented, set to 644 or moved before the dump;
  and each of the three doc claims removed all fail. The `pg_dump` call removed
  gives exit 2. With no `*_hash` column in the migrations, the unfixed script
  and doc pass: the demand disappears with its premise.
- USER-09: the `setup --local` line restored and the `--check` line removed
  both fail.
- USER-06: the id moved back before the flag fails.

## Verification

| Finding | Red at the red commit | Green at HEAD |
| --- | --- | --- |
| rv4-USER-06 | `README.md:170: ... puts the id before a flag` (exit 1) | `ops hardening ok` (exit 0) |
| rv4-REL-05 | two messages: no `stop_grace_period` on the server; not in the deployment doc | ok |
| rv4-USER-09 | `scripts/bootstrap.sh:127: the closing message offers setup --local` | ok |
| rv4-COMP-07 | five messages: no umask, no chmod, three doc claims missing | ok |

By hand, because the guard only reads text:

- **USER-06.** The built CLI under `env -i` against a dead localhost port: with
  the id first it exits 2 and prints only the usage line; with the flags first
  it reaches the network (exit 1, connection refused).
- **REL-05.** `docker` is not installed on this machine, so
  `docker compose config -q` did not run. The file was parsed with a YAML
  parser instead: `stop_grace_period` is `'35s'` on `server` and absent from
  `postgres` and `client`.
- **USER-09.** The real `scripts/bootstrap.sh` ran under `env -i` with a
  throwaway home against a local file-based release (checksum verified, both
  binaries installed): the closing message has no `setup --local`. The built
  CLI's `setup --local` outside a checkout exits 2 ("no repository checkout
  found").
- **COMP-07.** Against a private PostgreSQL 18 cluster (migrated, torn down
  afterwards), with fake credential rows seeded in the `insideout` schema only:
  the base script wrote the dump 0644 under umask 022 and 0664 under 002; the
  fixed script writes it 0600 under 022, 002 and 000 when run as
  `sh scripts/backup-db.sh` or with `bash`, a new output directory is 0700, an
  existing 0755 directory is left alone, and the dump still carries every
  seeded row the runbook names.

**Pending first CI run after push.** Nothing here has run on a runner. The
guard runs in CI only once the `check_*.py` loop from row 1.6 lands, and
`docker compose config -q` has not run anywhere.

## Left open, with owners

1. **The dialog half of row 1.12** (rv4-USER-11, the Edit-project label) belongs
   to `w1-client-polish`.
2. **The README half of rv4-USER-09.** `README.md:86` still lists
   `insideout setup --local --postgres bundled` under "Once installed". It is a
   wave-3 README hunk (row 3.8, `w3-docs-truth`).
3. **The CLI prints the same hint.** After `insideout setup --hosted`,
   `server/cmd/insideout/cli_setup.go:168` prints the `setup --local` line, so a
   prebuilt-binary user still sees it. Outside this cluster's files; a
   candidate for the row 3.8 code cluster (`w3-first-run-copy`).
4. **A second wrong-order `prd update`.** The root `SKILL.md:69-70` writes
   `prd update <id> --section goals=...`, the defect of rv4-USER-06 in the
   agent-facing skill. No wave-1 cluster owns that file. The fix is the
   two-line edit `prd update --section goals=… <id>`; the guard's USER-06 check
   should then scan `SKILL.md` as well as the README (it reads the README only,
   because adding the file now would leave the guard red).
5. **`sh` under dash.** `scripts/backup-db.sh` has a bash shebang and `set -o
   pipefail`, so `sh scripts/backup-db.sh` fails under dash (Ubuntu's `sh`)
   before and after this change. The documented form is
   `./scripts/backup-db.sh`; it was not changed.

## Operator notes

No migration, dependency or configuration change. The new grace period applies
from the next stop or recreate of the server container. Dumps taken before
this change keep their old mode: run `chmod 600 backups/*.sql` and
`chmod 700 backups` once on any machine that has them.
