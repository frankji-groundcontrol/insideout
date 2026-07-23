# SECURITY DEFINER, same-owner FORCE RLS, and textual policy merging

**Date**: 2026-07-21

## What was learned

Three interlocking, non-obvious PostgreSQL RLS behaviors:

1. **`SECURITY DEFINER` does not bypass `FORCE ROW LEVEL SECURITY` when the
   function owner is the same role as the table owner.** A self-referential
   membership policy ("is the current user also a member of this row's
   workspace?") wrapped in a `SECURITY DEFINER` helper only defeats the
   *static* recursion detection (SQLSTATE 42P17). Once real rows exist, the
   function's own query re-triggers the same forced policy, which calls the
   function again — genuine unbounded runtime recursion (`stack depth limit
   exceeded`, SQLSTATE 54001). An empty table never reaches this path, which
   is why the wrong fix looks like it works.
2. For a **single-role, owner-only model** the working fix is `NO FORCE ROW
   LEVEL SECURITY` on the self-referential table: the owner's normal
   (non-forced) bypass applies, application-level transactional checks keep
   enforcing the rules, and the policies stay defined as documentation and
   for any future lower-privileged role.
3. **Policy expressions are merged textually into the application query's
   scope.** A policy subquery alias (e.g. `m`) collides with an application
   query that aliases the same table the same way (SQLSTATE 42702,
   "column reference is ambiguous"); a *bare* column name passed as a policy
   function argument is likewise ambiguous whenever the outer query joins
   another table sharing that column name. The fix that composes safely:
   move every cross-table policy check into a helper function (no alias
   leaks into the outer query) and pass every argument in fully
   table-qualified form (`insideout.projects.workspace_id`), which Postgres
   rebinds correctly regardless of outer aliasing.

## Evidence

[BUG-007, items 2, 3, and the "alias collision" bonus](../issues/2026-07-20-bug-007-rls-against-real-postgres.md)
— all surfaced only by running migrations and integration tests against a real
PostgreSQL database with real rows.

## Scope

Any PostgreSQL schema using RLS, especially: self-referential membership
tables, `FORCE ROW LEVEL SECURITY`, single-role deployments where the
connecting role owns the tables, and any policy whose expression contains a
subquery or function call.

## When to apply again

- Writing a policy on a table that must query itself: do not reach for
  `SECURITY DEFINER` as the recursion fix if function owner == table owner
  under FORCE; either un-force (owner-only models with app-level enforcement)
  or introduce a genuinely distinct owner role.
- Writing any cross-table policy: use a helper function rather than an inline
  subquery, and fully table-qualify every argument.
- Testing RLS: an empty table proves nothing about recursion — seed real rows
  before declaring a policy correct.
