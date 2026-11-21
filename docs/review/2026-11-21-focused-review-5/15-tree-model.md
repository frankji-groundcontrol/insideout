# 15 · Task tree for roles: backend

**Lens:** data model, authorization, API and agent context against the owner's delegation concept
**Score:** **2.5 / 10**
**Bottom line:** The backend has a sound single shared tree per project and one nullable responsible-person column on each node, but none of the delegation concept on top of it: no record of who delegated what to whom, no write authority for the person a node is assigned to, no read that returns one node's subtree or the caller's own tree, no server-side roll-up, no notification to the delegator, and no way to confine an agent to a subtree. Of the eleven properties, one is implemented, four are partial and six are missing. The concept is also written down nowhere in PRODUCT.md or docs/issues, so nothing tracks it. Not checked: the Flutter client beyond one helper, the full go test run (only three pure packages were run, all pass), and live database behaviour of the row-level-security policies (read from migrations only).

Read at `main` @ `4d186da` together with the uncommitted working tree. A skeptic re-checked every High in this file; the verdicts are in the table near the end and in the [findings register](./01-findings-register.md).

---

## Scope read

All migrations touching `roadmap_nodes`, `roadmap_evidence`, `project_updates` and their policies; `server/internal/store/roadmap*.go`, `memberships.go`, `proposal_items.go`, `project_updates.go`; `server/internal/api/roadmap*.go`, `agent_routes.go`, `agent_scope.go`, `presence_routes.go`; `server/internal/agentcontext`, `roadmaptime`, `presence`; `PRODUCT.md` (Collaboration and authority, Roadmap Graph, current status); the focus-surface, roadmap-lenses and promise-gap issue records; `docs/architecture/database-and-rls.md` and `agent-token-scope.md`. The plan `docs/plans/2026-08-19-owner-app-roles.md` is about database roles, not user roles. Not checked: the Flutter client beyond one helper, the full test run, and live policy behaviour.

## What is good

- The tree itself is sound. `parent_id` self-reference, a recursive cycle guard and sibling position repair run under one per-project advisory lock (`server/internal/store/roadmap_move.go:32-86`).
- One function decides every plan write: `requirePlanWriter` (`server/internal/store/memberships.go:236`), ten call sites. A subtree rule has one place to go.
- Owner assignment is membership-checked in the write transaction and tested on real PostgreSQL (`server/internal/store/roadmap.go:268-272`, `roadmap_acceptance_owner_test.go:79`).
- Agent routes are closed by default with a coverage test (`server/internal/api/agent_scope.go:27`, `:39`).
- `roadmaptime` and `agentcontext` are pure, and `parent_id` and `owner_user_id` are already indexed by the catalog-driven migration `20261018210000_index_foreign_keys.sql`.

## Verdict against the owner's concept

One of eleven properties is implemented (a node has a responsible person). Four are partial (manager view, roll-up, role differences, row-level-security parity). Six are missing (delegation record, delegate write authority, subtree read, my-tree read, delegator notification, agent subtree scope). Score 2.5 of 10.

## Findings, worst first

| Id | Where | Deciding line | Fix |
|---|---|---|---|
| TREEB-01 High | `server/db/migrations/20261005120000_roadmap_acceptance_owner.sql:8` | `ADD COLUMN IF NOT EXISTS owner_user_id uuid REFERENCES insideout.users(id) ON DELETE SET NULL;` is the only ownership data | Append-only `roadmap_node_assignments` table; `POST /roadmap/{nid}/delegate`; history read |
| TREEB-02 High | `server/internal/store/roadmap.go:261` | `if err := requirePlanWriter(ctx, tx, projectID, actorID); err != nil {` | `requireNodeWriter`: plan writer, or member owning the node or an ancestor |
| TREEB-03 High | `server/internal/api/roadmap.go:17` | `GET /api/v1/projects/{pid}/roadmap` is the only tree read | `root={nid}` parameter, `ListRoadmapSubtree`, `ancestors` in the response |
| TREEB-04 High | `server/internal/store/roadmap.go:201` | `WHERE n.project_id = $1` is the only node filter | `GET /workspaces/{id}/roadmap/mine` returning the caller's topmost owned nodes |
| TREEB-05 Medium | `server/internal/auth/jwt.go:69` | `return t.mint(userID, ScopeAgent)` carries no node claim | Optional `rootNodeId` claim enforced on roadmap write routes |
| TREEB-06 Medium | `server/internal/agentcontext/agentcontext.go:111` | `out["leaves"] = leavesContext(leaves, in)` uses all project leaves; `TestImplementationModeFocusScoped` expects two | Filter leaves to the focus subtree, add `focus.path`, 404 on unknown focus |
| TREEB-07 Medium | `server/db/migrations/20260722120000_roadmap_nodes.sql:56` | `CREATE POLICY roadmap_nodes_update` admits any member | `_can_write_node` function shared by store and policies; write-walk test rows |
| TREEB-08 Medium | `server/internal/roadmaptime/roadmaptime.go:81` | `if !n.Leaf {` then `continue` | Pure `Rollup`, `rollup` field on node views, `root=` on progress |
| TREEB-09 Medium | `server/internal/store/pool.go:51` | `s.onRoadmapChange(projectID)` is the whole signal | `node_id` and three kinds on `project_updates`, written in the same transaction |
| TREEB-10 Medium | `server/internal/store/roadmap_plan.go:58` | `DELETE FROM insideout.roadmap_nodes WHERE project_id = $1` | Refuse replace or delete over others' assigned or evidenced nodes without an explicit flag; never for agent tokens |
| TREEB-11 Medium | `PRODUCT.md:246` | The only responsibility sentence; grep for delegat or subtree finds nothing relevant | PRODUCT.md subsection, dated issue, promise-gap row, plan |
| TREEB-12 Low | `server/internal/store/memberships.go:149` | Member removal leaves node ownership; proposals resolve parents by title (`proposal_items.go:187`) | Hand back on removal; `parent_id` on proposal items; validate checkpoint node id |

## Lead verification

| Lead | Result | Line that decided it |
|---|---|---|
| 1 | Confirmed | `20261005120000_roadmap_acceptance_owner.sql:8`; grep for `owner_user_id` finds no second column |
| 2 | Confirmed | `memberships.go:255` `if role != "admin" {`; ten call sites, none reads the node owner |
| 3 | Confirmed | `api/roadmap.go:17-27`; grep for subtree, owner=, mine, assigned returns one error string |
| 5 | Confirmed | `mcpserver/product_tools.go:67-71` lists title, description, status, deadline, completion_reason only; no owner in `apiclient` |
| 6 | Confirmed, with an addition | Per-Project roles are disclosed but not tracked (`2026-10-11-product-promise-gaps.md:42`); the delegation concept is in no record |
| 7 | Not rerun in full | `go test` on agentcontext, roadmaptime, presence passes |
| 9 | Consistent | `store/roadmap.go` 335 lines, `api/roadmap.go` 344: under the rule, close to it |
| 13 | Partly | 91 literal `mux.HandleFunc("` registrations in non-test api files by my grep; not reconciled with 107 |
| 14 | One stale row | R2-04 is still listed open, but `roadmap_plan.go:133` now calls `requirePlanWriter` and inserts via `createRoadmapNodeTx` |

## Recommendations

1. Record the concept before building it (TREEB-11).
2. Build reads before rights: subtree read, then delegate authority with its database twin, then history, my-tree, roll-up, timeline.
3. Keep one ancestor rule in SQL, used by Go and by the policies.
4. Ship each addition to REST, apiclient, MCP and CLI together.
5. Guard destructive tree writes before any delegate can write.

## Target design

**Tables.** Keep `roadmap_nodes.owner_user_id` as the current responsible person. Add `roadmap_node_assignments (id, node_id, from_user_id, to_user_id, assigned_by, note, created_at)`, append-only. Add `project_updates.node_id` (nullable) and kinds `node_delegated`, `node_handed_back`, `node_done`.

**Authorization rule.** One SQL function `insideout._can_write_node(node, user)`: true for the project owner, a workspace admin, or a current member who owns the node or any ancestor. The store helper `requireNodeWriter` and the `roadmap_nodes` insert, update and delete policies both call it. A delegate may edit, add, expand and move inside the subtree, may delete strict descendants, and may re-delegate descendants. Moving, deleting or reassigning the delegated root, validating, and replacing the tree stay with plan writers. Reads stay workspace-wide, so managers can open anything.

**Routes.** `GET /projects/{pid}/roadmap?root={nid}` (subtree plus `ancestors`); `GET /projects/{pid}/roadmap/progress?root={nid}`; `GET /workspaces/{id}/roadmap/mine`; `POST /roadmap/{nid}/delegate {toUserId, note}`; `GET /roadmap/{nid}/assignments`; `GET /projects/{pid}/updates?node={nid}`. Node views gain `rollup`. `POST /agent/token` accepts `rootNodeId`; `GET /agent/context?focus=` returns only that subtree.

**Migration order.** (1) `_can_write_node` and replaced policies, behaviour-neutral for owners and admins. (2) assignments table, backfilled with one row per existing owner. (3) `project_updates.node_id` and kinds. Each migration ships with its store change and real-database tests.

## What this role would do first

Add the `root={nid}` subtree read with its ancestor path. It changes no permission, reuses the recursive query already in `roadmap_move.go`, and gives every surface one definition of a node's tree. Delegate authority, the manager's drill-in, scoped progress and scoped agent context all depend on that definition existing on the server.

## Property checks

| Property | Status | Evidence |
| --- | --- | --- |
| (a) A node has a responsible person | implemented | roadmap_nodes.owner_user_id (server/db/migrations/20261005120000_roadmap_acceptance_owner.sql:8), written through PATCH /roadmap/{nid} ownerId with a workspace-membership check (server/internal/store/roadmap.go:268-272) and returned with the owner's name (roadmap.go:196-200). Nullable; nothing enforces 'exactly one owner' on Now items. |
| (b) A node can be delegated to a next person, with who delegated, when, and a way to hand back or re-delegate | missing | The owner can be overwritten (roadmap.go:295) but no delegator, timestamp, note or history is stored and no delegate or hand-back route exists (routes at server/internal/api/roadmap.go:17-27). TREEB-01. |
| (c) The delegate gains write authority over that node and its descendants and nothing above it | missing | All ten write sites call requirePlanWriter, which allows only the project owner or a workspace admin (server/internal/store/memberships.go:236-258); owner_user_id is never consulted. TREEB-02. |
| (d) A read that returns one node's subtree as a tree in its own right | missing | Only GET /projects/{pid}/roadmap exists, filtering `WHERE n.project_id = $1` (server/internal/store/roadmap.go:201); no single-node or subtree route. The recursive CTE needed already exists in roadmap_move.go:42-47. TREEB-03. |
| (e) A read that returns 'my tree' for the caller | missing | No route or store function filters by owner_user_id; grep for subtree, owner=, mine, assigned in server/internal/api returns only an error message. TREEB-04. |
| (f) A manager can open any delegated subtree and see its detail | partial | Any workspace member, so any manager, can read the full project tree and each node's evidence (ListRoadmap requireMember at roadmap.go:185; GET /roadmap/{nid}/evidence). There is no way to open one subtree as a view with its path and roll-up, and no assignment history to inspect. TREEB-03, TREEB-08. |
| (g) Status and progress roll up from a subtree to its parent node | partial | Server-side there is none: roadmaptime.Assemble skips non-leaves (roadmaptime.go:81) and covers the whole project. The web client computes a one-level done share that ignores validated children (client/lib/features/project/roadmap_node_card.dart:50). TREEB-08. |
| (h) The delegator is told when the subtree changes or completes | missing | The only signal is an in-memory per-project ping to open Progress streams (server/internal/store/pool.go:49-53); no timeline kind for delegation or completion, and project_updates has no node column. TREEB-09. |
| (i) Roles differ in what they see and may do (workspace role, project owner, node owner, agent token) | partial | Three of four differ: workspace admin and project owner write the plan, plain members read (memberships.go:236-258), agent tokens are refused deletes, decisions, validation and evidence entry (server/internal/api/agent_scope.go:95-126). Node owner has no distinct right or view. Per-Project roles are disclosed as not built (PRODUCT.md:429-436). All members see every node. |
| (j) Row-level security enforces the same boundaries as the store code | partial | roadmap_evidence INSERT matches the store (owner or admin, 20260821130000_roadmap_evidence.sql:30-39). roadmap_nodes UPDATE and DELETE policies admit any workspace member (20260722120000_roadmap_nodes.sql:56-72) while the store requires owner or admin. TREEB-07. |
| (k) An agent can be scoped to one subtree and cannot write outside it | missing | FocusNodeID only adds a focus block; leaves remain project-wide (server/internal/agentcontext/agentcontext.go:95, :111) and the agent token has no project or node claim (server/internal/auth/jwt.go:69), with roadmap write routes open by pattern (agent_scope.go:72-81). TREEB-05, TREEB-06. |

## Skeptic verdicts on this lens's Highs

| ID | Verdict | Reason | What the skeptic ran |
| --- | --- | --- | --- |
| TREEB-01 | DOWNGRADED to Medium | The facts hold: owner_user_id is the only ownership column on a node and a reassignment overwrites it with no history row (my probe on the private database lists only roadmap_evidence and roadmap_nodes as roadmap tables, and only projects.owner_id and roadmap_nodes.owner_user_id as owner/assign/delegate columns). The severity does not hold. The owner's stated requirement asks that a node can be delegated to the next person and opened as that person's view; it does not ask for a delegation ledger, a note, hand-back or undo. The two blocking steps in the failure scenario (the colleague passes a child on; the colleague hands the workstream back) fail today because of the permission rule in TREEB-02, not because the delegator was never stored: in my probe the delegate's attempt to set another owner returned forbidden, and with write rights a delegate could pick the manager from the member list without any history table. The same UPDATE also stamps updated_by, so the last editor is kept (a weak trace that the next edit overwrites). What remains is an audit-trail gap that will bite once delegation chains exist, which is Medium. It does not restate an open record under docs/issues/. | go test ./internal/store/ -overlay /tmp/insideout-rv5/tree-backend/overlay.json -run TestRv5TreeBackendProbe -v -count=1 (private PG18) -> "manager reassigns Payments to third: err=<nil>"; "roadmap tables: [roadmap_evidence roadmap_nodes]"; "owner/assign/deleg columns: [projects.owner_id roadmap_nodes.owner_user_id]" |
| TREEB-02 | CONFIRMED | Reproduced on the private database with my own probe. The manager assigned node Payments to a plain member (err=nil, owner set). The member was then refused on every write inside that node: set own node in_progress, rename a leaf under it, add a child, move a leaf within it, expand it, record an observation on its leaf, re-delegate the leaf, delete the leaf: all forbidden. No call site of requirePlanWriter reads owner_user_id, and the cited line numbers are right (create :136, update :261, delete :324, move roadmap_move.go:25, expand roadmap_plan.go:133, observation and validate roadmap_validation.go:44 and :74). The server accepts the assignment of a member who can then do nothing with it, and the web node menu is not gated by role, so the delegate reaches the 403. This is the owner-required delegation capability absent, and it also contradicts PRODUCT.md:246 (each Now item has one human responsible owner). I did not downgrade on 'recorded decision': BUG-020 (docs/issues/2026-10-06-bug-020-workspace-member-writes-the-plan.md, fixed, not open) is the rule that produces this, but it was written for an unassigned reader and never considers an assigned owner. Not a restatement of an open record. Same root cause as TREEX-01. | go test ./internal/store/ -overlay /tmp/insideout-rv5/tree-backend/overlay.json -run 'TestRv5TreeBackendProbe\|TestRoadmap_MemberCannotWrite' -v -count=1 (private PG18) -> "manager assigns Payments to delegate: err=<nil> owner=true"; "delegate sets own node in_progress: err=forbidden"; "delegate adds child under own node: err=forbidden"; TestRoadmap_MemberCannotWrite PASS |
| TREEB-03 | DOWNGRADED to Medium | The facts hold: my route inventory of server/internal/api shows GET /api/v1/projects/{pid}/roadmap (plus /progress and its stream) as the only tree reads and no GET /api/v1/roadmap/{nid}; the query filters on project only; the probe's delegate read returned all 4 nodes including the sibling branch. But a subtree is a strict cut of a response every caller already holds (each node carries parentId and ownerId), and the web client already makes that cut for its band columns (the quoted line). Reading is deliberately open to every workspace member and the owner's requirement keeps managers able to look in, so no permission boundary depends on a server-side subtree read. What blocks 'a node opens as its own tree' is the missing view, not this endpoint. The harms the finding itself lists (each surface cutting by its own rule, no ancestor path, no payload limit) are design debt, which is Medium. Adjacent open record: docs/issues/2026-10-11-roadmap-lenses.md, whose fix prompt already directs per-role views to be derived client-side from the existing graph payload; it does not cover per-node subtrees, so this is not a restatement. Shares the evidence line roadmap.go:201 with TREEB-04 but needs a different fix, so not a duplicate. | grep -rn 'HandleFunc(' server/internal/api/*.go (non-test), sorted -> roadmap GETs are only 'GET /api/v1/projects/{pid}/roadmap', '.../roadmap/progress', '.../roadmap/progress/stream', 'GET /api/v1/roadmap/{nid}/evidence'; probe: "delegate ListRoadmap: 4 nodes err=<nil>" |
| TREEB-04 | CONFIRMED | I could not refute it. The only node listing filters by project alone, no route under /me or /workspaces/{id} touches the roadmap (route inventory run), and no surface filters by owner: the Flutter client only prints ownerName on a card and in the progress view, the CLI roadmap verbs (list, add, update, move, delete, progress, presence) have no owner or mine flag, and the MCP roadmap tools take none. So a person who was handed nodes has no way to find them except opening every project and scanning the whole graph, and no single existing response contains the answer across projects. The owner's stated requirement (each role has its own work tree; everyone focuses on their own task tree) makes this an owner-required capability that is absent, which is High. One overstatement: 'no surface can open on it' is too strong inside a single project, where a client could filter the existing list on ownerId; the cross-project entry point is what only the server can supply. Adjacent open records (docs/issues/2026-10-11-focus-surface.md on the per-session focus pointer and team Now; 2026-10-11-roadmap-lenses.md) do not cover an owner-scoped read, so this is not a restatement. Related to TREEB-03 through the shared evidence line but a different query and fix. | grep -rn 'HandleFunc(' server/internal/api/*.go (non-test), sorted -> no roadmap route under /api/v1/me or /api/v1/workspaces/{id}; grep -rnI -i 'subtree\|myNodes\|ownedBy\|assignedToMe\|onlyMine\|ownerFilter' client/lib -> only roadmap_canvas.dart _subtree; grep owner\|mine server/cmd/insideout/cli_roadmap.go -> no match |

## Ordered recommendations

1. Write the concept down first: a PRODUCT.md subsection and a dated issue with the slice order, listed in the promise-gap record so the existing check tracks it (TREEB-11).
2. Deliver in this order so each slice is usable alone: subtree read (TREEB-03), then delegate write authority with its row-level-security twin (TREEB-02, TREEB-07), then the assignment history and delegate route (TREEB-01), then my-tree (TREEB-04), then roll-up and timeline rows (TREEB-08, TREEB-09).
3. Put the ancestor walk in one place: a single SQL function insideout._can_write_node used by both the store helper and the policies, so Go and the database cannot drift again.
4. Ship every new read and the delegate write to all four surfaces in the same change (REST route, apiclient method, MCP tool, CLI verb); the node owner field already shows what happens otherwise (lead 5: web can set it, MCP and CLI cannot).
5. Guard tree replacement and subtree deletion against other people's assigned work before granting delegates write access (TREEB-10).
6. Scope agents after humans: restrict the context leaves to the focus subtree now (small, pure change), add the token node claim once requireNodeWriter exists (TREEB-06, TREEB-05).
7. Close the stale R2-04 row in docs/issues/2026-10-18-review-4-open-findings.md:106: ExpandRoadmapNode now calls requirePlanWriter (roadmap_plan.go:133) and inserts through createRoadmapNodeTx, which takes the lock.
