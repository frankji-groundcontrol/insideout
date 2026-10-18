# 21 · Code quality

**Lens:** dimension  
**Score:** **8 / 10**  
**Bottom line:** Disciplined, small-file, well-commented code base with clean vet, gofmt -s and race build; the defects are medium design debt: a nondeterministic agentcontext helper, a divergent duplicate client-IP extractor, unescaped apiclient URLs, no context in apiclient, swallowed bookkeeping errors, and two mega-functions hiding under the file budget. No High or Critical found.

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

---

## Scope

Whole Go tree (`server/cmd`, `server/internal`, tests included), the Dart tree (`client/lib`, `client/test`), `scripts/*.sh|py` and `config/git-hooks`, read at main @ 9f0b1c7 in `/private/tmp/insideout-review4`. Mechanical checks run in a sandboxed env (nothing in the worktree edited): `go vet ./...` clean, `gofmt -s -l .` clean, `go build -race ./...` clean. `golangci-lint 2.14` with `--no-config` and extra linters (errcheck, unused, errorlint, gocritic, unparam, noctx, unconvert) gave 41 issues on non-test code, all minor. I also wrote a throwaway AST function-length scan and four tiny repro tests in a scratch copy of `server/` under `/tmp/rv4-scratch/code-quality/`. I did not start Postgres; no DB was needed. Shell was reviewed by hand (shellcheck is not installed).

## Scores

| Aspect | Score | Why |
|---|---|---|
| File size and layout | 9.0 | Largest non-test Go file is 348 lines and largest Dart file 349; no tracked source file is over budget. |
| Function size and cohesion | 6.5 | Files are small but two functions are huge: `addProductTools` (298 lines) and `Coach.HandleMessage` (218). |
| Error handling | 7.0 | Sentinel errors and `errors.Is` are used consistently. Store errors are mostly unwrapped, and terminal-state bookkeeping discards errors. |
| Duplication and dead code | 7.0 | Few copies, but some diverge: a "mirrors on purpose" copy of `clientIP` and a nondeterministic twin of `blankSections`. |
| Context and goroutine lifecycle | 8.0 | Server goroutines are bounded (reaper, watchdog, base context). `apiclient` ignores context entirely. |
| Shell and scripts | 8.5 | Every executable script has `set -u`, mostly with `-e` and `pipefail`; the three `set -u`-only `sh` scripts are deliberate exit-code design. Nothing unquoted that bites. |
| Dart | 7.5 | Analyzer clean; no `print`, no `TODO`. Raw `Map<String,dynamic>` pages and several silent `catch (_)`. |
| **Overall** | **8.0** | Disciplined, well-commented code; defects are medium-size design debt, not breakage. |

## What is good

- Modularity is real. Every Go file is under 350 lines. `server/internal/linebudget/line_budget_test.go:27` ratchets it with an empty allowlist.
- Goroutines are bounded. `cmd/insideout/main.go:159` derives every request context from a cancellable `baseCtx`. `runReaper` (`main.go:238-258`) exits on `ctx.Done()`. `idleWatchdog` (`internal/agent/taxonomy.go:91`) has a `done` channel.
- The coach turn lifecycle is the best code in the repo. `internal/agent/coach.go:185-198` runs a terminal-state guarantee on a detached context, with panic recovery that re-panics so the recover middleware still sees it.
- Logging is `slog` everywhere in `internal/`. Zero `fmt.Print` or `log.Print` outside the CLI.
- Error classification uses sentinels and `errors.Is`/`errors.As` (`retryAfterFrom`, `ErrContextLength`, `ErrUpstreamStall`). I found no string matching on `err.Error()`.
- Comments explain why, and only one code comment points at a missing doc: `client/lib/theme/cobalt.dart:73` cites the nonexistent `docs/plans/2026-10-06-frontend-cobalt-neo.md`.
- Clean time handling in Dart tests: `client/test/project_updates_display_test.dart:12` builds fixtures relative to now (`_ago`), so the client has no L1-style bomb.

## Findings (worst first)

All are Medium or Low. I found no High or Critical in this lens.

**QUAL-01 (Medium). `agentcontext.blankSections` is nondeterministic and disagrees with `readiness.blankSections`.**
`server/internal/agentcontext/agentcontext.go:177`: `for k, v := range sections {` feeds `openQuestions` (line 70). Confirmed by running it: the same eight-blank PRD yielded 8 distinct orders in 20 calls. A whitespace-only section is not "open" here, but `internal/readiness/readiness.go:113` uses `trimLen(v) == 0`. The test (`agentcontext_test.go:80`) accepts any of six keys at `open[0]`, so it cannot catch the ordering. Fix: build the list from a fixed ordered slice (the readiness `allSections()` order) with the same trim rule, ideally by calling one shared helper.

**QUAL-02 (Medium). A second client-IP extractor diverges from the first, behind a comment that says it mirrors it.**
`server/internal/api/auth_cooldown.go:79`: "It mirrors the throttle middleware's IP extraction on purpose". `cooldownClientIP` takes only the last XFF part; `clientIP` (`auth_throttle.go:294`) walks back past empty parts. Confirmed by running: `X-Forwarded-For: "9.9.9.9, "` gives `clientIP`=9.9.9.9 and `cooldownClientIP`=the proxy RemoteAddr, so every such caller shares one cooldown bucket. Only `clientIP` has edge-case tests (`auth_client_ip_test.go`). Fix: delete `cooldownClientIP` and call `clientIP` at `auth_email_flows.go:35,112,200`.

**QUAL-03 (Medium). `apiclient` builds paths and queries by string concatenation, with no escaping.**
`server/internal/apiclient/prd.go:83`: `return c.get("/prds/" + prdID + "/view?audience=" + audience)`. 36 such concatenations, 0 `PathEscape`/`QueryEscape` calls. The MCP tools pass LLM-supplied strings straight in. Confirmed by running: `audience="decision&admin=1#frag"` arrived at the server as `/prds/p1/view?audience=decision&admin=1`. Fix: use `url.PathEscape` for path segments and `url.Values` for queries (as `prd.go:93` already does for export).

**QUAL-04 (Medium). `apiclient` has no context, and the MCP handlers discard theirs.**
`server/internal/apiclient/apiclient.go:87`: `req, err := http.NewRequest(method, c.base+path, reader)`; the same at lines 131 and `auth.go:96`, `roadmap.go:152`. `mcpserver/product_tools.go:21` receives `ctx` and never uses it. A cancelled MCP call or a CLI Ctrl-C does not abort the in-flight request; only the fixed 20 s client timeout (`apiclient.go:27`) bounds it. Fix: add `ctx context.Context` to `do`/`getBytes` (and thread it from the MCP handler and CLI root context); use `NewRequestWithContext`.

**QUAL-05 (Medium). Sibling of L1: two more view mappers hard-wire the wall clock, with no test.**
`server/internal/api/roadmap.go:70`: `v.Pressure = roadmaptime.Pressure(*n.Deadline, time.Now())`; likewise `roadmap_progress.go:111` (`roadmaptime.Assemble(in, time.Now())`). `roadmaptime` itself takes `now`, and its tests pin one, but I found no API-level test of `pressure`. A pinned-date test written against these would hit the same bomb as `projectResponse`. Not failing today. Fix: give the view builders a `now` argument (or a `Server.now` func, like the throttle's) when L1 is fixed, and add one API test per mapper.

**QUAL-06 (Medium). Terminal-state bookkeeping discards its errors, against the "guarantee" the comments promise.**
`server/internal/agent/coach.go:194`: `_ = c.store.MarkAIRunFailed(cleanupCtx, userID, run.ID, fmt.Sprintf("panic: %v", r))`. Same pattern at `coach.go:191,195,302`, `coach_fail.go:38,40`, `gate.go:89-97`, `critic.go:46`, and `critic.go:27` (`GetConversationForOwner` error returns with no log). On a DB blip a run stays `running` until the 10-minute reaper, with nothing in the logs. Fix: one `c.logErr(msg, err)` helper (or `slog.Warn` inline) at those ~10 sites.

**QUAL-07 (Medium). The file-size budget is satisfied while single functions are the real monoliths.**
`server/internal/mcpserver/product_tools.go:18`: `func addProductTools(s *server.MCPServer, api string) {` runs 298 lines of inline closures; `agent/coach.go:101` `HandleMessage` is 218 lines of interleaved SSE, locking, persistence and retry. Next are `runLocalSetup` (143), `runVersionCommands` (117), `parseAnthropicStream` (115). CLAUDE.md says ~350, linebudget enforces 400 (noted once, as asked), and neither measures functions. Fix: split `addProductTools` into per-resource `addPrdTools/addRoadmapTools/...`, and extract `HandleMessage`'s setup and success-persist phases; consider a 120-line function ratchet in `linebudget`.

**QUAL-08 (Low). UUIDs are stringified at the API/Coach seam and re-parsed, with an unreachable 500.**
`server/internal/api/conversations.go:209`: `s.coach.HandleMessage(w, r, cid.String(), userID.String(), conv.WorkspaceID.String(), req.Content)`, then `agent/coach.go:104-110` parses them back and answers 500 on failure. Fix: change `api.Coach` (`server.go:23`) to take `uuid.UUID` and delete the parse block.

**QUAL-09 (Low). Dead and duplicate helpers.**
`server/internal/api/cors.go:43`: `func splitCORSOrigins(raw string) []string {` is unused (the live one is `config/config.go:172`). `agent/critic.go:197` `hasOpenBlockingFindings` is called only from tests. `tokenFromLink` is copied in `cmd/insideout/cli_auth.go:205` and `setup/flow/flow.go:188` (the comment admits "Mirrors"; cmd already imports the package). `firstLine` exists twice, and the `github_webhook_evidence.go:148` one slices `s[:120]` bytes and can cut a multibyte rune. Fix: delete the dead two, export one `tokenFromLink`, use a rune-safe `firstLine`.

**QUAL-10 (Low). The real-DB test preamble is hand-copied in about 31 test files.**
`server/internal/api/roadmap_build_status_test.go:27`: `url := os.Getenv("DATABASE_URL")` followed by `t.Skip(...)`, `store.Open`, `t.Cleanup`. Each copy re-implements user/workspace fixtures. The `t.Skip` also means a bare `go test ./...` quietly skips most of the integration suite (CI's skip-count gate covers that, local runs don't). Fix: one `internal/testdb` helper (`Open(t)`, `NewUser(t, st)`) and migrate files opportunistically.

**QUAL-11 (Low). Error context and handler boilerplate.**
`internal/store` has 263 bare `return err` against 22 `%w` wraps, so log lines carry a bare pgx message with no operation name. `internal/api` has 69 `ErrNotFound` and 111 `StatusInternalServerError` ladders, and 67 handlers do `userID, _ := UserID(r.Context())` (`roadmap.go:90`), safe only because `routes_require_auth_test.go` guards the middleware. Fix: wrap at the store boundary (`fmt.Errorf("store.X: %w", err)`) and add an `authed(func(w,r,uid))` wrapper plus one `writeStoreError` helper.

**QUAL-12 (Low). Dart: `ProjectPage` works on raw maps, with silent catches.**
`client/lib/features/project/project_page.dart:87`: `Future<Map<String, dynamic>> _load() async {` although a typed `Project` exists (`api/models_project.dart:1`); `_load` also mutates `repo.text` and focus as a side effect. 214 `as` casts across `lib/`, and silent `catch (_) {}` at `ideas_page.dart:62`, `roadmap_page.dart:85`, `prd_page.dart:110`. Fix: return a typed `ProjectDetail`, move the side effects into the `then`, and at least `debugPrint` in the catches.

## Lead verification

| Lead | Result | Evidence |
|---|---|---|
| L1 | Extended | Only one failing pinned-clock test (coordinator's). Siblings: uninjected clock at `api/roadmap.go:70`, `roadmap_progress.go:111` (QUAL-05). Client is safe: tests use relative `_ago`; hard-dated fixtures flow through `timeAgo` (14-day cutoff, `chrome/time_ago.dart:15`) but no assertion depends on them. |
| L3 | Partly re-verified | I did not rerun the suite (no DB). `go vet`, `gofmt -s -l`, `go build -race ./...` are clean. |
| L12 | Confirmed, extended | Largest non-test Go file 348, largest Dart 349, no tracked source file over budget. Extended: function length is unmeasured (QUAL-07). |
| L13 | Partly confirmed | In code, one dead doc path (`client/lib/theme/cobalt.dart:73`); all other docs paths cited from source resolve. |

## Recommendations

1. Replace `cooldownClientIP` with `clientIP` and add the `"9.9.9.9, "` case to the shared test (QUAL-02).
2. Escape every `apiclient` path and query value, then add a hostile-input test for `PrdView` and `AgentContext` (QUAL-03).
3. Thread `context.Context` through `apiclient` and the MCP handlers (QUAL-04).
4. Make `agentcontext.blankSections` ordered and trim-aware, and tighten its test to an exact expected list (QUAL-01).
5. Log every discarded store error in the coach, gate and critic cleanup paths (QUAL-06).
6. Inject a clock into the roadmap view mappers when fixing L1 (QUAL-05).
7. Split `addProductTools` and `HandleMessage`; add a function-length ratchet beside the file-line one (QUAL-07).
8. Add a shared `testdb` helper and an `authed` handler wrapper; convert files as they are touched (QUAL-10, QUAL-11).

## What this role would do first

Land the two small correctness-flavored fixes (QUAL-02 and QUAL-03), because both are a few lines and both are reachable from untrusted input (a proxy header and MCP tool arguments). Then do the two mechanical cleanups in separate commits. Commit one: delete the three dead helpers (`splitCORSOrigins`, `stopSingletonGuard`'s test-only exposure reworded or moved to a `_test` file, `hasOpenBlockingFindings`), drop the duplicate `tokenFromLink`, apply `unconvert`, the `S1002`/`QF1001` staticcheck rewrites and the six `errorlint` `%w`/`errors.Is` fixes. Commit two: the `apiclient` escaping pass, which is a mechanical sweep of 36 call sites. After that, add a `.golangci.yml` that turns on `unused`, `errorlint`, `errcheck` (excluding `Close`/`Rollback`) and `noctx` so these do not regrow.
