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

Later owner changes, 2026-11-22, which supersede the two-level answer:

- > "In canvas I want 1. drag to move center view 2. zoom in and out"
- > "I do not want to see hierarchy on canvas, only things in order. hierarchy
  > is represented when I click into a certain step, namely node"

  The Canvas shows **one level**: the steps under the current node, ordered
  left to right by their dependencies. A link between any descendants of two
  steps counts as a link between the steps. Opening a step shows its own
  steps.
- > "check if cli mcp api also follow this product logic. I want each person
  > cares about what they need to care at the first glance. namely
  > people-in-charge mode"

  An audit of the API, CLI, MCP and agent context against these rules runs
  before any surface change.

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

- [x] C1, band polish (client):
  - Now is the only emphasised band; every slab is neutral.
  - Pressure shows as a chip, and "normal" is hidden.
  - Bands' Later moves finished work to Done and groups the rest by
    top-level branch.
  - Bands cap at 1120 and their cards fill the row; on a phone the label sits
    above the cards.
  - The header scrolls away.
  - Light muted text passes AA on lanes.
  - One presence chip per user; empty bands are hidden.
- [x] C2, dependencies (server, CLI, MCP, docs): the contract above, with
      real-PostgreSQL tests.
- [x] C3, first pass: a two-level hierarchy graph, then superseded by the
      owner's later changes.
- [x] C3, Canvas as an ordered flow (client), with lane routing for links
      that skip a column:
  - One level of steps, ordered by lifted dependencies, with merge arrows.
  - Opening a step shows its own steps; a leaf opens its detail.
  - Drag to pan, wheel or pinch to zoom, and +/−/fit buttons.
  - A "Depends on…" picker in the node menu.
  - Mine shows the caller's topmost owned nodes as the steps.
- [x] People-in-charge audit of the API, CLI, MCP and agent context. The owner
      took all four proposals. "In charge" of an unowned root means the
      project owner **and** the workspace admins.
- [ ] S1, ordered steps and agents:
  - `GET /projects/{pid}/roadmap/steps?root=` with the Go client, CLI
    `roadmap steps` and MCP `roadmap_steps`.
  - The agent context defaults to what its human is in charge of, with waits
    and blocks across steps.
  - Root-scoped tokens default list and progress to their root.
  - The MCP instructions say where to start.
- [ ] S2, first glance and order:
  - `mine` counts the in-charge root, and each root carries its Now items,
    what it waits on and what it blocks.
  - `mine` works across all of a person's workspaces.
  - Progress Next puts ready items first.
  - The feed gains `?node=` and `?pending=`.
- [ ] C5, client:
  - The Canvas reads the steps endpoint.
  - My work shows Now, waits on and blocking.
- [x] The seed's Matching → Launch link, which made a loop, is replaced by
      one inside Matching.
- [x] The Spark demo seed adds eight dependencies and checks three refusals.
- [ ] Integrated gates green; design QA and changelog recorded; UI checked live
      on the Spark data.
- [ ] PR merged.
