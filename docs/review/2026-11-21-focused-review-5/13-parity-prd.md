# 13 · Surface parity: project and PRD

**Lens:** REST, web, MCP and CLI for projects, PRDs, coach, versions, proposals
**Score:** **5.6 / 10**
**Bottom line:** The PRD verbs (read, edit, readiness, commit, versions, revisions, snapshot, audience view) line up well across REST, web, MCP and CLI, and the agent-token wall is enforced on the server. The project half of the domain does not line up: beyond list and create, projects exist only on the web. Four High gaps were found by reading the code. The admin review gate before Build is enforced only by a web button. The Build "draft" call, documented on three surfaces as writing nothing, creates a project owned by whoever called it. No surface can find a PRD the caller did not create. SKILL.md promises 1:1 parity and CLI deletes that do not exist.

Read at `main` @ `4d186da` together with the uncommitted working tree. A skeptic re-checked every High in this file; the verdicts are in the table near the end and in the [findings register](./01-findings-register.md).

---

## Scope read

I read the route registrations and handlers in `server/internal/api` (`projects.go`, `project_updates.go`, `prds.go`, `prd_commits.go`, `prd_view.go`, `prd_readiness.go`, `conversations.go`, `agent_routes.go`, `agent_scope.go`, `roadmap_ai.go`, `roadmap_draft.go`, `guide.go`), the store functions behind them, the whole MCP registry (`server/internal/mcpserver`), the shared Go client (`server/internal/apiclient`), the CLI dispatch and verbs (`server/cmd/insideout/cli*.go`, `help.go`), the Flutter client (`client/lib/api`) with its callers in `features/prd`, `features/project` and `features/workspace`, and the documents `SKILL.md`, `docs/usage/cli-mcp-and-guidance.md`, `docs/architecture/agent-token-scope.md` and `docs/architecture/cli-mcp-and-client-config.md`. I built the CLI to a temporary directory and ran six commands offline to check usage behaviour. No database was touched, so findings about store behaviour are from reading the code.

## What is good

- `server/internal/api/agent_scope.go:39` is one fail-closed table for agent tokens. Commit, proposal decisions, deletes and admin routes are refused on the server, so the rule is the same for curl, the CLI and MCP.
- `server/internal/apiclient/prd.go` and `crud.go` are the one client behind both the CLI and MCP, and reads return the API's raw JSON. The two agent surfaces cannot rename a field.
- `PATCH /prds/{id}` is a true partial update (`server/internal/api/prds.go:86-93`) and all four surfaces send the same body.
- The PRD verbs (read, edit, readiness, commit, versions, revisions, snapshot, audience view) exist on all four surfaces with matching field names.
- Deliberate absences are written down: the coach is web-only (`SKILL.md:128`), human-only routes are listed in `docs/architecture/agent-token-scope.md:67-76`.

## Findings, worst first

| Id | Severity | Where | Deciding line | Fix |
|---|---|---|---|---|
| PARB-01 | High | `server/internal/api/roadmap_draft.go:50` | "It does not gate on PRD status or a commit", while the web has `onPressed: status == 'approved' ? onBuild : null` (`prd_desk_actions.dart:67`) | Check the status in `loadBuildablePrd`, or drop the web condition. |
| PARB-02 | High | `server/internal/api/roadmap_draft.go:64` | `s.store.EnsureProjectForPrd(...)` runs in the draft call; `roadmap_plan.go:233` inserts a project owned by the caller; `draft_tools.go:18` says "Writes nothing until build_confirm." | Create the project only at confirm, for the author or an admin. |
| PARB-03 | High | `server/internal/api/projects.go:26` | `projectView` has no PRD id; `store/ideas.go:75` lists ideas `AND author_id = $2`; no route lists PRDs | Add `prdId` to the project payload and `GET /workspaces/{id}/prds` on all surfaces. |
| PARB-04 | High | `SKILL.md:107`, `:119-120` | "Every verb exists 1:1 as CLI command ↔ MCP tool ↔ API route" and deletes "web, or the CLI with your login"; the Go client has no project read, update, delete or add-update method | Add the client methods and CLI verbs, an agent read of the feed, and correct the page. |
| PARB-05 | Medium | `client/lib/api/client_project.dart:94` | `bool apply = true,` against `fs.Bool("apply", false, ...)` in `cli_idea.go:38` | One default: apply on accept. |
| PARB-06 | Medium | `server/cmd/insideout/cli_version.go:85` | `c.PrdExportAudience(fs.Args()[0], *audience)` is the CLI's only export | Add `insideout export` over the existing `PrdExport`. |
| PARB-07 | Medium | `server/internal/api/projects.go:185` | `ListProjectUpdates(..., store.ProjectUpdatesPageSize, nil)`; `nextCursor` has no consumer | Accept `before` and add load-more. |
| PARB-08 | Medium | `client/lib/api/requests.dart:71` | `'ownerId': ownerId,` always sent; no owner control in any client; PATCH requires title and status | Owner picker; partial PATCH. |
| PARB-09 | Medium | `SKILL.md:65`, `:90` | `projects create <ws-id> --title T` exits 2 when run | Flags-first examples and a CLI to MCP name table. |
| PARB-10 | Medium | `server/internal/api/agent_routes.go:116` | `content += " (node " + req.NodeID + ")"` with no check; six sibling text fields uncapped | Validate the node; cap the fields. |
| PARB-11 | Medium | `client/lib/api/client_project.dart:83` | The web has bind and sync but no call to `GET /projects/{pid}/guide` | Add the download to the GitHub card. |
| PARB-12 | Low | `server/cmd/insideout/cli_idea.go:16` | `usage: insideout idea <create|convert> …` hides three subcommands; names are not 1:1; four verbs missing from the usage table | Fix usage and docs; implement the rv4-ARCH-07 test. |

PARB-01 and PARB-02 share one cause. When Build was split into draft and confirm, the status check and the read-only promise were both left out, and the code comment says so. The consequence is that the only review gate in the running authority model is a disabled button, and a call advertised as safe changes ownership of a new project.

PARB-03 affects every surface, including the web. A workspace admin who must approve a PRD has no way to reach one except a link sent by its author.

Smaller differences, not filed separately: `GET /prds/{id}/view` requires `audience` while the CLI and MCP default it to `decision`; the 404 text is "PRD not found" in `prds.go` and "prd not found" in `roadmap_draft.go:55`; the Build routes name a PRD id `{pid}`; snapshot is author-only (`store/prd_revisions.go:67`) while edit and commit admit an admin; `docs/usage/cli-mcp-and-guidance.md:167` names a CLI verb `context` that is `agent-context`; `docs/architecture/agent-token-scope.md:57` says 36 of 72 routes and the table now holds 74.

## Lead verification

| Lead | Result | Deciding line |
|---|---|---|
| 5 | Confirmed, and it extends to projects | No project-owner parameter in `product_tools.go`, `cli_crud.go` or `apiclient`; the web resends the loaded owner (`project_page.dart:271`). |
| 6 | Confirmed for the authority model | `PRODUCT.md:432-435`: the author submits and a workspace admin approves or rejects. |
| 7 | Not re-run in full | `go test ./internal/mcpserver ./internal/guide ./internal/agentcontext` passes. |
| 13 | Partly refuted | 87 literal `mux.HandleFunc("METHOD /...")` registrations in non-test files plus the GitHub webhook, not about 107; 52 MCP tools confirmed; `cliVerbs` has 41 rows; `agentRoutePolicy` has 74 entries (36 agent-allowed, 38 human-only). |
| 14 | Checked | None of PARB-01 to PARB-11 is in `docs/issues/2026-10-18-review-4-open-findings.md`; PARB-12 cites rv4-ARCH-07; the unfenced CLI output is already R2-09. |

## Recommendations

1. Move rules that only a widget enforces onto the server, each with an API test.
2. Make the Build draft read-only and create the project at confirm.
3. Add PRD discovery (list and project link) to all four surfaces in one change.
4. Close the project gap on the CLI and give agents a read of the updates feed.
5. Correct `SKILL.md` and the usage doc in the same change as the code.
6. Turn the capability matrix in this report into the rv4-ARCH-07 test, with a reason string for every deliberate blank.

## What this role would do first

Fix the Build path in `server/internal/api/roadmap_draft.go`: refuse a PRD that is not approved (or remove the web condition if that is the product decision), and stop the draft call from inserting a project. One real-database test should prove that a non-author member's `build` on a draft PRD writes nothing and is refused. This is the smallest change that removes two High findings and restores the one review gate the product currently has.

Not checked: nothing was run against PostgreSQL, so the orphan-project repetition in PARB-02 rests on reading the row-level-security policy; the roadmap, auth and workspace domains were left to their own reviewers.

## Surface matrix

| Capability | REST | Web | MCP | CLI | Note |
| --- | --- | --- | --- | --- | --- |
| List projects in a workspace | GET /workspaces/{id}/projects | listProjects (workspace_page board) | projects | projects <ws-id> | Consistent. |
| Create project | POST /workspaces/{id}/projects | createProject (project_dialogs) | project_create | projects create --title T [--desc D] <ws-id> | Consistent. SKILL.md:65 shows the flags after the id, which exits 2 (PARB-09). |
| Read one project with its updates feed | GET /projects/{pid} | getProject (project_page) | none | none | Agent refusal is listed in agent-token-scope.md as a read no tool calls, not as a reasoned ban. CLI absence is an accident (PARB-04). |
| Update project title, description, status | PATCH /projects/{pid} (full replace) | updateProject (project_page edit dialog) | none | none | MCP absence deliberate (project edits are human-only). CLI absence accidental (PARB-04). PATCH is not partial (PARB-08). |
| Set or clear project owner | PATCH /projects/{pid} ownerId | none (resends the loaded owner) | none | none | Accident: no client can perform the decided handover rule (PARB-08). |
| Delete project | DELETE /projects/{pid} | deleteProject (project_page) | none | none | MCP deliberate (every delete is human-only). CLI absent although SKILL.md:119-120 says the CLI can (PARB-04). |
| Link a GitHub repository | PUT /projects/{pid}/repo | setRepo (project_github_card) | repo_set | repo set <project-id> <url> | Consistent; body field repoUrl everywhere. |
| GitHub sync now | POST /projects/{pid}/sync-github | syncGithub (project_github_card) | sync | sync <project-id> | Consistent. |
| Fetch insideout.yaml guide | GET /projects/{pid}/guide (text/yaml) | none | guide | guide [--out FILE] <project-id> | Web absence is an accident (PARB-11). |
| Post a progress, blocker or note update | POST /projects/{pid}/updates | addUpdate (project_page) | none | none | MCP deliberate: checkpoint is the agent's form. CLI absence accidental (PARB-04). |
| Edit or delete an update | PATCH and DELETE /updates/{uid} | editUpdate, removeUpdate (project_updates_feed) | none | none | MCP deliberate. CLI absent although SKILL.md says deletes run from the CLI (PARB-04). |
| Older pages of the updates feed | none (nextCursor is emitted, no route takes it) | none | none | none | Accident on every surface (PARB-07). |
| Agent proposal (propose) | POST /agent/propose | none | propose (kind required; items as objects) | propose --kind K [--detail D] [--item Title@Parent] <project-id> <summary> | Web absence deliberate (agent vocabulary). CLI kind defaults to structure; a CLI item title cannot contain @; MCP silently drops an item with an empty title where the API answers 400. |
| Decide a proposal | POST /agent/proposals/{uid}/decision | decideProposal, apply defaults true (project_updates_feed) | proposal_decide, apply defaults false; withheld under a JWT login | idea proposal-decide --accept\|--reject [--apply] [--reason R] <update-id> | MCP withholding deliberate (agent_scope.go:47). Default for apply differs (PARB-05). The CLI cannot list update ids. |
| Agent checkpoint | POST /agent/checkpoint | none (shown in feed) | checkpoint | checkpoint [--node id] <project-id> <summary> | Web absence deliberate. nodeId is not validated (PARB-10). |
| Agent context | GET /agent/context?project_id=&mode=&focus= | none | agent_context | agent-context [--mode M] [--focus node] <project-id> | Web absence deliberate. Query key is snake_case where bodies are camelCase; an unknown mode is silently treated as implementation. |
| Read a PRD | GET /prds/{id} | getPrd (prd_page) | prd | prd <prd-id> | Consistent. |
| Find a PRD (list, or project to PRD) | none | only through the caller's own idea card | none (idea_list shows own ideas) | none (ideas shows own ideas) | Accident on every surface (PARB-03). |
| Edit PRD title and sections | PATCH /prds/{id} (partial) | updatePrd (prd_desk) | prd_update (sections_json string) | prd update [--title T] [--section k=v]... <prd-id> | Same rule and limits; only the argument encoding differs. |
| PRD status transition | POST /prds/{id}/status | updatePrdStatus (prd_page, prd_desk_actions) | prd_update status (agent token: draft or reviewing only) | prd update --status S <prd-id> | Agent limit deliberate (agentSubmit). On CLI and MCP the edit and the status are two calls; a refused status reports failure after the edit was saved. Valid PRD statuses are not listed in the usage doc. |
| Coach conversation: start, resume, history, streaming turn | GET and POST /prds/{id}/conversation, GET /conversations/{id}, GET and POST /conversations/{id}/messages (SSE) | conversationForPrd, startPrdConversation, listMessages, SSE send (coach_panel) | none | none | Deliberate: SKILL.md:128 sends users to the web for the coached experience. idea_convert still opens a conversation neither surface can continue. |
| Stage proposal confirm and decline | POST /conversations/{id}/stage and /stage/decline | confirmConversationStage, declineConversationStage (coach_stage_proposal) | none | none | Deliberate, human-only. |
| Readiness (form a version now) | GET /prds/{id}/readiness | prdReadiness (versions_page) | readiness | readiness <prd-id> | Consistent. |
| Commit a version | POST /prds/{id}/commit | commitPrd (versions_commit_dialog) | commit (withheld under a JWT login; 403 for an agent token) | commit --name N --audience A [--summary S] [--unresolved U]... [--note D] <prd-id> | Deliberate. Flags are short forms of primaryAudience, changeSummary, decisionNote; summary and note are uncapped (PARB-10). |
| List versions | GET /prds/{id}/commits | listPrdCommits (versions_page) | versions | versions <prd-id> | Consistent. |
| List revisions | GET /prds/{id}/revisions | listRevisions (revisions_page) | revisions | revisions <prd-id> | Consistent. |
| Snapshot a revision | POST /prds/{id}/revisions | createRevision (snapshot_flow) | snapshot | snapshot [--note N] <prd-id> | Consistent across surfaces. Author-only, while edit and commit also admit a workspace admin. |
| Export | GET /prds/{id}/export?format=markdown\|print[&audience=] | exportPrd (export_page) | prd_export (format, audience) | view --export FILE (markdown, audience always applied) | CLI partial by accident (PARB-06). format is required on REST and defaulted by the Go client. |
| Audience view | GET /prds/{id}/view?audience= (required) | prdView (audience_view_page) | view (defaults to decision) | view [--audience A] <prd-id> (defaults to decision) | The default exists only in the clients. |
| Build a roadmap from a PRD (draft, confirm) | POST /prds/{pid}/build and /build/confirm | buildFromPrd, confirmBuildFromPrd; button enabled only when approved | build, build_confirm | build, build-confirm | The approved gate is web-only (PARB-01); the draft call creates a project (PARB-02); both verbs are missing from the usage doc's table. |

## Skeptic verdicts on this lens's Highs

| ID | Verdict | Reason | What the skeptic ran |
| --- | --- | --- | --- |
| PARB-01 | CONFIRMED | Reproduced: a workspace admin set the PRD to rejected, then an agent token called build (200) and build/confirm (201, nodeCount 3) on it. The server has never carried the check (the comment in loadBuildablePrd says so, and git history shows no status gate on the roadmap routes), while the web disables the button unless status == 'approved' (prd_desk_actions.dart:67). So the same core flow follows a different rule per surface, and the admin review gate that agent-token-scope.md protects ('an agent holding an admin's token cannot approve its own submission') gates nothing on the server. One limit on the finding: no usage document promises that Build needs approval; the rule lives only in the web client and in the EnsureProjectForPrd comment ('an approved/refined PRD becomes a project'), so the owner may decide that any status may build. Not in an OPEN record. Separate root cause from FEAT-01 and PARB-02, though with FEAT-01 any member can do this on anyone's PRD. | go test -overlay /tmp/insideout-rv5/build-path/overlay_api.json ./internal/api/ -run TestRV5BuildPathAPI_AgentAndStatus -v -count=1 -> 'human admin -> rejected: 200', 'agent build draft on rejected PRD: status=200', 'agent build/confirm on rejected PRD: status=201 body={"nodeCount":3,...}' |
| PARB-02 | CONFIRMED | Reproduced through the real handler chain with the repository's stub planner: POST /prds/{id}/build as a plain non-author member returned 200 twice with two different projectIds, the workspace project count went 1 -> 2 -> 3, and the PRD still had no projectId. The author's own draft with no confirm took the count to 4. handleBuildFromPrd calls loadBuildablePrd, which calls EnsureProjectForPrd before the planner runs. The contradicted claims are in the tree: draft_tools.go:18 'Writes nothing until build_confirm.' and requests.dart:152 'Nothing is written'. Same root cause as FEAT-01; this framing adds the documented-contract breach. No OPEN record covers it. | go test -overlay /tmp/insideout-rv5/build-path/overlay_api.json ./internal/api/ -run TestRV5BuildPathAPI_DraftWrites -v -count=1 -> 'member draft #2: status=200 body={"projectId":"a1020086-..."', 'projects after member draft #2: 3', 'prd after member drafts: ... projectId-present=false', 'projects after author draft (no confirm): 4' |
| PARB-03 | CONFIRMED | Confirmed by a real-database probe. A member converted an idea and moved the PRD to reviewing; the workspace admin then read the list and detail routes and the PRD id appeared in none of them. Covered: workspaces, workspace detail, ideas, projects, members, me, and the project, roadmap, progress and guide of the workspace's other project. Not exercised: a project built from that PRD, since the probe built none (projectView has no prdId field), and agent context, which answered 400 on my query form; by reading, agent_routes.go:81-83 passes the PRD's title and sections, not its id. Both guessed list routes answer 404. The same admin reads and approves the PRD once given the id, so only discovery is missing. PRODUCT.md states that a workspace admin approves or rejects, and approval gates the build, so a promised core step cannot be reached on web, CLI or MCP without an out-of-band link. Not an open record under docs/issues. | go test ./internal/api/ -overlay /tmp/insideout-rv5/client-and-cli/overlay.json -run TestRV5ProbeAdminFindsPrd -v -> every list route `names PRD id: false`; `admin GET /api/v1/prds -> 404`; `admin POST /prds/<known id>/status approved -> 200` |
| PARB-04 | CONFIRMED | Both sentences are false as written. The Go client has two DELETE calls only (ideas, roadmap nodes) and no method for GET, PATCH or DELETE on a project, for project updates, for workspace PATCH or DELETE, or for members; the built CLI answers `unknown command` to each. The API has all of these routes and the web uses them. Reading one project, reading its updates feed and posting a progress or blocker note are the product's project-tracking flow, so a human at the CLI cannot do what the web can for a core flow. Agent context carries no updates feed, so an agent cannot read back its own checkpoint or a proposal decision. Two limits on the claim: the agent-token refusal on these routes is deliberate (agent_scope.go humanOnly), and the verb table in docs/usage/cli-mcp-and-guidance.md is accurate; the overclaim is SKILL.md's alone. Not an open record under docs/issues. | HOME=/tmp/insideout-rv5/home-client-and-cli /tmp/insideout-rv5/insideout project delete x -> `insideout: unknown command "project delete x"` exit=2 (same for `project x`, `projects delete x`, `workspaces delete x`, `update add x`, `members x`); grep MethodDelete server/internal/apiclient -> crud.go:34 (ideas) and roadmap.go:127 only |

## Ordered recommendations

1. Move every rule that only a web widget enforces onto the server, starting with the approved-before-Build gate, and add an API test for each so the CLI and MCP meet the same answer.
2. Make the Build draft route read-only and create the project only at confirm, restricted to the PRD's author or a workspace admin.
3. Add a member-visible PRD list and a prdId on the project payload, then project them to all four surfaces in the same change.
4. Close the project gap on the CLI (read one project with its feed, update, delete, add an update) and give agents a read of the feed so they can see the decision on their own proposals.
5. Correct SKILL.md and docs/usage/cli-mcp-and-guidance.md in the same change: flags-first examples, the real list of web-only actions, the four planner verbs, the CLI verb name `agent-context` (line 167 says `context`), and the route counts in docs/architecture/agent-token-scope.md:57 (36 of 72 is now 36 of 74).
6. Implement the rv4-ARCH-07 parity test as a table of capability, route, apiclient method, CLI verb and MCP tool, with an explicit reason string for every deliberate blank; the matrix in this report is that table's first draft.
7. Unify small differences while touching these files: one default for proposal apply, a server-side default audience for GET /prds/{id}/view (today only CLI and MCP default to decision), one spelling of the 404 text ("PRD not found" in prds.go, "prd not found" in roadmap_draft.go:55), and the same author-or-admin rule for snapshot as for edit and commit (server/internal/store/prd_revisions.go:67).
