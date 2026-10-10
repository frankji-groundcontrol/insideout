# Backend fixes from the deferred-findings records

Every item below came from a deferred-findings record
([pre-commit review](../../issues/2026-07-23-backend-precommit-review-findings.md),
[optimization review](../../issues/2026-07-26-backend-optimization-deferred.md),
[coach line budget](../../issues/2026-10-04-coach-go-line-budget.md)). Each fix
was written test-first on real PostgreSQL 18: the new test failed on the old
code for the stated reason, then passed. The file splits that made room for
them are in [backend-splits.md](backend-splits.md).

## Coach

- **Sections written earlier in a turn were invisible to later tool
  iterations.** `runLoop` rebuilt the system prompt each iteration but kept the
  pre-turn PRD snapshot, so a section written by `update_prd_section` did not
  show up until the next turn. It now re-reads the PRD before every provider
  call (and keeps the old snapshot if the read fails).
- **A failed final write still reported success.** The error from persisting
  the assistant message was discarded and the turn was marked succeeded with an
  empty reply. It is now logged and the turn is not marked succeeded, so the
  existing partial-persistence path runs.
- **`loadHistory` fetched the whole conversation to replay a window.**
  `ListRecentChatMessages` returns only the replayed tail (user rows and
  non-empty assistant rows, newest N, in order), backed by the existing
  `(conversation_id, created_at)` index.

## PRD revisions and commits

- **Concurrent snapshots collided on `UNIQUE(prd_id, revision)`.** Every writer
  read the same `current_revision`, so the losers got a spurious 409.
  `CreateRevision` and `CreatePrdCommit` now queue behind one per-PRD advisory
  lock (`pg_advisory_xact_lock(hashtext('prd_revision:' || prd_id))`), taken
  first. The second change matters: with the lock only on snapshots, a commit
  racing a snapshot took the PRD row lock and the advisory lock in opposite
  orders and could deadlock. Both orders are now one order. The unique
  violation still maps to a typed conflict as a backstop.

## Membership and login

- `ListMembers` checked membership only through row-level security; it now
  checks in the same transaction and the API answers 404 when the caller left.
- Logging in with an unknown email returned faster than a wrong password, which
  tells an attacker which emails exist. Unknown emails now spend the same
  argon2 cost.

## Database

- **`ai_runs_reaper_idx`** (`20261010100000_ai_runs_reaper_index.sql`): a
  partial index on `ai_runs (updated_at) WHERE status IN ('pending','running')`.
  Measured on 200k finished runs: the reaper's UPDATE went from a Seq Scan at
  16–20 ms to an Index Scan at 0.06–0.08 ms. The index makes `status` and
  `updated_at` HOT-blocking for `ai_runs`; at a few updates per agent turn that
  is negligible.
- **Legacy invite codes** (`20261010110000_rotate_short_invite_codes.sql`):
  codes were 6-digit numbers (10^6 keyspace, the only credential on the join
  endpoint) until 2026-07-27, when new ones became 128-bit hex; older rows kept
  the short code. One idempotent UPDATE replaces every code shorter than 32
  characters. This closes the join-endpoint rate-limit item by entropy rather
  than by throttling.

## Measured and not shipped

The `ListWorkspacesForUser` correlated-count rewrite was 6.3% faster in the
measured scenario (100 workspaces of 50 members): inside noise for a rewrite of
an authorization-shaped query, so it stays as it is
([learning](../../learning/2026-10-10-measure-before-rewriting-a-query.md)).

## How it was verified

`go test ./... -count=1 -v` against the scratch PostgreSQL 18 database:
566 PASS, 0 FAIL, 1 SKIP (`TestReadSecretTerminal`), with `gofmt -l`,
`go build` and `go vet` clean. The racing-writer tests (eight snapshots at
once; six snapshots against six commits) fail on the old code and pass on the
new.

## Operator notes

- **Apply two migrations** on deploy: `ai_runs_reaper_idx` and the invite-code
  rotation. Nothing is deployed from this change set.
- **Outstanding 6-digit invite codes stop working** after the rotation. Admins
  read the new code in the workspace and re-share it. The `updated_at` of the
  rotated workspaces moves (the code did change).
