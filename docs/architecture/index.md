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
- [Frontend](frontend.md) — Nuxt 4 SSR app, service layer, design tokens.
- [Deployment](deployment.md) — docker-compose topology, environment
  configuration, the two supported database-provisioning models.

## System overview

```
Browser
  │  same-origin (httpOnly cookies)
  ▼
Nuxt 4 (SSR) ── Nitro routeRules proxy /api/v1/** ──▶ Go API server
                                                          │
                                                          ├── PostgreSQL (insideout schema, RLS)
                                                          └── Anthropic Messages API (direct client)
```

The frontend never talks to Postgres or the LLM provider directly — the Go
server is the sole backend, and the Nitro proxy keeps everything same-origin
so cookie-based auth works for both the browser and SSR without CORS.
