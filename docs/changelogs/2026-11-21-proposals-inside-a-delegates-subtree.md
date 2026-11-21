# 2026-11-21 — Proposals inside a delegate's subtree

Owner decision 7 ([review 5 open findings](../issues/2026-11-21-review-5-open-findings.md)),
which narrows [BUG-020](../issues/2026-10-06-bug-020-workspace-member-writes-the-plan.md).

## What changed

- `POST /api/v1/agent/propose` takes an optional `nodeId`. Without it, only a
  plan writer (project owner or workspace admin) files, as before. With it,
  the node must be in the project (404 otherwise, and for a node the caller
  cannot see) and the caller must be able to write it (`_can_write_node`,
  403 otherwise); the proposal row stores `node_id`. The checks run in the
  proposal's own transaction (`store.AddProposal`).
- A root-scoped agent token must name a node inside its root; it gets 403
  `agent_scope_denied` outside the root or with no node.
- Deciding a proposal is unchanged: plan writers only, and never an agent
  token.
- Surfaces: Go client `AgentPropose(projectID, nodeID, …)`, CLI
  `propose --node <id>`, MCP `propose` argument `node_id`.
- Docs: [roadmap tree](../architecture/roadmap-tree.md),
  [agent token scope](../architecture/agent-token-scope.md),
  [CLI and MCP usage](../usage/cli-mcp-and-guidance.md), `SKILL.md`.

## Verification

Real-database API tests in `server/internal/api/rv5_proposals_subtree_test.go`
(delegate on their node and a descendant, sibling and parent refused, plain
member refused, plan writer with and without a node, a node of another
project, a root-scoped agent inside and outside its root, the delegate cannot
decide their own proposal); the surface parity test covers `nodeId` on the
client, CLI and MCP.
