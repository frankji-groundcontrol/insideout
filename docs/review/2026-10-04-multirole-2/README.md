# Multirole review 2 — 2026-10-04, main @ 26db979

Second multirole review of the day, run on the tree that closed the first review's
five remediation waves (2f04a8e, 26db979). Mandate was broader than the first:
**what to fix, what to harden, what to improve, and the key features left to
implement**. Ten reviewers (five roles, five dimensions), a three-agent skeptic
pass over every High finding, code-level reproduction of each confirmed High,
and a completeness critic. 108 unique findings; 7 confirmed High; 0 Critical.

> **Bottom line:** The five-wave cleanup held — no Critical findings, and the new
> Wave 3–4 work (quiet flag, idea privacy, SSE test, CI spine) is among the
> strongest code in the repo. What remains splits cleanly in three: (1) one
> production-invisible agent bug — the fresh-context critic has sent an empty
> `messages` array since it landed, so every real critic call 400s and silently
> skips; (2) a web failure-path cluster — the coach's SSE never retries on 401,
> Drop never visibly drops, popup-blocked print export silently no-ops, and raw
> English server errors render inside a zh-CN UI; (3) the features-left ledger —
> PRODUCT.md's Branch/Merge, Decision Log and per-project authority are
> explicitly end-state, real, and unbuilt. Closing (1) and (2) is roughly three
> focused days; the feature ledger is slice-sized roadmap work, not defects.

## Findings at a glance

**108 unique findings: 0 Critical · 7 High · 53 Medium · 46 Low.**

*Corrected 2026-10-11: recounted, the severity tables in
[01-findings-register.md](01-findings-register.md) contain 106 rows (7 + 53 +
46), so the 108 headline does not match its own severity sum. The methodology
below records a third count (105 unique from 105 filed), and 22-security.md's
two Info findings (SEC-19, SEC-20) never entered the register — 106 table
rows + those 2 Info rows = 108, the likely origin of the headline. The table
rows are the ground truth.*

Skeptic verdicts on all 15 filed Highs: **7 confirmed, 8 downgraded, 0 refuted**;
one sub-claim inside PM-13 ("all five commits undeployed") was refuted by the
coordinator's execution evidence. Cross-role mean score: **7.3 / 10**.

| Aspect | File | Score | Highlights |
|---|---|---|---|
| Architect | [10-architect.md](10-architect.md) | 7.6 | CI toolchain ≠ artifact toolchain (ARCH-01, ↓Med); CI spine praised |
| AI Developer | [11-ai-developer.md](11-ai-developer.md) | 6.8 | Critic sends empty `messages` — never runs in prod (AIDEV-09 ✓High) |
| Fullstack | [12-fullstack.md](12-fullstack.md) | 7.2 | `window.open` null ignored, fallback dead (FS-01 ✓High) |
| Project Manager | [13-project-manager.md](13-project-manager.md) | 6.5 | Features-left ledger PM-10..18; [Correction] deploy claim refuted |
| User (seat) | [14-user.md](14-user.md) | 6.5 | Drop, SSE 401 loop, raw errors — all three confirmed High |
| Feature Implementation | [20-feature-implementation.md](20-feature-implementation.md) | 7.5 | Branches/Merge/Decision Log absent (↓Med, end-state); proposals have no web loop (✓High) |
| Code Quality | [21-code-quality.md](21-code-quality.md) | 8.4 | Line-budget rule holds; band-widget duplication; guarded-date helper bypassed |
| Security | [22-security.md](22-security.md) | 7.5 | XFF trusted-proxy (SEC-10 — see correction); ex-member idea-convert hole |
| Testing | [23-testing.md](23-testing.md) | 7.0 | Skip-guard scope (↓Med); no `-race`, no CI smoke |
| Reliability & Ops | [24-reliability-ops.md](24-reliability-ops.md) | 7.5 | Rightmost-XFF collapses per-IP throttle sitewide (REL-13 ✓High) |
| Completeness critic | [30-completeness-critic.md](30-completeness-critic.md) | n/a | CLI/observability/backup/mail unreviewed; CC-01..03 |

## The Criticals

None. The first review's Critical (committed key material) stays closed, and the
Wave 0–4 remediation introduced no new Critical-severity defects.

*Corrected 2026-10-11: the first review recorded 0 Critical (its register
totals: 0 Critical · 12 High · 25 Medium · 38 Low), and its security reviewer
verified "No secret has ever been committed — git history shows only
`.env.example` files ever tracked" — there was no committed-key-material
Critical to stay closed; the sentence above is factually wrong about review 1.*

## Bugs reproduced by running the code

Every confirmed High was reproduced by the coordinator at code level (read-only
review: no DB, no network, no browser — see methodology).

| Bug | Command | Result |
|---|---|---|
| AIDEV-09 critic sends empty messages | `rg -n 'StreamChatForcingTool' server/internal/agent/critic.go roadmap_planner.go` + `sed -n '99p;384p' anthropic.go` | `critic.go:231` passes `nil` where the planner passes `[]Message{{Role: RoleUser, …}}`; `Messages` has no `omitempty`; `make([]anthropicMessage, 0, len(msgs))` → wire body `"messages": []` |
| REL-13 rightmost XFF is the edge | `sed -n '292,297p' server/internal/api/auth_throttle.go` + `rg X-Forwarded-For client/nginx.conf` | `parts[len(parts)-1]` returned; nginx `proxy_add_x_forwarded_for` appends the Railway edge hop — every user shares one key |
| FS-01 `window.open` null ignored | `sed -n '17,24p' client/lib/api/download_web.dart` + grep in package:web 1.1.1 | return discarded, `return true` unconditional; SDK declares `external Window? open([` |
| USER-09 dropped ideas still listed | `sed -n '62,70p' server/internal/store/ideas.go` + `rg -l dropConfirm client/lib` | `WHERE workspace_id = $1 AND author_id = $2` — no status predicate; `dropConfirm` exists only in the two string tables, zero references |
| USER-10 web SSE has no 401 refresh | `sed -n '23,25p' client/lib/api/sse_transport_web.dart` | `throw ApiException('stream failed', status: res.status)` on `!res.ok`; no refresh anywhere on the fetch path |
| USER-11 raw English errors | `rg -c 'hasError \? Text' client/lib` | literal pattern in 7 files / 10 sites, plus 4 equivalent `Text(error!)` sites; no l10n error mapping exists |
| FEAT-03 no proposal handling in web | `rg -il proposal client/lib \| wc -l` | `0` files; `agent_routes.go:21` decision route exists server-side |

## Methodology

Eight phases: (1) scout the tree and verify leads myself; (2) fan out 10
role/dimension reviewers with one finding schema, max 12 findings each; (3) —
during the fan-out, one reviewer (code-quality) died on an account rate limit
and was re-run serially; (4) dedup with `register.py` (105 unique from 105
filed); (5) three skeptic agents, refute-by-default, over all 15 filed Highs,
each handed the finding plus the coordinator's execution evidence (the PM-13
deploy evidence disproved its headline claim); (6) code-level reproduction of
each confirmed High; (7) completeness critic; (8) this folder. 14 agents total.
Recorded subagent spend for the last five agents ≈ 5.2M tokens (code-quality
0.35M, skeptics 0.98/1.23/0.38M, critic 2.2M); the first nine reviewers' usage
was not retained. No finding was refuted outright; the rejected-leads story this
round is the PM-13 sub-claim, recorded as a [Correction] in
[13-project-manager.md](13-project-manager.md).

Secret scans over the folder before writing: key-shaped patterns and deployment
identifiers — **clean** (commands and output in the changelog for this review).

## Reading order for a fixer

1. [40-remediation-plan.md](40-remediation-plan.md) — waves, cheapest first.
2. [01-findings-register.md](01-findings-register.md) — all 108 rows by severity.
3. [14-user.md](14-user.md) then [11-ai-developer.md](11-ai-developer.md) — the two confirmed-High clusters.
4. [30-completeness-critic.md](30-completeness-critic.md) — what nobody covered.

*Note 2026-10-11: [00-executive-summary.md](00-executive-summary.md) —
verdict, top ten by impact, effort table — was written after this list;
read it first.*

## Related

- Prior review: [../2026-10-04-multirole/](../2026-10-04-multirole/README.md) (75 findings, five remediation waves).
- What this tree shipped since: [../../changelogs/2026-10-04-remediation-wave-3-4.md](../../changelogs/2026-10-04-remediation-wave-3-4.md), [../../changelogs/2026-10-04-remediation-wave-2.md](../../changelogs/2026-10-04-remediation-wave-2.md), [../../changelogs/2026-10-04-remediation-wave-0-1.md](../../changelogs/2026-10-04-remediation-wave-0-1.md).
- Review index: [../README.md](../README.md).
