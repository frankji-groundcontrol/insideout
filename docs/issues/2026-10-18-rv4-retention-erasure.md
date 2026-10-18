# Retention and erasure: the policy only the owner can set

Open, recorded 2026-10-19. Row 3.5 of the
[review-4 remediation plan](../review/2026-10-18-multirole-4/40-remediation-plan.md)
covers two findings. rv4-PUB-02 (no privacy notice) is closed by
[PRIVACY.md](../../PRIVACY.md) and a link on the sign-in and sign-up door.
rv4-DATA-06 (no way to erase a user, conversations kept for ever) is a policy
choice, not a bug fix: the
[owner decisions record](2026-10-18-rv4-owner-privacy-decisions.md) holds no
retention window and no erasure rule, so this wave wrote none and changed no
migration, store routine or route. This record gives the options, what each
costs, a recommended default and the one question to answer. It spells no
address, host or key.

## Decision

| Row | Register IDs | The owner decides | Holds until decided | Closes when |
|---|---|---|---|---|
| 3.5 | rv4-DATA-06 | How long a coach conversation may live after its last message, and what an erasure request does to an account. | Today's behaviour, stated as built in PRIVACY.md: conversations kept until their workspace is deleted, accounts deleted by hand, no response time promised. | The owner fills the answers in "The question" below. |

## What the code does today

| Data | Kept for | Mechanism |
|---|---|---|
| `ai_runs` (raw prompt and reply) and `ai_run_events` (model, tokens, latency) | 30 days from creation | `purgePastRetention`, run by the reaper every 5 minutes and at start (`retentionWindow` in `server/internal/store/agent_messages.go`). |
| `sessions`, `auth_tokens`, `invitations` | 30 days after they expire | The same routine. |
| `agent_conversations`, `agent_messages` (full transcript, tool calls, rolling summary) | For ever | Nothing purges them. The `abandoned` status is allowed by the CHECK and set by no code. They go only through `ON DELETE CASCADE` when their workspace is deleted; no route deletes a PRD or a conversation. |
| `users` | For ever | No route and no store routine. A manual `DELETE FROM insideout.users` fails while the account is still named in any table below. |

Two facts that shape any fix:

- The reaper cannot purge conversations as it stands. It runs as the runtime
  role `insideout_app`, which is subject to RLS, and `agent_conversations_all`
  and `agent_messages_all` are owner-only with FORCE ROW LEVEL SECURITY and no
  system-context branch (unlike `ai_runs_all`, which admits an unset
  `app.user_id`, migration `20260721160000`). See
  [database and RLS](../architecture/database-and-rls.md).
- Dropping an idea only sets its status to `dropped`; the text stays. An
  erasure that deletes "the user's ideas" has to delete those rows, dropped
  ones included.

## The 11 user references that block a delete

All `NO ACTION`: deleting the user fails while any such row exists.

| # | Column | Null | Migration | What it ties to the user |
|---|---|---|---|---|
| 1 | `workspaces.creator_id` | no | `20260720135751` | A workspace others may use; only the creator can delete it. |
| 2 | `projects.owner_id` | yes | `20260720135752` | Project ownership. |
| 3 | `projects.created_by` | no | `20260720135752` | Project authorship. |
| 4 | `project_updates.author_id` | no | `20260720135752` | Update text. |
| 5 | `ideas.author_id` | no | `20260720135753` | Private idea text, dropped ones included. |
| 6 | `prds.author_id` | no | `20260720135753` | PRD text other members rely on. |
| 7 | `prd_revisions.created_by` | no | `20260720135753` | Revision history. |
| 8 | `prd_commits.committed_by` | no | `20260821150000` | The commit signature, the product's "a person put their name on it". |
| 9 | `proposal_decisions.decided_by` | no | `20260822100000` | Accept and reject decisions on agent proposals; the log is insert-only by design. |
| 10 | `agent_conversations.user_id` | no | `20260720135754` | The coach transcript. |
| 11 | `ai_runs.user_id` | no | `20260720135754` | Raw prompts and replies. |

Already safe: `sessions`, `auth_tokens` and `workspace_memberships` cascade;
`invitations.created_by` and `used_by` and the roadmap attribution columns
(`created_by`, `updated_by`, `owner_user_id`, `validated_by`) set null.

## What an erasure must cover

| Class | Rows | Treatment |
|---|---|---|
| Private to the user | Conversations with their messages, run records with their events, ideas of every status, sessions, tokens, memberships | Delete. |
| Shared work other members rely on | PRDs, revisions, commits, projects, project updates, proposal decisions, workspaces the user created | Needs a rule: keep attributed to a placeholder, reassign, or delete. |
| The account row | E-mail address, username, password hash, avatar address, bio, keywords | Overwrite or delete. |

## Options

| | Erasure rule | Cost | Risk |
|---|---|---|---|
| A | Tombstone: keep the `users` row, overwrite the personal fields with unique placeholders and an unusable password hash, delete the private data, leave shared work attributed to a deleted user. | No foreign-key change; no migration for the 11 references. | The user id stays and links to the kept work; a strict reading of "erased" may not accept that. |
| B | Hard delete: change the 11 references to cascade (or set null where nullable) by migration. | One migration, one routine. | Erasing a member deletes PRDs and commits others rely on, and a creator's workspace with it, and it breaks the insert-only decision log (`proposal_decisions`). Destructive and irreversible. |
| C | Reassign then delete: move shared work to a placeholder user, then delete the row. | A placeholder user and a routine. | The same end state as A with more moving parts. |
| D | By hand on request: an operator runbook, no route. | None to build. | Slow, error-prone, and the response time is a promise someone must keep. |

Recommended: **A, first as D under a runbook, then as a self-service
`DELETE /me`** that asks for the password again and is human-only in
`server/internal/api/agent_scope.go`. A keeps other members' work intact,
needs no foreign-key change and deletes everything private.

## Retention proposals

These are proposals for the owner, not policy. PRIVACY.md invents no window.

| Data | Today | Options | Recommended |
|---|---|---|---|
| `agent_conversations`, `agent_messages` | None | Keep; or delete the whole conversation 90, 180 or 365 days after its last message | 365 days after the last message. Each new message resets the clock, so an active conversation is never touched; the PRD is separate and stays. |
| `ai_runs` | 30 days | Keep 30; shorten; lengthen | Keep 30. The rate limiter reads one hour of history, and the text duplicates the transcript. |
| `ai_run_events` | With `ai_runs` | Same | Keep. They hold no message text today and follow `ai_runs` through the cascade. |
| `sessions`, `auth_tokens`, `invitations` | 30 days after expiry | Same | Keep. |

## The question

| # | Question | Options | Recommended | Owner's answer |
|---|---|---|---|---|
| 1 | How long is a coach conversation kept after its last message? | Keep / 365 / 180 / 90 days | 365 days | |
| 2 | What does an erasure request do to an account? | Tombstone / hard delete / by hand only | Tombstone | |
| 3 | Build a self-service delete? | After 1 and 2 / by hand only | After 1 and 2 | |
| 4 | Export an account before erasure? | Required / not required | Not required; per-PRD export exists | |
| 5 | Who is `<privacy-contact>`, and may PRIVACY.md promise a response time? | A monitored address or a form; a number of days or none | A form; no promise | |

## Bounded fix prompt

To run only after the owner fills rows 1 and 2, as a cluster with a database:

> Retention: add a migration giving `agent_conversations_all` and
> `agent_messages_all` the `current_user_id() IS NULL` branch, mirroring
> `20260721160000_ai_runs_reaper_system_context.sql`, or run the purge from an
> operator command on `DATABASE_OWNER_URL` (the `BYPASSRLS` role) outside the
> server. Add `conversationRetention` and `purgeIdleConversations` beside
> `purgePastRetention`: delete conversations whose newest message (its
> creation time when it has none) is older than the answered window; messages
> cascade and `ai_runs.conversation_id` already sets null.
> Erasure: add `Store.EraseUser(ctx, userID)`, one transaction, following the
> answered rule; for option A delete the private data in the table above,
> overwrite the account fields, and transfer or delete workspaces the user
> created. Then add `DELETE /api/v1/me` behind the password and mark it
> `humanOnly`.
> Tests on real PostgreSQL, via `./scripts/dev.sh -C server go test
> ./internal/store/... -run TestErase -v`: no row names the user afterwards
> except the placeholder, other members' PRDs and commits are unchanged, and
> a conversation older than the window is gone while a newer one stays.
> Update the Retention and Deleting sections of PRIVACY.md, `docs/usage/`
> and a changelog; then close this record.
