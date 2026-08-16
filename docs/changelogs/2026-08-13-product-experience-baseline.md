# 2026-08-13 — Canonical product-experience baseline

The completed product interview is now consolidated into one authoritative
target-experience document instead of remaining only in conversation history
or competing product summaries.

## What changed

- Rewrote [`PRODUCT.md`](../../PRODUCT.md) as the canonical product-experience
  baseline: version-first coaching, audience projections, evidence and
  assumptions, human product version control, project-level authority, one
  role-aware Roadmap Graph, deadline-bound Now work, Git evidence, and shared
  Web/CLI/MCP/Agent semantics.
- Defined the minimum coherent end-state loop and explicit non-goals so the
  baseline can guide vertical-slice implementation without introducing parallel
  PRDs, Roadmaps, or evidence systems.
- Labeled delivery defaults separately from confirmed experience decisions.
  The current implementation is explicitly described as a foundation, not as
  already satisfying the target.
- Reduced [`docs/INIT.md`](../INIT.md) to a compatibility pointer and linked
  the canonical baseline from the documentation map and README. Added one
  dated target-delta banner to `docs/TODO.md` so its historical completion
  record is not mistaken for the new target specification.

The baseline supersedes the former fixed eight-section PRD, rigid four-stage
interview, Workspace-admin product approval, Workspace-visible Idea, untimed
Roadmap, and independent GitHub timeline assumptions. Historical plans were
left unchanged as records of their time.

## Verification

- Reviewed against the confirmed interview decisions and current code/docs.
- Checked relative Markdown links and inspected the final diff.
- Documentation only: no source, schema, API, configuration, or runtime
  behavior changed; no application tests were required.

## Operator notes

None. A future implementation plan should use `PRODUCT.md` as product
authority and architecture documents as current-state authority.
