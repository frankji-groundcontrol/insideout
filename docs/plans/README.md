# Plans and task board

This is the authoritative status board for concurrent repository work. Update
the table whenever a task starts, changes state, becomes blocked, or reaches a
checkpoint. Each linked plan owns its detailed checklist and decisions;
`docs/HANDOFF.md` only explains how the next agent should resume.

## Current board

| Priority | Task | Status | Next action | Blocker / note |
| --- | --- | --- | --- | --- |
| P1 | [Product-experience baseline](2026-08-13-product-experience-baseline.md) | **Finished — awaiting checkpoint** | Recheck the documentation diff and links, then partial-stage and checkpoint it independently. | Shared README and index files contain changes from other tasks. |
| P2 | First version-first product slice from [`PRODUCT.md`](../../PRODUCT.md) | **Pending — ready to plan** | After the finished workstreams are checkpointed, create a dated implementation plan for the smallest useful slice and run its engineering review. | None; planning has not started. |

**Status meanings:** In flight = active work; Pending = not started; Finished
= recorded scope and verification complete in the local worktree; Completed =
a checkpoint commit exists; Blocked = work cannot safely progress. Because
several tasks share documentation files, do not use `git add .`; review and
stage the appropriate hunks for one task at a time.

Unscheduled limitations and debt stay in [`docs/TODO.md`](../TODO.md) and
[`docs/issues/`](../issues/README.md). Promote one to this board when it is
selected for work; backlog is not in-flight work.

## Completed

- [2026-07-27 — Shared auth modal + Design-QA record surface](../changelogs/2026-07-27-auth-door-modal.md)
  — `/login` + `/register` rebuilt on one shared `AuthDoor` dialog with the
  a11y behavior extracted from `BaseModal` into `useDialogA11y`; new
  `docs/design-qa/` surface for verbatim user design feedback plus the
  standing router rule. Typecheck green; 16 files / 60 tests passed.
- [2026-07-30 — Env catalog, TUI, and contract-scoped propagation](2026-07-30-env-catalog-propagate.md)
  — `env.sh init|edit|check|propagate` around a machine-honest `.env.example`
  catalog, contract-scoped `app/.env` generation, and a `dev.sh` preflight
  gate; env unit batteries 83/83 green.
- [2026-08-13 — Task board and handoff responsibility correction](2026-08-13-task-board-and-handoff.md)
  — `docs/plans/README.md` is now the authoritative concurrent task board and
  `docs/HANDOFF.md` a concise resume guide; agent routers and the doc map
  state the responsibility split. Documentation governance only.
- [2026-07-27 — Cross-surface security hardening pass](2026-07-27-hardening/README.md)
  — security and concurrency fixes across the Coach, Roadmap, GitHub sync,
  project updates, and authorization; real-DB verification recorded.
- [2026-07-24 — Roadmap canvas: collaborative model](2026-07-24-roadmap-canvas-collab.md)
  — partial-update safety, guarded tree replacement, review mode, attribution,
  freshness, transitions, sibling bands, and minimap; all planned work complete.
- [2026-07-23 — Comprehensive frontend](2026-07-23-frontend-pages.md)
  — shared application chrome, workspace/member surfaces, PRD revisions, and
  the detachable Coach panel; all phases complete.
- [2026-07-23 — Ink & Seal landing](2026-07-23-ink-seal-landing.md)
  — current visual world and public landing, complete.
- [2026-07-21 — Documentation reorganization](2026-07-21-clean-repo-org.md)
  — modular documentation and record surfaces, complete.

## Superseded or historical foundations

- [2026-07-24 — Roadmap tree on a canvas](2026-07-24-roadmap-canvas.md)
  — completed foundation, superseded by the collaborative-canvas follow-up.
- [2026-07-22 — Idea → Reality](2026-07-22-idea-to-reality.md)
  — completed implementation record; its Prisma styling was later reverted,
  and its linear product assumptions are superseded by `PRODUCT.md`.
- [2026-07-21 — PRD agent harness redesign](2026-07-21-prd-agent-harness/index.md)
  — historical plan whose core hardening, fact-ledger, critic, streaming, and
  run-lifecycle mechanics largely shipped. Do not reimplement them or execute
  its superseded fixed-stage, fixed-section, and completion-gate experience;
  regenerate the next implementation plan from `PRODUCT.md` and current code.
- [2026-07-20 — Go rewrite](2026-07-20-go-rewrite/README.md)
  — completed historical foundation. Current runtime authority is
  [`docs/architecture/`](../architecture/index.md).
