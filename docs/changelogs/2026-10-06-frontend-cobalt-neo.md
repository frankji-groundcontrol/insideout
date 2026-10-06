# 2026-10-06 — Cobalt-Neo remediation closed

Waves 1–3 were already checked. This pass lands waves 4–7 in the
worktree and closes the
[plan](../plans/2026-10-06-frontend-cobalt-neo/2026-10-06-frontend-cobalt-neo.md).
On `main` as `28e1fb1`. Nothing here is deployed.

## What changed

- **Writes (4.1–4.2).** Proposal accept/reject and the version commit
  go through `guardWrite` and the localized `write.failed` snackbar.
  Idea, workspace (including settings), and the other project writes
  already did. A failed idea capture or convert leaves the card and
  the capture field; the field still clears only after success. A
  failed version commit leaves the stack empty.
- **Coach stream (4.3).** `readCoachSse` idles out at 75 seconds
  (`coachSseIdleTimeout`). An explicit `Timer` restarts on every
  complete frame, including heartbeats. `Stream.timeout` did not
  elapse under the widget-test clock, so the reader no longer uses it.
  The coach tests stall through that reader.
- **Dock targets (4.4).** Already true, and kept: evidence on its
  project page pulses a scroll-highlight; version opens
  `/prd/<id>/versions` or stays disabled; cold tiles do not go to
  `/dashboard`.
- **Minimap (5.1).** Done, pending, in-progress, and locked already
  differed without color (filled, half, hollow, dashed) plus semantics
  labels. Marks are square (`BorderRadius.zero`). The widget test
  fingerprints paint without color and fails if those four states
  collapse to one signature. Theme files were not edited. The closing
  `flutter test` re-ran `cobalt_test` and it passed (5.2).
- **Harness (6.1–6.2).** The dock test mounts `AppScaffold`
  (`extendBody`) at 1400×900 and asserts the dock rect misses the body
  landmark. At 390px the unfinished label is icon-only, ellipsized when
  shown, and the pill stays under 360 wide. The auth test is geometric,
  not a golden: at 1440×900, light and dark, zh-CN, the wordmark sits
  inside the card.
- **Format gate and Flutter pin (6.3–6.4).** CI runs
  `dart format --output=none --set-exit-if-changed lib test` beside
  analyze. `docs/usage/local-development.md` and `client/README.md` say
  Flutter 3.44.x. CI and the Dockerfile already pin 3.44.0.
- **Web fonts (7.1).** `assets/fonts/OFL.txt` stays an unconditional
  asset. Both variable TTFs are limited to android, ios, linux, macos,
  and windows, so a web build omits them. Native code still
  `rootBundle`-loads those files, so the directory entry was not
  reduced to `OFL.txt` alone. The TTF files stay on disk. The Google
  Fonts link is unchanged. The web bundle was not rebuilt.
- **URL secrets (7.2).** Invite, reset, and magic-link tokens are
  copied into state, then web-only `history.replaceState` drops those
  query keys. The pure rewrite is `urlWithoutQueryKeys` (no Flutter
  binding, no `GoRouter.replace`).

`project_page.dart` was already over the ~350-line budget. The decide
edit shortened it; the file is 640 lines and was not split.

## Verification

Local gate in `client/`, 2026-10-06:

- `flutter analyze --no-fatal-infos` — exit 0. No `error` or `warning`
  lines. The summary is `32 issues found. (ran in 0.9s)`; those 32 are
  infos.
- `flutter test` — `00:09 +143: All tests passed!` then `TEST_EXIT:0`.
- `dart format --output=none --set-exit-if-changed lib test` — exit 0.

GitHub CI was not watched.

## Operator note

No migration and no server change. A deployed web client does not pick
this up until the client is rebuilt and shipped. The release bundle
still predates this pass, including the font-asset limit and the URL
secret strip.

Deferred: three uncaught minified console errors from the Wave 3
narrow dock tap pass
([W3-N2](../issues/2026-10-06-wave3-minified-console.md)). No
user-visible impact was recorded. Not fixed here.
