# 2026-09-18 — Cobalt Flutter against the live API

Status: **Finished** — signed-in Flutter web walk on a local seed
`/api/v1` completed the loop; Coach web POST SSE and dock IDs are wired.

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
- [x] Home lists server workspaces
- [x] Ideas capture persists
- [x] PRD + Coach live SSE (fetch POST; offline template is one fast delta)
- [x] Human Commit appears in the stack
- [x] Roadmap item appears in a band (Next: new nodes are `pending`)
- [x] Evidence loads; dock hops real IDs
- [x] `flutter analyze --no-fatal-infos && flutter test` (57 tests)
- [x] Changelog + board + HANDOFF

## Review

Skipped: this is a connect-and-verify slice. The user asked for the
goal prompt so an agent can implement it.
