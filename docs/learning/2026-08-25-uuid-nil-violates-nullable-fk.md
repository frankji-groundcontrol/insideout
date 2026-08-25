# uuid.Nil does not mean NULL — nullable FK columns reject the zero UUID

**Learned** 2026-08-25 from the invitation-only slice
([changelog](../changelogs/2026-08-25-invitation-only-signup.md)) while
live-exercising invitation registration.

## The lesson

`ConsumeInvitation` claimed an invitation **before** the user existed, so
the handler passed `uuid.Nil` as `used_by` — and Postgres rejected the
claim with `insert or update on table "invitations" violates foreign key
constraint "invitations_used_by_fkey" (SQLSTATE 23503)`. The zero UUID
(`00000000-…`) is a real, non-null value: a UUID FK column that allows
NULL still refuses it, because no users row carries that id.

When a Go caller must express "not set yet" into a nullable uuid FK,
map the sentinel in SQL, not in application branching:

```sql
used_by = NULLIF($3::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
```

## When to apply

Any two-phase write that records an actor/owner reference before that
row exists (claims, reservations, attribution-after-the-fact patterns).
Trust signal: a 500 with `23503` on a column you believe is nullable,
right after introducing a uuid.Nil default.

## Evidence

Valid-code registration 500'd exactly there; after the `NULLIF` fix the
same request returned 201 and the full invitation matrix went green.
