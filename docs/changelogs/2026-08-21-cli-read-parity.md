# 2026-08-21 — CLI read-surface parity (Stage 1 of one-truth-many-views)

Plan: [cli-mcp-parity](../plans/2026-08-21-cli-mcp-parity.md).
PRODUCT.md principle 3 requires Web, GitHub, CLI, MCP, and Agents to
project the same facts; until now only the web client existed. This is
the first aligned increment.

## What changed

- `server/internal/apiclient` — shared HTTP client over the same
  `/api/v1` contract the Flutter client uses (bearer auth). Resource
  reads return the API's raw JSON: projections cannot drift.
- `insideout login|whoami|workspaces|projects|prd` — product CLI
  subcommands on the existing binary, dispatched before server
  config/DB so no `.env` is needed. `login` prompts on stderr and
  prints only the token; other commands read `INSIDEOUT_TOKEN` and
  `INSIDEOUT_API` (default: hosted API).

## Verification

- Unit tests (httptest) for login parsing, bearer header, raw-JSON
  reads, and the unauthorized hint; full `go test ./...` green;
  `go vet` and `gofmt` clean.
- Live against the hosted API (2026-08-20): registered scratch user
  `cli-parity-check@example.com` and a labeled workspace (pre-launch
  instance; delete when convenient), then exercised every verb —
  `whoami` and `workspaces`/`projects` with real data, `prd` reaching
  a clean 404 `{"error":"PRD not found"}` on an unknown id.

## What is deliberately not here

- MCP server (Stage 2): tools must be 1:1 with these verbs — same
  names, arguments, output. The tool list is frozen in the plan.
- Write verbs and the agent vocabulary (`context`, `focus`,
  `checkpoint/report`, `propose`, `version`) land as API routes first,
  then project to CLI/MCP (Stage 3).
