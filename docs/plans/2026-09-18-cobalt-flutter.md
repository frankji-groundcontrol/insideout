# 2026-09-18 — Cobalt Flutter restyle

Status: **In flight** — lock accepted; Flutter still Ink & Seal.

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
- [ ] Failing Cobalt token tests
- [ ] Theme rename: InkSeal → Cobalt; seals off live chrome
- [ ] Bottom dock in `AppScaffold`
- [ ] Home / Ideas / PRD+Coach / Commit / Roadmap / Evidence match locks
- [ ] Landing + AuthDoor retokened (no chops)
- [ ] `DESIGN.md` + frontend architecture + design-system changelog
- [ ] `flutter analyze --no-fatal-infos && flutter test`
- [ ] Browser walk: light/dark × zh-CN/en-US × product loop
- [ ] Changelog + board + HANDOFF

## Review

Skipped as a design review: the user already locked color and layout.
This slice implements pixels.
