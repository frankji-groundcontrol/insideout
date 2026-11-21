# 14 · Surface parity: roadmap and delivery

**Lens:** REST, web, MCP and CLI for the roadmap, evidence, validation, GitHub; the cross-surface contract
**Score:** **6.3 / 10**
**Bottom line:** The roadmap's structural verbs (list, add, move, delete, build, expand, progress, presence, repo, sync, guide, evidence read) are reachable on all four surfaces. The fields the delegation design depends on are not: node owner and acceptance can only be written from the web, and the human validation flow (observation and validate) has no CLI verb. Nothing in the test suite would catch the next field added to the PATCH body and forgotten in the Go client, the CLI and the MCP. All findings are from reading code; no server or database was run, and PARC-06 was not reproduced live.

Read at `main` @ `4d186da` together with the uncommitted working tree. A skeptic re-checked every High in this file; the verdicts are in the table near the end and in the [findings register](./01-findings-register.md).

---

## Scope read

I read the REST handlers for the roadmap and delivery domain (`server/internal/api/roadmap.go`, `roadmap_ai.go`, `roadmap_evidence.go`, `roadmap_validation.go`, `roadmap_progress.go`, `github.go`, `agent_scope.go`), the store functions behind update, move and evidence, the Go client (`server/internal/apiclient/roadmap.go`, `apiclient.go`, `crud.go`), the CLI (`server/cmd/insideout/cli.go`, `cli_roadmap.go`, `cli_crud.go`, `cli_guide.go`), the MCP tools (`server/internal/mcpserver/product_tools.go`, `draft_tools.go`, `server.go`), the Flutter client (`client/lib/api/client_project.dart`, `requests.dart`, `models_roadmap.dart`, `errors.dart`, `errors_l10n.dart`) and the node edit dialog (`client/lib/features/project/roadmap_node_menu.dart`). I read both real-API contract tests. Nothing was run against a server or a database; every finding is from reading code and from greps that are quoted with the finding.

## What is good

- `server/internal/api/agent_scope.go:40` is a working route registry. Every authenticated mux pattern must have an agent decision, and `TestAgentRoutePolicyCoversEveryRoute` (`routes_require_auth_test.go:160`) fails when a route is added without one or an entry names no route.
- `server/cmd/insideout/contract_real_api_test.go:24` runs the real CLI binary, the Go client and the MCP tools against the real handler on real PostgreSQL. No route is re-implemented by the test.
- Reads on CLI and MCP return the API's JSON untouched (`product_tools.go:52`, `apiclient/roadmap.go:61`), so response shapes cannot drift between the REST API, the CLI and the MCP.
- Errors have one body shape (`server/internal/httpx/json.go:30`) and request decoding rejects unknown fields (`json.go:49`).
- The draft and confirm split for build and expand is identical on all four surfaces, including the `replace_conflict` and `expectedCount` handshake.
- JSON bodies are camelCase throughout the domain; the only snake_case keys are GitHub's own webhook payload and the error extras named in PARC-08.

## Findings, worst first

| Id | Sev | Where | Deciding line | Fix |
| --- | --- | --- | --- | --- |
| PARC-01 | High | `server/internal/apiclient/roadmap.go:82` | `func (c *Client) RoadmapUpdate(nodeID string, title, description, status, deadline, completionReason *string)` | Node owner is web-only. Add owner to a `RoadmapPatch` struct, `--owner` on the CLI, `owner_id` on `roadmap_update`, and a members listing. |
| PARC-02 | High | `server/internal/mcpserver/product_tools.go:67` | `"Partially update a node (title/description/status)"` | Acceptance is web-only. Add `acceptance` to the same struct, flag and tool. |
| PARC-03 | Medium | `server/cmd/insideout/contract_real_api_test.go:193` | `cliJSON[doc] (t, "roadmap", "add", "--title", ...)` is the only CLI roadmap write tested | Add a test that derives the field set from the API request structs and checks the Go client, CLI and MCP against it, with a named allowlist. |
| PARC-04 | Medium | `server/internal/api/roadmap.go:27` | `"POST /api/v1/roadmap/{nid}/validate"` has no CLI verb or Go client method | Add `evidence add` and `roadmap validate` to the CLI; document the MCP omission as deliberate. |
| PARC-05 | Medium | `server/internal/api/roadmap.go:291` | `// parentId present-and-null → make it a root; absent → root too` | A position-only move on CLI or MCP lifts the subtree to the root. Make absent mean "keep parent". |
| PARC-06 | Medium | `client/lib/features/project/roadmap_node_menu.dart:263` | `ownerId: ownerId ?? '',` | The web dialog sends every field on every save; a departed owner's id makes the save 403. Send only changed fields. Not reproduced live. |
| PARC-07 | Medium | `server/internal/mcpserver/product_tools.go:79` | `deadline, hasDeadline := req.GetArguments()["deadline"].(string)` | Empty string clears the deadline but means "leave" for every other field; the CLI uses `clear`. Use `clear` on both. |
| PARC-08 | Medium | `server/internal/api/github.go:119` | `map[string]any{"retry_after_seconds": ...}` against `"liveCount"` at `roadmap_ai.go:146` | 369 of 382 errors carry no code; codes and extras use two styles. Define codes in `httpx`, camelCase the extras. |
| PARC-09 | Low | `server/cmd/insideout/cli_roadmap.go:194` | `func strPtrIfSet(v string) *string {` | CLI and MCP cannot clear a description. Detect passed flags with `flag.Visit`. |
| PARC-10 | Low | `server/internal/api/roadmap_evidence.go:18` | `rows, err := s.store.ListRoadmapEvidence(...)` | Unknown or foreign node answers 200 `[]`; the forbidden branch is unreachable. Resolve the node first and answer 404. |
| PARC-11 | Low | `SKILL.md:107` | `Every verb exists 1:1 as CLI command ↔ MCP tool ↔ API route` | False for owner, acceptance, observation and validate. Replace with the list of deliberate exceptions. |
| PARC-12 | Low | `client/lib/api/client_project.dart:83` | `await _dio.post('/projects/$projectId/sync-github');` | Half the Dart roadmap calls bypass the testable `ApiCall` builders. Move them into `requests.dart`. |

Field by field, the PATCH body has seven editable fields. Title, status and completion reason are consistent on all four surfaces. Deadline is writable everywhere but with two clear idioms. Description is writable everywhere but clearable only on the web. Acceptance and owner are writable only on the web. `validationReason` is written only through the validate route, which is web-only.

## Lead verification

| Lead | Verdict | Line that decided it |
| --- | --- | --- |
| 2 | Confirmed | `roadmap.go:156` answers `only the project owner or a workspace admin can change the roadmap`; the owner check in `store/roadmap.go:261` is `requirePlanWriter` only. |
| 3 | Confirmed | The route table at `roadmap.go:17-27` plus `roadmap_ai.go:68-71` has no subtree-scoped or owner-scoped read. |
| 5 | Confirmed for writes, refuted for reads | No owner parameter in `apiclient/roadmap.go:82`, `cli_roadmap.go:76` or `product_tools.go:68-71`. But `roadmap_list` returns the raw API JSON (`product_tools.go:54`), which carries `ownerId` and `ownerName` (`roadmap.go:45-46`), so CLI and MCP can read a node's owner. |
| 13 | Partly verified (roadmap domain only) | 21 routes in the domain, 15 MCP tools, 16 CLI verbs; the matrix lists each. |
| 14 | Checked | Neither the review-4 register nor the open-findings record mentions the owner or acceptance gap; review 4 marked it `IMPLEMENTED ... none` (`20-feature-implementation.md:59`). The unfenced CLI `evidence` output is open record R2-09 and is not refiled. |

## Cross-cutting answers

There is no single source of truth for the whole surface. There is no OpenAPI file. The agent route policy is a registry of routes, not of fields, and nothing links a route to a Go client method, a CLI verb or an MCP tool. The contract test proves the covered verbs work end to end but covers `roadmap add`, `roadmap list` and a title-only `roadmap_update` in this domain. The Go client and the Dart client agree on every request key they both send; the Go client simply sends fewer. Response shapes cannot disagree because the Go client does not parse them.

## Recommendations

1. Add owner and acceptance to the Go client, the CLI and the MCP in one change, with a members listing.
2. Add the field-level parity test so the next node field cannot be forgotten.
3. Make an absent `parentId` keep the parent.
4. Give the CLI the validation verbs.
5. Make the web dialog send only changed fields.
6. Settle one "clear" convention and one error-code style.

## What this role would do first

I would write the field-level parity test before touching any surface, and let owner and acceptance fail it. Then I would add the two fields to the Go client, the CLI and the MCP until it passes. The delegation design the owner asked about will add more node fields (a delegate, a subtree root); without the test each of them will repeat this gap, and with it the gap closes once.

Not checked: nothing was run live, so PARC-05 and PARC-06 are from reading the handler, the store and the dialog; areas outside the roadmap and delivery domain were not inventoried.

## Surface matrix

| Capability | REST | Web | MCP | CLI | Note |
| --- | --- | --- | --- | --- | --- |
| List nodes | GET /projects/{pid}/roadmap | listRoadmap; canvas and tree | roadmap_list(project_id) | roadmap list <project-id> | Consistent. Reads return raw JSON, so ownerId, ownerName, acceptance, validationReason are readable everywhere. |
| Add node | POST /projects/{pid}/roadmap {title, description, parentId} | createRoadmapNode(title, parentId); no description on create | roadmap_add(project_id, title, description, parent_id) | roadmap add --title --description --parent | Web omits description on create; it is set in the edit dialog. Accident, harmless. |
| Field: title | PATCH /roadmap/{nid} title | edit dialog | roadmap_update title | roadmap update --title | Consistent. |
| Field: description | PATCH description | edit dialog; can clear | roadmap_update description; cannot clear | --description; cannot clear | Empty string is dropped on CLI and MCP (PARC-09). Accident. |
| Field: status | PATCH status (locked, pending, in_progress, done; validated refused as a transition) | status picker | roadmap_update status | --status | Consistent. |
| Field: deadline | PATCH deadline (RFC3339, empty string clears) | date picker and clear button | roadmap_update deadline; empty string clears | --deadline RFC3339 or the word clear | All four can write it; the clear idiom differs between MCP and CLI (PARC-07). |
| Field: acceptance | PATCH acceptance | acceptance-field in edit dialog | none | none | Accident (PARC-02). Go client has no parameter. |
| Field: owner | PATCH ownerId (empty string clears) | owner-picker in edit dialog | none | none | Accident (PARC-01). No member listing on CLI or MCP either. |
| Field: completion reason | PATCH completionReason | completion prompt when moving in_progress to done | roadmap_update completion_reason | --completion-reason | Consistent; usage doc row omits the flag (PARC-11). |
| Move node | POST /roadmap/{nid}/move {parentId, position} | moveRoadmapNode; always sends both | roadmap_move(node_id, parent_id, position) | roadmap move --parent --position | Absent parentId means root, so position-only moves on CLI and MCP re-parent to root (PARC-05). |
| Delete node | DELETE /roadmap/{nid} (humanOnly) | node menu delete | roadmap_delete; withheld once a login is held | roadmap delete | Deliberate: agents never delete. |
| Build from PRD (draft and confirm) | POST /prds/{pid}/build and /build/confirm {tree, expectedCount} | prd_page build and replace prompt | build, build_confirm(tree, expected_count) | build, build-confirm -tree -expected-count | Consistent. |
| Expand node (draft and confirm) | POST /roadmap/{nid}/expand and /expand/confirm {children} | roadmap_expand_draft | expand, expand_confirm(children) | expand, expand-confirm -children | Consistent. |
| Evidence list | GET /roadmap/{nid}/evidence | listNodeEvidence | roadmap_evidence(node_id) | evidence <node-id> | Name differs between CLI and MCP; unknown node answers 200 [] (PARC-10). CLI output unfenced is open record R2-09. |
| Evidence add (validation observation) | POST /roadmap/{nid}/evidence {detail, sourceUrl, observedAt} (humanOnly) | addValidationObservation form | none (deliberate) | none | CLI gap is an accident (PARC-04). |
| Validate node | POST /roadmap/{nid}/validate {reason} (humanOnly) | validateRoadmapNode | none (deliberate) | none | CLI gap is an accident (PARC-04). |
| Progress view | GET /projects/{pid}/roadmap/progress | roadmapProgress | progress(project_id) | roadmap progress <project-id> | Consistent; name differs between CLI and MCP. |
| Progress stream | GET /projects/{pid}/roadmap/progress/stream (SSE, humanOnly) | progressStream | none | none | Deliberate: live streams are the web's. |
| Presence snapshot | GET /projects/{pid}/presence | via stream | presence(project_id) | presence, roadmap presence | Consistent. |
| Presence stream and cursor | GET /projects/{pid}/presence/stream, POST /projects/{pid}/cursor (humanOnly) | presenceStream, cursor | none | none | Deliberate. apiclient.Cursor exists with no caller (PARC-12). |
| GitHub repo set | PUT /projects/{pid}/repo {repoUrl} | project_github_card setRepo | repo_set(project_id, repo_url) | repo set <project-id> <repo-url> | Consistent. |
| GitHub sync | POST /projects/{pid}/sync-github | syncGithub | sync(project_id) | sync <project-id> | Consistent. |
| GitHub webhook | POST webhook path (public, signature-gated) | none | none | none | Deliberate: server-to-server. |
| Guide (insideout.yaml) | GET /projects/{pid}/guide (text/yaml) | none | guide(project_id) | guide [--out file] <project-id> | Web omission is undocumented; plausible, since the file is committed from a terminal (PARC-12). |

## Skeptic verdicts on this lens's Highs

| ID | Verdict | Reason | What the skeptic ran |
| --- | --- | --- | --- |
| PARC-01 | CONFIRMED | Confirmed by reading and by running the built CLI. The REST handler accepts ownerId (roadmap.go:182) and the web edit modal has an owner picker, but the shared Go client method has no owner parameter, the roadmap_update tool lists node_id, title, description, status, deadline, completion_reason only, and the CLI rejects --owner. No member listing exists in apiclient, the CLI verb table or the MCP tools, so a caller cannot discover a teammate's user id either. Delegation is the owner-stated core flow and the usage doc claims parity with web and MCP, so High stands. One overstatement: 'one surface of four' counts REST as unable, but raw REST can set the owner; three surfaces (web, REST) versus two that cannot (CLI, MCP) is the accurate picture. Same root cause as TREEX-04. | HOME=/tmp/insideout-rv5/home-tree-surfaces /tmp/insideout-rv5/insideout roadmap update --owner x 00000000-0000-0000-0000-000000000000 -> 'flag provided but not defined: -owner'; grep -rn -i 'owner\|acceptance\|member' over apiclient/*.go, cli_roadmap.go, mcpserver/*.go (non-test) -> no owner, acceptance or member-listing hit. |
| PARC-02 | DOWNGRADED to Medium | The fact is right: roadmap_update has no acceptance option and the CLI rejects --acceptance, while PATCH accepts it. The severity is not. Acceptance is not part of the owner's stated delegation requirement, and nothing in the server gates on it: outside the store column and the PATCH handler it is only copied into agent context and progress items (agentcontext.go, roadmaptime.go, roadmap_progress.go), so an empty acceptance blocks no flow. The scenario's dependency on PRODUCT.md:250 does not exist in code either, since agent-advanced evidence is not built (POST /roadmap/{nid}/evidence is humanOnly). That leaves a field-level parity miss under a doc heading that claims parity, which is design debt rather than a broken core flow. It is also the acceptance half of TREEX-04 and is fixed by the same change as PARC-01. | HOME=/tmp/insideout-rv5/home-tree-surfaces /tmp/insideout-rv5/insideout roadmap update --acceptance x <uuid> -> 'flag provided but not defined: -acceptance'; grep -rn -i acceptance server/internal server/cmd (non-test) -> only display and context copies, no gate. |

## Ordered recommendations

1. Close the field gap in one change: a RoadmapPatch struct in the Go client carrying owner and acceptance, the matching CLI flags and MCP parameters, and a members listing so an owner id can be found without the web.
2. Add a field-level parity test that derives the expected parameter set from the API request structs and fails when the Go client, the CLI or the MCP lacks one, with a named allowlist for deliberate omissions.
3. Make move safe: an absent parentId keeps the parent, an explicit null makes a root, on the server and in both Go surfaces.
4. Give the CLI the human validation verbs (evidence add, roadmap validate) and record the MCP omission as deliberate in the usage doc.
5. Change the web edit dialog to send only changed fields, so it behaves like the CLI against the same partial-update route.
6. Settle one convention for 'clear' across CLI and MCP (the word clear, with empty meaning absent), and one style for error codes and error extras.
7. Replace the '1:1' sentence in SKILL.md with the short list of deliberate exceptions once the gaps are closed.
