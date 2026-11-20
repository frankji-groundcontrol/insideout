# Retention and erasure

Closed 2026-10-20. The owner answers in
[the retention record](../../issues/2026-10-18-rv4-retention-erasure.md) are
the ones this plan built. Pushed to `origin/main` 2026-11-21.

## Done

- [x] Migration: system-context branch on coach conversations and messages,
  author DELETE on ideas, definers for the creator move and the tombstone.
- [x] Purge idle conversations from the reaper, in a new store file.
- [x] `Store.EraseUser` in one transaction.
- [x] `DELETE /api/v1/me`, current password, human-only.
- [x] Red then green on real PostgreSQL.
- [x] PRIVACY.md Retention and Deleting, and the owner-answer column.

## Not in this plan

Projects, memberships, roadmap and the MCP server were left to their owners.
No address was written into the notice.
