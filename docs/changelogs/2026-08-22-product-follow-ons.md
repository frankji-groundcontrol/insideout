# 2026-08-22 — Product follow-ons: coach gaps, proposal acceptance, idea verbs, canvas v1

The five follow-ons from the week's product arc, tracked in
[agent-vocabulary](../plans/2026-08-22-agent-vocabulary.md) follow-ons,
[cli-mcp-parity](../plans/2026-08-21-cli-mcp-parity.md) Stage 3, and
[restore-ink-seal](../plans/2026-08-19-restore-ink-seal.md).

## Coach weaves gap explanations (principle 7)

- `systemPrompt` now injects a per-audience gap block
  (`formatGapsForPrompt`, from `internal/readiness`): priorities,
  sections, reader-facing reasons; `validate_later` marked
  non-blocking.
- The clarify-stage discipline requires every question to state its
  priority, who the answer serves, and why now — and forbids blocking
  「现在成版」.
- Verified: prompt unit tests (blank PRD carries must/should gaps and
  the form-now allowance; complete PRD shows no blocking gaps) and a
  live conversation where the coach answered the form-now question
  with the carried-unknowns framing.

## Proposal acceptance workflow

- Migration `20260822100000`: `proposal_decisions` (immutable decision
  log; latest state per proposal) under FORCE RLS; migration
  `20260822110000` added the UPDATE policy the re-decide upsert needs
  (caught live: deciding twice returned 500 without it).
- `POST /api/v1/agent/proposals/{uid}/decision` (owner/admin; accept or
  reject with reason); each decision appends a visible timeline note —
  history in the timeline, latest state in the table.
- CLI `insideout idea proposal-decide --accept|--reject`; MCP
  `proposal_decide`. Verified live: propose → accept → reverse to
  reject.

## Idea write verbs (parity Stage 3)

- CLI `insideout idea create --title [--content] <ws>` and
  `idea convert <id>`; MCP `idea_create` / `idea_convert` (23 tools).
  Verified live: created (status inbox) and converted (prd +
  conversation) through the CLI.

## Collaborative canvas v1 (Ink & Seal)

- `roadmap_canvas.dart`: sibling bands — each root branch a horizontal
  column of its subtree, same editing affordances as the list (shared
  node-tile builder) — plus a status-tinted minimap that scrolls to a
  band; list ↔ canvas SegmentedButton toggle (`AppScaffold` gained an
  `actions` slot).
- Verified by a widget test (toggle, bands, child placement, minimap
  tap); deployed to the hosted app. Real-time multi-user presence
  stays a follow-on (v1 is the visual canvas the plan named).

## Android — verified blocked

- `flutter doctor`: "Unable to locate Android SDK". The release build
  stays blocked on an SDK existing; nothing to ship until then.

## Housekeeping

- Full suites green (server + client 45/45), vet, gofmt; both Railway
  services deployed; verification scratch users purged.
