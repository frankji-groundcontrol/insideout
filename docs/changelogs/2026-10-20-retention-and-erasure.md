# Coach conversations expire, and an account can be erased

Date: 2026-10-20. Closes
[rv4-DATA-06](../issues/2026-10-18-rv4-retention-erasure.md). Not pushed.

## What changed

The owner adopted the recommended answers: a coach conversation lives 365 days
after its newest message (or after its own creation when it has none), erasure
is a tombstone, self-service delete comes after that, a whole-account export
is not required, and the notice promises no response time. The contact in
[PRIVACY.md](../../PRIVACY.md) stays the `<privacy-contact>` placeholder. No
address was added.

- Migration `20261020140000` gives `agent_conversations_all` and
  `agent_messages_all` the `current_user_id() IS NULL` branch, the same shape
  as `20260721160000`, so the reaper can delete idle conversations. Messages
  cascade. `ai_runs.conversation_id` is already `ON DELETE SET NULL`. An author
  may also delete their own ideas, which previously had no DELETE policy.
- `purgeIdleConversations` runs beside `purgePastRetention` from
  `ReapStaleAIRuns`. The new logic is in
  `server/internal/store/conversation_retention.go` because
  `agent_messages.go` was already near the line cap.
- `Store.EraseUser` tombstones one account in one transaction. Private rows
  go. Shared PRDs, revisions, commits and projects stay. A workspace the user
  created moves to a remaining member, or is deleted when there is none.
  `creator_id` and the account's e-mail are pinned against the app role, so
  the move and the overwrite go through `SECURITY DEFINER` functions owned by
  `insideout_owner`.
- `DELETE /api/v1/me` requires the current password and is human-only.

## Verification

On a private Postgres 18 at `127.0.0.1`, migrated from this worktree:

- Red, before the code: `TestErase_IdleConversations` left a conversation
  older than 365 days in place, and `TestErase_Account` still found the old
  address in the user row, ideas, messages and runs.
- Green, after: both tests passed. A newer conversation stayed. After erasure
  no scanned text column named the old address, the password hash did not
  verify, and another member's PRD and commit were unchanged.
- `TestRLSWalk`, `TestRLSWriteWalk_NoRowChanges` and the existing reaper tests
  passed. `go test ./internal/api/` passed.
- `TestCanonicalRepoURLInSQLMatchesGo` failed on this cluster. It compares
  `_canonical_repo_url` with Go for two invisible Unicode characters. This
  change does not touch that function.

## Operator note

Apply migrations before serving the build that calls `EraseUser`. Until
`20261020140000` is applied, idle-conversation purge and account erasure fail
closed (the policies and the two functions are missing). Backups still hold
deleted rows until the operator rotates them. No response time is promised
for a request sent to `<privacy-contact>`.
