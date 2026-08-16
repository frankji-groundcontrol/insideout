# 2026-08-13 — Task board and handoff responsibility correction

Status: complete; awaiting checkpoint.

## Goal

Restore the documentation responsibilities expected by maintainers:

- `docs/plans/README.md` is the authoritative multi-task board. It records
  what is in flight, pending, finished but awaiting checkpoint, completed, and
  superseded.
- `docs/HANDOFF.md` is a short resume guide. It names the next objective,
  explains where to start, and warns about current worktree constraints without
  repeating plan history or internal codenames.

## Scope

- Audit every task currently represented by the dirty worktree and plans
  index.
- Rebuild the task board around explicit states and next actions.
- Replace the chronological handoff with plain-language resume instructions.
- Make the responsibility split durable in the agent routers and doc map.
- Correct plan-status drift found during the audit.

No product or runtime behavior changes are part of this task.

## Checklist

- [x] Audit plan bodies, recorded verification, and the dirty worktree.
- [x] Rebuild `docs/plans/README.md` as the concurrent task board.
- [x] Rewrite `docs/HANDOFF.md` as the concise resume guide.
- [x] Update the agent routers, doc map, and changelog.
- [x] Review task states, links, readability, and diff scope.

## Review

This is a documentation-governance correction, so engineering review is not
applicable. Two independent read-only audits covered task-state accuracy and
handoff readability before editing.
