# Review 5 open findings

Open, recorded 2026-11-21. Source:
[focused review 5](../review/2026-11-21-focused-review-5/README.md) over
`main` @ `4d186da` plus its uncommitted files. This record lists the High
findings and the owner's decisions; the 81 Medium and 25 Low rows stay in the
[register](../review/2026-11-21-focused-review-5/01-findings-register.md) and
are ordered in the
[remediation plan](../review/2026-11-21-focused-review-5/40-remediation-plan.md).
Delete a row here when its fix lands, and name the commit in the changelog.

## The task tree for roles is one quarter built

The owner's requirement (each person has their own work tree; a node can be
delegated and opens as that person's view; a manager can open any node) is
written in no product or architecture record and is mostly absent from the
code. Target design and four decisions:
[41-task-tree-target-design.md](../review/2026-11-21-focused-review-5/41-task-tree-target-design.md).

| ID | Finding | Where | Plan row | Priority | Owner |
| --- | --- | --- | --- | --- | --- |
| TREEB-02, TREEX-01 | The person a node is assigned to has no write right over it or anything under it (reproduced on PostgreSQL) | `server/internal/store/memberships.go:236` | 2.2 | P0 | unassigned |
| TREEX-02, FE-03 | A node cannot be opened as its own tree | `client/lib/router.dart:43` | 2.1, 2.4 | P0 | unassigned |
| TREEB-04, TREEX-03 | There is no "my tree" read, filter or landing | `server/internal/store/roadmap.go:201`, `client/lib/features/dashboard/dashboard_page.dart:118` | 2.5 | P0 | unassigned |
| PARC-01, TREEX-04 | The node owner can be set only from the web; MCP and CLI cannot delegate | `server/internal/apiclient/roadmap.go:82` | 2.3 | P0 | unassigned |
| TREEX-05 | The agent briefing called focus-scoped returns every leaf of the project | `server/internal/agentcontext/agentcontext.go:111` | 2.1 | P1 | unassigned |

## The build path writes before it checks

| ID | Finding | Where | Plan row | Priority | Owner |
| --- | --- | --- | --- | --- | --- |
| FEAT-01, PARB-02 | The Build draft call creates a project; a non-author member's builds create unlinked projects (reproduced) | `server/internal/store/roadmap_plan.go:242` | 1.2 | P0 | unassigned |
| FEAT-03 | A rebuild erases every evidence row and validation record (reproduced) | `server/internal/store/roadmap_plan.go:58` | 1.5 | P0 | unassigned |
| FEAT-02 | An agent token may replace the whole roadmap | `server/internal/api/agent_scope.go:79` | 1.4 | P0 | unassigned |
| PARB-01 | The approval gate before Build is a web button only | `server/internal/api/roadmap_draft.go:50` | 1.3 | P1 | unassigned |

## Other Highs

| ID | Finding | Where | Plan row | Priority | Owner |
| --- | --- | --- | --- | --- | --- |
| IDX-02, COMP-01 | The pushed tip is red in three CI jobs: the docs truth gate, the expand response-shape test, the analyzer baseline (reproduced) | `docs/architecture/backend.md:147`, `server/internal/api/testdata/roadmap_expand_confirm.json` | 0.1 | P0 | unassigned |
| PARA-02 | A title-only idea update from CLI or MCP erases the idea body (reproduced) | `server/internal/apiclient/crud.go:24` | 1.1 | P0 | unassigned |
| FE-01 | The coach transcript does not update during a session | `client/lib/features/prd/coach_panel.dart:112` | 1.7 | P0 | unassigned |
| FE-02, PARB-03 | Nobody can find a PRD they did not create | `server/internal/api/projects.go:26` | 3.3 | P1 | unassigned |
| PARB-04 | `SKILL.md:107` and `README.md:35` promise every verb on every surface; projects beyond list and create are web-only | `SKILL.md:107` | 3.4, 3.5 | P1 | unassigned |

## The uncommitted slice

The install front door and password-reset work (35 modified and 22 untracked
files) has no plan and no board row, and leaves five tests and two contract
gates red. Its closing checklist is plan row 0.4 (`PARA-01`, `PARA-08`,
`PARA-10`, `STRC-01`, `FE-06`, `IDX-09`, `COMP-04`, `COMP-06`).

## Decisions only the owner can make

1. Is "own work tree" per person or per role?
2. Is a delegate's tree a default view or a wall?
3. May a delegate re-delegate inside their subtree?
4. May an agent token change a node's owner (`COMP-03`)?
5. May a PRD be built before it is approved (`PARB-01`)?
6. May an agent token write the completion override reason (`FEAT-02`)?

Recommended answers for 1 to 4 are in the
[target design](../review/2026-11-21-focused-review-5/41-task-tree-target-design.md).

**Decided 2026-11-21 by the owner** (answers given when the remediation was
started; execution plan:
[2026-11-21-rv5-remediation](../plans/2026-11-21-rv5-remediation/2026-11-21-rv5-remediation.md)):

1. Per person: each person's tree is derived from the nodes they own; roles
   govern what a person may do, not which tree they see.
2. A default view, not a wall: everyone can still read the whole project.
3. A delegate may re-delegate descendants, not the node they were handed.
4. An agent token may not change a node's owner; it files a proposal.
5. No: the server refuses to build a roadmap from a PRD that is not approved,
   on every surface.
6. No: only a human may mark a node done with a reason instead of evidence;
   an agent may still mark done when evidence exists.
