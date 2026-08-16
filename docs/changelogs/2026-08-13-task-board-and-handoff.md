# 2026-08-13 — Task board and handoff responsibility correction

The planning index and handoff had their responsibilities reversed:
`docs/HANDOFF.md` had grown into a chronological history with internal task
codes, while `docs/plans/README.md` claimed no task was active and did not show
several finished-but-uncommitted workstreams.

## What changed

- Rebuilt [`docs/plans/README.md`](../plans/README.md) as the authoritative
  concurrent task board. Each current task now has an explicit priority,
  status, next action, blocker, and plan or record link.
- Distinguished **Finished — awaiting checkpoint** from historical Completed,
  so local work is not reported as landed before its commit exists.
- Replaced [`docs/HANDOFF.md`](../HANDOFF.md)'s history with a short resume
  guide: the next objective, the smallest recommended starting slice, current
  dirty-worktree boundaries, and non-negotiable constraints.
- Removed unexplained implementation codes from the handoff; their detail
  remains in the plans and changelogs that own it.
- Updated `AGENTS.md`, `CLAUDE.md`, and the documentation map to preserve the
  split: the plans index owns multi-task status, while the handoff owns one
  human-readable resume path.
- Corrected the completed hardening plan's stale `in progress` header.

## Verification

- Audited every task listed in the plans index against its plan body and the
  current dirty worktree.
- Checked that all current dirty workstreams appear on the board and that no
  completed historical plan is labeled in flight.
- Checked relative Markdown links and `git diff --check`.
- Documentation only; no application behavior changed.

## Operator notes

Update the board whenever a task's status, priority, next action, or blocker
changes. Keep the handoff short and link back to the board and individual plans
instead of copying their history.

