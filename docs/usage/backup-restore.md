# Backup and restore

The product's value is stored member work — PRDs, revisions, facts,
roadmaps, evidence. This is the recovery story (CC-03, review 2).

## Where the data lives

The hosted deployment's database is a managed Postgres (Supabase) reached
through the app-role DSN in `DATABASE_URL`; the product only ever writes
the `insideout` schema. Two layers of protection:

1. **Managed backups (operator toggle).** Supabase takes automated daily
   backups on paid plans and exposes point-in-time restore. The operator
   should confirm this is enabled for the project — it is not something
   this repository can see or set.
2. **Portable dumps (this repo).** `scripts/backup-db.sh` produces a
   plain-SQL dump of the `insideout` schema that restores on any Postgres
   with the two roles present.

Honest state: nothing schedules the dump — no cron entry, no CI job, the
repo carries neither — and the Supabase toggle above is unconfirmed, so
recovery today rests on the operator remembering three duties:

- run the dump on a cadence you choose, at minimum after every migration
  lands on the hosted database (the last verified dump and drill are the
  2026-10-10 walk below);
- confirm once, in the Supabase dashboard, that automated backups and
  point-in-time restore are enabled for the project — until then treat
  the portable dump as the only protection;
- re-walk the restore drill below whenever this runbook changes, and
  record the walk's date there.

## Taking a dump

```sh
DATABASE_OWNER_URL=<owner-role DSN> ./scripts/backup-db.sh            # → backups/insideout-<stamp>.sql
DATABASE_OWNER_URL=<owner-role DSN> ./scripts/backup-db.sh /secure/dir # elsewhere
```

It needs the **owner** DSN. The app role is an RLS subject, so `pg_dump`
against it stops at the first policy-protected table ("query would be
affected by row-level security policy"); the owner role bypasses RLS and
exists for schema-level work like this.

The script never prints the DSN. Dump files contain user data: keep them
on the operator's disk, never in the repository (`backups/` is ignored).

## Restoring

Prerequisites: a target Postgres with the `insideout_owner` and
`insideout_app` roles (see `docker/postgres-init/`), and the dump file.

```sh
psql "$TARGET_OWNER_DSN" -v ON_ERROR_STOP=1 -f backups/insideout-<stamp>.sql
```

The dump carries `--clean --if-exists`, so it replaces an existing
`insideout` schema in the target. After restoring, run
`go run ./cmd/insideout migrate` (a no-op when the dump is current) and
point `DATABASE_URL` at the target.

## Verification walkthrough (scratch stack)

The runbook was walked end-to-end on a disposable compose stack
(2026-10-04): dump from one scratch database, restore into a second
scratch database, `migrate` as a no-op, and the store suite green against
the restored target. Record the date of the next walk here when the
runbook changes.

Second walk, 2026-10-10 (PostgreSQL 18, scratch cluster, bare target
database): dump with the owner DSN (244 KB), restore with the `psql` line
above, `migrate` a no-op, 23 tables, and the store and api suites green
against the restored database with no skips. The same walk showed the
earlier "Taking a dump" block was wrong — it named the app-role DSN, which
`pg_dump` rejects — and the block now names `DATABASE_OWNER_URL`.
