# 2026-10-04 — Remediation Wave 2 (server hardening)

Second wave of the [multirole-review remediation](../review/2026-10-04-multirole/40-remediation-plan.md)
(register: [01-findings-register.md](../review/2026-10-04-multirole/01-findings-register.md));
plan of record [docs/plans/2026-10-04-remediation-wave-2.md](../plans/2026-10-04-remediation-wave-2.md).
Waves 0–1 landed in [2026-10-04-remediation-wave-0-1.md](2026-10-04-remediation-wave-0-1.md).
Also on board: the compose stack moves to `postgres:18`.

## What changed

### Auth abuse defenses (SEC-01, SEC-02)

- `server/internal/api/auth_throttle.go` (new, wired in `server.go`):
  in-memory per-IP + per-account throttle. Credential endpoints (login,
  login-code/verify): 8 failures → 15-min lockout keyed on email+IP plus
  per-IP, uniform `429` + `Retry-After`, success resets; send endpoints
  (magic-link, forgot-password, login-code): 3 per 10 min per email+IP with
  an IP backstop; volume endpoints (register, refresh, magic-link/consume,
  reset-password): 30/min per IP. Unlisted routes (including the GitHub
  webhook) pass through untouched, and locked responses are identical for
  known and unknown emails (no enumeration oracle). Closes
  [2026-08-24 — auth endpoints have no rate limiting](../issues/2026-08-24-auth-endpoints-no-rate-limiting.md).
- `server/internal/api/auth_cooldown.go` (new, wired into
  `auth_email_flows.go`): 60 s resend cooldown per email AND per IP on all
  three send endpoints; on cooldown the same uniform 200 returns but nothing
  is sent, and blocked attempts do not extend the window.

### Mail log hygiene + config validation (REL-02, ARCH-04)

- `mail.NewSender` gained a `devBody` flag: dev mode logs recipient+subject
  only unless `INSIDEOUT_MAIL_DEV_BODY=1`, so live magic links no longer
  land in logs by default. `scripts/dev.sh` exports
  `INSIDEOUT_MAIL_DEV_BODY=1` locally to keep the
  copy-the-link-from-the-logs workflow.
- `config.Load` fails fast when `INSIDEOUT_MAIL_API_KEY` is set without an
  explicit `INSIDEOUT_PUBLIC_URL` (emailed links must never silently point
  at the localhost default).

### Agent runtime (AIDEV-01, REL-06, REL-08)

- `retryStreamChat` takes a `canRetry` guard: a turn that already streamed a
  delta to the client is never retried (a retry would duplicate output);
  nil-callback callers (critic, roadmap planner) still retry.
- `INSIDEOUT_AI_MAX_CONCURRENT` (default 4) and `INSIDEOUT_AI_QUEUE_WAIT`
  (default 15s) are now env-tunable; unset/invalid values fall back to the
  previous hardcoded defaults, and the 503 `retry_after_seconds` hint
  reflects the configured wait.
- SSE event writes return errors end-to-end: a dead client aborts the turn
  (run marked interrupted, partial content persisted) instead of wedging
  the goroutine.

### Containers + compose (FS-07/CRIT-03, ARCH-05)

- `server/Dockerfile`: runs as `nonroot:nonroot`; `HEALTHCHECK` via the new
  `insideout health` subcommand (`server/cmd/insideout/cli_health.go` —
  distroless ships no shell/curl; the probe dispatches before
  `config.Load`, so it needs no `DATABASE_URL`).
- `client/Dockerfile`: runtime switched to
  `nginxinc/nginx-unprivileged:1.27-alpine` (uid 101, port 8080) + wget
  healthcheck. `client/nginx.conf` is now an envsubst template: the `/api/`
  upstream is `$NGINX_API_UPSTREAM` (Dockerfile bakes the Railway default
  `http://server.railway.internal:8080`; compose overrides it to
  `http://server:8080`) — the hardcoded internal hostname previously made
  the client image unbootable outside Railway's DNS.
- `docker-compose.yml`: new `client` service (`${CLIENT_PORT:-5173}`),
  healthchecks on all three services, and the server env block now forwards
  the full operator catalog — mail (`INSIDEOUT_MAIL_API_KEY`,
  `INSIDEOUT_MAIL_FROM`, `INSIDEOUT_PUBLIC_URL`, `INSIDEOUT_MAIL_DEV_BODY`),
  GitHub (`INSIDEOUT_GH_*`, `GITHUB_TOKEN`), browser knobs (CORS, cookie,
  TTLs), and the AI limits.

### postgres 18

- The bundled `postgres` service is now `postgres:18` (was 17). The data
  mount moved to `/var/lib/postgresql` because the 18 image keeps `PGDATA`
  in a versioned subdir. **A 17 volume is not in-place compatible**: wipe
  it (`docker compose down -v`) for dev data, or `pg_upgrade` for data you
  keep. Docs updated (`usage/deployment.md`, `usage/environment.md`,
  `usage/local-development.md`, `architecture/deployment.md`,
  `docs/PLAN.md`).

## Verification

- `go build ./... && go vet ./... && go test ./...` — all packages `ok`
  (DB-gated tests skip without `DATABASE_URL` by convention).
- New tests: throttle (9 — lockout trip/expiry, per-email+IP keying, XFF
  rightmost, webhook passthrough), cooldown (unit + handler + DB-gated
  end-to-end), mail dev-body, config PUBLIC_URL/MailDevBody, retry guard
  (no-retry-after-delta / retry-when-clean, `-race`), AI concurrency env,
  SSE write-error propagation.
- `scripts`: `python3 -m unittest test_env_catalog test_env_writes` 47/47;
  `bash -n dev.sh`; `./scripts/env.sh check server` 5 ok / 0 fail.
- `docker compose config -q` exit 0.
- **Live stack on postgres 18.6** (scratch env, throwaway volume): all
  three containers `(healthy)`; 28 migrations applied; in-container
  `/insideout health` OK; `localhost:8080/healthz` 200;
  `localhost:5173/healthz` 200; `localhost:5173/api/v1/me` → 401 through
  the nginx proxy. Torn down with `down -v`.

## Operator notes

- **Local dev**: dev-mode mail logs no longer contain magic links unless
  `INSIDEOUT_MAIL_DEV_BODY=1` (`scripts/dev.sh` sets it for you). If your
  `.env` sets `INSIDEOUT_MAIL_API_KEY`, `INSIDEOUT_PUBLIC_URL` must now be
  explicit or the server refuses to boot.
- **Deployed hosts**: auth endpoints can now answer `429` + `Retry-After`;
  response shapes the Flutter client depends on are unchanged. Rebuilt
  images run unprivileged — server as uid 65532, client as uid 101 on
  container port 8080 (Railway `PORT=8080` already matches).
- **Self-hosted compose**: see the postgres-18 note above; `CLIENT_PORT`
  (default 5173) publishes the web client.
