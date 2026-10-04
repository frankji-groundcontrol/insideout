# InsideOut Development Progress

> Slim tracker — rewritten 2026-10-04. Authoritative task board:
> [`docs/plans/README.md`](plans/README.md); running system:
> [`docs/architecture/`](architecture/index.md).

## Current baseline

Shipped and running:

- Go API over PostgreSQL with row-level security; embedded migrations + seed;
  the `insideout` CLI and a 50-tool MCP server (`insideout-mcp`) projecting
  the same `/api/v1` contract.
- Flutter Cobalt web client (the Cobalt restyle merged 2026-09-18), with the
  roadmap canvas and live presence.
- Invitation-only signup (`signup_mode = invitation_only` on a fresh
  database) with email auth — password, 6-digit code, and magic link; the
  first admin is bootstrapped out-of-band with `insideout promote-admin`.
- GitHub evidence sync: the `insideout.yaml` guide plus webhook deliveries
  attach commit/PR/deploy evidence to roadmap leaf nodes.
- Roadmap canvas with presence; PRD sections, revisions, and audience views;
  the agent vocabulary (`context`, `focus`, `checkpoint/report`, `propose`,
  `version`) across API, CLI, and MCP.

Where things live now:

- **Active work, next actions, and blockers:** the task board in
  [`docs/plans/README.md`](plans/README.md).
- **Known limitations and structural debt:**
  [`docs/issues/README.md`](issues/README.md) (with the nearest dated record
  for each).

## Superseded

The former phase-by-phase delta tracker (P1–P8, last updated 2026-08-13)
described the build-out of the since-deleted Nuxt frontend and is superseded
by the baseline above plus the plans board. Its record survives in git
history and in the changelogs and issue records it referenced
([`docs/changelogs/`](changelogs/README.md),
[`docs/issues/`](issues/README.md)) — including the shipped-database,
auth, coach, roadmap-canvas, and CLI/MCP-parity threads now folded into the
baseline.
