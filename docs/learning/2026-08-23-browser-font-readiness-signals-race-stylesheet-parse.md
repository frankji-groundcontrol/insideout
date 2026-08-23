# Browser font-readiness signals race the stylesheet parse

**Learned** 2026-08-23 while gating the Flutter web first frame on CJK
font readiness ([changelog](../changelogs/2026-08-23-web-font-flash.md)).

## The lesson

To delay a CanvasKit first frame until web fonts exist, only one
browser signal is free of the stylesheet-parse race: the array
`document.fonts.load(font, text)` **resolves with**. It is empty while
no matching `@font-face` rule exists and non-empty once the matching
faces finish loading. The tempting signals are not safe:

- `FontFaceSet.check(font, text)` returns **true** when no matching
  face is registered ("a system font will do"), i.e. vacuously true
  before the CSS parses — releasing any gate onto fallback glyphs.
- `document.styleSheets` lists a `<link>`'s sheet (with its `href`) as
  soon as the element parses, long before its rules load.
- An async bootstrap script is not blocked on a render-blocking
  stylesheet, so Dart `main()` can easily run before the font CSS
  parses; which side wins the race varies run to run.

## When to apply

Any web first-frame/FOIT gate over CDN fonts (Flutter CanvasKit or
otherwise), and any debugging of "font gate works locally but flashed
in production" — measure with a time-stamped sampler
(`performance.now()` + the candidate signals) instead of trusting one
green run; a 100 ms-interval sample caught all three races here.

## Evidence

Cold-load sampling showed `check()` true at t≈0.1–1.6 s (CSS
unparsed), false 2.1–4.6 s (parsed, loading), true again from ≈6.6 s
(loaded). Gates built on `check()` or `styleSheets` released during
the first window; the `load()`-array gate released only at the third.
