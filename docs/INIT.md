# InsideOut Product Definition

This file is a compatibility pointer. It no longer carries a second product
definition.

- The canonical **target product experience** is
  [`PRODUCT.md`](../PRODUCT.md).
- The **currently running system** is described by
  [`docs/architecture/`](architecture/index.md) and
  [`docs/HANDOFF.md`](HANDOFF.md).
- Delivery status and known implementation gaps live in
  [`docs/TODO.md`](TODO.md).
- Historical decisions for the 2026-07-20 JuanLeMe → InsideOut rewrite remain
  in [`docs/plans/2026-07-20-go-rewrite/`](plans/2026-07-20-go-rewrite/README.md).

The product-experience baseline supersedes the old fixed eight-section PRD,
rigid four-stage coaching, Workspace-admin PRD approval, and linear
Idea-to-shipped assumptions previously duplicated here. Those descriptions
may still appear in the README, technical tracker, and historical plans when
they describe today's implementation or what existed at the time; they are not
target-product authority.

## Current technical foundation

InsideOut currently runs as a Go API over PostgreSQL with row-level security,
a Nuxt 4 SSR application, a direct Anthropic PRD Coach integration, and an Ink
& Seal bilingual interface. This is implementation context, not a constraint
on the target product model. See the architecture documents for authoritative
details.
