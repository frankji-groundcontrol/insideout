# 2026-10-10 — Run a setup doc on its own platform in CI

## Context

The install skill's Ubuntu commands had never been executed on Ubuntu; every one
was "check-then-choose" prose that macOS testing could not exercise. A workflow
that extracts the skill's own command blocks (`scripts/tests/skill_blocks.py`)
and runs them on an `ubuntu-24.04` VM found two defects in its first runs, one
in the doc and one in the test:

1. The Go block appended `/usr/local/go/bin` to `PATH`. An older `go` already on
   the machine (apt's 1.22, or a preinstalled toolchain) won, and the doc's own
   `go version` check printed the stale version. The fix is ordering: prepend.
2. The workflow asserted that `install.sh` edits `~/.bashrc`, but the script
   skips that edit when `~/.local/bin` is already on `PATH`, which the runner's
   is. The branch a fresh user hits (clean `PATH`) was the one not exercised.

A later independent review found that the workflow could not fail on the thing
it extracts. It ran each block as `bash -c "$(helper N)"`: when the helper
fails (a renamed heading in the skill), `$(...)` yields an empty string, bash
runs an empty script and exits 0, and two modules were checked only by tools
the runner already had. A Flutter archive check that piped `curl` into `head`
passed on a 404 for the same reason (no `pipefail` in the default step shell).
Fixed in `b25bce1`.

## Lesson

Extract the commands from the document instead of retyping them in the
workflow, so the doc is what gets tested. Run them under a deliberately clean
environment (`env -i` plus the minimal `PATH`): CI runners carry tools and
`PATH` entries a real new machine does not, and they silently take the "already
done" branch. Verify the *claim* the doc makes ("a new shell finds the
binaries"), not only that the commands exit 0.

Capture the extracted text in a variable before running it
(`blocks=$(helper N)` stops the step when the helper fails; `bash -c "$(helper N)"`
does not), and never put a pipe after a command whose failure must fail the step
unless the shell has `pipefail`.

## When to apply

- Any onboarding or install doc whose commands branch on what is already
  present.
- Anything that edits a shell rc file: prove it with a fresh interactive shell
  (`bash -ic`), because non-interactive shells return early from `~/.bashrc`.
