# 2026-08-21 — CLI / MCP parity with Web (one truth, many views)

Status: **in flight**. Implements PRODUCT.md principle 3 — Web, GitHub,
CLI, MCP, and Agents project the same facts and never drift — starting
from the current reality: the Flutter web client owns the whole product
surface, the CLI has only operator commands (`migrate`, `seed`), and no
MCP server exists.

## Architecture (alignment by construction)

One contract, three projections:

- The REST API (`/api/v1`, bearer or cookie auth) stays the only source
  of truth. No client gets a private server path.
- Every product verb exists as a CLI subcommand on the existing
  `insideout` binary and as an MCP tool with the same name, arguments,
  and output shape. A verb is "aligned" only when all three surfaces
  agree; the plan's checklists track per-verb alignment.
- Later stages add the agent-facing vocabulary (`context`, `focus`,
  `checkpoint/report`, `propose`, `version`) as API routes first, then
  project it to CLI/MCP like any other verb.

Auth: `POST /auth/login` returns `accessToken`; clients send
`Authorization: Bearer`. CLI v1 reads the token from `INSIDEOUT_TOKEN`
and the API base from `INSIDEOUT_API` (default: hosted API), so no
secrets are written to disk by tooling.

## Stage 1 — read surface (this increment)

- [x] `server/internal/apiclient`: minimal client — login, whoami,
      list workspaces, list projects, get PRD; reads return the API's
      raw JSON so the CLI cannot drift from the truth
- [x] CLI subcommands: `insideout login`, `whoami`, `workspaces`,
      `projects`, `prd <id>` — dispatched before server config/DB so
      no `.env` is required
- [x] Unit tests for the client (httptest, no mocks of the API contract)
- [x] Live verification against the hosted API (2026-08-20): scratch
      user `cli-parity-check@example.com` + labeled workspace; login
      (token, prompt on stderr), whoami, workspaces, projects all
      return real data; `prd` verified for auth+routing via a clean 404
      on an unknown id (its data path needs a real PRD — Stage 3)
- [x] Plan/board/changelog updated; checkpoint pushed

## Stage 2 — MCP server (shipped 2026-08-21)

- [x] Go MCP server `server/cmd/insideout-mcp` (stdio, mcp-go) exposing
      tools 1:1 with the CLI verbs; token via `INSIDEOUT_TOKEN`
- [x] Frozen tool list (14): `whoami, workspaces, projects, prd,
      roadmap_list, roadmap_add, roadmap_update, roadmap_move,
      roadmap_delete, build, expand, guide, repo_set, sync`
      (`login` stays CLI-only — the token is env state)
- [x] Live verification against the hosted API (guide tool + full
      register→build→guide chain;
      [changelog](../changelogs/2026-08-21-guide-and-mcp.md))

## Stage 3 — write surface + agent vocabulary

- [x] Idea write verbs on CLI/MCP (2026-08-22): `idea create`,
      `idea convert` + `proposal_decide` (23 tools)
- [ ] PRD-revision write verbs on CLI/MCP (remaining)
- [ ] Agent vocabulary routes (`context`, `focus`, `checkpoint/report`,
      `propose`, `version`) designed against PRODUCT.md, then projected

## Sources

- PRODUCT.md principle 3 and "Git, CLI, MCP, and Agents"
- Route inventory: `server/internal/api/` (auth, ideas, prds, projects,
  roadmap, conversations, workspaces)
