# 2026-07-25 — Canvas failure feedback shipped as a banner, not the specced toast

**Status:** open, deferred split from collab Workstream A3. The corrective note
below also amends the T4 line in the
[collab plan](../plans/2026-07-24-roadmap-canvas-collab.md).

## What the collab plan specced (A3)

A minimal `useToast` composable and a fixed **bottom-center** outlet: error
glyph + text in `text-fg-danger` (never vermilion), `role="alert"` /
`aria-live="assertive"`, 5s auto-dismiss, a 44px close button, max 3 stacked
(newest on top). Every mutation — status, edit, create, add-child, move,
delete, load, expand — surfaces through it. Plus a *safer delete*: the confirm
states the client-computed descendant count ("this deletes N nodes").

## What actually shipped (2026-07-25 hardening pass)

No `useToast` exists in `app/src`. Failure feedback rides a single in-canvas
error banner instead:

- `RoadmapCanvas.vue` holds a `loadError` string ref that drives an
  error-state overlay with a Retry button (distinct from the empty state).
- `RoadmapCanvasNode.vue` catches every CRUD handler and
  `emit('error', msg)` (`// ponytail:` minimal surface) so the canvas banner
  shows it; recovery is "reload the tree".
- Delete still uses a **static** `window.confirm(roadmap.deleteConfirm)` — the
  descendant-count confirm was not built (the M6 fix only added focus restore).

So the A3 *intent* — no mutation fails silently — is met: where the pre-pass
code had handlers with no `catch` at all (rejections went unhandled), every
path now surfaces. What is deferred is the specced *shape* (the stacked,
auto-dismissing, screen-reader-announced toast) and the count in the delete
confirm.

## Why the banner is an acceptable stopgap

The banner is visible, retryable, and on-brand (no new vermilion), and it
reuses one ref instead of introducing a toast outlet + queue the single-player
canvas does not otherwise need. The cost is UX, not correctness: a banner
overlay is heavier than a transient toast, it is scoped to the canvas (a
mutation failure elsewhere would not show), and it does not stack or
auto-dismiss. The delete confirm without a count understates blast radius on
deep branches.

## Fix (when picked up)

Build the specced `useToast` + bottom-center outlet and route the canvas's
`onNodeError` / `loadError` (initial load only) through it; replace the static
delete confirm with the descendant-count string (reuse the `isDescendant` tree
walk already in `tidyTree.ts`). Then this banner overlay can be removed. This
is the natural home for the work already pencilled into collab T4/A3.

## Plan correction

The collab plan's T4 line reads "A1 partial payloads (sparse body) + A3
useToast + safer delete". The contract-critical half — sparse partial payloads
and the build-count threading — did ship and is pinned by the D9 sparse-PATCH
test, so the box stays checked; but "A3 useToast + safer delete" shipped as
the banner + static confirm described above, not the specced toast + count.
Treat the toast + count as open under this issue.
