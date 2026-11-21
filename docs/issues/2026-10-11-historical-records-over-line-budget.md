# Historical records over the line budget (annotated-only growth)

> **Closed 2026-11-21 as not applicable.** The [2026-11-21 responsibility
> rule](../../CLAUDE.md) (review 5, COMP-05) replaced the ~350-line rule: source
> code has one responsibility per file and shared file and function thresholds
> ([config/code-budget.json](../../config/code-budget.json)); workflows, YAML
> and prose have no line rule. These dated records are prose; the bounded fix
> prompt below no longer applies.

**Date:** 2026-10-11
**Kind:** structural debt (line budget, policy-bound)
**Files:**

- [`docs/plans/2026-10-07-setup-tui.md`](../plans/2026-10-07-setup-tui.md) — 361 lines
- [`docs/history/nuxt-migration.md`](../history/nuxt-migration.md) — 605 lines
- [`docs/plans/2026-07-21-prd-agent-harness/plan.md`](../plans/2026-07-21-prd-agent-harness/plan.md) — 500 lines

## Why they are over

Each crossed the ~350-line budget through appended italic
`Corrected/Superseded` annotations added by the docs-order pass
(2026-10-11), not through code growth.

## Why they are not split now

All three are dated records (a retired migration spec, two historical
plans). Splitting a dated record violates the record-immutability policy
the same pass enforces: original sentences stay, corrections annotate.
This differs from
[`docs/changelogs/README.md`](2026-10-11-changelogs-readme-line-budget.md),
which is a **living index** and was split into per-month files.

## Bounded fix prompt

If any of these needs another appended record, first extract the file
into a dated folder with an `index.md` (the changelogs
`index/YYYY-MM.md` pattern): the original file moves verbatim under the
folder, the index links it plus the annotations. Until then: no action —
growth is annotation-only.
