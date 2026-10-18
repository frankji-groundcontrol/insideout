# There is no Change Proposal surface for delivery-revealed product change

Open, recorded 2026-10-18 (review-4 wave 3, finding COMP-08; row 116 of the
[promise register](../review/2026-10-10-multirole-3/50-promise-register.md),
a gap no earlier review list carried). Context in the
[promise-gap enumeration](2026-10-11-product-promise-gaps.md).

## What PRODUCT.md promises

> "When delivery reveals a product change, a person or Agent creates a Change
> Proposal that shows the impact on views, journeys, Roadmap, and existing
> work. A human accepts it before the baseline changes."
> ([PRODUCT.md](../../PRODUCT.md):326-328)

## What exists today

- The generic agent proposal flow works: `POST /api/v1/agent/proposals/{uid}/decision`
  (`server/internal/api/agent_routes.go:22`), kinds structure, scope or priority
  (`agent_routes.go:155`), a human decision recorded immutably
  (`server/internal/store/proposal_decisions.go:29`) and items applied once on
  acceptance (`store/proposal_items.go:88`).
- No Change Proposal exists as a concept: no kind, no route, no client view,
  and nothing shows the impact on views, journeys, Roadmap or existing work
  (a case-insensitive search for "change proposal" over `server/`, `client/lib/`,
  `docs/usage/` and `skills/` finds nothing).

## Fix prompt

Red first: an API test that a proposal of kind `change` returns an `impact`
block (PRD sections, audience views and roadmap nodes it touches) fails today.
Smallest slice: add the kind to the existing proposal flow, compute the impact
read-only from the sections and nodes the proposal's items reference, and keep
the human accept as the only path that moves the baseline. No second store.
