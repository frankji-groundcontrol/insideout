# 10 · Feature completeness: the product definition

**Lens:** PRODUCT.md promises against server and client code
**Score:** **5 / 10**
**Bottom line:** About half of PRODUCT.md runs today. Of 50 distinct promises I checked against the code, 16 run fully, 19 run in part and 15 have no code; of the ten end-state loop items, two run fully, seven in part and one (Decision Owner and Driver authority) not at all. Most gaps are honestly tracked, but four are not, and the shipped build-roadmap path has three defects that I derived from the code and the row-level-security policies without executing them: a non-author member's build mints unlinked projects, an agent token can replace the whole roadmap, and a rebuild deletes all evidence. Not checked: nothing was run against a database, and the CLI, MCP and Flutter surfaces were read only where a flow passed through them.

Read at `main` @ `4d186da` together with the uncommitted working tree. A skeptic re-checked every High in this file; the verdicts are in the table near the end and in the [findings register](./01-findings-register.md).

---

## Scope read

I read PRODUCT.md end to end, docs/TODO.md, the promise-gap enumeration and its nine slice records, the six promise-partial records, the 63-row promise register, and the open review-4 findings. I then traced nine flows through handler, store and migration: capture an idea, start progressing, a coach turn and form a version now, commit a version, build a roadmap from a PRD and confirm it, mark a node done, GitHub evidence, the audience view, and proposal acceptance. I ran `scripts/tests/check_product_promise_gaps.py` (exit 0). Nothing was executed against a database, so the three build-path defects below are derived from code and row-level-security policy, not observed.

## How much of the product definition runs

Of 50 distinct promises, 16 run fully, 19 run in part and 15 have no code. Counting a partial as half, about half of PRODUCT.md runs. Of the ten end-state loop items, two run fully (items 1 and 4), seven in part, and one (item 7, Decision Owner and Driver authority) not at all. The register's own count of 36 implemented rows out of 63 is too generous by seven rows.

## What is good

- The human Commit is one transaction, immutable and closed to agent tokens (`server/internal/store/prd_commits.go:35-108`, `server/internal/api/agent_scope.go:48`).
- The completion rule is enforced in the store, so every surface meets it (`server/internal/store/roadmap_completion.go:20-44`); only a human endpoint reaches validated (`roadmap_validation.go:60-104`).
- Evidence mapping is conservative: release only on a successful deployment status, failures appended (`server/internal/api/github_webhook_evidence.go:112-121`).
- Proposal apply is idempotent and atomic (`server/internal/store/proposal_items.go:124-172`).
- The nine parked slices are each tracked with a verbatim promise and a fix prompt (`docs/issues/2026-10-11-product-promise-gaps.md`).

## Findings, worst first

| ID | Severity | Where | Deciding line | Fix |
|---|---|---|---|---|
| FEAT-01 | High | `server/internal/store/roadmap_plan.go:242` | `UPDATE insideout.prds SET project_id = $2 WHERE id = $1` with the row count unread, behind `requireMember` only | Require the PRD author or an admin, check the row count, create the project only on confirm |
| FEAT-02 | High | `server/internal/api/agent_scope.go:79` | `"POST /api/v1/prds/{pid}/build/confirm": agentOK,` while every DELETE is humanOnly | Make replace human-only; refuse an agent-written completion reason |
| FEAT-03 | High | `server/internal/store/roadmap_plan.go:58` | `DELETE FROM insideout.roadmap_nodes WHERE project_id = $1`, and evidence is `ON DELETE CASCADE` | Refuse a replace that would delete evidence or validated nodes; later merge or archive |
| FEAT-04 | High | `docs/issues/2026-10-11-product-promise-gaps.md:41` | "Per-Project roles" listed under "not re-tracked here" | File a dated record with a first slice and add it to the guard |
| FEAT-05 | Medium | `server/internal/roadmaptime/roadmaptime.go:90` | `p.Now = append(p.Now, item)` with no owner condition | Add a needs-owner list; restamp the register row |
| FEAT-06 | Medium | `server/internal/store/roadmap_plan.go:95` | `WHERE n.project_id = $1 AND n.parent_id IS NOT NULL` with no leaf test | Add the leaf condition; test a three-level tree |
| FEAT-07 | Medium | `server/internal/store/roadmap_completion.go:31` | `SELECT EXISTS (SELECT 1 FROM insideout.roadmap_evidence WHERE node_id = $1)` | Count only implementation, release or validation evidence |
| FEAT-08 | Medium | `server/internal/store/ideas.go:245` | `insertPrd(...)` and no project insert | Create the Project at convert |
| FEAT-09 | Medium | `client/lib/features/prd/prd_page.dart:140` | `result = await api.confirmBuildFromPrd(widget.id, tree);` | Show the draft before confirming |
| FEAT-10 | Medium | `server/internal/agent/tools.go:283` | `cannot draft yet — no recorded fact for: %s` | Add a human-only form-now action and button |
| FEAT-11 | Medium | `docs/review/2026-10-10-multirole-3/50-promise-register.md:73` | "Coach recommends Commit moments ... **IMPLEMENTED**"; the prompts never mention a Commit | Restamp seven rows; file three records |
| FEAT-12 | Medium | `server/internal/store/project_updates.go:56` | `if kind == "agent_proposal" {` then `requirePlanWriter` | Let members propose; make decisions append-only |

FEAT-01 in plain terms: a member who did not write the PRD taps Build MVP. The server inserts a project owned by that member, the statement that links it to the PRD is filtered to zero rows by row-level security, and nobody checks. Each tap leaves two unlinked projects. The draft route, described to agents as writing nothing, creates a project even for the author.

FEAT-02 and FEAT-03 share one statement. Replacing the roadmap deletes every node and, by cascade, every evidence row, validation record and completion reason. An agent token is refused when deleting one node but allowed to replace them all, and the usage guide tells operators that node ids are stable.

## Lead verification

| Lead | Result | The line that decided it |
|---|---|---|
| 1 | Confirmed | `20261005120000_roadmap_acceptance_owner.sql:8` adds `owner_user_id` only |
| 2 | Confirmed | `memberships.go:248-256`: owner, or a member whose role is admin |
| 3 | Confirmed | `roadmap.go:17-27` and `roadmap_ai.go:68-71` are the whole roadmap route set |
| 5 | Confirmed | `mcpserver/product_tools.go:67-71`: roadmap_update takes title, description, status, completion_reason |
| 6 | Confirmed | the guard prints "9 parked slices with live premises, 0 tracking failures" |
| 7 | Not re-run | outside this lens |
| 13 | Partly | 91 `HandleFunc` registrations counted in `server/internal/api`; not reconciled with 107 |
| 14 | Used | R3-37 covers the stale validated rows; R2-04 is fixed in code (`roadmap_plan.go:133`) but still listed open |

## Recommendations

1. Guard the replace path: author-or-admin project creation with a checked link, human-only replace, no silent evidence loss.
2. File the four untracked gaps: per-Project roles, Coach recommendations for audience and Commit moments, a working version from an earlier Commit, unmatched pull requests and deployments.
3. Restamp the register and make the guard require a record for DOCS-ONLY rows as well.
4. Create the Project when an Idea starts progressing.
5. Show the roadmap draft before the web writes it, and add a form-a-version-now control.
6. Tighten completion evidence to meaningful kinds and require an owner for Now.

## What this role would do first

Write three red tests against real PostgreSQL on the build path: a non-author member building an unlinked PRD, an agent token confirming a build over a populated roadmap, and a rebuild over a node that has evidence. All three should fail today. Then change `EnsureProjectForPrd` and `ReplaceRoadmapTree` until they pass. That one piece of work proves or refutes the three High code findings and removes the only places where the shipped product can lose a team's history.

## Property checks

| Property | Status | Evidence |
| --- | --- | --- |
| One-step Idea capture, private to its author by default (PRODUCT.md:68) | implemented | server/internal/store/ideas.go:75 lists only rows WHERE author_id = the caller |
| Start progressing creates the Project, keeps title and body, opens the working version and Roadmap workspace (PRODUCT.md:69-71) | partial | ideas.go:245-262 creates PRD, conversation and first message; no project until roadmap_plan.go:233 (FEAT-08) |
| Existing project or GitHub repository entry without a retrospective Idea (PRODUCT.md:73) | implemented | server/internal/api/projects.go:17 manual create; server/internal/api/github.go:15 repo link |
| Roadmap is an authoritative baseline only when tied to a human-confirmed Commit (PRODUCT.md:71-72, 264-265) | missing | server/internal/api/roadmap_draft.go:50 'It does not gate on PRD status or a commit'; tracked in docs/issues/2026-10-18-promise-partial-commit-baseline-tie.md |
| Git required when code work becomes someone's main Focus (PRODUCT.md:79-81) | missing | no gate in server/internal/api; tracked in docs/issues/2026-10-11-focus-surface.md |
| Adaptive coaching depth, no fixed sequence (PRODUCT.md:87-90) | partial | prompts_en.go:28 adaptive clarify; fixed four stages gated at server/internal/agent/tools.go:268-318 |
| Every question carries priority, reader and why-now (PRODUCT.md:92-97) | implemented | server/internal/agent/prompts_en.go:30 (prompt instruction; not a structured field) |
| User can always form a version now, no magic phrase (PRODUCT.md:102-104) | partial | tools.go:283 four-fact gate; no button or route (FEAT-10); readiness and Commit-with-gaps work (server/internal/api/prd_commits.go:61) |
| Critical product argument sought before a first version (PRODUCT.md:108-119) | implemented | tools.go:277 requires problem, segment, alternative, whynow facts |
| Concrete personas with priority, four fields and grounding labels (PRODUCT.md:123-134) | missing | users section is free text; tracked in docs/issues/2026-10-11-persona-grounding-labels.md |
| Severity-ranked critique with keep-as-is; critic failure disclosed and not a trap (PRODUCT.md:138-147) | implemented | prompts_en.go critique stage; tools.go:312-314 finalize allowed after a recorded skip; client/lib/features/prd/coach_trust_notes.dart:11 (retry not traced) |
| Correct the Coach by editing directly or continuing (PRODUCT.md:149-150) | implemented | server/internal/api/prds.go:21 PATCH; tools.go:198 compare-and-swap on the coach's write |
| One human-confirmed primary audience per version (PRODUCT.md:157) | implemented | server/internal/api/prd_commits.go:43 validates primaryAudience on every Commit |
| Coach may recommend the audience; secondary audiences attachable (PRODUCT.md:157-158) | missing | grep audience in prompts_en.go empty; secondary tracked in docs/issues/2026-10-18-promise-partial-secondary-audiences.md; recommend half untracked |
| Four audience projections of one core; switching grants no permission (PRODUCT.md:159-174) | implemented | server/internal/audienceview/audienceview.go:24-70; one requireAuth route at prds.go:19 |
| Audience-specific readiness, no universal completeness score (PRODUCT.md:171-173) | implemented | server/internal/readiness/readiness.go:65-107 |
| Coach records candidate facts; raw quotes stay internal; no cross-PRD reuse (PRODUCT.md:178, 189-195) | implemented | tools.go:57 record_fact; facts live in conversation meta (tools.go:233-243) |
| Critical facts presented for confirmation before a Commit, with confirm/correct/remove/discuss (PRODUCT.md:179-187) | missing | no facts route in server/internal/api; tracked in docs/issues/2026-10-11-facts-trust-layer.md |
| Autosaved working version (PRODUCT.md:201-202) | missing | manual save at client/lib/features/prd/prd_page.dart:112; tracked in docs/issues/2026-10-11-prd-autosave.md |
| Immutable human Commit with name, audience, summary, unresolved items and Diff (PRODUCT.md:203-205, 209) | implemented | server/internal/store/prd_commits.go:70-99; agent_scope.go:48 humanOnly |
| Product Branch and Merge (PRODUCT.md:206-210) | missing | tracked in docs/issues/2026-10-11-product-branch-merge-decision-log.md |
| Coach recommends Commit moments and never gates a Commit (PRODUCT.md:212-216) | partial | no gate (prd_commits.go:21-78); no recommendation: grep commit in prompts_en.go empty (FEAT-11) |
| Any Commit can begin a new working version without rewriting history (PRODUCT.md:220-221) | missing | prds.go:16-26 has no restore or start-from route; grep restore over client/lib/features/prd and apiclient empty; untracked |
| Decision Log: who, why, evidence, alternatives (PRODUCT.md:218-220) | partial | free-text decision_note (prd_commits.go:32); tracked with Product Branch |
| Workspace roles manage organisation access (PRODUCT.md:229) | implemented | server/internal/store/memberships.go:221-229 role admin or member |
| Per-Project roles: Decision Owner, Driver, Contributor, Reviewer, Viewer (PRODUCT.md:229-240) | missing | memberships.go:236 requirePlanWriter is the only project authority; disclosed, no dated issue (FEAT-04) |
| Contributions remain proposals until accepted (PRODUCT.md:243-244) | partial | project_updates.go:56-58 only plan writers may propose (FEAT-12) |
| Each Now item has exactly one human owner; an Agent cannot be the only owner (PRODUCT.md:246-247) | partial | owner_user_id references users (migration 20261005120000:8) but Now needs no owner (roadmaptime.go:87-90) (FEAT-05) |
| Focus pointer per session and team Now (PRODUCT.md:247-249) | missing | focus is only a query parameter (server/internal/api/agent_routes.go:38); tracked in focus-surface record |
| Scoped feedback invitations for people outside the Workspace (PRODUCT.md:254-259) | missing | tracked in docs/issues/2026-10-11-scoped-feedback-invitations.md |
| One shared hierarchical Roadmap graph (PRODUCT.md:263-270) | implemented | server/internal/store/roadmap.go:80 parent_id; one table serves progress, canvas and agent context |
| Progress view: Now dominant, at most three Next, Done with its evidence (PRODUCT.md:272-273) | partial | roadmaptime.go:103-108 caps Next at three; DoneItems carry no evidence (roadmaptime.go:50-58) |
| Canvas and History views of the same graph (PRODUCT.md:273-275) | implemented | client/lib/features/project/roadmap_canvas.dart; client/lib/features/prd/versions_page.dart; project_updates_feed.dart |
| Five Lenses over the graph (PRODUCT.md:277-288) | partial | agent context only (server/internal/agentcontext/agentcontext.go:96); four human lenses tracked in roadmap-lenses record |
| Product outcomes carry target time ranges (PRODUCT.md:292) | missing | tracked in docs/issues/2026-10-18-promise-partial-outcome-time-ranges.md |
| No deadline, no Now; pressure ladder; agents see pressure (PRODUCT.md:293-303) | implemented | roadmaptime.go:21-35 and 87-88; agentcontext.go:173-174 |
| Overdue work resolved with a recorded reason (PRODUCT.md:299-301) | missing | tracked in docs/issues/2026-10-18-promise-partial-overdue-resolution.md |
| Dependency delay shows the path and a time Diff (PRODUCT.md:301-302) | missing | tracked in docs/issues/2026-10-11-roadmap-dependency-edges.md |
| GitHub recommended right after an Idea is saved and on manual project create, skippable (PRODUCT.md:307-309) | partial | link card on the project page only (client/lib/features/project/project_github_card.dart); no prompt after capture (ideas_page.dart:163-171) |
| Matched GitHub activity advances leaf evidence; observations have their own source (PRODUCT.md:312-321) | implemented | github_webhook_evidence.go:86-92, 112-121; roadmap_validation.go:47-50 kind validation |
| Unmatched activity stays visibly unassociated (PRODUCT.md:324-325) | partial | commits reach the timeline (github_webhook.go:92); an unmatched pull request or deployment returns 0 and leaves nothing (github_webhook.go:136-141) |
| Reverts and failures append evidence; history is not erased (PRODUCT.md:325-326) | partial | append holds (roadmap_evidence.go:53-56); rebuild and delete cascade evidence away (FEAT-03) |
| Change Proposal showing impact, accepted by a human before the baseline changes (PRODUCT.md:326-328) | partial | generic proposal with add_node only (server/internal/api/agent_routes.go:190-193); tracked in promise-partial-change-proposal record |
| One source of truth across Web, CLI, MCP; vocabulary context, focus, checkpoint, propose, version (PRODUCT.md:330-332) | partial | all ride /api/v1; no focus verb (cli.go:179 has agent-context, checkpoint, propose); docs/TODO.md:22-23 lists focus as shipped |
| Compact Focus-scoped agent context in three modes (PRODUCT.md:334-340) | partial | agentcontext.go:96-116 modes; focus has no Why chain, Git context or latest checkpoint (agentcontext.go:127-148) |
| Agents cannot apply strategic changes, Commit or Merge (PRODUCT.md:342-346) | partial | commit and decision human-only (agent_scope.go:47-48); roadmap replace is agentOK (agent_scope.go:79) (FEAT-02) |
| Implemented, released and validated are distinguished (PRODUCT.md:350) | partial | validated status exists (roadmap_validation.go:92); released is only an evidence kind; register drift recorded as R3-37 |
| Completion links to evidence; human override only with a reason (PRODUCT.md:351-352) | partial | roadmap_completion.go:20-44 enforces; any evidence kind counts and an agent token may write the reason (FEAT-07, FEAT-02) |
| After release, compare outcomes with committed assumptions and begin the next version (PRODUCT.md:354-357) | missing | no surface; no dated issue, noted in R3-37 of docs/issues/2026-10-18-review-4-open-findings.md |
| English and Simplified Chinese are first-class (PRODUCT.md:451-452) | partial | audience purpose and why strings are server English shown raw (client/lib/features/prd/audience_view_page.dart:146, 190); export headings R3-47 |

## Skeptic verdicts on this lens's Highs

| ID | Verdict | Reason | What the skeptic ran |
| --- | --- | --- | --- |
| FEAT-01 | CONFIRMED | Reproduced on the private database at both the store and the HTTP layer. A non-author member's two calls each returned err=nil with a different project owned by that member; prds.project_id stayed NULL (the prds_update policy in 20261010130000 admits only the author or an admin, and the Exec result is discarded). The member's ReplaceRoadmapTree on the orphan project succeeded (n=2). The author's later draft created a third project and linked it with nothing confirmed. The only gate above the INSERT is requireMember. The web Build button is enabled for any viewer of an approved PRD, so this is on the documented web path as well as CLI and MCP. No OPEN record covers it: docs/issues/2026-07-26-backend-optimization-deferred.md only covers the concurrent double-create race, which the advisory lock fixed. Same root cause as PARB-02. | go test -overlay /tmp/insideout-rv5/build-path/overlay.json ./internal/store/ -run TestRV5BuildPath_NonAuthorEnsure -v -count=1 -> 'non-author ensure #2: err=<nil> ... sameAsFirst=false', '[after non-author #2] projects in workspace: 2', 'prd.project_id after non-author ensures: <nil>', 'non-author ReplaceRoadmapTree on own orphan project: n=2 err=<nil>', '[after author ensure (draft only, nothing confirmed)] projects in workspace: 3' |
| FEAT-02 | CONFIRMED | Reproduced through the handler chain with an agent token for the project owner: DELETE /roadmap/{nid} -> 403 agent_scope_denied; build/confirm without a count -> 409 with liveCount 3 (the server hands the agent the number it needs); build/confirm with expectedCount 3 -> 201, the old node gone and its evidence row gone with it; PATCH status done with a one-sentence completionReason -> 200. Allowing the four build/expand routes to agents is a recorded decision (docs/architecture/agent-token-scope.md lists them as agent-allowed), so that alone would be a downgrade, but the same document makes every DELETE human-only and names prompt injection as what the scope defends against, and the replace is an unrecorded bulk delete that cascades to evidence. It is not among the accepted limits in R2-15 of the review-4 open findings. The completion-reason half is weaker: the 2026-10-20 changelog deliberately ships 'MCP: roadmap_update argument completion_reason', and PRODUCT.md:351-352 ('may be overridden by a human only with a reason') can be read either way; taken alone it would be Medium. The evidence loss overlaps FEAT-03, but the scope hole is separate. Needs the agent to hold a plan writer's token (requirePlanWriter still applies). | go test -overlay /tmp/insideout-rv5/build-path/overlay_api.json ./internal/api/ -run TestRV5BuildPathAPI_AgentAndStatus -v -count=1 -> 'agent DELETE one node: status=403 body={"code":"agent_scope_denied"', 'agent build/confirm expectedCount=3: status=201 body={"nodeCount":2', 'evidence on old leaf after: status=200 body=[]', 'agent PATCH done + completionReason: 200 ... "status":"done"' |
| FEAT-03 | CONFIRMED | Reproduced: one evidence row on a leaf, then ReplaceRoadmapTree with the live count and the identical tree -> the old leaf id is gone and the project holds 0 evidence rows. roadmap_evidence.node_id is ON DELETE CASCADE, and validation and completion_reason are columns on the deleted rows. Replace is the only way to derive a roadmap from a later PRD version, and PRODUCT.md loop item 10 says 'start the next version without erasing history'. Two points in the finding are overstated: the web prompt does say 'The current tree will be lost' (it names nodes, not evidence or validation), and the quoted cli-mcp-and-guidance.md:244-245 sentence is about regenerating the guide, not rebuilding the roadmap; the consequence still holds because the guide matches by node id. The replace itself is a deliberate, confirmed design (the expectedCount guard), so this stays High and not Critical. No OPEN record covers the evidence loss: the two build records in docs/issues (2026-07-25 conflict guard, BUG-022) do not mention evidence. | go test -overlay /tmp/insideout-rv5/build-path/overlay.json ./internal/store/ -run TestRV5BuildPath_ReplaceCascadesEvidence -v -count=1 -> 'before replace: nodes=3 evidence on leaf a=1', 'replace with expectedCount=3: n=3 err=<nil>', 'after replace: nodes=3 oldLeafIDSurvives=false evidenceRowsOnProject=0' |
| FEAT-04 | DOWNGRADED to Medium | The facts hold: no project-role storage exists (grep for project_members or project_role across server/db/migrations and server/internal returns nothing), requirePlanWriter at memberships.go:236 admits only the project owner or a workspace admin, and the guard script exits 0 with '9 parked slices with live premises, 0 tracking failures' because it never asks for a roles record. But the defect this finding states, and its fix, is a missing tracking record, and that omission is a recorded, deliberate choice in the OPEN record docs/issues/2026-10-11-product-promise-gaps.md (heading 'Already disclosed in PRODUCT.md itself (not re-tracked here)'); the gap is also stamped DOCS-ONLY in row 82 of the 50-promise register and conceded in PRODUCT.md:429-436. So it restates an open record and argues its disposition. The absent capability itself is High under the owner's stated delegation requirement (coordinator measurement R1 shows an assigned member cannot write the delegated subtree), but that harm belongs to the delegation finding, which is not in this cluster; this framing adds no separate runtime harm. Medium, not Low, because the owner's requirement now makes this the first slice to plan and the issues index gives no entry point or fix prompt. | python3 -I scripts/tests/check_product_promise_gaps.py -> '9 parked slices with live premises, 0 tracking failures' (exit 0); grep -rln 'project_members\|project_roles\|project_role' server/db/migrations server/internal -> no match |

## Ordered recommendations

1. Guard the roadmap replace path first: restrict project creation and linking to the PRD author or an admin with a checked row count, close build/confirm to agent tokens, and refuse a replace that would delete evidence or validated nodes.
2. File the four untracked gaps as dated records with fix prompts: per-Project roles, the Coach recommending an audience and Commit moments, starting a working version from an earlier Commit, and the visibility of unmatched pull requests and deployments.
3. Restamp promise register rows 27, 73, 74, 84, 85, 96, 114 and 118 and update the two stale rows named in R3-37, then extend scripts/tests/check_product_promise_gaps.py so a DOCS-ONLY row also needs a linked record.
4. Create the Project when an Idea starts progressing, so the GitHub link, the timeline and the agent context exist from the first working version.
5. Show the drafted roadmap to a person before the web writes it, and add an explicit form-a-version-now control beside the coach.
6. Tighten the completion rule to implementation, release or validation evidence, and require an owner (or surface a needs-owner list) for items in Now.
7. Mark R2-04 closed in docs/issues/2026-10-18-review-4-open-findings.md: the code and docs/changelogs/2026-10-18-rv4-remediation/w4-rv-roadmap-expand.md:24 say it is fixed, and line 200 of that changelog still carries the undone integrator note.
