# A node can be completed without evidence or a reason

Open, recorded 2026-10-18 (review-4 wave 3, finding COMP-08; row 126 of the
[promise register](../review/2026-10-10-multirole-3/50-promise-register.md),
a gap no earlier review list carried). Context in the
[promise-gap enumeration](2026-10-11-product-promise-gaps.md).

## What PRODUCT.md promises

> "Completion links to verifiable evidence and may be overridden by a human only
> with a reason." ([PRODUCT.md](../../PRODUCT.md):351-352)

## What exists today

- Evidence attaches to nodes (`AddRoadmapEvidence`,
  `server/internal/store/roadmap_evidence.go:29`), but nothing requires it.
- The status machine lets any plan writer (`requirePlanWriter`) move `in_progress` to `done` with
  neither evidence nor reason (`legalStatusTransitions`,
  `server/internal/store/roadmap.go:225`; the store test
  `roadmap_acceptance_owner_test.go:133-134` walks exactly that move).
- Only `validated` takes a human reason (`roadmap_validation.go:60`), and it
  starts from `done`, so the weaker step is the unguarded one.

## Owner decision needed

Should moving a node from `in_progress` to `done` require linked evidence or a
short reason? Recommendation: require either at least one linked evidence row
or a reason of one sentence, recorded on the node (the `validated` reason
column is the precedent), enforced server-side in `UpdateRoadmapNode` so the
CLI, MCP and web clients all meet the same rule. Until decided, this stays
open and unimplemented.

## Fix prompt

Red first: a store test that `in_progress` to `done` with no evidence and no
reason is refused fails today. Then add the `completion_reason` column and the
rule above; the client status menu asks for the reason when no evidence exists.
