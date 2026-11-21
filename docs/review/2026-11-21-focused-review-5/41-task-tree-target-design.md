# 41 · The task tree for roles: where it stands and what to build

The owner's requirement, as given for this review:

> Ensure that each user role has its own work tree. When clicking a specific
> node, it can be delegated to the next person, and opening it shows that
> person's view. Essentially, each node represents a subtree that can be
> expanded into a new tree, allowing everyone to focus on their own task tree.
> At the same time, managers can view these nodes when needed to check
> details.

This file joins the two task-tree lens files
([backend](./15-tree-model.md), [experience](./16-tree-experience.md)) into
one answer. It is a direction for the owner to accept or change, not a built
design. IDs point at the [findings register](./01-findings-register.md); rows
in the last column point at the [remediation plan](./40-remediation-plan.md).

## Requirement by requirement

| The requirement says | Today | Evidence | Plan row |
| --- | --- | --- | --- |
| Each person has their own work tree | Missing. One shared tree per project; no read or view scoped to a person | `TREEB-04`, `TREEX-03` | 2.5 |
| Clicking a node offers delegation | Partial. An owner select sits inside the generic Edit form on the web. Not on MCP or CLI. Children stay ownerless and nothing is recorded | `TREEX-08`, `PARC-01` | 2.3, 2.4, 2.6 |
| The node is delegated to the next person | Partial. One nullable column, `roadmap_nodes.owner_user_id`. No record of who delegated, when, or to hand back | `TREEB-01` | 2.6 |
| The delegate can work in it | Missing. Reproduced (R1): the assigned owner is refused on every write inside the node | `TREEB-02` | 2.2 |
| Opening the node shows that person's view | Missing. One roadmap route per project, a node card has no tap action, no server read returns one subtree | `TREEX-02`, `TREEB-03` | 2.1, 2.4 |
| Each node is a subtree that expands into a new tree | Partial. "Break down with AI" adds children in place. The branch cannot be opened, collapsed or linked to | `TREEX-09` | 2.4, 2.7 |
| Everyone focuses on their own tree | Missing for people. For agents, the briefing called focus-scoped still returns every leaf of the project | `TREEX-05`, `TREEB-05` | 2.1, 2.8 |
| Managers can open these nodes and check details | Partial. A manager can read the whole project tree, so nothing is hidden, but cannot open a branch, see its progress as a unit, read a node's description outside the Edit form, or reach its evidence from the card | `TREEX-11`, `TREEB-08` | 2.4, 2.7 |
| The manager learns when delegated work changes | Missing. The only signal is an in-memory ping to open Progress pages | `TREEB-09` | 2.6 |
| The rule holds in the database too | Missing. Reproduced (R11): row-level security lets any workspace member update or delete any node; only the Go check stops it | `TREEB-07` | 2.2 |
| The concept is written down | Missing. It is in no product, architecture, issue or plan record | `TREEB-11`, `TREEX-12`, `IDX-07` | 0.4 |

Two of the eleven are implemented in some form only because the owner column
exists. That is the "one quarter" in the bottom line.

## Four people, walked through today's product

1. **A project owner hands "Payments" to a teammate.** Opens the node menu,
   chooses Edit, finds the owner select among title, status and deadline,
   saves. It works. Nothing tells the teammate, nothing is written to the
   project's update feed, and the children of "Payments" stay ownerless.
2. **The teammate, a plain member, signs in.** The landing lists workspaces.
   They open the project and see the whole tree with their name on one card.
   Every write control is shown to them. They fill the Edit form and get
   "You do not have permission to do that". They cannot start a leaf, add a
   child, break the node down or attach an observation.
3. **A manager checks the teammate's branch.** They can read everything,
   which is the part that works. They cannot open the branch on its own, the
   per-node progress bar counts only direct children in status `done`, and
   the description and evidence are not on the card.
4. **A coding agent is given one node.** `agent-context --focus <node>` adds
   one block about the node and still lists every leaf of the project, with
   no parent ids, so the agent cannot tell which leaves are its own. Its token
   can patch any node in any project the user can write.

## Target design

**Data.** Keep `roadmap_nodes.owner_user_id` as the current responsible
person. Add an append-only `insideout.roadmap_node_assignments` (node, from
user, to user, assigned by, note, time). Add a nullable `node_id` to
`project_updates` with the kinds `node_delegated`, `node_handed_back` and
`node_done`.

**One authorization rule, in one place.** A SQL function
`insideout._can_write_node(node, user)` is true for the project owner, a
workspace admin, or a current workspace member who owns the node or any
ancestor of it. The store helper `requireNodeWriter` calls it, and so do the
insert, update and delete policies on `roadmap_nodes`, so the Go code and the
database cannot drift apart again. Inside their subtree a delegate may edit,
add, break down, move, attach evidence, and re-delegate descendants. Moving,
deleting or reassigning the delegated root itself, validating, and replacing
the whole tree stay with the project owner and admins. Reads stay
workspace-wide: that is how a manager opens anything.

**Reads.** One server definition of "this node's tree":
`GET /projects/{pid}/roadmap?root={nid}` returns the subtree with an
`ancestors` array for the breadcrumb, and `GET /roadmap/{nid}` reads one node.
`GET /workspaces/{id}/roadmap/mine` returns the caller's topmost owned nodes
across the workspace, each a root. Node views gain `rollup` (leaves done,
validated, total, earliest deadline) and `canWrite`.

**Writes.** `POST /roadmap/{nid}/delegate {toUserId, note}` sets the owner,
appends the assignment row and a project update. Handing back is the same
call with the previous delegator as the target.

**Web.** A route `/projects/:id/roadmap/:nodeId` renders the existing three
views over the subtree, with a breadcrumb to the parent. Tapping a card opens
it. The node menu gains "Delegate to" and "Open as tree"; write items are
hidden when `canWrite` is false. The roadmap page gains a "Mine" toggle and
the signed-in landing lists the caller's nodes. The canvas can collapse a
branch.

**MCP and CLI.** The same four things, in the same change as each server
slice: `roadmap_list` and `roadmap list` take a root, `roadmap_mine` and
`roadmap mine` exist, `roadmap_delegate` and `roadmap delegate` exist, and a
`members` tool and verb let a caller find a teammate's id.

**Agents.** `agent-context --focus` returns only the focus subtree and
answers 404 for an unknown node. Later, a token may carry a root node and is
refused outside it.

## Four decisions for the owner

1. **Is "own work tree" per person or per role?** The design above derives
   each person's tree from the nodes they own. The per-Project roles in
   PRODUCT.md (Decision Owner, Driver, Contributor, Reviewer, Viewer) are a
   separate slice with no record today (`FEAT-04`). Recommended: per person
   now; roles decide what a person may do, not which tree they see.
2. **Is a delegate's tree a default view or a wall?** Recommended: a default
   view. Everyone can still read the whole project; the delegate lands on
   their own tree. A wall would also hide the context a delegate needs and
   would need a second row-level-security design.
3. **May a delegate re-delegate inside their subtree?** Recommended: yes for
   descendants, no for the root they were given.
4. **May an agent token change a node's owner?** Today it can, through the
   allowed `PATCH` route, and once ownership grants write rights that becomes
   a grant of authority (`COMP-03`). Recommended: human-only, like member
   roles; an agent files it as a proposal.

## Order

Reads before rights, and destructive writes guarded before either: plan rows
1.4 and 1.5 (an agent token and a rebuild can each delete a whole tree) land
before a delegate gains write access in row 2.2. Each slice ships on REST, the
Go client, MCP, CLI and web together; the node owner field shows what happens
otherwise.
