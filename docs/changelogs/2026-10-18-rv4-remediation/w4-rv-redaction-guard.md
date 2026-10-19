# Wave 4, redaction guard — review-4 remediation

Branch `rv4/w4-rv-redaction-guard`, base `582a4bd4` (the integration head after
wave 3). Eleven commits `111ecc67..1754e886`: five fixes, each behind a red
commit that fails for the stated reason (rv4-FR-F45 has two red commits), then
this fragment. Findings are cited as `rv4-<ID>` because review IDs restart per
review. Nothing here spells a host, an address, a secret or a provider id: the
tests build every address from parts at run time, and the one fixture value
`.gitleaks.toml` has to name is described, not quoted. No database was used and
no port was opened.

Nine findings in three groups: the allow rule of `scripts/check-redaction.sh`
(F43, F44, F52, F56, F63), two guards that went quiet instead of saying so
(F45 the ledger history arm, F46 the provider-id scan) and the gitleaks range
scan that the security workflow runs on a push (F53, F55).

## Dispositions

| ID | Status | Red | Fix | Note |
| --- | --- | --- | --- | --- |
| rv4-FR-F43 | `FIXED` | `111ecc67` | `5c861279` | x.com, x.co and b.co are no longer exempt for every local part; 15 exact fixture addresses, only inside Go tests under `server/` |
| rv4-FR-F44 | `FIXED` | `2313d78e` | `caf5c2f3` | only example.com, example.net and example.org (and subdomains) are reserved; example.io and kin fail |
| rv4-FR-F45 | `FIXED` | `6e540ef9`, `9d6be97a` | `6461da50` | the history arm resolves ids through cherry-pick trailers, says when it did not run, fails under `CI`; needs `fetch-depth: 0` in `ci.yml` (INTEGRATOR TODO) |
| rv4-FR-F46 | `FIXED` | `06faf866` | `62a30965` | the provider-id scan reads every tracked file under `docs/`; one named exception (INTEGRATOR TODO) |
| rv4-FR-F52 | `FIXED` | `111ecc67` | `5c861279` | the same hole as F43, probed on two `docs/` pages across all three domains |
| rv4-FR-F53 | `FIXED` | `91347cb1` | `1754e886` | repo `.gitleaks.toml`: default rules plus one path-and-value allowance; pending first CI run |
| rv4-FR-F55 | `FIXED` | `91347cb1` | `1754e886` | the same file; the allowance does not depend on a commit id |
| rv4-FR-F56 | `DEFERRED_OWNER` | `111ecc67` | `5c861279` (guard half) | the guard hole is closed; replacing the 48 fixture occurrences in four server tests is an owner decision in files this cluster does not own (below) |
| rv4-FR-F63 | `FIXED` | `111ecc67` | `5c861279` | the same hole as F43 |

## What changed

### The allow rule of `scripts/check-redaction.sh` (F43, F44, F52, F56, F63)

- **x.com, x.co and b.co were exempt for every local part in every file** (F43,
  F52, F56, F63; red `111ecc67`, fix `5c861279`). The rule was
  `@(x\.com|x\.co|b\.co)$`, so a person's address on any of the three live
  domains committed cleanly in docs, the README, `hosts/*/host.yml` or a test.
  The reviewers' probes are the test's twelve rows plus wrapped forms (angle
  brackets, `mailto:`, parentheses, quotes) and a new address inside a Go test.
  The domains are no longer exempt. What replaces the rule is a list:
  `fixture_addresses()` holds 15 lines of `<local part> <domain>  # <file>`, one
  per address a Go test under `server/` spells today, the test that needs it as
  the comment, the at sign left out so the guard still cannot match itself. A
  `sed | paste` step turns the list into one alternation anchored on
  `^server/<path>_test.go:<line>:(...)$`, so an address passes only if it is on
  the list **and** in a Go test under `server/`. Red: 14 tests, 23 failures
  (`AssertionError: 0 != 1 : the address passed the guard` on each probe, and
  `no fixture list in the guard`). Green: 14 tests OK. A new fixture takes a
  reserved name such as `example.test`; the list only ever shrinks.
- **`example.<any letters>` was treated as reserved** (F44; red `2313d78e`, fix
  `caf5c2f3`). Only example.com, example.net and example.org (RFC 2606), with
  their subdomains, are reserved; example.io, example.ai, example.co and
  example.dev, and subdomains of them, are registrable. The tail
  `example\.[a-z]+$` became `example\.(com|net|org)$`, and the header and the
  report message say so. Red: 16 tests, 6 failures (six registrable names, each
  `0 != 1`). Green: 16 OK.

### Two guards that went quiet (F45, F46)

- **The ledger history arm checked nothing in a shallow clone and said it had**
  (F45; red `6e540ef9` and `9d6be97a`, fix `6461da50`). A depth-1 clone holds
  none of the 65 fix commits the review-3 ledger cites; the arm skipped each and
  the success line still read "labels match their diffs" (red run on a two-row
  ledger: `0 of 2 cited fix commits visible to git, labels match their diffs`,
  exit 0 with and without `CI`). The review's suggestion, fail in CI whenever
  fewer commits are seen than cited, would be red forever and not only on
  shallow clones: the ledger cites pre-integration worker-branch ids, and of its
  63 distinct ids one is reachable from `main`; the other 62 are there only as
  the `(cherry picked from commit ...)` trailer that `git cherry-pick -x` left on
  the integrated copy (checked against `origin/main`). `9d6be97a` pins that
  premise: a full single-branch clone of an integrated history must still see 2
  of 2 and still catch a mislabelled row. The fix does three things. An id git
  cannot show is looked up through that trailer (direct lookup first, trailer
  only as the fallback). The summary says `history arm NOT run on N of M cited
  fix commits (git cannot see them here, e.g. a shallow clone)` when it did not
  run. When `CI` is set, an unseen commit fails the guard with its cure
  (`fetch-depth: 0` on the checkout step). Red: 4 tests with 2 failures, then 6
  tests with 5 failures. Green: 6 OK. On the real ledger: this worktree and a
  full single-branch clone see 65 of 65 and exit 0 also under `CI=true`; a
  depth-1 clone prints `NOT run on 65 of 65` and exits 0 plain, 1 under
  `CI=true`.
- **The provider-id scan named three files** (F46; red `06faf866`, fix
  `62a30965`). `TrackedTree` looked for UUIDs in a hard-coded list of three dated
  records, so a deploy or project id pasted anywhere else under `docs/` passed.
  The scan now walks `git ls-files -z -- docs`, skips binary files the way git
  decides (a NUL in the first 8000 bytes), reports `path:line` and never the id,
  and does not count the nil UUID. Red: ids planted in a dated changelog, a
  `.track.yaml` plan and a usage page all pass (`an unlisted record's provider id
  passed the arm`; 9 tests, 3 failures). Green: 3 `ProviderIdArm` tests OK. On
  the current tree the wider scan finds exactly one spot: the findings register's
  quote of the PUB-12 evidence. That one path is the single named exception
  (`QUOTES_EVIDENCE`), and a second test fails once the exception no longer
  covers a spot, so it cannot outlive the scrub.

### The gitleaks range scan (F53, F55)

- **No gitleaks config, and the range scan reports the `BASE_ENV` fixture**
  (F53, F55; red `91347cb1`, fix `1754e886`). `security.yml` runs
  `gitleaks git --redact ... --log-opts=<range> .`, which reads `.gitleaks.toml`
  from the scan root; the repo had none. With the version the workflow pins
  (v8.30.1, built privately outside the repo) over the range a push of this
  branch scans, `d53035a8..HEAD`: one finding, rule `generic-api-key`, the
  `INSIDEOUT_JWT_SECRET` line of `BASE_ENV` in
  `scripts/tests/keys/database/test_database-owner-dsn.py` (line 44), exit 1
  (`332 commits scanned`, `leaks found: 1`). The value is 48 hex digits made up
  so that `env.sh check` accepts a JWT secret of 32 or more characters; it is
  not a secret. The fix is `.gitleaks.toml`: `[extend] useDefault = true`, so
  every default rule stays on, and one `[[allowlists]]` block with
  `condition = "AND"`, a `paths` entry anchored to that one file and a `regexes`
  entry anchored to that exact value. Both must match, and the comment in the
  file says why. Green: the same range, `333 commits scanned`, `no leaks found`,
  exit 0. Narrowness is tested with the real binary (the same value in another
  file, and a different secret inside the fixture file, are both still
  reported); two static tests check that the config extends the default rules and
  that the one allowance carries its reason. Red: 5 tests, 3 failures; green: 5
  OK.
- **Full history.** A scan of all 1843 commits reports 7 findings with the
  default rules and 5 with this config. The two cleared are the fixture value in
  that one file (it was reported at two commits); the other five are the
  historical findings of R3-16, which this cluster does not touch and which stay
  the owner's.

The three gitleaks tests that run the binary skip when `gitleaks` is not on the
path (the env-suites job has none); the static tests always run. The new tests
live in `scripts/tests/check_rv4_review_redaction.py` (294 lines), a new file
the `check_*.py` glob picks up, because `check_redaction_addresses.py` is at 289
lines.

## Not taken, and why

- **A lower-entropy fixture value** (editing the owned test file). It cannot
  help: gitleaks scans every commit's patch, so a later edit leaves the commit
  that added the line in the scanned range, and the allowance is needed either
  way. The test file stays byte-identical.
- **A `.gitleaksignore` fingerprint, or a commit-id allowlist.** Both embed the
  commit id, and the integrator's `git cherry-pick -x` writes new ids, so the
  entry would stop matching at the moment it is needed. The allowance in
  `.gitleaks.toml` is keyed on path and value, not on a commit.
- **A `gitleaks:allow` comment** on the fixture line: it does not clear the
  earlier commits' patches either.
- **"Seen fewer than cited fails in CI" without the trailer lookup** (the
  review's wording): permanently red, as F45 above shows.

## Decision for the owner (rv4-FR-F56)

After this cluster the guard exempts exactly 15 addresses, 48 occurrences, on
x.com, x.co and b.co, all inside four Go tests that are not owned here:
`server/internal/api/auth_cooldown_sweep_test.go` (9),
`auth_cooldown_test.go` (11), `auth_email_flows_test.go` (3) and
`auth_throttle_test.go` (25). Question: **do these fixtures move to a reserved
name?**

- **(a) Replace them** with a reserved name such as `example.test` and delete
  `fixture_addresses()` with its rule, which removes the last live-domain
  exemption. One mechanical substitution over the four files, but those tests
  need a private PostgreSQL (`go test ./internal/api/...`), so it belongs to a
  cluster that owns the files and can run them. Recommended: the list is an
  exemption someone must keep in step with the tests for good.
- **(b) Keep the exact list.** The guard already refuses any new address on those
  domains, and the list can shrink later.

Nothing was changed for the decision itself.

## Integrator to-dos

- INTEGRATOR TODO: `.github/workflows/ci.yml`: the `env-suites` checkout step
  (the one with `fetch-tags: true`) needs `fetch-depth: 0`; apply it in the same
  step as the F45 commits or the contract-guards step turns red, because the
  ledger guard now fails under `CI` when git cannot see the cited commits.
  `check_ci_gates.py` (`guards_see_the_tags`) already accepts `fetch-depth: 0`.
- INTEGRATOR TODO: `docs/review/2026-10-18-multirole-4/01-findings-register.md`:
  line 130 (the PUB-12 row) quotes a Railway deploy id; replace it with the
  `<deploy-id>` placeholder the row itself asks for, then delete the
  `QUOTES_EVIDENCE` entry from `scripts/tests/check_redaction_addresses.py` (its
  stale-entry test fails until both are done).
- INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md`: R3-16
  lists seven historical gitleaks findings and proposes a `.gitleaksignore`; the
  repo now has `.gitleaks.toml`, which clears two of the seven (the `BASE_ENV`
  fixture) and leaves five for the owner.
- The wave-3 fragment `w3-docs-truth.md` says the history arm only reports in a
  shallow checkout and that CI runs only the git-free arms; that was true before
  this cluster and is left as the dated record it is.

## Verification

| Check | Result |
| --- | --- |
| each finding's tests at its red commit and at its fix commit | fail for the reasons above, then pass |
| `for f in scripts/tests/check_*.py; do python3 -I "$f"; done` (Python 3.9.6) | 15 guards, all exit 0; the new file is 14 OK with the private gitleaks on the path, 14 with 3 skipped without it |
| `sh scripts/check-redaction.sh`, `sh scripts/check-docs-recorded.sh` | exit 0, exit 0 |
| gitleaks v8.30.1 over `d53035a8..HEAD`, the range `security.yml` scans | `no leaks found`, exit 0 |
| `test -z "$(cd server && gofmt -l internal cmd)"`; Go or Dart files over 350 lines | clean; none changed |
| line counts | `check-redaction.sh` 93, `check_redaction_addresses.py` 289, `check_ledger_honesty.py` 153, `check_rv4_review_redaction.py` 294, `.gitleaks.toml` 23 |

## Pending first CI run after push

- The `env-suites` contract-guards step with `fetch-depth: 0`: the ledger history
  arm should print `65 of 65` on the runner. Until the INTEGRATOR TODO is applied
  that step is red by design.
- The `security` workflow's gitleaks job reading `.gitleaks.toml` from the scan
  root. The same pinned version, config discovery and range were run locally; the
  Actions run itself has not happened.
