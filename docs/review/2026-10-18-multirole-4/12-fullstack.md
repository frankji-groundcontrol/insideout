# 12 · Fullstack

**Lens:** client/server contract, session and SSE, CLI, docker/compose/nginx, env and docs agreement, accessibility and i18n, file size  
**Score:** **6.5 / 10**  
**Bottom line:** The stack mostly holds together, but the web Expand with AI button is always broken by a draft/confirm contract drift, and the CLI persists a 15-minute access token with no refresh. Docs and env agree except for a few small drifts.

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

## Skeptic verdicts on this lens's Highs

| ID | Filed | Verdict | Final | Reason |
|---|---|---|---|---|
| FULL-01 | High | CONFIRMED | High | Server handleExpandNode (server/internal/api/roadmap_ai.go) answers 200 with map{"children": views}, and the draft is written only by the separate POST /roadmap/{nid}/expand/confirm route. The client's expandRoadmapNode (client_project.dart:227-235) does `(d as List)` on res.data, and grep finds no client caller of the confirm … |
| FULL-02 | High | CONFIRMED | High | Reproduced with a go test overlay (tree untouched) that runs the real flow.Run with the real setupClient, setupProber and setupPersister against a local server with INSIDEOUT_ACCESS_TTL=4s. The persisted config contained api_base and token and no refresh field. whoami succeeded immediately, then after 6s returned 'unauthorized … |

---

## Scope

Role: fullstack (client/lib, client tests, CLI in `server/cmd/insideout`, `scripts/*.sh`, `docker/`, compose, `.env.example`, `docs/usage/keys/*`, nginx). Tree: clean worktree at `main` @ 9f0b1c7.

Ran: `flutter analyze` (1 info, `prefer_initializing` at `client/lib/api/client_core.dart:17-20`), `flutter test` (all pass, 458/458 in the contract test once `server/internal/api` was present), `go build` of the CLI, the CLI against a local server on a private Postgres, compose startup with only the app DSN, and a stub-adapter Dart test for the expand flow. Postgres and the local server are stopped.

Disclosure: two early CLI calls (`setup --check`, `whoami`) ran without `INSIDEOUT_API` set and reached the hosted default once (a reachability GET and an unauthenticated `GET /me`, no credentials). All later calls pinned `INSIDEOUT_API` to localhost. No hosted data was read or written. The default hostname is redacted as `<prod-host>`.

## Scores

| Aspect | Score | Why |
|---|---|---|
| Client/server contract | 5.5 | Tests pin paths only; the expand shape has drifted (FULL-01) |
| Session and refresh | 6.0 | Single-flight 401 refresh is good; web SSE and refresh error handling are inconsistent (FULL-04, FULL-05) |
| CLI | 6.0 | Verbs match labels, but `--help`/`version` need a DB URL and login does not persist (FULL-02, FULL-03) |
| Docker/compose/nginx | 7.5 | Runs the product; the owner-DSN default is a trap (FULL-06) |
| Env/docs agreement | 7.0 | Mostly coherent; two drifts (FULL-08, FULL-09) |
| A11y and i18n | 8.0 | en/zh key parity, Semantics on key widgets, no hardcoded UI strings found |
| File size | 8.0 | All reviewed files under ~350 lines (`sse.dart` 296) |

Overall 6.5.

## What is good

- `client/lib/api/client_core.dart`: single-flight 401 refresh, timeouts (connect 10s, receive 30s), `validateStatus < 500` so errors become typed `ApiException`s.
- `client/lib/api/sse.dart`: buffer parser with an idle timer (75s) and a refresh-and-retry wrapper for the coach stream.
- `client/lib/l10n/{en,zh}_*.dart`: exact key parity across the four files in each language.
- `client/nginx.conf`, `client/Dockerfile`, `docker-compose.yml`: unprivileged nginx with an envsubst template; the compose file passes the owner DSN only when the operator supplies it.
- `docker/postgres-init/001-create-app-role.sh` and `scripts/env-write.sh` (`quote_val`): the role split is real and written values are quoted.

## Findings

1. **FULL-01 (High): web "Expand with AI" always fails.** `client/lib/api/client_project.dart:231`: `(d as List)`. The server's `handleExpandNode` (`server/internal/api/roadmap_ai.go`) drafts only and returns `{"children":[...]}`; adding children needs a second `POST` to the confirm route (201). The client casts a map to a list, `guardWrite` (`roadmap_node_menu.dart:89-93`) turns it into the generic write-failed snackbar. Confirmed by a Dart stub test (cast error) and live curl. `requests_test.dart:77-79` and `feature_wiring_test.dart:49` pin the path only, so nothing caught it. The CLI/MCP `ExpandNode`/`ConfirmExpandNode` already follow the two-step shape.
2. **FULL-02 (High): CLI `login`/`setup` persist a 15-minute access token with no refresh path.** `server/internal/setup/flow/flow.go:151`: `Persister.Save(clientconfig.Config{APIBase: base, Token: deps.Client.Token()})`; the config has only APIBase and Token, and the default access TTL is 15m (`config.go:89`, measured exp minus iat = 900s). After 15 minutes every CLI/MCP call 401s, while `docs/usage/installation.md` (~40-60) promises "no export lines to copy". Also `login` only prints the token (`cli_auth.go:126,153`). Earlier reviews covered the web side only.
3. **FULL-03 (Medium): `insideout --help` and `version` demand a server DATABASE_URL.** `server/cmd/insideout/main.go:34-36`: `runClientCommand` returns `handled=false` for unknown verbs and help flags, so execution falls into `config.Load()` and prints "DATABASE_URL is required". Confirmed by running both. A first-time user's first command fails with a server error.
4. **FULL-04 (Medium): a 401 permanently stops live progress and presence on web.** `client/lib/features/project/roadmap_page.dart:84`: `if (e.status == 401 || e.status == 403 || e.status == 404) return;`. `fetchSseStream` (`sse_transport_web.dart`) throws on status >= 400 with no refresh, unlike the coach stream. After the access token expires mid-session the roadmap silently goes static until reload.
5. **FULL-05 (Medium): any refresh error logs the user out.** `client_core.dart:57`: `await _clearTokens();` inside a bare `catch (_)`. A network blip or 5xx during refresh clears the session; `session.dart` hydrate clears only on 401/403, so the two paths disagree. Clear only on 401/403.
6. **FULL-06 (Medium): compose with only the app DSN fails at startup.** `scripts/env.sh:220`: `info DATABASE_OWNER_URL "empty — migrate uses DATABASE_URL (must already be insideout_owner)"`. For the bundled Postgres that is an info, not an error, while `.env.example` marks the variable optional and compose defaults it to empty. Confirmed: migrate then fails with the owner-DSN error. Make the check an error for the bundled database.
7. **FULL-07 (Low): 20s CLI/MCP HTTP timeout covers AI endpoints.** `server/internal/apiclient/apiclient.go:27`: `Timeout: 20 * time.Second`. Draft/expand calls and the PRD build can run longer than the client allows even though the server turn deadline is configurable. Not reproduced against a slow model.
8. **FULL-08 (Low): unquoted value with spaces and angle brackets.** `.env.example:111`: `#INSIDEOUT_MAIL_FROM=InsideOut <onboarding@resend.dev>`. `scripts/dev.sh` loads `.env` with `set -a; . "$repo/.env"`; an operator who just uncomments the line gets a shell parse error. Quote it.
9. **FULL-09 (Low): undocumented knob.** `server/internal/agent/coach.go:50-54` reads `INSIDEOUT_AI_TURN_DEADLINE`; it is absent from `.env.example`, compose and `docs/usage/keys/`, which the env-catalog rules say should list every read variable.
10. **FULL-10 (Low): seed has hardcoded credentials and no non-local guard.** `server/cmd/insideout/seed.go` (seedEmail/seedPassword) runs against whatever `DATABASE_URL` points at, including a shared instance. Refuse unless the host is localhost or an explicit flag is passed.

## Lead verification

| Lead | Verdict | Evidence |
|---|---|---|
| L1 | Not filed (per instructions) | Static scan for sibling wall-clock dependence found none in Go or Dart tests |
| L2-L6, L9-L16 | Not individually reproduced | Related areas reviewed as above; no contradicting evidence |
| Web expand drift (lead) | Extended | Confirmed and shown to be a two-step flow (FULL-01) |
| CLI token lifetime (lead) | Extended | Measured 900s; no refresh persisted (FULL-02) |
| Compose owner DSN (lead) | Confirmed | Startup failure reproduced (FULL-06) |
| L7, L8, L17, L18 | Other reviewers | Not examined |

## Recommendations

1. Fix `expandRoadmapNode` to read `children`, show the draft, and call the confirm route; add a body-shape test that decodes a real server fixture.
2. Persist a refresh token (or have the CLI refresh on 401) so `login`/`setup` survive past 15 minutes; update the installation doc.
3. Add `--help`/`version` handling to `runClientCommand` before `config.Load`.
4. Route SSE 401s through the same single-flight refresh used by the coach stream, and clear tokens in the interceptor only on 401/403.
5. Make an empty `DATABASE_OWNER_URL` an error for the bundled database in `scripts/env.sh`.
6. Quote `INSIDEOUT_MAIL_FROM` in `.env.example`; document `INSIDEOUT_AI_TURN_DEADLINE`.
7. Raise or split the CLI HTTP timeout for AI calls.
8. Guard `seed` against non-local databases.

## What this role would do first

The cheapest user-visible fix is the CLI help/version case (about 15 lines, FULL-03), but the web expand repair (about 25 lines, FULL-01) removes an always-broken button, so I would do that first and add the contract-shape test in the same change.
