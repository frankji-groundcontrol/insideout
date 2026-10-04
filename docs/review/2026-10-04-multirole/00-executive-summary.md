# 00 · Executive summary

**Repository:** insideout · **Branch:** main · **SHA:** 877ad03 · **Date:** 2026-10-04
**Cross-role mean:** 6.2 / 10 (filed) · 6.1 (critic-adjusted)

## Verdict

A sound engine behind broken doors. The Go server is well-structured,
honestly documented at the architecture layer, schema-confined, and projects
one API contract across REST, CLI, and a 50-tool MCP surface. Tests are green
where they run. But a new operator cannot get through the README: signup is
invitation-only with no documentation, the invite link itself drops its code,
the documented Flutter dev run is CORS-blocked, and the compose file cannot
be parsed for the documented database setup. The live app silently breaks 15
minutes after login. Auth endpoints accept unlimited credential attempts. The
security-critical RLS test layer has never executed in CI — because there is
no CI. And the records meant to prevent exactly this drift are themselves
stale: the handoff still orders a merged PR to be merged and then deployed.

## Score by role and dimension

| Reviewer | Score | One-line reason |
|---|---|---|
| Architect | 6.0 | internals match docs; front-door run paths and INIT.md do not |
| AI developer | 7.2 → **5.8** (critic) | coach seam is narrow and honest, but the confirmed stream-duplication High plus grounding-window and offline-gate Mediums were under-weighted |
| Fullstack | 5.5 | onboarding and dev-run wiring broken in three places despite green builds |
| Project manager | 5.0 | records stale in the exact spots the next agent reads first; no CI, tags, or owners |
| User | 5.5 | the first three minutes dead-end as written |
| Feature implementation | 6.5 | most advertised features exist; the two headline board promises do not |
| Code quality | 7.2 | clean idioms; oversize files and substring tests are the worst of it |
| Security | 7.0 | clean history and solid RLS design; unthrottled auth on a live host caps it |
| Testing | 5.5 | suites pass locally but the security layer never runs anywhere and one suite is red |
| Reliability-ops | 6.5 | deliberate limits exist for the coach; they are hardcoded, and SSE/compose gaps remain |

## Top ten by user impact

| # | Finding | Where | Severity | Fix size |
|---|---|---|---|---|
| 1 | Quick start dead-ends at registration (invitation-only undocumented) | README.md:41 | High (confirmed) | docs: ~15 lines across README/SETENV |
| 2 | Invite deep links drop the code | client/lib/router.dart:28 | High (confirmed) | ~10 lines, mirrors magic-link pattern |
| 3 | Live app breaks at token expiry; no 401 refresh | client/lib/api/client.dart:35 | High (confirmed) | one Dio onError interceptor |
| 4 | Local-dev quickstart CORS-blocked | server/internal/api/server.go:81 | High (confirmed) | dev.sh export + one docs line |
| 5 | `docker compose up -d` cannot parse; "whole stack" has no frontend | docker-compose.yml:17 | High (confirmed) | `:?`→`:-` + README wording |
| 6 | No rate limiting on credential endpoints | server/internal/api/auth.go:153 | High (confirmed) | middleware + per-account counter, ~80 lines |
| 7 | Coach provider-retry duplicates the streamed answer | server/internal/agent/taxonomy.go:38 | High (confirmed) | guard retry on `streamed.Len()==0`, ~5 lines |
| 8 | "Quiet projects get flagged" — zero code | README.md:21 | High (confirmed) | implement (small: `LatestUpdateAt` already returned) or cut the line |
| 9 | RLS test layer never executes; no CI | server/internal/store/authz_test.go:22 | High (confirmed) | one minimal workflow file |
| 10 | HANDOFF/board order a merged PR merge + deploy | docs/HANDOFF.md:10 | High (confirmed) | docs refresh, ~20 lines |

## What each role would say at the door

- **Architect:** "Decide the compose file's identity — one-command whole
  stack or dev database helper — because the front door fails before it
  starts."
- **AI developer:** "Ship the record_fact grounding-contract fix first; it is
  where the prompt contract and the code contract diverge."
- **Fullstack:** "Fix the invitation deep link — ten lines restoring an
  onboarding flow that is silently broken today."
- **Project manager:** "Run the release gate — deploy main to the hosted app,
  verify a signed-in page load, then update the records."
- **User:** "Rewrite the first three minutes to insert the missing
  signup-policy step."
- **Feature implementation:** "Strip the false board promises from the
  README — or implement them this week; both are small."
- **Code quality:** "Two mechanical cleanups, one commit each: dead code plus
  a staticcheck config, and the substring wiring tests."
- **Security:** "Rotate nothing — history is clean — but do not expose the
  hosted API past localhost until SEC-01 and SEC-02 land."
- **Testing:** "Stand up the minimal CI job and watch it fail — TEST-01 is
  red right now, so the first run must fail."
- **Reliability-ops:** "Wire AI_MAX_CONCURRENT and AI_QUEUE_WAIT to env
  before the next run; they are the only thing between a busy coaching hour
  and a wall of 503s."

## Effort table

| Bucket | Items | Estimate |
|---|---|---|
| Docs & records (Wave 0) | PM-02, ARCH-02, ARCH-07, PM-03, PM-06, TEST-01, USER-01 docs, ARCH-01 wording, redaction sweep (PM-08/SEC-08/CRIT-02), CRIT-01, CRIT-05 | 1 day |
| Client unblocks (Wave 1) | FS-02, FS-03, USER-02 | 1 day |
| Server hardening (Wave 2) | SEC-01, SEC-02, FS-07/CRIT-03, ARCH-05, REL-06, AIDEV-01, REL-08, REL-02 | 2 days |
| Truth in advertising (Wave 3) | FEAT-01, FEAT-02, FEAT-03, USER-05 decision, FEAT-05 | 2 days |
| Verification spine (Wave 4) | PM-05/TEST-02 CI, TEST-04, TEST-05, CQ-02, CQ-01 | 1 day |
| **Total** | 33 of 75 findings close the confirmed gap | **~1 week** |

## What not to do yet

Do not start new PRODUCT.md frontier features (Product Branches, Decision
Log UI, multi-provider Git), do not optimize go-to-market onboarding around
one persona, and do not add accessibility/performance programs until the top
ten close — the product loop the README promises must work end-to-end first.
Do not purge the 582 legacy users (operator call already on the board). Do
not seed a shared remote database with the local demo identity.
