# 2026-10-06 — Close the four simulation bugs

**Goal:** fix the four bugs the live simulation found, each with a
failing test first, and leave the simulated projects' data alone.

- [BUG-017](../issues/2026-10-06-bug-017-roadmap-move-collides-sibling-position.md)
  — moving a node onto a sibling's position must shift the others, so
  no two siblings share a position.
- [BUG-018](../issues/2026-10-06-bug-018-roadmap-pressure-label-untranslated.md)
  — the progress view must say "正常" / "normal", not print the key
  `roadmap.pressure.normal`.
- [BUG-019](../issues/2026-10-06-bug-019-applied-proposal-reorders-items.md)
  — applying a proposal must create its nodes in the order they were
  proposed.
- [BUG-020](../issues/2026-10-06-bug-020-workspace-member-writes-the-plan.md)
  — a plain member may read, but not add, edit, move, or delete roadmap
  nodes, and not file a proposal. The owner and a workspace admin still
  can. The web stops offering Accept/Reject once a proposal is decided.

## Checklist

- [x] Plan + board row.
- [x] Failing tests for 017, 019, 020 (real Postgres) and 018 (widget).
- [x] Smallest change that makes each pass.
- [x] Store tests green; the widget test green.
- [x] Issues marked fixed, changelog, indexes, handoff.
      `roadmap.go` crossed 350 lines; the split is filed, not done.

## Decisions

- Shift siblings in the same transaction as the move, under the
  per-project advisory lock that move already takes. Positions stay
  dense: the gap closes and the target slot opens.
- Give `proposal_items` an explicit `position` set at insert, and order
  by it. `created_at` is `now()` per statement, so a batch of inserts
  ties and the tie-break reorders them.
- Authority follows the decision gate that already exists: project
  owner or workspace admin. A member keeps read access.
