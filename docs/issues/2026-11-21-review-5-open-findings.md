# Review 5 open findings

*2026-11-21: every High finding is fixed on `rv5/remediation` (waves 0 to 5, [changelog](../changelogs/2026-11-21-rv5-remediation/index.md)). The owner's last two decisions are at the end of this record; decision 7 is built as a follow-up.*

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
delegated and opens as that person's view; a manager can open any node) was
written in no product or architecture record and is mostly absent from the
code. Wave 0 of the remediation wrote it into PRODUCT.md and
[roadmap-tree.md](../architecture/roadmap-tree.md). Target design and four decisions:
[41-task-tree-target-design.md](../review/2026-11-21-focused-review-5/41-task-tree-target-design.md).
Current state and the decided target:
[architecture/roadmap-tree.md](../architecture/roadmap-tree.md).

*Fixed on `rv5/remediation` in wave 2 ([record](../changelogs/2026-11-21-rv5-remediation/wave-2.md)): TREEB-02, TREEX-01, TREEX-02, FE-03, TREEB-04, TREEX-03, PARC-01, TREEX-04, TREEX-05, and with them the Medium task-tree rows of plan rows 2.1 to 2.9, plus the rest of TREEB-10. The write rule, the reads, delegation and its record, the Mine view, the roll-up and the agent root are described in [roadmap-tree.md](../architecture/roadmap-tree.md).*

## The build path writes before it checks

*Fixed on `rv5/remediation` in wave 1
([record](../changelogs/2026-11-21-rv5-remediation/wave-1.md)): FEAT-01,
PARB-02, FEAT-03, FEAT-02 and PARB-01. TREEB-10 (Medium) was finished in wave 2: a
subtree delete with history needs `force`, and a forced discard leaves a
project update.*

## Other Highs

| ID | Finding | Where | Plan row | Priority | Owner |
| --- | --- | --- | --- | --- | --- |
| IDX-02, COMP-01 | Fixed in wave 0 (`50b7437`): the docs truth gate, the expand response-shape test and the analyzer baseline | `docs/architecture/backend.md`, `server/internal/api/testdata/roadmap_expand_confirm.json` | 0.1 | done | claude |
| PARA-02 | Fixed in wave 1 (`a695b8f`): a title-only idea update keeps the body | `server/internal/apiclient/crud.go` | 1.1 | done | claude |
| FE-01 | Fixed in wave 1 (`7941126`, `458ff76`): the coach transcript keeps every turn | `client/lib/features/prd/coach_panel.dart` | 1.7 | done | claude |
| FE-02, PARB-03 | Fixed in wave 3 (`5cd05c87`, `bd028b73`): `GET /workspaces/{id}/prds` and a PRD list on the workspace page, a project links to its PRD; on the CLI and MCP too | `server/internal/api/prd_list.go` | 3.3 | done | claude |
| PARB-04 | Fixed in wave 3 (`f612ca26`, `4279225a`, `96ea5a53`): project show, update and delete on the CLI, project and workspace tools on MCP; SKILL.md and README state the parity that exists; the parity test (`server/internal/surfaceparity`) keeps it | `SKILL.md` | 3.4, 3.5 | done | claude |

## The uncommitted slice

*Finished and committed in wave 0 (`454ab0b`,
[record](../changelogs/2026-11-21-rv5-remediation/wave-0.md)).*

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

## Owner decisions after the remediation (2026-11-21)

7. **Proposals inside a delegate's subtree: yes.** Anyone who may write a node
   (the delegate, and their agent) may file a proposal about that node or its
   descendants. Only a plan writer decides it, as before. This narrows BUG-020
   and is built as a follow-up to the remediation.
8. **Agent edits of projects and workspaces: keep both.** Agent tokens keep
   `PATCH /projects/{pid}` and `PATCH /workspaces/{id}` as wave 3 built them.
   An owner change stays human-only.
