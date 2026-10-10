# CI and host fixes

`main` was red after the 2026-10-09 sync pushed nine older local commits for
the first time. Three separate problems were stacked, and the first hid the
second.

## What was wrong

1. **Go tests shared one config file on Linux**
   ([BUG-021](../../issues/2026-10-09-bug-021-go-tests-share-one-config-file-on-linux.md)).
   The tests pinned `HOME`, but Linux's `os.UserConfigDir()` reads
   `XDG_CONFIG_HOME` first and GitHub's runner sets it. Two tests read a
   config that earlier tests saved.
2. **The skip gate would have failed next.** The CI step pipes `go test`
   through `tee` under `pipefail`, so it stopped at the failing tests and never
   reached the check that no test skipped. Two tests skip on a CI runner:
   `TestReadSecretTerminal` (needs a terminal) and
   `TestLocalBundledFullRunFailsCleanlyWithoutDocker` (skips when the host has
   docker, and the runner does).
3. **The python key-store audit failed on any fresh checkout.** Its docstring
   promised "CI-safe", but two tests asserted that real `.key` files exist.
   It was already red before the sync.

Also found: six Go files had drifted from `gofmt`, because only the Dart job
had a format gate.

## What changed

- Every test that pins `HOME` for the client config also pins
  `XDG_CONFIG_HOME` (10 sites in 4 files); the `mcp-config.sh` test points the
  script at an absent config file.
- `TestLocalBundledFullRunFailsCleanlyWithoutDocker` empties `PATH`, so the
  missing-docker branch runs on every host. The CI gate now fails on any skip
  except `TestReadSecretTerminal`, the one test that needs a real terminal.
- `scripts/tests/keys/test_store.py` skips the two store audits when the
  checkout holds no keys, as its docstring says.
- The six files were formatted and CI gained `gofmt -l` in the build step.
- Git hooks are enabled in the MBP2 clone (`core.hooksPath`).
- MBP's `main` now tracks `origin/main`, and that clone alone (not `--global`)
  uses HTTP/1.1 to talk to GitHub.
- CI and `client/Dockerfile` run `flutter pub get --enforce-lockfile`. A lock
  written under another SDK (MBP's 3.47.0) now fails the build instead of being
  re-resolved silently. Flutter 3.44.0 and 3.44.8 declare the same five
  SDK-pinned packages, so the lock MBP2 produced is valid on CI's 3.44.0.

## How it was verified

- Real PostgreSQL 18 on a disposable local cluster: `go test ./... -v`,
  465 PASS, 0 FAIL, 1 SKIP, exit 0. The gate snippet passes on that log and
  fails when a database-gated skip is injected.
- `TestReadSecretTerminal` passes under a real pty with the documented
  `script` recipe.
- Python: on a pristine worktree (no `.key` files) the unchanged audit fails
  two tests (the CI failure, reproduced); with the change the CI commands exit
  0 (`OK`, 33 skipped).
- macOS cannot reproduce the Linux-only `XDG_CONFIG_HOME` path, so the Go job
  on GitHub was the verdict for item 1. **CI result:** the run on `8ee97fc` is
  green in all three jobs. The two new gates both executed to the end: gofmt
  inside "Build and vet", and the skip gate inside "Test". That is also the
  first run where the step behind the old hidden failure got past `go test`.
- MBP: five consecutive `git ls-remote origin` runs after the repo-local
  `http.version HTTP/1.1` change returned in 2–3 s each (the HTTP/2 route had
  been dropping them); `git rev-parse --abbrev-ref @{u}` now prints
  `origin/main`.

## Operator notes

None. No runtime code changed.
