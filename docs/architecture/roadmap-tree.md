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

`owner_user_id` is a label. It is set through `ownerId` on
`PATCH /api/v1/roadmap/{nid}` (`server/internal/api/roadmap.go:180-182`); the
store refuses an owner who is not a member of the node's workspace
(`server/internal/store/roadmap.go:268-272`). Setting it grants the owner no
right, records no history (no "delegated by" or "when"), and leaves the
node's children ownerless. Only the web sets it, through the owner picker in
the Edit form (`client/lib/features/project/roadmap_node_menu.dart:153`); the
Go client's `RoadmapUpdate` carries no owner
(`server/internal/apiclient/roadmap.go:82`), so neither the CLI nor MCP can
delegate.

### One write rule

`requirePlanWriter` (`server/internal/store/memberships.go:232-259`) is the
single write rule for the tree: the project owner or a workspace admin may
write; a plain member may read and leave notes. Every tree write calls it:

| Write | Call site |
| --- | --- |
| Create a node | `server/internal/store/roadmap.go:136` |
| Edit a node, including status and owner | `server/internal/store/roadmap.go:261` |
| Delete a node and its subtree | `server/internal/store/roadmap.go:324` |
| Move a node | `server/internal/store/roadmap_move.go:25` |
| Replace the whole tree (Build) | `server/internal/store/roadmap_plan.go:59` |
| Add confirmed children (Break down) | `server/internal/store/roadmap_plan.go:155` |
| Add a validation observation | `server/internal/store/roadmap_validation.go:44` |
| Validate a node | `server/internal/store/roadmap_validation.go:74` |
| File an agent proposal | `server/internal/store/project_updates.go:57` |
| Apply proposal items | `server/internal/store/proposal_items.go:131` |

The rule never reads `owner_user_id`, so the person a node is assigned to is
refused on every write inside it unless they already own the project or are
an admin (`TREEB-02`).

### Row-level security

The policies on `roadmap_nodes` are looser than the store:

- `roadmap_nodes_select`, `roadmap_nodes_update` and `roadmap_nodes_delete`
  admit any member of the project's workspace
  (`server/db/migrations/20260722120000_roadmap_nodes.sql:38-72`).
- `roadmap_nodes_insert` admits any workspace member whose `created_by` is
  the caller
  (`server/db/migrations/20260726100000_roadmap_attribution.sql:20-28`).

A plain member therefore passes the database check on update and delete;
only the Go rule stops the write (`TREEB-07`, reproduced on PostgreSQL). The
evidence insert policy does match the store: project owner or workspace admin
(`server/db/migrations/20260821130000_roadmap_evidence.sql:30-39`).

### Routes

| Route | Effect | Code |
| --- | --- | --- |
| `GET /api/v1/projects/{pid}/roadmap` | Whole tree as a flat list, any member | `server/internal/api/roadmap.go:17`, `server/internal/store/roadmap.go:178` |
| `POST /api/v1/projects/{pid}/roadmap` | Create a node | `server/internal/api/roadmap.go:18` |
| `PATCH /api/v1/roadmap/{nid}` | Edit fields, status and owner | `server/internal/api/roadmap.go:19` |
| `POST /api/v1/roadmap/{nid}/move` | Re-parent or reorder | `server/internal/api/roadmap.go:20` |
| `DELETE /api/v1/roadmap/{nid}` | Delete with subtree | `server/internal/api/roadmap.go:21` |
| `GET /api/v1/projects/{pid}/roadmap/progress` and `/stream` | Progress view and its live ping | `server/internal/api/roadmap.go:23-24` |
| `GET`, `POST /api/v1/roadmap/{nid}/evidence` | Read evidence, add an observation | `server/internal/api/roadmap.go:25-26` |
| `POST /api/v1/roadmap/{nid}/validate` | Human validation | `server/internal/api/roadmap.go:27` |
| `POST /api/v1/prds/{pid}/build` and `/confirm` | Draft (read-only, approved PRD only) and replace the tree; confirm creates the project and refuses to erase evidence or done or validated nodes without `force` | `server/internal/api/roadmap_ai.go:68-69` |
| `POST /api/v1/roadmap/{nid}/expand` and `/confirm` | Break a node down | `server/internal/api/roadmap_ai.go:70-71` |
| `GET /api/v1/agent/context` | Agent briefing | `server/internal/api/agent_routes.go:19` |

There is no read of one node or one subtree, and no read of the nodes a
person owns. An agent token may call `PATCH /roadmap/{nid}`, so it can change
an owner (`server/internal/api/agent_scope.go:73`); delete is human-only
(`agent_scope.go:97`). The only change signal is an in-memory callback
(`server/internal/store/pool.go:49`) that pings open progress streams. The web
has one roadmap route per project (`client/lib/router.dart:43`). MCP exposes
`roadmap_list`, `roadmap_add`, `roadmap_update`, `roadmap_move`,
`roadmap_delete` and `roadmap_evidence`
(`server/internal/mcpserver/product_tools.go`).

### Agent context focus

`GET /api/v1/agent/context?focus={nid}` reads the whole project tree
(`server/internal/api/agent_routes.go:38-50`). In implementation mode the
assembler adds one `focus` block for the node and then lists every leaf of
the project, without parent ids
(`server/internal/agentcontext/agentcontext.go:95-111`). An unknown focus id is
ignored rather than refused (`TREEX-05`).

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

None of the target is built. It lands in wave 2 (rows 2.1 to 2.9) of the
review 5 remediation, after the destructive-write guards of rows 1.4 and 1.5:
[execution plan](../plans/2026-11-21-rv5-remediation/2026-11-21-rv5-remediation.md),
[row detail](../review/2026-11-21-focused-review-5/40-remediation-plan.md).
Update this page as each slice lands.
