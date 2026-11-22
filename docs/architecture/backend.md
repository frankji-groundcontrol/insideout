# Backend

Go 1.26+, stdlib `net/http` (1.22+ method-pattern routing, no web framework),
[`pgx/v5`](https://github.com/jackc/pgx) (no ORM), `golang-jwt/jwt/v5`,
`golang.org/x/crypto` (argon2id + bcrypt), `log/slog`. Module:
`github.com/frankji-groundcontrol/insideout/server`.

## Layout

```
server/
  cmd/insideout/       main.go (serve/migrate/seed/promote-admin), cli*.go
                       (product CLI + setup TUI, dispatched from cli.go
                       before any server env or DB), seed.go (dev seed data)
  cmd/insideout-mcp/   the stdio MCP server binary — same /api/v1 contract
  db/                  embed.go (go:embed migrations), migrations/*.sql
  internal/
    agent/             PRD Coach: prompts, tools, SSE, the critic, the fact ledger,
                       the roadmap planner
    agentcontext/      the compact, focus-scoped context agent-context briefs from,
                       with an `untrusted` block and the one fence rule (FenceData)
    api/               HTTP routes, middleware, per-domain handlers
    apiclient/         the shared HTTP client every non-web surface speaks: renews
                       the saved session once on 401, trades a full token for an agent token;
                       every call rides a context (`SetContext`) with a per-route deadline
                       (20 s, 5 min for build and expand) and no client-wide timeout
    audienceview/      audience projections of one PRD
    auth/              password hashing, JWT (with the scope claim), refresh-token helpers
    clientconfig/      one Resolve(flagAPI) for CLI and MCP: base from --api, INSIDEOUT_API,
                       the 0600 config, then the hosted default; a saved token (and refresh
                       token) only for the base it was saved for
    config/            environment parsing and fail-fast validation
    export/            on-demand PRD markdown/HTML rendering
    github/            linked-repo commit fetch feeding project progress
    guide/             the repo-side matching guide (insideout.yaml) generator
    httpx/             JSON request/response helpers, the error contract
    linebudget/        the code-budget guard: a non-test Go file or function over the
                       thresholds in config/code-budget.json fails the tests
    llm/               the model provider clients (a leaf: imports nothing internal):
                       ChatStreamer, the Anthropic Messages and Responses streamers,
                       their stream parsers, the retry and error taxonomy
    mail/              transactional email through the Resend HTTP API
    mcpserver/         the MCP surface: one tool per CLI verb, an agent profile that
                       withholds the four password tools, and the result fence (fence.go)
    modtidy/           the module tidiness guard: `go mod tidy -diff` must print nothing
    presence/          in-memory who-is-here registry behind live canvas presence
    progresshub/       in-memory fan-out for the live roadmap progress view
    prdcommit/         the human-only Commit that pins a PRD version
    readiness/         per-audience gap disclosure for a working PRD
    roadmaptime/       the time-first deadline and pressure rules
    setup/             the guided-setup flow engine behind `insideout setup`
    sparkworld/        the Spark demo tree and dependencies, shared by the store and API tests
    store/             all pgx queries; files grouped by domain resource (Store layer below)
    surfaceparity/     the request-field parity guard: every field an API handler decodes
                       is reached by the Go client, the CLI and MCP, or is a listed omission
                       ([practice](../practices/2026-11-21-surface-parity-guards.md))
    testlock/          cross-process advisory locks and the strict-mode host guard shared by the
                       DATABASE_URL-gated tests (imported by tests only)
```

The non-web clients (`cmd/insideout-mcp`, `apiclient`, `clientconfig`, `mcpserver`,
`setup`, the CLI verbs) are described in
[CLI, MCP and client configuration](./cli-mcp-and-client-config.md).

## Request flow

`cmd/insideout/main.go` wires `config.Load()` → `store.Open(DATABASE_URL)` →
`api.NewServer(...).Handler()`. The middleware chain (`internal/api/server.go`,
`middleware.go`) is: `withRequestID` (installs a mutable per-request
`userHolder`, see `context.go`) → `withLogging` → `withRecover` →
`withMaxBody` (caps request bodies at 1 MiB everywhere except the GitHub
webhook route, which gets 8 MiB — a push delivery listing every commit's
files can outgrow 1 MiB) → the
`net/http` mux → route-level `requireAuth` where needed. Inside `withMaxBody`,
`withRequestDeadline` gives every route a 25 s context except five exempt
patterns (the two passive streams, the coach message POST, `build`, and
`expand`). The two confirms run under the 25 s deadline. `/healthz` pings the
database under a 2 s deadline and answers 503 when it cannot (rv4-REL-02).
The two passive streams (presence, roadmap progress) are capped at 20 per user
and 100 per project, with
429 `app_stream_limit` past the cap, cursor posts are limited to 25 a second per
user, and each 10 s heartbeat re-verifies the stream's own access token and the
caller's project membership, so a stream ends when its session or membership
does (rv4-REL-01, REL-03, SEC-08).

Authentication is a short-lived JWT access token plus a rotating opaque
refresh token (SHA-256 hash stored at rest, rotated on every `/auth/refresh`
call). Rotated tokens form a session family (`sessions.family_id`, migration
`20261018170000_sessions_family.sql`: a nullable column and an index, no
backfill; a session from before it reads as a family of its own id): a spent
token offered again after `refreshReuseGrace` (10 s, which covers a second tab
or CLI process that lost a refresh race) ends the whole family, while a replay
inside the grace is only a lost race (rv4-SEC-04). An access token issued before
such a kill still works until `INSIDEOUT_ACCESS_TTL` runs out; `iss`/`aud`
checks and a logout denylist are open. It is Bearer-first: `requireAuth` takes the access token from
`Authorization: Bearer` and only then from the httpOnly access cookie
(`bearerOrCookie` in `cookies.go`). Every first-party client sends the header
(the Flutter client, the CLI, MCP); sign-in still sets the httpOnly cookies and
the server still accepts them, which keeps a plain browser session working.
`/auth/refresh` reads the refresh token from the httpOnly cookie, else a
`{"refreshToken"}` body, and the cookie wins when both are present. The hosted
Flutter web client is served by an nginx that proxies `/api/v1/**` to this
server same-origin (see [deployment](./deployment.md)), so it needs no CORS; local
Flutter web dev is cross-origin and uses the `INSIDEOUT_CORS_ORIGINS`
allow-list (`dev.sh` defaults it to the `localhost` token).

An access token may also carry a `scope` claim. `scope: agent` is minted by
`POST /agent/token` (authenticated, itself human-only; one access-token life,
no cookie, no refresh token) and is what `insideout-mcp` sends instead of the
user's own token. `requireAuth` is the only code that verifies a token: it
looks up the route's mux pattern in `agentRoutePolicy`
(`internal/api/agent_scope.go`) and answers 403 `agent_scope_denied` for an
agent token on every human-only route (the proposal decision, Commit, the
admin routes, the deletes). A pattern missing from the table is human-only,
and a route-table test fails the build for a route with no entry. A token with
no claim is full scope, so nothing issued earlier changes. See
[agent token scope](./agent-token-scope.md).

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
subcommand on the owner pool, which first creates the account (password read
from stdin) when the address has none. Their tokens live in
`insideout.auth_tokens` (SHA-256 hash at rest, single-use, expiring —
magic links 15 min, resets 60 min, sign-in codes 10 min with a 5-miss
attempt budget; issuing a new one supersedes the previous unused token of
that kind). Delivery is `internal/mail` (Resend HTTP API; an empty
`INSIDEOUT_MAIL_API_KEY` selects dev mode, which logs messages instead of
sending them), and request endpoints answer a uniform 200 so account
existence is not enumerable. Sends leave the request path through a small
bounded queue in `internal/mail` (4 workers, 64 waiting; a full queue drops the
send and logs it), so the 200 is uniform in time as well as in body (rv4-SEC-03,
rv4-REL-07). Per-client throttles key on the TCP peer, or on the address
`INSIDEOUT_TRUSTED_PROXY_HOPS` hops from the right of `X-Forwarded-For` when the
server sits behind proxies (rv4-SEC-02; see [deployment](../usage/deployment.md)).
`NGINX_XFF` in the nginx template chooses that chain: the hosted image passes
the edge's header through, and the compose stack ends it in the address nginx
saw, so both topologies are one hop.

## Store layer

`internal/store` wraps a single `*pgxpool.Pool` in a `Store` struct — no code
generator, hand-scanned SQL. Files are grouped by resource family, one file per
concern inside a family:

- accounts: `users.go`, `sessions.go`, `auth_tokens.go` (magic links, resets,
  sign-in codes), `invitations.go`, `reset_watch.go` (the password-reset watch
  the CLI polls; outside RLS, purged by the reaper's retention purge),
  `erasure.go` (the account tombstone behind `DELETE /me`)
- teams: `workspaces.go`, `memberships.go`
- work: `ideas.go`, `project*.go` (projects, updates, the linked repo), `prd*.go`
  (PRDs, revisions, Commit)
- the coach: `agent_*.go` (conversations, messages, the context queries),
  `ai_runs.go` (the run lifecycle, the stale-run reaper, run events),
  `ai_ops.go` (the rate limiter and circuit breaker), `retention.go` (the
  reaper's retention purge and the idle-conversation purge)
- proposals: `proposal_*.go` (the human decision, the structured items)
- roadmap: `roadmap*.go` (nodes, reads, evidence, moves, plans, validation;
  `roadmap_views.go` the node views with roll-ups, `roadmap_assignments.go`
  the owner history and delegation, `roadmap_mine.go` a person's own tree,
  `roadmap_delete.go` the guarded subtree delete)
- GitHub: `github_webhook.go` (the system-run webhook path)
- plumbing: `pool.go`, `errors.go` (the sentinels the api maps to statuses),
  `migrate*.go`, `serve_role.go`

Authorization is enforced twice, deliberately:

1. **Go app layer** — every mutating function re-checks the caller's
   membership/ownership/admin role inside the same transaction as the write
   (the TOCTOU-safe pattern; see [database and RLS](./database-and-rls.md) for
   why explicit row locks were removed from these checks). The predicates
   live in `authz.go` (`requireMember`, `requireAdminMember`,
   `requirePlanWriter`, and the roadmap tree's `requireNodeWriter` and
   `requireNodeManager`, which call the same SQL function as the policies).
2. **PostgreSQL RLS** — a database-level backstop matching the same rules,
   described in [database and RLS](./database-and-rls.md).

## API surface

Domain handlers live in `internal/api`, grouped by resource:

- wiring: `server.go` (routes), `middleware.go`, `context.go`, `cors.go`,
  `cookies.go`, `util.go`, `url_scheme.go`, `healthz.go`, `single_replica.go`
  (the second-process guard, see the client doc below); `client_ip.go` (the
  trusted-proxy client address), `stream_caps.go` and `stream_recheck.go` (the
  SSE per-user/per-project caps and the heartbeat membership recheck),
  `request_deadline.go` (the per-request deadline on non-stream routes)
- accounts: `auth*.go` (sign-in, the emailed one-time flows and their
  templates, the throttle and resend cooldown; `auth_reset_watch.go` answers
  forgot-password with a watch id and serves `GET /auth/reset-watch/{id}`),
  `me.go`, `me_delete.go` (the human-only account erasure), `admin.go`
  (invitations and the signup-mode switch)
- agents: `agent_*.go` (the agent vocabulary routes and `agentRoutePolicy`;
  `agent_root.go` the token mint and the root-scoped token's subtree check)
- teams and work: `workspaces.go`, `members.go`, `projects.go`,
  `project_updates.go` and `project_updates_page.go` (the paged feed,
  `GET /projects/{pid}/updates?before=&limit=&node=&pending=`), `ideas.go`, `guide.go`
- PRDs: `prd*.go` (PRDs, audience views, readiness, Commit; `prd_list.go`
  the workspace's PRD list, `GET /workspaces/{id}/prds`, member-visible and
  agent-allowed; a project view's `prdId` names the PRD it was built from),
  `conversations.go`
  (the coach's SSE turns, the stage confirm and the human-only `stage/decline`
  route, and the conversation view with `criticSkipped` and `criticRounds`),
  `conversations_trust.go` (the parked finding resolutions the view lists as
  `pendingResolutions` on the confirm card)
- roadmap: `roadmap*.go` (nodes, reads and `mine`, drafting, evidence,
  progress, validation; `roadmap_delegation.go` delegate, hand-back and the
  owner history), `presence_routes.go`
- GitHub: `github*.go` (repo sync, the webhook and its evidence)

The single-replica contract and its guard are in
[CLI, MCP and client configuration](./cli-mcp-and-client-config.md#the-single-replica-contract).
The error contract is `{"error": string, "code": string, ...extra}` (see
`internal/httpx/json.go`). Every code is lowercase snake_case; an error with
no specific code carries its status's name (`not_found`, `bad_request`,
`internal_server_error`), and extra keys are camelCase (`retryAfterSeconds`,
`liveCount`). `internal/httpx/error_style_test.go` reads every `WriteError`
and SSE `errorEvent` call and fails on another style (rv5-PARC-08).
Handlers report a store error through `writeStoreError` in
`internal/api/store_error.go`. It maps each store error (`ErrNotFound`,
`ErrForbidden`, `ErrConflict`, `ErrValidation`, the errors that wrap them, and
`ErrRepoTaken`) to one status and code, and the handler supplies only its own
wording. A handler checks a store error itself only where it answers a
different status on purpose (a refusal answered 404, a membership pre-check,
the auth handlers' uniform answers), with a one-line comment saying so; the
[ladders issue](../issues/2026-11-21-store-error-ladders.md) lists them
(rv5-STRS-11).
AI-specific throttling responses keep the
`app_throttle` (429) and `circuit_open` (503) shapes with `retryAfterSeconds`
that the frontend countdown reads; a provider-side rate limit hits after the
SSE stream is already open, so it surfaces as an SSE `error` event with code
`anthropic_rate_limit` rather than an HTTP status.

Response builders take the instant as a parameter instead of reading the wall
clock: `Server` owns a `clock` behind `s.now()` (nil means the wall clock;
only tests set it), handlers pass `s.now()` to `projectResponse` and
`roadmapNodeResponse`, and `view_clock_guard_test.go` fails on any
`time.Now`, `time.Since` or `time.Until` in a non-test file of the package
outside its allowlist. Request field limits count characters, not bytes,
through `tooLong` in `util.go`.

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
