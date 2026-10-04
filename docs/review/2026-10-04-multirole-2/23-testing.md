# 23 · Testing

**Lens:** 
**Score:** **7.0 / 10**
**Bottom line:** Verification spine went from non-existent to real in one pass (postgres:18 with production two-role split, DB-gated suites execute, mutation-hard boundary tests, a genuine flush-through-the-stack SSE proof) — but the skip-guard only watches ./internal/store/..., the live smoke runs nowhere, no run uses -race, and web-only surfaces are compile-checked not behavior-tested.

Reviewed main @ 26db979; a skeptic re-checked every High finding from this review. [Correction] TEST-09 was downgraded High→Medium by the skeptic: the uncovered skips are real (6 agent, 2 conversations, 1 cooldown) but every guard reads the same job-level `DATABASE_URL`, so a green build with silent api/agent skips is not reachable via today's single shared cause — the gap is defense-in-depth for future divergent skip conditions.

---

This review was persisted findings-only; the section below is assembled verbatim from its structured findings (the prose wrapper was not retained). 10 findings as filed, worst first.

## Findings

### TEST-09 · High ·  **Skeptic: DOWNGRADED → Medium.**

**The skip-guard greps '--- SKIP' only for ./internal/store/...; the 9 DB-gated tests in ./internal/api (both SSE conversation tests + cooldown e2e) and ./internal/agent (6 stage-gate tests) can silently skip on a green build — TEST-02's failure mode one level up.**

- Where: `.github/workflows/ci.yml:64`
- Evidence: ci.yml:64 'skiplines=$(go test ./internal/store/... -count=1 -v …)' while skips also gate at conversations_test.go:30, auth_cooldown_test.go:183, agent/gates_test.go:19 (tally: 38 store / 6 agent / 3 api).
- Fix: Run the suite once verbosely with tee and fail on any '--- SKIP' line anywhere (every current skip is a DB gate); also removes the duplicate store re-run.
- Diff size: ~5 lines

### TEST-10 · Medium · 

**The five-surface live smoke (S1 coach SSE … S5 cross-tenant authz) still executes nowhere on any schedule — CI has no nightly, no smoke step; last recorded run remains 2026-07-30.**

- Where: `server/scripts/smoke.sh:1`
- Evidence: grep -rn 'smoke' .github/ returns nothing; no changelog records a smoke run after 2026-07-30.
- Fix: Scheduled nightly job: compose up postgres+server, run smoke.sh, upload transcript as artifact; keep off the PR path.
- Diff size: ~40 lines

### TEST-11 · Medium · 

**The web-only conditional-import surfaces — fetch/ReadableStream SSE loop and the export download path — are compile-checked by flutter build web but behavior-tested nowhere (no integration_test/, no browser run).**

- Where: `client/lib/api/sse_transport_web.dart:1`
- Evidence: sse_transport_web.dart:5-8 the documented XHR-buffering workaround loop; download_web.dart:5-15 anchor.click + revokeObjectURL untested; client/test/sse_test.dart:36 tests only the chunk consumer.
- Fix: One headless-chrome integration test exercising fetchSseStream against a real stream and one export download; until then note the gap beside the CI build-web step.
- Diff size: ~80 lines + CI step

### TEST-12 · Medium · 

**No test run anywhere uses -race (repo-wide grep: zero hits), while this remediation wave added shared mutable state: authThrottle maps, sendCooldown, presence registry.**

- Where: `.github/workflows/ci.yml:61`
- Evidence: ci.yml:61 'go test ./... -count=1 -timeout 20m' (no -race); store/prd_revisions_test.go:41-54 fires 8 goroutines at CreateRevision.
- Fix: Add -race to the CI go test invocation.
- Diff size: 1 line

### TEST-13 · Medium · 

**73 routes are registered; handler-level tests drive roughly 10 — admin (5), agent proposals (4), me (2), github/webhook (3), presence (3), roadmap_ai (2), workspaces (9), prds (12), roadmap (8), ideas (6) have no HTTP-layer test at all.**

- Where: `server/internal/api:1`
- Evidence: Route inventory from grep 'HandleFunc' sums to 73; handler-driving tests only in conversations_test.go, auth_cooldown_test.go:108-252, and validation-path posts in auth_email_flows_test.go.
- Fix: Prioritize by blast radius: one httptest each for admin (non-admin 403 + invite create), agent proposal decision, presence stream auth, roadmap_ai (nil-planner 503); add a table-driven all-routes-require-auth sweep.
- Diff size: ~200 lines

### TEST-14 · Low · 

**flutter analyze --no-fatal-infos has no ratchet: exactly 20 infos pass today, and the 21st passes just as silently — the info count can only grow.**

- Where: `.github/workflows/ci.yml:87`
- Evidence: ci.yml:85-87 comment + 'flutter analyze --no-fatal-infos'; local run counts 20 info / 0 warning / 0 error, nothing pins the 20.
- Fix: Fail if the info count exceeds 20 (ratchet down over time), or fix the 20 and drop the flag.
- Diff size: ~5 lines

### TEST-15 · Low · 

**Flutter is pinned to 3.47.0 in CI only; no .fvmrc and docs say just 'Flutter 3' — local runs and CI can silently diverge.**

- Where: `.github/workflows/ci.yml:81`
- Evidence: ci.yml:81 'flutter-version: 3.47.0'; no client/.fvmrc; docs/usage/local-development.md:11 '- **Flutter 3**'.
- Fix: Commit client/.fvmrc (or document the exact version) and have CI read the pin from it.
- Diff size: 1-3 lines

### TEST-16 · Low · 

**Prior TEST-06/07 remain open: 9+ tests still name the deleted Nuxt app as their contract oracle, and both source-scan tests read cwd-relative 'lib/...' paths.**

- Where: `client/test/requests_test.dart:5`
- Evidence: requests_test.dart:5 'match Nuxt'; l10n_test.dart:7 'matches Nuxt zh-CN'; feature_wiring_test.dart:14 cwd-relative read.
- Fix: Rename to self-describing contracts; resolve scan paths robustly with an explicit cwd assertion.
- Diff size: ~20 lines

### TEST-17 · Low · 

**Minor CI hardening gaps: postgres:18 floats across minors, no test output/artifact upload for the server job, bare python3 from the runner image.**

- Where: `.github/workflows/ci.yml:19`
- Evidence: ci.yml:19 'image: postgres:18'; no upload-artifact anywhere; python3 unpinned.
- Fix: Pin postgres:18.x, tee go test -v output and upload on failure, pin python-version.
- Diff size: ~10 lines

### TEST-18 · Low · 

**swapCooldown mutates package-global state with restore-on-cleanup — correct today because no test calls t.Parallel, but the safety rests on an unstated invariant; DB fixtures accumulate on a developer DATABASE_URL.**

- Where: `server/internal/api/auth_cooldown_test.go:30`
- Evidence: auth_cooldown_test.go:30-35 swap+restore of authSendCooldown; grep t.Parallel over server tests → 0 hits; only invitations_test.go deletes rows.
- Fix: One comment line on swapCooldown ('no t.Parallel in this package — global gate'), plus optional truncate-helper for test-prefix users.
- Diff size: ~5 lines

