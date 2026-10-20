# Completion needs evidence or a sentence

Moving a roadmap node from `in_progress` to `done` now requires either at
least one linked `roadmap_evidence` row or a one-sentence
`completion_reason` stored on that update. The check lives in the store
(`requireDoneEvidenceOrReason`), so the API, CLI and MCP all meet it.
Other status transitions are unchanged. A reason left on the row from an
earlier completion does not cover a later close after the node is reopened.

The web status control asks for the sentence when the node has no evidence,
and sends it as `completionReason`. CLI: `roadmap update --completion-reason`.
MCP: `roadmap_update` argument `completion_reason`.

## Operator

Apply migration `20261020150000_roadmap_completion_reason.sql` before
serving this server. It adds `insideout.roadmap_nodes.completion_reason`
(`text NOT NULL DEFAULT ''`). Existing done nodes are left done.

## Verification

Private Postgres 18 on `127.0.0.1:55471`, roles `insideout_owner`
(BYPASSRLS, not superuser) and `insideout_app` (NOBYPASSRLS). Migrated
from this worktree, then torn down.

Red, before the rule (`TestRoadmapCompletion_RefusesDoneWithoutEvidenceOrReason`):

```
in_progress → done with no evidence and no reason: err = <nil>, want ErrValidation
--- FAIL
```

Green, after the rule: that test passes, including a one-sentence reason
(stored and listed) and a node with evidence and no reason. Related roadmap
store and API tests, `go build ./...`, `go vet` on the touched packages,
and the Flutter status-control tests passed.

Record: [completion override](../issues/2026-10-18-promise-partial-completion-override-reason.md).
Plan: [2026-10-20](../plans/2026-10-20-completion-evidence-or-reason/2026-10-20-completion-evidence-or-reason.md).
