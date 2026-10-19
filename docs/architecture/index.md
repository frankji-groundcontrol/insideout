# Architecture

Current-state system architecture for InsideOut. These documents describe
what is actually running today; for the decision history and rationale behind
the design (the Go rewrite, the RLS defense-in-depth addition, the `juanleme`
data cutover, the langchaingo removal), see
[`docs/plans/2026-07-20-go-rewrite/`](../plans/2026-07-20-go-rewrite/README.md)
and the bug records it links to.

## Documents

- [Backend](backend.md) — Go service layout, auth, store layer, API surface.
- [Database and RLS](database-and-rls.md) — schema ownership model,
  migrations, JWT+RLS defense-in-depth, known Postgres gotchas.
- [PRD Coach agent](prd-coach-agent.md) — the four-stage coaching agent,
  tool calling, SSE streaming, the direct Anthropic client.
- [Frontend](frontend.md) — Flutter client (`client/`) on Railway, wearing
  Cobalt; the Nuxt tree was deleted 2026-08-18.
- [Product subsystems](product-subsystems.md) — the 2026-08 product arc:
  version control (Commit + readiness + audience views), the agent
  vocabulary (with proposal application), the time-first roadmap,
  real-time presence/cursors, and the GitHub evidence loop.
- [CLI, MCP and client configuration](cli-mcp-and-client-config.md) — the
  non-web clients: `clientconfig.Resolve` (which server, which identity), the
  saved 0600 session and its renewal, the MCP process and its agent profile,
  install and release, and the single-replica contract.
- [Agent token scope](agent-token-scope.md) — the signed `scope` claim, the
  route policy that refuses an agent token on human-only routes, the MCP
  profile and token exchange, and the fencing of user-written text.
- [Deployment](deployment.md) — docker-compose topology, environment
  configuration, the two supported database-provisioning models; the
  hosted Railway instance is documented in
  [usage/deployment.md](../usage/deployment.md#railway-current-public-deploy).

## System overview

```
Browser
  │  same-origin /api/v1 (Bearer; cookies still accepted)
  ▼
Flutter web (nginx) ── /api/ proxy ──▶ Go API server
                                         │
                                         ├── PostgreSQL (insideout schema, RLS)
                                         └── LLM {base}/messages or {base}/responses
```

The frontend never talks to Postgres or the LLM provider directly — the Go
server is the sole backend. Hosted Flutter web keeps `/api/v1` same-origin
through nginx on the `app` service. Native Flutter talks to the public
`server` URL with Bearer tokens.
