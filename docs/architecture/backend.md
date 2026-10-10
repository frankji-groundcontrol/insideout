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
`userHolder`, see `context.go`) → `withLogging` → `withRecover` →
`withMaxBody` (caps request bodies at 1 MiB everywhere except the GitHub
webhook route, which gets 8 MiB — a push delivery listing every commit's
files can outgrow 1 MiB) → the
`net/http` mux → route-level `requireAuth` where needed.

Authentication is httpOnly-cookie based: a short-lived JWT access token plus a
rotating opaque refresh token (SHA-256 hash stored at rest, rotated on every
`/auth/refresh` call). The hosted Flutter web client is served by an nginx
that proxies `/api/v1/**` to this server same-origin (see
[deployment](deployment.md)), so browser cookies work without CORS; local
Flutter web dev is cross-origin and uses the `INSIDEOUT_CORS_ORIGINS`
allow-list (`dev.sh` defaults it to the `localhost` token).

Beyond email+password, emailed one-time-credential flows share the same
session issuance: magic-link login (`POST /auth/magic-link` then
`/auth/magic-link/consume`), password reset (`POST /auth/forgot-password`
then `/auth/reset-password`, which also revokes all the user's sessions),
and typed code login (`POST /auth/login-code` then
`/auth/login-code/verify`). Signup policy is settings-driven
(`insideout.settings.signup_mode` ∈ invitation_only/open/closed, seeded
fail-closed): invitation-only registration claims an admin-minted,
email-bound, single-use invitation code (`insideout.invitations`,
SHA-256 hash at rest) before provisioning; `users.system_role` gates the
admin routes (invitations + the mode switch, `requireAdmin`), and the
only path to admin is the `insideout promote-admin <email>` operator
subcommand on the owner pool. Their tokens live in
`insideout.auth_tokens` (SHA-256 hash at rest, single-use, expiring —
magic links 15 min, resets 60 min, sign-in codes 10 min with a 5-miss
attempt budget; issuing a new one supersedes the previous unused token of
that kind). Delivery is `internal/mail` (Resend HTTP API; an empty
`INSIDEOUT_MAIL_API_KEY` selects dev mode, which logs messages instead of
sending them), and request endpoints answer a uniform 200 so account
existence is not enumerable.

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
`INSIDEOUT_ACCESS_TTL`/`INSIDEOUT_REFRESH_TTL`, `INSIDEOUT_LLM_BASE_URL`,
`INSIDEOUT_LLM_API_KEY`, `INSIDEOUT_LLM_MODEL`, `INSIDEOUT_LLM_SCHEMA`,
`INSIDEOUT_COOKIE_SECURE`,
`INSIDEOUT_DEV_CORS` — fail-fast unless `INSIDEOUT_PUBLIC_URL` is a localhost
or loopback origin, because the dev middleware reflects any Origin with
credentials. Missing `INSIDEOUT_LLM_API_KEY` is not an error — the PRD
Coach falls back to an offline template reply (`internal/agent/template.go`),
which doubles as local dev mode. See [usage](../usage/local-development.md)
for the full environment variable reference.
