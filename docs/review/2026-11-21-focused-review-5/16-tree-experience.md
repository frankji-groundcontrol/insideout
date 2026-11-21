# 16 · Task tree for roles: experience

**Lens:** the Flutter roadmap, MCP tools, CLI and agents against the owner's delegation concept
**Score:** **2.5 / 10**
**Bottom line:** The task-tree concept the owner calls crucial is about one quarter built. A roadmap node can carry one owner, the web edit form can set it, and every card shows it. Nothing else of the concept exists: the owner of a node cannot edit that node or anything under it, no node can be opened as its own tree, there is no "my tree" view or landing, the canvas cannot collapse a branch, the MCP tools and the CLI cannot set a node owner at all, and the agent briefing that is described as focus-scoped still returns every leaf of the project. Not checked: I did not run the Flutter app or the server, so the four walks are read from the code and the tests, not exercised live.

Read at `main` @ `4d186da` together with the uncommitted working tree. A skeptic re-checked every High in this file; the verdicts are in the table near the end and in the [findings register](./01-findings-register.md).

---

Score: 2.5 / 10.

## Scope read

All roadmap widgets in `client/lib/features/project/`, `client/lib/router.dart`, `client/lib/api/models_roadmap.dart` and `client_project.dart`, the dashboard and workspace pages, the roadmap widget tests, `server/internal/mcpserver/product_tools.go` and `draft_tools.go`, `server/cmd/insideout/cli_roadmap.go`, `server/internal/agentcontext`, `server/internal/guide`, the roadmap store and API handlers, `server/internal/api/agent_scope.go`, `PRODUCT.md`, `DESIGN.md` and `docs/design-qa`. Nothing was run live; the walks below are read from code and tests.

## What is good

- The tree is one flat list with parent links, a cycle guard and a per-project lock (`server/internal/store/roadmap_move.go:25-50`), so a subtree view needs no schema change.
- Every roadmap write passes one gate (`server/internal/store/memberships.go:236`), so delegated rights can be added in one place.
- The owner is readable end to end: `server/internal/api/roadmap.go:45-46`, `client/lib/features/project/roadmap_node_card.dart:101-108`, and a passing widget test (`client/test/roadmap_acceptance_owner_test.dart:114`).
- Agent tokens meet a closed-by-default route policy (`server/internal/api/agent_scope.go:72-78`).

## Four people, walked through today's product

1. **Project owner hands over "Payments".** Roadmap, Canvas view, card menu, Edit, owner select, Save: works, and the card reads "Owner: name". It stops there: the twelve children stay ownerless, nothing is recorded, nobody is told, and the CLI and MCP cannot do it at all.
2. **The teammate, a plain member.** Signs in to a list of workspaces; nothing says a node is theirs. Opens the whole project tree and finds their card by reading labels. Edits their own node and saves: 403, "You do not have permission to do that".
3. **The manager, a workspace admin.** Can open the whole project tree and edit anything. Cannot open the teammate's branch alone, cannot fold other branches, sees a bar of direct children only, and must open the Edit form to read a description.
4. **A coding agent given one node.** With only a node id it cannot find the project. With the project id, `agent_context` with `focus` returns the node's title and siblings plus every leaf of the project. It can update the node only if its user is the project owner or an admin.

## Findings, worst first

| Id | Sev | Where | Deciding line | Fix |
|---|---|---|---|---|
| TREEX-01 | High | `server/internal/store/roadmap.go:261` | `if err := requirePlanWriter(ctx, tx, projectID, actorID); err != nil {` | Add `requireNodeWriter` in `memberships.go`: plan writer, or owner of the node or an ancestor. Test `TestDelegateWritesInsideSubtreeOnly`. |
| TREEX-02 | High | `client/lib/router.dart:43` | `'/projects/:id/roadmap': (c, s) =>` | Route `/projects/:id/roadmap/:nodeId`, filter with `roadmapSubtree`, breadcrumb in `RoadmapHeader`, "Open as tree" menu entry. |
| TREEX-03 | High | `client/lib/features/dashboard/dashboard_page.dart:118` | `return HomeField(` | "Mine" view toggle (client only), then `GET /api/v1/me/roadmap` and a "My work" block on the dashboard. |
| TREEX-04 | High | `server/internal/apiclient/roadmap.go:82` | `func (c *Client) RoadmapUpdate(nodeID string, title, description, status, deadline, completionReason *string)` | `owner_id` and `acceptance` on `roadmap_update`; `--owner`, `--acceptance` on the CLI; a `members` read. |
| TREEX-05 | High | `server/internal/agentcontext/agentcontext.go:111` | `out["leaves"] = leavesContext(leaves, in)` | Restrict leaves to the focus's descendants, add description and path, 404 on an unknown focus, correct the test at `agentcontext_test.go:31`. |
| TREEX-06 | Medium | `server/internal/api/roadmap.go:19` | `mux.HandleFunc("PATCH /api/v1/roadmap/{nid}", ...` (no GET beside it) | `GET /api/v1/roadmap/{nid}?subtree=1`, MCP `roadmap_get`, CLI `roadmap show`. |
| TREEX-07 | Medium | `client/lib/features/project/roadmap_node_menu.dart:21` | `itemBuilder: (_) => [` | Return `canWritePlan`; hide write entries; check the writer rule before the AI draft. |
| TREEX-08 | Medium | `roadmap_node_menu.dart:153` | `key: const Key('owner-picker'),` | A "Delegate" menu entry, inherited owner labels, a feed record on change. |
| TREEX-09 | Medium | `client/lib/features/project/roadmap_canvas.dart:101` | `nodes: _subtree(roots[i]),` | Collapsed-id set on the page, chevron with "+N" on cards. |
| TREEX-10 | Medium | `client/lib/features/project/roadmap_node_card.dart:50` | `final done = children.where((c) => c.status == 'done').length;` | Leaf-based `(done, total)` counting validated; print the numbers. |
| TREEX-11 | Medium | `roadmap_node_card.dart:79` | `Text(node.title, style: theme.textTheme.titleMedium),` | Tap opens a read-only detail sheet with description and evidence. |
| TREEX-12 | Low | `PRODUCT.md:246` | `Each Now execution item has exactly one human responsible owner.` | Record the concept in design-QA, one issue, PRODUCT.md and the BUG-020 record. |

## Lead verification

| Lead | Result | Line that decided it |
|---|---|---|
| 1 | Confirmed | `ADD COLUMN IF NOT EXISTS owner_user_id uuid` in the 20261005120000 migration; grep for delegat, assignee, subtree over the migrations is empty. |
| 2 | Confirmed | `server/internal/store/memberships.go:254` `if role != "admin" {` after the project-owner check. |
| 3 | Confirmed | `server/internal/api/roadmap.go:17-27`; no GET for a single node. |
| 4 | Confirmed | `client/lib/features/project/roadmap_canvas.dart:74` is the only subtree helper; the owner select is at `roadmap_node_menu.dart:152-161`. |
| 5 | Confirmed, and wider | `product_tools.go:67`; acceptance is missing on MCP and CLI as well as owner. |
| 6 | Confirmed by record | `docs/issues/2026-10-11-focus-surface.md` and `PRODUCT.md:246-249`. |
| 9 | Noted | `roadmap_node_menu.dart` is 349 lines and `roadmap_page.dart` 337; both will cross the limit with the fixes above, so new widgets belong in new files. |
| 13 | Partly checked | Eleven roadmap routes plus two expand routes and five agent routes were read; the wider inventory was not counted. |
| 14 | One stale row | R2-04 still reads as open, but `server/internal/store/roadmap_plan.go:133` now calls `requirePlanWriter`. None of the twelve findings here is in the review-4 register, which marks owner and acceptance as implemented. |
| 7, 8, 10, 11, 12 | Not touched | Outside this lens. |

## Recommendations

1. Write the rule down first: an owner covers the subtree and may edit inside it; deleting the delegated root, validating and re-assigning the root stay with the plan writer.
2. Server slice: TREEX-01, TREEX-06, TREEX-05, each red-first.
3. MCP and CLI level with the web in the same release: TREEX-04.
4. Web slice: subtree route, Mine filter, Delegate entry; then collapse, progress and the detail sheet.

## Target experience

The project owner taps "Payments", chooses "Delegate", and picks a teammate. The card and every card under it read "Owner: teammate", and the project feed records the hand-over. The teammate signs in and sees "My work" on the dashboard: one row per branch they own, with "3/8 leaves done". Tapping it opens `/projects/<id>/roadmap/<node>`: only their branch, a breadcrumb "Project / Payments" back to the parent tree, and full edit rights inside the branch and none outside it. They delegate "Refunds" one level down the same way, and that person gets their own tree. A manager opens the project tree, sees each branch folded with its owner and its leaf count, unfolds one, taps a card to read its description and evidence without an edit form, or opens it as its own tree to see exactly what its owner sees. From the CLI, `insideout roadmap update --owner <user> <node>` and `insideout roadmap show --subtree <node>` do the same; the MCP tools `roadmap_update` (with `owner_id`) and `roadmap_get` mirror them. An agent given one node calls `agent_context` with `focus` and receives that node, its path, its description and only its own leaves.

## What this role would do first

Write the ownership rule into the product record and implement `requireNodeWriter` with a failing-first real-database test. Until the person a node was handed to can change it, every view built on top of ownership is a label, not a work tree.

## Property checks

| Property | Status | Evidence |
| --- | --- | --- |
| Clicking a node offers "delegate to" | partial | The card has no tap handler; the three-dot menu lists edit, child, up, down, expand, observe, validate, delete (client/lib/features/project/roadmap_node_menu.dart:21-42) and no delegate entry. The owner is one select inside the Edit form (roadmap_node_menu.dart:152-161, key 'owner-picker'). It sets that one node only, writes no record and notifies nobody. |
| A node can be opened as its own tree (route, breadcrumb back to the parent, deep link) | missing | The only roadmap route is '/projects/:id/roadmap' (client/lib/router.dart:43) and RoadmapPage takes only projectId (client/lib/features/project/roadmap_page.dart:28). grep for breadcrumb\|context.push\|onTap in client/lib/features/project found only the minimap band jump and the view toggles. |
| A "my tree" landing or filter | missing | The dashboard renders HomeField with workspaces only (client/lib/features/dashboard/dashboard_page.dart:118-124). The roadmap page has three views, progress, bands and canvas (roadmap_page.dart:36), and no owner filter. No server route reads nodes by owner (server/internal/api/roadmap.go:17-27). |
| The canvas shows who owns each node | implemented | client/lib/features/project/roadmap_node_card.dart:101-108 prints 'Owner: <name>' on every card in the Canvas and Bands views; the Progress view does the same (roadmap_progress_view.dart:144-151). The minimap does not, and the viewer's own nodes are not distinguished. |
| A delegate can edit inside their subtree | missing | server/internal/store/roadmap.go:261 gates UpdateRoadmapNode on requirePlanWriter, which admits only the project owner or a workspace admin (server/internal/store/memberships.go:248-257). The same gate covers create, move, delete, expand confirm, observation and validation. |
| A manager can open any subtree | partial | A workspace admin can open the whole project roadmap and edit any node, but only as the full project tree; there is no subtree route and no per-person view (client/lib/router.dart:43). The node description is visible only inside the Edit form (roadmap_node_menu.dart:119). |
| Collapse and expand of subtrees | missing | client/lib/features/project/roadmap_canvas.dart:101 passes _subtree(roots[i]), the full descendant list, to every band. grep for collaps\|ExpansionTile in client/lib/features/project returned nothing. |
| Progress shown per subtree | partial | client/lib/features/project/roadmap_node_card.dart:47-52 draws a bar from the share of direct children whose status is 'done'. It ignores deeper descendants and 'validated' children and shows no numbers. The server progress reading is project-wide only (server/internal/api/roadmap_progress.go:107-123). |
| MCP: delegate a node, open a subtree, list my nodes, see the owner | partial | roadmap_list returns the API JSON, so ownerId and ownerName are readable (server/internal/mcpserver/product_tools.go:52-55). roadmap_update has no owner or acceptance parameter (product_tools.go:67-71). There is no subtree or owner-scoped tool among the roadmap tools (product_tools.go:52-108, draft_tools.go:18-50). |
| CLI: delegate a node, open a subtree, list my nodes, see the owner | partial | insideout roadmap list prints the API JSON with ownerId. insideout roadmap update accepts --title, --description, --status, --deadline, --completion-reason only (server/cmd/insideout/cli_roadmap.go:69-76). The subcommands are list, add, update, move, delete, progress, presence (cli_roadmap.go:21). |
| An agent can ask for the subtree it was given | partial | agent_context accepts focus (product_tools.go:192-193) and adds one 'focus' block with the node and its sibling titles (server/internal/agentcontext/agentcontext.go:105-109), but 'leaves' stays every leaf of the project (agentcontext.go:95, 111), the focus node's description and children are absent, and there is no single-node read route. |

## Skeptic verdicts on this lens's Highs

| ID | Verdict | Reason | What the skeptic ran |
| --- | --- | --- | --- |
| TREEX-01 | CONFIRMED | Same root cause and same probe result as TREEB-02: after the manager assigned Payments to a plain member, the member's status change on that node, child add, move, expand and observation all returned forbidden on the private database. The web wording in the scenario is real: client/lib/l10n/en_shell.dart has 'write.failed': 'Save failed — {error}' and 'error.forbidden': 'You do not have permission to do that', and the node edit menu (client/lib/features/project/roadmap_node_menu.dart) is not hidden from a plain member, so the delegate reaches the refusal. The server's own 403 text confirms the rule: 'only the project owner or a workspace admin can change the roadmap, and an assigned owner must be a member'. Owner-required delegation is absent, so High stands; this adds no harm beyond TREEB-02 and should be counted once. BUG-020 (fixed, not open) is the rule behind it and does not consider an assigned owner; not a restatement of an open record. | go test ./internal/store/ -overlay /tmp/insideout-rv5/tree-backend/overlay.json -run TestRv5TreeBackendProbe -v -count=1 (private PG18) -> "delegate sets own node in_progress: err=forbidden"; "delegate moves leaf inside own node: err=forbidden"; "delegate expands own node: err=forbidden"; "delegate records observation on leaf under own node: err=forbidden" |
| TREEX-02 | CONFIRMED | Every part of the claim holds. router.dart has exactly one roadmap entry; RoadmapPage's only constructor parameter is projectId; RoadmapNodeCard is a FieldCard with no InkWell/GestureDetector/onTap (the only onTap in the roadmap files are the view toggles and the minimap band scroll); the node menu offers edit, child, up, down, expand, observe, validate, delete and nothing that opens a branch; RoadmapCanvas._subtree is a private layout helper. The owner's stated requirement says each node is a subtree that expands into a new tree, so this is an owner-required core capability that is absent, which is High on the given scale. No open record covers it: docs/issues/2026-10-11-focus-surface.md is about a per-session Focus pointer and team Now, and says 'Focus is a pointer, never ownership'. | grep -n roadmap client/lib/router.dart -> only lines 17 (import) and 43; grep -rn 'onTap\|rootId\|subtree' client/lib/features/project/roadmap*.dart -> roadmap_page.dart:213 (view toggle), roadmap_minimap.dart:53 (band scroll), roadmap_canvas.dart:74 (_subtree layout helper); no tap on the card. Read only otherwise. |
| TREEX-03 | CONFIRMED | All three halves hold. DashboardPage builds HomeField from List<Workspace> only, and home_field.dart contains no roadmap, node, owner or 'assigned' reference. The roadmap views are exactly progress, bands and canvas; ownerId is read only in the edit modal's owner picker and ownerName only printed as text. On the server the only /me routes are GET, PATCH and DELETE /api/v1/me, and owner_user_id appears in the store only in the column list, the owner-name join and the UPDATE, never in a WHERE clause. The owner's requirement opens with 'each user role has its own work tree', so the absence is High. Not a restatement of an open record: the Focus surface record covers a session pointer, not owner-scoped trees. | grep -n -i 'roadmap\|owner\|mine\|assigned\|node' client/lib/features/dashboard/home_field.dart -> no output; grep -n '/me' server/internal/api/agent_scope.go -> only GET/PATCH/DELETE /api/v1/me; grep -rn owner_user_id server/internal/store/*.go -> roadmap.go:80, :200, :295 only (no filter). Read only otherwise. |
| TREEX-04 | CONFIRMED | The Go client's RoadmapUpdate sends only title, description, status, deadline and completionReason, and it is the only path both the CLI verb and the MCP tool use, while the REST body accepts ownerId and acceptance. The built CLI refuses both flags, and there is no members verb or client method to find a user id. docs/usage/cli-mcp-and-guidance.md heads the verb table 'Verbs (parity with web and MCP)', so this is a surface that cannot do what another can, and for the owner half it is the delegation flow the owner named, which makes it High. The acceptance half alone is weaker (see PARC-02). Same root cause as PARC-01. The general convention-only parity gap was filed as rv4-ARCH-07 in the review-4 register, but no open record names the owner or acceptance fields. | HOME=/tmp/insideout-rv5/home-tree-surfaces /tmp/insideout-rv5/insideout roadmap update --owner x <uuid> -> 'flag provided but not defined: -owner'; same with --acceptance -> 'flag provided but not defined: -acceptance'; 'insideout help' lists no members verb. |
| TREEX-05 | CONFIRMED | Every part reproduced. With a three-level fixture (MVP -> Payments{Refunds, Invoices}, Search{Indexing, Ranking}) and focus=Payments, Assemble returned a focus block of id, title, status, evidence and siblings only (no description, no children) and leaves [pay1 pay2 srch1 srch2]; leaf rows carry no parent id, so an agent cannot tell which two belong to the focus. With focus=no-such-node there was no focus block, no error key and all 4 leaves, and handleAgentContext passes the query value straight through with no lookup or 404. NodeIn has no Description field. The test TestImplementationModeFocusScoped uses root -> a, b with focus a and requires len(leaves) == 2, so it pins the unscoped list under a name that says scoped and would go red if scoping were implemented. This contradicts PRODUCT.md:334 ('compact, Focus-scoped context rather than the entire graph'), the package doc ('never the entire graph') and the MCP tool description, on the documented agent-context --focus path, and the test misstates what it checks: High on both counts. The '40 leaves' figure in the scenario is illustrative. The open record docs/issues/2026-10-11-focus-surface.md is adjacent (it notes focus exists only as this input and quotes the 'scopes the assembly' comment) but is about the missing per-session focus pointer, so this is not a restatement. | go test ./internal/agentcontext/ -overlay /tmp/insideout-rv5/tree-backend/ac_overlay.json -run 'TestRv5FocusProbe\|TestImplementationModeFocusScoped' -v -count=1 -> "focus=pay: leaves=[pay1 pay2 srch1 srch2]"; "focus=no-such-node: focus present=false leaves=4 keys error=<nil>"; TestImplementationModeFocusScoped PASS |

## Ordered recommendations

1. Decide the ownership rule in writing before any code: an owner set on a node covers its whole subtree and may edit inside it, but may not delete the delegated root, validate, or re-assign the root. Record it in PRODUCT.md and in the BUG-020 record (TREEX-12).
2. Build the server half first in one slice: requireNodeWriter (TREEX-01), GET /api/v1/roadmap/{nid} with a subtree option (TREEX-06), and a true focus scope in the agent briefing (TREEX-05). All three are small and each has a real-database or unit test that fails today.
3. Bring the MCP tools and the CLI level with the web in the same release: owner_id and acceptance on roadmap_update, roadmap_get, a members read, and matching CLI flags (TREEX-04, TREEX-06).
4. Build the web half as three small widgets on the existing page: the subtree route with a breadcrumb (TREEX-02), the Mine filter (TREEX-03 step one), and a Delegate menu entry with inherited owner labels (TREEX-08). Collapse, leaf-based progress and the read-only detail sheet follow (TREEX-09, TREEX-10, TREEX-11).
5. Hide write actions from readers and stop the AI draft for a caller who cannot confirm it (TREEX-07); do this with the per-node canEdit flag once TREEX-01 is in.
6. Mark record R2-04 in docs/issues/2026-10-18-review-4-open-findings.md as closed: the code now gates expand confirm on requirePlanWriter (server/internal/store/roadmap_plan.go:133) and the wave-4 changelog says so, but the row still reads as open.
