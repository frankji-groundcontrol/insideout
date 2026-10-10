# BUG-023 — bundled Postgres setup never wrote the owner DSN

## Symptom

`insideout setup --local --postgres bundled` (and its shell twin
`scripts/setup.sh`) started the bundled database, wrote `.env`, and then
stopped at the first migration with:

```
migrate: connected as "insideout_app"; use DATABASE_OWNER_URL (insideout_owner). …
```

The run could not finish on a fresh machine without the operator editing
`.env` by hand to add a second connection string they had never been told
about.

## Root cause

The setup flow wrote only `DATABASE_URL`, which points at the least-privileged
app role (`insideout_app`). That role may not apply migrations; the migrator
(`store/migrate_role.go`) requires `DATABASE_OWNER_URL` (the `insideout_owner`
role) and refuses to run as the app role. The flow read `DATABASE_OWNER_URL`
from `.env` only if it was already there, and the bundled branch never wrote
one, so a fresh bundled setup always ended at that refusal.

It stayed unseen because the path needs Docker and neither the 2026-10-07 live
walk nor the later waves had it. The helpers around it (`bundledDSN`) were
covered by pure unit tests, which proved the helpers and not the path. A
reviewer reading the code raised the suspicion; running the nearest real piece
confirmed it.

## Fix

`fdc279f`, test first: `TestLocalBundledNoStartWritesTheOwnerDSN` failed on the
missing value, then the fix. Both the Go flow and `scripts/setup.sh` now write
`DATABASE_OWNER_URL` (`insideout_owner`, `POSTGRES_OWNER_PASSWORD` or its
development default, same host, port, and database), and the Go flow attaches
it before migrating.

## Status

**Fixed 2026-10-10.** Verified live without Docker: a no-op `docker` on `PATH`
and a real PostgreSQL 18 database behind the DSN. The binary built from the
commit before the change fails at the migration with the message above; the
binary built from the fix applies the pending migrations and exits 0. The
own-URL path got the matching treatment in the same wave (roles probe and the
not-ready-database stop), see
[setup roles and the bundled owner](../changelogs/2026-10-10-close-open-gaps/setup-roles-and-bundled-owner.md)
and the [own-URL record](2026-10-07-own-url-fresh-db-bootstrap.md).

Lesson: [a path nobody can run hides a missing write](../learning/2026-10-10-a-path-nobody-can-run-hides-a-missing-write.md).
