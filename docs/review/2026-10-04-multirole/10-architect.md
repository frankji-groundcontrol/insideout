# 10 · Architect

**Lens:** system structure vs. its own architecture docs; single vs. parallel
stacks; documented run path vs. real boot path; extension points and their
defaults; client/server architectural honesty.

**Score:** **6.0 / 10**

**Bottom line:** The Go server is well-built and matches its architecture
documents — clean `cmd`/`internal` split, one file per domain in `store/`
and `api/`, a two-schema LLM seam, minimal dependencies, one API contract
projected three ways (REST/CLI/MCP). The rot is at the doors: the README's
one-command compose path fails before it starts, `docs/INIT.md` still
describes the deleted Nuxt + Ink & Seal stack as today's foundation, the
resume path in `HANDOFF.md` orders an already-merged PR merge, and shipped
defaults (`INSIDEOUT_PUBLIC_URL`, the CLI/MCP default API base) still point
at the superseded stack or at production. The structure underneath is sound;
the maps at the entrance disagree with the building.

Reviewed against main@877ad03; High/Critical findings re-checked by an
independent skeptic — verdicts in 01-findings-register.md.

---

## Scope

README.md, AGENTS.md, CLAUDE.md, docs/index.md, docs/INIT.md, docs/PLAN.md,
docs/TODO.md, docs/HANDOFF.md, docs/plans/README.md board rows, all of
docs/architecture/, both server entry points (`cmd/insideout`,
`cmd/insideout-mcp`), `internal/config`, `internal/api/server.go`,
`internal/store/` layout + `migrate.go`, `internal/agent` layout,
`internal/mcpserver` (tool count from source), `internal/guide`,
`server/go.mod`, `docker-compose.yml`, `.env.example` (keys/comments only),
`client/pubspec.yaml`, `client/lib/main.dart`, `app.dart`, `router.dart`,
the `client/lib` feature layout, and the four client tests that name Nuxt.

## What is good

- **The server is structured exactly as documented.** `docs/architecture/backend.md`
  promises `cmd/insideout` (serve/migrate/seed) + `internal/{config,store,auth,httpx,api,agent,export}`;
  the tree delivers precisely that, one file per domain resource in `store/`
  and `api/` (`server.go:56-71` registers fourteen route groups). `go.mod`'s
  only non-core dependencies are pgx, jwt, crypto, mcp-go, and yaml — no web
  framework, no ORM, as claimed.
- **Migrations are embedded, transactional, and schema-confined.**
  `store/migrate.go:15-106` bootstraps `insideout.schema_migrations`, applies
  each embedded SQL file in its own transaction, and touches only the
  `insideout` schema — the AGENTS.md rule is enforced in code.
- **The LLM extension seam is real and narrow.** `agent/llm.go:103` is the
  only interface the coach depends on; `llm_factory.go:7-16` switches
  Anthropic Messages (default) vs. OpenAI Responses on `INSIDEOUT_LLM_SCHEMA`;
  `cmd/insideout/main.go:122-125` swaps in the offline template streamer when
  no key is set. Docs and code agree on every default without drift.
- **One truth, many views is implemented, not just preached.** CLI and MCP
  ride one `internal/apiclient` and return the API's raw JSON
  (`cmd/insideout/cli.go:3-8`, `mcpserver/product_tools.go:3-6`), so surfaces
  cannot drift from the contract. The board's "46 → 50 tools" matches the
  code: 39 product + 9 auth + 2 admin `AddTool` registrations.
- **Scaling boundaries are documented honestly.** Presence is process-local
  memory with the single-instance caveat stated (`product-subsystems.md:57-59`),
  and `deployment.md:44-50` documents boot-time migrations — verified in
  `main.go:113`.

## Findings

**ARCH-01 — High — the README's one-command compose path is broken twice.**
`docker-compose.yml:17-18` puts `:?` guards on `POSTGRES_OWNER_PASSWORD` /
`POSTGRES_APP_PASSWORD`, so `docker compose config` fails to parse even when
`DATABASE_URL` targets a remote instance (re-verified: exit 15, "required
variable POSTGRES_OWNER_PASSWORD is missing"). And README.md:72-77 promises
"bring up the whole stack with docker-compose", but the compose file has no
frontend service — `deployment.md:23` admits "Local compose does not build a
frontend." Failure scenario: a new operator on a clean checkout runs the
documented command; compose errors out before doing anything, and even after
satisfying the guards they get a backend-only "whole stack". Fix: gate the
postgres service behind an opt-in compose profile so the file parses and runs
with a remote `DATABASE_URL`, and either add the Flutter web service or
reword README to "backend + database". diff_size: ~10 lines.

**ARCH-02 — High — `docs/INIT.md` names the deleted stack as the current
foundation.** The doc map sends readers here, and it states: "InsideOut
currently runs as a Go API over PostgreSQL with row-level security, a Nuxt 4
SSR application, a direct Anthropic PRD Coach integration, and an Ink & Seal
bilingual interface" (`docs/INIT.md:25-27`). The same docs system says the
Nuxt tree was deleted 2026-08-18 and production is Flutter wearing Cobalt
(`docs/architecture/frontend.md:8-15`). Failure scenario: an agent (human or
AI) boots from the doc map in order, plans work against a Nuxt app that does
not exist, and distrusts the architecture docs. Fix: rewrite the section to
"Go API + Flutter 3 web client (Cobalt) + direct Anthropic/Responses-schema
client". diff_size: 2 lines.

**ARCH-03 — Medium — the resume path is stale and contradicts itself.**
`docs/HANDOFF.md:10-11` still orders "merge PR #3 … then `railway up
--service app`" (the board shows both Cobalt plans Finished with the same
next action, and PR #3 merged 2026-09-18). Numbering runs 1, 2, 2, 4, 5
(lines 10/16/24/29/37); item 2 says "27 MCP tools" (line 22), item 3 says
"16 tools" (line 26) — the code registers 50. AGENTS.md requires HANDOFF to
be the short, accurate resume path; today its first command is a no-op at
best and an unplanned deploy at worst. Fix: drop item 1, renumber, sync the
tool count from `internal/mcpserver`. diff_size: ~8 lines.

**ARCH-04 — Medium — `INSIDEOUT_PUBLIC_URL` defaults to the deleted Vite
dev port.** `config.go:78` defaults it to `http://localhost:5173` (mirrored
in `.env.example:109` and documented at `docs/usage/environment.md:102` as
"the base of every emailed magic/reset link"). The documented Flutter run
command picks an ephemeral port; nothing serves on 5173 on the documented
local path, so emailed links 404 unless the operator pins
`--web-port=5173` — which is exactly the workaround `HANDOFF.md:42` bakes in
instead of fixing the default. Fix: default to the real Flutter web origin
or fail loudly when mail is enabled without an explicit `INSIDEOUT_PUBLIC_URL`.
diff_size: ~4 lines.

**ARCH-05 — Medium — compose forwards only the LLM variables.**
`docker-compose.yml:38-52` passes `DATABASE_URL`, JWT secret, and the four
`INSIDEOUT_LLM_*` keys — nothing else from the catalog (`.env.example:99-133`):
no `INSIDEOUT_MAIL_API_KEY`/`MAIL_FROM`/`PUBLIC_URL` (mail silently runs
log-only), no `INSIDEOUT_GH_*` (webhook/guide token path dead), no
`INSIDEOUT_CORS_ORIGINS`, `INSIDEOUT_COOKIE_SECURE`, or TTLs. The compose
server is a feature-amputated build of the same binary. Fix: forward the
full catalog with `:-` defaults. diff_size: ~12 lines.

**ARCH-06 — Medium — CLI and MCP binaries default to the hosted production
API.** `cmd/insideout/cli.go:19` and `cmd/insideout-mcp/main.go:20` both
hardcode `defaultAPIBase = "https://<railway-host>/api/v1"`. A clean
checkout's `insideout login` silently reads and writes production unless
`INSIDEOUT_API` happens to be set — and the hostname is baked into source
(redaction-rule overlap). Fix: make `--api`/`INSIDEOUT_API` explicit (or
default to `127.0.0.1:8080`) and hoist the constant into one shared package.
diff_size: ~6 lines.

**ARCH-07 — Medium — `docs/TODO.md`'s current delta describes a build that
no longer exists.** Its header (last updated 2026-08-13) says the build "does
not yet implement the version-first Coach, audience projections, Product
Branches and human version authority, private Ideas, project roles,
deadline-bound Now work, automatic Roadmap evidence, or shared CLI/MCP/Agent
context" (`TODO.md:11-13`). The board records audience views (2026-08-22),
deadline/pressure Progress (2026-08-22), the evidence loop (2026-08-21), and
CLI/MCP parity (2026-08-21, 46→50 tools 2026-09-03) all shipped. The
"shipped-system tracker" contradicts the shipped system. Fix: refresh the
delta section. diff_size: unknown (docs pass).

**ARCH-08 — Low — live config surfaces still explain behavior via Nuxt.**
`.env.example:89-90` tells operators permissive CORS is "rarely needed: the
browser talks same-origin through the Nuxt proxy"; `config.go:31` comments
"(Nuxt same-origin needs none)"; `api/auth.go:246` says fields stay "top-level
so Nuxt can keep assigning the JSON to UserProfile". The proxy is nginx on the
Flutter service; operator-facing comments name a deleted component. Fix:
rename to nginx/Flutter in all three. diff_size: 3 lines.

**ARCH-09 — Low — client tests use the deleted Nuxt app as the parity
oracle.** `client/test/requests_test.dart:5` ("workspace update and delete
match Nuxt"), `l10n_test.dart:7` ("default locale matches Nuxt zh-CN"),
`appearance_test.dart:5`, `api_errors_test.dart:21` — the reference
implementation no longer exists, so the contract these tests pin is
unverifiable against its stated source. Fix: rename to "legacy contract"
and keep the assertions. diff_size: ~8 lines.

**ARCH-10 — Low — three over-budget files lack the dated modularity issue
AGENTS.md requires.** `server/internal/agent/coach.go` (410 lines),
`server/internal/agent/anthropic.go` (406), and `client/lib/api/models.dart`
(472) exceed the ~350 budget; `docs/issues/` tracks only the two apiclient
files (`2026-09-03-apiclient-line-budget.md`,
`2026-09-18-flutter-apiclient-line-budget.md`). Fix: file the dated issues
with target structures, or split. diff_size: unknown.

**ARCH-11 — Low — frontend layout docs drifted from `client/lib/`.**
`docs/architecture/frontend.md:43` lists a `roadmap` feature directory; the
roadmap UI lives under `features/project/` (`roadmap_page.dart`,
`roadmap_bands.dart`, …), and `PLAN.md:33-38` omits `chrome/`, `theme/`,
`router.dart`, `app.dart`. Fix: one-line syncs in both docs. diff_size: 2 lines.

## Lead verification

| Lead | Verdict | Evidence (first-hand unless noted) |
|---|---|---|
| L2 compose fails + overpromises | Confirmed | `docker compose config` re-run: exit 15 on POSTGRES_OWNER_PASSWORD; no frontend service |
| L3 README = superseded product model | Confirmed | `README.md:27-33` four-stage Coach + admin approve/reject; PRODUCT.md marks both superseded (coordinator-verified) |
| L5 HANDOFF/board stale | Confirmed | HANDOFF:10-11, 22, 26; numbering 1,2,2,4,5; board rows `plans/README.md:13-14` still say "Merge PR #3"; MCP tools from source = 50, not 27/16 |
| L6 stale Nuxt references | Confirmed | `.env.example:89-90`, `config.go:31`, `auth.go:246`, `INIT.md:26`, four client tests |
| L7 size-rule breaches | Confirmed | `wc -l`: apiclient.go 478, coach.go 410, anthropic.go 406, client.dart 529, models.dart 472; issues cover only the apiclient files |
| L9 compose env gaps | Confirmed | `docker-compose.yml:38-52` forwards only LLM vars beside DATABASE_URL/JWT |
| L11 identifiers in docs/source | Confirmed | README.md:64, HANDOFF.md:43, frontend.md:24, cli.go:19, insideout-mcp/main.go:20 (redacted here) |
| L12 build/vet clean | Confirmed | `go build ./... && go vet ./...` re-run: OK |
| L13 flutter analyze/test | Not run | Plans board claims analyze + 54 tests green 2026-09-18 — unverified by me |
| L14 hosted deploy behind main | Not run (consistent) | Autodeploy off per HANDOFF:52-56; cli/MCP defaults still point at the hosted instance |

## Recommendations

1. Decide what the compose file is for — one-command whole stack (add a
   Flutter service, forward the full env catalog) or dev database helper
   (delete the "whole stack" claim, profile-gate postgres) — and make
   `docker compose config` parse with a remote `DATABASE_URL`.
2. Rewrite `docs/INIT.md:25-27` to the actual stack (Go + Flutter/Cobalt +
   direct Anthropic/Responses-schema client).
3. Rewrite the `docs/HANDOFF.md` item list (drop the merged PR, renumber, 50
   MCP tools) and refresh `docs/TODO.md`'s delta to the shipped state.
4. Fix the `INSIDEOUT_PUBLIC_URL` default (plus `.env.example` and
   `environment.md`) to the real Flutter dev origin; delete the
   `--web-port=5173` workaround from HANDOFF.
5. Decide the CLI/MCP default-API policy (explicit flag vs. localhost
   default), hoist the constant out of two mains, keep the hostname out of
   source.
6. Sweep Nuxt out of live surfaces (`.env.example`, `config.go`, `auth.go`
   comments); rename the four client parity tests to the legacy contract.
7. File dated modularity issues for `coach.go`, `anthropic.go`, `models.dart`
   (or split them); sync `frontend.md`/`PLAN.md` layouts with `client/lib/`.

## What this role would do first

Decide the compose file's identity — it is the cheapest decision that unblocks
the most: today the README's front door (`docker compose up -d`) fails before
it starts, the same file silently amputates mail, GitHub hooks, CORS, and TTL
configuration, and every fix for ARCH-01/ARCH-05 forks on whether compose is
a demo of the product or a backend helper. Once the owner says "one-command
whole stack" or "dev database only", the profile-gating, env forwarding,
README wording, and healthcheck work follow mechanically — and the documented
path finally matches the real boot path.
