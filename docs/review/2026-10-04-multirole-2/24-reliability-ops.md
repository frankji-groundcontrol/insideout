# 24 · Reliability & Ops

**Lens:** 
**Score:** **7.5 / 10**
**Bottom line:** Waves 0-4 genuinely closed the packaging rot and the Go runtime core remains strongest — what still breaks in operation is everything that assumes exactly one long-lived replica or a well-behaved client: boot-time migrations have no advisory lock, the per-IP throttle/cooldown keys on the shared Railway edge hop, presence SSE writes still discard errors with no write deadline, every restart with a live stream burns the full 30s shutdown window, and the nginx DNS ritual is still a human runbook with no dated issue.

Reviewed main @ 26db979; a skeptic re-checked every High finding from this review. REL-13 was confirmed High by the skeptic: the rightmost XFF entry on the documented Railway topology is the edge IP for every user, so per-IP keys collapse into one shared bucket (fail-closed sitewide lockout). The canary curl through the domain is still unrun — the topology is inferred from docs.

---

This review was persisted findings-only; the section below is assembled verbatim from its structured findings (the prose wrapper was not retained). 11 findings as filed, worst first.

## Findings

### REL-13 · High ·  **Skeptic: CONFIRMED.**

**Per-IP throttle and cooldown keys collapse to the shared proxy IP on the Railway path (edge + nginx = two appending proxies), turning the IP backstop into a sitewide shared bucket.**

- Where: `server/internal/api/auth_throttle.go:292`
- Evidence: auth_throttle.go:294-295 rightmost-XFF split justified by 'the rightmost entry is the address nginx saw'; client/nginx.conf:25 proxy_add_x_forwarded_for appends the edge hop on Railway.
- Fix: first_check: observe the XFF chain the server actually receives through the Railway domain (canary curl). Then pass the edge-supplied XFF through at nginx / prefer X-Real-IP, or make trusted-proxy depth configurable; add a two-hop test.
- Diff size: ~15 lines + test

### REL-14 · Medium · 

**Migrate has no advisory lock; two replicas booting concurrently race on non-idempotent DDL and the loser exits(1).**

- Where: `server/internal/store/migrate.go:15`
- Evidence: migrate.go:60 applied-list read then per-file tx + PK insert at :95 — no pg_advisory lock; precedent exists at store/roadmap_plan.go:48.
- Fix: pg_advisory_xact_lock on a fixed key at the top of Migrate, re-reading the applied list inside the lock.
- Diff size: ~10 lines

### REL-15 · Medium · 

**Presence SSE writes discard errors and nothing bounds a blocked write — the REL-08 fix covered only the coach stream; even the coach path blocks (rather than errors) on a black-holed client.**

- Where: `server/internal/api/presence_routes.go:101`
- Evidence: presence_routes.go:103/129/135 Fprintf + Flush with returns ignored; no SetWriteDeadline anywhere; main.go:151 no WriteTimeout by design.
- Fix: Check write errors and return; the real fix for both streams is a per-write deadline: type-assert SetWriteDeadline and stamp now+30s before each event.
- Diff size: ~20 lines

### REL-16 · Medium · 

**Shutdown still burns the full 30s whenever any SSE client is connected (REL-09 open), and a killed in-flight coach turn loses its terminal state.**

- Where: `server/cmd/insideout/main.go:167`
- Evidence: main.go:170 30s timeout then httpServer.Shutdown; no RegisterOnShutdown, no server-lifetime context; presence loop exits only on r.Context().Done().
- Fix: Cancel a server-lifetime context in http.Server.RegisterOnShutdown and select on it in both stream loops; converts the coach turn into its normal disconnect path.
- Diff size: ~10 lines

### REL-17 · Medium · 

**Build/expand LLM endpoints still bypass rate-limit, circuit breaker, and dispatcher permits (REL-04 open, unfixed since review 1).**

- Where: `server/internal/api/roadmap_ai.go:73`
- Evidence: roadmap_ai.go:73 'tree, err := s.planner.PlanMVP(r.Context(), prd.Title, prd.Sections)' — same at :130; no gates, while HandleMessage runs all three.
- Fix: Extract the gate from HandleMessage into a shared helper used by both handlers; keep the 429/503 JSON contract.
- Diff size: ~25 lines

### REL-18 · Medium · 

**FS-06 half-fixed: envsubst made the upstream portable, but the literal proxy_pass still resolves once at startup — and the promised dated issue does not exist in docs/issues/.**

- Where: `client/nginx.conf:22`
- Evidence: nginx.conf:22 'proxy_pass ${NGINX_API_UPSTREAM}/api/;' with no resolver directive; grep of docs/issues/ for nginx/DNS/upstream finds nothing.
- Fix: resolver from /etc/resolv.conf + variable proxy_pass (set $upstream …) so nginx re-resolves per request; file the dated issue.
- Diff size: ~8 lines

### REL-19 · Medium · 

**Resend-cooldown maps grow without bound — the throttle got a 10k-entry sweep, the cooldown got nothing.**

- Where: `server/internal/api/auth_cooldown.go:32`
- Evidence: auth_cooldown.go:32-33 emails/ips maps; allow() at :48-61 only inserts; contrast auth_throttle.go:110 throttleMaxEntries = 10_000.
- Fix: Mirror the throttle's bounded-map sweep (cap + drop entries older than the 60s window).
- Diff size: ~12 lines

### REL-20 · Low · 

**pgxpool still entirely on library defaults (REL-11 open); MaxConns is max(4, NumCPU) by accident, and compose sets no resource limits on any service.**

- Where: `server/internal/store/pool.go:46`
- Evidence: pool.go:47 ParseConfig with only the pgbouncer protocol flip; no MaxConns/MaxConnLifetime anywhere.
- Fix: Explicit MaxConns (INSIDEOUT_DB_MAX_CONNS, default 8-10) + MaxConnLifetime; add basic compose mem limits.
- Diff size: ~10 lines

### REL-21 · Low · 

**The single-replica assumption lives only in code comments; no operator-facing doc says scale-to-2 breaks throttle, cooldown, conversation locks, permits, and presence.**

- Where: `docs/architecture/deployment.md:1`
- Evidence: presence.go:3-4 and dispatch.go:15-17 in-code notes; deployment.md (72 lines) never states a replica constraint.
- Fix: One paragraph in deployment.md: 'run exactly one server replica; what breaks at two'.
- Diff size: ~10 lines docs

### REL-22 · Low · 

**CI runs the concurrency-heavy suite without -race; the suite contains dispatcher/throttle/presence/SSE primitives and already produced one race fixed by hand at 26db979.**

- Where: `.github/workflows/ci.yml:59`
- Evidence: ci.yml:61 'go test ./... -count=1 -timeout 20m'; commit 26db979 'fix(ci): SSE test race…'.
- Fix: Add -race to the server job (or a separate scoped race job).
- Diff size: 1 line

### REL-23 · Low · 

**REL-10 residual: promote-admin still logs the personal email at Info; seed still logs the demo password (dev constant).**

- Where: `server/cmd/insideout/main.go:90`
- Evidence: main.go:90 log.Info("promote-admin: done", "email", email); seed.go:68 logs email+password.
- Fix: Drop the email attribute; guard seed logging with a dev-only comment.
- Diff size: 1-2 lines

