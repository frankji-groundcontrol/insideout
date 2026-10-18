# The Focus surface (per-session pointer, team Now) is promised but not built

Open, recorded 2026-10-11 (review-3 wave 4, finding FEAT-06; slice 8 of the
[promise-gap set](2026-10-11-product-promise-gaps.md)).

## What PRODUCT.md promises

> "Focus is a non-exclusive pointer for one person or one Agent session,
> with one main Focus per session; team Now is the shared set that
> aggregates active Focuses." ([PRODUCT.md](../../PRODUCT.md):246-249)

## What exists today

- Focus exists only as an optional input of the agent-context briefing:
  `server/internal/agentcontext/agentcontext.go:41`
  `FocusNodeID string // optional; scopes the assembly`.
- No focus route or store: `focus` matches nothing in the API route
  inventory or `server/internal/store/roadmap.go` (checked 2026-10-11);
  team Now is per-node in_progress status plus canvas presence
  (`server/internal/api/presence_routes.go:22-25`).

## Failure

Two teammates each believe they are driving the same item; nothing records
who currently focuses where, and an agent asked for context gets no notion
of the humans' current focuses to align with.

## Slice order

8 of the set — reuses the presence registry's TTL, so it lands after the
graph slices it points into.

## Fix prompt

Red first: an API test that `POST /projects/{pid}/focus {node}` then GET
returns the caller's focus — one main Focus per session, non-exclusive
across people — fails. Smallest slice: a per-session focus pointer reusing
the presence registry's TTL (`server/internal/presence`); the Progress view
shows "team Now" headers from the aggregated focuses;
agentcontext's `FocusNodeID` defaults to the caller's focus when unset.
Focus is a pointer, never ownership — the one-human-owner rule stays on
the node (PRODUCT.md:246-247).
