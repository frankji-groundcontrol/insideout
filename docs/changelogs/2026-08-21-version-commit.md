# 2026-08-21 — Version-first slice Stage 1: the human Commit

Plan: [version-commit](../plans/2026-08-21-version-commit.md). First P2
product slice: PRODUCT.md's product version control — "without a
version, people remain stuck in the Idea stage."

## What changed

- Migration `20260821150000_prd_commits.sql`: `prd_commits` (name,
  primary audience ∈ decision|management|delivery|validation, change
  summary, carried unresolved items, decision note, recorded diff,
  frozen revision number, committer) under FORCE RLS (members read;
  PRD author/Driver or workspace admin writes). No update/delete paths
  exist — immutability is structural.
- `POST /api/v1/prds/{id}/commit` snapshots the working sections into
  `prd_revisions` and records the commit **with a section-level diff
  versus the previous commit** (`internal/prdcommit.Diff`: added /
  removed / changed per section, deterministic) — one transaction with
  the `current_revision` bump.
- `GET /api/v1/prds/{id}/commits` lists versions newest-first with
  committer names.
- CLI: `insideout commit --name --audience [--summary --unresolved…
  --note] <prd-id>` and `insideout versions <prd-id>`. MCP tools
  `commit` and `versions` join the frozen list (now 16 tools).

## Verification

- Unit tests for the diff (added/removed/changed, unchanged absent,
  deterministic, empty) — green; full suite, vet, gofmt green.
- Migration applied by Railway boot-migrate as owner (20/20).
- **Live** through the domain: register → idea → PRD → `commit v1`
  (8 sections added vs empty) → edit the `background` section →
  `commit v2` returned `{changed: 1, background: changed}` → `versions`
  lists both immutable versions with names + audiences. A wrong PATCH
  section key is rejected (`unknown section key`) — sections stay a
  fixed template set. Verification scratch users purged after.

## What is deliberately next (Stage 2 of the plan)

- "Form a version now": readiness as per-audience disclosed gaps
  (missing critical-question answers, ungrounded personas, carried
  assumptions) instead of any completeness score.
- Web rendering: version list + commit affordance in the Flutter client.
