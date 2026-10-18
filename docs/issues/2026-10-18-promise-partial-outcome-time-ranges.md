# Product outcomes carry no target time range

Open, recorded 2026-10-18 (review-4 wave 3, finding COMP-08; row 100 of the
[promise register](../review/2026-10-10-multirole-3/50-promise-register.md)).
A PARTIAL promise with no dated record until now; context in the
[promise-gap enumeration](2026-10-11-product-promise-gaps.md).

## What PRODUCT.md promises

> "Product outcomes carry target time ranges. Milestones and execution items
> carry explicit deadlines. An item without a deadline cannot enter Now."
> ([PRODUCT.md](../../PRODUCT.md):292-293)

## What exists today

- Every roadmap node has one nullable `deadline timestamptz`
  (`server/db/migrations/20260822120000_roadmap_deadlines.sql:4`); deadline
  pressure derives from it alone (`server/internal/roadmaptime/roadmaptime.go:21`).
- The "no deadline, not in Now" half is implemented (`roadmaptime.go:62`,
  `needsDeadline` in `server/internal/api/roadmap_progress.go:19`).
- There is no outcome-level range: no start/end field on any node, no schema
  column, no client control (a search for time-range columns in
  `server/db/migrations/` and `server/internal/` finds none).

## Fix prompt

Red first: a store test that sets a target range on a top-level outcome node
and reads it back fails today. Smallest slice: nullable `target_start` and
`target_end` on `roadmap_nodes`, accepted only on root and first-level nodes,
`start <= end` enforced by a CHECK; show the range in the roadmap header and
in the agent context; keep `deadline` the only pressure input for leaf items.
