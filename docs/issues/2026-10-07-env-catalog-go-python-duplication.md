# 2026-10-07 — Env-catalog truth will be duplicated once the Go setup TUI lands

> **Resolved 2026-10-10.** Decision: option (b), keep both readers and make drift fail CI.
> `server/internal/setup/catalog/parity_test.go` feeds one skeleton and one live
> `.env` through both the Go catalog and `scripts/env_catalog.py` and diffs the
> results (`55a355c`); it runs in the existing `server (go + postgres)` job and
> fails, rather than skips, without `python3`. The same pass found that the CI step
> for the Python batteries had been running only the first of them
> (`b492843`). The decision is recorded in
> [usage/environment.md](../usage/environment.md). See
> [CI gaps](../changelogs/2026-10-10-close-open-gaps/env-ci-and-skill.md).
>
> The original record follows, unedited.

## Problem

The [setup TUI plan](../plans/2026-10-07-setup-tui.md) (revised
2026-10-07 to standardize on Go) introduces a Go reader of
`.env.example` (`server/internal/setup/catalog.go`) while the
established Python reader (`scripts/env_catalog.py`, driven by
`env.sh` and covered by the CI env-suites batteries) keeps working.
Two readers of one schema can drift: form rules (uncommented =
required, commented = default in effect), the secret-by-name decision,
status computation, and write behavior.

## Immediate guard (owned by the plan)

Go tests assert the documented form rules directly, and the Python
batteries stay green untouched in CI. Drift between them is therefore
*detectable*, but only by a human comparing behaviors.

## Resolution path (this issue)

After the Go setup slices land and are exercised:

1. Decide the single home for the truth: either (a) Go owns the
   catalog and the Python tools shrink to thin wrappers (or retire
   where `dev.sh`'s gate can call the binary instead), or (b) keep
   both readers permanently with an explicit parity test in CI that
   feeds one fixture `.env.example` to both and diffs the results.
2. Record the decision here and in
   [usage/environment.md](../usage/environment.md) so docs name one authority.

## Constraints

- `dev.sh`'s `env.sh check` gate and the CI env-suites job must not
  break during any consolidation step.
- The setup TUI itself must not silently diverge: its catalog behavior
  is covered by its own tests (plan, slice 2).
