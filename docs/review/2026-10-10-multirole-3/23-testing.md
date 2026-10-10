# 23 · Testing

**Score:** **8.2 / 10** · 8 findings (2 High+)

**Bottom line:** The 814+421 suite is the strongest part of this repo: real-PostgreSQL DB-gated tests with a CI skip-gate, an AST-derived route-auth scan, a real-Chrome web lane, and a remediation wave that landed red tests first — I re-ran the full server suite against the scratch cluster (25 packages ok, 0 FAIL, 1 named SKIP) and the full Flutter VM suite (421/421 in 34s). What it cannot catch is concentrated at the boundaries: every CI path migrates a fresh empty database while four pending migrations will first meet the hosted populated DB; the invitation-only funnel's email flows (invite/magic-link/reset) have no positive round-trip test anywhere and have never touched the real mail provider; the nightly smoke boots its own throwaway stack and never probes the hosted deployment or the nginx+SPA deploy shape; and client↔server contract drift is invisible because both sides test against self-authored fakes.

---

# Review 3 — Testing

**Scope.** CI workflows (ci/release/smoke/ubuntu-install), the server test tree (149 `_test.go` files across 25 packages, DB-gated on real PostgreSQL), the Flutter suite (114 files, VM + Chrome lanes), scripts/tests, and the just-added guards (line budget, feature wiring scan, route-auth scan, the three nginx checks, breaker pins). Repo tip verified: HEAD `d2d3e20` is one docs-only commit past the review base `754c502` (two docs files, +31 lines — source identical).

**What I ran myself.** From `server/` with `source /tmp/insideout-gap/env-io_int_final.sh` (never printed): `go test ./... -count=1 -timeout 20m` — **all 25 packages ok, 0 FAIL, 1 SKIP** (`TestReadSecretTerminal`, the CI-named exception at ci.yml:91); a second `-v` run confirmed 0 unexpected skips. Slowest single test: `TestParityFixture` at 0.94s — there is no slow-test problem in this suite. From `client/`: `flutter test --reporter compact` — **+421, all passed, 00:34**. I did not run the `-race` lane, the Chrome lane, the Android APK build, the Python env suites, or `smoke.sh`; those are coordinator-confirmed green today (lead 2) and are reported here as their confirmation, not my run.

## Strengths the guards already prove

- **The route-auth scan is the best guard in the tree** (`server/internal/api/routes_require_auth_test.go`): it AST-discovers every `HandleFunc`/`Handle` pattern, probes each with no credentials expecting 401, fails on non-literal patterns that would escape the scan, and carries an anti-vacuity floor — `if checked < 40 {` (line 151) — so the guard cannot silently stop guarding.
- **A skip is a red build** (ci.yml:85-96): unexpected `--- SKIP:` fails the job and uploads the `-v` log, killing the classic "integration tests quietly not running" failure.
- **The web lane answers the VM-vs-web lesson structurally** (ci.yml:173 + `client/test/web/sse_transport_web_test.dart`): the real fetch/ReadableStream transport runs in Chrome against a live cross-origin Dart origin, not a compile-only check.
- **Breaker semantics pinned on real rows** (`server/internal/agent/breaker_pins_test.go`): an open-circuit refusal is asserted to spend zero run rows (lines 165-172).
- **The wave landed red-first**: fix `9f608bb` (model saw the user turn twice) is paired with test `8a70856`; `coach_history_test.go:88` — "the latest user turn must appear exactly once, last" — is exactly the payload-assertion shape that had been missing.

## Findings

| ID | Sev | Bucket | Where | Gap |
| --- | --- | --- | --- | --- |
| TEST-01 | High | FIX | `.github/workflows/ci.yml:73` | No migration-upgrade lane; fresh-DB-only everywhere |
| TEST-02 | High | FIX | `server/internal/api/auth_email_flows_test.go:14` | Email auth funnel has no positive round-trip test |
| TEST-03 | Med | HARDEN | `.github/workflows/smoke.yml:71` | Smoke never touches the hosted stack or deploy shape |
| TEST-04 | Med | FIX | `client/test/feature_wiring_test.dart:5` | Client↔server contract drift untested |
| TEST-05 | Med | HARDEN | `server/scripts/smoke.sh:59` | Provider contracts (LLM, mail) exercised only via own fakes |
| TEST-06 | Med | HARDEN | `server/internal/agent/breaker_pins_test.go:17` | Breaker pins coupled across parallel package binaries |
| TEST-07 | Low | IMPROVE | `client/test/line_budget_test.dart:8` | Line-budget guard exists only for Dart |
| TEST-08 | Low | ADD | `PRODUCT.md:218` | Unimplemented loop items (Branch/Merge, Decision Log) unflagged |

**TEST-01 — the deploy's first migration run is also its first test run.** Both gates apply migrations to an empty database: ci.yml:72-73 (`run: go run ./cmd/insideout migrate`) and smoke.yml:59-61, each on a just-booted postgres:18.6. Four migrations are pending against the hosted DB (lead 10): `20261010100000_ai_runs_reaper_index.sql`, `20261010110000_rotate_short_invite_codes.sql`, `20261010120000_ideas_require_membership.sql`, `20261010130000_authored_rows_require_membership.sql` — the last is DROP/CREATE POLICY on live tables. Fix: a job that replays migrations only up to `20261008*`, seeds rows (including a legacy invite code), migrates to HEAD, runs smoke.sh.

**TEST-02 — the only front door real users have is the one thing never tested open.** Signup is invitation-only (lead 15), yet the positive path has no test at any layer: `auth_email_flows_test.go:14-15` — "builds a Server whose store is nil: these tests only exercise request validation, which runs before any store access"; the sender is built inside `NewServer` (`server.go:62`) so route tests cannot capture a send; the dev-mode body-log seam exists (`mail.go:44`, "only when NewSender got devBody") but no test extracts a token and consumes it; smoke.sh's S0–S5 skip these routes entirely. The fix is cheap because the seam already exists: capture the devBody log, parse the link, POST `/magic-link/consume`, assert the session.

**TEST-03 — what smoke does not smoke.** The nightly suite that would fail first if the hosted deploy went out boots its *own* server on a *throwaway* database (smoke.yml:70-71, smoke.sh:57-64). Not smoked: the hosted URL, nginx (BUG-027 is undeployed), the SPA/client image, MCP, export, admin, email flows, and the real-key coach path (smoke.sh:59-60 deliberately forces "the offline template coach/planner"). A stale-image or broken-upstream deploy leaves nightly green. smoke.sh:20 already supports `SMOKE_BASE` against a running server — wiring a hosted/compose mode is small.

**TEST-04/05 — the fake-contract lesson is still structural.** `feature_wiring_test.dart:5-7` is honest about itself ("a LINT-TIER BACKSTOP only… they prove presence, not behavior"), but the behavioral suites inherit the gap: Flutter widget tests fake `ApiClient` wholesale, `requests_test.dart` pins strings the client itself authors, and the Go `apiclient` tests run against hand-rolled handlers (`apiclient_test.go:10`, `func newTestServer(t *testing.T, handler http.HandlerFunc)`). Nothing fails when the two sides drift. Same shape outward: `mail_test.go:31-33` points the sender at an httptest server — we assert what we send, never what the provider accepts; the LLM path has never seen a real provider (lead 13). This week's client bugs (deep-link dock `eb337af`, stale export preview `8e551c4`) shipped through a fully green 421.

**TEST-06 — the one documented flake seed.** No test in the suite exceeds 0.94s, so flake risk is not duration; it is state. `breaker_pins_test.go:17-20` documents that the api package's tests "reset the row mid-flight" in a parallel package binary and that the pins survive only because "Those resets only ever zero the counts — they never touch last_failure_at" — an invariant owned by a different package's tests. The `-race` lane that just landed (ci.yml:84) widens exactly these interleavings. De-share the row (`anthropic-test` id) before it bites.

**TEST-07/08.** The line-budget guard records its own lesson — "coach_panel.dart drifted to 354 lines within hours" (`line_budget_test.dart:7-8`) — but guards only `['lib', 'test']`; the server (max 344, lead 3) has no gate. And lead 14, verified: Product Branch/Merge, the Decision Log (PRODUCT.md:218, loop item 5 at 369-370), and scoped feedback invitations have zero implementation (grep across `server/` and `client/lib`: no matches) and therefore zero tests, while the status paragraph disclaims only the roles gap (PRODUCT.md:427) — the four audience views, by contrast, are all implemented (`audienceview.go:58`).

## The three tests that would have caught real bugs this week

1. **Route-parity suite, client builders vs the server's real route table.** Export `registeredRoutes()` from `routes_require_auth_test.go` as the source of truth and assert every request the Flutter builders and Go `apiclient` construct resolves to a registered route with the expected shape. Would have caught the deep-link dock (`eb337af`), the stale export preview (`8e551c4`), and every future "client greps green while the server moved" bug — the largest single class of this week's fixes.
2. **Positive email round trips via the devBody seam.** Request magic-link/invite/reset → parse the token from the captured dev-mode mail body → consume → assert the session. Would have caught any break in link minting, token-kind wiring, or link formatting in the only funnel a real user can enter through — currently proven nowhere.
3. **Compose-up smoke behind nginx with the SPA.** One lane that boots nginx+client+server+postgres and drives the five surfaces through the public port. Would have caught all three config-plane bugs fixed this week — BUG-027 DNS re-resolve, the 8 MiB webhook cap (`542a9c2`), the SPA security headers (`9ac7ece`) — none of which any unit test can see, and all of which currently sit undeployed.

**Verdict: 8.2.** For a product at this stage I would ship on this suite — real-PostgreSQL gating, skip-gates, a Chrome lane, and test-first fixes are disciplines most repos never reach. The 1.8 it does not yet own lives at the boundaries: the migration upgrade path, the email funnel, the deployed stack, and the two provider contracts — all currently scheduled to be tested for the first time on deploy day.

## What this role would do first

Seed a scratch database at the pre-2026-10-08 migrations, insert one row per core table (including a legacy invite code), then run `go run ./cmd/insideout migrate` to HEAD — that single check answers whether the four pending migrations are deploy-safe before anything else is built.
