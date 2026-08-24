# Gated integration tests must derive unique keys per run — rows persist

**Learned** 2026-08-24 from the email-auth slice
([changelog](../changelogs/2026-08-24-email-auth.md)) while running the
full `DATABASE_URL`-gated store suite a second time.

Related same-day operational note: when `DATABASE_URL` points at the
hosted Supabase pooler (not the local bundled Postgres), the full store
package's per-test pools push the run past Go's default 10-minute test
timeout — it panics with a goroutine dump that looks like a connection
hang. That is latency, not deadlock: rerun with
`go test ./internal/store/ -timeout 25m` (observed ~12 min). Also beware
piping test output through `tail` — the pipe's exit code masks the
failure; check `${pipestatus[1]}`.

## The lesson

The gated store tests follow the mkUser convention: throwaway rows are
created with random identifiers and **never cleaned up**, so every run
shares the table with every earlier run. A test that inserts a row with a
hard-coded natural key (`'hash-roundtrip'`, `'sess-mine'`) into a
globally-unique column passes on a fresh table and fails on the second run
with `23505 duplicate key value violates unique constraint` — while still
passing under `-run <Test>`, because the collision is with *history*, not
with other tests in the same run.

Natural keys in gated tests must be derived per run (`uuid.NewString()`),
or the test must clean up after itself.

## When to apply

Any `DATABASE_URL`-gated test (or any test against a persistent shared
database) that inserts rows keyed by anything other than a fresh uuid.
Trust signal: a suite that passed once and fails on rerun with duplicate
keys while the focused `-run` still passes.

## Evidence

`auth_tokens_test.go` passed alone (fresh table), then six of seven tests
failed in the full-suite rerun with `auth_tokens_token_hash_key`
violations; `-count=2` after switching to `freshHash(t)` (`hash-` +
`uuid.NewString()`, same treatment for session hashes) is green, and the
stale fixed-hash rows were deleted via the owner DSN.
