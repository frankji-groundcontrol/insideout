# 2026-10-05 — Simulating real users across CLI, Web, and MCP

**Goal:** exercise the product the way its users would — not by reading
code. **Three personas × three surfaces (CLI, Web, MCP) × at least
three real cases each**, with parity spot-checks (same operation via
two surfaces must agree). Live local server + fresh local database;
every command, click, and tool call captured as evidence.

- **Persona A — everyday builder ("Mira")**: video-editing app, never
  written a PRD, no engineering background. Web-first, but every case
  is mirrored through CLI and MCP (she may ask an agent to do things
  for her).
- **Persona B — professional operator ("Dr. Chen")**: file-management
  app for pharmaceutical delivery workflows. MCP/agent-first (agent
  checkpoints, proposes; human decides), mirrored through CLI and web.
- **Persona C — team lead / stakeholder ("Ravi")**: oversight, not
  authorship. Management-audience views, progress, versions,
  presence — read paths across all three surfaces.

**Why:** the product's core promise is complete clarity at every stage
(day zero → finish line; concrete engineering + high-level thinking;
traceable from milestone to module to code; know where you stand;
update on the fly). Only live use across every surface shows whether
the promise holds; code review cannot.

**Scope:** isolated local Postgres (`insideout_sim` db on the Homebrew
instance — no container runtime on this machine; never the remote
database), server on `127.0.0.1:8091` with the dev mailer (no real
email) and the live LLM coach, both binaries built from `server/`, a
small reusable MCP stdio driver (`scripts/sim/`), Flutter web via
`flutter run` driven by browser automation. Findings → `docs/issues/`;
practice record for the harness; changelog + indexes. No product code
changes (blockers recorded as issues; the one local migration patch
for PG18 is reverted and never committed).

## Checklist

- [x] Plan + board row.
- [x] Sim database + roles + migrations (PG18 RLS finding — see issues).
- [x] Server live on :8091 (dev mailer, live LLM coach).
- [x] Bootstrap: `seed` → `promote-admin` demo → Mira (CLI),
      Chen (MCP `auth_register`), Ravi (web form, 201 → `/dashboard`).
      Note: invites are email-bound — the code only redeems for the
      address it was minted for.
- [x] Server runs with `INSIDEOUT_DEV_CORS=1`; web is a release build
      served on :5174 (debug `flutter run` crashes in its injected
      client). Drivers under `scripts/sim/`: `mcp_session.py` and
      `web_driver.py` (CDP; `launch-ui` opens a visible Chrome with
      DevTools). Web boot hang fixed —
      [BUG-016](../issues/2026-10-06-bug-016-web-boot-gate-hangs-on-font-load.md).
- [x] Persona A — 3 cases (day-zero coach PRD → audience view;
      structure + deadlines; follow-ups: move, commit version,
      readiness), each mirrored CLI/Web/MCP. Move collided a sibling
      position ([BUG-017](../issues/2026-10-06-bug-017-roadmap-move-collides-sibling-position.md)).
- [x] Persona B — 3 cases (agent context/checkpoint/propose → human
      accepts with apply; deadlines + empty evidence; scope cut
      rejected with a reason), each mirrored. Apply reordered the
      items ([BUG-019](../issues/2026-10-06-bug-019-applied-proposal-reorders-items.md)).
- [x] Persona C — 3 cases (join + tour; progress + empty evidence;
      Day zero version and the rejected proposal), each mirrored.
      Joined as `member`. A member can add nodes and file proposals
      ([BUG-020](../issues/2026-10-06-bug-020-workspace-member-writes-the-plan.md));
      the stray node was deleted and the stray proposal rejected by
      the owner. Chen's workspace stayed hidden (404 / blank page).
- [x] Parity spot-check table — in the
      [changelog](../changelogs/2026-10-06-user-simulation-three-personas.md).
- [x] Findings triaged: BUG-017,
      [BUG-018](../issues/2026-10-06-bug-018-roadmap-pressure-label-untranslated.md) (raw pressure key on web),
      BUG-019, BUG-020. All four fixed 2026-10-06
      ([fixes](../changelogs/2026-10-06-simulation-bug-fixes.md)).
      BUG-016 was fixed during bootstrap.
- [x] Practice record and changelog, indexes updated.
- [x] Commit + push + MBP sync — on `main` as `f185e94`; MBP synced
      2026-10-09 ([sync plan](2026-10-09-host-sync/2026-10-09-host-sync.md)).

## Decisions

- Fresh LOCAL database only; DSNs live in `/tmp/insideout-sim/`
  (0600), values never printed. Persona passwords are throwaway
  simulation values, not secrets.
- No real mail: server runs without `INSIDEOUT_MAIL_API_KEY` (dev
  mailer logs links) + `INSIDEOUT_MAIL_DEV_BODY=1`.
- Web automation drives the real Flutter web client; server restarts
  with `INSIDEOUT_DEV_CORS=1` (local sim only) so localhost origins
  pass.
- Evidence format: per-case transcript (command/tool/click → outcome),
  summarized in the changelog with the full matrix table.
