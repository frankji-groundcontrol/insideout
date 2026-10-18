# Overdue work can be closed or moved without a recorded reason

Open, recorded 2026-10-18 (review-4 wave 3, finding COMP-08; row 103 of the
[promise register](../review/2026-10-10-multirole-3/50-promise-register.md),
which names it parked review-2 FEAT-09 / PM-18 but no dated record). Context in
the [promise-gap enumeration](2026-10-11-product-promise-gaps.md).

## What PRODUCT.md promises

> "Overdue work is never silently closed. A human can continue it, change the
> deadline, transfer it, or abandon it, and records the reason."
> ([PRODUCT.md](../../PRODUCT.md):299-300)

## What exists today

- Overdue is a visible pressure state (`server/internal/roadmaptime/roadmaptime.go:17`),
  shown by `client/lib/features/project/roadmap_progress_view.dart:178` and
  `roadmap_node_card.dart:87`.
- Nothing else happens to an overdue node: the plain PATCH changes its status,
  deadline or owner with no reason (`server/internal/store/roadmap.go:245`),
  so an overdue node can be marked done or re-dated silently. There is no
  transfer or abandon action. The only reason-bearing move is the human
  confirm into `validated` (`server/internal/store/roadmap_validation.go:60`).

## Fix prompt

Red first: a store test that moving an overdue node to done, changing its
deadline or changing its owner without a reason is refused fails today.
Smallest slice: an append-only `roadmap_node_resolutions` row (node, kind
continue / new_deadline / transfer / abandon, reason, actor, time) written by
one endpoint, required whenever the node's pressure is `overdue` and the PATCH
touches status, deadline or owner. Abandon maps to the existing `pending`
status plus the recorded reason, so the status machine does not change.
