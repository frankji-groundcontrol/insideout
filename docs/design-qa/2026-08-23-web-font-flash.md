# 2026-08-23 — Web first-refresh font flash

## User comment (verbatim)

> This is what happens the first time I refresh: it shows garbage
> characters for a second before the Chinese finally appears. Could you
> take a look and see what's going on?

## Resolution

Root cause: the CanvasKit renderer paints the first frame before the
Google Fonts CJK subsets (unicode-range woff2 slices, `display=swap`)
arrive, so every hanzi renders as a tofu box for a second or two on a
cold load. Fixed by gating the web first frame on font readiness:
`loadWebFonts()` polls `document.fonts.load()`/`check()` for the six
theme faces with a common-hanzi sample (pre-warming the slices the app
uses) and a 6s deadline, awaited in `main()` before `runApp`. The flash
becomes blank bootstrap time instead of garbage.

Implementation: [changelog](../changelogs/2026-08-23-web-font-flash.md).

## Affected files

- `client/lib/theme/web_fonts_web.dart` (new)
- `client/lib/theme/web_fonts_stub.dart` (new)
- `client/lib/main.dart`
- `client/pubspec.yaml`
