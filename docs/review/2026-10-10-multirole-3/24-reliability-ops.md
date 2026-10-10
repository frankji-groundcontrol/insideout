# 24 · Reliability & Ops

**Score:** **8 / 10** · 7 findings (1 High+)

**Bottom line:** The operational core is genuinely strong — a real shutdown drain with bounded detached persists, per-write SSE deadlines, a 90s upstream idle watchdog, DB-backed rate limit and breaker, a reaper with a fresh partial index, a healthz that pings the pool, timeouts on every outbound HTTP client, and honest single-replica documentation. What remains before real users: one cross-platform latency defect (the native Flutter coach stream is capped at 30s by dio's inter-chunk receiveTimeout while the server tolerates 90s and web 75s), two unbounded resources (no statement_timeout on the app pool, no retention for ai_runs/ai_run_events/sessions), a manual-only backup script, and a mail funnel that has never run against the real provider.

---

## Reliability & Ops review (review 3, tip 754c502)

Scope read this session: `server/cmd/insideout/main.go` (drain design), `server/internal/store/pool.go`, the SSE stack (`httpx/sse.go`, `agent/sse.go`, `presence_routes.go`, `roadmap_progress.go`), the agent gates (`gate.go`, `dispatch.go`, `coach.go`, `coach_fail.go`, `taxonomy.go`, `anthropic.go`), reaper + cooldown/throttle maps, `docker-compose.yml`, the deployment and backup docs, and the client's stream/retry layer (`client_core.dart`, `client_stream.dart`, `sse.dart`, `roadmap_page.dart`). Checks I executed: `go test ./cmd/insideout/ -run 'Shutdown|ShutdownCoach' -count=1` → **ok** (the drain tests pass at this tip); greps for `statement_timeout` (zero hits in server/docs/migrations), backup automation (zero hits in workflows/scripts), retention issues (none in docs/issues). Coordinator leads 1-13 were taken as given; leads 14-15 are answered below. I could not run a live coach exercise (no key, per lead 13) or a load test of the compose memory caps.

### What is genuinely good

- **Shutdown is designed, not accidental.** `main.go:161-181` arms ReadHeader/Read/Idle timeouts, refuses WriteTimeout on SSE grounds, cancels the base context 20s into the 30s Shutdown window so in-flight coach turns take their `context.Canceled` path, and `RegisterOnShutdown` closes passive streams first — the comment block at `main.go:116-120` even does the arithmetic (20s drain + 10s persist = 30s window).
- **Terminal-state discipline.** Every coach turn either flips `succeeded` or the defer at `coach.go:160-177` persists partial text and marks the run failed — on a `WithoutCancel` context still bounded by `persistTimeout` (10s, `coach.go:49`), so nothing hangs forever and nothing is lost silently.
- **Every write to a client is deadline-armed.** `httpx/sse.go:13` re-arms a 30s write deadline per event; the provider stream has a 90s idle watchdog (`taxonomy.go:24`) with a named error and honest SSE error to the still-connected user (`coach_fail.go:59-65`).
- **The shared limits live in Postgres.** Rate limit and breaker are SQL with `pg_advisory_xact_lock` serialization (`20260720135755_ai_ops.sql:37`), so they survive restarts and are correct across instances; the in-process pieces (conversation lock, permits) are honestly labeled (`dispatch.go:16-17`).
- **The reaper got real engineering.** Partial index for the stale scan (`20261010100000_ai_runs_reaper_index.sql`), heartbeats distinguishing long-but-healthy from abandoned runs, and a comment explaining why reaping does not affect rate limiting.
- **All in-memory maps are bounded** (throttle/cooldown sweep at 10k entries; presence TTL-prunes; progress hub drops slow consumers non-blockingly), and **request logs carry no PII** (`middleware.go:79-88` logs method/path/status/duration/user_id only; `promote-admin` deliberately declines to echo the email).
- **Every outbound HTTP client has a timeout** — LLM transports (dial 10s / TLS 10s / response-header 30s + idle watchdog), mail 10s, GitHub 15s, CLI 20s, health self-probe 3s — and `/healthz` actually pings the pool (`healthz.go:10`), so the Docker healthcheck tells the truth during a DB outage.
- **The single-replica contract is documented with per-surface precision** (`usage/deployment.md:117-147`), including pool math (replicas × 13 ≤ pooler per-role limit).

### Findings

| ID | Sev | Where | Finding | Fix |
|----|-----|-------|---------|-----|
| REL-01 | High | `client/lib/api/client_core.dart:25` | Native coach SSE capped at 30s of silence by dio's inter-chunk `receiveTimeout: Duration(seconds: 30)` — while the server tolerates 90s (`taxonomy.go:24`) and web budgets 75s (`sse.dart:132`). Verified in dio 5.11.1 source: `response_stream_handler.dart:60-70` closes the stream on a 30s chunk gap. The coach stream sends no heartbeats (only presence/progress do), and the auto-tighten retry (`coach.go:237`) can run a whole second provider call with zero bytes sent to the client. | `receiveTimeout: Duration.zero` on the coach POST's per-request Options (dio treats ≤0 as off; the 75s `readCoachSse` idle timer then governs), and/or heartbeat comments on the coach stream. |
| REL-02 | Med | `server/internal/store/pool.go:60` | No per-query bound: no `statement_timeout` anywhere (grep: zero hits), no request-context deadline (BaseContext is deadline-free). A hung query or `FOR UPDATE` wait on the single breaker row (`ai_ops.sql:81`) pins handler + connection past the 30s shutdown window; the drain cannot complete. | Set `statement_timeout` (15s — above the 10s persistTimeout) via `RuntimeParams` in `poolConfig` or the DSN; pgx passes unknown DSN params through, so the next deploy needs no code change. |
| REL-03 | Med | `server/internal/store/agent_messages.go:225` | Unbounded growth: the reaper only flips ai_runs to `failed`; rows (with raw `prompt`/`response` text, `agent.sql:43-44`), ai_run_events, sessions (`sessions.go:24`), auth_tokens and invitations are retained forever. No retention issue recorded. | Extend the reaper ticker with retention DELETEs (ai_runs older than N days, cascade to events; expired sessions/tokens/invitations); record the window. |
| REL-04 | Med | `docs/usage/backup-restore.md:12` | Recovery = a manual script nothing schedules (grep of workflows/scripts: zero) + a Supabase PITR toggle the doc itself says the repo cannot verify. The runbook is excellent — walked twice — but the walk is remembered, not scheduled. | Nightly cron/CI dump with retention + an automatic dated drill line; record the PITR confirmation once checked. |
| REL-05 | Low | `docs/usage/deployment.md:117` | One-replica contract enforced by prose only; a second replica "do[es] not fail loudly" (line 120) — double throttle budgets, two turns per conversation, split presence. | Boot-time `pg_try_advisory_lock` singleton canary: log ERROR and mark /healthz if a second holder appears. |
| REL-06 | Low | `server/internal/mail/mail.go:43` | Lead 15: invitation-only signup means the mail path is the entire real-user funnel, and it has never delivered a real link (no key set; tests use httptest). | Pre-deploy live check in the deployment guide: send + click one real magic link and one reset; record the date like the backup walkthrough. |
| REL-07 | Low | `PRODUCT.md:348` | Lead 14 (spot-check, not a full audit): the "implemented / released / validated" distinction has no schema backing — roadmap statuses stop at `done` (`20260722120000_roadmap_nodes.sql:16-17`). PRODUCT.md honestly self-declares this (`PRODUCT.md:425`). | When built, model release/validation as evidence-linked milestones, not new free-form statuses. |

### PostgreSQL blips, in-flight turns, and the drain

On a DB blip: `/healthz` fails fast (pool ping), requests 500 cleanly, in-flight coach turns' terminal writes run on the 10s detached context and fail benignly — the run stays `running` only until the next reaper tick (≤5+10min) marks it failed. That backstop is the right shape. The gap is the unbounded wait *before* the blip resolves: without a statement timeout, a wedged query (not a refused connection) holds its handler forever (REL-02). Shutdown coverage: passive streams end at once, coach turns get 20s + their own 10s persist budget, the reaper exits on the signal context. The one tight edge: a persist started at T+20 finishes at T+30, exactly when `Shutdown`'s window expires and `defer st.Close()` closes the pools — a turn persisting at that instant loses the write and is later reaped. Acceptable, but a 15s statement timeout would make the whole chain deterministic.

### The two limits to set before the next deploy

1. **`statement_timeout = 15s` on the `insideout_app` pool** — today zero; a DSN parameter on the next Railway `DATABASE_URL` needs no code change and converts every hung-query scenario from "handler pinned past the drain" into a 500 the client already knows how to surface.
2. **A retention limit for `ai_runs`/`ai_run_events` (e.g. 90 days) enforced by the reaper** — today infinite; every user message, with raw prompt text, is kept forever, and the rate limiter counts over that same table. Both limits are cheapest to set now, before the first real users make the data worth protecting and the tables worth sizing.

## What this role would do first

Land the one-line per-request receiveTimeout override on the native coach POST (REL-01), then set statement_timeout=15s on the app pool's DSN (REL-02) — both before the pending deploy.
