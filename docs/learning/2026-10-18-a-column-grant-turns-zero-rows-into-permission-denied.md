# A column grant turns "RLS hid the row" into "permission denied"

**Date:** 2026-10-18
**Source:** [multirole review 4](../changelogs/2026-10-18-multirole-review-4.md), findings DATA-01 to DATA-03 (the write side of row-level security)
**Fixed in:** [wave 2, RLS](../changelogs/2026-10-18-rv4-remediation/w2-rls.md)
([architecture](../architecture/database-and-rls.md#membership-and-user-writes))

## The lesson

Narrowing what the runtime role may write has two tools, and they answer
differently. A column `GRANT` is checked before any row is read, so an `UPDATE`
of a column the role lacks fails with SQLSTATE 42501 "permission denied" even
when RLS would have hidden every row. A `BEFORE UPDATE` trigger only sees rows
RLS let through, so for everyone else the answer stays "0 rows changed".

Where the empty answer is part of the contract, use the trigger. Here
`promote-admin` run without `DATABASE_OWNER_URL` updates someone else's
`system_role` on the app pool; "no row" is what the CLI turns into its "set
`DATABASE_OWNER_URL`" hint. A column grant would have replaced the hint with a
bare permission error. So `INSERT` on `users` is cut down by column grants (a
new column is refused until someone grants it), and `UPDATE` is limited by the
`users_pin_columns` trigger (a new column stays pinned until someone adds it to
the list of columns a profile edit may change).

## Two traps on the way

- A table-level privilege masks column grants: `GRANT INSERT (cols)` does
  nothing while the role still holds `INSERT` on the table. `REVOKE` the table
  privilege first, which also clears the column grants and makes the migration
  re-runnable.
- A read-only policy walk cannot see a write hole: a policy that lets the wrong
  `INSERT` or `UPDATE` through is invisible to `SELECT` counts. The write twin
  (`TestRLSWriteWalk_*`) runs as the app role in rolled-back transactions and
  reads the column list from the catalog, so a new column fails it until it is
  classified.

## When to apply

- Any column-level restriction on a table that RLS also protects: decide first
  whether "no row" or "permission denied" is the answer callers rely on.
- Any new RLS table or policy: extend the write walk, not only the read walk.

## Evidence

Before the fix a session could insert itself as `admin` of a workspace whose id
it knew, register with `system_role = 'admin'` and rewrite its own `system_role`
(the walk's red run: five membership and twelve `users` cases, "expected
42501, got nil"). After migration `20261018140000` all of them are refused, and
dropping the trigger or widening a grant turns the walk red again.
