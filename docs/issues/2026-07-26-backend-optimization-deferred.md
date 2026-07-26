# Backend optimization review — deferred findings

**Date:** 2026-07-26
**Status:** open — bounded fix prompts below; none are push-blockers for the
current pass. The five highest-value findings from this review were fixed
in the same pass (see `docs/changelogs/2026-07-26-backend-optimization-pass.md`):
request-body cap + server timeouts, the dispatcher conversation-lock TOCTOU,
the roadmap move/create cycle race, the coach success-path detached-context
persist, and the unchecked SSE frame-index assertions.

Two independent adversarial workflow audits (security/concurrency/SQL lenses)
confirmed the items below against source. Ranked most-severe first.

## HIGH — invite code is a brute-forceable cross-tenant credential

`generateInviteCode` (`server/internal/store/workspaces.go:36-43`) returns
`fmt.Sprintf("%06d", n%1_000_000)` — ~20 bits. `JoinWorkspace`
(`workspaces.go:101-136`) treats knowing the code as the *sole* authorization
to become a `member`, and `POST /api/v1/workspaces/join` is behind
`requireAuth` only, with no rate limiting or failed-attempt lockout anywhere
in the middleware chain. Any self-serve authenticated user can spray 6-digit
codes and, on a hit, gain member read/write over another tenant's projects,
ideas, and PRDs.

> **Product note:** the 6-digit code is deliberately human-shareable, so
> raising its entropy is a UX decision, not a pure code fix. The two
> mitigations are independent — either weakens the attack, both remove it.

**Bounded fix prompt:**
> In `server/internal/store/workspaces.go`, replace the `n%1_000_000` code
> with a `crypto/rand` base32 string of ≥8 chars (keep it typeable; no
> ambiguous glyphs), and add a per-user failed-attempt rate limit / lockout
> on `POST /api/v1/workspaces/join` (a small in-memory counter keyed by user
> id is enough at this scale — see the agent coach's `CheckRateLimit` for the
> existing pattern). Verify with a real-DB test that a wrong code no longer
> joins and that N rapid wrong codes start returning 429.

## MEDIUM — ConvertIdea double-create race

`ConvertIdea` (`server/internal/store/ideas.go:174-211`) reads the idea with
an unlocked `SELECT`, guards `status == "converted"`, then inserts a PRD +
conversation and does an unconditional `UPDATE … SET status='converted'`.
Under READ COMMITTED a double-click (two concurrent converts) both read
`refining`, both pass the guard, both insert — `prds.idea_id` is only a
non-unique index, so two PRDs + two conversations end up referencing one idea
and the earlier pair is orphaned.

**Bounded fix prompt:**
> Claim the idea atomically before inserting: run
> `UPDATE insideout.ideas SET status='converted' WHERE id=$1 AND status<>'converted'`
> first and return `ErrConflict` when `RowsAffected()==0` (the row lock
> serializes the two txns; the loser sees 0 rows and rolls back before
> inserting). Optionally back it with a partial unique index
> `ON insideout.prds(idea_id) WHERE idea_id IS NOT NULL`. Verify with a
> concurrent-convert real-DB test asserting exactly one PRD.

## MEDIUM — no rate limit on login/register + argon2 timing enumeration

`register`/`login` (`server/internal/api/auth.go:33,87`) have no throttling,
and login's argon2 verify path can leak account existence via timing. Same
class of gap as the join endpoint above.

**Bounded fix prompt:**
> Add a per-IP + per-account rate limit on `POST /auth/login` and
> `POST /auth/register` (reuse the coach `CheckRateLimit` shape), and make
> the login path run an argon2 hash even on unknown-email so success/failure
> timings are indistinguishable.

## MEDIUM — PRD UpdateSections check-then-act CAS

`UpdateSections` (`server/internal/store/prds.go:118-155`) reads `updated_at`,
compares in Go, then writes — a classic check-then-act that two concurrent
editor saves can both pass, last-writer-wins silently dropping a section edit.

**Bounded fix prompt:**
> Make it atomic: `UPDATE insideout.prd_sections … WHERE prd_id=$1 AND
> updated_at=$expected` and return `ErrConflict` when `RowsAffected()==0`.

## LOW — `EnsureProjectForPrd` double-create race

`roadmap_plan.go:165-210`: two concurrent `POST /prds/{pid}/build` both see
`prds.project_id NULL`, each insert a project (with a generated roadmap), and
the last `UPDATE` wins, orphaning one project + roadmap. The build handler is
not behind the conversation dispatcher lock.

**Bounded fix prompt:**
> Serialize per-PRD builds inside the tx with the same pattern
> `ReplaceRoadmapTree` already uses: `SELECT pg_advisory_xact_lock(hashtext($1))`
> keyed on `prdID` right after `requireMember`. The second call blocks, then
> reads the now-set `project_id` and returns the existing project.

## LOW — coach `loadHistory` over-fetches

`coach.go:327` / `agent_messages.go:62` load *all* messages for a conversation
then truncate to the window in Go. Push the `LIMIT` and the `NOT content=''`
filter into SQL.

## LOW — GitHub sync N+1 (one tx per commit)

`github.go:106` / `project_updates.go:23` open a transaction per synced commit.
Batch into a single multi-row insert in one tx.

## LOW — reaper seq-scans `ai_runs` (no index)

`agent_messages.go:152` (the stale-run reaper) scans `ai_runs` with no index on
`status`/`updated_at`. Add a partial index:
`CREATE INDEX ON insideout.ai_runs (updated_at) WHERE status IN ('pending','running')`
(insideout schema only).

## LOW — `ListWorkspacesForUser` correlated count

`workspaces.go:167` runs a correlated subquery per workspace row. Fold the
member/project counts into the main query with `LEFT JOIN … GROUP BY` or
lateral counts.

## LOW — provider/error detail leaks to clients

Two spots return raw upstream detail: `github.go:90-93` echoes the GitHub API
error verbatim, and `anthropic.go:154,263` forwards provider error detail over
SSE. Map both to stable, client-safe error codes/messages; log the raw detail
server-side.
