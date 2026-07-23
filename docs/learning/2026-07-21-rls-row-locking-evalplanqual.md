# RLS + row locking: FOR UPDATE / FOR KEY SHARE silently returns zero rows

**Date**: 2026-07-21

## What was learned

Adding `FOR UPDATE` / `FOR KEY SHARE` to a `SELECT` against a table whose RLS
policy references *another* table makes the query silently return **zero rows**
for rows that are perfectly visible without the locking clause. No error is
raised — just wrong behavior. This is a genuine PostgreSQL limitation:
the `EvalPlanQual` re-check during row locking does not correctly re-evaluate
a cross-table RLS policy reference. Wrapping the check in a `SECURITY DEFINER`
function does not help; the same query failed identically.

The corollary: the property that actually matters for TOCTOU safety is that
the **check and the write happen in the same transaction** — an explicit row
lock is an extra protection that may simply be unaddable under RLS. In
InsideOut, the lock was removed from all 7 affected store functions; if the
narrower race ever matters, the upgrade path is embedding the same condition
directly in the mutating statement's `WHERE` clause (marked with `// ponytail:`
comments at each site).

## Evidence

[BUG-007, item 4](../issues/2026-07-20-bug-007-rls-against-real-postgres.md) — isolated
by running the identical query with and without the locking clause against a
real PostgreSQL database. Invisible to `go build`, `go vet`, and unit tests.

## Scope

Any PostgreSQL table that (a) has RLS enabled and (b) has a policy whose
expression references another table (directly or via a helper function), when
queried with any explicit row-locking clause.

## When to apply again

- Before adding `FOR UPDATE` / `FOR SHARE` / `FOR KEY SHARE` / `FOR NO KEY
  UPDATE` to a query on an RLS-protected table: check whether the policy
  crosses tables. If it does, expect silent zero-row results — test the query
  with and without the lock against a real database.
- When a check-then-write authorization query "finds nothing" for a row that
  demonstrably exists and is visible: suspect the locking clause before
  suspecting the policy logic.
- When reviewing TOCTOU protection: same-transaction check-then-write is the
  baseline requirement; treat an explicit row lock as optional hardening that
  RLS may veto.
