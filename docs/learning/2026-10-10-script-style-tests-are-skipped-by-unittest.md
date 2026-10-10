# 2026-10-10 — Script-style tests are silently skipped by `unittest`

## Context

`scripts/test_env_catalog.py` and `scripts/test_env_writes.py` are plain-assert
scripts: they define check functions and end with
`raise SystemExit(run(globals()))`. The CI "env catalog suites" step ran them as
`python3 -m unittest test_env_catalog test_env_writes`, which imports each
module. Importing the first one ran its checks and then raised `SystemExit`,
which ended the whole process with that battery's status, so the second battery
(and anything discovered after it) never started. The step stayed green.
Reproduced locally: the old command prints `47 passed, 0 failed` and exits 0;
the second battery has 37 checks that never ran. The same wrong command was
copied into `docs/usage/local-development.md` and the install skill.

## Lesson

A runner that merely *imports* test modules cannot run tests that execute at
import time, and it reports success for the part that did run. A green check
proves nothing about the tests that were never reached. Count the executed
checks, not just the exit status: each battery prints its own total, so the
step should show both lines.

## When to apply

- Any test file whose last line calls `sys.exit`/`raise SystemExit`, or that
  does its work at module level, run through `unittest`, `pytest`, or
  `discover`.
- When adding a second suite to a CI step that already passes: confirm its
  name appears in the step's log.
- Running a documented command is part of reviewing the doc that contains it.
