# 2026-10-06 — `server/internal/store/roadmap.go` over the ~350-line budget

> **Resolved 2026-10-10.** Split as targeted: `server/internal/store/roadmap.go` is 309 lines and
> `roadmap_move.go` 103, with the tests along the same seam (`roadmap_test.go` 300,
> `roadmap_move_test.go` 116, `roadmap_attribution_test.go` 159). A pure move,
> checked line by line (`7a02c27`); see [backend splits](../changelogs/2026-10-10-close-open-gaps/backend-splits.md).
>
> The original record follows, unedited.

`roadmap.go` is 403 lines. It already mixed node CRUD, the status
machine, and the move cycle guard; the sibling-shift added for
[BUG-017](2026-10-06-bug-017-roadmap-move-collides-sibling-position.md)
pushed it past the budget.

## Why not split now

The four simulation fixes landed together and the store tests pin the
behavior. Splitting in the same change would bury the bug fixes.

## Target structure

- `roadmap.go` — node type, scan, create, list, update (~250 lines).
- `roadmap_move.go` — `MoveRoadmapNode`, the cycle guard, and the
  sibling shift (~150 lines).

Split at the next structural touch of this file.
