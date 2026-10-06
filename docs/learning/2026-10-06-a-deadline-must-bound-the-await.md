# A deadline must bound the await, not just the loop around it

## What was learned

A timeout wrapped around a polling loop does not time out the thing
being polled. If the awaited call can pend forever, the loop body never
returns to check the deadline, and the deadline is dead code. The
timeout has to sit on the await itself.

## Evidence

`loadWebFonts()` (`client/lib/theme/web_fonts_web.dart`) loops until a
10-second deadline, awaiting `document.fonts.load()` each pass. When
`fonts.gstatic.com` served 404s, `fonts.load()` never resolved, so the
loop never rechecked the clock and the app's boot gate never lifted —
the spinner ran indefinitely with no error and no network call of the
app's own
([BUG-016](../issues/2026-10-06-bug-016-web-boot-gate-hangs-on-font-load.md)).
Adding a per-await timeout made the existing deadline work.

## Scope

Any poll, retry, or gate whose bound is checked between awaits rather
than on them: font/asset readiness, health probes, "wait until ready"
boot sequences.

## When to apply it again

When reviewing a loop that claims to give up after N seconds, check
whether every await inside it can outlive N. If one can pend without a
bound, the deadline is decorative.
