# BUG-016 — Web boot gate hangs forever when Google Fonts is unreachable

## Symptom

The Flutter web client never leaves `BootLoadingPage` (cobalt disc +
spinner) when `fonts.gstatic.com` does not serve the font files. No
error, no timeout, no API call — the app is simply unusable. Found
2026-10-06 while driving the web client for the user simulation
([plan](../plans/2026-10-05-user-simulation-cli-mcp.md)).

## Root cause

`client/lib/theme/web_fonts_web.dart` polls `document.fonts.load()`
inside a loop bounded by a 10-second deadline. The deadline only bounds
how often the loop iterates; it does not bound the await itself.
`fonts.load()` resolves empty while a face is unparsed, but it **never
settles** when the font host is unreachable (here the host answered but
every face URL 404'd — 737 declared faces, zero loaded). One pending
await pins `Future.wait` in `main()`, and `BootController.complete()` is
never called, so the gate stays down. The gate's own contract says a
failed hydrate must lift it; the font path violates that.

## Fix

Each `fonts.load()` now carries a 2-second timeout that counts as "not
loaded", so the outer deadline can give up and the app boots with
fallback glyphs. Verified live: the release build reaches the landing
page, and registration through the web UI succeeds (201).

## Follow-up

The timeout lifts the gate but leaves CJK as tofu boxes until the fonts
arrive — acceptable degradation, but a self-hosted or bundled CJK
subset would remove the external dependency entirely. Not done here.
