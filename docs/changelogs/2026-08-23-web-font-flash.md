# 2026-08-23 — Web first-refresh font flash gated on CJK font readiness

User report (design-QA): on a cold refresh the landing page shows
garbage/tofu boxes for a second before the Chinese renders
([design-qa](../design-qa/2026-08-23-web-font-flash.md)).

## Root cause

The hosted web build renders with CanvasKit, which paints the first
frame with whatever glyphs are available. The theme's `Noto Sans SC` /
`Noto Serif SC` families exist only as Google Fonts CSS faces
(`display=swap`, unicode-range woff2 subsets); on a cold load those
subsets arrive 1–2 s after the first frame, so every hanzi paints as a
tofu box until they land. Captured cold: first painted frame all boxes,
Latin text fine; correct a second later.

A single `document.fonts.load()` at startup is not enough: the async
`flutter_bootstrap.js` is not blocked on the fonts stylesheet, so the
`@font-face` rules may still be unparsed when Dart runs and `load()`
resolves empty.

## Change

- `client/lib/theme/web_fonts_web.dart` (new): polls
  `document.fonts.load()` for the six theme faces using a common-hanzi
  sample (pre-warms the unicode-range slices the app's screens use) and
  requires each call's resolved face array to be non-empty — `check()`
  and `document.styleSheets` both race the stylesheet parse and were
  measured releasing the gate onto tofu. 10 s deadline (cold subset
  completion measured ~7 s on a throttled link) so a dead font CDN
  degrades to today's behavior instead of a blank page.
- `client/lib/theme/web_fonts_stub.dart` (new): no-op off-web.
- `client/lib/main.dart`: awaits `loadWebFonts()` in the existing
  pre-`runApp` `Future.wait`, same conditional-import pattern as the
  URL strategy.
- `client/pubspec.yaml`: `web: ^1.1.1` promoted to a direct dependency
  (was transitive).

The flash becomes blank bootstrap time: cold capture shows blank frames
until the faces are ready, then the first painted frame is already
correct Chinese.

## Verification

- `flutter analyze` clean of new issues; `flutter test` 45/45.
- Built `flutter build web` and served locally; cold-cache headless
  Chromium capture (fresh browser context): pre-fix first painted frame
  all tofu boxes, post-fix first painted frame correct.
- Non-web targets unaffected: the gate is a web-only conditional import
  and `loadNativeFonts()` still registers the bundled variable fonts.
- The signal-race lessons (why `check()`/`styleSheets` failed and the
  `load()` array worked) are distilled in
  [learning](../learning/2026-08-23-browser-font-readiness-signals-race-stylesheet-parse.md).

## Operator notes

Ship with `railway up --service app` (client change only); no server or
database changes.
