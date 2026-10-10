# 2026-10-10 — Measure a deferred optimization before shipping it

## Context

The 2026-07-26 backend optimization record deferred two database items on
instinct: a partial index for the stale-run reaper and a rewrite of the
correlated member count in `ListWorkspacesForUser`. Each was seeded with
realistic volume on a scratch PostgreSQL 18 database and measured with
`EXPLAIN (ANALYZE, BUFFERS)` as the application role (so row-level security
applied), before and after:

- **Reaper index** on `ai_runs (updated_at) WHERE status IN ('pending','running')`,
  200k finished runs plus a few in flight: Seq Scan at 16–20 ms became an Index
  Scan at 0.06–0.08 ms. Shipped as a migration.
- **`ListWorkspacesForUser` rewrite** (100 workspaces of 50 members for the
  measured user): the rewritten query was 6.3% faster. Not shipped: a rewrite
  of a security-relevant query for a gain inside run-to-run noise.

## Lesson

The two items looked equally "obviously better" on paper and measured an order
of magnitude apart in opposite directions. A measurement that can say "no" is
what makes the cheap, certain change shippable and the clever one droppable.
Record the number with the decision so the item does not get re-opened on
instinct.

## When to apply

- Before turning a deferred optimization into a change: seed, `EXPLAIN
  ANALYZE` as the runtime role, change, measure again, keep the evidence in
  the issue record.
- Query rewrites that touch authorization-shaped SQL need a gain that is
  clearly outside noise; indexes that only add a read path do not.
