# 2026-08-21 — Version-first slice, Stage 1: human Commit

Status: **complete** (2026-08-21). The first P2 product slice from
[`PRODUCT.md`](../../PRODUCT.md): "Without a version, people remain
stuck in the Idea stage." The system has working-version autosave
(PATCH + numbered revisions) but no product version control — the human
Commit does not exist.

## What PRODUCT.md requires of a Commit

An immutable, meaningful checkpoint confirmed by a human, recording a
name, primary audience, change summary, unresolved validation items,
and a Diff from the preceding Commit. Working versions stay mutable;
every Commit can start the next working version without rewriting
history. A Decision Log accompanies meaningful Commits (light in v1:
an optional decision note on the commit itself).

## Stage 1 (this increment) — the Commit act, API-first

- [x] Migration `prd_commits` (20260821150000): FORCE RLS, members
      read / Driver-or-admin insert, no update/delete — immutability is
      structural
- [x] `POST /api/v1/prds/{id}/commit` — snapshot + commit + diff in one
      transaction; `GET /api/v1/prds/{id}/commits` newest-first
- [x] CLI `insideout commit` / `insideout versions`
- [x] MCP tools `commit` / `versions` (16 tools total)
- [x] Diff unit tests; live verification (2026-08-21): two commits,
      second diff `{changed: 1, background: changed}`; migration 20/20
      ([changelog](../changelogs/2026-08-21-version-commit.md))

## Stage 2 — form a version now (readiness shipped 2026-08-21)

- [x] `GET /api/v1/prds/{id}/readiness` — per-audience gap disclosure
      (`internal/readiness`): priorities (must clarify now / should
      clarify this version / validate later), reader-facing reasons,
      no completeness score, never blocks a Commit
- [x] CLI `insideout readiness <prd-id>` + MCP tool `readiness` (17
      tools)
- [x] Live verified: decision audience disclosed 3 gaps with reasons;
      a "form a version now" Commit carried all 3 as unresolved items
- [x] Web rendering (2026-08-21): `/prd/{id}/versions` page — audience
      chips, readiness gaps with priorities and reasons, version list
      with per-commit diffs, carried items, and a commit dialog that
      shows the carry count; PRD toolbar gains the Versions entry;
      en/zh strings. Verified by a full interaction widget test
      (chips → gaps → dialog → commit → list+detail update, 44/44
      suite) and deployed to the hosted app.

## Sources

- PRODUCT.md: thesis, "Product version control", "The Coach and the
  first workable version"
- Current model: `prds.current_revision`, `prd_revisions` (anonymous
  snapshots), statuses draft/reviewing/approved/rejected
