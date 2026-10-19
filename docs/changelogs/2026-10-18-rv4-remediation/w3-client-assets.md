# Wave 3, client assets — review-4 remediation

Branch `rv4/w3-client-assets`, base `4adea1a`. Eight commits, `3530169` to
`dea8dab`, plus this fragment. Findings are cited as `rv4-<ID>` elsewhere
because review IDs restart per review. Row 3.11 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md);
the findings are in the
[register](../../review/2026-10-18-multirole-4/01-findings-register.md).

| Finding | State | Red | Fix |
| --- | --- | --- | --- |
| rv4-COMP-03, the web client needs Google hosts | `PARTIAL` | `3530169`, `95408ad`, `069decd` | `11b6f65`, `b52d404` (`dea8dab` is a comment) |
| rv4-COMP-06, cleartext HTTP in every Android build type | `FIXED` at the manifest level | `2ed33a7` | `d2840bd` |

**rv4-COMP-03 was real, and what is left is one file.** The register marked it
unverified, so the first check below measured it. The base web build asks
`www.gstatic.com` for CanvasKit and `fonts.gstatic.com` for the fonts the engine
paints with, and links a Google Fonts stylesheet (`fonts.googleapis.com`) that
CanvasKit never reads. Block the first and the page never starts; block the
second and it starts but draws no text. In the branch CanvasKit and the
stylesheet link are gone, the app starts with every off-origin host blocked, and
every font the engine can ask for is served from the app's own origin. **One host
is left**: the policy still names `fonts.gstatic.com` in `connect-src`, and the
engine still asks there, because moving its font downloads needs a custom
`client/web/flutter_bootstrap.js`, a new path outside `files_owned`. A verified
two-commit patch for it is ready ([The follow-up patch](#the-follow-up-patch)).
Until it is applied, a network that cannot reach `fonts.gstatic.com` gets an app
that starts but draws no text, as the base did.

**rv4-COMP-06 is fixed in the manifests.** `usesCleartextTraffic` is `false` in
the main manifest, which every build type merges. Debug and profile, never
shipped, reach a local dev server through a network security config that allows
cleartext for `localhost`, `127.0.0.1` and `10.0.2.2` only.

**Nothing here ran on a runner, in the image or on a device.**

- `.github/workflows/ci.yml`: one `flutter build web` line changed. The file
  parses as YAML; `actionlint` is not installed here. **Pending first CI run
  after push.**
- The image was not built. Its inputs were exercised alone: `flutter build web
  --release --no-web-resources-cdn` on Flutter 3.44.8 (Dockerfile and CI pin
  3.44.0; the engine's font tables are identical) and the rendered
  `client/nginx.conf` on nginx 1.31.7 from an isolated conda-forge prefix (the
  image has 1.27, the CI job the distro package).
- Browser runs: headless Google Chrome 155 under Playwright 1.64, against a small
  static server that serves `build/web` the way `client/nginx.conf` does for the
  SPA (fallback to `index.html`, 404 for a missing `/fonts/` file, no backend)
  with the Content-Security-Policy that file declares.
- Android: the five manifests and configs were read as text and parse as XML. No
  SDK, so no APK was built; the merged manifest and device behavior are unchecked.

## First check (rv4-COMP-03)

The register's first check: build with `--no-web-resources-cdn`, serve it,
confirm no request leaves the origin. Before changing anything, the base
(`4adea1a`) was built with the default flags and loaded in Chrome (locale zh-CN,
viewport 1280 by 800), recording every request by host. "Blocked" aborts every
request to a host other than the origin; the open runs reach Google through this
machine's proxy. A screen counts as drawn when the screenshot passes 20 KB (the
boot page is about 5 KB, the landing 49 to 53 KB); "Landing" is the time after
the view mounted.

| Build | Network | Off-origin requests | Mounted | Landing |
| --- | --- | --- | --- | --- |
| base | open | `www.gstatic.com` x2 (CanvasKit script and wasm), `fonts.googleapis.com` x1 (stylesheet), `fonts.gstatic.com` x13 (Roboto, 12 Noto Sans SC slices), all 200 | 1.4 s | 9 to 12 s (bracketed by screenshots) |
| base | everything off-origin blocked | 4, all blocked (the two CanvasKit files, the stylesheet, Roboto) | never (46 s) | none: `Failed to fetch dynamically imported module` (CanvasKit script) |
| base | stylesheet blocked | as open, the stylesheet blocked | 1.9 s | 10.6 s |
| base | both font hosts blocked | `fonts.gstatic.com`: 6,997 blocked requests in 34 s | 1.6 s | none: blank page |
| branch `dea8dab` | open | `fonts.gstatic.com` x13 | 1.4 s | 10.5 s |
| branch `dea8dab` | everything off-origin blocked | `fonts.gstatic.com`: 4,456 blocked requests in 25 s | 0.6 s | none: blank page |
| branch + patch, zh-CN | open | none (13 requests to `/fonts/s/`) | 1.0 s | 10.4 s |
| branch + patch, zh-CN | blocked | none (13 requests to `/fonts/s/`) | 0.3 s | 10.5 s |
| branch + patch, en-US | open and blocked | none (2 to `/fonts/s/`: Roboto, one Noto Sans Symbols file) | 0.6 s | 10.4 s |

Screenshots compared by SHA-256: the base open, base stylesheet blocked, branch
open and both patched zh-CN renders are one file; the two blank pages (base with
both font hosts blocked, branch blocked) are one file; en-US open and blocked are
one file.

- **The finding holds, with a correction.** The page needs CanvasKit
  (`www.gstatic.com`, a dynamic import, fatal when blocked) and the engine's
  fonts (`fonts.gstatic.com`), not the stylesheet: blocked, the render is the
  same. The 737 faces the stylesheet declares were never fetched; every font file
  the page loads is an engine `fetch()`.
- **Every row that finishes lands about 10.5 s after mount, the base included.**
  That matches the 10 s ceiling of the boot gate in
  `client/lib/theme/web_fonts_web.dart`, which polls for CSS faces CanvasKit does
  not use. The file is not in this cluster; the wait is as it was.

## Design spike: how a CanvasKit page gets its fonts

Each way of putting the fonts on the origin was tried or read against the SDK
source (Flutter 3.44.8):

- **A pubspec `fonts:` entry, or a CSS `@font-face`.** CanvasKit ignores CSS
  faces (the stylesheet row above). A pubspec font is an asset font the app names
  in its own text styles; it leaves the engine's default Roboto and its Noto
  download for a glyph no declared font has. Wave 7.1 took the two CJK variable
  fonts (17.8 and 25.1 MB) out of the web bundle on purpose
  (`client/test/web_font_assets_test.dart`).
- **`fontFallbackBaseUrl`, the engine option that moves the downloads.** It is
  read from the loader's `config`, in `_flutter.loader.load({config})`. The legacy
  global `window.flutterConfiguration` does not work: set before any page script,
  the engine still asked `fonts.gstatic.com` (1,123 blocked requests in 24 s,
  none to the mirror). An inline script would do it, but the CSP bars inline
  script and a hash cannot cover a per-build value. So the option needs a custom
  `web/flutter_bootstrap.js`, the file outside `files_owned`.
- **A partial mirror (the families this UI uses).** Rejected. The engine picks
  its font from the browser language and the user's text: zh-TW asks for Noto
  Sans TC, zh-HK for HK, ja for JP, ko for KR; an arrow in an English locale asks
  for Noto Sans Symbols; an emoji for Noto Color Emoji. A missing file is a 404
  the engine retries every frame (4,456 to 6,997 blocked requests in 25 to 34 s in
  the table above). The first mirror had 102 files and the red test of `069decd`
  rejected it; the mirror is the engine's whole table, as unmodified Google copies
  (a simple licence, and `SHA256SUMS` can check them).
- **Rejected as well:** nginx proxying a miss to the Google host (the server
  becomes a Google client at runtime), and a Dockerfile step that writes the
  bootstrap file (`flutter run`, the CI build and the image would build three
  different things).
- **Bundling CanvasKit.** `--no-web-resources-cdn` (the register's direction)
  puts `"useLocalCanvasKit":true` in the generated config. The build copies
  `canvaskit/` (36.7 MiB) into `build/web` either way, so the base image already
  carried it unused; the flag points the loader at it.

## What changed

### rv4-COMP-03

- `client/Dockerfile`, `.github/workflows/ci.yml`: `flutter build web` gets
  `--no-web-resources-cdn` (the one web build line in each).
- `client/web/index.html`: the two `preconnect` links and the Google Fonts
  stylesheet are gone.
- `client/nginx.conf`: the CSP drops `www.gstatic.com/flutter-canvaskit/` (from
  `script-src` and `connect-src`), `fonts.googleapis.com` (`style-src`) and
  `fonts.gstatic.com` (`font-src`); `connect-src` keeps `https://fonts.gstatic.com`
  for the engine's font downloads, the one host left. `gzip_types` gains
  `application/wasm`. A new `location /fonts/ { try_files $uri =404; }` has no
  `add_header`, because one inside a location replaces the server-level set and
  the CSP would vanish.
- `client/web/fonts/`: `s/` holds 725 woff2 files in 145 families (21.8 MB), byte
  copies of what `fonts.gstatic.com/s/` serves, in the same layout; `licenses/`
  holds the 145 SIL OFL 1.1 texts (0.64 MB); `SHA256SUMS` passes
  `shasum -a 256 -c`; `NOTICE.txt` says why the whole table, who holds the
  copyright, and how to refresh after a Flutter upgrade.
- `scripts/check_nginx_headers.py`: the policy may name no host beyond
  `CSP_HOSTS_ALLOWED` and gives `default-src`, `script-src` and `style-src` to this
  origin alone; `/fonts/` answers 200 for a mirrored file and 404, never the SPA
  shell, for another; a CanvasKit `.wasm` comes gzip'd (the stub nginx declares
  the one `application/wasm` mime mapping, so `gzip_types` is what is tested).
- `client/test/web_csp_self_test.dart` (new, 162 lines, 7 tests): `index.html`
  asks no other origin; scripts, styles and `default-src` list no host; the CSP
  names no host beyond `engineFontHosts` (empty after the patch); every
  `flutter build web` line has the flag; the whole engine table ships (725 files,
  `wOF2` magic, 145 families, a licence each); `SHA256SUMS` lists exactly the
  shipped files; the `/fonts/` block is `try_files $uri =404` with no `add_header`.
- `client/pubspec.yaml`, `client/test/web_font_assets_test.dart`: comments only
  (they said web loads Noto from the CDN).

### rv4-COMP-06

- `client/android/app/src/main/AndroidManifest.xml`:
  `android:usesCleartextTraffic="false"`, with a comment that says why.
- `src/debug` and `src/profile` (never in a release APK): the manifest overlay
  adds `android:networkSecurityConfig="@xml/network_security_config"`, and
  `res/xml/network_security_config.xml` has `<base-config
  cleartextTrafficPermitted="false" />` plus one `<domain-config
  cleartextTrafficPermitted="true">` for `localhost`, `127.0.0.1` and `10.0.2.2`
  (no subdomains).
- `client/test/android_cleartext_test.dart` (new, 118 lines, 7 tests): main says
  `false`; no manifest in any source set says `true`; release carries no network
  security config; debug and profile each point at one that exists and allow
  cleartext for loopback names only, with `base-config` staying `false`.

## Evidence

Outputs are kept under `/private/tmp/insideout-rv4/reports/` as
`w3-client-assets-red-<ID>.txt` and `-green-<ID>.txt` (variants `-gzip`,
`-mirror`, `-fontbase-patch`), with the browser runs (JSON and screenshots) in
`w3-client-assets-evidence/`.

| Step | Result |
| --- | --- |
| red `3530169`, `web_csp_self_test.dart` at the base code | 6 of 6 tests fail (off-origin links, CSP hosts, no build flag, no fonts, no `/fonts/` block); the checker reports 13 failures on nginx 1.31.7 |
| red `95408ad`, checker with the wasm gzip check | 14 failures; with the fixed `nginx.conf` but no `application/wasm` in `gzip_types`, exactly the gzip check fails |
| fix `11b6f65` | the CSP, `index.html`, build-flag and `/fonts/` checks pass; the browser rows above |
| red `069decd`, whole-table test against the 102-file mirror | `Expected: <725> Actual: <102>`; the other tests pass |
| fix `b52d404` | 725 files, 145 families, `SHA256SUMS` consistent; `web_csp_self_test.dart` 7 of 7 |
| red `2ed33a7`, `android_cleartext_test.dart` at the base files | 6 of 7 fail (main `true`, no network security config in debug or profile) |
| fix `d2840bd` | 7 of 7 pass |
| mutation check of the COMP-06 test, each edit alone then reverted | a non-loopback `<domain>` in the cleartext block, `base-config` set to `true`, and a config under `src/main/res/xml` each fail the intended test; `usesCleartextTraffic` back to `true` fails two |

Gates at `dea8dab`, before this fragment:

| Gate | Result |
| --- | --- |
| `flutter pub get --enforce-lockfile`, `flutter analyze --no-fatal-infos`, `flutter test` | exit 0; 14 infos, all in `lib/api/`, none in this cluster's files; 497 passed; `pubspec.lock` untouched |
| `TZ=Asia/Shanghai flutter test` | 497 passed |
| `scripts/check_nginx_headers.py` on nginx 1.31.7 | 65 PASS, `OK` |
| `grep -nE "fonts\.(googleapis\|gstatic)\.com\|gstatic\.com/flutter" client/web/index.html client/nginx.conf` | **not empty in the branch**: `nginx.conf:33` (a comment) and `:40` (`connect-src`); `index.html` is clean. Empty with the follow-up patch applied (checked on a scratch copy) |
| `grep -n usesCleartextTraffic` on the main manifest | `false` (line 11) |
| 350-line budget on the `.go` and `.dart` files in the diff | nothing over; the largest is `web_csp_self_test.dart`, 162 |
| `scripts/tests/check_*.py` | 10 scripts, all exit 0 (each prints its own ok line, so the loop is not silent) |
| `scripts/check-redaction.sh`, `scripts/check-docs-recorded.sh` | silent, exit 0 (the second after this fragment is committed) |
| also | `dart format --set-exit-if-changed` on the three test files, 0 changed; `check_nginx_webhook_body.py` passes; `check_nginx_reresolve.py` passes except its chunk-by-chunk timing check, which is load-sensitive (load averages of 55 to 150: 4 of 16 runs failed with this branch's `nginx.conf`, 2 of 15 with the base's) |

## Design decisions

- **The CSP and the build flag are one change.** An image built without
  `--no-web-resources-cdn` would ask the CDN for CanvasKit and the new policy
  would block it: a page that never starts. `web_csp_self_test.dart` fails if
  either build line drops the flag.
- **A network security config, not a debug overlay with the flag `true`.** That
  overlay would allow cleartext to every host in debug. The config allows loopback
  names only and overrides the legacy flag on every supported API level (minimum
  SDK 24, where the config began); with the main manifest `false` and target SDK
  36, release is denied twice over. A config under `src/main` would ship in
  release, so the test fails if one appears.
- **The CI release APK** has an https `API_BASE`, so `false` costs it nothing. iOS
  already limits App Transport Security to `NSAllowsLocalNetworking`.

## Known issues and ceilings

- **Weight.** Fonts add 21.8 MB and licences 0.64 MB to the repository and the
  image (`build/web` 41.7 to 65.1 MiB by `du`; `canvaskit/` is 36.7 MiB in both).
- **Repeat visits lose the CDN's cache headers.** A HEAD request to the Google
  hosts showed `cache-control: public, max-age=31536000` on the CanvasKit wasm and
  on a font, and the wasm sent Brotli (1.65 MB; gzip level 6 gives 2.19 MB of the
  5.76 MB file). `client/nginx.conf` sends no `Cache-Control` for `/fonts/` or
  `/canvaskit/` (nor for any static path), so a repeat load revalidates or relies
  on the browser's heuristic cache; not measured in a browser. Google's font paths
  carry a version (`v37`), so `expires 1y;` on `/fonts/` would be safe;
  `/canvaskit/` is not versioned in its path and should not get one.
- **The mirror is not checked against the SDK.** A Flutter upgrade can add fonts
  to the engine's table, and a missing font is a 404 plus a retry loop.
  `client/web/fonts/NOTICE.txt` has the refresh recipe (read the table in
  `font_fallback_data.dart`, fetch the new paths, regenerate `SHA256SUMS`, update
  the two counts in the test); no test reads the SDK.
- **Browsers and servers.** Only Chrome 155 was driven, against a stand-in
  server, not the image; Firefox, Safari, nginx 1.27 and the CI package were not
  run. The nginx directives used (`try_files`, `gzip_types`, `add_header ...
  always`) are not version-specific.
- **The web theme names faces no canvas can draw.** `client/lib/theme/cobalt.dart`
  names Archivo, Source Serif 4 and Noto Sans/Serif SC for web. They came from the
  stylesheet this change removes, and the identical renders show the stylesheet
  never reached the canvas, so web has drawn in the engine's Roboto and Noto
  either way. If those faces are wanted on web they must be Flutter asset fonts (a
  design and weight decision for the client owner).
- **The boot gate is unchanged** and costs every web cold start about 10 s (see
  the first check); it is the client owner's.
- **COMP-06 is a manifest guarantee.** `dart:io` sockets, which the app's `dio`
  client uses, do not consult `usesCleartextTraffic` or the network security
  config. Secondary sources say so (the Flutter forum thread
  `forum.itsallwidgets.com/t/how-to-prevent-http-unsecure-calls-in-flutter-app/957`
  and flutter/engine PR 20218); not verified on a device here. The change governs
  the Java stack (WebView, plugins on `HttpURLConnection` or OkHttp) and satisfies
  scanners; a runtime guarantee for the app's own calls needs a Dart-side guard in
  `client/lib/session/session.dart`, which this cluster does not own.
- **Dev flow on a physical Android device changes.** A LAN address no longer works
  over http in debug: use `adb reverse tcp:8080 tcp:8080` and `API_BASE` at
  `127.0.0.1`; the emulator reaches the host as `10.0.2.2`.
- **The brief said "two flutter build lines" in `ci.yml`;** there is one `flutter
  build web` line. The android job's `flutter build apk` takes no web flag.

## The follow-up patch

`/private/tmp/insideout-rv4/reports/worker-w3-client-assets-r0-font-base.patch`,
two commits (red `3947549`, fix `eb92161`, written on `b52d404`). It applies
unchanged on `dea8dab`, which differs from `b52d404` in one comment of
`client/nginx.conf`. From the repository root, `git apply` it (or `git am` to keep
the two commits). One path is outside `files_owned`; the rest are owned files.

1. New `client/web/flutter_bootstrap.js`, the default template plus one option
   (and a six-line comment on why it exists):

   ```
   {{flutter_js}}
   {{flutter_build_config}}

   _flutter.loader.load({
     config: {
       fontFallbackBaseUrl: "fonts/s/",
     },
     serviceWorkerSettings: {
       serviceWorkerVersion: {{flutter_service_worker_version}},
     },
   });
   ```

2. Owned files: `client/nginx.conf` (`connect-src 'self'`, and the comment above
   the CSP names no Google host), `client/test/web_csp_self_test.dart`
   (`engineFontHosts` becomes `{}`; a new test checks that the bootstrap file keeps
   the tokens the Flutter tool fills in and sets `fontFallbackBaseUrl`),
   `scripts/check_nginx_headers.py` (`CSP_HOSTS_ALLOWED = ()`) and
   `client/web/fonts/NOTICE.txt` (the status paragraph).

Verified: red at the first commit (the CSP test and the new bootstrap test fail,
4 checker failures), green at the second. On a scratch copy of `dea8dab` with the
patch applied: the two web test files pass (9 tests), the checker prints `OK`, the
Google-host grep prints nothing, and the "branch + patch" rows above show no
off-origin request on an open or a blocked network, text drawn, and the landing
screen identical to the base render.

## Integrator to-do

- INTEGRATOR TODO (pre-gate): apply the follow-up patch, or give
  `client/web/flutter_bootstrap.js` to a wave-4 cluster. Until then COMP-03 stays
  `PARTIAL` and a network that blocks `fonts.gstatic.com` gets no text.
- INTEGRATOR TODO: `THIRD_PARTY_NOTICES.md`: add the fonts under
  `client/web/fonts` (Noto and Roboto, SIL OFL 1.1; licence texts in
  `client/web/fonts/licenses`, copyright holders in `client/web/fonts/NOTICE.txt`).
- INTEGRATOR TODO: `PRIVACY.md` (row 3.5, when it lands): its list of hosts the
  web client contacts must match this outcome. `www.gstatic.com` and
  `fonts.googleapis.com` are no longer contacted; `fonts.gstatic.com` still is
  until the patch lands, and not after.
- INTEGRATOR TODO: stale text. `docs/usage/deployment.md` (lines 233 to 236) still
  lists CanvasKit from the gstatic CDN, the engine's fallback fonts and the Google
  Fonts stylesheet; `README.md` (lines 217 and 224, one per language) says the web
  build loads its faces from Google Fonts; comment-only fixes in
  `client/lib/theme/native_fonts.dart` (line 9, "web keeps index.html's CDN
  fonts") and the header comment of `client/lib/theme/web_fonts_web.dart`.
- INTEGRATOR TODO: `docs/usage/local-development.md`: an Android note (emulator
  `API_BASE` is `http://10.0.2.2:8080/api/v1`; a physical device needs
  `adb reverse tcp:8080 tcp:8080`; LAN addresses no longer work in debug).
- INTEGRATOR TODO: hand `loadWebFonts()` (`client/lib/theme/web_fonts_web.dart`,
  called from `client/lib/main.dart`) and the `cobalt.dart` web faces to the
  client owner (see the ceilings); record them in `docs/issues/` if wanted.
- INTEGRATOR TODO (optional): a Dart-side guard that refuses a non-https,
  non-loopback `API_BASE` in a release build (`client/lib/session/session.dart`);
  `expires 1y;` in the `/fonts/` block with a matching `Cache-Control` check in
  `scripts/check_nginx_headers.py`, if repeat visits matter.
- INTEGRATOR TODO: `docs/changelogs/README.md`,
  `docs/changelogs/index/2026-10.md` and
  `docs/changelogs/2026-10-18-rv4-remediation/index.md`: index this fragment.
- INTEGRATOR TODO: `docs/plans/README.md`, `docs/HANDOFF.md`: row 3.11 state
  (COMP-06 fixed at the manifest level; COMP-03 partial until the bootstrap file
  lands), and the first-CI-run follow-up (the `client` job builds the web target
  with the flag; the `nginx` job runs the extended checker on the distro nginx).
- INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md` and
  `docs/issues/README.md`: COMP-03 `PARTIAL` (one CSP host and one file), COMP-06
  `FIXED` with the `dart:io` ceiling, the mirror-versus-SDK gap and the missing
  cache headers.
- INTEGRATOR TODO: `docs/learning/README.md`: a lesson on CanvasKit fonts. CSS
  `@font-face` is ignored; the engine fetches its fonts with `fetch()`, so the CSP
  directive is `connect-src` and not `font-src`; a failed font is retried on every
  frame; only the loader's `config` can move the base URL.
