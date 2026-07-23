# Summary — What Changed

Sources: [rewrite plan](../../plans/2026-07-20-go-rewrite/README.md),
[TODO.md](../../TODO.md), and the bug book entries linked inline. All facts
below are recorded in those documents; the code is authoritative where they
disagree.

## Backend: Supabase edge functions → Go

The two TypeScript/Deno edge functions (`juanleme-ai-generate`,
`juanleme-export-document`) were replaced by a single Go 1.25 service
(`server/`, module `github.com/frankji-groundcontrol/insideout/server`):
stdlib `net/http` (no framework), `pgx/v5` (no ORM), `golang-jwt/v5`,
`x/crypto/argon2`, `slog`. SQL migrations live in `server/db/migrations/`
and are embedded and applied by the server's own runner
(`go run ./cmd/insideout migrate`). Endpoints cover auth
(register/login/refresh/logout), workspaces + invite-code join, projects +
update timelines, idea inbox, PRD sections/revisions/review lifecycle,
on-demand export (markdown/print, no object storage), and the SSE coach
stream. See plan decision D5 and phases P2–P3 in
[TODO.md](../../TODO.md).

## Auth: email+password, owned by the Go server

argon2id hashes, short-lived JWT access token + rotating opaque refresh
token, both httpOnly cookies (plan D3). Migrated legacy accounts keep their
Supabase Auth bcrypt hashes via a compatibility path in
`server/internal/auth/password.go`, silently upgraded to argon2id on first
successful login.

## Database: JWT+RLS defense-in-depth

Everything lives in the `insideout` schema, owned by a single
`insideout_app` role; nothing touches `public` (plan D1/D2,
[BUG-008](../../issues/2026-07-20-bug-008-shared-instance-db-provisioning.md)). On top
of the primary Go app-layer authorization checks, every transaction sets
`app.user_id` from the JWT-validated caller (`withUserContext` in
`server/internal/store/pool.go`) and 11 business tables carry RLS policies
matching the authorization checklist. `workspace_memberships` has policies
defined but not forced — self-referential policies plus row-locking made
full enforcement impractical under a single-role model
([BUG-007](../../issues/2026-07-20-bug-007-rls-against-real-postgres.md)); Go's own
checks fully cover it. `sessions`, `ai_run_events`, and
`ai_circuit_breaker` have no RLS by design. Two provisioning models share
the same migration files: a dedicated instance (bundled docker-compose
`postgres:17`, `insideout_app` owns the database) or a shared multi-tenant
instance (`insideout_app` owns only the `insideout` schema).

## juanleme → insideout real-data cutover

The real data — **5 users, 4 workspaces, 7 memberships, 1 project** — was
migrated into `insideout` and verified under RLS by querying as a real
migrated user. The four retired tables (`workshop_nodes`, `documents`,
`document_revisions`, `export_jobs`) and operational telemetry (`ai_runs`,
`ai_circuit_breaker`, `ai_run_events`) were archived as JSON outside the
repo, then `DROP SCHEMA juanleme CASCADE` was run. Details: plan Q1 and
[TODO.md](../../TODO.md) P1.

## AI: langchaingo removed, direct Anthropic client

langchaingo was dropped entirely from `go.mod` (unmaintained upstream, plus
a real bug where a leading extended-thinking content block silently ate the
answer). Replaced by a direct Anthropic Messages API client
(`server/internal/agent/anthropic.go`, stdlib `net/http` + hand-rolled SSE
parser) behind the same `ChatStreamer` interface. Env vars renamed
accordingly: `AI_BASE_URL` → `ANTHROPIC_BASE_URL`, `AI_AUTH_TOKEN` →
`ANTHROPIC_AUTH_TOKEN` (`AI_MODEL` unchanged, default
`claude-sonnet-4-20250514`; see `server/internal/config/config.go`). See
[BUG-009](../../issues/2026-07-21-bug-009-langchaingo-removed.md). The coach keeps the
four-stage state machine (clarify/draft/critique/finalize), the
`get_prd`/`update_prd_section`/`advance_stage` tools, and the SQL rate
limiter (10/min, 60/hr) + circuit breaker; a logging-middleware bug that
swallowed SSE flushes was fixed
([BUG-010](../../issues/2026-07-21-bug-010-sse-flusher-swallowed-by-logging-middleware.md)).

## Frontend: service-layer swap to the Go API

The Nuxt 4 app now talks only to the Go API through a Nitro same-origin
proxy (`/api/v1/**` → Go). The mock and supabase service adapters were
deleted, along with the retired workshop-domain pages/components/stores;
`@supabase/supabase-js` and all `@tiptap/*` deps were removed from
`app/package.json`. Real cookie auth with SSR-aware middleware replaced the
half-fake localStorage token. A real SSR hydration bug in `BaseInput` was
fixed ([BUG-002](../../issues/2026-07-20-bug-002-baseinput-ssr-hydration.md)).

## Ink & Seal token wiring

`app/src/assets/tokens.css` (previously written but never loaded) was wired
into the build; a `style.css` hardcoded `system-ui` override was fixed; all
components were codemodded off raw indigo/gray utilities onto semantic
tokens (verified zero remaining); new `BaseCard`/`BaseBadge`/
`PrdStatusBadge` components; the HelloWorld homepage was replaced with a
real landing page; brand hygiene (favicon, titles, nav/footer copy) moved to
InsideOut ([TODO.md](../../TODO.md) P6).

## Repo cleanup

The entire `supabase/` directory (edge functions, old migrations, tests) was
deleted. docker-compose now ships `postgres:17` + server + app images, both
Dockerfiles verified by `docker compose build`
([BUG-006](../../issues/2026-07-20-bug-006-pnpm-ignored-build-scripts.md),
[BUG-004](../../issues/2026-07-20-bug-004-compose-nested-interpolation.md)). Docs
refreshed; bug book entries BUG-001 through BUG-010 recorded.
