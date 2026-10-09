# 2026-10-09 — Pinning `HOME` isolates a config path on macOS only

## Context

The CLI's Go tests pinned `HOME` to a temp dir and trusted that to keep the
developer's real config file out of every result ("HOME is pinned per test so
the user's real config never influences a result"). They passed on both Macs
and failed on GitHub's Linux runner
([BUG-021](../issues/2026-10-09-bug-021-go-tests-share-one-config-file-on-linux.md)).

## Lesson

`os.UserConfigDir()` reads `HOME` on macOS but `XDG_CONFIG_HOME` first on
Linux, and GitHub's Ubuntu runner sets that variable. So the pinned `HOME`
was ignored there: every test saved to one real file, and later tests read
what earlier ones left.

To isolate code that resolves a per-user directory, read the resolver's source
and pin every variable it consults (here: also set `XDG_CONFIG_HOME` to
empty), or hand the code an explicit path. A comment that says "the real
config never leaks" is a claim about one OS, and a test that passes only where
it was written has not been checked.

## When to apply

- Any test that reaches `os.UserConfigDir` or `os.UserCacheDir`, and any
  script it shells out to: `scripts/mcp-config.sh` re-implements the same
  per-OS rule.
- Before a backlog of local commits goes up: those commits have never met CI,
  so run their tests on the CI OS rather than assuming the dev OS agrees.
