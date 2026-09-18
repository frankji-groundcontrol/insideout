# 2026-09-18 — Cobalt Flutter browser walk

Walk of the restyled Flutter web client (lock preview of the live
widgets; `.env` / hosted login were not available). Light and dark,
zh-CN and en-US. Screenshots:
`/opt/cursor/artifacts/screenshots/` (`landing-*`, `home-*`, `ideas-zh`,
`prd-coach-zh`, `commit-zh`, `roadmap-zh`, `evidence-zh`, `dock-en-dark`).

## What the screens show

Home matches lock F (index wash + giant `03` + numbered cards + dock).
Ideas matches lock A (giant 想法, stadium capture, three posters).
PRD + Coach matches lock D (split desk, dark serif Coach column).
Roadmap matches lock B (Now / Next / Later shade bands with item cards).
Commit empty state is a centered card on the field (not a 300px list).
Evidence is cards on a field with the dock. No left rail. No celadon,
vermilion, chops, or wax stamps.

## Follow-ups seen on the walk (not new language)

Audience chips on Commit were raw English API keys on zh-CN; they now
use `versions.audience.*`. Coach SSE was not reachable (no local
`DATABASE_URL`, invitation-only hosted signup).

Record: [`docs/changelogs/2026-09-18-cobalt-flutter-restyle.md`](../changelogs/2026-09-18-cobalt-flutter-restyle.md).
