# Wave 4, roadmap expand and build confirmation — review-4 remediation

Branch `rv4/w4-rv-roadmap-expand`, base `582a4bd`, eight commits: four red tests
(`8786a6c`, `521d5b5`, `b22bf4f`, `cc889ae`), three fixes (`cbc3ff8`, `7f32a67`,
`c4a6cb4`) and one comment correction (`0e8a8f1`), then this fragment. Findings
are cited as `rv4-<ID>` because review IDs restart per review. The seven come
from the whole-change review of the remediation range. Six of them share one
root: the two AI write routes of the roadmap, `POST /roadmap/{nid}/expand/confirm`
and `POST /prds/{pid}/build/confirm`, checked workspace membership only, while
every other roadmap write takes the project owner or a workspace admin
([BUG-020](../../issues/2026-10-06-bug-020-workspace-member-writes-the-plan.md)).
Record R2-04 of the [open-findings record](../../issues/2026-10-18-review-4-open-findings.md)
carried the expand half as a lead and [w2-proposal-apply](w2-proposal-apply.md)
listed it as a ceiling; this cluster closes it. The seventh finding, and the
agent half of one of the six, are an owner decision (`DEFERRED_OWNER`) and
change nothing. No address, host, secret or one-time code is spelled in this
fragment; every account in the runs below was a throwaway on a private
database.

## Dispositions

| ID | Severity | Status | Red | Fix | Note |
| --- | --- | --- | --- | --- | --- |
| rv4-FR-F198 | High | `FIXED` | `8786a6c`, `521d5b5` | `cbc3ff8` | expand confirm needs a plan writer and takes the project lock; closes R2-04 |
| rv4-FR-F116 | Medium | `FIXED` | `8786a6c` | `cbc3ff8` | the same defect, the same pair |
| rv4-FR-F126 | Medium | `FIXED` | `8786a6c`, `521d5b5` | `cbc3ff8` | role and lock as above; the cap request is `7f32a67` |
| rv4-FR-F204 | Medium | `FIXED` | `8786a6c` | `cbc3ff8` | expand loops over `createRoadmapNodeTx`, as the finding suggests; the "any workspace member" comments corrected |
| rv4-FR-F122 | Low | `FIXED` | `b22bf4f` | `7f32a67` | 40 subtasks, 200-character titles, 2000-character descriptions; the role and lock half is `cbc3ff8` |
| rv4-FR-F120 | Medium | `PARTIAL` | `cc889ae` | `c4a6cb4` | the member half is fixed; the agent half is the owner's question below |
| rv4-FR-F101 | Medium | `DEFERRED_OWNER` | none | none | PRODUCT.md and the agent scope document read differently; question below, nothing changed |

## What changed

- **Expand confirmation writes as a plan writer, behind the project lock**
  (rv4-FR-F198, F116, F126, F204; red `8786a6c` for the store and `521d5b5` for
  the route, fix `cbc3ff8`). `ExpandRoadmapNode` read the parent's project,
  checked membership and ran its own `INSERT ... COALESCE(MAX(position)+1, 0)`
  with no lock. It now resolves the project, calls `requirePlanWriter` first
  and inserts each child through `createRoadmapNodeTx`, the function
  `CreateRoadmapNode` and the proposal apply use. Calling it first means a plain
  member is refused even with nothing to add, while a stranger still gets
  `ErrNotFound` and learns nothing. The transaction is still one: the whole
  expansion lands or none of it does, the `expandFailAt` seam is kept and
  `TestRoadmap_ExpandAtomic` passes unchanged. The project's advisory lock is
  taken on the first child and held to commit; the later children take it again
  at no cost. `handleConfirmExpandNode` maps `ErrForbidden` to 403 with the
  sentence the sibling handlers use ("only the project owner or a workspace
  admin can change the roadmap"). The comments on `ExpandRoadmapNode` and
  `ReplaceRoadmapTree` said "any workspace member"; they now say what the code
  does.
- **Caps on what expand confirmation writes** (rv4-FR-F122, and the cap request
  of F126; red `b22bf4f`, fix `7f32a67`, comment `0e8a8f1`). The handler applied
  none of the caps COMP-01 put on the other writes. It now refuses, with a 400
  that names the limit and before anything is written, more than 40 subtasks, a
  title over 200 characters and a description over 2000. The counts are
  characters (`tooLong`), so 200 three-byte characters still pass. 40 and 200 are
  the planner's own output bounds (`maxSubtasks`, `maxTitleRunes`), so no honest
  draft is refused; 2000 is the cap the other free-text fields carry (validation
  detail, agent summary). The create handler caps a title but not a description;
  that is left alone.
- **Build confirmation replaces the roadmap only for a plan writer**
  (rv4-FR-F120, the member half; red `cc889ae`, fix `c4a6cb4`).
  `ReplaceRoadmapTree` deletes every node of a project and writes the caller's
  tree, and it checked membership only, so a plain member who is refused
  `DELETE /roadmap/{nid}` could wipe the plan. `expectedCount` is a staleness
  guard any member can read from `GET /projects/{pid}/roadmap`; it was never an
  authorisation. The store now calls `requirePlanWriter` and
  `handleConfirmBuildFromPrd` maps `ErrForbidden` to the same 403. **This is a
  behaviour change:** before, any workspace member could "Build the MVP" over an
  existing roadmap; now the project owner or a workspace admin can. The first
  build of a PRD is unchanged, because `EnsureProjectForPrd` makes the first
  builder the owner of the project it creates. The revert is the one line in
  `ReplaceRoadmapTree`.
- **The tests** are two new files, `store/rv4_review_roadmap_expand_test.go`
  (four tests) and `api/rv4_review_roadmap_expand_test.go` (three), on real
  PostgreSQL. The store tests: a plain member, a stranger, an unknown node, a
  promoted admin and an empty expansion (`TestRvExpandNeedsAPlanWriter`); a
  second transaction holding the project lock open while an expansion has to
  wait (`TestRvExpandWaitsForTheProjectPlanLock`); eight workers, four
  expanding ten children each and four adding ten by hand under one parent,
  where no two siblings may share a position
  (`TestRvExpandAndCreateNeverShareASiblingPosition`); and the same member gate
  for the tree replace (`TestRvExpandReplaceTreeNeedsAPlanWriter`). The route
  tests go through the whole handler chain with the owner, a plain member and an
  outsider (`TestRvExpandConfirmRefusesAPlainMember`,
  `TestRvExpandConfirmCapsWhatItWrites`,
  `TestRvExpandBuildConfirmRefusesAPlainMember`). The lock and concurrency tests
  use the project owner, so at the red commit they fail for a reason the role
  gate cannot explain: they pin the lock, not the role.

## Design note: one insert path, not a second lock

The gap opened when `a512e0a` (rv4-FEAT-02) factored `createRoadmapNodeTx` out
of `CreateRoadmapNode` for the proposal apply and left `ExpandRoadmapNode` on its
own copy of the insert. Adding `requirePlanWriter` and the advisory lock to that
copy would have closed today's hole and kept the cause, two statements that must
agree on the position rule. Looping over `createRoadmapNodeTx` costs a few small
statements per child (a role read, the lock, the parent check, the insert), at
most 40 children now that the handler caps the batch, and the next change to the
guard, the lock or the parent check reaches the AI expand with no further edit.
A single multi-row `INSERT` would be faster and would be a third copy of the
position rule. The finding named the loop as its suggested fix.

## Awaiting the owner: agent tokens and the two confirm routes

Covers rv4-FR-F101 and the agent half of rv4-FR-F120. Nothing is changed:
`agentRoutePolicy` (`agent_scope.go`, the four AI routes at lines 78 to 81) and
the two route lists in `agent_scope_routes_test.go` (the four routes sit in the
agent-allowed list, lines 59 to 62) are as they were, so no other route moved.

**What is true today.** A scratch test, not committed, minted a token with the
agent scope and called the routes. `POST /prds/{pid}/build/confirm` with the
right expected count and a one-node tree answered 201 and turned a three-node
roadmap into one node. `POST /roadmap/{nid}/expand/confirm` answered 201 and
created nodes. `DELETE /roadmap/{nid}` with the same token answered 403
`agent_scope_denied`. So an agent can already do through the confirm routes what
the policy refuses it one node at a time.

**The question.** Is [PRODUCT.md](../../../PRODUCT.md) lines 343 to 345 the
promise ("They cannot directly apply strategic changes, Commit a product
version, or Merge a Product Branch. The server enforces the first two for any
token issued to an agent"), or is the
[agent token scope](../../architecture/agent-token-scope.md) table the contract
(it lets an agent add, move and update roadmap nodes, confirm the AI build and
expand drafts and edit PRD sections)? The two disagree only on what "strategic
changes" covers: the architecture document reads it as deciding a proposal and
Commit, which the server does enforce.

**Evidence for each reading.** For "agents propose, humans apply": PRODUCT.md
principle 6 (lines 56 to 58) and the sentence above, and the skill's agent
vocabulary (`SKILL.md`: `propose --kind structure|scope|priority`, "a **human**
accepts/rejects"). For "these are agent work": line 64 of the architecture
document lists "the four AI build/expand routes" among what an agent may call;
the MCP tools are registered as "Agent: write a previously drafted roadmap tree"
and "Agent: write previously drafted subtasks" (`internal/mcpserver/draft_tools.go`),
and `build_confirm` carries `expected_count` so that an agent can confirm a
`replace_conflict` retry (`TestBuildPassesExpectedCount`); the skill tells an
agent to "grow the roadmap" with `roadmap_add/update/move/delete` and `expand`;
`TestAnAgentTokenStillDoesTheAgentsWork` pins the neighbouring direct writes
(roadmap add and update, PRD section edits) as intended, and
`agent_scope_routes_test.go` pins the four routes themselves.
[w2-agent-scope](w2-agent-scope.md) records how the table was drawn.

**Options.**

- **(a) Keep the table, change the words.** Reword PRODUCT.md lines 343 to 345
  and the architecture document to list exactly what the server enforces
  (deciding a proposal, Commit, admin routes, deletes, approve and reject) and to
  say that agents may edit roadmap and PRD structure directly. No code change;
  "proposals until accepted" stays a convention of the propose route.
- **(b) Make the structure writers human-only.** At least `build/confirm`;
  optionally `expand/confirm` and roadmap create, move and update. Flip them in
  `agentRoutePolicy` and move them between the two lists in
  `agent_scope_routes_test.go` in the same commit. The MCP confirm tools tell the
  human to confirm in the web client (or are removed) and agents write through
  `/agent/propose`. Cost: the agent's direct roadmap workflow in the skill and
  the tool descriptions, and `TestAnAgentTokenStillDoesTheAgentsWork`, change.
- **(c) A middle path for the one destructive route.** An agent may confirm a
  build only while the live roadmap is empty, so nothing is deleted; a non-empty
  roadmap answers 403 to an agent token. Cost: the first handler that branches on
  scope, which the policy table was built to avoid, and the MCP tool's
  `replace_conflict` retry (`expected_count`), which exists so an agent can
  replace a roadmap, would be refused.

Which is the smaller change: (a) edits two documents and no code, while (b) and
(c) edit the policy table, its test, the MCP tools and the skill. Which is safer
for data: (b) or (c) on `build/confirm` alone, since it is the only agent-reachable
route that deletes. The worker takes no side.

## Known ceilings, recorded on purpose

- **The web client still offers Expand with AI and Build to every member.** The
  confirm now answers a plain member with the 403 sentence; before, the write
  went through. `roadmap_expand_draft.dart` and `prd_page.dart` do no role check.
  Hiding or disabling the two actions for a plain member is a client follow-up
  (listed below), outside this cluster's files.
- **The draft routes stay member-level.** `POST /prds/{pid}/build` and
  `POST /roadmap/{nid}/expand` write nothing to the roadmap. `loadBuildablePrd`
  calls `EnsureProjectForPrd` for the build draft and the build confirm alike, so
  a member's first draft or build of a PRD still creates the project and makes
  that member its owner. That is existing behaviour, and it is why the first
  build is unchanged by the build-confirmation change above.
- **The build tree has no caps.** `handleConfirmBuildFromPrd` checks only that
  the root has a title; no title length, description length, node count or depth
  is bounded beyond the 1 MiB request body. No finding in this cluster claims it
  and the caps here are for the expand route, so the build tree is recorded, not
  changed.
- **An expansion now waits for the project lock, and the 15 s statement timeout
  is its bound.** The four AI routes are exempt from the 25 s request deadline
  (`deadlineExempt`), so a wait behind a long plan write ends at the request
  pool's server-side `statement_timeout`, with an internal-error answer, not at a
  deadline. Plan writes are short transactions and a manual add waits the same
  way, so the wait is short in practice; before this change the route never
  waited at all.

## Left for the integrator

Documentation, comment and index edits only; no code is waiting.

INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md`: R2-04
(line 100) closed with a link to this fragment (red `8786a6c`, fix `cbc3ff8`,
use the cherry-picked SHAs); the rv4-FULL-01 row (line 42) drops its "R2-04"
sentence; rows for rv4-FR-F198, F116, F126, F204 and F122 `FIXED`, rv4-FR-F120
`PARTIAL` and rv4-FR-F101 `DEFERRED_OWNER`; the owner's question under the Owner
decisions heading. `docs/issues/README.md` likewise.

INTEGRATOR TODO: `docs/issues/2026-10-06-bug-020-workspace-member-writes-the-plan.md`:
one sentence that expand confirmation and build confirmation now follow the same
rule (they were the two plan writers left on workspace membership).

INTEGRATOR TODO: `PRODUCT.md` (lines 343 to 345) and
`docs/architecture/agent-token-scope.md` (line 64, the agent-allowed list): the
wording follows the owner's answer above. If the answer is (b) or (c), reopen
this cluster for `server/internal/api/agent_scope.go`,
`server/internal/api/agent_scope_routes_test.go` (both lists) and the confirm
tools in `server/internal/mcpserver/draft_tools.go`.

INTEGRATOR TODO: `server/internal/api/limits_runes_test.go`: add
`POST /roadmap/{nid}/expand/confirm` (title 200, description 2000, 40 children)
to the rune-limit table, as rv4-FR-F122 asks. The file is not in this cluster's
files; `TestRvExpandConfirmCapsWhatItWrites` already pins the same three caps
with three-byte characters.

INTEGRATOR TODO (optional): `client/lib/features/project/roadmap_expand_draft.dart`
and `client/lib/features/prd/prd_page.dart`: hide or disable Expand with AI and
Build for a plain member. The server now refuses the confirm with 403 and the
client offers both actions to every member.

INTEGRATOR TODO: indexes: the `w4-rv-roadmap-expand` row in this folder's
`index.md`; `docs/changelogs/README.md` and `docs/changelogs/index/2026-10.md`;
the plan board `docs/plans/README.md` for the cluster's state; and
`docs/HANDOFF.md` if the wave's resume path lists clusters.

## Evidence

Red is the output at the red commit, with the fix absent; green is the same test
at the fix commit.

| Finding | Red | Green |
| --- | --- | --- |
| rv4-FR-F198, F116, F126, F204 | store: "member ExpandRoadmapNode: want ErrForbidden, got 2 nodes and err <nil>"; "ExpandRoadmapNode returned (<nil>) while another writer held the project lock, want it to wait"; the eight-worker test failed in 9 of 9 runs, with 27 to 33 of 80 siblings sharing a position. Route: "member confirms an expansion: status 201, want 403" | the three expand store tests and the route test pass under `-race`; all seven new tests are stable over 25 repeated runs (`-race -count=25`). Guards that came with the fix: a stranger is `ErrNotFound`, an unknown node is `ErrNotFound`, a member promoted to admin expands and lands on position 2 |
| rv4-FR-F122 | three refusals answered 201: a 201-character title, a 2001-character description and 41 subtasks; the roadmap grew to 44 nodes | all three answer 400 naming the limit and write nothing; a 200-character title with a 2000-character description, and exactly 40 subtasks, still answer 201 |
| rv4-FR-F120 (member half) | store: "member ReplaceRoadmapTree: want ErrForbidden, got 2 nodes written and err <nil>". Route: "member replaces the roadmap: status 201, want 403" | both pass; the roadmap is unchanged after the refused replace; a stranger is `ErrNotFound`; the owner replaces it with the right count |
| rv4-FR-F101, F120 (agent half) | reproduced by the scratch test described above | unchanged by design, pending the owner's answer |

## Verification

Run against a private PostgreSQL 18 cluster on a port of its own (roles
`insideout_owner` and `insideout_app`, the same two the project uses), migrated
from this worktree, with role passwords that exist only for the run, under a
scrubbed environment. No live server was started.

- `go build ./...` and `go vet ./...` clean; `gofmt -l internal cmd` prints
  nothing.
- `go test ./internal/store/... ./internal/api/... -race -count=1`: both ok.
- A verbose run of the new tests and their neighbours (`TestRvExpand`,
  `Roadmap`, `Expand`, `Build`, `ReplaceRoadmapTree`, `AgentScope`,
  `AnAgentToken`): 44 top-level tests passed, 0 failed, 0 skipped.
- Every package outside store and api that names the two routes or the two store
  functions passes: `go test ./cmd/... ./internal/apiclient/...
  ./internal/mcpserver/... -count=1`. The whole server suite was not run.
- Every `scripts/tests/check_*.py` (14) exits 0; `sh scripts/check-redaction.sh`
  prints nothing; `sh scripts/check-docs-recorded.sh` is quiet; no Go file this
  branch touches is over 350 lines (the largest is `roadmap_ai.go` at 303).
