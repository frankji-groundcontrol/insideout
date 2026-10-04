# Multirole review — 2026-10-04

Reviewed tree: `main` @ `877ad03` (604 tracked files). Ten reviewers (five
roles, five dimensions), five skeptic verifiers, and one completeness critic
— 16 agents total. Every High finding was re-checked by an independent
skeptic; the top findings were reproduced by running the code.

> **Bottom line.** The engine is sound and honestly built: the Go server
> matches its architecture docs, migrations are transactional and
> schema-confined, one API contract is genuinely projected three ways
> (REST/CLI/MCP, 50 tools), tests are green, and git history is clean of
> secrets. The rot is at the doors and in the records: the README quick start
> dead-ends three separate ways (invitation-only signup undocumented, invite
> deep links drop their code, default CORS blocks the documented Flutter dev
> run), `docker compose up -d` cannot even parse for the documented setup,
> the live app breaks every 15 minutes when the access token expires with no
> refresh retry, auth endpoints have no rate limiting, the DB-integration
> suite that guards workspace isolation never executes anywhere, and the
> handoff/board still order an already-merged PR merge followed by a
> production deploy. Roughly one week of focused work closes the confirmed
> gap; most of it is small, verified, and ordered in
> [40-remediation-plan.md](40-remediation-plan.md).

## Findings at a glance

75 unique findings after dedup: **0 Critical · 12 High · 25 Medium · 38
Low.** High-severity verification: **12 confirmed · 2 downgraded · 0
refuted.**

| File | Aspect | Score | Highlights |
|---|---|---|---|
| [10-architect.md](10-architect.md) | structure, packaging, run paths | 6.0 | server matches docs; compose front door broken; INIT.md describes deleted stack |
| [11-ai-developer.md](11-ai-developer.md) | coach trust boundary, prompts, retries | 7.2 → 5.8* | retry duplicates streamed answers (confirmed High); quote grounding window; offline finalize gate |
| [12-fullstack.md](12-fullstack.md) | UI/API/deploy wiring | 5.5 | invite link drops code; CORS quickstart failure; nginx DNS-at-boot fragility |
| [13-project-manager.md](13-project-manager.md) | records, release gate | 5.0 | HANDOFF/board order merged-PR merge + deploy; no CI/tags; README sells superseded model |
| [14-user.md](14-user.md) | happy path, strings | 5.5 | quick start dead-ends at step 1; 15-min token expiry breaks live app |
| [20-feature-implementation.md](20-feature-implementation.md) | advertised vs implemented | 6.5 | quiet-project flag: zero code; board is title+status only |
| [21-code-quality.md](21-code-quality.md) | size rule, dead code, tests | 7.2 | oversize files untracked; substring "wiring" tests; dead helpers |
| [22-security.md](22-security.md) | auth, RLS, containers | 7.0 | no auth rate limiting (confirmed); mail send endpoints uncooled; root containers |
| [23-testing.md](23-testing.md) | suites, gates, CI | 5.5 | env-catalog suite red ~6 weeks; 43 DB tests skip forever; no CI |
| [24-reliability-ops.md](24-reliability-ops.md) | limits, timeouts, shutdown | 6.5 | AI concurrency limits hardcoded; SSE writes discard errors; compose forwards 7/~24 vars |
| [30-completeness-critic.md](30-completeness-critic.md) | coverage, calibration | n/a | fonts claim false (CRIT-01); two 7.2 scores incompatible |

\* critic-adjusted; see [30-completeness-critic.md](30-completeness-critic.md)
§3. Cross-role mean (filed): **6.2 / 10**.

## The confirmed Highs

1. **USER-01** — quick start step 1 dead-ends: signup is invitation-only by
   default; no quick-start doc says so (`README.md:41`).
2. **FS-02** — admin invite links drop `?invite=` before the register form;
   invitation onboarding is broken end-to-end (`client/lib/router.dart:28`).
3. **FS-03** — documented local-dev Flutter run is CORS-blocked with no hint
   (`server/internal/api/server.go:81`).
4. **ARCH-01** — `docker compose up -d` cannot parse without bundled-postgres
   passwords even for a remote `DATABASE_URL`; "whole stack" has no frontend
   (`docker-compose.yml:17`). Filed independently by five reviewers.
5. **USER-02** — 15-minute access-token expiry breaks the live app; the
   client refreshes only at boot, no 401 retry
   (`client/lib/api/client.dart:35`).
6. **SEC-01** — no rate limiting on any credential-verifying endpoint
   (`server/internal/api/auth.go:153`).
7. **AIDEV-01** — provider-retry re-streams a full second answer after a
   mid-stream failure; the parallel retry path is guarded, this one is not
   (`server/internal/agent/taxonomy.go:38`).
8. **FEAT-01** — the README's headline "quiet projects get flagged" has zero
   implementing code (`README.md:21`).
9. **FEAT-02** — the "shared board with live progress, blockers, and notes"
   renders title + status only (`workspace_page.dart:128`).
10. **PM-02** — HANDOFF and the task board still say "merge PR #3" sixteen
    days after merge; the next action ends in `railway up --service app`
    (`docs/HANDOFF.md:10`).
11. **ARCH-02** — `docs/INIT.md` presents the deleted Nuxt 4 + Ink & Seal
    stack as the current foundation (`docs/INIT.md:26`).
12. **TEST-02** — the RLS/workspace-isolation test layer never executes
    anywhere: 43 skips, no CI (`server/internal/store/authz_test.go:22`).

## Bugs reproduced by running the code

| Bug | Command | Result |
|---|---|---|
| Compose file unparseable for the documented remote-DSN setup (ARCH-01) | `docker compose config -q` | fails: "required variable POSTGRES_OWNER_PASSWORD is missing a value" (twice) |
| Env-catalog suite red (TEST-01) | `python3 -m unittest test_env_catalog` (in `scripts/`) | `46 passed, 1 failed` — orphans `INSIDEOUT_GH_CLIENT_ID/SECRET` |
| Quiet-project flag unimplemented (FEAT-01) | grep `stale\|quiet\|flagged\|inactive` over `server/internal` + `client/lib` | only unrelated AI-run reaper / presence TTL matches |
| DB-integration layer never runs (TEST-02) | `go test ./... -v` census | 188 top-level results, **43 SKIP**; no `.github/workflows` |
| Invitation-only signup undocumented (USER-01) | grep `invitation` in README/SETENV/local-development | zero matches; migration seeds `invitation_only` |

## Methodology

Branch enumeration first (only `main` @ `877ad03` carries the product; the
one feature branch is fully merged). Coordinator scouting produced 15 leads;
cheap ones were run before fan-out (build/vet/test green; compose config
fails; secret scans clean). Ten reviewers fanned out on one finding schema
(`file:line` + verbatim quote + fix-or-direction on every row). Five skeptics
re-verified all 14 High clusters refute-first: 12 confirmed, 2 downgraded, 0
refuted. The coordinator reproduced the five top findings above. A
completeness critic closed coverage, score calibration, and measurement
disputes, adding five verified findings (CRIT-01…05). Secret scan of this
folder: clean of live credentials and instance identifiers — the one
key-pattern hit is the *documented* compose dev-default superuser password
(quoted in SEC-07 to argue for its removal; it is a placeholder committed in
`docker-compose.yml` itself). `<railway-host>` / `<prod-domain>` placeholders
are used throughout. Flutter
tests accepted as reported (57/57 by the testing reviewer); the 43 skipped
DB-integration tests were not executed by anyone — no `DATABASE_URL`.

Leads not forwarded: none refuted at scouting; L13 (Flutter suite) delegated
to the testing reviewer; L14 (deploy freshness) assessable from records only.

## Reading order for a fixer

1. [40-remediation-plan.md](40-remediation-plan.md) — start at Wave 0, row one.
2. [00-executive-summary.md](00-executive-summary.md) — top ten, effort, door quotes.
3. [01-findings-register.md](01-findings-register.md) — every finding with evidence and verdict.
4. Role files for depth; [30-completeness-critic.md](30-completeness-critic.md) for what this review could not cover.

## Related

- Plan of record: [../../plans/2026-10-04-multirole-review.md](../../plans/2026-10-04-multirole-review.md)
- Changelog: [../../changelogs/2026-10-04-multirole-review.md](../../changelogs/2026-10-04-multirole-review.md)
- First review under `docs/review/`; conventions in [../README.md](../README.md).
