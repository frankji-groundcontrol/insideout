# 2026-11-22 — Roadmap canvas as a hierarchy graph, node dependencies, band polish

The owner took every fix the
[roadmap critique](../../design-qa/2026-11-21-roadmap-critique.md) proposed.
They also asked for a new Canvas:

> "Canvas show with a hierarchy, no drop downs. user can click to see the tree
> that node represents. for example, I am the boss, I would like to see global
> and general things being managed by my descendant managers. the work tree can
> be in parallel, and merge somewhere. each flow connected nodes. when boss
> clicked a node, then he/she can see what the manager will see as root."

Owner answers, 2026-11-22:

- "Merge" means **real dependencies**: a node can wait on other nodes, across
  branches.
- The Canvas draws **two levels** below the current root. A deeper subtree
  shows as a "+N" summary. Clicking any node opens it as the root, which is
  what its owner sees.

Three clusters run in parallel from `main` at `59e45b34`, each in its own
worktree. Their files do not overlap, except `roadmap_page.dart` (C1 owns the
header, C3 only the canvas state) and the l10n tables. The coordinator
resolves those by hand, reviews each diff, runs the full gates once on
`feat/roadmap-graph`, and merges one PR with `[skip ci]`.

## Dependency contract (C2 builds it, C3 consumes it)

- **Table:** `insideout.roadmap_dependencies (node_id, depends_on_id,
  created_by, created_at)`. The primary key is the pair. Both ids reference
  `roadmap_nodes` with `ON DELETE CASCADE`, and a check refuses
  `node_id = depends_on_id`. RLS follows the node's project, like
  `roadmap_node_assignments`.
- **Rules:** both nodes are in one project. A link between a node and its own
  ancestor or descendant is refused, because the tree already says that. A
  link that closes a cycle is refused. The writer must write `node_id`, the
  waiting node. Any node of the project the caller can see may be
  `depends_on_id`. The write runs behind the per-project advisory lock and
  notifies roadmap change.
- **`POST /api/v1/roadmap/{nid}/dependencies`** `{"dependsOnId": "<uuid>"}`
  answers 201 with the node view of `{nid}`. An existing link is idempotent.
- **`DELETE /api/v1/roadmap/{nid}/dependencies/{did}`** answers 204, or 404
  when there is no such link.
- **Errors:**
  - 400: a bad id, or `dependency_self`.
  - 403: not a writer of `{nid}`.
  - 404: a node not found or in another project.
  - 409: `dependency_cycle` or `dependency_in_tree`.
- **Agent tokens:** allowed. A root-scoped token needs `{nid}` inside its root.
- **Node view:** every node view gains `"dependsOn": ["<uuid>", ...]`, always
  present and ordered by `created_at`. The agent context lists a node's
  unfinished dependencies as `waitsOn` titles.
- **Surfaces:**
  - Go client: `AddRoadmapDependency`, `RemoveRoadmapDependency`.
  - CLI: `roadmap depend <node> --on <node>`, `roadmap undepend <node> --on <node>`.
  - MCP: `roadmap_dependency_add`, `roadmap_dependency_remove`.

## Checklist

- [ ] C1, band polish (client):
  - Now is the only emphasised band; every slab is neutral.
  - Pressure shows as a chip, and "normal" is hidden.
  - Bands' Later moves finished work to Done and groups the rest by
    top-level branch.
  - Bands cap at 1120 and their cards fill the row; on a phone the label sits
    above the cards.
  - The header scrolls away.
  - Light muted text passes AA on lanes.
  - One presence chip per user; empty bands are hidden.
- [ ] C2, dependencies (server, CLI, MCP, docs): the contract above, with
      real-PostgreSQL tests.
- [ ] C3, Canvas graph (client):
  - Top-down hierarchy: two levels below the root, connector lines, no fold
    toggles.
  - A "+N" summary for deeper subtrees.
  - Clicking a node opens it as the root.
  - Dependency edges are drawn between nodes in view; a "waits on" chip
    covers nodes out of view.
  - A "Depends on…" action in the node menu.
  - Mine uses the same graph.
- [ ] Integrated gates green; design QA and changelog recorded; UI checked live
      on the Spark data.
- [ ] PR merged.
