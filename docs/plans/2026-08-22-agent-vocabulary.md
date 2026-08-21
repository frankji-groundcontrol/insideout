# 2026-08-22 — Agent vocabulary v1 (context, checkpoint, propose)

Status: **in flight**. The last designed piece of
[cli-mcp-parity](2026-08-21-cli-mcp-parity.md) Stage 3: PRODUCT.md's
minimum agent-facing vocabulary — `context`, `focus`,
`checkpoint/report`, `propose`, `version`.

## Semantics (PRODUCT.md)

- Agents receive **compact, Focus-scoped context**, not the entire
  graph (modes: brainstorming / implementation / review).
- Agents can **checkpoint work** and **propose structure, scope, or
  priority changes**; they cannot apply strategic changes, Commit
  versions, or Merge branches.
- `version` is already served by the human Commit
  ([version-commit](2026-08-21-version-commit.md)) — context surfaces
  the latest Commit rather than duplicating it.
- `focus` v1 is a context parameter (node id) scoping the assembly; a
  persisted per-session Focus pointer needs agent sessions and stays
  out of v1.

## Checklist

- [x] Migration: extend `project_updates.kind` CHECK with
      `agent_checkpoint` and `agent_proposal` (constraint name
      confirmed on the live instance)
- [x] `internal/agentcontext`: pure assembly (mode-shaped, focus-scoped)
      with unit tests for all three modes and focus scoping
- [x] Routes: `GET /api/v1/agent/context`, `POST /api/v1/agent/checkpoint`,
      `POST /api/v1/agent/propose` (member auth; typed timeline records)
- [x] CLI `agent-context` / `checkpoint` / `propose`; MCP tools
      `agent_context` / `checkpoint` / `propose` (21 tools)
- [x] Live verification (2026-08-22): focus-scoped implementation
      context (focus node + siblings + 12 leaves + vocabulary
      contract), checkpoint and proposal recorded with their kinds in
      the timeline, `accepted: false` on proposals — humans decide
- [x] Docs closure; checkpoint pushed

## Sources

- PRODUCT.md "Git, CLI, MCP, and Agents", "Collaboration and authority"
