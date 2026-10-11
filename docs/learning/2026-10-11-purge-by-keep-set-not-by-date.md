# Purge by an enumerated keep-set, never by a date rule

## What was learned

"The 582 legacy users" from a board row grew into 864 accounts, and a
date-based purge rule ("delete everyone created before the invitation
cutover") would have deleted **the dogfood account that owns the GitHub
repo binding** and could have deleted the owner's admin account — both
predate the cutover. The junk itself defied the expected shape: 840 of the
864 were `@test.local` fixtures a test run had once written against
production, spanning both eras, so "legacy by date" was wrong in both
directions.

The safe procedure:

1. **Census before delete** — total by era, by email domain, by ownership
   (workspaces/projects/ideas), by last session. The decision table of
   every survivor-candidate (masked) is what the operator actually
   approves.
2. **Enumerate the keep-set by identity, not by date** — admins, accounts
   owning live integrations, accounts with recent sessions, documented
   rehearsal accounts. Everything not kept is purge; the date never
   appears in the predicate.
3. **Snapshot the schema first**, one transaction for the delete,
   FK-ordered (this schema cascades sessions/memberships/evidence but not
   ownership), and verify the app is healthy immediately after.

## Evidence

The 2026-10-11 purge ([changelog](../changelogs/2026-10-11-owner-deploy-and-mail-truth.md)):
census found 864 = 840 fixtures + 19 dead old-era accounts + 5 real; the
keep-set (admin, dogfood binding owner, active demo, two rehearsal
accounts) was enumerated explicitly; 859 users, 446 workspaces and 214
projects went in one transaction after a `pg_dump --schema=insideout`
snapshot; hosted probes stayed 6/6. The shared Supabase instance also
reminded the other half of the rule: the snapshot must be
schema-scoped — the instance hosts other projects' schemas our role
correctly cannot read.

## Scope

Any production purge of "old" data: build the keep-set by identity,
snapshot first, delete in one transaction, and expect the junk's shape to
differ from the story the board row told.
