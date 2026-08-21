# 2026-08-21 — Guidance system scaffolding + MCP server

Users can now bootstrap the repo-side matching guide (the "guidance
system") themselves from every surface, and the MCP projection of the
product exists. Plans:
[cli-mcp-parity](../plans/2026-08-21-cli-mcp-parity.md),
[roadmap-parity-and-github](../plans/2026-08-21-roadmap-parity-and-github.md).

## insideout.yaml matching guide (format v1)

Generated from a project's roadmap — node-keyed, leaves carry editable
matchers, branch nodes are documented as comments:

```yaml
version: 1
nodes:
  <node-uuid>:
    title: "交付 MVP"
    branches: [] # exact name or prefix ending in /*
    labels:   [] # exact PR label
    paths:    [] # prefix of any touched file
```

An event attaches evidence to every matched **leaf**; unmatched events
stay visible but attach to nothing (PRODUCT.md). The full rules ship in
the generated file's header.

## Surfaces (1:1 by construction)

- **API**: `GET /api/v1/projects/{pid}/guide` → `text/yaml`
  (`curl -H "Authorization: Bearer …" …/guide > insideout.yaml`).
- **CLI**: `insideout guide [--out insideout.yaml] <project-id>`.
- **MCP**: `server/cmd/insideout-mcp` (mark3labs/mcp-go, stdio) with 14
  tools 1:1 with the CLI verbs — `whoami, workspaces, projects, prd,
  roadmap_list/add/update/move/delete, build, expand, guide, repo_set,
  sync`. No login tool: the token is env state (`INSIDEOUT_TOKEN`,
  `INSIDEOUT_API` defaults to the hosted API).

## Verification

- Unit: guide generator (leaves vs branch comments, escaping, empty
  roadmap); full `go test ./...`, `go vet`, `gofmt` green.
- MCP stdio smoke: initialize handshake + `tools/list` → 14 tools.
- Live against the hosted API: scratch user → idea → convert → agent
  `build` → `guide` through the **MCP tool** and the **CLI**
  (`--out /tmp/insideout.yaml`), including a double-quoting bug in the
  header caught in the first live run, fixed, and re-verified.
- Server deployed (`railway up --service server` ×2), `app` restarted
  per the nginx DNS gotcha; `/api/v1/me` → 401 via the domain.

## Still open

- **Loader**: the webhook does not yet parse `insideout.yaml` to attach
  leaf-node evidence (needs the installation-token flow to read private
  repos; public repos could load unauthenticated).
- Install the GitHub App on the repo (user) for real deliveries.
