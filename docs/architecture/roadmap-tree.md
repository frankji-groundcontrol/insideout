# Roadmap tree

The roadmap is one tree of nodes per project. This page describes how the tree
is stored, who may write it, and how it is read today, then the target that
gives each person their own work tree. The product rule is in
[PRODUCT.md](../../PRODUCT.md) under "Personal work trees and delegation". The
design source is
[41-task-tree-target-design.md](../review/2026-11-21-focused-review-5/41-task-tree-target-design.md);
the owner's decisions are recorded in the
[review 5 open findings](../issues/2026-11-21-review-5-open-findings.md).

## Current state

### Data

`insideout.roadmap_nodes` is created in
`server/db/migrations/20260722120000_roadmap_nodes.sql:10-22` and widened by
later migrations:

| Column | Meaning | Added in |
| --- | --- | --- |
| `id`, `project_id` | Node and its project (`ON DELETE CASCADE`) | `20260722120000_roadmap_nodes.sql:11-12` |
| `parent_id` | Self reference; `NULL` marks a root; deleting a node cascades to its subtree | `20260722120000_roadmap_nodes.sql:13` |
| `title`, `description`, `position`, `meta` | Content and sibling order | `20260722120000_roadmap_nodes.sql:14-19` |
| `status` | `locked`, `pending`, `in_progress`, `done`, `validated` | `20261011140000_roadmap_validation.sql:17-19` |
| `created_by`, `updated_by` | Attribution, `ON DELETE SET NULL` | `20260726100000_roadmap_attribution.sql:9-11` |
| `deadline` | Explicit deadline | `20260822120000_roadmap_deadlines.sql:4` |
| `acceptance` | The sentence that says what done means | `20261005120000_roadmap_acceptance_owner.sql:7` |
| `owner_user_id` | The one human owner, nullable, `ON DELETE SET NULL` | `20261005120000_roadmap_acceptance_owner.sql:8` |
| `validation_reason`, `validated_by`, `validated_at` | The human validation milestone | `20261011140000_roadmap_validation.sql:24-29` |
| `completion_reason` | The override sentence for done without evidence | `20261020150000_roadmap_completion_reason.sql:6-7` |

The store selects every column except `meta`
(`server/internal/store/roadmap.go`). Evidence rows hang off a node in
`insideout.roadmap_evidence` (`20260821130000_roadmap_evidence.sql`). Every
change of `owner_user_id` appends a row to the append-only
`insideout.roadmap_node_assignments` (node, from, to, assigned by, kind
`delegate`, `handback` or `cleared`, note, time), and `project_updates` carries
a nullable `node_id` with the kinds `node_delegated`, `node_handed_back` and
`node_done` (`20261121110000_roadmap_node_assignments.sql`).

### The owner column

`owner_user_id` is the one human responsible for a node, and owning a node is
write authority over it and everything below it (next section). It is set
through `ownerId` on `PATCH /api/v1/roadmap/{nid}`
(`server/internal/api/roadmap.go`); the store refuses an owner who is not a
member of the node's workspace, and an agent token is refused on any `ownerId`
(owner decision 4). `POST /api/v1/roadmap/{nid}/delegate {toUserId, note}` sets
the owner too, and `POST /api/v1/roadmap/{nid}/handback {note}` lets the current
owner return the node to whoever made its latest `delegate` row (nobody when
that is unknown, themself, or no longer a member); both are human-only
(`server/internal/api/roadmap_delegation.go`,
`server/internal/store/roadmap_assignments.go`). Every owner change, through
either route or `PATCH`, appends an assignment row and a `node_delegated` or
`node_handed_back` feed row naming the node; a `PATCH` that re-sends the shown
owner is no change. The feed text names the node, not the people: who-to-whom is
read from `GET /roadmap/{nid}/assignments`, newest first with names resolved at
read time. A node reaching done writes a `node_done` feed row. Moving an
ownerless leaf to `in_progress` makes the acting human its owner (the feed row
keeps the kind `node_delegated` but reads "Took on", since the actor assigned
it to themself), and a build
seeds its first in-progress leaf owned by the builder (one `delegate`
assignment row by the builder, no feed row), so work in progress has
one owner (`claimStartedLeaf`, `server/internal/store/roadmap_completion.go`).
A member who leaves or is removed from a workspace stops owning its nodes in the
same transaction, each recorded as a `cleared` row and a `node_handed_back`
feed row by the remover (`RemoveMember`, `server/internal/store/memberships.go`);
an account erasure records `cleared` rows too (`server/internal/store/erasure.go`).
A node's children stay ownerless; ownership is inherited as authority, not
copied. The web hands a node on through the node menu's "Delegate to…" and
"Hand back" items, and the Edit form's owner picker shows only to a caller who
may change that node's owner
(`client/lib/features/project/roadmap_node_delegate.dart`,
`roadmap_node_edit.dart`); the CLI delegates with `roadmap delegate`/`handback` and `update --owner`; the
MCP has no delegation tool, since every MCP tool acts through an agent-scope
token (see the MCP paragraph below).

### One write rule

The tree has one write rule, in SQL:
`insideout._can_write_node(node, user)`
(`server/db/migrations/20261121100000_roadmap_node_writer.sql`) is true for the
project owner, a workspace admin of the node's workspace
(`insideout._is_plan_writer`), or a current workspace member who owns the node
or any ancestor of it. The store calls it through `requireNodeWriter` and the
row-level security policies call it directly, so the two cannot drift. The
store's predicates live in `server/internal/store/authz.go`:

| Write | Rule |
| --- | --- |
| Edit a node (title, description, status, deadline, acceptance, completion reason) | `requireNodeWriter` on the node |
| Change a node's owner (`PATCH ownerId`, delegate) | `requireNodeManager`: a plan writer, or a writer of the node's parent |
| Hand a node back | its current owner only |
| Create a node | a root: `requirePlanWriter`; a child: `requireNodeWriter` on the parent |
| Move a node | `requireNodeManager` on the node, and `requireNodeWriter` on the new parent (a new root: `requirePlanWriter`) |
| Delete a node and its subtree | `requireNodeManager`; a subtree with evidence or done or validated nodes only with `?force=true`, which leaves a note in the feed |
| Break a node down (expand and its confirm) | `requireNodeWriter` on the node; the draft route refuses before the planner runs |
| Add a validation observation | `requireNodeWriter` on the node |
| Validate a node | `requirePlanWriter` |
| Replace the whole tree (Build) | `requirePlanWriter` |
| File an agent proposal about no node | `requirePlanWriter` (project-level writes) |
| File an agent proposal about a node (`nodeId`) | `requireNodeWriter` on that node, which must be in the project (owner decision 7); a root-scoped agent token must name a node inside its root |
| Decide a proposal, apply proposal items | `requirePlanWriter` |

`requireNodeManager` is what keeps a delegate inside the subtree they were
handed: they may edit their root R and everything under it, add and break
down anywhere in it, and move, delete or re-delegate nodes strictly below R,
but never move, delete or reassign R itself, validate, or replace the tree.
Plan writers keep every right. Pinned on PostgreSQL by
`server/internal/store/roadmap_delegate_test.go`.

### Row-level security

`roadmap_nodes_update` and `roadmap_nodes_delete` call
`_can_write_node(id, current_user_id())`; `roadmap_nodes_insert` pins
`created_by` to the caller and calls it on the parent, with
`_is_plan_writer` for a root; `evidence_owner_insert` on `roadmap_evidence`
calls it on the node. A child's `project_id` must equal its parent's
(`_node_project`), on insert and on update, so a writable parent never links
a row into another project, and an update may not change `project_id`. `roadmap_nodes_select` stays workspace-wide, so
every member reads the whole project (owner decision 2). The functions are
`SECURITY DEFINER`, owned by `insideout_owner`, so a policy on
`roadmap_nodes` never queries the table it guards (BUG-007).

What the database holds is narrower than the store's rule: a row can be
updated or deleted only when its current position is writable, and an insert
needs a writable parent in the same project. The policies judge the tree as
it stood when the statement started, so they do not check where an updated
row lands inside the project (a raw reparent under a sibling branch), the
rules for R itself (move, delete, reassign), or validation. Those are
enforced only in the store (`requireNodeManager`, `requirePlanWriter`).

### Routes

| Route | Effect | Code |
| --- | --- | --- |
| `GET /api/v1/projects/{pid}/roadmap` | Whole tree as a flat list, any member; `?root={nid}` restricts it to that node and its descendants, root first (404 for a node outside the project); with no `?root`, a root-scoped agent token reads its root's subtree when the root is a node of the project | `server/internal/api/roadmap_read.go`, `server/internal/store/roadmap_read.go` |
| `GET /api/v1/projects/{pid}/roadmap/steps` | One level in dependency order (see Steps below); `?root={nid}` opens that node | `server/internal/api/roadmap_steps.go`, `server/internal/roadmaptime/steps.go` |
| `GET /api/v1/roadmap/{nid}` | One node: `{node, ancestors: [{id, title, ownerId}] root first, projectId, projectName}` (`ownerId` is a user id or null, so a client sees that the caller owns the opened tree through an ancestor; `mine` ancestors carry it too); 404 when not visible | `server/internal/api/roadmap_read.go` |
| `POST /api/v1/projects/{pid}/roadmap` | Create a node | `server/internal/api/roadmap.go` |
| `PATCH /api/v1/roadmap/{nid}` | Edit fields, status and owner | `server/internal/api/roadmap.go` |
| `POST /api/v1/roadmap/{nid}/move` | Re-parent or reorder | `server/internal/api/roadmap_move.go` |
| `DELETE /api/v1/roadmap/{nid}` | Delete with subtree; 409 `roadmap_delete_has_history` with the counts unless `?force=true` (the CLI: `roadmap delete --force`) | `server/internal/api/roadmap.go`, `server/internal/store/roadmap_delete.go` |
| `POST /api/v1/roadmap/{nid}/delegate`, `/handback` | Hand a node to a member (400 for a non-member), hand it back | `server/internal/api/roadmap_delegation.go` |
| `GET /api/v1/roadmap/{nid}/assignments` | The node's owner history, newest first | `server/internal/api/roadmap_delegation.go` |
| `POST /api/v1/roadmap/{nid}/dependencies`, `DELETE .../dependencies/{did}` | Make a node wait on another, or stop it (see Dependencies below) | `server/internal/api/roadmap_dependencies.go` |
| `GET /api/v1/workspaces/{id}/roadmap/mine` | The caller's topmost owned nodes in the workspace, each `{node, projectId, projectName, ancestors}`, by project then position | `server/internal/api/roadmap_read.go`, `server/internal/store/roadmap_mine.go` |
| `GET /api/v1/projects/{pid}/roadmap/progress` and `/stream` | Progress view and its live ping; `?root={nid}` restricts it to one subtree, and a root-scoped agent token defaults to its root as the list does (`requestRoot`, `server/internal/api/roadmap_steps.go`) | `server/internal/api/roadmap_progress.go` |
| `GET`, `POST /api/v1/roadmap/{nid}/evidence` | Read evidence, add an observation | `server/internal/api/roadmap_validation.go` |
| `POST /api/v1/roadmap/{nid}/validate` | Human validation | `server/internal/api/roadmap_validation.go` |
| `POST /api/v1/prds/{pid}/build` and `/confirm` | Draft (read-only, approved PRD only) and replace the tree; confirm creates the project and refuses to erase evidence or done or validated nodes without `force` | `server/internal/api/roadmap_ai.go` |
| `POST /api/v1/roadmap/{nid}/expand` and `/confirm` | Break a node down | `server/internal/api/roadmap_ai.go` |
| `GET /api/v1/agent/context` | Agent briefing; with no focus it opens on what the caller is in charge of (see Agent context focus below) | `server/internal/api/agent_routes.go`, `server/internal/api/agent_context_charge.go` |

Every node view carries `dependsOn` (see Dependencies below), `canWrite`, the
caller's right under `_can_write_node`, and `rollup {leaves, done, validated, earliestDeadline}`
over the leaves at or below it (a leaf counts itself; `done` counts done and
validated, the Progress rule; `earliestDeadline` is the earliest deadline among
the unfinished leaves). A list computes both in its one query plus Go
(`roadmaptime.Rollups`); a single read, `mine` and every write's response
re-read the node with one more query for the nodes below
(`server/internal/store/roadmap_views.go`), so a write that leaves the caller
without authority (handing back their own node) answers `canWrite: false`.
Project views carry `canWritePlan`, the caller's plan-writer right. An agent
token may read one node, the subtree list, `mine` and the owner history, and
may edit through `PATCH`, but is refused on `ownerId`, delegate and hand-back
(`server/internal/api/agent_scope.go`); delete is human-only. An agent token
minted with a `rootNodeId` writes only inside that subtree
([agent token scope](./agent-token-scope.md)). The only change signal is an in-memory callback
(`server/internal/store/pool.go`) that pings open progress streams. The web
has one roadmap route per project (`client/lib/router.dart`). MCP exposes
`roadmap_add`, `roadmap_move` and `roadmap_evidence` (no `roadmap_delete`: delete is human-only)
(`server/internal/mcpserver/tools_roadmap.go`), and the tree tools
`roadmap_list` and `progress` (both take `root`), `roadmap_node`,
`roadmap_mine`, `roadmap_update` (with `acceptance`; no owner or completion reason, which an agent token is refused) and
`roadmap_assignments` (`server/internal/mcpserver/tools_roadmap_tree.go`); it
has no delegate, hand-back or member-list tool, since every MCP tool acts
through an agent-scope token and those routes are human-only. The CLI has the
tree verbs and those three (`roadmap show`, `mine`, `delegate`,
`handback`, `assignments`, `--root`, `update --owner/--acceptance`, and
`members`; `server/cmd/insideout/cli_roadmap_tree.go`) over the Go client
(`server/internal/apiclient/roadmap.go`, `roadmap_tree.go`).

### Dependencies

A node may wait on other nodes of its project, across branches, so parallel
work can merge. `insideout.roadmap_dependencies (node_id, depends_on_id,
created_by, created_at)` holds one row per link: `node_id` waits on
`depends_on_id`. The pair is the primary key, both ids reference
`roadmap_nodes` with `ON DELETE CASCADE` (deleting either node, or replacing
the whole tree, drops the link), and a check refuses a node waiting on itself
(`server/db/migrations/20261122100000_roadmap_dependencies.sql`). RLS follows
the node's project: every member of its workspace reads the links; an insert
needs `created_by` to be the caller, `_can_write_node` on `node_id`, and both
ends in one project; a delete needs `_can_write_node` on `node_id`.

The store (`server/internal/store/roadmap_dependencies.go`) adds and removes a
link behind the per-project advisory lock and pings the progress streams. It
refuses a link between a node and its own ancestor or descendant (the tree
already says that, `ErrDependencyInTree`) and one that closes a cycle of
waiting (`ErrDependencyCycle`); re-adding an existing link is no change. The
writer of the waiting node decides; any node of the project the caller can see
may be waited on.

| Route | Answer |
| --- | --- |
| `POST /api/v1/roadmap/{nid}/dependencies {"dependsOnId"}` | 201 with the node view of `{nid}` |
| `DELETE /api/v1/roadmap/{nid}/dependencies/{did}` | 204, or 404 `dependency not found` when there is no such link |

Errors: 400 for a bad id or `dependency_self`; 403 for a caller who does not
write `{nid}`; 404 for a node not visible or in another project; 409
`dependency_in_tree` or `dependency_cycle`
(`server/internal/api/roadmap_dependencies.go`). Agent tokens may call both
routes; a root-scoped token needs `{nid}` inside its root (the waited-on node
may lie anywhere in the project). Every node view carries `dependsOn`, the
waited-on ids, oldest link first, `[]` when none. The agent context lists a
node's unfinished dependencies (not done or validated) as `waitsOn` titles on
each leaf, and on the focus what its whole subtree waits on (see Agent
context focus below). The Go client has `AddRoadmapDependency` and
`RemoveRoadmapDependency`, the CLI `roadmap depend --on <node> <node>` and
`roadmap undepend --on <node> <node>`, MCP `roadmap_dependency_add` and
`roadmap_dependency_remove`.

Known limit: a move is not checked against existing links, so moving a node
under the node it waits on (or the reverse) keeps a link the add would now
refuse as `dependency_in_tree`.

### Agent context focus

`GET /api/v1/agent/context?focus={nid}` answers 404 for a focus that is not a
node of the project. In implementation mode the `focus` block carries the
node's description and `path` (ancestor titles, root first), and `leaves`
lists only the leaves of the focus subtree (the focus itself when it is a
leaf) (`server/internal/agentcontext/agentcontext.go`). With no `focus`, a
root-scoped agent token is briefed on its root.

Otherwise, with no `focus`, implementation mode opens on what the caller's
human is in charge of in the project (owner decision, 2026-11-22: the nodes
they own, and an unowned root when they are a plan writer, that is the
project owner or a workspace admin; topmost only). One such node becomes the
focus. Several are named in `inCharge: [{id, title}]` with no focus, and
`leaves` keeps to the leaves below them. None leaves the whole-project
briefing as before (`server/internal/api/agent_context_charge.go`). Human
and agent tokens behave alike.

The `focus` block also carries, as titles:

- `waitsOn`: the unfinished nodes outside the focus that unfinished work
  inside it waits on (for a leaf focus, its own unfinished dependencies).
- `blocks`: the unfinished nodes outside the focus that wait on unfinished
  work inside it.
- `steps`: `[{id, title, stage}]`, the focus's steps in order (see Steps
  below), when the focus has children.

These live in `server/internal/agentcontext/across.go`, and each list is
named in the `untrusted` block when present.

### Steps

The Canvas shows one level: the steps under the opened node, ordered by what
they wait on. `GET /api/v1/projects/{pid}/roadmap/steps?root={nid}` gives the
same answer from the server, so the web, CLI, MCP and agents read one order.
It answers 200 with:

```json
{"root": <node view or null>,
 "ancestors": [{"id", "title", "ownerId"}],
 "steps": [{"node": <node view>, "stage": 0, "loop": null,
            "after": ["<step id>"], "links": {"<step id>": 1},
            "itemsInside": 3, "waitsOutside": [{"id", "title"}]}]}
```

- **Root.** `?root` opens that node; a leaf is its own one step. With no
  `?root`, a root-scoped agent token opens its root (when it is a node of the
  project), else a project with exactly one root opens that root, else
  `root` is null and the project's roots are the steps. `ancestors` is the
  opened node's path, root first, as `GET /roadmap/{nid}` gives it.
- **Lifting.** Every dependency in the project is lifted to the steps that
  hold its two ends. A link inside one step drops, and so does a link up or
  down one branch. A link from a step's work to a node under no shown step
  becomes `waitsOutside` on that step.
- **Order.** Steps that reach each other through their links form a loop
  group and share `loop` (numbered from 0 in order; null outside a loop) and
  a stage. `stage` is the longest path to the step's group over the groups.
  Steps come by stage, then group (by its first position), then position.
  `after` and `links` name the steps of other groups a step waits on, with
  their link counts, in step order. `itemsInside` counts the nodes below the
  step.

The order is the pure `roadmaptime.Steps` over the project's `ListRoadmap`
(`server/internal/roadmaptime/steps.go`, the same rules as
`client/lib/features/project/roadmap_step_flow.dart`); the handler is
`server/internal/api/roadmap_steps.go`. Errors: 400 for a malformed root; 404
for a project the caller cannot see or a root not in the project; 403
`agent_scope_denied` for a root-scoped agent token asking for a root outside
its subtree. The Go client has `RoadmapSteps`, the CLI `roadmap steps [--root
node-id] <project-id>`, MCP `roadmap_steps(project_id, root?)`.

## Target

Decided by the owner on 2026-11-21:

1. A person's tree is derived from the nodes they own; roles govern what a
   person may do, not which tree they see.
2. A personal tree is a default view, not a wall. Reads stay workspace-wide,
   which is how a manager opens any node.
3. A delegate may re-delegate descendants, not the node they were handed.
4. An agent token cannot change a node's owner; it files a proposal.

**Data.** `owner_user_id` stays the current responsible person. A new
append-only `insideout.roadmap_node_assignments` records node, from user, to
user, assigned by, note and time. `project_updates` gains a nullable
`node_id` and the kinds `node_delegated`, `node_handed_back` and `node_done`,
so the delegator sees delegation, hand-back and completion in the feed.

**One write rule in two places.** A SQL function
`insideout._can_write_node(node, user)` is true for the project owner, a
workspace admin, or a current workspace member who owns the node or any
ancestor of it. A store helper `requireNodeWriter` calls it, and so do the
insert, update and delete policies on `roadmap_nodes`, so the Go rule and the
database cannot drift. Inside their subtree a delegate may edit, add, break
down, move, attach evidence and re-delegate descendants. Moving, deleting or
reassigning the delegated root itself, validating, and replacing the whole
tree stay with the project owner and admins.

**Reads.** `GET /projects/{pid}/roadmap?root={nid}` returns one subtree with
an `ancestors` array for the breadcrumb; `GET /roadmap/{nid}` reads one node;
`GET /workspaces/{id}/roadmap/mine` returns the caller's topmost owned nodes,
each a root. Node views gain `rollup` (leaves done, validated, total,
earliest deadline) and `canWrite`.

**Writes.** `POST /roadmap/{nid}/delegate {toUserId, note}` sets the owner and
appends the assignment row and a project update. `POST /roadmap/{nid}/handback`
returns the node to its last delegator. An agent token is refused on both and
on an owner change through `PATCH`.

**Clients.** The web gains `/projects/:id/roadmap/:nodeId` with a breadcrumb;
tapping a card opens it; the node menu gains "Open as tree" and "Delegate
to"; write items hide when `canWrite` is false; a "Mine" toggle and the
signed-in landing list the caller's nodes. MCP and the CLI ship the same
slice at the same time: a root on `roadmap_list` and `roadmap_mine`; delegate
and a `members` lookup are CLI-only, since every MCP tool acts through an
agent-scope token that those routes refuse.

**Agents.** `agent-context --focus` returns only the focus subtree and answers
404 for an unknown node. An agent token may carry a root node and is refused
outside it.

## Status

Built (wave 2 of the review 5 remediation): the one write rule in SQL and its
policies, `canWrite` on node views, the subtree and single-node reads, the
focus-scoped agent context, the agent-token refusal on owner changes, owner
clearing on member removal, the owner of started work, the assignment
history, delegate and hand-back routes, node rows in the project feed, the
per-person `mine` read, roll-ups and subtree Progress, agent tokens carrying a
root node, and the history guard on a subtree delete; on the Go client, the
CLI and MCP, owner and acceptance edits, the subtree root, the node and `mine`
reads, delegation, its history and the member list. On the web: the
`/projects/:id/roadmap/:nodeId` route with its breadcrumb, a card tap that
opens a branch as its tree (a leaf opens its read-only detail), the "Open as
tree", "Details", "Delegate to…" and "Hand back" menu items, and write items
shown only where `canWrite` (and, for move, delete, delegate and validate, the
plan-writer or parent rule) says they succeed
(`client/lib/features/project/roadmap_node_menu.dart`). The roadmap page's
"Mine" view lays out, as canvas bands, the caller's topmost owned nodes and
everything below them (`roadmapMine` in
`client/lib/features/project/roadmap_tree.dart`, from the node views'
`ownerId` and the signed-in user); the signed-in landing's "My work" lists
`mine` for each workspace, each item opening its tree
(`client/lib/features/dashboard/my_work_section.dart`). In an opened tree,
Mine shows the whole tree when the caller owns the opened root or any
ancestor above it (the ancestors' `ownerId`). A card and the node detail with
no owner of their own name the owner they inherit, "Owner: dana (via
Payments)", from the nearest owned node in the loaded list or, in an opened
tree, the ancestors (`roadmapInheritedOwner`). The canvas folds a
branch behind a chevron that shows how many nodes it hides, remembered for
the page session; every card's bar and "n of m done" come from the server
`rollup`, so validated counts as finished and every leaf below counts, not
only direct children. A delete over a branch with history shows the 409's
counts in the build flow's words and resends with `force=true` only when the
person confirms; deleting the opened root opens its parent's tree, or the
project's whole tree for a top-level node. Known limitation:
in an opened tree the root's parent is not loaded, so a delegate who holds a
branch above it (but is not a plan writer) sees no Move, Delete or Delegate
on the opened root's card; they do it from the parent's tree. The slice
landed in wave 2 of the review 5 remediation
([execution plan](../plans/2026-11-21-rv5-remediation/2026-11-21-rv5-remediation.md),
[row detail](../review/2026-11-21-focused-review-5/40-remediation-plan.md)).
Update this page as each slice lands.

Surfaces (wave 3): the CLI carries the human-only node verbs too, `evidence
add` and `roadmap validate`, and the project verbs around the tree (`project`,
`project update`, `project delete`, `updates`, `update add`). MCP registers
no tool on a route the server keeps for a human: account deletion,
validation, Commit, the proposal decision, every delete (`roadmap_delete` is
gone), delegation and member roles. An agent reads and edits a project
(`project`, `project_update`, which refuses an owner change) and posts
updates (`update_add`).
