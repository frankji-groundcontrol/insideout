# 2026-08-21 — Dogfood project made real; scratch data purged

Follow-ups to the [evidence loop](2026-08-21-evidence-loop.md), executed
the same day.

## Dogfood

- Durable account `dogfood@insideout.yalotein.net` (credentials in the
  local untracked `~/.zcode-tracks/insideout-dogfood-credentials.txt`).
- Real workspace → idea → PRD ("InsideOut 自举") → agent `build` →
  project `d8d6ab41-8048-49ba-ab2e-72b3ed43341f`, repo-bound to
  `frankji-groundcontrol/insideout`.
- `insideout.yaml` regenerated from that project (no more scratch node
  ids) with three matchers: `拆解核心需求` ← branch `main`;
  `明确用户问题` ← paths `server/`; `划定 MVP 边界` ← paths `client/`.

## Scratch purge

- Deleted the five labeled scratch users created during verification
  (`cli-parity-check`, `cli-roadmap-check`, `mcp-guide-check`,
  `guide-final`, `guide-loop` — all `@example.com`) with their
  workspaces, projects, roadmaps, evidence, ideas, PRDs, conversations,
  and sessions — children-first in one admin transaction.
- Left in place: 12 older `@example.com` test accounts that predate
  this work (user's call), and a real-user project `GH+Roadmap Verify`
  (owner `live-…@test.local`) that is also bound to this repo and
  therefore also receives push syncs.
