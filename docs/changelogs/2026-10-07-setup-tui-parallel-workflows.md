# 2026-10-07 — Setup TUI plan gains a binding parallel-workflows execution strategy

> **Amended same day:** the workflow mechanism is **ZCode's
> dynamic-workflow feature** (facade scripts — `ask`/actor calls,
> fan-out, phase graphs, journal, returned artifact), per user
> clarification with link; the `.grok` Rhai workflow is cited only as
> prior art. Progress recorded under this plan the same day: slice 0
> skeptic review **PASS-WITH-CHANGES** applied (per-slot subpackages,
> `internal/clientconfig` home for the precedence helper, hidden
> prompts widened to every secret `prompt()` caller, `x/term` in the
> deps scope, boot-offer env injection, owner-pool migrate,
> `--no-start` parity), and the deps change landed — bubbletea
> v1.3.10, huh v1.0.0, x/term v0.46.0 — with baseline binary
> 17,827,666 bytes and 19 test packages green. Implementation slots
> not yet launched.

**Scope:** planning + the first implementation steps — the plan and
its execution strategy, the slice-0 review, and the isolated deps
change (`server/go.mod`/`go.sum` only; no other source). The setup TUI plan
([`docs/plans/2026-10-07-setup-tui.md`](../plans/2026-10-07-setup-tui.md))
now mandates workflow-driven implementation with maximal parallelism
(user direction: a serial pass is too slow). No source changed.

## What changed

- A binding **Execution strategy** section: implementation follows the
  established `.grok/workflows/finish-cobalt-neo.rhai` pattern —
  parallel worker slots over disjoint files, structured per-slot
  status (`done|partial|blocked`), one gate, repair on failure, then
  closure.
- A concrete six-slot track table with disjoint Go package
  boundaries (persistence / flow+UI / auth wiring / catalog+writer /
  probes / release pipeline), each slot's owned files and
  dependencies — after the prerequisite deps commit, all six can hold
  workers simultaneously.
- One prerequisite serial step made explicit: bubbletea/huh land in a
  lone deps commit with before/after binary-size numbers, so the
  shared `go.mod` never conflicts across parallel tracks.
- Serial-by-nature join points named (cmd wiring + flags parity,
  local-mode assembly, live TTY gates) and the shared-docs rule
  (per-task hunk staging, never `git add .`).

## Verification

- Slot ownership cross-checked against the plan's slices: every
  slice-1/2 deliverable maps to exactly one slot; the two real join
  points are the only places slots meet.
- The referenced workflow file exists (`.grok/workflows/
  finish-cobalt-neo.rhai`) and its phases (Implement → Gate → Repair
  → Close) match the strategy's structure.
- Links resolve across the plan, this record, and HANDOFF.

## Operator notes

Nothing to deploy. The strategy binds whoever implements; the
engineering-review gate still precedes all code.
