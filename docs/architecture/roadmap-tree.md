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
| File an agent proposal, apply proposal items | `requirePlanWriter` (project-level writes) |

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
| `GET /api/v1/projects/{pid}/roadmap` | Whole tree as a flat list, any member; `?root={nid}` restricts it to that node and its descendants, root first (404 for a node outside the project) | `server/internal/api/roadmap_read.go`, `server/internal/store/roadmap_read.go` |
| `GET /api/v1/roadmap/{nid}` | One node: `{node, ancestors: [{id, title, ownerId}] root first, projectId, projectName}` (`ownerId` is a user id or null, so a client sees that the caller owns the opened tree through an ancestor; `mine` ancestors carry it too); 404 when not visible | `server/internal/api/roadmap_read.go` |
| `POST /api/v1/projects/{pid}/roadmap` | Create a node | `server/internal/api/roadmap.go` |
| `PATCH /api/v1/roadmap/{nid}` | Edit fields, status and owner | `server/internal/api/roadmap.go` |
| `POST /api/v1/roadmap/{nid}/move` | Re-parent or reorder | `server/internal/api/roadmap_move.go` |
| `DELETE /api/v1/roadmap/{nid}` | Delete with subtree; 409 `roadmap_delete_has_history` with the counts unless `?force=true` (the CLI: `roadmap delete --force`) | `server/internal/api/roadmap.go`, `server/internal/store/roadmap_delete.go` |
| `POST /api/v1/roadmap/{nid}/delegate`, `/handback` | Hand a node to a member (400 for a non-member), hand it back | `server/internal/api/roadmap_delegation.go` |
| `GET /api/v1/roadmap/{nid}/assignments` | The node's owner history, newest first | `server/internal/api/roadmap_delegation.go` |
| `GET /api/v1/workspaces/{id}/roadmap/mine` | The caller's topmost owned nodes in the workspace, each `{node, projectId, projectName, ancestors}`, by project then position | `server/internal/api/roadmap_read.go`, `server/internal/store/roadmap_mine.go` |
| `GET /api/v1/projects/{pid}/roadmap/progress` and `/stream` | Progress view and its live ping; `?root={nid}` restricts it to one subtree | `server/internal/api/roadmap_progress.go` |
| `GET`, `POST /api/v1/roadmap/{nid}/evidence` | Read evidence, add an observation | `server/internal/api/roadmap_validation.go` |
| `POST /api/v1/roadmap/{nid}/validate` | Human validation | `server/internal/api/roadmap_validation.go` |
| `POST /api/v1/prds/{pid}/build` and `/confirm` | Draft (read-only, approved PRD only) and replace the tree; confirm creates the project and refuses to erase evidence or done or validated nodes without `force` | `server/internal/api/roadmap_ai.go` |
| `POST /api/v1/roadmap/{nid}/expand` and `/confirm` | Break a node down | `server/internal/api/roadmap_ai.go` |
| `GET /api/v1/agent/context` | Agent briefing | `server/internal/api/agent_routes.go` |

Every node view carries `canWrite`, the caller's right under
`_can_write_node`, and `rollup {leaves, done, validated, earliestDeadline}`
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
([agent token scope](agent-token-scope.md)). The only change signal is an in-memory callback
(`server/internal/store/pool.go`) that pings open progress streams. The web
has one roadmap route per project (`client/lib/router.dart`). MCP exposes
`roadmap_add`, `roadmap_move`, `roadmap_delete` and `roadmap_evidence`
(`server/internal/mcpserver/product_tools.go`), and the tree tools
`roadmap_list` and `progress` (both take `root`), `roadmap_node`,
`roadmap_mine`, `roadmap_update` (with `acceptance` and `owner_id`) and
`roadmap_assignments` (`server/internal/mcpserver/roadmap_tree_tools.go`); it
has no delegate, hand-back or member-list tool, since every MCP tool acts
through an agent-scope token and those routes are human-only. The CLI has the
tree verbs and those three (`roadmap show`, `mine`, `delegate`,
`handback`, `assignments`, `--root`, `update --owner/--acceptance`, and
`members`; `server/cmd/insideout/cli_roadmap_tree.go`) over the Go client
(`server/internal/apiclient/roadmap.go`, `roadmap_tree.go`).

### Agent context focus

`GET /api/v1/agent/context?focus={nid}` answers 404 for a focus that is not a
node of the project. In implementation mode the `focus` block carries the
node's description and `path` (ancestor titles, root first), and `leaves`
lists only the leaves of the focus subtree (the focus itself when it is a
leaf) (`server/internal/agentcontext/agentcontext.go`). With no `focus`, a
root-scoped agent token is briefed on its root.

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
