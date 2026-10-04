# 23 · Testing

**Lens:** test-suite honesty, coverage of documented paths, mocks-vs-real-DB policy, CI/enforcement
**Score:** **5.5 / 10**
**Bottom line:** The test craft itself is unusually disciplined — honest `t.Skip` gates, zero mock libraries, real `httptest` servers, captured real SSE streams — but the verification system around it is hollow: one suite (scripts env catalog) has been failing for six weeks with exit code 1 and nobody noticed, 43 of 188 Go test results skip to a green `ok`, the API layer holding the coach SSE endpoint has 4 test files for 30 source files, and with no CI (L1) and git hooks that only check docs-recorded, nothing on earth runs any of these tests automatically. The tests that exist are good; the promise they make is unkept.

Reviewed against main@877ad03; High/Critical findings re-checked by an independent skeptic — verdicts in 01-findings-register.md.

---

## Scope

Every server `*_test.go` (55 files, all packages under `server/internal/...` and `server/cmd/insideout`), all 19 files in `client/test/`, `scripts/test_env_catalog.py`, `scripts/test_env_writes.py`, `scripts/env_testlib.py`, `server/scripts/smoke.sh`, git hooks under `config/git-hooks/`, and CI configuration (absent — L1). Commands run on this tree (all re-run with `-count=1` / fresh where applicable):

- `cd server && go test ./...` → all packages `ok`; verbose tally **145 PASS, 43 SKIP, 0 FAIL**
- `cd server && go vet ./...` → clean, exit 0
- `cd client && flutter test` → **All 57 tests passed!** (0 skips)
- `cd client && flutter analyze` → 20 info-level issues, **exit 1** (infos are fatal-by-default to the CLI)
- `cd scripts && python3 test_env_catalog.py` → **46 passed, 1 failed, exit 1**
- `cd scripts && python3 test_env_writes.py` → 37 passed, 0 failed, exit 0

## What is good

- **The no-mocks rule is real and legible.** `server/go.mod` has no testify-mock/gomock entries; `server/internal/store/authz_test.go:12-17` states the policy in a doc comment ("RLS policies can't be verified against a mock … Run with: DATABASE_URL=... go test"), and the skip at line 22 — `t.Skip("DATABASE_URL not set, skipping integration test")` — is explicit and honest. Every one of the 13 store test files plus the agent stage-gate tests funnels through this one gate; I found no test that silently substitutes an in-memory store.
- **LLM/SSE parsing is tested against reality.** `server/internal/agent/anthropic_test.go:18-40` embeds a real captured SSE transcript ("a real SSE response captured from a live Anthropic-compatible endpoint … the exact scenario that broke the old langchaingo-based implementation") and `anthropic_test.go:281-285` spins a stalling `httptest.Server` to exercise hang timeout "over real HTTP (no mocks)". This is exactly how provider clients should be tested.
- **The client suite asserts behavior, not shapes.** `client/test/sse_test.dart:47-57` feeds split frames through `parseSseBuffer` and asserts the partial-tail contract; `client/test/session_test.dart:75-103` injects `fetchMe`/`refreshSession` closures and counts calls (2 me-calls after refresh) — dependency injection without a mocking framework. `client/test/requests_test.dart` pins method/path/body for every REST verb.
- **Security invariants have dedicated integration tests.** `server/internal/store/roles_test.go` asserts the runtime pool user is `insideout_app`, `insideout_owner` is not superuser but has `BYPASSRLS`, table/function ownership, and that `workspace_memberships` has `FORCE ROW LEVEL SECURITY` — the schema-hygiene contract behind the "never write outside the `insideout` schema" rule.
- **`server/scripts/smoke.sh` is a genuine live check** over all five product surfaces with rerun-safe naming, and the env tests are logic tests over temp scaffolds whose docstring (`test_env_catalog.py:1-14`) honestly states what they cannot see (the curses rendering layer).

## Findings

**TEST-01 · High · scripts env-catalog suite is red and has been since 2026-08-22.** `python3 scripts/test_env_catalog.py` prints `FAIL  every declared variable has a consumer (orphans: ['INSIDEOUT_GH_CLIENT_ID', 'INSIDEOUT_GH_CLIENT_SECRET'])` / `46 passed, 1 failed` and exits 1. The orphan declarations were added to `.env.example:131-132` by commit `4f228f0` (2026-08-22, "follow-ons — coach gap weaving…") as `#INSIDEOUT_GH_CLIENT_ID=` / `#INSIDEOUT_GH_CLIENT_SECRET=`, and no non-test Go file or `docker-compose.yml` consumes them (`config.go:72-75` reads only `INSIDEOUT_GH_WEBHOOK_SECRET/APP_ID/PRIVATE_KEY[_FILE]`). Failure scenario: any operator following the skeleton exports two dead variables believing GitHub App client auth is wired, or a needed consumer was dropped in the same commit; either way the repo's own contract test says the env contract is broken and nobody has run it in ~6 weeks (changelogs record "47 passed" last on 2026-08-19). Fix: decide per-variable — delete the two dead lines from `.env.example`, or implement the missing consumer — then re-run the suite to green. diff_size: 2 lines (or a small consumer wiring).

**TEST-02 · High · The entire DB-integration and live-smoke layer never executes anywhere.** `go test ./...` exits 0 while 43 of 188 results are `--- SKIP` (37 store, 6 agent stage-gate/ledger); the only five-surface live check, `server/scripts/smoke.sh` (S1 coach SSE … S5 cross-tenant authz), has no recorded run after 2026-07-30 in any changelog or plan; `.github/workflows/` is absent and `config/git-hooks/pre-commit` invokes only `check-docs-recorded.sh`. Failure scenario: an RLS policy regression in the next migration PR (e.g. `workspaces_select` join-code carve-out loosened) — `go test ./...` stays green, review sees green, and the authz suite that would catch it never runs; this has already happened in miniature with TEST-01. The individual skips are honest (`authz_test.go:22`); the system around them is not. Fix: add one CI job (or a nightly) that boots compose postgres, sets `DATABASE_URL`, runs `go test ./internal/store/ ./internal/agent/`, and runs `smoke.sh`; make "integration suite skipped" a hard fail in that job. diff_size: one workflow file (~60 lines).

**TEST-03 · Medium · `feature_wiring_test.dart` asserts string presence, not wiring.** `client/test/feature_wiring_test.dart:16` — `expect(settings.contains('updateWorkspace'), isTrue);` (and 30 siblings, lines 16-51, all `source.contains(...)` over page sources). A developer can satisfy this test by writing the symbol in a comment while the actual call site is deleted; conversely a genuine behavior break (wrong arguments, no refresh after mutation) passes as long as the substring survives. Failure scenario: `ideas_page.dart` stops calling `updateIdea` but keeps a `// updateWorkspace` style TODO — wiring test stays green, the inbox edit path is dead in production. Fix: keep this file only as a lint-tier guard and add per-surface widget tests with the `_FakeApi` pattern already used in `versions_page_test.dart` that assert the mutation method is invoked with the right arguments when the user taps save. diff_size: ~150 lines of new widget tests.

**TEST-04 · Medium · The REST layer has 4 test files for 30 source files; the coach SSE endpoint has zero HTTP-level coverage.** `server/internal/api/` contains 30 non-test `.go` files and exactly four `_test.go` files (`auth_email_flows`, `auth_session`, `cors`, `project_updates`); `conversations.go:15` registers `mux.HandleFunc("POST /api/v1/conversations/{id}/messages", s.requireAuth(s.handleSendConversationMessage))` — the product's flagship streaming feature — with no test driving its handler, and the only safety net is the manual smoke script last evidenced 6+ weeks before the Cobalt PR (TEST-02). AGENTS.md's own practice doc (`docs/practices/2026-07-21-live-exercise-streaming-endpoints.md`) exists because two SSE bugs (BUG-009/010) were "independent, silent, and only surfaced live" — yet nothing repeatable guards the middleware chain against another `Flusher`-swallowing regression. Fix: add an `httptest` test that POSTs a message against a stubbed coach and asserts incremental flushed frames (`message_start` → `delta` → `message_end`), and wire `smoke.sh` into the release checklist. direction + first_check: confirm whether `handleSendConversationMessage`'s coach dependency is injectable — `grep -n "coach" server/internal/api/server.go | head`. diff_size: unknown.

**TEST-05 · Medium · A branch in `agentcontext_test.go` can never fail.** `server/internal/agentcontext/agentcontext_test.go:42-47`:

```go
open := out["openQuestions"].([]string)
if len(open) == 0 || open[0] != "risks" && open[0] != "users" && open[0] != "nonGoals" && open[0] != "constraints" && open[0] != "requirements" && open[0] != "stories" {
    // risks is blank; any blank key is a valid open question entry
    t.Logf("open questions: %v", open)
}
```

If `openQuestions` comes back empty — the actual failure mode for "risks is blank" — the test logs and passes. Failure scenario: a regression in `Assemble` drops blank-section detection; this test, the only one covering brainstorming open-questions, stays green. Fix: replace the `t.Logf` branch with `t.Errorf("open questions = %v, want a blank-section entry", open)` (an empty list must fail). diff_size: 2 lines.

**TEST-06 · Low · Deleted Nuxt remains the named contract partner in four client test files.** `client/test/l10n_test.dart:7` — `test('default locale matches Nuxt zh-CN', () {`; same pattern at `appearance_test.dart:5` ("starts on Nuxt default locale"), `api_errors_test.dart:21` ("coach backoff matches Nuxt APP_THROTTLE…"), and six of the ten tests in `requests_test.dart` ("… match Nuxt"). The app was deleted 2026-08-18; these names now describe parity with a codebase that does not exist, so a future contract change will read as "breaks parity with nothing". Failure scenario: none immediate — this is misleading-specification debt that bites the next editor of these tests. Fix: rename to state the contract itself (e.g. "default locale is zh-CN", "requests match the Go API routes"). diff_size: ~14 renamed strings.

**TEST-07 · Low · The two source-scan tests are cwd-fragile.** `client/test/leftover_scan_test.dart:8-11` and `feature_wiring_test.dart:7-14` read `'lib/...'` relative to the process cwd. Run `flutter test` from anywhere but `client/` and both die with `FileSystemException` — a false red that confuses anyone running the suite from the repo root. Failure scenario: a contributor runs `flutter test client/` from the root, gets two failures unrelated to any change, and learns to distrust the suite. Fix: resolve paths from `Platform.script`/`Directory.current` with an explicit assertion, or document `cd client` in the test header. diff_size: 6 lines.

**TEST-08 · Low · The scripts suites have no declared runner.** There is no `scripts/requirements*.txt`, `pytest` is not installed in the system Python (`No module named pytest`), and `scripts/dev.sh` never invokes either test file — they run ad-hoc as `python3 scripts/test_env_*.py`. Failure scenario: a fresh machine following `docs/SETENV.md` skips them silently; TEST-01's six-week red run is the demonstrated consequence. Fix: add a `scripts/run-tests.sh` (or a make target) that runs both files and is called from the same job that fixes TEST-02. diff_size: ~15 lines.

## Lead verification

| Lead | Verdict | Evidence |
|---|---|---|
| L1 (no CI) | Confirmed | `.github/workflows/` absent; `config/git-hooks/pre-commit` runs only `check-docs-recorded.sh`; nothing executes any test on commit/merge |
| L6 (stale Nuxt refs incl. `client/test/api_errors_test,appearance_test.dart`) | Confirmed, and wider | plus `l10n_test.dart:7`, `requests_test.dart` (6 of 10 test names) — see TEST-06 |
| L12 (`go build/vet/test` clean, store skips without DATABASE_URL) | Confirmed | re-ran all three: `go vet` exit 0; `go test -count=1 ./...` all `ok`; skips are explicit `t.Skip("DATABASE_URL not set, skipping integration test")` with the run command in the doc comment — the gate itself is honest |
| L13 (flutter analyze + tests claimed green) | Partially confirmed | `flutter test`: 57/57 pass, 0 skips (board said 54 — suite grew). `flutter analyze` exits **1** on 20 info-level lints, so "analyze green" is only true with `--no-fatal-infos` |
| L15 (no key-shaped secrets in tests) | Confirmed | test fixtures use obvious placeholders (`access-from-login`, `tok`, `ada@example.com`, `*.test.local`); nothing key-shaped |

## Recommendations

1. Add one minimal CI job now — `go build/vet/test ./...`, `flutter analyze --no-fatal-infos` + `flutter test`, `python3 scripts/test_env_catalog.py && python3 scripts/test_env_writes.py` — and make integration skips fail a second, nightly DB job. This single change would have caught TEST-01 six weeks ago.
2. Resolve TEST-01 today: delete `INSIDEOUT_GH_CLIENT_ID`/`INSIDEOUT_GH_CLIENT_SECRET` from `.env.example` or implement their consumer; re-run the catalog to "47 passed".
3. Schedule the DB-gated suites (`store`, `agent` gates/ledger) and `server/scripts/smoke.sh` on a nightly compose postgres; record runs in changelogs the way TestAuthz runs were recorded.
4. Promote `feature_wiring_test.dart` from substring checks to `_FakeApi`-backed widget tests that assert each mutation fires with correct arguments (keep the scan only as a lint-tier backstop).
5. Write the httptest SSE test for `POST /conversations/{id}/messages` that asserts incremental flushes end-to-end through the middleware chain, so AGENTS.md's SSE practice has automated teeth between manual smoke runs.
6. Convert the log-only branch in `agentcontext_test.go:42-47` to a real `t.Errorf`, and sweep the other server test files for sibling log-only conditions (none found, but pin it with a convention note).
7. Rename Nuxt-referencing test names to self-describing contracts (TEST-06) as the files are next touched; batch with any test-file edits to avoid churn.
8. Make the scan tests path-robust or cwd-asserting (TEST-07), and give the scripts tests a single entry point referenced from `docs/SETENV.md` (TEST-08).

## What this role would do first

Before anything else: stand up the CI job in Recommendation 1 and watch it fail — because it will, immediately (TEST-01 is red right now). That one change converts every other finding in this file from "discovered by a human audit" into "caught the day it happened," and it is the only fix that retroactively pays for the six weeks the env-catalog suite has been screaming into an empty room. Once the wire is live, fix TEST-01's two dead env declarations so the new job goes green, then schedule the nightly DB job so the 43 currently-skipped tests start guarding the RLS layer they were written for.
