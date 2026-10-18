# 24 · Reliability and ops

**Lens:** graceful shutdown and in-flight SSE, timeouts on network and DB calls, unbounded memory/goroutine growth, migration-on-boot and locking, pool sizing and statement timeouts, singleton guard and restart behaviour, health/readiness truthfulness, log content, compose and Dockerfile deployment settings, retention reaper  
**Score:** **7 / 10**  
**Bottom line:** The service shuts down gracefully and recovers from a killed database, but it has no cap on SSE streams per user and no deadline on the request or health path. One user can push the container past its 512m limit, and a stalled database leaves /healthz hanging. No Critical or High defect was found at runtime.

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

---

## Scope

Reliability-ops review of InsideOut at main @ 9f0b1c7 (clean worktree). I built the real binary and drove it against a private PostgreSQL cluster (port 55450) with a fake LLM on localhost: health and authenticated endpoints, SIGTERM during open SSE streams, SIGSTOP/kill of Postgres mid-request, a 1000-stream SSE load, a rolling-redeploy simulation against the singleton guard, a held migrate lock, and a 400k-row retention purge. Code-only items are labelled as such. No network beyond localhost was used.

## Scores

| Aspect | Score | Why |
|---|---|---|
| Graceful shutdown | 8.5 | SIGTERM during open SSE exited cleanly inside the drain; streams closed first, then base context cancelled. |
| DB failure recovery | 8.0 | Killing Postgres mid-request returned errors and the pool recovered on restart with no process restart. |
| Timeouts on outbound calls | 7.5 | LLM, mail, GitHub and apiclient all have bounds; the 20 s apiclient total is questionable for planner calls. |
| Timeouts on inbound/DB path | 6.0 | `statement_timeout` is set, but there is no request deadline and `/healthz` has none. |
| Unbounded growth | 5.0 | SSE streams per user are uncapped; memory is O(n) per stream and broadcast cost is O(n^2). |
| Migration and deploy safety | 6.5 | Advisory lock is correct, but waits silently; singleton guard misfires on rolling deploys. |
| Observability and logs | 7.5 | Structured logs, no tokens; connect errors expose DB host and user. |
| Overall | 7.0 | No Critical or High defect found at runtime; several Medium limits missing. |

## What is good

- Shutdown is designed in layers: `RegisterOnShutdown` closes streams then cancels the base context after a drain (`server/cmd/insideout/main.go:180-181`), with a 30 s `Shutdown` cap (`main.go:197`). Verified live.
- Server timeouts are set deliberately and `WriteTimeout` is omitted on purpose for SSE, with a per-write re-armed deadline (`main.go:166-172`, `httpx.ArmWriteDeadline`).
- Pool and statement bounds exist: `appPoolMaxConns` 10, `migratePoolMaxConns` 3, `appStatementTimeout` 15 s (`server/internal/store/pool.go`). A frozen DB makes statements fail rather than pile up forever.
- Migrations serialise under `pg_advisory_xact_lock` (`server/internal/store/migrate_role.go:66`).
- The singleton guard hijacks its connection so pool Close cannot hang shutdown (`server/internal/api/single_replica.go:58-67`).
- Container health is a real probe with a 3 s bound (`server/cmd/insideout/cli_health.go`), and the Dockerfile is distroless nonroot.

## Findings

**REL-01 (Medium) Unbounded SSE streams per user.** `presence.New(30*time.Second, nil)` and `progresshub.New()` (`server/internal/api/server.go:64-65`) cap channels but not subscribers. Presence key is user id plus a caller-chosen `?client=` (`presence_routes.go:95`). Measured: 1092 streams gave 428 MB RSS and 13 s CPU; 1060 streams gave 657 MB RSS, against `mem_limit: 512m` (`docker-compose.yml:47`). One authenticated user can OOM the container; every join/leave broadcasts a full snapshot to all subscribers.

**REL-02 (Medium) No request or health deadline under DB stall.** `if err := s.store.Pool.Ping(r.Context()); err != nil {` (`healthz.go:10`) has no timeout. With Postgres SIGSTOPped, `/healthz` hung until the client gave up and requests held connections for the full 15 s statement limit. The container probe still reports unhealthy via its 3 s client timeout, but a load balancer probe would hang.

**REL-03 (Medium) Presence stream outlives membership and token.** Membership is checked once (`presence_routes.go:91`); the heartbeat case only calls `s.presence.Touch(pid.String(), key, userID.String(), name)` (`:154`). A removed member kept receiving the project snapshot until disconnect. Confirmed live.

**REL-04 (Medium) Singleton guard fights rolling deploys.** A single `pg_try_advisory_lock` at construction (`single_replica.go:69`), never retried. During overlap the new instance logs the singleton ERROR and runs without the guard; afterwards, a genuine second replica is undetected. The docs tell operators to alert on that ERROR, so every deploy pages.

**REL-05 (Medium) Compose has no `stop_grace_period`.** The design needs 20 s drain (`main.go:121`) plus persistence; Docker's default stop timeout is 10 s, so `docker compose stop` SIGKILLs mid-drain. The server block (`docker-compose.yml:44-47`) sets `restart` and `mem_limit` only.

**REL-06 (Medium, code-only) apiclient total timeout.** `hc:   &http.Client{Timeout: 20 * time.Second},` (`server/internal/apiclient/apiclient.go:27`) bounds the whole exchange; planner calls forced through a tool can exceed 20 s on a slow provider.

**REL-07 (Low) Mail send is synchronous and a timing oracle.** Unknown users return at once (`auth_email_flows.go:41-44`); known users wait on `return s.mail.Send(r.Context(), to, subject, text, html)` (`:302`, up to the 10 s client). The comment promises uniform behaviour, but latency leaks account existence.

**REL-08 (Low) Unbatched, unindexed retention purge.** `DELETE FROM insideout.ai_runs WHERE created_at < ...` (`server/internal/store/agent_messages.go:261`) has no index on `created_at` alone. 400k rows took 6.7 s against a 15 s statement limit; a backlog beyond ~900k rows would hit the limit every 5 minutes and never finish.

**REL-09 (Low) Silent migrate-lock wait.** `SELECT pg_advisory_xact_lock(hashtext('insideout:migrate'))` (`migrate_role.go:66`) has no timeout and logs nothing while waiting; the server is not listening, so the platform sees a hung boot.

**REL-10 (Low) pgx connect errors include host and user.** Boot failures print the DB host and role into logs; redact before shipping logs to third parties.

**REL-11 (Low) Shutdown-cancelled coach stream ends with no terminal event.** `errors.Is(loopErr, context.Canceled)` (`server/internal/agent/coach_fail.go:66`) emits nothing to the client, so the UI cannot tell redeploy from success.

## Lead verification

| Lead | Verdict | Evidence |
|---|---|---|
| L1 TestProjectQuietFlagged time bomb | Confirmed, not filed; no new siblings | Searched the test tree for fixed-date constants. |
| L2 Graceful shutdown with SSE | Confirmed good | SIGTERM with open streams exited inside the drain. |
| L3 Pool and statement timeouts | Extended | Timeout exists server-side; no request deadline (REL-02). |
| L4 Singleton guard | Extended | Misfires on rolling deploy, blind afterwards (REL-04). |
| L9-L12 Hubs, reaper, mail | Extended | REL-01, REL-03, REL-07, REL-08. |
| L16 Dockerfile and image job | Confirmed consistent | HEALTHCHECK is `["/insideout", "health"]`. |

## Recommendations

1. Cap concurrent SSE streams per user and per project, sized from measured RSS under 512m.
2. Recheck membership on each presence heartbeat and close the stream on removal.
3. Wrap `/healthz` Ping in a 2-3 s context and add a per-request deadline middleware for non-stream routes.
4. Set `stop_grace_period: 35s` on the server service.
5. Make the singleton guard retry for a bounded window, and downgrade the overlap log so deploys do not page.
6. Send mail asynchronously with a bounded queue so response latency is uniform.
7. Batch the retention purge with a LIMIT loop and add an index on `ai_runs(created_at)`.
8. Add a lock-wait log line and a `lock_timeout` for migrations.

## What this role would do first

Before anything else I would put a cap on streams per user and a deadline on the request and health path, because those are the two ways a single account or a slow database takes the whole single-replica service down. Then the compose grace period, which is one line. The two limits to set before the next release: a per-user/per-project cap on concurrent SSE streams sized under the 512m memory limit, and a per-request deadline (including a 2-3 s `/healthz` Ping) paired with `stop_grace_period: 35s`.
