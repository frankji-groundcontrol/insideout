# 2026-07-25 — Build-from-PRD runs the LLM before the live-count conflict guard

**Status:** open, deferred. Not a data-corruption bug — a wasted-cost /
latency wart. The in-transaction advisory-locked guard remains the correctness
authority (see decision D8 in the
[collab plan](../plans/2026-07-24-roadmap-canvas-collab.md)); this issue is
purely about not throwing away an LLM call.

## Symptom

"Draft the roadmap" (build-from-PRD) against a project that already has a
non-empty roadmap pays the full LLM `PlanMVP` call and only then learns the
tree must be replaced, returning 409. The generated tree is discarded.

## Root cause

In `server/internal/api/roadmap_ai.go`, `handleBuildFromPrd` calls
`s.planner.PlanMVP(...)` (an LLM round-trip, seconds + token cost) and only
afterwards calls `s.store.ReplaceRoadmapTree(..., req.ExpectedCount, ...)`,
which is where the non-empty-without-count case yields the 409
(`replace_conflict`). On the *first* build call `req.ExpectedCount` is nil, so
the path is: decode → authz → **LLM** → replace → 409. The expensive step sits
in front of the cheap conflict decision.

## Why it is safe to defer

Correctness is unaffected. `ReplaceRoadmapTree` re-checks the count inside its
transaction behind a per-project advisory lock (`pg_advisory_xact_lock`), so a
stale or missing count can never wipe a live tree — the 409 the user sees is
the correct, authoritative answer. The only loss is the LLM latency and tokens
spent on a tree that is then dropped, plus the user waiting through it before
the confirm prompt appears.

## Fix (when picked up)

Add a cheap pre-check *before* `PlanMVP`: when `req.ExpectedCount == nil`, read
the live node count for the project; if it is non-zero, return the 409
(`replace_conflict` with `liveCount`) immediately and never call the planner.
When `ExpectedCount` *is* provided (the confirm retry) skip the pre-check and
let the in-transaction guard decide as today. The pre-check is itself racy
(check-then-act) and that is fine — it is a cost/latency short-circuit for the
common "oops, it is non-empty" case, **not** a replacement for the
advisory-locked guard, which stays the authority. A `CountRoadmapNodes`-style
store read is the only new surface needed.

## Verification (when fixed)

Real-DB handler test: a build call with no body against a project that already
has N>0 nodes returns 409 *without* invoking the planner (assert via the
existing planner interface seam / a no-op planner that records calls). The
confirm-retry path (body carries the live count) still reaches the planner and
replaces under the lock.
