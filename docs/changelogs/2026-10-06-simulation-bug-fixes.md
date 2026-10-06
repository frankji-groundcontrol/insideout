# 2026-10-06 — The four simulation bugs are fixed

## What changed

The four bugs the live user simulation found
([plan](../plans/2026-10-06-close-simulation-bugs.md)):

- A roadmap move shifts its siblings, so two nodes no longer share a
  position
  ([BUG-017](../issues/2026-10-06-bug-017-roadmap-move-collides-sibling-position.md)).
- The progress view translates `normal` pressure to "normal" / "正常"
  ([BUG-018](../issues/2026-10-06-bug-018-roadmap-pressure-label-untranslated.md)).
- Proposal items record the order they were given, and applying them
  keeps it. Migration `20261006120000` adds `proposal_items.position`
  ([BUG-019](../issues/2026-10-06-bug-019-applied-proposal-reorders-items.md)).
- Only the project owner or a workspace admin may add, edit, move, or
  delete roadmap nodes, or file a proposal. A member still reads and
  leaves notes. A decided proposal no longer offers Accept and Reject
  ([BUG-020](../issues/2026-10-06-bug-020-workspace-member-writes-the-plan.md)).

`server/internal/store/roadmap.go` is now 403 lines. The split is
recorded, not done
([issue](../issues/2026-10-06-roadmap-go-line-budget.md)).

## Verification

`TestRoadmap_MoveShiftsSiblings`, `TestProposal_ApplyKeepsItemOrder`,
and `TestRoadmap_MemberCannotWrite` failed against the sim database,
then passed with the rest of `TestRoadmap` and `TestProposal`. The
roadmap widget test renders "normal" and not the raw key.

Against the restarted sim server: a member's add and propose both
returned 403, moving the collided node left positions 0, 1, 2, and the
rejected proposal came back with `decision: rejected`.

## Operator note

Deployments pick up the new column on the next server start, which
runs migrations. The release web bundle already running on :5174
predates the two client changes (the pressure word, and hiding
Accept/Reject), so it needs a rebuild to show them.
