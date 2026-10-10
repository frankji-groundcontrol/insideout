# BUG-024 — the "env catalog suites" CI step ran one battery of two

## Symptom

CI job `env catalog suites (python)` was green while half of its tests never
ran. `scripts/test_env_catalog.py` and `scripts/test_env_writes.py` are
plain-assert scripts, and the step was:

```
python3 -m unittest test_env_catalog test_env_writes
```

Run locally, that prints `47 passed, 0 failed` and exits 0. The second battery,
`test_env_writes.py`, has 37 checks and none of them executed. A regression in
the `.env` writers would not have turned the job red.

The same wrong command was copied into
[`docs/usage/local-development.md`](../usage/local-development.md) and the
install skill (`skills/insideout-setup/SKILL.md`), so anyone following those
docs got the same half-run.

## Root cause

Both files do their work at import time and end with
`raise SystemExit(run(globals()))`. `unittest` runs a named module by importing
it. Importing the first module ran its checks and raised `SystemExit`, which
ended the whole process with that battery's status before the second module
was ever imported. Neither file defines `unittest.TestCase` classes, so there
was nothing for `unittest` to collect even had the exit not intervened.

## Fix

`b492843`: the step runs each file as a script, as the files' own docstrings
say (`python3 test_env_catalog.py`, then `python3 test_env_writes.py`; the key
suites still go through `unittest discover`), and the command is corrected in
the two docs. The same commit made the Android job
retry its Gradle fetch once and cache Gradle, after one run failed on a
truncated POM from the plugin portal (the same code had passed the run
before).

## Status

**Fixed 2026-10-10.** CI shows both batteries' totals in the step log, and the
job stays green with the second battery now executing. Found while pinning the
Go-versus-Python env-catalog parity test; recorded in
[CI gaps and the Ubuntu install skill](../changelogs/2026-10-10-close-open-gaps/env-ci-and-skill.md).

Lesson: [script-style tests are silently skipped by `unittest`](../learning/2026-10-10-script-style-tests-are-skipped-by-unittest.md)
— count the checks that executed, not only the exit status.
