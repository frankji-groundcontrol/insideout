# Store-error ladders outside the converted handlers

**Date:** 2026-11-21
**Area:** `server/internal/api`
**Status:** resolved 2026-11-21 (gaps round after the review 5
remediation) for the ladders; QUAL-09 and QUAL-10 below stay open. Records
review 4's QUAL-08 to QUAL-11 and review 5's STRS-11 and PARC-08.

> **Resolved 2026-11-21.** Every ladder in the table below now answers
> through `writeStoreError` (115 calls in the package), and so does every
> hand-written 500 that reported a bare store call. `ErrRepoTaken` joined
> the mapping with its `github_repo_already_linked` code, and `isStoreError`
> lets the GitHub sync tell a store failure from a GitHub one. One answer
> changed: the last admin's refusal (`PATCH`/`DELETE
> /workspaces/{id}/members/{userId}`) now carries code `last_admin` instead
> of `conflict`, as every other route meeting `ErrLastAdmin` already did.
> `rv5_store_error_routes_test.go` pins the status, code and message of every
> converted path.
>
> **Residue: 35 `errors.Is(err, store.Err…)` checks in 29 places, all
> deliberate**, each with a one-line comment: the refusal-as-404 checks
> (`roadmap_read.go` 2, `roadmap_delegation.go`, `agent_root.go`,
> `members.go`, `roadmap_progress.go`), the membership and PRD pre-checks
> (`members.go`, `projects.go`, `ideas.go`, `prds.go` 2), the auth handlers'
> uniform and dead-token answers (`auth_email_flows.go` 7, `auth.go` 4,
> `auth_reset_watch.go`), the 409 for a stage not waiting
> (`conversations.go`), an absent roadmap read as empty (`guide.go`,
> `agent_routes.go`), `ErrRepoHeldOutOfReach` answered as a repo GitHub does
> not show (`github.go`), `ErrRoadmapNotEmpty` (`roadmap_ai.go`) and the
> stream heartbeat's yes/no (`stream_recheck.go`). 18 hand-written 500s
> remain, none for a bare store call: token, hash and planner failures,
> `issueSession` (a token and a store write together), the two "streaming
> unsupported" answers, the panic recoverer, and the GitHub webhook's
> repo lookup, which keeps its own message and logs the repo name.

## Problem

Each handler used to map store errors to a status by hand: a ladder of
`if errors.Is(err, store.ErrNotFound) {...}` branches ending in a logged 500.
Review 5 counted 69 such ladders and 119 hand-written 500s in the package.
Wave 3 of the review 5 remediation added one helper,
`(*Server).writeStoreError` in
[store_error.go](../../server/internal/api/store_error.go). It maps every
store error to its status, its code and a default message once, and the
handler passes only its own wording (`errMsgs`). The roadmap, project,
project-update, PRD, idea and agent handlers now use it (43 calls).

These still hold their own ladders (count of
`errors.Is(err, store.Err…)` checks per file, non-test, at the time of
writing):

| File | Checks |
| --- | --- |
| `members.go` | 8 |
| `auth_email_flows.go` | 7 |
| `workspaces.go` | 6 |
| `github.go` | 6 |
| `conversations.go` | 6 |
| `auth.go` | 4 |
| `guide.go` | 3 |
| `presence_routes.go` | 2 |
| `stream_recheck.go`, `auth_reset_watch.go` | 1 each |

The converted files keep a few checks on purpose: those that answer 404 for
a refusal so a non-member cannot learn that a row exists (`roadmap_read.go`,
`agent_root.go`, `roadmap_progress.go`), the membership pre-checks
(`projects.go`, `ideas.go`, `prds.go`), and `ErrRoadmapNotEmpty`
(`roadmap_ai.go`). 81 hand-written 500s remain in the package.

The ok flag of `UserID` is still discarded in most handlers, and
`internal/store` still returns bare errors far more often than it wraps them
(QUAL-09 and QUAL-10). Neither is changed here.

## Target

Every handler that reports a store error calls `writeStoreError`, unless
the handler answers a different status on purpose. A handler that does
says why in a one-line comment.

## Bounded fix prompt

> In `server/internal/api`, convert the store-error ladders in `members.go`,
> `workspaces.go`, `conversations.go`, `guide.go` and `presence_routes.go`
> to `s.writeStoreError(w, op, err, errMsgs{...})`, keeping each handler's
> messages. Convert only ladders whose statuses match `storeErrors` in
> `store_error.go`. Leave a ladder alone when it answers 404 for a refusal on
> purpose, and add a one-line comment saying so. Leave the auth handlers'
> deliberate uniform answers alone (login, reset and verify must not reveal
> whether an account exists). Run `go test ./internal/api` against a real
> database before and after; no status or message may change. Then update the
> counts in this record, or close it.
