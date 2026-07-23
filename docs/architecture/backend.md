# Backend

Go 1.25+, stdlib `net/http` (1.22+ method-pattern routing, no web framework),
[`pgx/v5`](https://github.com/jackc/pgx) (no ORM), `golang-jwt/jwt/v5`,
`golang.org/x/crypto` (argon2id + bcrypt), `log/slog`. Module:
`github.com/frankji-groundcontrol/insideout/server`.

## Layout

```
server/
  cmd/insideout/       main.go (serve/migrate subcommands), seed.go (dev seed data)
  db/                  embed.go (go:embed migrations), migrations/*.sql
  internal/
    config/            environment parsing and fail-fast validation
    store/             all pgx queries; one file per domain resource
    auth/               password hashing, JWT, refresh-token helpers
    httpx/              JSON request/response helpers, the error contract
    api/                HTTP routes, middleware, per-domain handlers
    agent/              PRD Coach: prompts, tools, SSE, the Anthropic client
    export/             on-demand PRD markdown/HTML rendering
```

## Request flow

`cmd/insideout/main.go` wires `config.Load()` → `store.Open(DATABASE_URL)` →
`api.NewServer(...).Handler()`. The middleware chain (`internal/api/server.go`,
`middleware.go`) is: `withRequestID` (installs a mutable per-request
`userHolder`, see `context.go`) → `withLogging` → `withRecover` → the
`net/http` mux → route-level `requireAuth` where needed.

Authentication is httpOnly-cookie based: a short-lived JWT access token plus a
rotating opaque refresh token (SHA-256 hash stored at rest, rotated on every
`/auth/refresh` call). The Nuxt frontend proxies `/api/v1/**` to this server
same-origin (see [deployment](deployment.md)), so cookies work for both the
browser and Nuxt SSR without CORS.

## Store layer

`internal/store` wraps a single `*pgxpool.Pool` in a `Store` struct — no code
generator, hand-scanned SQL. One file per domain resource: `users.go`,
`sessions.go`, `workspaces.go`, `memberships.go`, `projects.go`,
`project_updates.go`, `ideas.go`, `prds.go`, `prd_revisions.go`,
`agent_conversations.go`, `agent_messages.go`.

Authorization is enforced twice, deliberately:

1. **Go app layer** — every mutating function re-checks the caller's
   membership/ownership/admin role inside the same transaction as the write
   (the TOCTOU-safe pattern; see [database and RLS](database-and-rls.md) for
   why explicit row locks were removed from these checks).
2. **PostgreSQL RLS** — a database-level backstop matching the same rules,
   described in [database and RLS](database-and-rls.md).

## API surface

Domain handlers live in `internal/api/*.go`, one file per resource
(`auth.go`, `me.go`, `workspaces.go`, `members.go`, `projects.go`,
`project_updates.go`, `ideas.go`, `prds.go`, `conversations.go`). The error
contract is `{"error": string, "code"?: string, ...extra}` (see
`internal/httpx/json.go`); AI-specific throttling responses preserve the
`APP_THROTTLE` (429) and `CIRCUIT_OPEN` (503) JSON shapes the original
frontend countdown logic expects; a provider-side rate limit hits after the
SSE stream is already open, so it surfaces as an SSE `error` event with code
`ANTHROPIC_RATE_LIMIT` rather than an HTTP status.

Full endpoint-by-endpoint detail (request/response shapes) lives in
[`docs/plans/2026-07-20-go-rewrite/02-backend-go.md`](../plans/2026-07-20-go-rewrite/02-backend-go.md).

## Export

`internal/export/render.go` renders a PRD's sections to markdown or escaped,
paragraph-wrapped HTML on demand — no object storage, no background jobs.

## Configuration

`internal/config/config.go` parses and fail-fast validates: `INSIDEOUT_ADDR`,
`DATABASE_URL` (required), `INSIDEOUT_JWT_SECRET` (required, ≥32 chars),
`INSIDEOUT_ACCESS_TTL`/`INSIDEOUT_REFRESH_TTL`, `ANTHROPIC_BASE_URL`,
`ANTHROPIC_AUTH_TOKEN`, `AI_MODEL`, `INSIDEOUT_COOKIE_SECURE`,
`INSIDEOUT_DEV_CORS`. Missing `ANTHROPIC_AUTH_TOKEN` is not an error — the PRD
Coach falls back to an offline template reply (`internal/agent/template.go`),
which doubles as local dev mode. See [usage](../usage/local-development.md)
for the full environment variable reference.
