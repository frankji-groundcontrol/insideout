# Practice: Verify database-dependent work against real PostgreSQL

**Date**: 2026-07-21

## Trigger

Any change that touches migrations, RLS policies, SQL text, pgx parameter binding, or store-layer authorization logic. "It compiles and unit tests pass" is not a completion signal for this class of work.

## Sequence / guardrail

1. Apply migrations to a real database: `go run ./cmd/insideout migrate` (from `server/`, with `DATABASE_URL` set).
2. Run the DATABASE_URL-gated integration suite:
   ```bash
   cd server
   DATABASE_URL=postgres://insideout_app:<password>@<host>:5432/<db> \
     go test ./internal/store/... -run TestAuthz -v
   ```
   The suite (`server/internal/store/authz_test.go`) walks the full authorization checklist — including deny paths (non-member reads, non-admin mutations, cross-user conversation access) — against real PostgreSQL. It self-skips when `DATABASE_URL` is unset, so a green CI run without the variable proves nothing.
3. If the change involves pooled connections, also run it through a transaction-pooling URL (`pgbouncer=true`), which switches pgx to the simple query protocol (`server/internal/store/pool.go`) and has its own failure modes.
4. Only then mark the item done.

## Verification

All `TestAuthz_*` tests pass **and were actually executed** (look for `PASS`, not `SKIP`, in the `-v` output). Deny-path assertions matter as much as allow-path ones.

## Failure signals

- Test output shows `SKIP` because `DATABASE_URL` was unset.
- Errors that only exist at Postgres runtime: `infinite recursion detected in policy` (42P17), `stack depth limit exceeded` (54001), `column reference is ambiguous` (42702), `permission denied for database`, silent zero-row results from `FOR UPDATE`/`FOR KEY SHARE` under cross-table RLS, jsonb encode failures under the simple query protocol.
- Five distinct real bugs in this repo were invisible to `go build`, `go vet`, and pure-Go unit tests and surfaced only in this suite's first real-database run.

## Related

- [BUG-007: five bugs found only against real PostgreSQL](../issues/2026-07-20-bug-007-rls-against-real-postgres.md)
- [BUG-008: shared-instance DB provisioning](../issues/2026-07-20-bug-008-shared-instance-db-provisioning.md)
- [Learning records](../learning/README.md)
