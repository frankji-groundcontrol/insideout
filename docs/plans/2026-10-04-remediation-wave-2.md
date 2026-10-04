# 2026-10-04 — Remediation Wave 2 (server hardening)

**Status:** Finished 2026-10-04
**Source:** [`docs/review/2026-10-04-multirole/40-remediation-plan.md`](../review/2026-10-04-multirole/40-remediation-plan.md)
Wave 2. Register IDs refer to
[`01-findings-register.md`](../review/2026-10-04-multirole/01-findings-register.md).
Waves 0–1 landed in `ecfe4d2`. Changelog:
[`docs/changelogs/2026-10-04-remediation-wave-2.md`](../changelogs/2026-10-04-remediation-wave-2.md).

## Checklist (smallest end-to-end first)

- [x] 2.8 REL-02 — mail dev-mode logs to/subject only, drops the body (the
      live sign-in link) unless `INSIDEOUT_MAIL_DEV_BODY=1`; `.env.example`
      "links included" comment synced; `dev.sh` opts local dev in.
- [x] 2.9 ARCH-04 — `config.Load` fails fast when real mail delivery is
      enabled without an explicit `INSIDEOUT_PUBLIC_URL`; config tests.
- [x] 2.5 AIDEV-01 — `retryStreamChat` guarded by `canRetry` (streamed ==
      0), so a mid-stream failure cannot duplicate the answer; nil-callback
      callers still retry; `-race` tests.
- [x] 2.6 REL-06 — `INSIDEOUT_AI_MAX_CONCURRENT` / `INSIDEOUT_AI_QUEUE_WAIT`
      from env with the old hardcoded values as defaults; 503 retry hint
      follows the configured wait; tests.
- [x] 2.7 REL-08 — SSE event writes return errors: a dead client aborts the
      turn instead of wedging the goroutine; tests.
- [x] 2.3 FS-07/CRIT-03 — `server/Dockerfile` `USER nonroot:nonroot` +
      `insideout health` subcommand HEALTHCHECK; `client/Dockerfile` on
      `nginxinc/nginx-unprivileged` + healthcheck; compose healthchecks.
      (`client/nginx.conf` became an `$NGINX_API_UPSTREAM` template so the
      image boots outside Railway DNS.)
- [x] 2.4 ARCH-05 — compose server service forwards the full `.env.example`
      catalog (mail, GitHub webhook/App, CORS, cookie, TTLs, PUBLIC_URL,
      AI limits) with `:-` defaults; new `client` service (`CLIENT_PORT`).
- [x] 2.2 SEC-02 — per-email + per-IP 60 s cooldown on magic-link /
      forgot-password / login-code; uniform 200 preserved; tests incl.
      DB-gated end-to-end.
- [x] 2.1 SEC-01 — per-IP + per-account throttle (8 failures → 15-min
      lockout, uniform 429 + `Retry-After`, send caps, per-IP volume caps;
      webhook route untouched). Closed
      [`docs/issues/2026-08-24-auth-endpoints-no-rate-limiting.md`](../issues/2026-08-24-auth-endpoints-no-rate-limiting.md).

Also on board: bundled `postgres` 17 → 18 (data mount moved to
`/var/lib/postgresql`; 17 volumes need wipe or `pg_upgrade`), with the
usage/architecture docs and `docs/PLAN.md` updated.

## Verification

- [x] `go build ./... && go vet ./... && go test ./...` green
- [x] `docker compose config -q` exits 0
- [x] Dockerfile builds + **live stack on postgres 18.6**: three services
      healthy, 28 migrations applied, `/insideout health` in-container,
      host `:8080/healthz` + `:5173/healthz` 200, `:5173/api/v1/me` → 401
      via the nginx proxy; torn down `down -v`
- [x] changelog `docs/changelogs/2026-10-04-remediation-wave-2.md` + index,
      board updated, plan closed
