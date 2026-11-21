# 40 · Remediation plan

Tree: `main` @ `4d186da` plus the uncommitted working tree. Ordered by impact
over effort: start at row 0.1 and work down. IDs point at the
[findings register](./01-findings-register.md); the quote, failure scenario
and skeptic verdict for each are in the lens files.

Rules for every row: a failing test first when the row names one, real
PostgreSQL for anything database-shaped
(`./scripts/dev.sh -C server go test ...`), one commit per row, and a
changelog entry per wave. Diff sizes are the reviewers' estimates, checked by
a skeptic where the finding was High. `dir` means the reviewer gave a
direction and a first check, not a patch.

Waves 0 and 1 are small and should go first. Wave 2 is the task tree and is
the largest. Waves 3, 4 and 5 do not depend on wave 2 and can run beside it,
with one exception: rows 4.1 and 4.2 touch the files wave 2 has to edit, so
doing them first makes wave 2 smaller.

## Wave 0 · Make the tip green and write the concept down

| Row | IDs | File | Change | Size |
| --- | --- | --- | --- | --- |
| 0.1 | `IDX-02`, `COMP-01` | `docs/architecture/backend.md`, `server/internal/api/testdata/roadmap_expand_confirm.json`, `client/lib/api/requests.dart` | Name `erasure.go`, `conversation_retention.go` and `me_delete.go` in the two lists; add `completionReason` to each fixture object; rewrite `requests.dart:95-101` with null-aware elements so the analyzer count drops under the baseline. One commit on a clean checkout of `origin/main` | under 20 lines |
| 0.2 | `IDX-04` | local checkout | Commit or stash the uncommitted slice, then fast-forward to `origin/main` (two documentation commits) | owner action |
| 0.3 | `IDX-01` | `docs/plans/` | Open a dated plan and an In flight board row for the install and password-reset slice | about 40 lines |
| 0.4 | `PARA-01`, `PARA-08`, `PARA-10`, `STRC-01`, `FE-06`, `IDX-09`, `COMP-04`, `COMP-06` | the uncommitted slice | Close the slice before its checkpoint commit: the terminal check at `cli_auth.go:55` (1 line); the escaping helper and four sweep rows in `apiclient`; `s.now()` in `auth_reset_watch.go:25`; the mail-copy test; `test/routes_test.dart`; the two gates `check_first_run_copy.py` and `check_supply_chain.py`; purge and list `reset_watches`; answer `accepted` for a dropped send | about 80 lines |
| 0.5 | `TREEB-11`, `TREEX-12`, `IDX-07` | `PRODUCT.md`, `docs/architecture/roadmap-tree.md`, `docs/issues/` | Write the task-tree concept into the product definition, add a current-state architecture page, and file a dated issue with the slice order of wave 2. [41](./41-task-tree-target-design.md) is the draft | about 120 lines of prose |
| 0.6 | `COMP-03` and the four decisions in [41](./41-task-tree-target-design.md) | owner | Decide: per person or per role; default view or wall; may a delegate re-delegate; may an agent token change an owner | owner decision |
| 0.7 | `COMP-07` | `.git/info/untracked-manifest.tsv`, `docs/HANDOFF.md:111` | Regenerate the untracked-file manifest and correct the folder names | 2 lines plus one command |

**Check:** `python3 -I scripts/tests/check_docs_truth_rv4.py` exits 0; the
`ci` run on the pushed tip is green in every job;
`(cd server && go build ./... && go vet ./... && go test ./...)` and
`(cd client && flutter analyze --no-fatal-infos && flutter test)` pass in the
working tree, the Go suite also with `DATABASE_URL` set.

## Wave 1 · Stop losing and misplacing data

| Row | IDs | File | Change | Size |
| --- | --- | --- | --- | --- |
| 1.1 | `PARA-02` | `server/internal/api/ideas.go`, `server/internal/store/ideas.go`, `server/internal/apiclient/crud.go` | Make `content` a pointer and write `content = COALESCE($3, content)`; send it only when given. Red test: a title-only update keeps the body | about 25 lines |
| 1.2 | `FEAT-01`, `PARB-02` | `server/internal/store/roadmap_plan.go:197-245`, `server/internal/api/roadmap_draft.go` | Make the draft route read-only. Create and link the project only in the confirm call, for the PRD author or a workspace admin, and fail when the link update touches no row. Correct `draft_tools.go:18` | about 40 lines plus a 60-line test |
| 1.3 | `PARB-01` | `server/internal/api/roadmap_draft.go:50` | Decide the rule on the server: refuse a PRD that is not approved with 409 and a code, or remove the web condition | about 15 lines |
| 1.4 | `FEAT-02` | `server/internal/api/roadmap_ai.go`, `agent_scope.go` | Refuse an agent token on build confirm when the live roadmap is not empty (the pattern at `agent_routes.go:222`). Decide whether an agent may write `completionReason` | about 30 lines plus two cases |
| 1.5 | `FEAT-03`, `TREEB-10` | `server/internal/store/roadmap_plan.go:58` | Refuse a replace while any node has evidence or is done or validated unless the caller passes an explicit force flag; the web dialog states how many evidence rows will be lost. Keeping node ids across a rebuild is the fuller fix | guard about 30 lines; merge about 150 |
| 1.6 | `PARC-05` | `server/internal/api/roadmap.go:291` | Treat an absent `parentId` as "leave the parent alone", so a position-only move from CLI or MCP does not make the node a root | about 10 lines |
| 1.7 | `FE-01` | `client/lib/features/prd/prd_page.dart`, `coach_panel.dart` | Re-read the conversation's messages when a turn ends and show the sent message at once. Red test: two turns, four bubbles | about 25 lines plus a test |
| 1.8 | `FEAT-09`, `FE-09` | `client/lib/features/prd/prd_page.dart:140` | Show the drafted tree before the confirm call | about 60 lines |

**Check:** the three real-database red tests from
[10](./10-feature-product.md) (a non-author member builds; an agent token
confirms over a populated roadmap; a rebuild over a node with evidence) pass;
`flutter test test/coach_panel_test.dart` passes with the two-turn case.

## Wave 2 · The task tree for roles

Each row ships on REST, the Go client, MCP, CLI and web in one change. Do not
start row 2.2 before rows 1.4 and 1.5 are in. Design:
[41](./41-task-tree-target-design.md).

| Row | IDs | File | Change | Size |
| --- | --- | --- | --- | --- |
| 2.1 | `TREEB-03`, `TREEX-06`, `TREEX-05`, `TREEB-06` | `server/internal/store/roadmap.go`, `server/internal/api/roadmap.go`, `server/internal/agentcontext/agentcontext.go` | Subtree read: `?root={nid}` on the roadmap list with an `ancestors` array, reusing the recursive query in `roadmap_move.go:42-47`; `GET /roadmap/{nid}`; agent context restricted to the focus subtree, 404 for an unknown focus | about 250 lines with tests |
| 2.2 | `TREEB-02`, `TREEX-01`, `TREEB-07`, `STRS-05`, `TREEX-07`, `FE-05` | new `server/internal/store/authz.go`, one migration | `insideout._can_write_node(node, user)` used by `requireNodeWriter` and by the insert, update and delete policies on `roadmap_nodes`; node views return `canWrite`; the web hides write items without it. Red test: the delegate writes inside, and is refused on the root's parent, a sibling and another subtree | about 70 lines of store, 1 migration, 150 lines of tests |
| 2.3 | `PARC-01`, `TREEX-04`, `PARC-02` | `server/internal/apiclient/roadmap.go:82`, `cli_roadmap.go`, `product_tools.go` | A `RoadmapPatch` struct carrying owner and acceptance; `--owner` and `--acceptance`; `owner_id` and `acceptance` on `roadmap_update`; a `members` verb and tool. Subject to decision 0.6 | about 180 lines |
| 2.4 | `TREEX-02`, `FE-03`, `TREEX-08`, `TREEX-11` | `client/lib/router.dart`, `roadmap_page.dart`, `roadmap_tree.dart`, `roadmap_node_menu.dart` | Route `/projects/:id/roadmap/:nodeId` with a breadcrumb; tapping a card opens it; menu items "Open as tree" and "Delegate to"; a read-only node detail with description and evidence | about 200 lines plus widget tests |
| 2.5 | `TREEB-04`, `TREEX-03` | store, api, `dashboard_page.dart`, `roadmap_page.dart` | `GET /workspaces/{id}/roadmap/mine`; a "Mine" toggle; the caller's nodes on the signed-in landing; `roadmap mine` and `roadmap_mine` | about 300 lines with tests |
| 2.6 | `TREEB-01`, `TREEB-09`, `TREEX-08` | one migration, store, api | Append-only `roadmap_node_assignments`; `POST /roadmap/{nid}/delegate {toUserId, note}`; node rows in `project_updates` so the delegator sees delegation, hand-back and completion in the feed | 1 migration, about 220 lines |
| 2.7 | `TREEB-08`, `TREEX-09`, `TREEX-10`, `FE-11` | `server/internal/roadmaptime`, `roadmap_canvas.dart`, `roadmap_node_card.dart` | Server roll-up per node (leaves done, validated, total, earliest deadline); progress accepts a root; the canvas collapses a branch; a validated child counts as finished | about 200 lines |
| 2.8 | `TREEB-05` | `server/internal/auth/jwt.go`, `agent_scope.go` | An agent token may carry a root node and is refused outside it | about 80 lines |
| 2.9 | `TREEB-12`, `FEAT-05` | `server/internal/store/memberships.go` | Clear or reassign node ownership when a member leaves; decide whether an in-progress leaf must have an owner | about 60 lines |

**Check:** the four walks in [41](./41-task-tree-target-design.md) pass on a
local server: the owner delegates from the web and from the CLI; the delegate
lands on their tree and edits inside it; a manager opens the branch by link
and sees its roll-up; an agent asks for its subtree and gets only that.

## Wave 3 · Keep the four surfaces in step

| Row | IDs | File | Change | Size |
| --- | --- | --- | --- | --- |
| 3.1 | `PARC-03`, `COMP-02` | `server/cmd/insideout/contract_real_api_test.go`, `client/test` | A test that walks the JSON tags of every API request struct and fails unless each field is reachable on each surface or listed as a deliberate omission; one more response fixture with owner, deadline, acceptance and completion reason set, read strictly by the Dart model | about 200 lines |
| 3.2 | `STRS-06`, `STRS-02`, `PARC-07`, `PARC-09`, `PARA-09` | `server/internal/apiclient`, `mcpserver`, `cmd/insideout` | One request struct per verb, shared by the Go client, the CLI flags and the MCP schema; state once what absent and empty mean | dir; start with roadmap update |
| 3.3 | `FE-02`, `PARB-03`, `PARB-07` | `server/internal/api/prds.go`, `projects.go`, `workspace_page.dart` | `GET /workspaces/{id}/prds`; `prdId` on the project view; accept the cursor the updates feed already emits | about 250 lines |
| 3.4 | `PARB-04`, `PARB-06`, `PARB-11`, `PARC-04`, `PARA-03`, `FE-07`, `FE-04`, `PARB-08` | `apiclient`, `cli_crud.go`, `product_tools.go`, `profile_page.dart` | The missing verbs: project read, update, delete and updates; members; workspace edit; account deletion on web and CLI; validation on CLI; whole-PRD export on CLI; the guide on web; a project owner control | about 500 lines |
| 3.5 | `PARB-04`, `PARA-06`, `PARC-11`, `PARB-09`, `PARA-11`, `PARB-12` | `README.md:35`, `SKILL.md:65,107`, `docs/usage/cli-mcp-and-guidance.md` | Until 3.4 lands, state the parity that exists; fix the two commands that exit 2 | about 30 lines |
| 3.6 | `PARC-08`, `PARC-10`, `PARB-10`, `PARA-12`, `STRS-11` | `server/internal/api` | One store-error-to-status helper with one code style; one field-bounds helper | dir |
| 3.7 | `PARB-05`, `PARA-04`, `PARA-07`, `PARA-05`, `PARC-06`, `PARC-12` | various | The remaining per-surface differences, one commit each | small |

**Check:** the test from 3.1 passes with an empty accidental-gap list.

## Wave 4 · Structure

| Row | IDs | File | Change | Size |
| --- | --- | --- | --- | --- |
| 4.1 | `COMP-05`, `STRS-08`, `STRC-06` | `CLAUDE.md:55`, `AGENTS.md:71`, `server/internal/linebudget`, `client/test/line_budget_test.dart` | Replace the line rule with a responsibility rule for source code: one threshold shared by the Go and Dart guards, a function-length guard, and no line rule for workflows or prose | about 60 lines |
| 4.2 | `STRS-01`, `STRS-02` | `server/internal/mcpserver/product_tools.go:20` | One registration helper and per-resource tool files in place of the 315-line function; split the 219-line coach turn into open, run and close | moves, no behaviour change |
| 4.3 | `STRS-03`, `STRC-02`, `STRC-03`, `STRC-04`, `STRC-11` | `server/cmd/insideout/cli_setup*.go`, `scripts/setup.sh`, `scripts/keys.py` | Move the setup engine out of package `main` into `internal/setup`; make the Go wizard the only `.env` writer and the only placeholder rule; reduce `scripts/setup.sh` to a caller of the binary | about -200 and +100 lines |
| 4.4 | `STRS-09`, `STRS-04`, `STRS-10` | `server/internal/agent`, `store/agent_messages.go`, `github/github.go` | A leaf package for the provider clients; split the files that hold three or four responsibilities | moves |
| 4.5 | `STRC-05`, `STRC-08`, `FE-08`, `FE-10` | `client/lib/features/project`, `client/lib/api` | A controller per page between the State class and the API client; typed models for the nine verbs that return maps | dir; start with the roadmap page |
| 4.6 | `STRS-07`, `STRC-07` | `rv4_review_*_test.go`, `check_*_rv4.py` | Rename tests and gates after the behaviour they protect | renames |
| 4.7 | `STRS-12`, `STRC-09`, `STRC-10`, `STRC-12` | various | Remove three unreferenced functions; tidy the scripts and test folders | small |

**Check:** both guards pass with the new rule; `go vet ./...` and the full
suites stay green after each move.

## Wave 5 · The documentation index

| Row | IDs | File | Change | Size |
| --- | --- | --- | --- | --- |
| 5.1 | `IDX-05`, `IDX-11`, `IDX-12` | new `.marksman.toml`, 298 links, one table cell | `title_from_heading = false`; write sibling and child links with a leading `./`; put the rule in the review report template; separate the two character classes of the regular expression at `docs/review/2026-10-18-multirole-4/01-findings-register.md:123` (backticks alone do not cure it: inside a table Marksman still splits the cell at the escaped pipe, measured on this report's own register) | 3 lines plus a mechanical rewrite |
| 5.2 | `IDX-03` | new `scripts/tests/check_doc_graph.py` | Commit the reachability check: fail on a markdown file unreachable from the four routers, on an index that misses a direct child, and on a bare link that matches more than one file | under 150 lines |
| 5.3 | `IDX-01`, `IDX-10`, `IDX-06` | `docs/plans/README.md`, `docs/issues/README.md`, `docs/changelogs/README.md` | Keep only open rows on the board and move finished rows to month files; give the issue index an Open table; drop the changelog flat list | moves |
| 5.4 | `IDX-08` | `docs/index.md` | Add `SKILL.md`, the second skill, `hosts/` and the root policy files; remove `docs/PLAN.md` and `docs/INIT.md` | about 15 lines |
| 5.5 | `IDX-09` | `scripts/tests/check_doc_links.py` | Check heading fragments in undated documents and include untracked files | about 40 lines |
| 5.6 | `FEAT-04`, `FEAT-11`, `FEAT-07`, `FEAT-08`, `FEAT-10`, `FEAT-12`, `FE-12` | `docs/issues/`, `docs/review/2026-10-10-multirole-3/50-promise-register.md` | File the per-Project roles record; annotate the seven promise rows stamped implemented that are not; redact the hostname in `docs/architecture/frontend.md:24` | about 100 lines |

**Check:** Marksman reports 0 diagnostics; `check_doc_graph.py` and
`check_doc_links.py` exit 0 in CI.

## Not in scope yet

- The full per-Project role model (Decision Owner, Driver, Contributor,
  Reviewer, Viewer). File the record in row 5.6 and decide 0.6 first; the
  node-owner rule in row 2.2 does not need it.
- The nine parked product slices in
  [the promise-gap record](../../issues/2026-10-11-product-promise-gaps.md)
  (Product Branch and Merge, feedback invitations, share links, lenses,
  dependency edges, the Focus pointer, autosave). The Focus pointer is not
  the task tree and should not be built in its place.
- Email or push notification of delegation. The project update feed in row
  2.6 comes first.
- Hiding the rest of the project from a delegate. Row 0.6 recommends a
  default view, not a wall.
- Sub-packages for `server/internal/api` and `store`. The package graph is
  acyclic and the files are small; split functions and responsibilities
  (wave 4) before moving packages.
- Rewriting dated records to cure Marksman warnings one by one. Row 5.1
  cures the cause.
