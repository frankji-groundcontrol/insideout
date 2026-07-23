# 2026-07-21 — Documentation reorganization (clean-repo-org)

Reorganized the repository's documentation to the clean-repo-org layout,
driven by [the dated plan](../plans/2026-07-21-clean-repo-org.md).

## What changed

- **New surfaces**: `docs/index.md` (doc map), `docs/architecture/` (6
  files), `docs/usage/` (3), `docs/changelogs/` (this system),
  `docs/learning/` (5 records), `docs/practices/` (5 records),
  `docs/plans/README.md`, `docs/HANDOFF.md`, root `CLAUDE.md` + `AGENTS.md`
  thin routers, root `LICENSE` (moved from `docs/LICENSE`).
- **Bug book merged into `docs/issues/`** (user decision, mid-task): the
  bilingual `docs/en/BUGS/` + `docs/cn/BUGS/` pair was retired; the ten
  English records moved to dated `docs/issues/2026-07-2x-bug-0NN-*.md`
  files, the Chinese copies were deleted, and every cross-reference
  repo-wide was rewritten. Docs are English-only from here on (the
  2026-07-20 rewrite-plan folder keeps its bilingual text as a historical
  record).
- **Live docs converted to English and de-staled**: `README.md`, `INIT.md`,
  `PLAN.md`, `TODO.md` — including fixing stale claims found by a
  23-agent audit of every doc against the actual code (langchaingo
  presented as current, `AI_AUTH_TOKEN` env names, migration counts, a
  self-contradicting TODO item, superseded mock-mode/Tiptap decisions
  presented as current in the plan folder, and two wording errors in the
  new architecture docs).
- **Legacy retired to `docs/history/`** with historical-record banners:
  the JuanLeMe README/mindmap, the completed Nuxt-migration spec, the old
  dashboard wireframe, the workshop class reviews. Deleted outright:
  `docs/package.json` (stale build manifest), `docs/INSTALL.md` (superseded
  by `docs/usage/local-development.md`), `docs/NOTE.md` (superseded by
  `docs/index.md`).
- **Privacy**: redacted a real Supabase project ref from two docs; swept
  all docs for refs/hostnames/passwords/emails.
- **Guardrail installed**: docs-recording git hooks + checkpoint gate +
  Claude Stop hook — see
  [the practice record](../practices/2026-07-21-docs-recording-guardrail.md).

## Verification

- Link check: every relative markdown link under `docs/`, `CLAUDE.md`,
  `AGENTS.md`, `README.md` resolves (scripted check).
- Router check: the clean-repo-org `check_target_routers.py` passes against
  this repo.
- Guardrail self-check: `check-docs-recorded.sh --staged` warns without a
  changelog entry, silent with one.

## Follow-ups

- [Tracked coding-tool scratch directories](../issues/2026-07-21-tracked-tool-scratch-dirs.md)
  — awaiting a user decision.
