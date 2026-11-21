# Per-Project roles have no implementation

Open, recorded 2026-11-21 (focused review 5, finding FEAT-04; plan row 5.6 of
the [remediation plan](../review/2026-11-21-focused-review-5/40-remediation-plan.md)).
Product scope, not a defect: the owner sets the priority. Until now the gap was
only conceded in PRODUCT.md and listed as "not re-tracked here" in the
[promise-gap enumeration](./2026-10-11-product-promise-gaps.md).

## The promise

PRODUCT.md, "Collaboration and authority":

> Workspace roles manage organization access. Product authority is assigned per
> Project, because the same person may play a different part on different work:
> Decision Owner, Driver, Contributor, Reviewer, Viewer.

PRODUCT.md's status section concedes that "the running v1 authority model is
still workspace-scoped" and that the per-Project roles "are the target, not the
current behavior". The promise register stamps the row
[DOCS-ONLY](../review/2026-10-10-multirole-3/50-promise-register.md).

## What exists (2026-11-21, after wave 2 of the review 5 remediation)

- **Workspace roles.** `workspace_memberships.role` is `admin` or `member`
  (a CHECK in the migrations). Every project authority check starts there.
- **Plan writers.** `requirePlanWriter` in `server/internal/store/authz.go`
  admits the project's `owner_id` or a workspace admin. Only a plan writer
  changes the project row, builds the plan or files a proposal
  ([BUG-020](./2026-10-06-bug-020-workspace-member-writes-the-plan.md)).
- **PRD authority.** The PRD author or a workspace admin edits a PRD and
  commits a version (`requirePrdAuthorOrAdmin` in `server/internal/store/prds.go`,
  the author check in `prd_commits.go`); only an admin approves or rejects it.
- **Node owners.** Wave 2 built the task tree for roles: a node's
  `owner_user_id` may write the node and everything below it
  (`requireNodeWriter`, `insideout._can_write_node`) and may re-delegate the
  nodes strictly below it (`requireNodeManager`), per the owner's decisions 1
  to 4 in the [review 5 open findings](./2026-11-21-review-5-open-findings.md).
  This is per-node authority, not a per-Project role.

## What is missing

- No storage for a per-Project role: no `project_members` table or role
  column anywhere under `server/db/migrations` or `server/internal`.
- No Decision Owner distinct from the Driver: the project owner and any
  workspace admin hold every power the five roles split.
- No Contributor who may propose without writing the plan, and no Reviewer
  who may comment or request changes without editing; a plain member can only
  read and leave notes, or propose about a node they may write (owner decision 7
  in the review 5 record narrowed this for delegates only).
- No Viewer narrower than workspace membership: every member reads every
  project in the workspace.
- No transfer of the Driver role, because there is no Driver role.

## Fix prompt (bounded)

First slice, server only, test first on real PostgreSQL:

1. Add migration `insideout.project_members (project_id, user_id, role)` with
   `role IN ('decision_owner','driver','contributor','reviewer','viewer')`, a
   primary key on `(project_id, user_id)`, RLS that lets workspace members
   read and plan writers write, and a backfill that makes each project's
   `owner_id` its `driver` and `decision_owner`.
2. Teach `requirePlanWriter` to admit a `driver` or `decision_owner` row, and
   add a `requireProposer` that admits a `contributor` as well; use it for
   `agent_proposal` in `addProjectUpdateTx` (`project_updates.go`) while the
   decision stays with the Decision Owner or an admin.
3. Red tests first: a contributor files a proposal but cannot add a node; a
   reviewer can do neither; a driver writes the plan; removing a member drops
   their project role.

Out of the first slice: client surfaces, the Viewer narrowing, Product Branch
Merge (its own [record](./2026-10-11-product-branch-merge-decision-log.md)), and
Driver transfer. Decide with the owner whether node ownership stays a separate
axis (recommended: yes, roles say what a person may do, node ownership says
where).
