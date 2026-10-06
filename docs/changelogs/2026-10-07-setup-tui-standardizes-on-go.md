# 2026-10-07 — Setup TUI standardized on Go (plan revision)

**Scope:** planning only — a material revision of the setup TUI plan
per user direction. No source changed. Plan:
[`docs/plans/2026-10-07-setup-tui.md`](../plans/2026-10-07-setup-tui.md).

## What changed

- **Decision:** the guided setup experience is Go —
  `insideout setup`, a subcommand of the existing CLI binary —
  superseding the same-day initial sketch of a scripts-side Python
  front door. Rationale: one language for product and setup; the
  login/config/migrate/probe capabilities are already Go, so the TUI
  composes in-process functions; no growth of the Python surface.
- **Plan consequences of the language call:**
  - TUI framework pinned to bubbletea + huh; only the setup packages
    import it (server-binary size measured and recorded as a gate).
  - Config persistence moved up into slice 1 (a 0600 config file with
    env-var precedence replaces the printed-export-lines stopgap the
    Python sketch needed).
  - The `.env.example` catalog is ported to Go with tests asserting
    the documented form rules; the Python reader stays green untouched,
    and the duplication is tracked for consolidation in
    [an issue](../issues/2026-10-07-env-catalog-go-python-duplication.md).
  - Probes become native (pgx ping, net/http) instead of the
    `pg_isready` bash pattern; migrate runs in-process.
  - A layer-split section records what stays scripts (bare-machine
    module install, pre-Go path) and why the plan does not retire them.
  - Flags parity: `insideout setup --hosted | --local --postgres
    bundled|own` mirrors `setup.sh`, tested; the script remains the
    pre-binary agent fallback.
- **Slice list after revision:** 0 engineering review (gate) · 1
  remote + config persistence · 2 local mode (Go catalog, writer,
  probes, migrate) · 3 re-verification (`setup --check`/`doctor`) ·
  4 optional release pipeline.

## Verification

- Plan-internal consistency check: the slices, decisions, risks, and
  non-goals now agree on Go; the Python tooling is consistently scoped
  as untouched-and-green, not removed.
- Links resolve across the plan, this record, the new issue, the board
  row, and HANDOFF (mechanical check).

## Operator notes

Nothing to deploy; nothing behaves differently yet. The engineering
review gate precedes slice 1 code.
