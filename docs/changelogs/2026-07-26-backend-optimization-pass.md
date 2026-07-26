# 2026-07-26 — Backend optimization pass

Five confirmed findings from the adversarial backend audit, fixed at root
cause. Deferred findings (incl. a HIGH invite-code brute-force) are tracked
in `docs/issues/2026-07-26-backend-optimization-deferred.md`.

## Security

- **Request-body cap + connection timeouts.** New `withMaxBody` middleware
  (`internal/api/middleware.go`) wraps every body in a 1 MiB
  `http.MaxBytesReader`; `http.Server` (`cmd/insideout/main.go`) gains
  `ReadTimeout: 15s` and `IdleTimeout: 60s`. No `WriteTimeout` — the coach SSE
  stream must stay open. Closes an unauthenticated slow-drip / unbounded-body
  DoS.

## Concurrency / data integrity

- **Dispatcher conversation-lock TOCTOU** (`internal/agent/dispatch.go`).
  `tryLockConversation` now takes the per-conversation lock while still holding
  the map mutex and only reclaims an entry that still points at its own
  instance. Removes the window where a releaser could delete the map entry
  between a new acquirer reading it and locking it — which let two turns run
  concurrently on one conversation.
- **Roadmap structural-write race** (`internal/store/roadmap.go`).
  `MoveRoadmapNode` and `CreateRoadmapNode` now take the per-project
  `pg_advisory_xact_lock(hashtext(project_id))` — the same idiom
  `ReplaceRoadmapTree` already used. Two opposing concurrent moves can no
  longer both pass the cycle guard and forge an `A↔B` cycle (which orphaned a
  subtree), and concurrent sibling creates can no longer collide on
  `MAX(position)+1`.

## Robustness

- **Coach success-path persist on a detached context** (`internal/agent/coach.go`).
  The success writes (`MarkAIRunSucceeded`, `RecordCircuitResult`,
  `UpdateAgentMessageContent`, `CompleteConversation`) now run on
  `context.WithoutCancel(ctx)`. A client disconnect right after the last token
  no longer leaves the run stuck `running` with an empty message while
  `succeeded` was already `true` (which skipped the failure defer). The critic
  pass stays on `ctx` — its SSE to a gone client is harmless to drop.
- **Checked SSE frame-index assertions** (`internal/agent/anthropic.go`).
  `content_block_start`/`content_block_delta` parse `index` with a comma-ok
  type assertion and skip the frame on mismatch, instead of panicking the whole
  turn (which discarded already-streamed text via the deferred recover).

## Verification

`go build ./…`, `go vet ./…`, `go test ./…` all green; `internal/store`
integration tests ran live against real PostgreSQL (not cached).
