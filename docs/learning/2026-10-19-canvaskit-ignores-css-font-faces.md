# CanvasKit ignores CSS font faces, so self-hosting fonts is a loader setting

**Date:** 2026-10-19
**Source:** [multirole review 4](../changelogs/2026-10-18-multirole-review-4.md), findings COMP-03 and COMP-06 (what the web client loads from third parties)
**Fixed in:** [wave 3, client assets](../changelogs/2026-10-18-rv4-remediation/w3-client-assets.md)
([privacy notice](../../PRIVACY.md))

## The lesson

A Flutter web app on CanvasKit draws text on a canvas with fonts the engine
downloads itself. A CSS `@font-face` or a `<link>` to a font stylesheet changes
nothing it draws, so the fix for "the page contacts Google hosts" is not a CSS
edit.

- **The option is `fontFallbackBaseUrl`,** and it is read from the loader's
  `config` in `_flutter.loader.load({config})`. Setting the legacy global
  `window.flutterConfiguration` does nothing: the engine still asked the Google
  host (1,123 blocked requests in 24 s, none to the mirror). The CSP bars inline
  script, so the config needs a custom `client/web/flutter_bootstrap.js`.
- **The mirror must be the engine's whole font table,** not the families this UI
  uses. The engine picks a font by browser language and by the user's text
  (zh-TW asks for Noto Sans TC, an arrow for Noto Sans Symbols, an emoji for Noto
  Color Emoji). A missing file is a 404 it retries every frame: 4,456 to 6,997
  blocked requests in 25 to 34 s. A 102-file mirror was rejected by a red test.
- **`--no-web-resources-cdn` bundles CanvasKit** (it writes
  `"useLocalCanvasKit":true`); the build copies `canvaskit/` into `build/web`
  either way, so the flag only points the loader at it.
- **Then the CSP can name no host.** The engine fetches fonts with `fetch()`, so
  the directive that mattered is `connect-src`, not `font-src`; `client/nginx.conf`
  now sends `default-src 'self'` and `connect-src 'self'`, and
  `scripts/check_nginx_headers.py` allows no host at all.

## When to apply

- Any request to remove a third-party host from a Flutter web page: find which
  engine path makes the request (CanvasKit, fonts, icons) before editing CSS.
- Any Flutter SDK bump: the engine's font table can change, and the mirror has to
  follow it. A guard that compares the two is open residue.

## Evidence

The cold-load request list before the change named `www.gstatic.com` (CanvasKit),
`fonts.googleapis.com` (a stylesheet CanvasKit never reads) and `fonts.gstatic.com`
(the Roboto and Noto fallback fonts). The rejected designs (nginx proxying a miss
to the Google host, a Dockerfile step writing the bootstrap file) are in the
fragment's design spike. Still open: `loadWebFonts()` waits up to 10 s for CSS
faces CanvasKit ignores (R3-30).
