# Roadmap nodes have no dependency edges, so no blocked path or rebaseline

Open, recorded 2026-10-11 (review-3 wave 4, finding FEAT-05; slice 7 of the
[promise-gap set](2026-10-11-product-promise-gaps.md)).

## What PRODUCT.md promises

> "A dependency delay shows the affected path and a proposed time Diff; a
> human confirms the rebaseline." (PRODUCT.md:300-302)

## What exists today

- The roadmap migrations carry parent links, deadlines and
  acceptance/owner — `server/db/migrations/20260722120000_roadmap_nodes.sql`,
  `20260822120000_roadmap_deadlines.sql`,
  `20261005120000_roadmap_acceptance_owner.sql` — no dependency column or
  table.
- `depends_on` appears nowhere in the migrations or the store; the only
  "dependenc" matches in `server/internal` are audience/readiness prose
  (`server/internal/audienceview/audienceview.go:45`,
  `server/internal/readiness/readiness.go:54`) — the model is docs-only.

## Failure

An item blocks on another team's work; the only expression available is a
text note, so Progress cannot show the blocked path and no rebaseline
proposal is ever computed.

## Slice order

7 of the set — after the lenses record that will surface it (the Product
lens lists "dependencies, blockers").

## Fix prompt

Red first: a store test creating a `depends_on` edge between two roadmap
nodes fails — no table. Smallest slice: a `roadmap_node_edges` table (node,
depends-on node) with set/clear on the node update path;
`roadmaptime.Assemble` marks a node blocked when a dependency it waits on is
not done, and the Progress view shows a blocked indicator. The time-Diff
rebaseline flow stays deferred until edges exist — it needs the path-impact
walk over real edges.
