# 2026-10-06 — Breakout is the app icon

The user named
[breakout.svg](../design-system/cobalt/icon-candidates/breakout.svg)
as the logo. The Flutter client icons now use that mark. Nothing in the
server or token contract changed. Not deployed.

## What changed

Rasterized the SVG onto the light ground `#F7F8FA`, with the mark at
66% of the tile so the offset square and its navy slab stay inside a
circular launcher mask. Replaced:

- web `client/web/icons/Icon-192.png`, `Icon-512.png`, and both
  maskable sizes; `client/web/favicon.png` is the same mark at 32px on
  a transparent ground
- iOS `client/ios/Runner/Assets.xcassets/AppIcon.appiconset/`
- Android `client/android/app/src/main/res/mipmap-*/ic_launcher.png`

`client/web/manifest.json` no longer uses the Flutter blue
(`#0175C2`) or the placeholder description. Name is InsideOut,
background `#F7F8FA`, theme `#3B5BFF`.

The candidate folder is unchanged. The pick is recorded in
[design-qa](../design-qa/2026-10-06-app-icon.md). The open install
issue is closed by this file
([issue](../issues/2026-10-06-app-icon-not-installed.md)).

## Verification

`rsvg-convert` wrote the master. The 1024 iOS icon and the 512 web
icon are RGB, corner pixel `#F7F8FA`. A pixel in the offset square is
`#3B5BFF`. The favicon is RGBA and its corner alpha is 0. No Flutter
test was run; these are static assets.

## Operator

Nothing to migrate. The hosted web app keeps the previous icon until
the client image is rebuilt and deployed.
