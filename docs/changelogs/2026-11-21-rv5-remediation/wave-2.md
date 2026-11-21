# Wave 2 · The task tree for roles

Rows 2.1 to 2.9 of the
[remediation plan](../../review/2026-11-21-focused-review-5/40-remediation-plan.md),
built to the [target design](../../review/2026-11-21-focused-review-5/41-task-tree-target-design.md)
and the owner's four decisions: each person's tree is derived from the nodes
they own; that tree is a default view, not a wall; a delegate re-delegates
descendants only, never the node they were handed; an agent token may not
change an owner. The slice ran in stacked stages: the write rule and reads
first (tree-core), then delegation, the CLI and MCP surfaces and the web on top
of it, then a follow-up stage that closed what the verifiers left as
should-fix. Each worker ran in its own git worktree with a failing test first;
an adversarial verifier re-ran every gate and probed each row. The coordinator
cherry-picked the accepted branches onto `rv5/remediation` and re-ran every
gate on the integrated tree. The concept itself is described in
[the roadmap tree page](../../architecture/roadmap-tree.md).

## Rows

| Row | IDs | Disposition | Commits |
| --- | --- | --- | --- |
| 2.1 | TREEB-03, TREEX-06, TREEX-05, TREEB-06 | FIXED: `GET /projects/{pid}/roadmap?root=` returns a subtree with its `ancestors`; `GET /roadmap/{nid}` reads one node; `agent-context --focus` keeps to the focus subtree and answers 404 for an unknown node | `167f9f6`, `0382ac8`, `2e22200`, `6180b31` |
| 2.2 | TREEB-02, TREEX-01, TREEB-07, STRS-05, TREEX-07, FE-05 | FIXED: one SQL rule, `insideout._can_write_node`, used by `requireNodeWriter` and by the row-level-security policies on `roadmap_nodes`; RLS keeps a node in its parent's project and refuses a project owner who left the workspace; node views carry `canWrite`, project views `canWritePlan`; the web shows write items only where they are allowed | `51e6a49`, `43b121c`, `5820f5e`, `4d9e8d5`, `2c8ba90`, `03c6a1f`, `272fd32`, `25ef18b`, `14a0c50`, `05afb34`, `2483e01`, `676b1ac`, `03d9885` |
| 2.3 | PARC-01, TREEX-04, PARC-02 | FIXED: owner and acceptance on the Go client, the CLI (`--owner`, `--acceptance`) and MCP `roadmap_update`; CLI `members`, `roadmap get`, `roadmap list --root`, `roadmap mine`, `roadmap delegate`, `roadmap handback`; MCP reads for the same; delegation stays human-only (decision 4), so MCP has no delegate tool | `41c4a0b`, `8da5b62`, `7054896` |
| 2.3 | COMP-03 | FIXED: an agent token may not change a node's owner | `ef6c978`, `a76394c` |
| 2.4 | TREEX-02, FE-03, TREEX-08, TREEX-11 | FIXED: route `/projects/:id/roadmap/:nodeId` with a breadcrumb; tapping a card opens it; "Open as tree", "Delegate to" and "Hand back" in the node menu; a read-only node detail | `a423df5`, `0ecf752`, `676b1ac`, `03d9885`, `8b88495`, `99c35d5`, `6c122a3` |
| 2.5 | TREEB-04, TREEX-03 | FIXED: `GET /workspaces/{id}/roadmap/mine` lists the caller's topmost owned nodes; a Mine view on the roadmap page and My work on the signed-in landing | `f4f88b1`, `be469df`, `b9bf1a1`, `ee39a77`, `0b13b80` |
| 2.6 | TREEB-01, TREEB-09, TREEX-08 | FIXED: append-only `roadmap_node_assignments`; `POST /roadmap/{nid}/delegate` and `/handback` (human tokens only); `GET /roadmap/{nid}/assignments`; `project_updates` rows `node_delegated`, `node_handed_back` and `node_done` | `7f80bb0`, `05fbe7d` |
| 2.6 | TREEB-10 | FIXED (with wave 1): a subtree with evidence, done or validated nodes deletes only with `force` (409 `roadmap_delete_has_history` otherwise); a forced discard writes a project update | `eee05d2`, `3d2284a` |
| 2.7 | TREEB-08, TREEX-09, TREEX-10, FE-11 | FIXED: every node view carries `rollup` (leaves, done, validated, earliest open deadline); progress takes `?root=`; the canvas folds a branch; card bars read the server roll-up, so a validated child counts as finished | `cf0b46f`, `883bf91`, `ee39a77`, `0b13b80` |
| 2.8 | TREEB-05 | FIXED: an agent token may carry `rootNodeId` and writes only inside that subtree; MCP reads `INSIDEOUT_AGENT_ROOT` | `2dad94a`, `a1c2891` |
| 2.9 | TREEB-12, FEAT-05 | FIXED: leaving a workspace and erasure clear node ownership; starting a leaf with no owner claims it for the caller | `1e05f68`, `0158052`, `25ef18b`, `14a0c50` |

The follow-up stage closed what the stage verifiers left as should-fix:

| Area | Change | Commits |
| --- | --- | --- |
| Web | A delete over history states what will be lost (nodes, evidence, done, validated) and forces only when the person confirms; deleting the opened root opens its parent; Mine and the owner label count an owned ancestor ("via" its owner); the fold chevron is the house icon button; the roll-up bar has a screen-reader label | `3123fde`, `2b82ed1` |
| Server | `PUT /projects/{pid}/repo` answers the computed `canWritePlan` | `3df2768`, `1ff9a2f` |
| Go client, CLI | `roadmap delete --force` | `5e88573`, `08c44ec` |
| Server | Ancestors on `GET /roadmap/{nid}` and in the Mine view carry `ownerId`; taking on an ownerless leaf reads as taking it on, not as a delegation; a build records the seeded leaf's owner in the assignment history | `d9c603a`, `d685590`, `b9d4a2f`, `1d6bf86`, `8e3f6d4`, `1301a6f` |
| Agent scope | The agent root check answers 404 for a node that does not exist | `a11e91f`, `18b4305` |
| MCP | The tools an agent token can never use (`members`, delegate, hand-back) are not registered; a root set in `INSIDEOUT_AGENT_ROOT` without an agent token is reported, not silently ignored | `0a80664`, `6b99837`, `b1e4863` |

## Gates on the integrated tree

- `go build`, `go vet`, `gofmt`: clean.
- `go test ./... -count=1` without a database and on a freshly migrated private
  PostgreSQL 18 database: every package ok, no skipped tests.
- `flutter analyze`: 8 infos (baseline 8); `dart format --set-exit-if-changed`:
  no change; `flutter test`: 618 passed.
- Every `scripts/tests/check_*.py` and `scripts/check-redaction.sh`: exit 0.

## Left open

- Proposals stay plan-writer only (BUG-020), so a delegate's agent cannot
  file one; recorded as an owner question in
  [the open findings](../../issues/2026-11-21-review-5-open-findings.md).
- In an opened tree, the items whose rule needs the parent (move to another
  parent, delegate the opened root) show only from the parent's tree.
- The agent root check reads the node's ancestry in its own transaction,
  before the write's transaction (marked in the code as a shortcut).
- An MCP process that holds no login still registers tools it can never use
  (`roadmap_delete` among them); that is PARA-07, in wave 3.
