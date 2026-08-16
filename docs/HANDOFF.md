# HANDOFF

Resume here. The authoritative multi-task board is
[`docs/plans/README.md`](plans/README.md). The target product experience is
[`PRODUCT.md`](../PRODUCT.md), and the running system is described by
[`docs/architecture/`](architecture/index.md).

## What to do next

Resume in this order:

1. Independently rerun the focused verification and checkpoint each of the
   three finished workstreams listed on the task board. Their shared files
   require hunk-level staging.
2. Then create and engineering-review the first implementation plan derived
   from `PRODUCT.md`.

Scope that plan to the smallest version-first slice:

- preserve an Idea's title and body when work begins;
- keep an Idea private to its author until explicitly shared;
- produce an immediately editable first working version;
- reuse the existing PRD revision model;
- prove the flow with one real-PostgreSQL end-to-end check.

Do not implement the whole product baseline in one pass. Before editing, trace
the current conversion and visibility paths in
[`server/internal/store/ideas.go`](../server/internal/store/ideas.go) and the
existing revision path in
[`server/internal/store/prd_revisions.go`](../server/internal/store/prd_revisions.go).

## Worktree to preserve

Three independent tasks are finished but have not been checkpointed. Their
files overlap in shared documentation, so do not mix, revert, or stage them as
one change:

- environment-management workflow;
- shared login/register modal and Design-QA record surface;
- product-experience baseline.

Their exact status, next action, blocker, and plan or record link are on the
[task board](plans/README.md). Review and checkpoint them one task at a time;
do not use `git add .`.

## Non-negotiables

- Never read or expose `.env` or `.env.local`.
- DB-dependent behavior requires a real PostgreSQL check and must stay inside
  the `insideout` schema.
- Commands and environment-safe launch instructions live in
  [`docs/usage/local-development.md`](usage/local-development.md).
