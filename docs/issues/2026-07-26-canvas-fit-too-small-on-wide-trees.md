# Canvas fit-to-view zooms wide roadmaps out too small

**Date:** 2026-07-26
**Status:** open — enhancement, not a defect. Recorded from the picky frontend
audit; no correctness or accessibility impact (zoom controls + Fit button let a
user recover), so deferred rather than gold-plated.

## Observation

On the 14-node demo roadmap, `fitTo` (`app/src/composables/usePanZoom.ts:50-65`)
clamps the fit scale to
`Math.min(availW / worldW, availH / worldH, 1)`. A tidy left→right tree is very
wide (depth × (CARD_W 244 + H_GAP 72)), so on a typical viewport the fit scale
lands around 0.35–0.5 and the first impression is a tiny, hard-to-read tree.
The node-card craft (seal chops, status chips, tapered edges, hairline rings)
only reads once the user manually zooms to ~0.75+.

## Why it isn't a one-line fix

`usePanZoom` is a shared composable with its own test suite, and its `MIN`
(0.25) is a *user zoom-out floor* — raising it would stop a user from zooming
out to see a whole large tree, a legitimate use case. The correct change is a
fit-policy change, separate from the zoom floor: clamp the **fit** scale to a
readable minimum (e.g. ~0.6) and, when the tree doesn't fit at that minimum,
anchor the top-left (root node) instead of centering a tiny tree — the standard
map "fit-to-bounds with a max-zoom-out" pattern.

## Bounded fix prompt

> Add an optional `minFit` (default ~0.6) parameter to `fitTo` in
> `app/src/composables/usePanZoom.ts`. Compute
> `s = clampScale(Math.min(availW/worldW, availH/worldH, 1))` as today; if
> `s < minFit`, set `s = minFit` and anchor the world's top-left at `pad.left /
> pad.top` (root visible) instead of centering. Pass `minFit` from the canvas
> `refit` path only — leave the user zoom floor (`MIN`) untouched. Verify with a
> `usePanZoom` unit test: a wide world fits at ≥ `minFit` with the root in-view,
> and a small world still fits centered at ≤ 100%. Re-screenshot the demo tree
> default view to confirm the first impression reads.
