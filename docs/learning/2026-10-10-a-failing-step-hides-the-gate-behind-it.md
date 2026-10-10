# 2026-10-10 — A failing step hides the gate behind it

## Context

The CI Go job ran `go test ... | tee` under `set -o pipefail`, then a skip
gate in the same step that fails the build if any test skipped. When two
tests failed
([BUG-021](../issues/2026-10-09-bug-021-go-tests-share-one-config-file-on-linux.md)),
the step stopped at `go test` and the gate never ran. The gate itself was
also wrong: two tests skip on a GitHub runner, so it would have failed the
moment the suite went green.

## Lesson

A check that runs after a command that can fail is not verified until that
command has passed once. The first green run is the first run of the later
check, so a red build can be hiding a second, independent failure. When a
pipeline goes red, read the steps behind the failure as well as the failure,
and run each trailing gate on a known log (passing and failing) instead of
trusting it because it has never complained.

## When to apply

- Any CI step that chains a test run and a post-check with `&&`, `pipefail`,
  or `set -e`.
- Before adding a gate to a suite that is currently red: prove the gate on a
  saved green log and on one with an injected violation (here: a
  database-gated skip), so its first real run is not also its first test.
