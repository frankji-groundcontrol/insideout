# Hardening pass — PRD coach, roadmap, GitHub sync, experience-marking, authz

**Date:** 2026-07-27
**Status:** complete (all findings resolved and verified)

## Context

Five product surfaces were audited adversarially: (1) PRD coach, (2) AI-created
roadmap, (3) GitHub sync + commit review, (4) the comments/notes
experience-marking system (project updates: `progress` / `blocker` / `note`),
and (5) cross-cutting authz/robustness. A fan-out → per-finding adversarial
refuter → synthesis workflow raised 27 candidate defects, refuted 2, and
consolidated the survivors into the 22 fixes below. Four further items from
independent code-tracing are reconciled at the end.

**Method rules (from CLAUDE.md):** real verification, no mocks — every
`needs_test` fix ships with a `DATABASE_URL`-gated integration test against
real PostgreSQL; SQL writes only the `insideout` schema; files under ~350
lines; English-only docs prose; record a changelog + update HANDOFF on landing.

**Fix discipline (ponytail):** surgical root-cause fixes, reuse the codebase's
existing patterns (notably `pg_advisory_xact_lock(hashtext(...))` for
serialization — see `store.ReplaceRoadmapTree`), mark deliberate
simplifications with `// ponytail:` comments naming their ceiling + upgrade
path.

## Wave 1 — HIGH + load-bearing MEDIUM (data integrity)

- [x] **F1 refresh-token-rotation-double-mint** (HIGH) —
  `server/internal/store/sessions.go:70`. `RotateSession`'s revoking `UPDATE`
  lacks `AND revoked_at IS NULL` and discards rows-affected, so two concurrent
  refreshes each mint a valid session from one token. Fix: add the guard,
  check `CommandTag.RowsAffected()==0` → abort tx, return `ErrConflict`
  (token reuse). `RevokeSession` already has the guard — this restores parity.
  Test: `TestSessions_RotateIsReuseSafe`.
- [x] **F2 roadmap-assemble-tree-subtree-loss** (MEDIUM) —
  `server/internal/agent/roadmap_planner.go:111`. Value-copies a child during
  random-order map iteration, freezing a partial `Children` slice and dropping
  whole subtrees (nearly every POST `/prds/{id}/build` persists an incomplete
  roadmap, since `store.RoadmapPlanNode.Children` is a slice of values). Fix:
  build a `parent → []childID` index and construct the tree top-down by id
  (deterministic, preserves all subtrees + model sibling order, cycle-safe).
  Test: `TestAssembleTree_PreservesSubtrees` (pure, no DB).
- [x] **F3 ensure-project-race-orphan-duplicates** (MEDIUM) —
  `server/internal/store/roadmap_plan.go:190`. `EnsureProjectForPrd` is a
  read-modify-write with no lock; two concurrent first-builds both read
  `project_id` NULL and both insert, orphaning one project. Fix: acquire
  `pg_advisory_xact_lock(hashtext(prdID))` at the top of the tx, mirroring
  `ReplaceRoadmapTree:48`. Test: `TestEnsureProjectForPrd_ConcurrentFirstBuild`.

## Wave 2 — MEDIUM (correctness + abuse surface)

- [x] **F4 prd-title-lost-update** — `store/prds.go:142`. Make title optional +
  `updated_at` CAS → `ErrConflict` (409). needs_test.
- [x] **F5 github-sync-cursor-window-loss** — `api/github.go:89`. Paginate
  (Link `rel=next`, `per_page≤100`, capped pages) until the cursor SHA is
  found; otherwise surface a "history truncated" signal instead of silently
  advancing the cursor. needs_test.
- [x] **F6 github-sync-non-atomic-duplicates** — `api/github.go:123`. Insert
  all synced commits in one store tx (or `ON CONFLICT (project_id, sha) DO
  NOTHING`) so a partial failure can't leave duplicates + a divergent cursor.
  needs_test.
- [x] **F7 coach-watchdog-masked-as-disconnect** — `agent/coach.go:269`.
  Distinguish an idle-watchdog cancel from a client disconnect
  (`ErrUpstreamStall` when the derived ctx is canceled but
  `r.Context().Err()==nil`); emit SSE error + record a circuit failure in
  `failTurn`. needs_test. Classified at the single decision point
  (`anthropic.go doStreamChat`, which holds both parent + derived ctx);
  idle timeout made an injectable calibration field so the watchdog is
  testable in ms. Tests: `TestStreamChat_UpstreamStall`,
  `TestStreamChat_ClientDisconnect` (real httptest, no mocks).
- [x] **F8 prd-non-author-silent-403-discard** —
  `app/src/pages/prd/[id]/index.vue:76`. `canEdit = isAuthor || isAdmin`
  computed mirrors `handleUpdatePrd`'s 403 rule; the title `<input>` and every
  section `<textarea>` are `:readonly="!canEdit"`, and both `saveSection` /
  `saveTitle` guard on `canEdit` + revert the DOM/ref to the last server value
  in a `catch` (a one-way `:value` binding never resets on a rejected save, so
  without the revert a non-editor is left with phantom text in the box). Backend
  authz already correct. Test: `app/src/pages/__tests__/PrdEditor.test.ts`
  (non-author/member readonly; author + admin editable; revert-on-rejected-save).
- [x] **F9 project-updates-unpaginated-embed** — `store/project_updates.go:56`.
  `ListProjectUpdates` now takes `limit` + an optional `before *uuid.UUID`
  keyset cursor and emits `ORDER BY created_at DESC, id DESC LIMIT $n`. The
  cursor is composite `(created_at, id)` — `created_at` alone is ambiguous
  because rows written in one tx share a statement-time timestamp (a GitHub
  sync batch), so `id` breaks ties. `handleGetProject` embeds only the newest
  page (`store.ProjectUpdatesPageSize = 50`) plus a forward-compatible
  `nextCursor`; the frontend still gets `updates`, just bounded (its load-more
  is the deferred follow-up). Test: `TestProjectUpdates_Pagination` — walks the
  cursor 2-at-a-time and asserts every row exactly once, DESC across page
  boundaries, terminates.
- [x] **F10 project-update-content-unbounded** — `api/project_updates.go:50`.
  Shared `validateUpdateContent` (trim + empty + `maxUpdateContentRunes = 5000`
  rune cap) now used by both add and edit, so the bound is enforced in one
  place; overlong → clean 400 "content too long". The 1 MiB body cap stays the
  transport ceiling. Test: `TestValidateUpdateContent` (pure; incl. multibyte
  boundary proving rune count, not byte count).
- [x] **F11 unbounded-persistence-contexts** — `agent/coach.go:163` (+ :225,
  :251, `telemetry.go:20`). All four genuinely-detached persistence ctxs now go
  through one shared `detachedContext(ctx)` helper =
  `context.WithTimeout(context.WithoutCancel(ctx), persistTimeout)` with a
  `defer cancel()` at each site, so a wedged terminal write can't hang its
  goroutine (and its held dispatch permit) forever. `persistTimeout = 10s` is
  the single calibration knob. Deviation: `critic.go:45/87` were **not**
  changed — they persist on the raw request ctx on purpose (`coach.go:233`
  "drop it if they're gone"), so they're already bounded by the request
  lifecycle, not an unbounded detach. (no test — infra; verified by
  build/vet + `go test ./internal/agent/...`.)

## Wave 3 — LOW (hygiene, info-leak, opaque errors)

- [x] **F12 github-upstream-error-info-leak** — `api/github.go:92`. The 502
  branch now logs the transport error server-side and returns a generic
  "GitHub sync failed" (folded into the F20 status switch) — upstream internals
  no longer reach the client.
- [x] **F13 truncated-answer-marked-complete** — `agent/anthropic.go:295`.
  `Turn` now carries `Truncated bool`; `parseAnthropicStream` sets it when
  `stop_reason == "max_tokens"`, and `coach.runLoop` appends `truncatedMarker`
  to the final text on that path so the partial answer is persisted + replayed
  with a visible "[response truncated…]" flag instead of being presented as a
  finished response. Tests: `TestParseAnthropicStream_MaxTokensIsTruncated`,
  `TestParseAnthropicStream_EndTurnNotTruncated`.
- [x] **F14 revision-snapshot-race-opaque-500** — `store/prd_revisions.go:55`.
  `CreateRevision` now maps the `23505` unique_violation (two concurrent
  snapshots both compute MAX+1) to `ErrConflict` via the shared
  `pgconn.PgError` pattern (`workspaces.go:77`), and `handleCreateRevision`
  maps `ErrConflict` → 409 "refresh and try again" (winner's row stands) — so
  the race loser gets a clean retryable 409 instead of an opaque 500. Test:
  `TestPrdRevisions_ConcurrentSnapshotConflict` (8-goroutine burst; asserts no
  opaque error ever leaks — every call is nil or `ErrConflict` — and revision
  rows == winners with no dups/gaps; reliably produces conflicts, e.g. 2
  winners / 6 conflicts across repeated runs).
- [x] **F15 create-revision-swallows-bad-body** — `api/prds.go:180`.
  `handleCreateRevision` decodes into `err` and 400s unless the error is
  `io.EOF` (a body-less POST, since `note` is optional) — mirroring the
  `roadmap_ai.go:46` optional-body pattern — so a genuinely malformed body is a
  client error, not a silent no-op snapshot.
- [x] **F16 context-length-retry-client-divergence** — `agent/coach.go:225`.
  Took the plan's "abort once deltas have flushed" option (no new SSE event —
  a `stream_reset` with no frontend listener would be dead code). `onDelta`
  writes to `streamed` and flushes the SSE delta in lockstep, so
  `streamed.Len()==0` is exactly "the client bubble is still empty". The
  tighten retry is now guarded on that, and the now-redundant `streamed.Reset()`
  is deleted. If deltas already flushed, we fall through to `failTurn` instead,
  so what the user saw is exactly what gets persisted. Context-length errors
  almost always arrive as a pre-stream 400 (empty bubble), so the retry still
  fires for the common case. (no new test — a retry-guard on in-flight SSE
  state; verified by build/vet + `go test ./internal/agent/...`.)
- [x] **F17 in-stream-error-loses-taxonomy** — `agent/anthropic.go:268`.
  The SSE `error` event now flows through `classifyInStreamError(errType, msg)`,
  which mirrors `anthropicHTTPError`'s switch by `error.type` onto the existing
  sentinels (`rate_limit_error`→`ErrProviderRateLimited`, `overloaded`/`api_error`
  →`ErrProviderTransient`, context-bearing `invalid_request`→`ErrContextLength`,
  auth/permission/not-found/invalid→`ErrProviderConfig`, else a plain error) — so
  coach.go's taxonomy treats a mid-stream failure identically to a non-200, with
  no new retry logic (the sentinel rides streamChat's existing single retry).
  Tests: `TestParseAnthropicStream_ErrorEventTaxonomy`,
  `TestParseAnthropicStream_ErrorEventUnknownType`.
- [x] **F18 roadmap-ai-output-unbounded** — `agent/roadmap_planner.go:205`.
  `ExpandNode` now caps `payload.Subtasks` at `maxSubtasks=40` (mirrors
  `assembleTree`'s `maxNodes`) and trims each title over `maxTitleRunes=200`
  by **rune** (not byte), using the same inline `[]rune` slice pattern
  `github.go` uses for commit subjects — so a runaway expansion can't flood a
  node and a multibyte CJK title never splits a character. Both bounds are
  package consts next to `assembleTree`. (no new test — a bound on already
  schema-validated model output; verified by build/vet.)
- [x] **F19 github-commit-subject-unbounded** — `github/github.go:93`. Commit
  subjects now capped at `maxSubjectRunes = 280` (rune count, not bytes) and the
  page body is read through `io.LimitReader(resp.Body, maxCommitPageBytes)`
  (5 MiB) so a runaway upstream can't make us buffer unbounded data. Test:
  `TestFetchCommitsSince_SubjectTruncatedByRune` (300-rune multibyte subject →
  280 back, proving rune not byte counting).
- [x] **F20 github-rate-limit-opaque-502** — `github/github.go:72`.
  `fetchCommitPage` now branches on status into typed errors — 404 →
  `ErrRepoNotFound`, 403/429 → `*RateLimitError{RetryAfterSeconds}` (parsed from
  `Retry-After`, default 60), other non-200 → plain error — and
  `handleSyncGithub` maps them to 404 / 429 `GITHUB_RATE_LIMITED`
  (+`retry_after_seconds`) / 502 respectively, instead of one opaque 502. Tests:
  `TestFetchCommitsSince_RepoNotFound`, `TestFetchCommitsSince_RateLimited`.
- [x] **F21 recover-writes-json-onto-sse** — `api/middleware.go:105`.
  `statusRecorder` now tracks `wroteHeader`, set in BOTH `WriteHeader` and a
  new `Write` override (a bare `Write` commits an implicit 200, so it must set
  the flag too). `withRecover` only calls `httpx.WriteError` when
  `!sr.wroteHeader` — once a response is committed (an SSE stream mid-flight)
  a JSON error body would be appended onto the event stream and corrupt it, so
  it logs and lets the stream's own error handling / client timeout take over.
  The existing `Flush()` forwarder is unchanged. (no test — panic-after-commit
  on a live SSE writer; verified by build/vet.)
- [x] **F22 empty-tool-call-bubbles** — `api/conversations.go:81` (NOT the
  store — the plan's `store/conversations.go:83` does not exist). The display
  loop in `handleListConversationMessages` skips `role=="tool"` rows and
  tool-call-only assistant rows (`role=="assistant" && content==""`), mirroring
  the `loadHistory` filter (coach.go). The store's `ListAgentMessages` stays
  raw on purpose — `coach.loadHistory` also consumes it and does its own
  filtering. (no test — a display filter over an already-tested query; verified
  by build/vet.)

## Reconciled independent items (from code-tracing, outside the workflow 22)

- [x] **R1 ConvertIdea race** — `store/ideas.go`. Real: two converters both
  read `status='pending'`, both passed the check, both committed a PRD —
  orphaning one (the idea's `prd_id` can only point at one). Fixed with an
  atomic conditional `UPDATE ... SET status='converted' WHERE status<>'converted'`
  claim BEFORE inserting (RLS-robust — NOT `FOR UPDATE`, whose policy joins
  `workspace_memberships` and returns zero rows under `EvalPlanQual`). On
  `RowsAffected()==0` → `ErrConflict`, whole tx rolls back, nothing orphaned.
  Verified: `TestConvertIdea_ConcurrentConvert` (real DB, 8 goroutines, ×3)
  asserts exactly 1 winner, exactly 1 committed PRD (counted through the
  actor-context path — a raw pool count is RLS-blind and returns 0), and the
  idea pointing at the winner's PRD.
- [x] **R2 invite-code keyspace** — `store/workspaces.go`. `generateInviteCode`
  now draws 128 bits from `crypto/rand`, hex-encoded (32 chars) — the sole join
  credential on an authenticated-but-unratelimited endpoint is no longer
  brute-forceable (old `%06d` = 10^6). `code` column is `text`, fits unchanged;
  the collision-retry loop still guards uniqueness. Verified: replaced the old
  `TestGenerateInviteCode_IsSixDigits` (asserted the vulnerable format) with
  `TestGenerateInviteCode_KeySpace` — pins `^[0-9a-f]{32}$` and proves 2000 draws
  never collide (a 10^6 space collides near the ~1.2k birthday bound).
- [x] **R3 DecodeJSON trailing-bytes** — `httpx/json.go`. `Decoder.Decode` stops
  at the first value, so `{}{}` / `{} garbage` were silently accepted, defeating
  `DisallowUnknownFields`. Now a second `Decode` must hit `io.EOF` or the body is
  rejected; trailing whitespace still decodes clean. An empty body still returns
  `io.EOF` from the FIRST decode (body-less POST callers check for it). Verified:
  new `TestDecodeJSON` covers clean / trailing-whitespace / empty / unknown-field
  / second-value / trailing-garbage.
- [x] **R4 UpdateProjectUpdate ErrNoRows→500** — `store/project_updates.go`. The
  final `UPDATE ... RETURNING` Scan had no `ErrNoRows` mapping, so a concurrent
  delete between the authorizing SELECT and the UPDATE leaked an opaque error as
  500. Now mapped to `ErrNotFound` (404), mirroring the SELECT's own mapping just
  above. Not independently race-tested: the window requires a delete at exactly
  that instant and can't be triggered deterministically without hooks; the
  mapping is a one-line mirror of the proven SELECT path and the existing
  `TestProjectUpdates` suite covers the happy path.

## Verification bar

`cd server && go build ./... && go vet ./... && go test ./...` all green; the
`DATABASE_URL`-gated integration tests (`-run 'TestSessions|TestEnsureProject|
TestProjectUpdates|TestPrdRevisions|TestRoadmap'`) pass against real
PostgreSQL; `cd app && pnpm test && npx nuxi typecheck` green. Then: dated
changelog entry, update `docs/HANDOFF.md`, record any modularity issues in
`docs/issues/`, update the nearest README indexes, verify links.
