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
(`server/internal/store/roadmap.go:80`). Evidence rows hang off a node in
`insideout.roadmap_evidence` (`20260821130000_roadmap_evidence.sql`).

### The owner column

`owner_user_id` is the one human responsible for a node, and owning a node is
write authority over it and everything below it (next section). It is set
through `ownerId` on `PATCH /api/v1/roadmap/{nid}`
(`server/internal/api/roadmap.go`); the store refuses an owner who is not a
member of the node's workspace, and an agent token is refused on any `ownerId`
(owner decision 4). Moving an ownerless leaf to `in_progress` makes the acting
human its owner, and a build seeds its first in-progress leaf owned by the
builder, so work in progress has one owner (`claimStartedLeaf`,
`server/internal/store/roadmap_completion.go`). A member who leaves or is
removed from a workspace stops owning its nodes in the same transaction
(`RemoveMember`, `server/internal/store/memberships.go`). No history is
recorded yet (no "delegated by" or "when"), and the node's children stay
ownerless. Only the web sets an owner, through the owner picker in the Edit
form (`client/lib/features/project/roadmap_node_menu.dart`); the Go client's
`RoadmapUpdate` carries no owner (`server/internal/apiclient/roadmap.go`), so
neither the CLI nor MCP can delegate.

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
| Change a node's owner | `requireNodeManager`: a plan writer, or a writer of the node's parent |
| Create a node | a root: `requirePlanWriter`; a child: `requireNodeWriter` on the parent |
| Move a node | `requireNodeManager` on the node, and `requireNodeWriter` on the new parent (a new root: `requirePlanWriter`) |
| Delete a node and its subtree | `requireNodeManager` |
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
a row into another project. `roadmap_nodes_select` stays workspace-wide, so
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
| `GET /api/v1/roadmap/{nid}` | One node: `{node, ancestors: [{id, title}] root first, projectId, projectName}`; 404 when not visible | `server/internal/api/roadmap_read.go` |
| `POST /api/v1/projects/{pid}/roadmap` | Create a node | `server/internal/api/roadmap.go` |
| `PATCH /api/v1/roadmap/{nid}` | Edit fields, status and owner | `server/internal/api/roadmap.go` |
| `POST /api/v1/roadmap/{nid}/move` | Re-parent or reorder | `server/internal/api/roadmap_move.go` |
| `DELETE /api/v1/roadmap/{nid}` | Delete with subtree | `server/internal/api/roadmap.go` |
| `GET /api/v1/projects/{pid}/roadmap/progress` and `/stream` | Progress view and its live ping | `server/internal/api/roadmap_progress.go` |
| `GET`, `POST /api/v1/roadmap/{nid}/evidence` | Read evidence, add an observation | `server/internal/api/roadmap_validation.go` |
| `POST /api/v1/roadmap/{nid}/validate` | Human validation | `server/internal/api/roadmap_validation.go` |
| `POST /api/v1/prds/{pid}/build` and `/confirm` | Draft (read-only, approved PRD only) and replace the tree; confirm creates the project and refuses to erase evidence or done or validated nodes without `force` | `server/internal/api/roadmap_ai.go` |
| `POST /api/v1/roadmap/{nid}/expand` and `/confirm` | Break a node down | `server/internal/api/roadmap_ai.go` |
| `GET /api/v1/agent/context` | Agent briefing | `server/internal/api/agent_routes.go` |

Every node view carries `canWrite`, the caller's right under
`_can_write_node`, computed inside the one list query (a write's own response
says `true`). There is no read of the nodes a person owns yet. An agent token
may read one node and the subtree list, and may edit through `PATCH`, but is
refused on `ownerId` (`server/internal/api/agent_scope.go`); delete is
human-only. The only change signal is an in-memory callback
(`server/internal/store/pool.go`) that pings open progress streams. The web
has one roadmap route per project (`client/lib/router.dart`). MCP exposes
`roadmap_list`, `roadmap_add`, `roadmap_update`, `roadmap_move`,
`roadmap_delete` and `roadmap_evidence`
(`server/internal/mcpserver/product_tools.go`); none of them takes a root yet.

### Agent context focus

`GET /api/v1/agent/context?focus={nid}` answers 404 for a focus that is not a
node of the project. In implementation mode the `focus` block carries the
node's description and `path` (ancestor titles, root first), and `leaves`
lists only the leaves of the focus subtree (the focus itself when it is a
leaf) (`server/internal/agentcontext/agentcontext.go`).

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
appends the assignment row and a project update. Handing back is the same call
with the previous delegator as the target. An agent token is refused on it and
on an owner change through `PATCH`.

**Clients.** The web gains `/projects/:id/roadmap/:nodeId` with a breadcrumb;
tapping a card opens it; the node menu gains "Open as tree" and "Delegate
to"; write items hide when `canWrite` is false; a "Mine" toggle and the
signed-in landing list the caller's nodes. MCP and the CLI ship the same
slice at the same time: a root on `roadmap_list`, `roadmap_mine`,
`roadmap_delegate` and a `members` lookup.

**Agents.** `agent-context --focus` returns only the focus subtree and answers
404 for an unknown node. Later, an agent token may carry a root node and is
refused outside it.

## Status

Built (wave 2 of the review 5 remediation): the one write rule in SQL and its
policies, `canWrite` on node views, the subtree and single-node reads, the
focus-scoped agent context, the agent-token refusal on owner changes, owner
clearing on member removal, and the owner of started work. Still to land:
the assignment history, delegate and hand-back routes, node rows in the
project feed, the per-person `mine` read, roll-ups, agent tokens carrying a
root node, and the web, MCP and CLI surfaces
([execution plan](../plans/2026-11-21-rv5-remediation/2026-11-21-rv5-remediation.md),
[row detail](../review/2026-11-21-focused-review-5/40-remediation-plan.md)).
Update this page as each slice lands.
