# 2026-07-27 — Cross-surface security hardening pass

Twenty-two workflow-surfaced findings (F1–F22) plus four independently
code-traced items (R1–R4) across all five product surfaces — PRD coach,
AI roadmap, GitHub sync, the project-updates timeline, and cross-cutting
authz — fixed at root cause and each backed by a real (no-mock) test where
the logic is deterministically triggerable. Full per-item detail and the
adversarial verification live in the
[plan](../plans/2026-07-27-hardening/README.md). This pass also resolves six
items from the [2026-07-26 deferred
list](../issues/2026-07-26-backend-optimization-deferred.md).

## Security

- **Invite code raised to 128 bits** (`store/workspaces.go`, R2).
  `generateInviteCode` was `fmt.Sprintf("%06d", n%1_000_000)` — a 10^6
  keyspace that was the *sole* credential for joining a workspace on an
  authenticated-but-unratelimited endpoint, so it was brute-forceable across
  tenants. Now 128 bits from `crypto/rand`, hex-encoded (32 chars); the
  `text` column fits unchanged and the collision-retry loop still guards
  uniqueness. The join-endpoint rate limit that was the second suggested
  mitigation remains deferred (entropy alone removes the brute-force). Test:
  `TestGenerateInviteCode_KeySpace` (pins the format; 2000 draws never
  collide — a 10^6 space collides near the ~1.2k birthday bound).
- **Refresh-token rotation made reuse-safe** (`store/sessions.go`, F1).
  `RotateSession`'s revoking UPDATE now carries `AND revoked_at IS NULL` and
  checks the row count, so replaying an already-rotated token affects zero
  rows → `ErrConflict` and mints nothing, instead of revoking-then-inserting
  a second live session from one token.
- **Upstream error detail no longer reaches clients** (`api/github.go` F12,
  `agent/anthropic.go` F17). The GitHub 502 branch logs the transport error
  server-side and returns a generic message; in-stream Anthropic errors flow
  through `classifyInStreamError` onto the existing sentinels so a mid-stream
  failure gets the same taxonomy as a non-200, with raw provider detail kept
  off the wire.

## Concurrency / data integrity

- **ConvertIdea double-create race** (`store/ideas.go`, R1). Two concurrent
  converts both read `status='pending'`, both passed the check, both inserted
  a PRD — orphaning one (`prds.idea_id` is non-unique). Now an atomic
  conditional `UPDATE … SET status='converted' WHERE status<>'converted'`
  claims the conversion BEFORE inserting; the loser sees 0 rows →
  `ErrConflict` and rolls back. Deliberately a conditional UPDATE, not
  `FOR UPDATE` — the `ideas` SELECT policy joins `workspace_memberships`,
  which returns zero rows under `EvalPlanQual` and would silently break a row
  lock. Test: `TestConvertIdea_ConcurrentConvert` (real DB, 8 goroutines:
  exactly 1 winner, exactly 1 committed PRD, idea points at the winner).
- **EnsureProjectForPrd first-build race** (`store/roadmap_plan.go`, F3) now
  takes `pg_advisory_xact_lock(hashtext(prdID))`, mirroring
  `ReplaceRoadmapTree`, so two concurrent first-builds can't each insert a
  project and orphan one.
- **PRD UpdateSections check-then-act CAS** (`store/prds.go`, F4) is now an
  atomic `updated_at` compare-and-swap → `ErrConflict` (409), so concurrent
  editor saves can't silently drop a section edit (last-writer-wins).
- **Revision snapshot race** (`store/prd_revisions.go`, F14) maps the `23505`
  unique-violation (two snapshots both compute MAX+1) to `ErrConflict` →
  a clean retryable 409 instead of an opaque 500.
- **GitHub sync made atomic** (`api/github.go`, F5/F6): paginates until the
  cursor SHA is found (surfacing a "history truncated" signal instead of
  silently advancing the cursor) and inserts all synced commits in one store
  tx so a partial failure can't leave duplicates + a divergent cursor.
- **Roadmap tree assembly no longer drops subtrees** (`agent/roadmap_planner.go`,
  F2): builds a `parent → []childID` index and constructs top-down by id
  (deterministic, cycle-safe) instead of value-copying children mid-map-iteration.

## Robustness / error contract

- **DecodeJSON rejects trailing bytes** (`httpx/json.go`, R3).
  `Decoder.Decode` stops at the first value, so `{}{}` / `{} garbage` were
  silently accepted despite `DisallowUnknownFields`. A second `Decode` must
  now hit `io.EOF` or the body is rejected; an empty body still returns
  `io.EOF` from the first decode (body-less POST callers check for it). Test:
  `TestDecodeJSON`.
- **UpdateProjectUpdate maps a concurrent-delete to 404** (`store/project_updates.go`,
  R4). The final `UPDATE … RETURNING` Scan now maps `pgx.ErrNoRows` →
  `ErrNotFound`, mirroring the authorizing SELECT just above, instead of
  leaking an opaque error as a 500.
- **Coach idle-watchdog vs client disconnect** (`agent/coach.go`, F7): an
  idle-watchdog cancel is now distinguished from a client disconnect
  (`ErrUpstreamStall`) so it emits an SSE error + records a circuit failure
  rather than being masked as a harmless hang-up.
- **Bounded persistence contexts** (`agent/coach.go`/`telemetry.go`, F11): the
  detached terminal-write contexts share one `detachedContext` helper with a
  10s timeout, so a wedged write can't hang its goroutine (and held dispatch
  permit) forever.
- **Truncated answers flagged, not presented as finished** (`agent/anthropic.go`,
  F13): `stop_reason == "max_tokens"` sets `Turn.Truncated` and appends a
  visible "[response truncated…]" marker to the persisted/replayed text.
- **Context-length retry no longer diverges from the client** (`agent/coach.go`,
  F16): the auto-tighten retry fires only while `streamed.Len()==0` (nothing
  flushed to the bubble yet); once deltas have streamed, we fail the turn so
  what the user saw is exactly what persists.
- **Panic recover writes an SSE error, not JSON** (`api/middleware.go`, F21),
  and **empty tool-call bubbles are skipped** (`api/conversations.go`, F22).
- **create-revision rejects a genuinely bad body** (`api/prds.go`, F15):
  400s on any decode error except `io.EOF` (the optional body-less POST).
- **GitHub rate-limit surfaces as a retryable 429** (`github/github.go`, F20)
  instead of an opaque 502.

## Bounds / abuse surface

- **AI roadmap output capped** (`agent/roadmap_planner.go`, F18): `ExpandNode`
  caps subtasks at 40 (mirroring `assembleTree`'s `maxNodes`) and trims titles
  by rune (200) so CJK isn't split — same inline pattern as commit subjects.
- **Commit subjects rune-trimmed** (`github/github.go`, F19); **project-update
  content capped** at 5000 runes via a shared `validateUpdateContent` used by
  add + edit (`api/project_updates.go`, F10).
- **Project-updates timeline paginated** (`store/project_updates.go`, F9):
  `ListProjectUpdates` takes a composite `(created_at, id)` keyset cursor and
  `GetProject` embeds only the newest page (50) plus a forward-compatible
  `nextCursor`.
- **Frontend: non-author PRD edits made read-only** (`app/src/pages/prd/[id]/index.vue`,
  F8): `canEdit` mirrors the backend 403 rule, fields are `:readonly`, and a
  rejected save reverts to the last server value (a one-way binding never
  resets on its own). Test: `PrdEditor.test.ts`.

## Test isolation

- **`TestSessions_RotateIsReuseSafe` made rerun-safe** (`store/sessions_test.go`).
  It used fixed token-hash literals, but `sessions_token_hash_key` is a
  table-global UNIQUE constraint and rotation revokes rather than deletes — so
  a previous run's still-present revoked row collided on any re-run. Hashes
  are now suffixed with the per-run user id. (Test-only defect; real token
  hashes are random and never collide.)

## Verification

`go build ./…`, `go vet ./…`, `go test ./…` all green; the `DATABASE_URL`-gated
integration tests (`TestSessions`, `TestEnsureProject`, `TestProjectUpdates`,
`TestPrdRevisions`, `TestRoadmap`, `TestConvertIdea`, `TestGenerateInviteCode`)
ran live against real PostgreSQL (store package green twice in a row). Frontend
`pnpm test` (60/60) and `npx nuxi typecheck` (exit 0) green.
