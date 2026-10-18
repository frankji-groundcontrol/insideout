# Review-4 remediation, wave 0: the privacy scrub

The wave-0 section of the [folder index](index.md), moved here unchanged so
each wave record stays under the line budget. The index lists every wave and
every cluster fragment.

## Wave 0

Cluster `w0-privacy`, plan rows 0.1 to 0.4: documents, `.gitignore`, two
`host.yml` placeholders, the redaction guard and its test; no Go or Dart file.
The verifier accepted it in round 1 with no repair: each of the five red
commits fails (12, 1, 1, 3 and 3 failing checks) and its fix passes, eight
mutations each turned a test red again, and the diff equals the 15 owned
files. The integrator cherry-picked the 13 commits with `-x` onto the contract
commit `2502f46` (`c9216d7..97d7529`, no conflicts); the records follow-ups
are the single commit `3d65d50`. Fragment:
[w0-privacy.md](w0-privacy.md).

### Dispositions

Status is the engineering status; the last column is what stays with the
owner. Every `FIXED` has a red test that turned green for what the guard can
see; the residue it cannot see is under the follow-ups, verified by grep. The
tests were re-run at these red and fix SHAs when this record was written (same
counts).

| Finding | Row | Status | Red | Fix | Left with the owner |
| --- | --- | --- | --- | --- | --- |
| rv4-PUB-01 | 0.1 | FIXED | `c9216d7` | `7df9dfe` | Rotating the hosted admin address: DEFERRED_OWNER |
| rv4-PM-11 | 0.1 | FIXED | `950f0ee` | `98f7c15` | Rotating the owner-domain account address: DEFERRED_OWNER; the release tag's tagger address is history, see rv4-PUB-06 |
| rv4-PUB-05 | 0.2 | DEFERRED_OWNER | `8b80c54` | `23b2617` | The code half (the dead `.gitignore` exception) is FIXED; the dashboard check and the key rotation are the owner's |
| rv4-PUB-10 | 0.3 | FIXED | `8b421b7` | `22f0665` | Example tree or today's layout: DEFERRED_OWNER |
| rv4-PUB-12 | 0.3 | FIXED | `848ad0c` | `c2b5caa` | Whether the deployed host also becomes a placeholder in dated records (see Carry forward) |
| rv4-PUB-06 | 0.4 | DEFERRED_OWNER | n/a | `67da7b2` (the decision record) | Accept the author identities (the default) or rewrite history |

The worker reported PUB-01, PM-11 and PUB-10 as `PARTIAL`: residue sat in
files outside its list. The follow-up commit closed that residue (below), so
the only parts still open are the owner's. Outcome lists for the wave:
`NOT_REAL` none, `BLOCKED` none, `PARTIAL` none, `DEFERRED_OWNER` PUB-05 and
PUB-06 whole plus the owner halves of PUB-01, PM-11 and PUB-10. The owner
halves are in [the owner record](../../issues/2026-10-18-rv4-owner-privacy-decisions.md).

SHAs are on `rv4/integration`. The fragment cites the worker-branch SHAs
(`0f5a02a`/`abcfb08` for PUB-01, `39f4652`/`1891cdd` PM-11, `a9510b1`/`3d7080d`
PUB-05, `c905aa3`/`e253503` PUB-10, `9ecc5ec`/`530eac6` PUB-12, `b5fc986`
PUB-06); each integration commit is patch-identical to its worker commit and
its body names the original in the `cherry-pick -x` line.

### Follow-ups closed in `3d65d50`

Residue the guard cannot see, from the worker's hand-over, in files outside the
cluster's list. All are documentation, comment or index edits, so there is no
red test; each is verified by grep and the guards still pass.

- `docs/HANDOFF.md` (2 lines), `docs/plans/README.md` (1 line) and the quoted
  finding in `26-public-release.md` (1 line): truncated address stems and the
  owner mail domain became `<admin-email>`, `<rehearsal-account>` and
  `<owner-domain>` (rv4-PUB-01, rv4-PM-11).
- `docs/history/nuxt-migration.md` (6 paths on 5 lines) and
  `docs/plans/2026-07-21-prd-agent-harness/research.md` (1 line): the real OS
  user became `/Users/<os-user>/` (rv4-PUB-10).
- `skills/insideout-setup/SKILL.md` describes the `host.yml` placeholders
  (rv4-PUB-10) and `config/git-hooks/pre-commit` says the guard covers any
  non-placeholder address (rv4-PUB-01; a comment-only change). The owner
  record, the fragment and `docs/issues/README.md` were updated to match.

### Gates

On `97d7529`, on a fresh private PostgreSQL (torn down afterwards):

- Cherry-pick: 13 commits, no conflicts. `gofmt`, `go build`, `go vet` clean.
- `go test ./... -race -count=1`: 25 packages ok, one failure (the gate note
  below), one skip (`TestReadSecretTerminal`).
- Flutter: `analyze` 14 infos; 458 tests pass, also under
  `TZ=Asia/Shanghai`; `pubspec.lock` untouched.
- Python: all six `scripts/tests/check_*.py` pass, including the new
  `check_redaction_addresses.py` (9 tests).
- `sh scripts/check-redaction.sh` and `sh scripts/check-docs-recorded.sh` exit
  0 on the whole tracked tree, again after `3d65d50`; the plan's wave-0 check
  (an address scan over `docs/`) prints nothing; `git ls-files hosts` is
  identical to the list taken before the wave.

### Gate note (coordinator, 2026-10-18)

The Go test gate was red on exactly one test, `server/internal/api`
`TestProjectQuietFlagged` (2 subtests: "active with fresh update is not
flagged" and "active recently created with no updates is not flagged"). It
fails identically at the wave base `d53035a` and at the integration head
`97d7529` (the coordinator re-ran it under `env -i`, 2026-10-18). The test pins
`now` to 2026-10-04 while the view builder reads the real clock, which is 14
days later, so fresh fixtures count as stale. `w0-privacy` changes no Go file,
so wave 0 is recorded as green apart from that proven baseline failure. Plan
row 1.3 (wave 1, cluster `w1-clock-and-runes`) fixes it; from wave 1 on the full
suite must be fully green. Source: `coordinator-w0.json` in the remediation
reports directory (outside the repository).

### Carry forward into waves 1 to 3

1. **Fixtures.** The address rule passes `noreply`, `example.*`, the reserved
   TLDs, `resend.dev` and the fixture domains `x.com`, `x.co` and `b.co`
   (tree-wide, not only in the four Go auth tests that use them). A new
   fixture on any other domain (`test.com`, a git SSH remote) fails
   `sh scripts/check-redaction.sh` and CI; use `example.*` names.
2. **Blind spots.** The rule sees whole addresses in text files. Truncated
   stems, bare domains and binary content are invisible, so review still has to
   catch them. The generic OS-user check is
   `git grep -n -I -E '/Users/[A-Za-z0-9._-]+/'`, which prints nothing today.
3. **CI shell.** The guard uses extended regular expressions only and ran
   under `/bin/dash` and bash 3.2 with BSD `grep`; **pending first CI run after
   push** under ubuntu `sh` and GNU `grep`. `ci.yml` already runs
   `sh check-redaction.sh`.
4. **A sixth guard.** Plan row 1.6 lists five `check_*.py` guards by name; the
   tree now has six. `w1-ci-gates` should loop over `scripts/tests/check_*.py`
   so `check_redaction_addresses.py` is not left out. Done in wave 1 (`352dbc2`):
   the loop globs the folder, so the three wave-1 guards joined it too.
5. **The ssh alias test.** It flags the `ssh_alias` field followed by a colon
   and anything but a `<placeholder>` in `docs/` or `hosts/`, code spans
   included; in prose write the field name without the colon.
6. **No UUID rule.** PUB-12 is pinned to the three named records by a test; a
   tree-wide rule would fire on legitimate ids in fixtures, seed SQL and
   project config.
7. **The deployed host.** Row 0.3 also says "deployed host"; the cluster
   replaced the ids only. The host is the shipped default (public by design)
   and the guard's host rule exempts dated records on purpose, so it stays
   unless the owner asks for a placeholder (detail in the fragment).

### Operator notes

Nothing to run or deploy: no migration, dependency or configuration change,
and the hook change is a comment. The hosted admin address and the other owner
items hold their defaults until the owner records a decision in
[the owner record](../../issues/2026-10-18-rv4-owner-privacy-decisions.md).
