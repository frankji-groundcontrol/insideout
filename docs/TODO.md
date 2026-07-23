# InsideOut Development Progress

> Live progress tracker. Full technical plan:
> [`docs/plans/2026-07-20-go-rewrite/`](plans/2026-07-20-go-rewrite/README.md);
> current system shape: [`docs/architecture/`](architecture/index.md).
>
> **Last updated**: 2026-07-21

## P1 — Database

- [x] SQL migrations (schema, all domain tables, rate limiter + circuit
      breaker functions).
- [x] Go embedded migration runner (`go run ./cmd/insideout migrate`).
- [x] Go embedded seed command (`go run ./cmd/insideout seed`).
- [x] Ran all 13 migration files against the real target (a shared
      multi-tenant instance; `insideout_app` owns only the `insideout`
      schema, never touches `public`) and verified grants: can create/DML in
      `insideout`, cannot create in `public`.
- [x] JWT + RLS defense-in-depth (added per explicit user direction): every
      transaction sets `app.user_id` from the JWT-validated caller; 11
      business tables carry RLS policies matching the authorization
      checklist — a DB-level backstop on top of the Go app-layer checks.
- [x] `juanleme` real-data cutover (executed per explicit user direction):
      migrated 5 real users, 4 workspaces, 7 memberships, 1 project into
      `insideout`; the four retired tables and operational telemetry were
      archived outside the repo, then the whole `juanleme` schema was
      dropped. Added legacy bcrypt verification + silent upgrade to argon2id
      on login (`server/internal/auth/password.go`). Verified data
      visibility under RLS by querying as a real migrated user.

## P2 — Go backend core

- [x] Server skeleton (config, store pool, slog logging, graceful shutdown).
- [x] argon2id password hashing + JWT access tokens + rotating refresh
      tokens (httpOnly cookies).
- [x] register/login/refresh/logout endpoints.
- [x] Workspace CRUD + invite-code join (collision-retried) + member
      management.
- [x] Unit tests (password hashing incl. legacy bcrypt, JWT, refresh
      tokens, invite-code format).
- [x] Integration tests against real PostgreSQL
      (`server/internal/store/authz_test.go`, full authorization checklist
      including deny paths) all pass.
- [x] Real HTTP end-to-end verification: register → login (httpOnly
      cookies) → create workspace → join by invite code, against the real Go
      server + real PostgreSQL.

## P3 — Domain model

- [x] Projects + updates timeline endpoints (board view).
- [x] Idea inbox CRUD + convert-to-PRD (atomic PRD + conversation creation).
- [x] PRD sections CRUD + revision snapshots + status lifecycle
      (draft → reviewing → approved/rejected → resubmit).
- [x] On-demand export endpoint (markdown/print, no object storage).
- [x] Integration tests (as above) all pass against the real database,
      including the PRD review lifecycle (author submits, admin
      approves/rejects, no self-review).

## P4 — PRD Coach agent

- [x] `ChatStreamer` interface + a direct Anthropic Messages API client
      (`server/internal/agent/anthropic.go`, stdlib `net/http` + hand-rolled
      SSE parsing, zero framework dependency — langchaingo removed entirely,
      see [BUG-009](issues/2026-07-21-bug-009-langchaingo-removed.md)).
- [x] Four-stage state machine (clarify/draft/critique/finalize) + tools
      (get_prd / update_prd_section / advance_stage).
- [x] SSE streaming endpoint + rate-limit/circuit-breaker port (fixed the
      old pending-leak bug on provider 429; also fixed logging middleware
      swallowing SSE flush, see
      [BUG-010](issues/2026-07-21-bug-010-sse-flusher-swallowed-by-logging-middleware.md)).
- [x] Offline template-reply fallback (when `ANTHROPIC_AUTH_TOKEN` unset).
- [x] Unit tests (stage transitions, system prompts, Anthropic stream
      parsing — using real captured SSE payloads as fixtures).
- [x] Agent storage path (conversations/messages/rate limiter) verified
      against the real database via the seed command.
- [x] Real-API smoke test: with the user-supplied `ANTHROPIC_AUTH_TOKEN`, a
      full real idea→PRD coaching exchange verified end-to-end (streamed
      reply, message history persisted and retrievable).

## P5 — Frontend API swap + real auth

- [x] `api` service layer (workspace/project/idea/prd/coach/export
      adapters).
- [x] Nitro same-origin proxy (`/api/v1/**` → Go server).
- [x] Real cookie auth + SSR-aware middleware.
- [x] New IA: dashboard, project board, idea inbox, project detail, PRD
      workspace (SSE coach chat), export page.
- [x] Deleted mock + supabase service layers and the retired
      workshop-domain pages/components/stores (user decision: no mocks, real
      APIs only).
- [x] Fixed a real SSR hydration bug in BaseInput (`Math.random()` id →
      `useId()`).
- [x] typecheck + build + all frontend tests pass.

## P6 — Prettify: wire Ink & Seal

- [x] Wired `tokens.css` into the build (written earlier but never loaded).
- [x] Fixed `style.css` hardcoding system-ui over the design-system font.
- [x] Codemodded all components off raw indigo/gray to semantic tokens
      (verified zero remaining).
- [x] New BaseCard / BaseBadge / PrdStatusBadge components.
- [x] Replaced the HelloWorld homepage with a real landing page.
- [x] Brand hygiene: real favicon, title, nav/footer copy.
- [x] Browser-verified light + dark (screenshots, no hydration warnings).

## P7 — Ship

- [x] docker-compose.yml (postgres:17 + server + app) + both Dockerfiles,
      verified with `docker compose build` (incl. fixing
      [BUG-004](issues/2026-07-20-bug-004-compose-nested-interpolation.md)
      and [BUG-006](issues/2026-07-20-bug-006-pnpm-ignored-build-scripts.md)).
- [x] Deleted the retired `supabase/` TypeScript backend, old migrations and
      tests, and the retired frontend adapters.
- [x] Docs refreshed; bug records BUG-001 through BUG-010 live in
      [`docs/issues/`](issues/README.md).
- [x] README.md rewritten for end users.
- [x] All tests pass against the real `DATABASE_URL`: migrations, grants,
      full authorization-checklist integration tests, real HTTP end-to-end
      flow, seed command, real-AI coaching smoke test.
- [x] Docs reorganized to the clean-repo-org layout (2026-07-21): thin
      `CLAUDE.md`/`AGENTS.md` routers, `docs/architecture|usage|changelogs|`
      `issues|learning|practices|history`, docs-recording guardrail — see
      [the reorg plan](plans/2026-07-21-clean-repo-org.md).

## P8 — Idea → Reality (roadmap, MVP, GitHub, Prisma re-theme)

- [x] Branched-tree roadmap: `roadmap_nodes` schema + RLS, store (CRUD, move
      with cycle guard, cascade delete), API, recursive tree UI with
      trunk-and-stub connectors + status seals; project page Roadmap/Progress
      tabs.
- [x] GitHub sync: `projects.repo_url` + `POST /projects/{pid}/sync-github`
      pulling real public commits into the timeline (cursor dedupe);
      `GithubSync.vue`.
- [x] Build the MVP: `RoadmapPlanner` (forced-tool JSON via the Anthropic
      client + template fallback); `POST /prds/{id}/build` + `POST
      /roadmap/{nid}/expand`; "Build the MVP" button + per-node AI expand.
- [x] Prisma cinematic re-theme: token layer → black/cream (dark default) +
      warm-paper light, Almarai + Instrument Serif fonts, noise textures,
      motion-v landing reveals (`docs/design-system/CHANGELOG.md` 0.3.0).
- [x] All verified real: roadmap integration test + live HTTP, GitHub live
      sync, real-Anthropic build/expand, real-UI build flow, light+dark
      browser checks. See
      [docs/changelogs/2026-07-22-idea-to-reality/](changelogs/2026-07-22-idea-to-reality/index.md).

## Known limitations

- Avatar upload is still a local-preview placeholder (no real upload yet).
- Theme/locale still live in localStorage, not cookies (possible SSR
  first-paint flash).
- PRD section editors are plain textareas — no rich-text editor.
- Self-hosted PuHuiTi fonts are full TTFs (~32 MB for 4 weights) — convert to
  WOFF2 (+ CJK subsetting) to shrink; see
  [2026-07-23 frontend VC changelog](changelogs/2026-07-23-frontend-version-control.md).
- GitHub sync is owner/admin + unauthenticated public API (no private repos);
  roadmap tree has no drag-to-reorder in the UI yet (API `move` exists).
- Open structural question: tracked coding-tool scratch directories, see
  [docs/issues/2026-07-21-tracked-tool-scratch-dirs.md](issues/2026-07-21-tracked-tool-scratch-dirs.md).
