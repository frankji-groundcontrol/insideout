# 2026-10-09 — Host setup drift between the two Macs

Found while syncing GitHub, MBP2, and MBP
([changelog](../changelogs/2026-10-09-host-sync.md)). None of these blocks
work; each makes the next sync noisier.

**Status 2026-10-10:** items 2–4 are fixed and verified. Item 1 is guarded in
the repo; only MBP's own SDK version is left, and that is the owner's call
(see its note).

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

**Guarded 2026-10-10.** CI and `client/Dockerfile` now run
`flutter pub get --enforce-lockfile`, so a lock written under another SDK
fails the build instead of being re-resolved silently. This is safe on the
pinned SDK: Flutter 3.44.0 (CI, Dockerfile) and 3.44.8 (MBP2) declare the same
five SDK-pinned packages (`meta` 1.18.0, `vector_math` 2.2.0, `intl` 0.20.2,
`test_api` 0.7.11, `matcher` 0.12.19; compared from the SDK tags), and the
command exits 0 on MBP2 with a clean `git status`. On MBP's 3.47.0 the same
command now stops with an error instead of rewriting the lock, which is the
signal to use 3.44.x there.

**Left for the owner:** installing Flutter 3.44.x on MBP. That changes the
global toolchain of a second machine that is currently overloaded (load
average in the hundreds), and nothing in this repo depends on it now.

## 2. Git hooks are not enabled on MBP2

`core.hooksPath` is `config/git-hooks` on MBP and unset on MBP2, so the nine
commits made on MBP2 before this sync skipped the docs guardrail (a warning
for source commits, a hard gate for `[checkpoint]` commits). The sync ran its
own commits with the hooks enabled per command instead.

Fix: `git config core.hooksPath config/git-hooks` once in the MBP2 clone
(CLAUDE.md, "Recording changes"). Verify: `git config core.hooksPath` prints
the path, and a source-only commit prints the guardrail warning.

**Fixed 2026-10-10:** `core.hooksPath` is `config/git-hooks` in the MBP2 clone.

## 3. MBP reaches GitHub over HTTP/2 only intermittently

`git ls-remote origin` on MBP failed twice in a row with `Error in the HTTP2
framing layer` during the final verification, after fetches had succeeded
earlier the same hour; with `-c http.version=HTTP/1.1` it succeeds. MBP's
only `http.*` key is `http.postbuffer`.

Failure scenario: a plain `git pull` or `git push` on MBP fails mid-sync.

Fix: `git config --global http.version HTTP/1.1` on MBP. Verify: five
consecutive `git ls-remote origin` runs exit 0.

**Fixed 2026-10-10**, scoped to the repo instead of `--global`:
`git config --local http.version HTTP/1.1` in the MBP clone. Five consecutive
`git ls-remote origin refs/heads/main` runs returned the right commit in 2–3 s
each.

## 4. MBP's `main` has no upstream

`git rev-parse --abbrev-ref @{u}` fails on MBP and `git status -sb` shows no
ahead/behind, because every update there has been a manual fetch followed by
`git merge --ff-only origin/main`. The sync's checks compare commits
directly, so nothing depended on it.

Fix: `git branch --set-upstream-to=origin/main main` on MBP. Verify:
`git status -sb` prints `## main...origin/main`.

**Fixed 2026-10-10:** `git branch --set-upstream-to=origin/main main` on MBP;
`git rev-parse --abbrev-ref @{u}` prints `origin/main`.

## 2026-10-11 update (closed)

Flutter 3.44.8 (arm64) is installed on MBP at `~/flutter` with the PATH
line in `~/.zshrc`; `flutter pub get --enforce-lockfile` passes there and
`client/pubspec.lock` stays unwritten. The `app/.env` leftover is confirmed
absent. Both halves of the drift are closed; MBP is back on the shared
commit.

The interim note below is kept for the record: the first archive grab
(`flutter_macos_3.44.8-stable.zip`, no arch suffix) is the **Intel-only**
build — on an arm64 Mac its dart binary answers "Bad CPU type in
executable"; the arm64 archive is `flutter_macos_arm64_3.44.8-stable.zip`.

## 2026-10-11 update (interim)

The `app/.env` leftover is confirmed absent on MBP. Flutter 3.44.8
(repo-pinned line, matching MBP2's homebrew 3.44.8) is **staged, not yet
extracted**: the 2.2 GB archive sits at `/tmp/flutter.zip` on MBP and the
install went offline mid-task (laptop left the network). One command when
it is back:

```bash
ssh MBP 'cd ~ && rm -rf flutter && unzip -q /tmp/flutter.zip && rm -f /tmp/flutter.zip &&   grep -q flutter/bin .zshrc || printf "\nexport PATH="\$HOME/flutter/bin:\$PATH"\n" >> .zshrc;   ~/flutter/bin/flutter --version'
```

(macOS clears /tmp after ~3 days — if the zip is gone, re-download
`flutter_macos_3.44.8-stable.zip` from Flutter's stable releases first.)

