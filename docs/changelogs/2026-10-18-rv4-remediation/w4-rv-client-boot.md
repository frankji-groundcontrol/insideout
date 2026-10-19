# Wave 4, client boot — review-4 remediation

Branch `rv4/w4-rv-client-boot`, base `582a4bd`. Five commits, `829004c` to
`80817ef`, plus this fragment. Seven findings of the whole-change review of
`d53035a..rv4/integration` (row `whole-change-review` of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md)),
in two independent roots. Findings are cited as `rv4-<ID>` because review IDs
restart per review.

| Finding | State | Red | Fix |
| --- | --- | --- | --- |
| rv4-FR-F14, High: every web cold start holds the boot page for the 10 s gate deadline | `FIXED` | `829004c` | `7709dbc` |
| rv4-FR-F215, High: same root, seen as a ten-second text-free spinner | `FIXED` | `829004c` | `7709dbc` |
| rv4-FR-F210, Medium: same root, and the stale header | `FIXED` | `829004c` | `7709dbc` |
| rv4-FR-D01, High: same root, plus DESIGN.md and `cobalt.dart` | `FIXED` for the gate; DESIGN.md is an `INTEGRATOR TODO`; `cobalt.dart` web face names are "not taken: file not owned" | `829004c` | `7709dbc` (`80817ef` is formatting) |
| rv4-FR-F212, High: the manifest does not stop dart:io sending tokens over http | `FIXED` in Dart; the docs and the test header are `INTEGRATOR TODO`s | `96cc876` | `21dbb1b` |
| rv4-FR-F209, Medium: same root | `FIXED` (same caveat) | `96cc876` | `21dbb1b` |
| rv4-FR-F11, Medium: same root, the https guard in `Session` | `FIXED` | `96cc876` | `21dbb1b` |

Rows of one root share a commit pair, as the brief allows.

## Root B: the web boot gate (open residue R3-30)

`loadWebFonts()` polled `document.fonts.load()` for 14 CSS faces (Archivo, Noto
Sans SC, Source Serif 4, Noto Serif SC). rv4-COMP-03 took the only stylesheet
that declared them out of `web/index.html`, and CanvasKit never read CSS
`@font-face` in the first place: it paints with the fonts it downloads itself,
from `web/fonts/s` through `fontFallbackBaseUrl` in `web/flutter_bootstrap.js`.
So `load()` can only resolve empty and the poll ran to its 10 s deadline on every
cold start, while `main.dart` kept the boot loading page up behind it.

**Reproduced twice before any change.**

- Chrome lane (`829004c`, base code):
  `Expected: a value less than Duration:<0:00:02.000000>`,
  `Actual: Duration:<0:00:10.056800>`; and the second test, `Expected: empty`,
  `polled 83 times, first [400 16px "Archivo", 400 16px "Archivo"]`.
- The built bundle, cold, in headless Chrome (below): the landing page after
  10.7 to 10.8 s, 83 `load()` calls, every one resolved with no face,
  `document.fonts.size` 0.

| Build | Runs | Landing page after navigation | `document.fonts.load` calls | The 12 Noto Sans SC slices asked for at | Hosts contacted (1 run) |
| --- | --- | --- | --- | --- | --- |
| base `582a4bd` | 4 | 10.77, 10.82, 10.72, 10.75 s | 83 each (3 runs counted) | 10.53 s | this origin only |
| fix `21dbb1b` | 4 | 0.80, 0.67, 0.63, 0.68 s | 0 | 0.56 s | this origin only |

Both builds ask for the same font files (one Roboto, the 12 Noto Sans SC slices
and two others) and for nothing else off the page. The screenshots 3 s after the
landing page appeared are one file across the six runs that saved one (three
per build; SHA-256 `ea8be6d0...`), so the end state did not change.

**Decision: no gate, not a shorter one.** Three ways out were weighed.

- *Wait on something that exists.* Nothing does. The engine registers nothing in
  `document.fonts` (size 0 in every run) and Flutter has no public hook for
  "the fallback font fetch finished". Anything Dart could poll is a guess.
- *Keep the poll with a short deadline.* It would still cost its deadline on
  every start and wait for nothing.
- *No-op (taken).* The wait never protected the first Chinese frame either: in
  the base build the Noto Sans SC slices were requested only after the gate
  lifted (10.53 s), when the landing page was first laid out; in the fix they are
  requested at 0.56 s, at the same moment. The first-landing frames of the fix
  (three runs, about 100 ms spacing; the first frame that passes 20 KB) draw the
  Chinese heading with its glyphs, no missing-glyph boxes.

No third-party host came back: the fix removes code only, and
`web_csp_self_test.dart` (the CSP names no host) still passes.

## Root A: cleartext in a release build

`android:usesCleartextTraffic="false"` and a network security config govern the
platform HTTP stack. The client's dio runs on dart:io sockets, which read
neither, so a release APK built with `--dart-define=API_BASE=http://<lan host>/...`
sent the login, the bearer token and the refresh token in clear text, and the
rv4-COMP-06 test only read XML.

**Where the check runs.** `Session` is the only place an `ApiClient` is built
(`ApiClient(` appears once in `lib/`, and the only `Dio(` is inside it), and every
request and SSE URL derives from its `baseUrl`; no absolute URL appears in
`lib/api` or `lib/session` but the default base itself. A guard in `Session`
therefore covers all of it.

`checkApiBase(base, {releaseMode = kReleaseMode})` is a pure function and the
`Session` constructor calls it. `kReleaseMode` is a compile-time constant and
`flutter test` is always a debug build, so `releaseMode` is a parameter. Release
mode only; debug and profile keep every base for the dev loop.

| `API_BASE` in a release build | Result |
| --- | --- |
| `https://...` (any case) | allowed |
| `/api/v1` (a path: the hosted web build, same origin behind nginx) | allowed |
| `http://` to `localhost`, `127.0.0.1`, `::1`, `10.0.2.2` | allowed |
| `http://` to any other host, `0.0.0.0`, a look-alike (`http://localhost@evil.example/...`, `http://127.0.0.1.example.org/...`) | refused |
| `ftp://...`, `//host/...` (the scheme is unknowable), `localhost:8080/...` (parses with the scheme `localhost`), an unparseable string | refused |

The message names `API_BASE` and `https`, says why, and leaves the base out
(it may hold a login). `Session()` throws a `StateError` from `main()` before
`runApp`: startup stops, which is the point.

**Reproduced.** At the base the test file does not compile, because there is no
guard to call:

```
test/rv4_review_client_boot_test.dart:31:17: Error: Method not found: 'checkApiBase'.
test/rv4_review_client_boot_test.dart:95:11: Error: No named parameter with the name 'releaseMode'.
```

That is the red the reviewers' own sketches imply (a `releaseMode` seam is the
only way a debug-build test can reach release behavior). To see it fail on
assertions too, a no-op `checkApiBase` and an ignored `releaseMode` were put in
scratch: four failures, among them `Expected: throws <Instance of 'StateError'>
with message: (contains 'API_BASE' and contains 'https')`, `Actual: <Closure:
() => Session>`, and `Expected: contains 'dart:io'` for the manifest comment.

## Mutation checks

Each fix hunk was reverted or its key condition flipped, and the guard went red.

| Mutation | Result |
| --- | --- |
| old `loadWebFonts()` body back (Chrome lane) | `00:20 +0 -2`: both gate tests fail |
| `<link rel="stylesheet">` added to `web/index.html` | `web_font_assets_test.dart` fails at the HTML check |
| `web/x.css` added | the same test fails at the `.css` check |
| `'http' => true` in `checkApiBase` | 2 failures (refuse http to a stranger, `Session` refuses a LAN base) |
| `Session` skips `checkApiBase` | 1 failure (`Session` refuses an http LAN base) |
| the old manifest comment back | 1 failure (the manifest does not promise what dart:io ignores) |

The new `web_font_assets_test.dart` test is green at the base by construction
(nothing declares a face there; that is why the gate is dead code). It guards
the precondition, not the fix. The red-then-green pair for root B is the Chrome
lane.

## What changed

- `client/lib/theme/web_fonts_web.dart` (68 to 16 lines): `loadWebFonts()` is a
  no-op, and the header says why instead of "STALE SINCE ... R3-30". The poll,
  the face list and the `dart:js_interop` and `package:web` imports are gone.
  `web_fonts_stub.dart` is unchanged: `main.dart` (not owned) still imports one
  of the pair by conditional import.
- `client/test/web/web_fonts_gate_web_test.dart` (new, 52 lines, Chrome lane):
  the gate returns in under 2 s, and records no `document.fonts.load` call (the
  call is wrapped and restored in `addTearDown`).
- `client/test/web_font_assets_test.dart` (+1 test): `web/index.html`, comments
  stripped, has no `@font-face`, `<style` or stylesheet `<link`, and no `.css`
  file is under `web/`. If a stylesheet returns, this fails and the gate must
  come back with it.
- `client/lib/session/session.dart` (144 to 179 lines): `checkApiBase`, and
  `Session({..., releaseMode = kReleaseMode})` calls it before building
  `ApiClient`.
- `client/test/rv4_review_client_boot_test.dart` (new, 145 lines, 9 tests):
  `checkApiBase` in release (refused, refused-because-unprovable, kept), in
  debug and profile, through `Session`, and the manifest comment.
- `client/android/app/src/main/AndroidManifest.xml`: the comment only. It now
  says the flag and the config govern the platform (Java) stack, that dart:io
  ignores them, and names `checkApiBase`. The attribute is unchanged, so
  `android_cleartext_test.dart` still passes untouched.

## Honest limits

- **No Android SDK, no device, no APK.** That dart:io ignores the manifest flag
  rests on Android's own documentation, which calls the flag best-effort and says
  "there's no expectation that the `Socket` API honors this flag", and on
  flutter/flutter#106678 ("Flutter doesn't respect Android's
  `android:usesCleartextTraffic="false"` settings", closed as not planned in
  February 2026). Neither was re-run on a device here. The Dart guard itself is
  tested.
- **The guard checks the configured base.** It does not look at a redirect a
  server sends or at an absolute URL passed per request. Nothing in `lib/`
  issues one today; dio's redirect defaults are untouched.
- **Web release builds:** the guard does not know the platform, so an absolute
  `http://` base to a stranger is refused there too. The documented web builds use
  `/api/v1` (`client/Dockerfile`, `ci.yml`) and are unaffected; a page served over
  https blocks http fetches as mixed content anyway.
- **Failure is quiet on Android.** A refused base stops `main()`; the message is
  in the log (`adb logcat`, the browser console), and the app shows only its
  launch screen. A
  friendlier screen needs `main.dart`, which is not owned.
- **The first-frame check has a gap.** The screenshot loop samples about every
  100 ms, so a one-frame flash of missing Chinese glyphs between first paint and
  the arrival of a slice could fall between samples. The gate never covered that
  moment (in the base the slices were first requested after it lifted), so
  removing it neither adds nor removes the flash.
- Browser runs: headless Google Chrome 155 under Playwright 1.64, against a small
  static server on an OS-assigned loopback port that serves `build/web` the way
  `client/nginx.conf` does for the SPA (fallback to `index.html`, 404 for a
  missing `/fonts/` file, 401 for `/api/`), without the CSP header. The bundle:
  `flutter build web --release --no-web-resources-cdn --dart-define=API_BASE=/api/v1`
  on Flutter 3.44.8. The image was not built.

## Not taken

- `client/lib/theme/cobalt.dart` still names web faces that never reached the
  canvas (the other half of the R3-30 row): not taken: file not owned.

## Gates

All run in this worktree on the code commits (this fragment is docs only).

| Gate | Result |
| --- | --- |
| `flutter pub get --enforce-lockfile` | exit 0; `pubspec.lock` unchanged |
| `flutter analyze --no-fatal-infos` | exit 0; 14 infos, none in a file of this cluster (`lib/api/client_core.dart`, `client_prd.dart`, `client_stream.dart`, `requests.dart`, which this branch does not touch) |
| `flutter test` | 547 passed |
| `TZ=Asia/Shanghai flutter test` | 547 passed |
| `dart format --output=none --set-exit-if-changed lib test` | exit 0 |
| Chrome lane, `flutter test test/web --platform chrome` (Chrome 155 through `CHROME_EXECUTABLE`) | 8 passed, the 2 new gate tests among them |
| `gofmt -l internal cmd` | clean (no Go change) |
| changed `.go` and `.dart` files over 350 lines | none (the longest, `session.dart`, is 179) |
| `scripts/tests/check_*.py` (14 guards) | all ok |
| `sh scripts/check-redaction.sh` | exit 0 |
| `sh scripts/check-docs-recorded.sh --worktree` | exit 0, no warning (it is warn-only; the fragment is the record) |

No database, no live server, no named port: this cluster needs none.

## Handed on (INTEGRATOR TODO)

- INTEGRATOR TODO: `DESIGN.md`: line 182: replace "Web loads faces from
  `client/web/index.html` and gates the first frame in
  `client/lib/theme/web_fonts_web.dart`." with "Web draws with the fonts the
  CanvasKit engine downloads itself (Roboto and Noto slices, served from
  `client/web/fonts/s` through `fontFallbackBaseUrl` in
  `client/web/flutter_bootstrap.js`); the web boot has no font gate."
- INTEGRATOR TODO: close R3-30 in `docs/issues/2026-10-18-review-4-open-findings.md`
  (the row at line 155) for the gate half: fixed by `7709dbc`, proven by the
  Chrome-lane pair `829004c` and `7709dbc`. Leave the `cobalt.dart` web face
  names open (not taken: file not owned), and make `docs/issues/README.md` say
  the same.
- INTEGRATOR TODO: `docs/usage/local-development.md`: lines 168 to 175 (the
  Android paragraph) claim a LAN http address "no longer works over http, even in
  debug" and that no build type allows cleartext. That holds for the Java stack
  only. Say that the Dart guard `checkApiBase` refuses, in a release build, an
  `API_BASE` that is not `https`, a path such as `/api/v1`, or `http` to
  `localhost`, `127.0.0.1`, `::1` or `10.0.2.2`; that debug and profile accept any
  base, a LAN http address included, because dart:io ignores the manifest; and
  that anything beyond your own machine wants https or `adb reverse`.
- INTEGRATOR TODO: `client/test/android_cleartext_test.dart`: the header (lines
  3 to 15) says "a release APK must not allow cleartext HTTP". Add that the test
  reads XML only, that dart:io ignores what it checks, and that the guard is
  `rv4_review_client_boot_test.dart`.
- INTEGRATOR TODO: `docs/learning/2026-10-19-canvaskit-ignores-css-font-faces.md`:
  line 46 says "Still open: `loadWebFonts()` waits up to 10 s"; it is closed (the
  gate is a no-op).
- INTEGRATOR TODO (cleanup, optional): `client/lib/main.dart` is the only caller
  of `loadWebFonts()`. Dropping the conditional import and the call, then
  deleting `client/lib/theme/web_fonts_web.dart`, `web_fonts_stub.dart` and
  `client/test/web/web_fonts_gate_web_test.dart`, leaves `web_font_assets_test.dart`
  as the only guard. Not done here: `main.dart` is not owned.
- INTEGRATOR TODO: `docs/changelogs/README.md`,
  `docs/changelogs/index/2026-10.md` and
  `docs/changelogs/2026-10-18-rv4-remediation/index.md`: index this fragment;
  `docs/plans/README.md`, `docs/HANDOFF.md`: the wave-4 row state.
- The older records of the gate
  (`docs/changelogs/2026-08-23-web-font-flash.md`,
  `docs/issues/2026-10-06-bug-016-web-boot-gate-hangs-on-font-load.md`,
  `docs/learning/2026-10-06-a-deadline-must-bound-the-await.md`) describe it as it
  was when it was real and stay as history.
