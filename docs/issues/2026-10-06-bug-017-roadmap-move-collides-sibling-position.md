# BUG-017 — roadmap move can land two siblings on the same position

## Symptom

`insideout roadmap move --position 0 <node>` succeeds, but the node
that already occupied position 0 keeps it. The list then shows two
siblings with `position: 0`. Seen 2026-10-06 on the live sim server,
both through the CLI and through MCP `roadmap_list` — the surfaces
agree, so the collision is in the store, not the projection.

## Cause

`Store.MoveRoadmapNode` writes the requested position onto the moved
row and does not shift its siblings
(`server/internal/store/roadmap.go`). Create avoids this with
`MAX(position)+1`; move does not. The list orders by position, so a
tie makes the displayed order depend on `created_at` and a later move
cannot express "put this first" unambiguously.

## Status

Fixed 2026-10-06. `MoveRoadmapNode` closes the gap the node leaves and
opens the slot it takes, inside the per-project lock it already held.
`TestRoadmap_MoveShiftsSiblings` pins the order. On the live sim
server, moving the song node to position 1 left Mira's three roots at
0, 1, and 2.
