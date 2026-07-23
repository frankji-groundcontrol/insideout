# Agent Contract — InsideOut

Operating contract for Codex and any coding agent working in this
repository. Keep this file a thin router to the modular docs under `docs/`
([doc map](docs/index.md)); it stays in parity with [CLAUDE.md](CLAUDE.md).

## Before editing

- Read the [doc map](docs/index.md), the relevant
  [architecture doc](docs/architecture/index.md), and
  [docs/HANDOFF.md](docs/HANDOFF.md) for in-flight state.
- For multi-step work, open a dated plan in
  [docs/plans/](docs/plans/README.md) **before editing** and maintain it as
  the live checklist; close it only when every item is resolved.
- Preserve user changes; do not revert unrelated work.
- Karpathy-inspired coding baseline: state assumptions, prefer the simplest
  useful change, edit surgically, define verification against explicit
  goals before claiming completion.

## Build, test, verify

Commands and environment reference:
[docs/usage/local-development.md](docs/usage/local-development.md). The
non-negotiables:

- No mocks. DB-dependent changes must pass the `DATABASE_URL`-gated
  integration tests (`server/internal/store/authz_test.go`) against real
  PostgreSQL; for substantial behavior changes, write the failing test
  first (test-driven development), then the smallest passing change.
- Never write outside the `insideout` schema
  ([why](docs/architecture/database-and-rls.md)).
- Exercise streaming (SSE) endpoints against a live server before calling
  them done ([practice](docs/practices/2026-07-21-live-exercise-streaming-endpoints.md)).
- For plan-level work, run an engineering review of the plan (or record why
  not) before implementation.

## Privacy

- Never read or expose `.env`/`.env.local`; never commit secrets, tokens,
  or credentials.
- Redact provider/project identifiers, database hostnames, workspace IDs,
  personal emails, and local runtime paths in docs and examples.
- Raw private transcripts and personal PII must never be committed — hold
  them out untracked (do **not** gitignore them) and record them in a local
  untracked-file manifest.

## Docs and records

- `docs/` prose is English-only (the 2026-07-20 plan folder is bilingual as
  a historical record; keep exact non-English text only when quoting).
- Record meaningful changes in [docs/changelogs/](docs/changelogs/README.md):
  a dated file for ordinary changes, a dated folder with focused child
  files for large changes. The git guardrail warns on unrecorded source
  commits; `[checkpoint]`-tagged commits block unless the changelog entry,
  active plan, and `docs/HANDOFF.md` are staged
  ([practice](docs/practices/2026-07-21-docs-recording-guardrail.md)).
- Keep the record system whole and indexed:
  [architecture](docs/architecture/index.md) when structure changes,
  [usage](docs/usage/README.md) when commands/install/operator flows
  change, [issues](docs/issues/README.md) for bugs and structural debt,
  [learning](docs/learning/README.md) for reusable lessons,
  [practices](docs/practices/README.md) for repeatable methods — plus the
  nearest README index for every added, moved, or retired record.
- Keep implementation modular: files under ~350 lines, single
  responsibility, no chunky monolithic scripts. A chunky or
  mixed-responsibility file that can't be split now gets a dated
  modularity issue in [docs/issues/](docs/issues/README.md) with the
  target structure and a bounded fix prompt.

## Review checklist

Before finishing: verification commands ran and passed (or the failure is
reported honestly); affected docs surfaces and indexes updated; links
resolve to existing files; no secrets or private identifiers committed;
router files (this file + CLAUDE.md) stay thin and in parity.
