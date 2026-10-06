# 2026-10-06 — Web boot gate no longer hangs on an unreachable font host

## What changed

`client/lib/theme/web_fonts_web.dart`: each `document.fonts.load()` in
the boot-gate poll now times out after 2 seconds and counts as "not
loaded", so the existing 10-second deadline can actually expire. Before
this, one `fonts.load()` that never settles (Google Fonts unreachable
or returning 404s) pinned the boot `Future.wait` forever and the app
never left the loading spinner
([BUG-016](../issues/2026-10-06-bug-016-web-boot-gate-hangs-on-font-load.md)).

## Why

Found during the live user simulation
([plan](../plans/2026-10-05-user-simulation-cli-mcp.md)): headless and
in-app browsers both stuck on the boot mark with zero API traffic. The
page's `FontFaceSet` showed 737 faces, all `unloaded`, and
`fonts.gstatic.com` answered 404 for the face URLs.

## Verification

Rebuilt the release web bundle
(`flutter build web --dart-define=API_BASE=…`) and drove it: the
landing page renders, and a full registration through the web form
returns 201 and lands on `/dashboard`.

## Operator note

None for deployed environments where Google Fonts loads normally. Where
it does not, the app now boots with fallback glyphs instead of hanging;
CJK text renders as boxes until the fonts are reachable.
