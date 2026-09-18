# 2026-09-18 — Cobalt Flutter restyle

Status: **Finished** — lock implemented; analyze + tests green; browser walk recorded.

## Why

The user accepted the Cobalt card lock for the live client:

> yes, use this one for flutter, give me the goal prompt

Exploration is closed. This plan is the implementation slice.

## Design

Implement the accepted lock. Do not invent a new language.

- Goal prompt (paste this):
  [`2026-09-18-cobalt-flutter-goal.md`](2026-09-18-cobalt-flutter-goal.md)
- Lock screens:
  [`docs/design-system/cobalt/`](../design-system/cobalt/README.md)
- QA thread:
  [`docs/design-qa/2026-09-17-cobalt-layouts.md`](../design-qa/2026-09-17-cobalt-layouts.md)

## Out of scope

- Reopening color, layout, or Ink & Seal.
- New product verbs, schema, or API routes.
- Android release (still SDK-blocked).

## Checklist

- [x] User accepted Cobalt for Flutter
- [x] Goal prompt written
- [x] Lock screens copied into the repo
- [x] Failing Cobalt token tests
- [x] Theme rename: InkSeal → Cobalt; seals off live chrome
- [x] Bottom dock in `AppScaffold`
- [x] Home / Ideas / PRD+Coach / Commit / Roadmap / Evidence match locks
- [x] Landing + AuthDoor retokened (no chops)
- [x] `DESIGN.md` + frontend architecture + design-system changelog
- [x] `flutter analyze --no-fatal-infos && flutter test` (infos only; 54 tests)
- [x] Browser walk: light/dark × zh-CN/en-US × product loop
  ([QA](../design-qa/2026-09-18-cobalt-flutter-walk.md))
- [x] Changelog + board + HANDOFF

## Review

Skipped as a design review: the user already locked color and layout.
This slice implements pixels.
