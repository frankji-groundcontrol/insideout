# ci.yml is at its line budget, and two CI additions are waiting on it

**Date:** 2026-10-18
**Kind:** structural debt (line-budget split), blocking two review-4 CI items
**File:** [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) — 346
lines against the ~350-line budget (it was 316 before wave 1 of the
[review-4 remediation](../changelogs/2026-10-18-rv4-remediation/index.md)).
**Found by:** [w1-ci-gates](../changelogs/2026-10-18-rv4-remediation/w1-ci-gates.md),
which grew the file by 30 lines (weekly schedule, concurrency group, guard
loop, tidy step) and could not add the two items below.

## Why it matters

`ci.yml` holds seven jobs in one file: `server`, `upgrade-replay`,
`server-image`, `nginx`, `client`, `android` and `env-suites`. Two additions
the review asks for do not fit in the four lines that are left:

- **The client image job** (rv4-TEST-08, second half): `docker build -t
  insideout-client-ci client/` beside `server-image`, so a broken client
  Dockerfile fails CI before a deploy finds it.
- **The deadline test in a zone ahead of UTC** (rv4-COMP-02): after the
  `flutter test` step, run `TZ=Asia/Shanghai flutter test
  test/roadmap_deadline_tz_test.dart`. CI's runner is UTC, where that test
  passes without ever meeting the bug it guards (it goes red only in a zone
  ahead of UTC), so without this step the regression test cannot fail in CI.

Both are workflow edits, so they are outside the records pass that found them.

## Target structure

Keep `ci.yml` as the triggers, the concurrency group and the jobs that gate
a merge (`server`, `client`, `server-image`, `nginx`, and the new client image
job). Move the two self-contained lanes to reusable workflows that `ci.yml`
calls with `uses: ./.github/workflows/<file>`:

- `.github/workflows/ci-android.yml` — the `android` job (`flutter build apk
  --release`, no keystore).
- `.github/workflows/ci-env-suites.yml` — the `env-suites` job, including the
  step that runs every `scripts/tests/check_*.py` guard with `python3 -I` and
  the `fetch-tags: true` checkout.

Nothing else about a job changes; this is a pure move plus a `workflow_call:`
trigger. A called workflow reports its checks as `<caller job> / <job>`, so if
branch protection lists required checks by name, update that list with the
move.

## What the move must not break

Three guards read `ci.yml` by text and must follow the moved jobs, not be
loosened:

- `scripts/tests/check_ci_gates.py` locates the `server`, `client` and
  `env-suites` jobs by name in `ci.yml`; its `env-suites` checks (the guard
  loop, `fetch-tags`) must read `ci-env-suites.yml`.
- `scripts/tests/check_docs_truth.py` requires `flutter build apk --release`
  in `ci.yml` (the android job); it must read `ci-android.yml`.
- `scripts/tests/check_contribution_surface.py` reads the CI lane from
  `ci.yml`.

## Bounded fix prompt

Split `ci.yml` as above, red-first: first change the three guards so they
look for the moved jobs in the new files (they fail), then move the jobs
(they pass). Then add the two items in "Why it matters" with their own red
guard checks in `check_ci_gates.py`. Keep every workflow file at or under 350
lines. `actionlint` is not installed on the author's machines, so the honest
status of the result is "YAML parses, guards pass locally, pending first CI
run after push" — a reusable-workflow call can only be proven by a real run.
