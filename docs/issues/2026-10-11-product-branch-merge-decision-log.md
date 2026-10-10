# Product Branch / Merge and the Decision Log are promised but not built

Open, recorded 2026-10-11 (review-3 wave 4, findings CQ-02 and SEC-04;
slice 1 of the [promise-gap set](2026-10-11-product-promise-gaps.md)).

## What PRODUCT.md promises

> "**Product Branch / Explore direction:** an alternative product hypothesis,
> such as a different core user, value proposition, business model, or scope."
> (PRODUCT.md:206-207)
>
> "**Merge / Combine conclusion:** a human decision to bring a Product Branch
> into the main product line." (PRODUCT.md:209-210)
>
> "A Decision Log accompanies meaningful Commits, Merges, overrides, and
> reversed decisions. It records who decided, why, the evidence considered,
> and the alternatives not chosen." (PRODUCT.md:218-220)

## What exists today

- No product-branch machinery anywhere: `rg -i "product.?branch|explore
  direction"` over `server/internal` and `client/lib` returns nothing
  (checked 2026-10-11) — no route, store type, CLI verb or MCP tool.
- The 36 migrations hold no branch or merge table; PRD versions are a linear
  revision list (`server/internal/store/prd_revisions.go`).
- The Decision Log exists only as the Commit note's free text
  (`server/cmd/insideout/cli_version.go:124`,
  `server/internal/mcpserver/product_tools.go:123`) plus agent-proposal
  accept/reject rows (`server/internal/store/proposal_decisions.go:13`);
  the version card renders the note
  (`client/lib/features/prd/versions_commit_details.dart:13`) — no
  who/why/evidence/alternatives surface.

## Slice order

1 of the set — the keystone: the doc's Commit/Merge language and the persona
hypotheses hang off Branch (CQ-02: "build the Product Branch slice first
since Commit/Merge language pervades the doc").

## Fix prompt

Red first: a store test that branches a committed PRD version (new row
referencing the source commit) fails — no table. Smallest slice: a
`prd_branches` table (source commit, title, current driver) with create/list
routes and a "Branch" action on the versions page. Merge and the Decision Log
surface build on the same rows: Merge is a human-confirmed combine back into
the mainline; Decision Log rows record who/why/evidence/alternatives for
Commits, Merges and overrides. A Product Branch is not a Git branch
(PRODUCT.md:223-225) — no git plumbing in this slice.
