# 2026-08-22 — Agent vocabulary v1 (context, checkpoint, propose)

Plan: [agent-vocabulary](../plans/2026-08-22-agent-vocabulary.md).
PRODUCT.md's minimum agent-facing vocabulary is now real: `context`,
`focus` (v1: a scoping parameter), `checkpoint/report`, `propose`, and
`version` (served by the human Commit system — surfaced in context,
never writable by agents).

## What changed

- Migration `20260822090000`: `project_updates.kind` now admits
  `agent_checkpoint` and `agent_proposal` alongside
  progress/blocker/note.
- `internal/agentcontext`: pure, mode-shaped assembly — brainstorming
  (product argument + open questions), implementation (focus node with
  siblings and evidence, leaf frontier), review (version baseline +
  PRD core) — each response carries the vocabulary contract stating
  what agents may and may not do.
- Routes: `GET /api/v1/agent/context?project_id&mode&focus`,
  `POST /api/v1/agent/checkpoint`, `POST /api/v1/agent/propose`
  (kind ∈ structure|scope|priority). Writes are typed timeline records
  with `accepted: false` semantics — a human decides.
- CLI `agent-context` / `checkpoint` / `propose`; MCP tools
  `agent_context` / `checkpoint` / `propose` (21 tools).

## Verification

- Unit tests for the assembly (all modes, focus scoping, vocabulary
  presence); full server suite, vet, gofmt green.
- Migration applied by Railway boot-migrate as owner; deployed through
  the domain with the app-proxy restart.
- Live: focus-scoped implementation context returned the focus node
  (title, evidence count, 2 siblings), a 12-leaf frontier, and the
  vocabulary contract; `checkpoint` and `propose` wrote
  `[agent checkpoint] …` and `[agent proposal/scope] …` timeline
  records with their kinds. Scratch user purged.

## Deliberately not in v1

- Persisted per-session Focus pointers (needs agent sessions).
- Proposal acceptance workflow (accept/reject into structure changes).
- Coach weaving gap explanations into conversation.
