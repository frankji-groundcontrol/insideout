# 2026-10-09 — Host setup drift between the two Macs

Found while syncing GitHub, MBP2, and MBP
([changelog](../changelogs/2026-10-09-host-sync.md)). None of these blocks
work; each makes the next sync noisier.

## 1. MBP runs Flutter 3.47.0; the repo pins 3.44.x

The pin lives in `.github/workflows/ci.yml`, `client/Dockerfile`, and
[local-development](../usage/local-development.md). MBP2 runs 3.44.8; MBP
runs 3.47.0. `flutter pub get` takes five packages straight from the SDK
(`flutter_test` pins `test_api` and `matcher` exactly; `flutter` and
`flutter_localizations` constrain `meta`, `vector_math`, and `intl`), so each
host rewrites them to its own SDK. The lock before `0fd171e` matched 3.47.0
and failed `flutter pub get --enforce-lockfile` under 3.44.8.

Failure scenario: the first `flutter pub get`, `analyze`, or `test` on MBP
rewrites `client/pubspec.lock` back to the 3.47 pins and leaves a dirty tree;
committing it from there flips the lock for MBP2 again.

Fix (bounded, pick one): put MBP on Flutter 3.44.x, the documented pin; or
move the pin to the newer SDK in CI, the Dockerfile, and the docs together
and re-resolve the lock once. Verify: `cd client && flutter pub get
--enforce-lockfile` exits 0 on both Macs and `git status` stays clean.

## 2. Git hooks are not enabled on MBP2

`core.hooksPath` is `config/git-hooks` on MBP and unset on MBP2, so the nine
commits made on MBP2 before this sync skipped the docs guardrail (a warning
for source commits, a hard gate for `[checkpoint]` commits). The sync ran its
own commits with the hooks enabled per command instead.

Fix: `git config core.hooksPath config/git-hooks` once in the MBP2 clone
(CLAUDE.md, "Recording changes"). Verify: `git config core.hooksPath` prints
the path, and a source-only commit prints the guardrail warning.

## 3. MBP reaches GitHub over HTTP/2 only intermittently

`git ls-remote origin` on MBP failed twice in a row with `Error in the HTTP2
framing layer` during the final verification, after fetches had succeeded
earlier the same hour; with `-c http.version=HTTP/1.1` it succeeds. MBP's
only `http.*` key is `http.postbuffer`.

Failure scenario: a plain `git pull` or `git push` on MBP fails mid-sync.

Fix: `git config --global http.version HTTP/1.1` on MBP. Verify: five
consecutive `git ls-remote origin` runs exit 0.

## 4. MBP's `main` has no upstream

`git rev-parse --abbrev-ref @{u}` fails on MBP and `git status -sb` shows no
ahead/behind, because every update there has been a manual fetch followed by
`git merge --ff-only origin/main`. The sync's checks compare commits
directly, so nothing depended on it.

Fix: `git branch --set-upstream-to=origin/main main` on MBP. Verify:
`git status -sb` prints `## main...origin/main`.
