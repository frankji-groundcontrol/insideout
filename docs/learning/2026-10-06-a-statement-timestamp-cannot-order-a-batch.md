# A statement timestamp cannot order a batch

**Date**: 2026-10-06

## Lesson

`now()` and `created_at` are one value for every row a single statement
inserts. Ordering a batch by `created_at` and breaking ties on a random
`id` does not preserve the order the caller handed in. If the order is
part of the meaning, store it as its own column at insert time.

## Evidence

Applying a proposal reordered Intake, Verification, Handoff, Archive
into Intake, Handoff, Verification, Archive
([BUG-019](../issues/2026-10-06-bug-019-applied-proposal-reorders-items.md)).
The items were inserted in a loop inside one transaction, and
`created_at` defaulted to `now()`, so all four tied and the `id`
tie-break shuffled them. The same fact is already noted on the project
timeline, which paginates on `(created_at, id)` precisely because rows
written in one transaction share a statement timestamp.

## When it applies

Any insert of several rows whose sequence matters: proposal items,
ordered steps, a batch import. Give the row an explicit position.
Do not rely on `created_at`, and do not rely on `id` unless the id
itself was assigned in order.
