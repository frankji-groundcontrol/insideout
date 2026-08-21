# Plans and task board

This is the authoritative status board for concurrent repository work. Update
the table whenever a task starts, changes state, becomes blocked, or reaches a
checkpoint. Each linked plan owns its detailed checklist and decisions;
`docs/HANDOFF.md` only explains how the next agent should resume.

## Current board

| Priority | Task | Status | Next action | Blocker / note |
| --- | --- | --- | --- | --- |
| P1 | [insideout_owner + insideout_app roles](2026-08-19-owner-app-roles.md) | **Completed** | None — plan closed. | Shared instance cut over, Railway deployed with owner URL, app password rotated 2026-08-20 ([changelog](../changelogs/2026-08-20-owner-app-roles-shared-instance.md)). Autodeploy off — ship with `railway up --service server`. |
| P1 | [Restore Ink & Seal on Flutter](2026-08-19-restore-ink-seal.md) | **In flight** | Collaborative canvas (sibling bands, minimap) — the only open slice item. | Native fonts bundled and visually signed off 2026-08-20 ([changelog](../changelogs/2026-08-20-native-fonts-bundling.md)). |
| P1 | [Agent vocabulary v1](2026-08-22-agent-vocabulary.md) | **Completed** | Follow-ons: proposal acceptance workflow; persisted Focus pointers; Coach gap explanations. | context/focus/checkpoint/propose live via API/CLI/MCP; version stays human-only ([changelog](../changelogs/2026-08-22-agent-vocabulary.md)). |
| P1 | [CLI / MCP parity (one truth, many views)](2026-08-21-cli-mcp-parity.md) | **In flight** | Remaining write verbs: idea create/convert, PRD revisions via CLI/MCP. | MCP server + agent vocabulary live (21 tools). |
| P1 | [Roadmap parity + GitHub-driven progress](2026-08-21-roadmap-parity-and-github.md) | **Completed** | None — thread closed 2026-08-21. | Webhook, guide (API/CLI/MCP), installation tokens, and the idempotent evidence loop all live; dogfood project owns the repo binding ([evidence](../changelogs/2026-08-21-evidence-loop.md), [dogfood](../changelogs/2026-08-21-dogfood-and-cleanup.md)). |
| P1 | [Replace Anthropic env names with INSIDEOUT_LLM_*](2026-08-18-llm-env.md) | **Finished** | Done; Railway `server` has the new names and Supabase DSN. | Dedicated Railway Postgres removed 2026-08-18. |
| P1 | [Delete leftover Nuxt `app/`](2026-08-18-delete-nuxt-app.md) | **Finished** | Done; `app/` gone; env/compose/docs updated. | Historical changelogs still cite `app/` as of their dates. |
| P1 | [Nuxt → Flutter client](2026-08-17-flutter-client.md) (web + iOS + Android, full current surface) | **In flight** | Android release build when an SDK is available. Visual language moved to the restore plan. | Hosted walk 2026-08-18. Nuxt `app/` deleted. |
| P1 | [Audience view projections](2026-08-22-audience-views.md) | **Completed** | None — closed 2026-08-22. | View API + audience markdown export + CLI/MCP/web ([changelog](../changelogs/2026-08-22-audience-views.md)). |
| P1 | [Version-first slice — the human Commit](2026-08-21-version-commit.md) | **Completed** | None — slice closed 2026-08-21. | Commits, readiness, and web rendering all live; interaction-tested and deployed ([changelog](../changelogs/2026-08-21-version-commit.md)). |

**Status meanings:** In flight = active work; Pending = not started; Finished
= recorded scope and verification complete in the local worktree; Completed =
a checkpoint commit exists; Blocked = work cannot safely progress. Because
several tasks share documentation files, do not use `git add .`; review and
stage the appropriate hunks for one task at a time.

Unscheduled limitations and debt stay in [`docs/TODO.md`](../TODO.md) and
[`docs/issues/`](../issues/README.md). Promote one to this board when it is
selected for work; backlog is not in-flight work.

## Completed

- [2026-08-13 — Product-experience baseline](2026-08-13-product-experience-baseline.md)
  — `PRODUCT.md` rewritten as the canonical target experience from the
  completed product interview; `docs/INIT.md` reduced to a compatibility
  pointer. Docs only.
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
