# 2026-09-18 — Cobalt Flutter against the live API

Status: **In flight** — local seed API is up; wiring for Coach SSE and
dock IDs is in. Signed-in Flutter web walk still required.

## Why

The Cobalt restyle walk used a fake `ApiClient`. The user asked for the
restyled frontend to work with the backend.

> you need to make the restyled frontend work with backend, give me
> the goal prompt to implement this

## Design

Connect, log in, exercise the existing loop, fix only wiring that
breaks. Do not restyle. Do not add routes or schema.

- Goal prompt (paste this):
  [`2026-09-18-cobalt-backend-goal.md`](2026-09-18-cobalt-backend-goal.md)
- Restyle:
  [`2026-09-18-cobalt-flutter.md`](2026-09-18-cobalt-flutter.md)

## Out of scope

- Color, layout, or Ink & Seal.
- New product verbs, schema, or API routes.
- `railway up --service app` unless the user asked to ship.

## Checklist

- [x] Goal prompt written
- [x] Real session (local seed API; hosted signup is invitation-only)
- [ ] Home lists server workspaces
- [ ] Ideas capture persists
- [ ] PRD + Coach live SSE
- [ ] Human Commit appears in the stack
- [ ] Roadmap Now item appears in a band
- [ ] Evidence loads; dock hops real IDs
- [ ] `flutter analyze --no-fatal-infos && flutter test`
- [ ] Changelog + board + HANDOFF

## Review

Skipped: this is a connect-and-verify slice. The user asked for the
goal prompt so an agent can implement it.
