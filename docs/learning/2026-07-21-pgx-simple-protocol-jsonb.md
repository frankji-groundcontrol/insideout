# pgx simple query protocol: jsonb parameters need a Go string + explicit ::jsonb cast

**Date**: 2026-07-21

## What was learned

Under `pgx.QueryExecModeSimpleProtocol` (required for pgbouncer/Supavisor
transaction pooling; enabled by `pgbouncer=true` in `DATABASE_URL` — see
`server/internal/store/pool.go`), pgx has no server round-trip to learn a
parameter's target column type, and two distinct failures follow when writing
to a `jsonb` column:

- A raw Go map (e.g. `map[string]string`) has **no default encoder at all**:
  `unable to encode ... into text format for unknown type (OID 0): cannot
  find encode plan`.
- A `[]byte` / `json.RawMessage` **does** have a default encoder — but it is
  `bytea` (hex-escaped), so even an explicit `::jsonb` cast fails with
  `invalid input syntax for type json`: the cast tries to parse the escaped
  hex string itself, not the underlying bytes.

The working pattern: `json.Marshal(...)` explicitly in Go, bind the result as
a Go **string** (not `[]byte`), and put an explicit `::jsonb` cast in the SQL
text. A string is embedded as a plain, unescaped text literal under the
simple protocol, which the cast can parse. For nullable columns, a small
`jsonParam(json.RawMessage) any` helper (nil stays SQL NULL, non-nil becomes
a string) covers the pattern — see `server/internal/store/agent_messages.go`.

## Evidence

[BUG-007, item 5](../issues/2026-07-20-bug-007-rls-against-real-postgres.md), and the
comment on `insertPrd` in
[`server/internal/store/prds.go`](../../server/internal/store/prds.go), which
documents this exact pattern at the code site where it applies.

## Scope

Every write of a Go map/struct/raw-JSON value to a PostgreSQL `jsonb` (or
`json`) column through pgx when the connection uses the simple query protocol
— i.e. any transaction-pooled deployment of this codebase.

## When to apply again

- Adding any new `jsonb` column write in `server/internal/store/`: marshal in
  Go, bind a string, cast with `::jsonb` — copy the `insertPrd` pattern or
  reuse `jsonParam`.
- Debugging `cannot find encode plan` or `invalid input syntax for type json`
  on an INSERT/UPDATE that works in dev but fails behind a pooler: check the
  parameter's Go type before touching the SQL.
- Evaluating any pooling change: remember the protocol mode changes parameter
  encoding semantics, not just connection behavior.
