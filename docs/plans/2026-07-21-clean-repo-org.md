# 2026-07-21 — Reorganize repo docs per clean-repo-org

Status: **complete**.

Applied the clean-repo-org skill format to this repository: thin agent-router
files, modular `docs/` records, a comprehensive staleness audit of every
existing doc against the actual code, and the docs-recording guardrail.

## Scope (as executed)

- Root `CLAUDE.md` + `AGENTS.md` thin routers (none existed).
- `docs/index.md` doc map; `docs/architecture/` (index + 5 files);
  `docs/usage/` (index + 2); `docs/changelogs/` (index + a dated folder for
  the Go rewrite + a dated file for this reorg); `docs/plans/README.md`;
  `docs/issues/` (index + records); `docs/learning/` (index + 5 records);
  `docs/practices/` (index + 5 records); `docs/HANDOFF.md`;
  `docs/history/` attic (index + 6 retired JuanLeMe-era files with
  historical banners).
- Docs-recording guardrail installed (git pre-commit warn hook,
  `[checkpoint]` commit-msg gate, Claude Stop hook, `core.hooksPath`).
- Comprehensive staleness audit: 18 auditors (one per doc) verified every
  current-system claim against the code; all 28 findings fixed.
- Privacy: real Supabase project ref redacted from two docs; full sweep for
  refs/hostnames/passwords/emails across `docs/` came back clean (the
  `demo12345` seed credential is intentionally public dev data).

## Decision changes during the task (user direction, mid-flight)

1. **Bilingual bug book retired.** The original scope kept
   `docs/en/BUGS/` + `docs/cn/BUGS/` as a pre-existing bilingual
   convention. The user directed otherwise: English-only docs, bugs merged
   into `docs/issues/`. Executed: the ten English records moved to dated
   `docs/issues/2026-07-2x-bug-0NN-*.md` files (keeping their BUG-NNN
   identity), Chinese copies deleted, all cross-references rewritten
   repo-wide, and one concurrently-authored bilingual index discarded.
2. **English-only extended to the live top-level docs.** `README.md` keeps
   its bilingual product copy (product-facing, not `docs/`); `INIT.md`,
   `PLAN.md`, `TODO.md` were converted to English with audit fixes baked
   in; `NOTE.md` was superseded by `docs/index.md` and removed. Exception,
   recorded: `docs/plans/2026-07-20-go-rewrite/` keeps its bilingual text —
   it is a historical decision record, and rewriting history wholesale
   risks corrupting it; staleness there was fixed with annotated
   supersession notes instead.

## Parallelization (as executed)

One 23-agent workflow fan-out, all units independent:

- **Audit lane (18 agents)** — one auditor per existing doc (+1 legacy
  triage), each verifying claims against the code with a shared
  ground-truth fact sheet, returning structured findings. Result: 28
  findings (3 fully-legacy files, stale langchaingo/env-var/mock-mode
  claims, two wording errors in the freshly written architecture docs, a
  self-contradiction in TODO.md, dead links).
- **Author lane (5 agents)** — usage docs, the Go-rewrite changelog folder,
  5 learning records, 4 practice records, bug-book indexes (the last
  discarded per decision change 1). Distinct target files, no conflicts;
  coordinator wrote architecture docs, routers, and all indexes, and
  applied every audit fix sequentially.

No worktrees needed (docs only, distinct files per lane).

## Checklist

- [x] Plan record opened first and linked from `docs/plans/README.md`.
- [x] Skill contract + references read; repo surveyed; leaks identified.
- [x] Root `CLAUDE.md` (thin router).
- [x] Root `AGENTS.md` (thin router).
- [x] `docs/index.md` doc map.
- [x] `docs/architecture/`: index, backend, database-and-rls,
      prd-coach-agent, frontend, deployment.
- [x] `docs/usage/`: README, local-development, deployment.
- [x] `docs/changelogs/`: README + `2026-07-20-go-rewrite-and-rls-cutover/`
      (index, summary, verification, migration-notes) +
      `2026-07-21-docs-reorganization.md`.
- [x] `docs/plans/README.md` index.
- [x] `docs/issues/`: README + 10 migrated bug records + the
      tracked-tool-dirs issue.
- [x] `docs/learning/`: README + 5 records.
- [x] `docs/practices/`: README + 5 records (incl. the guardrail record).
- [x] `docs/HANDOFF.md` seeded with real content.
- [x] Guardrail installed and its warn path verified.
- [x] Legacy triage executed: 6 files to `docs/history/` with banners +
      index; `docs/LICENSE` → root `LICENSE`; `docs/package.json`,
      `docs/INSTALL.md`, `docs/NOTE.md` removed (superseded).
- [x] All 28 audit findings fixed (incl. the 2 in my own architecture
      docs — the audit caught the coordinator too).
- [x] Privacy redaction + full sweep.
- [x] Link check: every relative link in `docs/`, routers, README resolves.
- [x] `check_target_routers.py` passes.
- [x] Plan marked complete, no unchecked items.

## Engineering review / TDD note

Documentation-organization task; no code behavior changed, so no
plan-eng-review or TDD cycle (the record's purpose — testing code behavior —
has no object here). Verification was the audit fan-out, the link check, the
router checker, and the guardrail self-check.

## Record closure matrix

| Aspect | Status |
|---|---|
| Architecture | Done — `docs/architecture/` (6 files) |
| Usage | Done — `docs/usage/` (3 files) |
| Issues | Done — bug book merged in + tool-dirs issue + index |
| Changelog | Done — rewrite folder + reorg entry + index |
| Learning | Done — 5 records + index |
| Practices | Done — 5 records + index |
| Plan | This file, complete |
| Routers/indexes | Done — CLAUDE.md, AGENTS.md, doc map, every folder README |
