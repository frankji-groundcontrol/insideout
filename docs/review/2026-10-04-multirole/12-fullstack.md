# 12 · Full-stack

**Lens:** Flutter web UI ↔ Go API route parity, env-var wiring across
compose / `.env.example` / code, Docker deliverables, README install
promise, API_BASE wiring, SSE streaming path, file-size rule.

**Score:** **5.5 / 10**

**Bottom line:** The two halves of this stack are individually healthy
and well-matched — every `ApiClient` method has a registered server
route, the web SSE path is correctly engineered on fetch +
ReadableStream after a documented XHR failure, and both toolchains
verify green (`go build ./...` OK; `flutter analyze` info-level only;
`flutter test` 57/57). But the seams break the product: the README's
flagship `docker compose up -d` fails at parse time, the compose stack
contains no UI at all, the admin invitation deep link drops its code
before reaching the register form, and the documented local-dev
quickstart silently fails on CORS. A new user cannot get from README to
a working product without hitting at least two dead ends.

Reviewed against main@877ad03; High/Critical findings re-checked by an
independent skeptic — verdicts in 01-findings-register.md.

---

## Scope

`client/lib/` (router, features, api/, session/), all `server/internal/api/`
route registrations, `client/lib/api/client.dart`, `docker-compose.yml`,
`server/Dockerfile`, `client/Dockerfile`, `client/nginx.conf`,
`.env.example`, `scripts/dev.sh`, `scripts/env.sh`,
`server/internal/config/config.go`, `docs/usage/*.md` (the three env/run
docs), `README.md`, `client/README.md`. Ran: `docker compose config`
under a remote-targeting env, `go build ./...`,
`flutter analyze --no-fatal-infos`, `flutter test`.

## What is good

- **Client↔server route parity is essentially perfect.** All ~70
  `ApiClient` methods in `client/lib/api/client.dart` match the
  `mux.HandleFunc` registrations across `server/internal/api/` — path,
  method, and shape, including the newer surfaces (`/prds/{id}/view`,
  `/prds/{id}/export`, roadmap move/expand, presence stream, GitHub sync).
- **Web SSE is solved properly.** `client/lib/api/sse_transport_web.dart`
  streams via `fetch` + `ReadableStreamDefaultReader` (comment explains
  why dio/XHR cannot, BUG-014), `sse.dart` handles multibyte splits, and
  nginx disables buffering with a 3600 s read timeout (`client/nginx.conf`).
- **The env tooling is unusually disciplined.** `.env.example` is a
  parsed schema; the `scripts/env.sh` catalog matches
  `server/internal/config/config.go:73-90` name-for-name; `scripts/dev.sh`
  preflights validation before launching.
- **The two-role Postgres model composes cleanly**: `depends_on:
  service_healthy`, an init script keeping the superuser away from
  `SECURITY DEFINER` functions, and honest comments about compose's
  limits (`docker-compose.yml:40-46`).
- **Flutter tests are real and green**: 57/57 pass, including a
  leftover-scan test forbidding Ink & Seal tokens in live Dart.

## Findings

### FS-01 — High — `docker compose up -d` cannot parse without POSTGRES_* passwords even for remote-only setups

`docker-compose.yml:17-18` uses `:?` guards:
> `INSIDEOUT_OWNER_PASSWORD: ${POSTGRES_OWNER_PASSWORD:?set POSTGRES_OWNER_PASSWORD in .env}`

`README.md:72-76` promises "一键启动完整技术栈 / bring up the whole stack"
via `docker compose up -d`, and `README.md:66` says only two variables are
required. Verified by running `docker compose config` with those two set
to a remote DSN: "required variable POSTGRES_OWNER_PASSWORD is missing a
value". The guards also contradict `.env.example:157-162`, which marks
the passwords required *only* for the bundled-postgres setup.

*Failure scenario:* fresh clone → set the two README-required vars →
`docker compose up -d` exits non-zero before creating anything.
*Fix:* relax both guards to `:-` (2 lines), matching `.env.example`;
`env.sh check` already enforces the pairing when bundled postgres is in
use.

### FS-02 — High — the admin invitation deep link drops `?invite=` before the register form

`server/internal/api/admin.go:129`:
> `link := s.cfg.PublicURL + "/register?invite=" + code`

but `client/lib/router.dart:28` registers
> `'/register': (c, s) => const RegisterPage(),`

and `RegisterPage` (`client/lib/features/auth/register_page.dart:7-12`)
takes no query parameter — unlike its neighbors `ResetPasswordPage` and
`MagicLinkPage`, which read `s.uri.queryParameters`. `AuthForm`
(`client/lib/features/auth/auth_form.dart:28,88-89`) has an invite field
that nothing pre-fills.

*Failure scenario:* admin creates an invitation (the UI offers "copy
link" for that URL) → invitee opens it under `invitation_only` signup →
the code is silently lost, registration 403s, and the only recovery is
the admin re-sending the raw code out of band.
*Fix:* thread the parameter: `RegisterPage(invite: s.uri.queryParameters['invite'])`
→ `LandingPage`/`AuthForm` prefill (~10 lines), mirroring the magic-link
page.

### FS-03 — High — the documented local-dev quickstart fails on CORS with no hint

`client/lib/session/session.dart:7-10` defaults `API_BASE` to
`http://127.0.0.1:8080/api/v1`, and `client/README.md` ("Local default")
plus `docs/usage/local-development.md:45,114` run `flutter run -d chrome`
against it. Flutter web serves on `http://localhost:<ephemeral>` — always
cross-origin to `127.0.0.1:8080` — and the server ships no CORS unless
`INSIDEOUT_DEV_CORS=1` or `INSIDEOUT_CORS_ORIGINS` is set
(`server/internal/api/server.go:79-83`). Neither quickstart mentions
either knob; only `docs/usage/environment.md:97` does. The hosted API
"already allows localhost" (HANDOFF.md:41), which masks this for
hosted-API dev but not the documented local-server path.

*Failure scenario:* new dev follows README → UI loads, every API call
blocked by the browser, no UI error hint; fix is buried in
`docs/usage/environment.md:97`.
*Fix:* add one line to both quickstarts: set `INSIDEOUT_DEV_CORS=1` (or
`INSIDEOUT_CORS_ORIGINS=localhost`) in `.env` for Flutter-web dev (~4
lines of docs). Also correct the stale claim at `.env.example:89` ("Rarely
needed … through the Nuxt proxy") — for Flutter web it is the norm.

### FS-04 — High — README promises the "whole stack"; compose delivers API-only, no UI

`README.md:73`: "Or bring up the whole stack with docker-compose". The
compose file has exactly two services (`postgres`, `server`); there is no
client service, and the Go server serves no static files (no embed or
file server anywhere in `server/internal/api` or `server/cmd`). A
`client/Dockerfile` exists — but its nginx proxies to a Railway-internal
host (see FS-06), so it cannot be dropped into compose either.

*Failure scenario:* dev fixes FS-01, runs `docker compose up -d` → gets
postgres + API on :8080 and no way to open a browser UI from that stack.
*Fix:* either add a client service to compose (build `client/` with
`API_BASE=/api/v1`, proxy to the `server` service) or reword README to
"the backend stack" and point to `client/Dockerfile` (~2 lines for the
reword; new service unknown/larger).

### FS-05 — Medium — compose server forwards 12 of ~24 cataloged vars; mail, GitHub hooks, CORS unconfigurable

`docker-compose.yml:38-52` forwards addr, both DSNs, JWT secret, and the
four LLM vars — but not `INSIDEOUT_MAIL_*` / `INSIDEOUT_PUBLIC_URL`,
`INSIDEOUT_GH_*` / `GITHUB_TOKEN`, `INSIDEOUT_CORS_ORIGINS` /
`INSIDEOUT_DEV_CORS`, `INSIDEOUT_COOKIE_SECURE`, or the TTLs, all present
in the `.env.example` catalog and `config.go`.

*Failure scenario:* compose-deployed server can never send auth email
(links silently dev-logged against the `localhost:5173` default),
`POST /api/v1/hooks/github` is always 503, and no browser client can talk
to it cross-origin.
*Fix:* add pass-through lines with `:-` defaults matching `.env.example`
(~10 lines).

### FS-06 — Medium — client nginx hardcodes the Railway-internal upstream and resolves it once at startup

`client/nginx.conf:14`:
> `proxy_pass http://server.railway.internal:8080/api/;`

This bakes a `<railway-host>` identifier into a tracked file, makes the
image runnable only inside that one Railway project, and — because nginx
resolves the upstream at startup — every server redeploy 504s `/api`
until the app service restarts, a fragility already recorded as a manual
runbook step (`docs/HANDOFF.md:53-57`).

*Failure scenario:* any server redeploy, or running the built image
anywhere else → `/api/*` 504s with no in-repo fix.
*Fix:* make the upstream a build arg / env-substituted variable with a
`resolver` entry, or run nginx on the official unprivileged image with a
startup template (~5-10 lines).

### FS-07 — Medium — server image runs as root with no healthcheck, in compose or the Dockerfile

`server/Dockerfile` (entire file, 11 lines) ends at
> `ENTRYPOINT ["/insideout"]`

with no `USER` directive and no `HEALTHCHECK`; the compose `server`
service likewise has none (only `postgres` does, `docker-compose.yml:24-28`).
`gcr.io/distroless/static-debian12` ships a `nonroot` user (65532) that
goes unused.

*Failure scenario:* a container-level exploit runs as root; orchestrators
get no readiness signal — a hung server looks "up" until requests time out.
*Fix:* add `USER nonroot:nonroot`, a HEALTHCHECK probing `/healthz` (the
handler exists at `server/internal/api/healthz.go`), and a compose
healthcheck (~6 lines).

### FS-08 — Low — emailed auth links default to `localhost:5173`, but the canonical dev doc never pins that port

`server/internal/config/config.go:78` defaults `INSIDEOUT_PUBLIC_URL` to
`http://localhost:5173` (a Vite-era port), and `HANDOFF.md:42` pins
`flutter run --web-port=5173` to match — but
`docs/usage/local-development.md:114` says plain `flutter run -d chrome`,
which serves on an ephemeral port.

*Failure scenario:* dev tests magic-link login locally → the emailed link
opens an origin where nothing listens.
*Fix:* add `--web-port=5173` to the two `flutter run` lines in
local-development.md (2 lines), or default `PUBLIC_URL` blank once mail
is configured.

### FS-09 — Low — `feature_wiring_test` proves string presence, not behavior — and it passed while FS-02 was broken

`client/test/feature_wiring_test.dart` asserts
`settings.contains('updateWorkspace')`-style checks over source text. It
gives real regression value, but as the only guard on UI↔API wiring it
cannot catch a dropped deep-link parameter: it was green at main@877ad03
while FS-02 silently discards the invitation code.

*Failure scenario:* a future route/param change that leaves the string in
place but breaks the flow ships green.
*Fix:* add a router-level widget test asserting `/register?invite=X`
reaches the form controller (and equivalents for `?token=`); ~30 lines.

### FS-10 — Low — `client/lib/api/models.dart` (472 lines) exceeds the 350-line budget with no dated issue

AGENTS.md requires a dated issue per over-budget file. `client.dart`
(529) is tracked by `docs/issues/2026-09-18-flutter-apiclient-line-budget.md`,
and the server apiclient pair by `2026-09-03-apiclient-line-budget.md` —
but `models.dart` has none (nor do `server/internal/agent/coach.go` 410
and `anthropic.go` 406, outside my reads but confirmed by count).

*Failure scenario:* the next oversized file lands with no tracking issue
and the size rule erodes file-by-file, as it already has in agent/.
*Fix:* file a dated issue with the target split (e.g. `prd_models.dart` /
`workspace_models.dart`) mirroring the existing two (~15 minutes).

### FS-11 — Low — `client/README.md` hardcodes the production API host

`client/README.md:24` shows
`--dart-define=API_BASE=https://<railway-host>/api/v1` with the literal
production hostname in place of `<railway-host>` — a deployment
identifier in a tracked doc, against the AGENTS.md redaction rule (same
class as lead L11, which lists other files but not this one).

*Failure scenario:* the deploy moves or the repo is published — a live
deployment identifier stays in git history and the copy-pasted command
silently rots.
*Fix:* replace with `<railway-host>` placeholder plus a "find yours in
the Railway dashboard" note (2 lines).

## Lead verification

| Lead | Verdict | Evidence |
|------|---------|----------|
| L2 compose parse failure | **Confirmed** | Ran `docker compose config` with only the two README-required vars set (remote DSN): "required variable POSTGRES_OWNER_PASSWORD is missing a value" (`docker-compose.yml:17-18`). "Whole stack" overpromise additionally refuted — FS-04. |
| L7 size breaches | **Partially refuted** | Counts confirmed (478/410/406/529/472), but "issues track only the two apiclient files" is stale: `docs/issues/2026-09-18-flutter-apiclient-line-budget.md` tracks client.dart. `models.dart`, `coach.go`, `anthropic.go` remain untracked (FS-10). |
| L8 Dockerfile no USER/HEALTHCHECK | **Confirmed** | `server/Dockerfile` is 11 lines, no `USER`, no `HEALTHCHECK`; only `postgres` has a compose healthcheck. |
| L9 compose env passthrough gaps | **Confirmed** | `docker-compose.yml:38-52` omits mail (`INSIDEOUT_MAIL_*`, `PUBLIC_URL`), GitHub (`INSIDEOUT_GH_*`, `GITHUB_TOKEN`), CORS knobs, `COOKIE_SECURE`, both TTLs — all in `.env.example` and `config.go`. |
| L13 flutter analyze/test | **Confirmed (ran)** | `flutter analyze --no-fatal-infos`: 20 info-level issues, no errors/warnings. `flutter test`: 57/57 pass. |
| L6 stale Nuxt refs in my reads | **Confirmed** | `.env.example:89` "same-origin through the Nuxt proxy"; Nuxt named in live test descriptions at `client/test/api_errors_test.dart:21`, `appearance_test.dart:5`. |
| L14 hosted deploy behind main | Not verified (deployment ops, outside my files) | — |

## Recommendations

1. Relax the two `POSTGRES_*` `:?` guards in `docker-compose.yml` to `:-`
   so the README command parses with the vars the README says are
   required (FS-01).
2. Thread `?invite=` from `router.dart` into the register form so the
   admin invitation flow works end-to-end (FS-02).
3. Add the CORS knob to both dev quickstarts and fix the stale "Nuxt
   proxy" comment in `.env.example` (FS-03).
4. Decide what compose is for: add a client service, or stop calling it
   the "whole stack" (FS-04).
5. Pass the remaining cataloged vars through the compose `server`
   service with `:-` defaults (FS-05).
6. Parameterize `client/nginx.conf`'s upstream on an unprivileged nginx
   with a startup template (FS-06).
7. Harden `server/Dockerfile`: `USER nonroot:nonroot`, a `/healthz`
   HEALTHCHECK, and a compose healthcheck (FS-07).
8. File the missing dated size issues and add a behavioral deep-link
   test beside `feature_wiring_test` (FS-09, FS-10).

## What this role would do first

Fix FS-02, the invitation deep link. It is the cheapest change on the
list (~10 lines, mirroring a pattern two routes over in the same router
file), entirely user-visible — an admin inviting a teammate is a
first-week flow for exactly the audience PRODUCT.md targets — and it
restores an end-to-end path silently broken today despite green builds,
green tests, and route parity everywhere else. The compose and CORS
findings matter more to newcomers overall, but they need a product
decision (what is compose for?) and a doc edit respectively; the invite
fix is pure repair with no trade-offs and should ship first.
