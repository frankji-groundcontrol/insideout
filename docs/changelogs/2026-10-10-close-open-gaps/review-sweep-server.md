# Review-register sweep: the server

The three review registers ([multirole 1](../../review/2026-10-04-multirole/01-findings-register.md),
[multirole 2](../../review/2026-10-04-multirole-2/01-findings-register.md),
[frontend-cobalt](../../review/2026-10-06-frontend-cobalt/01-findings-register.md))
held findings that no issue record or board item traced. Each was re-proved on
the base first; the ones that did not reproduce are marked "not a defect" in
the [disposition ledger](../../review/2026-10-10-disposition-ledger/README.md).
The rest were fixed test-first on real PostgreSQL 18: the test failed on the old
code for the stated reason and was committed red, then the fix turned it green,
and each fix was mutated once to prove the test goes red again. The client half
of the sweep is in [review-sweep-client.md](review-sweep-client.md).

## Row-level security, the store and migrations

- **A removed member kept their own ideas** (`r2/SEC-11`). The author branch of
  `ideas_select`, `ideas_update` and `prds_insert` did not ask for current
  membership, so a user removed from a workspace could still read, edit, drop
  and convert the ideas they had authored there. Migration
  `20261010120000_ideas_require_membership.sql` redefines the three policies to
  require `insideout._is_member(...)` on the author branch, and
  `GetIdeaForMember`, `UpdateIdea`, `DropIdea` and `ConvertIdea` check current
  membership in their own transaction. At the runtime role the row is simply
  not visible (`ErrNotFound`); on the owner pool, which bypasses RLS, the Go
  check answers `ErrForbidden`. `ConvertIdea` for a removed member used to fail
  only by accident, on a raw `42501` from `INSERT … RETURNING`.
- **`ListIdeas` answered a former member** (`r2/ARCH-08`). It and the sibling
  list now call `requireMember` inside the read's own transaction.
- **A policy walk that fails when a table loses RLS** (`r1/SEC-04`,
  `r2/SEC-13`). `TestRLSWalk` reads the row-level-security tables from
  `pg_class`, requires `FORCE` and an entry in its table list, seeds a row in all
  16 tables through the store, and checks that a member sees rows everywhere,
  an outsider sees none, a transaction with no `app.user_id` sees rows only in
  `users` and `ai_runs`, and no policy outside those two has a
  `current_user_id() IS NULL` branch. Seven mutations (a NULL branch, `NO FORCE`,
  an unlisted table, and others) each turned it red. The architecture text now
  says 16 tables and names `ai_runs` as the second NULL-context table.
- **Migrations ran unserialised** (`r2/REL-14`). `Migrate` now holds
  `pg_advisory_xact_lock(hashtext('insideout:migrate'))` on a holder
  transaction, so two replicas starting together queue instead of racing, and
  it refuses to run on a migrate pool with fewer than two connections (the
  holder would deadlock on its own lock).
- **Pool sizes were whatever the CPU count said** (`r1/REL-11`). The app pool is
  explicitly 10 connections and the migrate pool 3; a `pool_max_conns` in the
  DSN still wins. The connection budget is in
  [deployment.md](../../architecture/deployment.md).
- A misleading role-gate name and two comments were corrected (`r2/ARCH-10`).

## HTTP edge

- **`Vary: Origin`** (`r1/SEC-05`, `r2/SEC-15`). Both CORS middlewares now add
  `Vary: Origin` (added, not set, so an outer `Vary: Accept-Encoding` survives)
  on allowed, denied and Origin-less responses.
- **Dev-permissive CORS cannot start on a hosted origin** (`r1/SEC-06`,
  `r2/SEC-16`). `INSIDEOUT_DEV_CORS=1` beside a `INSIDEOUT_PUBLIC_URL` whose host
  is not `localhost`, `127.0.0.1` or `::1` now refuses to start, with an error
  that names `INSIDEOUT_CORS_ORIGINS`. Verified on the built binary.
- **The GitHub webhook accepts what GitHub sends** (`r2/SEC-17`). The global
  1 MiB body cap rejected real push payloads with `400`; the webhook path alone
  now takes up to 8 MiB (exactly 8 MiB passes, one byte more is `400`), and the
  signature is still checked before the body is trusted. The app's nginx needs
  the matching limit; see [the CI and infra record](review-sweep-ci-and-infra.md).
- **The web dev port is pinned** (`r1/AIDEV-06`, `r1/FS-08`). Magic links point
  at `http://localhost:5173`; the three `flutter run -d web-server` commands in
  the usage docs now pass `--web-port=5173`, so the link lands on the app instead
  of a refused connection. The default itself did not change.

## Streams, shutdown and presence

- **Shutdown no longer waits out a connected viewer** (`r1/REL-09`,
  `r2/REL-16`). `Server.CloseStreams` (a `RegisterOnShutdown` hook) ends the
  progress and presence streams and removes their presence entries. A coach turn
  that is mid-flight gets a 20 s drain, then is cut: its `ai_runs` row ends
  `failed` and the message keeps the streamed text plus an `[interrupted]`
  marker. Measured on the real binary: the old code hung the full 30 s shutdown
  window; the new one exits about 21 s after `SIGTERM`.
- **A viewer that stops reading cannot pin a handler** (`r2/REL-15`). Every
  presence, progress and coach write arms a 30 s write deadline first
  (`httpx.ArmWriteDeadline`), re-armed per write, so a long coach turn still
  completes while a stalled reader is dropped (95 s before, 33 s after, in the
  live check). `statusRecorder` now forwards `Unwrap` so the deadline reaches
  the connection through the logging wrapper.
- **Two tabs of one user are two presences** (`r2/SEC-18`, server half). The
  presence key is `userID:clientID` (client id capped at 64 characters; empty
  keeps the bare user id), so one tab leaving no longer removes the other, a
  pointer is not shown under the wrong session, and a stream skips its own
  pointer events. **The wire `sessionId` is now `<userId>:<clientId>`**; the
  client half is in the client record.
- **The promoted admin's address is no longer logged** (`r1/REL-10`,
  `r2/REL-23`): `promote-admin: done` logs `role=admin` only. The demo-login
  line in `seed` stays, because the usage docs rely on it; a comment says it is
  dev-only.
- **`seed` is safe to run twice** (`r1/USER-06`): a second run logs
  `seed: already seeded` instead of creating a second demo workspace.

## Resend cooldown

- The resend-cooldown state moved from a package global onto `Server`
  (`r2/ARCH-09`, `r2/TEST-18`): each server has its own email and IP windows,
  tests inject a fake-clock gate per server, and the `swapCooldown` helper and
  its "no `t.Parallel`" rule are gone (adding `t.Parallel` to the endpoint tests
  had produced `500`s and 11 data-race reports).
- The maps sweep expired keys once either passes 10 000 entries (`r2/REL-19`).
  Known ceiling, as for the throttle: while more than 10 000 keys are live, the
  sweep runs on each allow.

## Coach and agent surface

- **Fact grounding sees the whole conversation** (`r1/AIDEV-02`, `r2/AIDEV-11`).
  `record_fact` checked a quote against the 20-row replay window, so after about
  25 exchanges a verbatim quote of the user's first answer was refused. It now
  checks every user turn (`ListUserMessageTexts`, oldest first, RLS-scoped),
  still one turn at a time, so a quote cannot be stitched from several turns.
- **The coach's own failures no longer open the circuit** (`r2/AIDEV-12`). An
  `ErrLoopInternal` sentinel marks the iteration cap, an unreadable
  conversation and a failed history load; `countsAgainstCircuit` feeds the
  breaker only for rate-limited, transient, stalled and unclassified provider
  errors (not for config, context-length, refusal or cancelled).
- **The provider's error text stays off the wire** (`r1/AIDEV-08`,
  `r2/AIDEV-14`). A `PROVIDER_CONFIG` failure used to send the provider's
  response body to the browser. The stream now carries a fixed sentence, "AI
  provider is misconfigured - contact the operator", with the same code; the
  full text stays in the server log and in `ai_runs.error_message`.
- **Roadmap build and expand are rate-limited and circuit-gated**
  (`r1/REL-04`, `r2/REL-17`). `POST /prds/{id}/build` and
  `POST /roadmap/{nodeId}/expand` now pass the per-user rate limit, the
  circuit breaker and the concurrency permit before the planner runs: `429
  APP_THROTTLE` or `503 CIRCUIT_OPEN` like a coach turn, with `ai_runs` rows
  labelled `roadmap:plan-mvp` and `roadmap:expand` that count toward the
  per-user limit. The gate lives in `agent/gate.go` and the coach's message
  path uses the same code. Before, the 11th build inside a minute still got `200`.
- **Provider error labels** (`r1/AIDEV-05`, `r2/AIDEV-18`). An error from an
  OpenAI-compatible Responses server was labelled `anthropic`; the label now
  names the provider.
- **PRD text is data, not instructions** (`r1/AIDEV-07`, `r2/AIDEV-13`). The
  coach and critic system prompts wrap the title, sections, ledger and findings
  in `<prd_data>` with a clause saying the content is data, and neutralise any
  `<prd_data>` tag inside the text.
- **Tool descriptions say what the tools do** (`r2/AIDEV-10`). The idea tools
  and the `idea delete` output still described a drop as "stays in the list";
  six stale lines (including `SKILL.md`) were corrected.
- **`@` in a proposed title** (`r2/AIDEV-17`). `AgentPropose` took `Title@Parent`
  strings and split on the first `@`, so "Notify @team" under "Launch" became the
  title "Notify " with a bogus parent. It takes `ProposeItem{Title, ParentHint}`
  now; the CLI's `--item Title[@Parent]` form is unchanged.

## Operator notes

- **Apply migrations `20261010120000_ideas_require_membership.sql` and
  `20261010130000_authored_rows_require_membership.sql`** on deploy,
  together with `20261010100000` and `20261010110000` from the earlier waves.
  Both only redefine policies; they are idempotent and safe to re-run, and
  both were applied to a freshly created empty database as well as an
  existing one.
- **CI got slower and stronger on purpose**: the Go job runs the whole suite
  under `-race` (a 20-minute timeout), the client job fails past the
  16-info analyze baseline, and a nightly job runs the five-surface smoke
  against a throwaway database. Plan CI minutes accordingly.
- **Presence ids change shape** (`<userId>:<clientId>`). A new server with an
  old client still works (an empty client id keeps the bare user id); cursors
  line up once the new client is out.
- **Shutdown takes up to about 21 s** when a coach turn is in flight. A platform
  that kills the process sooner than that cuts the drain short; the turn is then
  recovered by the reaper.
- **Compose memory limits are unmeasured.** `docker-compose.yml` now carries
  `mem_limit` 512m for the server and 128m for the client; neither was run
  under Docker here. Treat them as starting values and read `docker stats`
  before relying on them.
- The new roadmap build/expand limits mean a client that hammers the planner
  now sees `429` and `503`; the web client shows its generic error banner.

## CI, security headers and the live smoke lane (i1)

- **CI never ran the race detector** (`r2/TEST-12`, `r2/REL-22`). The Go job
  now runs `go test ./... -race -count=1 -v` with a 20-minute timeout (the
  local race pass needed the headroom), keeping the one allowed skip gate.
- **Analyze could silently shed its baseline** (`r2/TEST-14`). The client job
  counts `info •` lines under `pipefail` and fails past `INFO_BASELINE=16`,
  so an info regression is a red build, not console noise.
- **Postgres and artifacts drifted between CI and reality** (`r2/TEST-17`).
  CI pins `postgres:18.6` (the minor the disposable cluster runs) and uploads
  the Go test log as an artifact on failure.
- **The hosted nginx sent no security headers** (`r2/SEC-12`). `nginx.conf`
  now sets `X-Content-Type-Options`, `Referrer-Policy`, `X-Frame-Options:
  DENY` and a CSP with `always`, and HSTS through a
  `map $http_x_forwarded_proto` so it applies only behind TLS. A new
  `scripts/check_nginx_headers.py` pins the set in the nginx CI job.
- **Records named the dogfood address and the production host**
  (`r2/SEC-14`). Both are redacted to `<dogfood-account>` and `<prod-domain>`
  in the dated records; the webhook comment names the route, not the host.
- **The one-replica rule was prose only** (`r2/REL-21`). A grep check keeps
  `docs/usage/deployment.md` honest about the state that makes the cooldown
  map and turn locks correct.
- **Routes could ship unauthenticated** (`r2/TEST-13`). Six new HTTP-layer
  files: a regexp scan over the package's route registrations that fails when
  a route outside the auth/healthz/webhook allowlist answers a caller with no
  credentials, admin-route tests against real PostgreSQL, and propose/decide
  tests including who may not decide. At cherry-pick time the webhook and
  presence files this cluster shares with s2/s3 were deduplicated — both
  independently wrote byte-identical presence tests, so one copy ships.
- **The five-surface smoke needed a hand-made `.env`** (`r2/TEST-10`).
  `smoke.sh` tolerates a missing `.env`, demands open signup, and drives
  build → confirm → expand → evidence; CI runs it nightly on a throwaway
  database.
- **The web SSE transport had no browser test** (`r2/TEST-11`).
  `client/test/web/sse_transport_web_test.dart` runs under
  `--platform chrome` against a hybrid loopback server: a frame delivered
  while the response is open, method/headers/body echoed, and a 401 that
  throws `ApiException`.

## Round 2: the follow-ups the round-1 verifiers left

- **Sibling RLS policies still trusted authorship** (round-2 follow-up to
  SEC-11). `prds_update`, `project_updates_update`, `project_updates_delete`
  and `prd_revisions_insert` now require current membership on the author
  branch too (migration `20261010130000_authored_rows_require_membership.sql`,
  applied to a fresh empty database as well as the worker's). The store
  writers check membership in-transaction, so the removed author gets
  `ErrForbidden` on the owner pool and silence at the runtime role.
- **The concurrency test that could not fail** (round-2 follow-up). The
  `EnsureProjectForPrd` first-build race test used a cold pool and stayed
  green with the advisory lock deleted; it now warms the pool the way the
  reworked conversation test does, and goes red without the lock.
- **The model saw the current turn twice** (`r2j`). The coach persisted the
  member's message before `loadHistory` read it back and `HandleMessage`
  appended it again; the stub provider now sees the final user turn exactly
  once.
- **Responses-API stream errors were labelled anthropic** (`r2j`).
  `classifyInStreamError` now classifies by the Responses vocabulary
  (`rate_limit_exceeded` and `server_error` are transient, like their
  Anthropic counterparts) and names the provider that failed.
- **Unfenced model text reached the critic and planner** (`r2j`). The critic's
  focus-section names (model output, unvalidated) and the roadmap planner's
  PRD title/sections are fenced data now; a stored finding carrying a
  `</prd_data>` injection is neutralised at parse time.
- **A provider outage could close the circuit through expands** (`r2j`,
  `r2m`). `ExpandNode`'s template fallback still answers the user, but the
  admission gate now records the provider error, so a dead provider trips the
  breaker instead of resetting it. `Admit.done` is idempotent (`sync.Once`)
  and deferred in both handlers, so a panicking planner can no longer leak a
  permit — and the breaker rules themselves are pinned by a sequential test
  that reads the shared row as a delta and restores it.
- **The loopback rejection was unpinned** (`r2l`). An unparsable
  `INSIDEOUT_PUBLIC_URL` now sits in the rejecting table, and the mutant that
  treated it as loopback goes red. The stale prose the s2 verifier flagged
  (ReadTimeout pairing, the two body-cap comments, the backend/usage doc
  lines) says what the code does.
- **nginx capped the webhook below the server** (`r2h`, closing SEC-17 for
  the hosted topology). An exact-match `location = /api/v1/hooks/github`
  carries the `/api/` proxy directives with `client_max_body_size 8m`, and
  `scripts/check_nginx_webhook_body.py` fails when the location, its size, or
  the variable upstream go missing — mutation-checked against a literal
  `proxy_pass` and a 1m cap.
