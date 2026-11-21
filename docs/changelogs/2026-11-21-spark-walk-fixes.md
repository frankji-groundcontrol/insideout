# 2026-11-21 — Spark walk fixes

The [Spark demo data](./2026-11-21-spark-demo-data.md), a walk of the UI as
each role, and a spec-by-spec functional test (237 checks: 232 pass, 2 fail,
3 skipped) found the items below. Each fix had a failing test first.

| Finding | Fix |
| --- | --- |
| A build seeded an arbitrary node in progress. One build inserts every node in one transaction, so `created_at` ties; in the demo the seed landed on a branch deep in another member's subtree. | `seedFirstLeaf` walks the tree depth first by position and takes the first leaf below the root. Test: `TestReplaceRoadmapTree_SeedsFirstLeafInTreeOrder`. |
| Done and validated cards still said "overdue". | `roadmaptime.NodePressure` returns no pressure for finished work. The node view, the Progress items and the agent context all use it. The route tests now expect finished work to carry no pressure. The node shape test reopens its node so every optional field is still set at once. |
| A root-scoped agent token's checkpoint could name a node outside its root (201), while a proposal naming the same node was refused. | The checkpoint runs the same subtree check (`agentRootRefuses`) and answers 403 `agent_scope_denied`. |
| Bands rows looked ragged ([design QA](../design-qa/2026-11-21-roadmap-band-rows.md)). | Each row of the shared band scaffold takes the height of its tallest card. Compact cards drop the unlabelled editor line. |
| `forgot-password` was documented as just emailing a link; it waits up to 60 minutes for the link to be used. | [CLI and MCP usage](../usage/cli-mcp-and-guidance.md) says so. |
| The architecture page said a second server logs an ERROR at once. It logs a WARN and retries for two minutes first. | [CLI, MCP and client config](../architecture/cli-mcp-and-client-config.md) matches [deployment](../usage/deployment.md). |

An impeccable critique of the roadmap views also ran. Its findings are recorded
in the [roadmap critique](../design-qa/2026-11-21-roadmap-critique.md) and wait
on the owner.

## Verification

Run locally; CI is disabled in the repository.

- `go build`, `go vet` and `gofmt`: clean.
- `go test ./...`: passes with no database, and on a freshly migrated private
  PostgreSQL 18 database with no skipped tests.
- `flutter analyze`: 8 infos (baseline 8). `flutter test`: 672 passed.
- Every `scripts/tests/check_*.py` and `scripts/check-redaction.sh`: pass.
