# InsideOut Development Plan

> **Status**: high-level overview for the dev team. The full phased technical
> plan (database, backend, agent, frontend) lives in
> [`docs/plans/2026-07-20-go-rewrite/`](plans/2026-07-20-go-rewrite/README.md);
> the current implemented shape is documented in
> [`docs/architecture/`](architecture/index.md) — treat those as authoritative.

## Technical architecture

| Layer | Technology |
|---|---|
| Database | PostgreSQL, `insideout_owner` (schema + DEFINER, NOSUPERUSER) + `insideout_app` (runtime), `insideout` schema, SQL migrations in `server/db/migrations/` applied as owner; JWT+RLS defense-in-depth |
| Backend | Go 1.25+, stdlib `net/http` (1.22+ method routing), `pgx/v5`, `golang-jwt/v5`, argon2id |
| Agent | Direct Anthropic Messages API client (`server/internal/agent/anthropic.go`), SSE streaming, four-stage state machine (clarify/draft/critique/finalize) |
| Frontend | Flutter 3 Material 3 (`client/`), go_router, Dio, zh-CN default / en-US |
| Deployment | docker-compose (postgres:17 + server), Railway Flutter web + Go, or point `DATABASE_URL` at any PostgreSQL 14+ instance |

## Repository layout

```text
server/                        # Go backend
├── cmd/insideout/             # main.go, migrate/seed subcommands
├── internal/
│   ├── api/                   # HTTP handlers, middleware, route registration
│   ├── auth/                  # argon2id (+ legacy bcrypt), JWT, refresh tokens
│   ├── store/                 # pgx queries, split by domain
│   ├── agent/                 # PRD coach: ChatStreamer, Anthropic client, tools, prompts
│   ├── export/                # Markdown / print HTML rendering
│   └── config/                # env parsing
└── db/migrations/             # SQL migrations (embedded in the binary)

client/                        # Flutter 3 frontend (web + iOS + Android)
└── lib/
    ├── api/                   # Dio client, models, errors
    ├── features/              # landing, auth, dashboard, workspace, project, prd
    ├── session/               # Session, Appearance, auth redirect
    └── l10n/                  # zh-CN + en-US

docs/                          # see docs/index.md for the documentation map
```

## Information architecture

| Route | Purpose |
|---|---|
| `/`, `/login`, `/register` | Landing, login, register |
| `/dashboard` | My workspaces, create/join |
| `/workspace/[id]` | Project board (group-leader view) |
| `/workspace/[id]/ideas` | Idea inbox |
| `/projects/[id]` | Project detail + updates timeline |
| `/prd/[id]` | PRD workspace: section editing + coach chat |
| `/prd/[id]/export` | Export preview (Markdown / print) |
| `/profile` | Profile |

## Conventions

- **Naming**: backend uses snake_case (DB columns/Go internals); the API
  JSON boundary is camelCase throughout; frontend uses camelCase.
- **Auth**: Bearer access + refresh tokens for Flutter; the server still
  sets httpOnly cookies. Hosted web is same-origin `/api/v1` via nginx.
- **Testing**: no mocks, real dependencies; the Go side has integration
  tests against real PostgreSQL (gated on `DATABASE_URL`), the frontend has
  real-logic unit tests; AI-touching tests only run when
  `ANTHROPIC_AUTH_TOKEN` is set.
- **Migrations**: SQL files live in `server/db/migrations/`, applied in
  filename order by the Go server's own embedded runner
  (`go run ./cmd/insideout migrate`) — no external migration tool.

## Milestones

See [`docs/plans/2026-07-20-go-rewrite/README.md`](plans/2026-07-20-go-rewrite/README.md)
§5 (phases P1–P7) and [`docs/TODO.md`](TODO.md) (live progress).
