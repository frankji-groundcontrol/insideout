# The Roadmap graph is not tied to an explicit PRD Commit baseline

Open, recorded 2026-10-18 (review-4 wave 3, finding COMP-08; row 94 of the
[promise register](../review/2026-10-10-multirole-3/50-promise-register.md),
which lists it among the rows no review carried). Context in the
[promise-gap enumeration](2026-10-11-product-promise-gaps.md).

## What PRODUCT.md promises

> "It is one shared, versioned, hierarchical graph tied to an explicit PRD
> Commit baseline." ([PRODUCT.md](../../PRODUCT.md):264-265)

## What exists today

- The graph is shared and hierarchical: `roadmap_nodes` with a `parent_id`
  self-reference per project (`server/db/migrations/20260722120000_roadmap_nodes.sql:10`).
- It is not tied to a Commit: no node or project row references a PRD Commit
  (`roadmapNodeColumns`, `server/internal/store/roadmap.go:74`, has no commit
  field), and "versioned" amounts to `updated_at` plus per-node attribution.

## Fix prompt

Red first: a store test that a roadmap read reports the Commit the graph is
based on fails today. Smallest slice: a nullable `baseline_commit_id` on
`projects` (or the roadmap root), set when a human confirms a Commit as the
baseline, returned by the roadmap read and shown in the roadmap header; later
edits do not change it, so the baseline can be diffed against the graph. The
time-Diff rebaseline stays with the dependency-edges record.
