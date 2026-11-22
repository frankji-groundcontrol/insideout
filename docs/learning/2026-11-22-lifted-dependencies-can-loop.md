# 2026-11-22 — Lifted dependencies can loop, and a layout must be checked on real data

## Lifted links can form a loop

The store refuses any dependency link that would close a cycle between
nodes. Even so, the Canvas's step view can still loop. It lifts each link to
the steps that hold its ends. If one link runs from a node under A to a node
under B, and another runs from a node under B to a node under A, then steps
A and B wait on each other, though no node does.

The Spark seed did exactly this. "Closed beta" (under Launch) waited on the
matching engine (under Matching). "Matching quality dashboard" (under
Matching) waited on "Launch metrics" (under Launch). The audit of the
surfaces caught it before the boss view shipped with a loop.

**Practice:**

- Any view that lifts links must group strongly connected steps rather than
  assume the graph has no cycles. The client's `roadmap_step_flow.dart`
  does this, as must any server version.
- Demo data should avoid such loops unless a loop is the point of the demo.

## Check a layout on the real data, not a fixture

The two-level hierarchy passed its widget tests. On the 33-node Spark tree it
ran about 5000px wide, and its root opened off screen. Links that skipped a
column looked right in a three-node fixture but ran under boxes on real data.

**Practice:** before calling a layout done, build it and look at it on the
largest demo data at a desktop and a phone width. Then turn what you saw
into a test. The lane-routing test came from this check.

Related: [changelog](../changelogs/2026-11-22-roadmap-graph-and-dependencies.md),
[design QA](../design-qa/2026-11-22-roadmap-canvas-graph.md).
