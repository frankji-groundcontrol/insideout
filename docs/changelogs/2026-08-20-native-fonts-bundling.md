# 2026-08-20 — Native Noto fonts bundled (Ink & Seal, iOS/Android path)

Plan: [restore-ink-seal](../plans/2026-08-19-restore-ink-seal.md). Web was
already served by `index.html`'s Google Fonts link; native targets had no
CJK serif at all.

## What changed

- `client/assets/fonts/`: Noto Serif SC and Noto Sans SC variable TTFs
  (25.1 MB + 17.8 MB; one file per family covers every weight the theme
  uses).
- `client/lib/theme/native_fonts.dart`: `loadNativeFonts()` registers the
  two families via `FontLoader` at startup, **only when `!kIsWeb`**; called
  from `main()` alongside hydration.
- `InkSeal.sans` / `InkSeal.serif` became `kIsWeb`-gated getters: web keeps
  the CDN family names, other targets use the bundled `*Native` names.
- `pubspec.yaml`: fonts live under plain `assets:` — **not** the `fonts:`
  section.

## Why not the pubspec `fonts:` section

Pubspec-registered fonts are engine-eager on every platform: the first
attempt put the 43 MB of TTFs into `FontManifest.json`, and a web build
came out at 82 MB with the fonts fetched at startup. As plain assets they
are lazily fetched; the `kIsWeb` guard means a web browser never requests
them (they only pad the static deploy artifact). iOS/Android load them
through `FontLoader` before the first frame.

## Verification

- `flutter test`: 43/43 pass with the gated families.
- Web build: `FontManifest.json` lists only icon fonts; Noto TTFs present
  as unreferenced assets.
- iOS simulator build: same clean `FontManifest.json`, TTFs present under
  `flutter_assets/assets/fonts/` for the startup loader.
- **Pending**: visual render sign-off on a real iOS/Android build. The
  local CoreSimulator wedged this session (app install/launch would not
  stay alive; screenshots kept showing springboard). Wiring, bundles, and
  tests are green; the plan keeps an open visual-check item.

## Notes

- Railway `app` (Flutter web host) was still serving the 2026-08-18
  build — stale through the whole Ink & Seal slice. Redeployed from
  `main` (f8c5e0f) with `railway up --service app`: new build verified
  live by fingerprint (font asset present under `/assets/assets/fonts/`,
  icon-only FontManifest), `/` 200, same-origin `/api` proxy 200.
- Flutter's tooling bumped the iOS deployment target 13.0 → 15.0
  (`ios/Podfile`, `ios/Runner.xcodeproj`) during the first local build —
  its standard migration, kept.
- The web deploy artifact grows by the font files' size; browsers do not
  download them. If that matters later, a web-flavored entrypoint can
  exclude the assets.
- CocoaPods is now installed locally (`brew install cocoapods`, 1.17.0) —
  required for any iOS build on this machine.
