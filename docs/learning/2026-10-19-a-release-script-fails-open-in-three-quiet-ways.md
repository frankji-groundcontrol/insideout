# A release script fails open in three quiet ways

**Date:** 2026-10-19
**Source:** [multirole review 4](../changelogs/2026-10-18-multirole-review-4.md), findings PUB-03, PUB-04, PUB-07 and PM-04 (the release path)
**Found in:** [wave 3, release and supply chain](../changelogs/2026-10-18-rv4-remediation/w3-release-supply.md)

## The lesson

Three defaults in a release recipe look like success when they are not. Each
produces a green run and a wrong artifact.

- **A failed check inside an `&&` list does not stop a `set -e` script.** `-e`
  is ignored for every command of an AND-OR list except the last, so
  `sha256sum -c - && rm -rf /usr/local/go && tar ...` followed by more lines
  carries on to the next line when the checksum is wrong. Write the check as
  `... || { echo "..."; exit 1; }`, or make the chain the last thing in the
  block so its status is the block's status.
- **An `ARG` before `FROM` is empty inside the stage.** An `ARG` placed before
  `FROM` is visible to the `FROM` line only. Declare it again after `FROM`
  (`ARG VERSION=dev`), or `go build -ldflags "-X ..."` stamps an empty string.
  The image context (`server/`) has no `.git`, so nothing else can supply it.
- **The linker ignores `-X` for a symbol that does not exist.** Rename
  `main.version` and the stamp becomes a no-op: the binary reports `dev` with no
  warning. Pin the symbol and the flag together; here
  `scripts/tests/check_release_gate.py` reads both, and the comment on the
  variable in `server/cmd/insideout/version.go` says why.

## When to apply

- Any shell block that verifies something (a checksum, a signature) and then
  acts on it: run it once with the check made to fail and see that the next line
  is not reached.
- Any build stamp (`-X`, a `VERSION` build argument): print the stamped value
  from the built artifact, never only from the recipe.

## Evidence

The Go toolchain block of the pinned-download recipe ran in a sandbox under
`set -euo pipefail` with a marker `echo` in place of the removal and unpack
line: a pin with one digit changed gave `FAILED`, exit 1, marker not reached;
the right pin gave `OK` and the marker. The first Flutter block left its `PATH`
export outside the chain, so a mismatch could pass; it is now one closed chain.
The `ARG` and `-X` cases come from the version-stamp work (rv4-PM-04), which
moved to the proposed wave 4 `w4-release-version`; the lesson stands without
it. The release workflow itself is only proved by its first CI run after the
push.
