# 10 · Architect

**Lens:** layering and seams (api vs agent vs store), parallel stacks across HTTP, CLI, MCP, documented architecture vs code, hidden coupling (wall clock, global state, baked DefaultBase), config surface and defaults, extension points and second deployment, module boundaries  
**Score:** **7 / 10**  
**Bottom line:** The server core has clean layering and honest operational design, but the client edge (CLI, MCP, hosted-default) has four disagreeing base/credential resolvers that can send a stored token to the hosted deployment, plus wall-clock, single-replica and doc-drift debt.

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

## Skeptic verdicts on this lens's Highs

| ID | Filed | Verdict | Final | Reason |
|---|---|---|---|---|
| ARCH-01 | High | CONFIRMED | High | Reproduced end to end on localhost. I built insideout and insideout-mcp from a scratch clone with apiclient.DefaultBase patched to a localhost sentinel (C, standing in for <prod-host>). The scratch HOME config had api_base=A and a dummy token; sentinels A, B and C recorded each request's path and whether a Bearer header was … |

---

## Scope

Architect lens on InsideOut at main @ 9f0b1c7 (clean worktree, read-only). I read CLAUDE.md, AGENTS.md, docs/architecture/*, server/cmd/*, the api/agent/store/agentcontext/mcpserver/apiclient/clientconfig/config packages, the client/lib layout, the Dockerfile and docker-compose. I traced the import graph, the three client surfaces (CLI, MCP, Flutter) and the process-state model. I ran two throw-away probe tests in a scratch copy of the server tree (never in the review tree) to confirm the client-resolution defects below. I did not start Postgres and did not touch the network.

## Scorecard

| Aspect | Score | Why |
|---|---|---|
| Layering (api / agent / store) | 8.5 | Import graph is acyclic and shallow: `store` imports only `prdcommit`; `agent` imports `store`, `httpx`, `readiness`; `api` consumes `agent` through a `Coach` interface. Leaf packages (readiness, audienceview, roadmaptime, prdcommit) import nothing. |
| Parallel stacks across HTTP / CLI / MCP | 5.0 | One shared `apiclient`, but the base-URL and credential resolution is implemented four separate times and they disagree (ARCH-01, ARCH-02). |
| Docs match code | 6.0 | Good structure, but several current-state claims are false (ARCH-06). |
| Hidden coupling | 6.0 | Wall-clock reads inside response builders, a global `sync.Map` and a DB lock hidden in a throttle constructor (ARCH-03, ARCH-04). |
| Config surface and defaults | 7.0 | Fail-fast, secure-by-default (`COOKIE_SECURE` on, dev-CORS loopback-only). Hosted default baked in as a const (ARCH-05). |
| Extension points / second deployment | 5.5 | One replica by design; the hosted API is not retargetable without a source edit. |
| Module boundaries | 8.0 | Files stay small, one resource per file, `internal/` boundaries respected. |

Overall: **7.0 / 10**. The server core is cleaner than most projects of this size. The weakest part is the client edge: the thing users actually install.

## What is good

- Clean seam between HTTP and the agent. `server/internal/api/server.go:21` defines `Coach` as a narrow interface, and `cmd/insideout/main.go` is the only place that wires concrete agent, store and config together.
- One source of truth for PRD section keys. `store.PrdSectionKeys` is used by api (`internal/api/prds.go:78`), the agent tools (`internal/agent/tools.go:165`) and prompts, so the HTTP and agent write paths cannot disagree on the vocabulary.
- Defense in depth is real and documented: app-layer membership checks plus RLS (`docs/architecture/database-and-rls.md`), a separate owner pool for admin promotion (`cmd/insideout/main.go` `runPromoteAdmin`).
- Operationally mature binary: shutdown drain with a base-context cancel, reaper for stale AI runs, self-probing HEALTHCHECK on a distroless nonroot image (`server/Dockerfile`).
- The singleton guard (`internal/api/single_replica.go`) makes the single-replica contract loud at startup rather than silent, which is the honest way to ship per-process state.

## Findings (worst first)

**ARCH-01 (High): the client API base is resolved four different ways, and they disagree.**
- `server/cmd/insideout-mcp/main.go:22`: `api := flag.String("api", envOr("INSIDEOUT_API", apiclient.DefaultBase), ...)` never consults the config file, while `internal/mcpserver/server.go` `sessionToken()` does read the token from it. After `insideout setup` against a self-hosted server (api_base and token written to config), the MCP server calls the hosted default carrying the self-hosted token.
- `server/cmd/insideout/cli_roadmap.go:27`: `c := apiclient.New(api)` is built before `applyAPIFlag(fs, &api)` runs. Confirmed by probe: `roadmap list --api <B>` hit the config server (A=1, B=0). The `--api` flag is dead for every `roadmap`, `presence` and version verb (`cli_version.go:16` has the same shape).
- `server/cmd/insideout/cli_roadmap.go:207`: `fs.StringVar(api, "api", envOr("INSIDEOUT_API", defaultAPIBase), ...)` overwrites the config-aware base passed in. Confirmed by probe: with `api_base` in config and no env, `guide` and any verb that builds its client after the flag resolves to the hosted default.
- `scripts/mcp-config.sh` is a fourth implementation (bash plus python reading the config file) that exists to paper over the Go binary not doing it.
- Failure scenario: a self-hosting user runs `insideout setup`, then `insideout guide <id>` or the MCP server; their bearer token goes to the maintainer's hosted deployment.
- Fix: add `clientconfig.Resolve(flagAPI string) (base, token string)` and use it in cli.go, every `applyAPIFlag` caller (build the client after parsing), and `mcpserver.Build`; delete `envOr`/`defaultAPIBase` copies. About 40 lines. Add a test that every verb honors `--api` over config.

**ARCH-02 (Medium): a stored token is not bound to its base, and a config error silently selects the hosted host.**
- `server/internal/clientconfig/clientconfig.go:51-56`: env overrides apply per field; the test pins it (`clientconfig_test.go:51` "api env only overrides APIBase" keeps the file token). Setting `INSIDEOUT_API` to any host sends the stored token to it.
- `server/cmd/insideout/cli_setup.go:32`: `if err != nil || cfg.APIBase == "" { return defaultAPIBase }`. `Load` returns before applying env on a parse error (probe confirmed: corrupt config plus `INSIDEOUT_API=http://127.0.0.1:1/api/v1` still resolved to the hosted default). `insideout login <email>` then posts the password to the hosted API.
- Fix: persist the base with the token and drop the file token when the effective base differs; make `Load` apply env before returning a parse error and make callers surface the error instead of defaulting.

**ARCH-03 (Medium): the single-replica contract is enforced by global state and a side effect hidden in a middleware constructor.**
- `server/internal/api/single_replica.go:34`: `var singletonGuards sync.Map // map[*Server]heldConn`, justified by "Server cannot grow fields without touching server.go" (line 31). The lock is acquired from `auth_throttle.go:208` (`s.startSingletonGuard()`) inside `withAuthThrottle`, i.e. inside `Handler()`.
- The guard only logs; a second replica (or a rolling deploy overlap) keeps serving with doubled throttle, cooldown and SSE budgets. The architecture docs never mention the contract except `docs/architecture/deployment.md:41`.
- Scenario: a second contributor adds a field or a second `Handler()` call and the lock behavior changes; a pooled (transaction-mode) `DATABASE_URL` makes the session advisory lock meaningless.
- Direction: put the guard on `Server` with an explicit `Start()` called from `runServe`; document the contract in `backend.md`. First check: is `DATABASE_URL` ever a transaction pooler on the hosted deploy.

**ARCH-04 (Medium): no clock seam in the API response layer (siblings of the L1 time bomb).**
- `server/internal/api/roadmap.go:70`: `v.Pressure = roadmaptime.Pressure(*n.Deadline, time.Now())`; `server/internal/api/roadmap_progress.go:111`: `return roadmaptime.Assemble(in, time.Now()), nil`; `server/internal/api/projects.go:84`: `Flagged: projectQuietFlagged(p, time.Now())`. Meanwhile `server.go:67` already injects a clock for the throttle (`newAuthThrottle(defaultThrottleConfig(), time.Now)`), so the pattern exists but is applied once.
- The two siblings are not failing today because their tests use relative dates; the fixed 2026 dates elsewhere (agentcontext, roadmaptime, presence) are injected and safe. But any future test that pins a date through these builders fails on a calendar boundary, exactly like L1.
- Fix: add `now func() time.Time` to `Server` and thread it into `projectResponse`/`roadmapNodeResponse`/the progress assembler; one repo-wide grep guard test that `time.Now()` does not appear in `internal/api` view builders.

**ARCH-05 (Medium): the hosted deployment is a compile-time constant, so a fork or second deployment cannot retarget the shipped binaries.**
- `server/internal/apiclient/default_base.go`: `const DefaultBase = "..."` (a const cannot be set with `-ldflags -X`). `default_base_test.go:13` pins the literal (`if DefaultBase != "https://..."`), a tautology that blocks retargeting. The host also lives in `.github/workflows/ci.yml:288` (APK `--dart-define=API_BASE`) and `smoke.yml:122`.
- Scenario: someone forks and runs the release workflow; their CLI, MCP and APK default to the upstream operator's server and send credentials there.
- Fix: make it a `var` set by `-ldflags` from `scripts/default-api.sh` or a repo variable, and have the test assert the default equals what the script prints.

**ARCH-06 (Medium): architecture docs make false current-state claims.**
- `docs/architecture/backend.md:24`: "clientconfig/ single source for CLI credentials: env overrides the 0600 config" (false for MCP, see ARCH-01).
- `docs/architecture/backend.md:53`: "Authentication is httpOnly-cookie based" while `frontend.md:15` says Bearer, and the index diagram says Bearer.
- `docs/architecture/backend.md:87-88` lists 11 store files; the directory has 30 (roadmap*, invitations, auth_tokens, proposal_*, github_webhook, etc. missing).
- There is no architecture document for the CLI/MCP/client-config surface at all, though it is half of the product. `server/internal/api/server.go:20` still says "api doesn't import agent until it's wired in P4".
- Fix: add `docs/architecture/clients.md` (resolution order, surfaces, the single-replica contract) and correct the three lines.

**ARCH-07 (Low): CLI to MCP to HTTP parity is by convention only.**
- `server/internal/mcpserver/product_tools.go:3`: "names/arguments/output 1:1 with the CLI". Route list, apiclient methods, CLI verbs and MCP tools are four hand-kept lists with no test relating them; ARCH-01 shows the surfaces already drift.
- Direction: a table-driven test that every CLI verb in `cli.go` has an MCP tool and an apiclient method. First check: diff `tools/list` names against the verb switch.

**ARCH-08 (Low): the AI default and the degraded mode are easy to miss in a hosted deploy.**
- `server/internal/config/config.go:76`: `getenvFirst("claude-sonnet-4-20250514", "INSIDEOUT_LLM_MODEL", ...)` is a dated snapshot default; startup only warns when the model is missing. With no key, `main.go` logs "using offline template-reply coach" at Info only, and `/auth/config` exposes `mailEnabled` but no AI flag, so users of a mis-keyed deploy get canned replies with no signal.
- Fix: expose `aiEnabled` in `/auth/config` and log the template fallback at Warn.

## Lead verification

| Lead | Result | Evidence |
|---|---|---|
| L1 | Extended | Not re-filed. Siblings: ARCH-04 (`roadmap.go:70`, `roadmap_progress.go:111`). Fixed-date tests elsewhere inject `now`, so no second failing test exists today. |
| L3 | Not re-run | I ran only targeted probes in a scratch copy, so I rely on the coordinator's 25 ok / 1 FAIL result. |
| L12 | Extended | Files are within budget, but `docs/architecture/backend.md` describes the `linebudget` guard as "~350-line" while the enforced budget is 400 (see lead). Counted in ARCH-06. |
| L16 | Confirmed | `server/Dockerfile` builds only `./cmd/insideout`; `insideout-mcp` is not in the image, which is fine. Builds on `golang:1.26-alpine`, runs as nonroot. |
| L13 | Partly relevant | Links are not my lens, but the architecture prose drift in ARCH-06 is a separate class the link check cannot find. |
| L5, L7-L11, L14, L15, L17, L18 | Out of lens | Not examined. |

## Recommendations

1. Fix client resolution once: `clientconfig.Resolve`, used by CLI, every `--api` flag and the MCP binary (ARCH-01).
2. Bind the stored token to its base and stop swallowing config-load errors (ARCH-02).
3. Add a `Server.now` clock and a guard test against raw `time.Now()` in `internal/api` view builders (ARCH-04).
4. Make `DefaultBase` an ldflags-set var tied to `scripts/default-api.sh` (ARCH-05).
5. Move the singleton guard to an explicit `Server.Start()` and write the single-replica contract into `backend.md` (ARCH-03).
6. Add `docs/architecture/clients.md` and correct the three false lines in `backend.md` (ARCH-06).
7. Add a CLI/MCP/apiclient parity table test (ARCH-07).
8. Surface `aiEnabled` in `/auth/config` (ARCH-08).

## What this role would do first

Fix ARCH-01 and ARCH-02 together in one change: a single `Resolve` function, then delete the three other implementations. It is the only place where the shipped product sends a user's credentials to a host they did not choose, it is small, and it removes the reason `scripts/mcp-config.sh` has to exist as a shadow resolver. The one structural decision the owner has to make: is this product permanently one replica on one hosted instance, or will it be deployed by others? If one replica, say so in the architecture docs, keep the in-memory state, and keep the hosted URL baked in. If others will deploy it, the base URL must become a build-time variable and the throttle, cooldown, presence and progress state must move into Postgres.
