# 2026-11-22 — Adding, linking and laying out steps on the Canvas

## Next step

> "what if I want to add a next for this layer"

The owner's screenshot marks an empty slot after Launch, the last step of the
flow. The Canvas could not add a step there.

## Parallel steps and links

> "what if I want to add a parallel node? and what if I want to link the node
> to other node?"

There was no parallel add. A link could only be made from the "Depends on…"
picker in the step menu.

## Layout

> "how this layout is determined? please come up with a layout algorithm for
> good representation"

The owner's screenshot shows the Spark flow:
- Matching → Chat crossed Onboarding → Launch.
- Chat floated between Onboarding and Matching.
- Launch was not level with what feeds it.

Columns came from the longest path, which was fine. The order inside a column
was only the manual position, and each column was centred on its own.

## Resolution

These are the owner's answers to the coordinator's questions:
- A "+ Next step" slot.
- "Add a step after" and "Add a parallel step", where a parallel step has
  the same links before and after.
- Drag-to-link from a handle.

The layout is now layered: each column is ordered to uncross its links, and
each box is aligned with what it links to
([plan](../plans/2026-11-22-canvas-add-and-link/2026-11-22-canvas-add-and-link.md)).

## Status

Fixed: see the [changelog](../changelogs/2026-11-22-canvas-add-link-and-layout.md).
