# InsideOut Product Definition

This file is a compatibility pointer. It no longer carries a second product
definition.

- The canonical **target product experience** is
  [`PRODUCT.md`](../PRODUCT.md).
- The **currently running system** is described by
  [`docs/architecture/`](architecture/index.md) and
  [`docs/HANDOFF.md`](HANDOFF.md).
- Delivery status lives on the task board,
  [`docs/plans/README.md`](plans/README.md); known gaps and debt in
  [`docs/issues/README.md`](issues/README.md), with the shipped-system
  baseline in [`docs/TODO.md`](TODO.md).
- Historical decisions for the 2026-07-20 JuanLeMe → InsideOut rewrite remain
  in [`docs/plans/2026-07-20-go-rewrite/`](plans/2026-07-20-go-rewrite/README.md).

The product-experience baseline supersedes the old fixed eight-section PRD,
rigid four-stage coaching, Workspace-admin PRD approval, and linear
Idea-to-shipped assumptions previously duplicated here. Those descriptions
may still appear in the README, technical tracker, and historical plans when
they describe today's implementation or what existed at the time; they are not
target-product authority.

## What this file used to define

This file carried the product definition from the former product-definition
era — the Nuxt 4 SSR application with its Ink & Seal bilingual interface and
a directly integrated PRD Coach. The Nuxt tree was deleted 2026-08-18 and
production is now the Flutter Cobalt web client over the Go API and
PostgreSQL with row-level security. Treat this section as historical context
only: the target experience lives in [`PRODUCT.md`](../PRODUCT.md), and the
running system (Go API, PostgreSQL/RLS, Flutter Cobalt web) is authoritative
in [`docs/architecture/`](architecture/index.md).
