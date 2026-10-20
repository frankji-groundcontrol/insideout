# Pinned columns and a creator pin block the app role from erasing an account

## Lesson

`insideout_app` cannot change `users.email` or `workspaces.creator_id`. The
users pin lists the columns a profile edit may change, and the workspace
trigger rejects a `creator_id` change unless the role has `BYPASSRLS`. A
tombstone that overwrites the address, and a transfer of a workspace that
still has another member, have to run as `insideout_owner` inside a
`SECURITY DEFINER` function. The function still checks `app.user_id`, so the
app role cannot aim it at someone else.

The same reaper that purges `ai_runs` with no actor needs that unset-actor
branch on `agent_conversations` and `agent_messages` before it can delete idle
conversations. `TestRLSWalk` and `TestRLSWriteWalk_NoRowChanges` treat a new
`current_user_id() IS NULL` clause as a decision: list the tables as
system-readable, and skip the actor-less write that would otherwise match
every row.

## Evidence

`Store.EraseUser` and `20261020140000_conversation_retention_and_erasure.sql`,
checked by `TestErase_Account` and `TestErase_IdleConversations` on a private
Postgres 18.

## When to apply

Any new flow that must change a pinned column, or any purge that runs with no
signed-in user against a `FORCE`d table.
