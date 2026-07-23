# 2026-07-22 — Idea → Reality: branched roadmap, MVP build, GitHub sync, UI reframe

Large change record. InsideOut pivoted from "a PRD tool" to "idea → reality":
the PRD is one step, and after it the product now helps you build. Three key
capabilities landed, plus a UI/UX prettify pass.

- [summary.md](summary.md) — what changed, subsystem by subsystem.
- [verification.md](verification.md) — the verification actually performed.

## Primary sources

- [Plan](../../plans/2026-07-22-idea-to-reality.md) (the live checklist this was driven from)
- Migration: `server/db/migrations/20260722120000_roadmap_nodes.sql`

## The pivot in one paragraph

Before: idea → coach → PRD, and separately projects → manual progress updates.
Now the spine is continuous: **idea → PRD → project with a branched roadmap
tree → progress (manual + GitHub) → shipped MVP.** A project is the "make it
real" container; its roadmap is a tree whose root is the MVP goal and whose
children are parallel workstreams; progress (manual updates and synced GitHub
commits) feeds the project timeline.
