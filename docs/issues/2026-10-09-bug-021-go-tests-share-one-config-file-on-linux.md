# BUG-021 — Go tests pin only `HOME`, so they share one config file on Linux

## Symptom

CI job `server (go + postgres)` went red when the nine commits MBP2 had
never pushed reached GitHub (runs on `8a7e9ab` and `6463bae`; the same job
was green on `9c572ce`). Two tests fail, both on Linux only:

- `TestClientDefaultsWithoutConfig`
  (`server/cmd/insideout/cli_setup_test.go`, lines 71 and 74) got the base
  `https://selfhosted.example/api/v1` and the token `cfg-token` instead of
  the hosted default and an empty token. Both values are what an earlier
  test in the same package saved.
- `TestMCPConfigUsesDefaultBaseWhenEnvUnset`
  (`server/internal/apiclient/default_base_test.go`, line 36) got the same
  `selfhosted.example` base back from `scripts/mcp-config.sh`.

Both pass on macOS.

The `env catalog suites (python)` job is also red, for a different and older
reason: `scripts/tests/keys/test_store.py` asserts that real `.key` files
exist on the machine ("expected at least one .key on this machine"), which a
fresh clone cannot have. It was red on every recent run checked (`1a8816e`,
`f185e94`, `9c572ce`) and is not part of this bug.

## Root cause

The tests isolate the client config with `t.Setenv("HOME", t.TempDir())`.
`os.UserConfigDir()` follows `HOME` only on macOS. On Linux it prefers
`XDG_CONFIG_HOME`, and GitHub's Ubuntu runner image sets that variable
(`$HOME/.config`, in the runner-images `configure-environment.sh`). So on the
runner every `clientconfig.Save` writes the same real file, and each later
test reads what an earlier one left. `scripts/mcp-config.sh` reads that file
too (`${XDG_CONFIG_HOME:-$HOME/.config}/insideout/config.json` on Linux), and
the apiclient test runs it with the real environment and no isolation, so
that failure can also depend on which package binary runs first.

The tests' own comment ("HOME is pinned per test so the user's real config
never influences a result") holds on macOS only. The failing values are the
fixtures themselves, and `DefaultPath()` did not error, so `XDG_CONFIG_HOME`
was set to an absolute path on the runner. macOS cannot reproduce this: it
ignores `XDG_CONFIG_HOME`.

It is not only a CI problem. On a Linux machine with `XDG_CONFIG_HOME` set,
`go test ./...` overwrites the developer's real `insideout/config.json` with
fixture values (`cfg-token`, `cfg-file-token`). On any machine that has a
real config, macOS included, `TestMCPConfigUsesDefaultBaseWhenEnvUnset`
fails because it reads that file.

Writers: `writeClientConfig` in `cli_setup_test.go` and the three tests in
`server/internal/mcpserver/session_config_test.go`. Ten `HOME` pins in five
test files rely on the same assumption. The tests came in with `d740b2d` and
`eacea2e` (2026-10-07), which had never run in CI before this sync.

## Fix (bounded)

Beside each `t.Setenv("HOME", t.TempDir())` add
`t.Setenv("XDG_CONFIG_HOME", "")`; Go then falls back to `$HOME/.config`
under the pinned `HOME`. In `default_base_test.go` add
`INSIDEOUT_CONFIG_FILE=<temp dir>/absent.json` to the script's environment
(the script honors it). No production change.

Verify: the `server (go + postgres)` job is green and macOS still passes. To
reproduce before the fix, run the tests on Linux with `XDG_CONFIG_HOME` set
to any absolute path.

## Status

Open (2026-10-09). Found by the CI run after the three-way host sync
([changelog](../changelogs/2026-10-09-host-sync.md)); not fixed there
because it is a `server/` change outside the sync.

Lesson: [pinning `HOME` isolates a config path on macOS only](../learning/2026-10-09-pinning-home-isolates-config-on-macos-only.md).
