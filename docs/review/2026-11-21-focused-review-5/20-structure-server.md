# 20 · Code structure: the Go server

**Lens:** modularity of packages, files and functions
**Score:** **6.3 / 10**
**Bottom line:** The Go server has a clean, acyclic package graph and every file sits under the enforced 400-line budget, but the budget is met by line counting rather than by modularity: one 315-line function registers 37 MCP tools, the coach turn is a 219-line function, the `cmd/insideout` main package carries three programs (server boot, product CLI, and 1,393 lines of local-setup logic that belong in `internal/setup`), and several files at 330-349 lines hold three or four responsibilities with their overflow pushed into 30-line satellite files. Nothing is over the rule except one uncommitted file; a good deal obeys the rule and is still not modular.

Read at `main` @ `4d186da` together with the uncommitted working tree. A skeptic re-checked every High in this file; the verdicts are in the table near the end and in the [findings register](./01-findings-register.md).

---

## Scope read

All of `server/cmd/insideout` (21 source files, 3,284 lines; 41 test files) and `server/internal` (29 packages), measured on main at 4d186da with the uncommitted working tree. I counted lines per file, measured every top-level function from its `func` line to its closing brace (1,009 functions), printed the internal import graph with `go list`, outlined the thirteen largest source files, scanned for functions with no production caller, and read `server/internal/linebudget/line_budget_test.go`, `docs/architecture/backend.md` and the modularity records in `docs/issues`. `go build ./...`, `go vet ./...` and `gofmt -l` are clean. I did not re-run the full test suite.

## What is good

- The package graph is acyclic and layered. `internal/api` does not import `internal/agent`; the seam is the `Coach` interface in `server/internal/api/server.go:22`. `internal/store` imports only `db` and `internal/prdcommit`.
- Structure is enforced by tests: `server/internal/linebudget/line_budget_test.go`, `server/internal/api/view_clock_guard_test.go`, and the client-method sweep in `server/internal/apiclient/escape_test.go`.
- Handlers are one resource per file, each with one `registerXRoutes`, listed together in `server/internal/api/server.go:124-136`.
- The roadmap store family (`server/internal/store/roadmap.go` with `roadmap_move.go`, `roadmap_evidence.go`, `roadmap_completion.go`, `roadmap_validation.go`, `roadmap_plan.go`) is split by concept, with tests on the same seams.
- `docs/architecture/backend.md` lists exactly the packages that exist.

## Over the rule versus under the rule but not modular

One file is over the written 350-line rule: `server/internal/setup/ui/ui.go` (355, uncommitted, mid-split). Everything else obeys it. Of the thirteen largest source files, six hold one responsibility (`api/roadmap.go`, `api/projects.go`, `api/auth.go`, `api/prds.go`, `store/roadmap.go`, `setup/ui/ui.go`) and seven hold several squeezed under the budget (`mcpserver/product_tools.go`, `agent/coach.go`, `agent/critic.go`, `agent/roadmap_planner.go`, `store/agent_messages.go`, `github/github.go`, `cmd/insideout/cli.go`).

## Findings, worst first

**STRS-01 (Medium). Functions are the monoliths.** `server/internal/mcpserver/product_tools.go:20`: `func addProductTools(s *server.MCPServer, api string) {` runs 315 lines; `server/internal/agent/coach.go:101` `HandleMessage` runs 219. This is review 4's QUAL-07, set aside by `41-remediation-contract.md:162`, with no row in `docs/issues`, and larger than recorded. Fix: per-resource `tools_*.go` files, `coach_turn.go` with open, run and close phases, and a 120-line function ratchet.

**STRS-02 (Medium). The MCP registration closure exists three times and one copy dropped the lock.** `server/internal/mcpserver/admin_tools.go:29`: `out, err := call(clientCtx(ctx, api), req)` with no `defer beginToolCall()()` before it, unlike `product_tools.go:24`. Fix: one helper in `register.go`; all four invite and signup tools in `admin_tools.go`.

**STRS-03 (Medium). `cmd/insideout` holds three programs and a second copy of setup.** `server/cmd/insideout/cli_setup_local_env.go:51`: `func readEnvValues(path string) map[string]string {` disagrees with `catalog.parseEnv` (`internal/setup/catalog/catalog.go:144`) on inline comments, and both read the same file in `cli_setup_local_boot.go:43` and `:127`. `tokenFromLink` and the terminal check are duplicated verbatim. Fix: `internal/setup/local` and `internal/setup/check`; export one parser.

**STRS-04 (Medium). `store/agent_messages.go` holds four responsibilities.** `server/internal/store/agent_messages.go:264`: `purges := []string{` followed by deletes on `sessions`, `auth_tokens` and `invitations`, beside messages, `ai_runs`, the rate limiter and the circuit breaker. Fix: `ai_runs.go`, `ai_ops.go`, `retention.go`.

**STRS-05 (Medium). Authorization predicates are scattered and one is duplicated.** `server/internal/store/project_repo.go:28`: `func requireProjectOwnerOrAdmin(...)` has the same body as `requirePlanWriter` at `memberships.go:236`. Fix: `store/authz.go` with one project-writer rule and a `requireNodeWriter` entry point.

**STRS-06 (Medium). Each verb is written three times with positional arguments.** `server/internal/apiclient/roadmap.go:82`: `func (c *Client) RoadmapUpdate(nodeID string, title, description, status, deadline, completionReason *string)`. The CLI clears a deadline with `clear`, the MCP tool with an empty string. Fix: request structs in `apiclient` filled by both surfaces.

**STRS-07 (Medium). Tests are named after a review round.** `server/internal/store/rv4_review_api_guards_test.go:14`: `func TestRvGuardsStrictModeRefusesARemoteDatabase(t *testing.T) {`. Twenty-eight `rv4_review_*` files (4,313 lines), nine `coverage_*`, nine `trust_*`. Fix: rename by behaviour and fold into the existing siblings.

**STRS-08 (Medium). The guard enforces 400, not 350, and exempts tests and functions.** `server/internal/linebudget/line_budget_test.go:27`: `const budget = 400`. `coach.go` went from 281 to 339 lines since its split was closed. Fix: 350 with an allowlist, a warning tier at 320.

**STRS-09 (Medium). `internal/agent` is five subsystems; the critic is cut mid-concept.** `server/internal/agent/critic.go:132`: `type CriticFinding struct {` with its parser in `prompts.go:178`, merge in `critic_merge.go`, skip path in `trust_critic.go`. Fix: leaf package `internal/llm` for the eight provider files (no import cycle), and three critic files by concern.

**STRS-10 (Low). `github.go`, `cli.go` and `roadmap_planner.go` each hold three concerns; CLI file names mislead.** `server/internal/github/github.go:220`: `func RepoIsPublic(...)` beside URL parsing and commit fetch. Fix: pure moves and renames listed in the finding.

**STRS-11 (Low). Handler error ladders (review 4 QUAL-11) have grown and have no record.** `server/internal/api/roadmap.go:251`: `if errors.Is(err, store.ErrNotFound) {` opens one of 69 ladders. Fix: `writeStoreError` and an `authed` wrapper.

**STRS-12 (Low). Three dead functions.** `server/internal/api/cors.go:43`: `func splitCORSOrigins(raw string) []string {`; also `store.UpdateConversationStage` and the test-only `hasOpenBlockingFindings`. Fix: delete.

## Lead verification

| Lead | Result | Deciding line |
| --- | --- | --- |
| 2 | Confirmed, extended | `memberships.go:236` `func requirePlanWriter(`; a twin exists at `project_repo.go:28` |
| 5 | Confirmed | `apiclient/roadmap.go:82` signature has no owner parameter |
| 7 | Not re-run | `go build` and `go vet` exit clean; the three red tests are structure guards |
| 8 | Confirmed | `grep -n 'reset_watch\|me_delete' docs/architecture/backend.md` returns nothing |
| 9 | Confirmed, refined | 12 non-test and 6 test Go files in the server sit at 330-349; `api` is 46 source and 102 test files, `store` 33 and 60, `agent` 31 and 61, `cmd/insideout` 21 and 41 |
| 14 | Confirmed, with a gap | QUAL-07 to QUAL-11 have no row under `docs/issues`; `41-remediation-contract.md:162` set them aside |

## Sub-package seams

`internal/llm` is free: the provider files import nothing internal. `internal/setup/local` is free: nothing under `internal/setup` imports `cmd`. `internal/api` and `internal/store` should stay single packages; their methods share unexported middleware and transaction helpers, and two thirds of their files are tests, so renaming tests does more than a package split would.

## Recommendations

1. Do the three splits the delegation feature depends on first: `store/authz.go`, request structs in `apiclient`, per-resource MCP tool files.
2. Move local setup out of `cmd/insideout` and delete the duplicate helpers.
3. Extract `internal/llm`.
4. Tighten the guard to 350 lines and add a function ratchet.
5. Rename round-named test files and write the naming rule down.
6. Open one dated record for the set-aside review-4 code-quality findings.

## What this role would do first

Add `server/internal/mcpserver/register.go` with one registration helper that always takes the tool-call lock, then split `addProductTools` into per-resource files. It removes the largest function in the server, repairs the drifted admin copy, and clears the ground where the node-owner argument and the subtree read tool have to be added.

## Ordered recommendations

1. Split by concept before the delegation feature starts: `store/authz.go` (STRS-05), request structs in apiclient (STRS-06) and per-resource MCP tool files with one registration helper (STRS-01, STRS-02) are the three places that feature has to touch.
2. Move the 1,393 lines of local-setup logic from `cmd/insideout/cli_setup*.go` into `internal/setup/local` and `internal/setup/check`, and delete the second dotenv parser, the second `tokenFromLink` and the second terminal check in the same change.
3. Extract the eight LLM provider files into a leaf package `internal/llm`; this is the only sub-package split the import graph makes free. Do not sub-package `internal/api` or `internal/store`: their 147 and 128 methods share unexported middleware and transaction helpers, and two thirds of their file count is tests.
4. Tighten the guard to match the rule: 350 lines with an explicit allowlist, a 120-line function ratchet, and a printed warning list at 320 lines.
5. Rename the 28 `rv4_review_*`, 9 `coverage_*` and 9 `trust_*` files by behaviour and record the naming rule in docs/practices so review 5 does not add a second generation.
6. Do the pure moves for agent_messages.go, github.go, cli.go, critic.go and roadmap_planner.go as one changelog entry of moves only, each checked with `go build ./... && go vet ./... && go test ./...`.
7. Open one dated record in docs/issues for the review-4 findings that the remediation contract set aside (QUAL-07 through QUAL-11), and name `store/reset_watch.go`, `api/me_delete.go`, the `ai_runs` and retention files in docs/architecture/backend.md.
