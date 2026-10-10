# `CREATE OR REPLACE` with a changed signature creates an overload

## What was learned

In PostgreSQL, `CREATE OR REPLACE FUNCTION` replaces a function only when
the signature matches. Add or change a parameter and the statement quietly
**creates a second function** alongside the old one — the migration applies
cleanly, the new code calls the new form, and the old zero-arg form keeps
answering. The failure surfaces later and elsewhere: callers that reference
the function **without arguments** (`SELECT ai_check_circuit()`, in dynamic
SQL or default-form call sites) now hit `function is not unique`, a runtime
500 that no migration-time check catches.

When a migration changes a function's parameter list, it must **DROP the old
form first**:

```sql
DROP FUNCTION IF EXISTS insideout.ai_check_circuit();
DROP FUNCTION IF EXISTS insideout.ai_record_circuit_result(boolean);
CREATE FUNCTION insideout.ai_check_circuit(p_id text DEFAULT 'anthropic') ...
```

Because the drop is what makes the migration correct — not the create — a
fresh database is not enough to prove it: a fresh clone replays the fixed
file and passes either way. The proof that matters is calling the function
**by its old argument shape** after migrating (that call must fail to
resolve), or asserting the overload count in `information_schema.routines`.

## Evidence

Wave 3's TEST-06 closure
([changelog](../changelogs/2026-10-11-rv3-remediation/server.md)) changed
`ai_check_circuit`/`ai_record_circuit_result` from zero-arg to row-id
functions via `CREATE OR REPLACE`. The overload survived undetected until
integration, where `ai_check_circuit() is not unique` 500'd agent and cmd
tests across per-cluster databases. Worse, the local clone helper applies
the tree's migrations on top of the migrated base, so the broken file
**poisoned every fresh clone** made after it landed — the bug reproduced on
databases the wave had already verified. The repair: fix the migration with
leading `DROP FUNCTION IF EXISTS`, then repair each affected database by
dropping the overloads, deleting the migration's `schema_migrations` row and
re-migrating, and finally audit every database via
`information_schema.routines` for stray duplicate signatures.

## Scope

Applies whenever a repo-owned SQL migration alters a function's parameters,
especially where callers may reference the bare name. The general rule:
**a migration is state-changing code, and a clone of the already-migrated
base hides migration bugs — test migrations from a base that predates the
change, and beware tools that silently layer the tree's migrations on top of
a migrated copy.**
