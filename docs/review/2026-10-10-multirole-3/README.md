# Multirole review 3 — 2026-10-10

Reviewed tree: `main` @ `754c502` (1346 tracked files, the tip that absorbed the
close-open-gaps wave). Ten reviewers (five roles, five dimensions), three
skeptic clusters, and one completeness critic — 14 agents — run as one dynamic
workflow (`dwfrun-b9216947`, 52 minutes). Every High finding went to an
adversarial skeptic; the coordinator reproduced the load-bearing Highs with
real commands against the tree and the live hosted deployment.

> **Bottom line.** The codebase the close-open-gaps wave left behind is
> genuinely hardened — the api/agent seam held, the fences are mechanical, the
> surfaces stay pinned, and no Critical finding exists. What review 3 found is
> concentrated in three clusters every role independently converged on:
> **(1) nothing is shipped** — the server Docker image cannot build (go.mod
> demands Go 1.26.0, the image pins 1.25), four security migrations sit
> unapplied on the hosted database, and the email funnel — the only way an
> external user enters — has never run against its provider and contradicts
> itself across three records; **(2) the human-override gates have model-sized
> holes** — a critique-stage restatement wholesale-replaces the stored critic
> findings the finalize gate reads, the stage-confirm endpoint has no caller on
> any surface so the persisted stage can never advance, and one planner path
> still ships user text unfenced; **(3) the product's own promises leak** —
> every LLM prompt speaks Chinese-only against a bilingual commitment, idea
> conversion drops the body the coach was supposed to inherit, and the offline
> template coach is indistinguishable from the real one. 83 findings: 17 High,
> 33 Medium, 33 Low; the workflow-shaped remediation plan is
> [40-remediation-plan.md](40-remediation-plan.md).

## Scores

| Reviewer | Score | High+ | One-line verdict |
|---|---|---|---|
| code-quality | 8.6 | 0 | Cleanest sweep of the set; only mechanical debt left |
| fullstack | ~~8.4~~ → **8.0**\* | 1 | Controls honest, async contracts hold — except the invite email's 60x TTL lie |
| testing | 8.2 | 2 | Strong suites; migration-upgrade and email-funnel legs untested |
| security | 8.0 | 1 | No new holes in code; the holes are undeployed fixes |
| reliability-ops | 8.0 | 1 | Kill paths real; native SSE dies at 30s, no query/turn bounds |
| completeness-critic | 7.6 | 1 | Review set strong; feature-implementation never actually performed |
| architect | 7.5 | 1 | Layering survived; deploy path broken at the toolchain floor |
| ai-developer | 7.5 | 2 | Gates mechanical but two model-defeatable seams remain |
| project-manager | 7.5 | 1 | Top-decile governance; zero releases, stale deploy, contradictory records |
| feature-implementation | 7.0 | 4 | Core loop real; bilingual promise, idea body, validation leg missing |
| user | 6.5 | 3 | The loop is real and honest — but stalls at trust: no stage handle, dead email paths, stub coach |

\* Recalibrated by the critic (COMP-03): 8.4 disagreed with its own confirmed
High (FULL-01) and carried a wrong "verified 375/375" count (the measured set
is 379/379, COMP-06).

## Severity totals

**0 Critical · 17 High · 33 Medium · 33 Low = 83** (77 reviewer findings, all
dedup-unique this round, plus 6 critic rows). Skeptics: **16 CONFIRMED, 0
downgraded, 0 refuted**; COMP-01 (critic's High) was not skeptic-verified.
Buckets: FIX 22 · HARDEN 22 · IMPROVE 25 · ADD 8 (+6 critic rows).

## Reproduced by the coordinator

| Finding | Command / read | Result |
|---|---|---|
| ARCH-01 | `grep '^go ' server/go.mod` + `grep FROM server/Dockerfile` | **Confirmed** — `go 1.26.0` vs `golang:1.25-alpine`. **Fix corrected:** `go list -m -json all` shows `golang.org/x/sys` and `golang.org/x/term` already declare `go 1.26.0`, so lowering go.mod to 1.25 (the architect's first option) is **not viable**; the Dockerfile must move to `golang:1.26-alpine`. Docker was unavailable here too, so the build break itself stays pin-established |
| AIDEV-01 | read `critic.go:112-126` | **Confirmed** — `captureRankedCritique` calls `saveCriticFindings(extra, findings)` which assigns a fresh array; only the critic pass (critic.go:79) merges |
| AIDEV-02 | read `roadmap_planner.go:182-208` | **Confirmed** — PlanMVP appends `dataFenceClause` and wraps fields in `fenceData`; ExpandNode's system prompt has neither and interpolates three user-writable fields raw |
| FULL-01 | read `admin.go:127-134` + `auth_email_flows.go:338-344` | **Confirmed** — `int(ttl.Hours())` (168 for the 7-day default) feeds a template hardcoding "expires in %d minutes"; the text body renders `humanTTL(ttl)` correctly. Also confirmed FULL-06: green `#1f6f54`, 8px radius |
| USER-01 | `grep -rn` for the stage endpoint across `client/lib`, `server/cmd`, `server/internal/mcpserver` | **Confirmed** — `POST /conversations/{id}/stage` is registered (conversations.go:16) and has zero callers anywhere |
| USER-02 / PM-01 / SEC-01 premise | `curl https://insideout.yalotein.net/api/v1/auth/config` | **Confirmed live** — response is `{"signupMode":"invitation_only"}`: no `mailEnabled` field, so the deployed binary predates the wave and the client's `?? true` default asserts mail-on |
| USER-03 | read `template.go:24-30` | **Confirmed** — offline reply is Chinese-only and instructs the reader to edit server env vars; no offline marker in the SSE stream |
| FEAT-01 | `grep -rn 'lang\|Language' server/internal/agent/*.go` | **Confirmed** — no language parameter exists; all stage prompts hard-coded Chinese |
| FEAT-02 | read `ideas.go` ConvertIdea | **Confirmed** — `idea.Content` is scanned and never used again; `insertPrd` gets only `idea.Title` |
| REL-01 | read `client_core.dart:20-30` + `client_stream.dart:46` | **Confirmed** — shared Dio `receiveTimeout: 30s`; the native coach stream rides `_dio.post` with no per-request override |
| COMP-02 | read `store/prds.go:15-21` vs `PRODUCT.md:420` | **Confirmed** — `PrdSectionKeys` is exactly the eight fixed sections PRODUCT.md declares superseded |

## Methodology

One dynamic workflow: 10 reviewer briefs (roles
[10](10-architect.md)–[14](14-user.md), dimensions
[20](20-feature-implementation.md)–[24](24-reliability-ops.md)) fanned out in
parallel over the same tree, each capped at 12 findings with a mandatory
`file:line` verbatim quote and a fix or a direction; in-script dedup on
`(file, summary[:60])`; every High/Critical handed to one of three adversarial
skeptic clusters (default REFUTED unless they could quote the motivating
line); a completeness critic over all bottom lines, findings, and verdicts.
Reviewers were briefed with the review-3 context: prior reviews are fully
dispositioned in the [ledger](../2026-10-10-disposition-ledger/README.md) —
re-filing a dispositioned row was out of scope; sweep commits were fair game.
Findings carry the owner's four buckets (FIX / HARDEN / IMPROVE / ADD) as
dimension prefixes. Credentials and deployment identifiers are redacted
throughout; no `.env` content was read.

## Reading order

0. **Outcome:** the plan was executed 2026-10-11 — four verified waves
   (`5c8f54f..4464330`) close 63 of 83 rows by code; the rest are 10 recorded
   (a docs-only fix; recomputed 2026-10-18, rv4-CLOSE-04), 3 satisfied by the
   report, 1 parked and 6 owner-blocked (closed by the owner the same day).
   Every row's disposition
   is in the [ledger](../2026-10-10-disposition-ledger/04-multirole-3.md)
   and the changes in the
   [changelog](../../changelogs/2026-10-11-rv3-remediation/index.md).
1. [00-executive-summary.md](00-executive-summary.md) — top ten, effort by bucket
2. [01-findings-register.md](01-findings-register.md) — every finding, verdict-joined
3. [50-promise-register.md](50-promise-register.md) — every PRODUCT.md promise stamped IMPLEMENTED/PARTIAL/DEAD/DOCS-ONLY (lead 14, run in remediation wave 4)
4. [40-remediation-plan.md](40-remediation-plan.md) — the workflow-executable plan
5. [30-completeness-critic.md](30-completeness-critic.md) — what this review could not cover
6. Role and dimension files, worst-first: [14-user](14-user.md), [20-feature-implementation](20-feature-implementation.md), [11-ai-developer](11-ai-developer.md), [10-architect](10-architect.md), [23-testing](23-testing.md), then the rest

*Note 2026-10-11: "the rest" is [12-fullstack](12-fullstack.md),
[13-project-manager](13-project-manager.md),
[21-code-quality](21-code-quality.md) and [22-security](22-security.md).*
