# 2026-07-23 — Backend pre-commit review: deferred findings

> **Resolved 2026-10-10.** Every confirmed item is fixed. Earlier passes fixed the conversation-lock
> race (`dispatch.go` now holds the registry lock across the check and the
> TryLock), the unbounded request body (`withMaxBody`, 1 MiB), the non-atomic
> GitHub sync (one transaction, 2026-07-27) and the unchecked SSE index
> assertions (comma-ok in `anthropic_parse.go`). This burn-down fixed the rest,
> each test-first on real PostgreSQL: the lost assistant reply (`3832ca4`),
> per-iteration section refresh (`a20df30`), the `CreateRevision` race with a
> per-PRD advisory lock (`071e767`, and `0107b29` for commits) and the
> `ListMembers` check (`3d37822`, `aca242c`). See
> [backend fixes](../changelogs/2026-10-10-close-open-gaps/backend-fixes.md).
>
> The original record follows, unedited.

An adversarial security + correctness review (5 package reviewers + per-finding
adversarial verification, workflow `backend-precommit-review`) swept the Go
backend before its first public commit. **No push-blockers**: no secret leaks,
no trivially-exploitable authorization/injection defects. The confirmed items
below are real but non-blocking hardening/correctness follow-ups, recorded here
rather than fixed inline — they predate the commit and are independent of it,
and the two medium concurrency fixes deserve their own tested change.

## Medium (confirmed real by adversarial verify)

- **`internal/agent/dispatch.go:53` — TOCTOU race in `tryLockConversation`
  cleanup.** The unlock func reads/creates the `*sync.Mutex` under `d.mu` but
  releases it before `l.TryLock()`, so a map entry can be deleted while another
  goroutine is about to lock the same conversation. Impact: two goroutines could
  interleave on one conversation's turn. Fix: hold `d.mu` across the
  check-and-delete, or use a ref-counted / `sync.Map`-based lock registry.
- **`internal/agent/coach.go:224` — assistant reply silently lost on final-write
  failure.** `UpdateAgentMessageContent` error is discarded (`_ =`) and
  unlogged, then `succeeded = true` is set unconditionally, so the defer's
  partial-persistence path is skipped and the turn reports success with the
  assistant message left empty. Fix: log the error and, on failure, persist the
  partial stream via the existing partial-persistence path.

## Low (fix opportunistically)

- **`internal/httpx/json.go:42`** — `DecodeJSON` never bounds the body
  (`http.MaxBytesReader`); unauthenticated endpoints buffer arbitrarily large
  JSON. Add a sane cap (e.g. 1 MiB).
- **`internal/api/github.go:105`** — progress-row inserts and the sync-cursor
  advance are non-atomic; a mid-loop failure re-imports the same commits as
  duplicate timeline entries on the next sync. Wrap in one transaction.
- **`internal/agent/anthropic.go:224,235`** — unchecked
  `event["index"].(float64)` panics on a malformed SSE event; every other
  dynamic access uses the comma-ok form. Make it comma-ok.
- **`internal/agent/coach.go:287`** — `runLoop` rebuilds the system prompt each
  tool iteration but keeps the pre-turn `sections` snapshot, so sections updated
  mid-turn by `update_prd_section` aren't reflected. Refresh per iteration.
- **`internal/store/prd_revisions.go:54`** — `CreateRevision` computes
  `nextRevision` without a row lock; concurrent inserts collide on
  `UNIQUE(prd_id, revision)`. Take a lock or `INSERT ... SELECT max+1`.
- **`internal/store/memberships.go:52`** — `ListMembers` leans on RLS that
  migration `20260720153000_unforce_membership_rls` weakened; the doc comment is
  now stale. Add a Go-level membership check (defense-in-depth).

## Judged a non-issue

- **"juanleme" in source comments** (`store/users.go:88`, `auth/password.go:50`,
  `api/auth.go:111`, `auth/password_test.go:43`) — these name the predecessor
  project's schema while explaining the bcrypt→argon2id migration. "juanleme" is
  a project *name*, already public across `docs/` (HANDOFF, changelogs,
  `docs/history/`), not a provider/project identifier, credential, hostname, or
  PII. Left as-is for consistency with the public record.
