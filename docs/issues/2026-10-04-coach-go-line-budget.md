# 2026-10-04 — `server/internal/agent/coach.go` over the ~350-line budget

> **Resolved 2026-10-10.** `server/internal/agent/coach.go` is 281 lines and the loop mechanics live next door:
> `coach_loop.go` (118, `runLoop`/`loadHistory`) and `coach_fail.go` (74, `failTurn`), pure moves checked at their
> own commits (`8ef010a`, `6725655`); see
> [backend splits](../changelogs/2026-10-10-close-open-gaps/backend-splits.md). The coach defects fixed on top of the
> split are in [backend fixes](../changelogs/2026-10-10-close-open-gaps/backend-fixes.md). `userTexts` was deleted
> (`2f9c3e9`) when record_fact quote grounding began reading the conversation's stored turns instead of the replay
> window, and `aiConcurrencyLimits`/`detachedContext` stayed in `coach.go`.
>
> The original record follows, unedited.

`coach.go` is 467 lines and mixes three responsibilities: the Coach
constructor/limits (env-driven concurrency, detached contexts), the
HTTP-facing turn orchestration (`HandleMessage` — validation, SSE setup,
turn persistence, the coach loop), and the loop mechanics
(`runLoop`, `failTurn`, `loadHistory`, `userTexts`). It crossed the budget
during the August coaching hardening; the 2026-10-04 remediation only
touched env knobs (2.6) and left the structure alone.

## Why not split now

The remediation waves landed behavioral fixes; splitting mid-wave would
have buried them. The file is still one coherent narrative (a turn, start
to finish) and the seams below are clean — record now, split at the next
structural touch.

## Target structure

- `coach.go` — `Coach` type, `New`, `HandleMessage`, `failTurn` (~250
  lines): the HTTP + turn-lifecycle responsibility.
- `coach_loop.go` — `runLoop`, `loadHistory`, `userTexts`,
  `aiConcurrencyLimits`, `detachedContext` (~180 lines): the agent-loop
  mechanics with no HTTP surface.

## Bounded fix prompt

"Move the loop mechanics (`runLoop`, `loadHistory`, `userTexts`,
`aiConcurrencyLimits`, `detachedContext`) from
`server/internal/agent/coach.go` into a new `coach_loop.go` in the same
package — no signature changes, per
[docs/issues/2026-10-04-coach-go-line-budget.md](2026-10-04-coach-go-line-budget.md).
Gate: `go build ./... && go vet ./... && go test ./...` green (the SSE
and retry tests in `internal/api` + `internal/agent` are the behavioral
pins). Both files end under ~350 lines. Record the split in a changelog
entry and close this issue."
