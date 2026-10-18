# 00 · Executive summary

**Repository:** InsideOut  
**Branch / commit:** `main` @ `9f0b1c7` (clean detached worktree)  
**Date:** 2026-10-18  
**Cross-lens mean:** **6.7 / 10** (6.3 across the 5 roles, 7.0 across the 8 dimensions; the critic reconciles these to a lower mean in [30](30-completeness-critic.md)).

## Verdict

Review 3 said the core was hardened but nothing shipped. Review 4 reads a product that has shipped (public since 2026-10-11, release `v0.1.0`) and the hardened core held: no Critical, 14 of 17 review-3 Highs verified closed at the cited location, `FORCE` RLS on 16 tables, and the real-database suite is green on a private PostgreSQL 18.4 cluster except one test. What is wrong is the **edge** and the **gates**. The front door is broken for a stranger: a self-hoster cannot create the first account, the binary's `--help` and `version` fail with a server-config error, and a saved login stops working after 15 minutes. The CLI, the MCP binary and the setup wizard resolve the API base four different ways and can send a stored token to the wrong server, the web "Expand with AI" button never works, re-accepting a proposal duplicates roadmap nodes, and the MCP server hands an agent the human-only Commit and proposal-decision tools. Underneath, the safety nets the records cite are not plugged in: `main` has been red since 2026-10-16 (a wall-clock test) and nothing noticed, because CI has no schedule, and the release workflow publishes on any `v*` tag without running a test. Ten Highs: nine confirmed by skeptic review (14 more downgraded to Medium) and one the completeness critic added (a deadline that displays a day early for every UTC+ user, which matters because the product is zh-first). None is a data-loss or remote-exploit bug. About three engineer-weeks close the register; the first day closes most of what a user would hit.

## Scores

| File | Lens | Score | Critic-reconciled | Why |
|---|---|---|---|---|
| [10](10-architect.md) | Architect (role) | 7 | 6.5 | Clean server layering; four disagreeing client base and token resolvers send a stored token to the hosted default. |
| [11](11-ai-developer.md) | AI developer (role) | 6 | 6 | Bounded coach loop with server-side gates; MCP tool policy is prose, so an owner-token agent can decide its own proposals. |
| [12](12-fullstack.md) | Fullstack (role) | 6.5 | 6 | Stack holds together; web Expand with AI is always broken and the CLI login dies in 15 minutes. |
| [13](13-project-manager.md) | Project manager (role) | 6 | 6 | Real discipline, but safety nets are unplugged: four to five guards run nowhere, CI has no schedule, the binary has no help. |
| [14](14-user.md) | User (role) | 5.8 | 5.8 | Core loop runs clean; the front door is broken (no first account, help fails, login does not persist). |
| [20](20-feature-implementation.md) | Feature implementation (dimension) | 6.9 | 6.3 | The advertised loop is built end to end; seams double-apply proposals and leave the human boundary unenforced. |
| [21](21-code-quality.md) | Code quality (dimension) | 8 | 7 | Disciplined and small-file; medium design debt, no High found. |
| [22](22-security.md) | Security (dimension) | 6.8 | 6.5 | Sound design (AST-enforced route auth, RLS, HMAC webhooks); abuse resistance and the release path are weak. |
| [23](23-testing.md) | Testing (dimension) | 6.6 | 6.2 | Real-DB, honest suite with seams: release ungated, no schedule, a global-row race, no CLI/MCP seam test. |
| [24](24-reliability-ops.md) | Reliability and ops (dimension) | 7 | 6.5 | Graceful shutdown and DB recovery work; no SSE stream cap and no deadline on request or health paths. |
| [25](25-closure-audit.md) | Closure audit (dimension) | 7.4 | 6.8 | 14 of 17 review-3 Highs verified closed; accounting over-claims and four guards unwired. |
| [26](26-public-release.md) | Public release (dimension) | 6 | 5.5 | No secret in tree or history; real emails, ungated and unsigned release path, no privacy notice. |
| [27](27-data-rls.md) | Data and RLS (dimension) | 7 | 6 | Schema confinement and FORCE RLS strong; write-side policies weak and untested. |

The critic re-scored each lens against its own findings and the measured reproductions; the reconciled mean is **6.24**, against 6.69 as reported. The reviewers' scores stay in the role files; the reasons are in [30](30-completeness-critic.md).

## Top ten by user impact

| # | Finding | Where | Severity | Fix size |
|---|---|---|---|---|
| 1 | **USER-01** A fresh self-hosted install cannot create its first account: `promote-admin` only updates an existing user, registration is invitation-only, admin settings return 401. Reproduced. | `store/invitations.go:193` | High | ~25 lines |
| 2 | **FULL-02 / USER-04** The CLI and MCP store a 15-minute access token and never use the refresh token; plain `login` persists nothing. Skeptics reproduced the expiry. | `apiclient/apiclient.go:39`, `setup/flow/flow.go:151` | High | ~60 lines |
| 3 | **ARCH-01** Four resolvers pick the API base; `insideout-mcp` ignores `api_base` in config and `roadmap ... --api B` silently talks to A with A's token. Reproduced. | `cmd/insideout-mcp/main.go:22` | High | ~40 lines |
| 4 | **FEAT-01 / AIDEV-04** The human-only Commit and proposal-decision boundary is a tool description: MCP exposes 52 tools, including `commit`, `proposal_decide`, `signup_mode` and `invite_*`, under one owner token. Reproduced. | `mcpserver/product_tools.go:120,176` | High | ~60 to 120 lines |
| 5 | **FEAT-02** Accepting a proposal with apply twice creates its nodes twice; apply is non-atomic and its error is swallowed; the web UI double-applies. Skeptic reproduced. | `api/agent_routes.go:254` | High | ~40 lines + migration |
| 6 | **FULL-01** Web Expand with AI casts the server's `{children:[...]}` to a `List` and never calls confirm, so the button always fails. | `client/lib/api/client_project.dart:231` | High | ~25 lines |
| 7 | **TEST-01** `release.yml` publishes unsigned artifacts on any `v*` tag with no test step and no dependency on CI. | `.github/workflows/release.yml:5` | High | ~6 lines |
| 8 | **L1 / PM-02 / ARCH-04** `main` is red: `TestProjectQuietFlagged` reads the real clock and fails after 2026-10-16; CI has no schedule, so it went unseen while HANDOFF says the head run is green. Reproduced. | `api/projects.go:80-84`, `ci.yml:4` | Medium | ~30 + 3 lines |
| 9 | **USER-02 / FULL-03 / PM-04** `--help`, `version`, bare `insideout` and any typo fail with "config: DATABASE_URL is required". Reproduced. | `cmd/insideout/main.go:42` | Medium | ~30 lines |
| 10 | **COMP-02** The deadline picker sends local midnight as UTC and the label shows the first 10 characters, so for every UTC+ user a picked 2026-10-18 displays as 2026-10-17. Measured with `TZ=Asia/Shanghai`. Found by the critic; the roadmap is time-first and the product is zh-first. | `client/lib/chrome/time_ago.dart:3` | High | ~10 lines |

## What each role would say at the door

- **Architect** ([10](10-architect.md)): "Fix ARCH-01 and ARCH-02 together: one clientconfig.Resolve used by CLI, --api flags and the MCP binary, then delete the other three resolvers including the scripts/mcp-config.sh shadow."
- **AI developer** ([11](11-ai-developer.md)): "Ship an enforced MCP agent tool policy (scoped agent token with no proposal_decide, commit, signup_mode set or invite tools, plus author-cannot-decide on proposals) together with the prompt contract that every tool result and MCP output carrying user-writable text is fenced and …"
- **Fullstack** ([12](12-fullstack.md)): "Fix web expandRoadmapNode (read children, call confirm) and add a contract-shape test; the CLI help/version fix (~15 lines) is the cheapest alternative."
- **Project manager** ([13](13-project-manager.md)): "Repair the one red test, then one small commit wiring the four orphaned guards into CI and adding a CI schedule, so the safety nets the records cite actually run."
- **User** ([14](14-user.md)): "Fix USER-01 and USER-02 together (upsert in promote-admin, ~25 lines; pre-dispatch help/version/unknown-verb block in main.go, ~30 lines), then USER-03/04 so a saved login survives past 15 minutes, then USER-07 (8 lines)."
- **Feature implementation** ([20](20-feature-implementation.md)): "Reproduce FEAT-02 and FEAT-01 as one regression test each, then fix FEAT-02 first because it silently doubles a roadmap people trust; then write a short design for the agent-scoped token."
- **Code quality** ([21](21-code-quality.md)): "Fix QUAL-02 and QUAL-03 (a few lines each, both reachable from untrusted input), then land two mechanical commits: (1) delete dead helpers (splitCORSOrigins, hasOpenBlockingFindings, duplicate tokenFromLink) and apply the unconvert, S1002/QF1001 and six errorlint %w/errors.Is …"
- **Security** ([22](22-security.md)): "Fix SEC-01 (atomic login-code verification, about 20 lines), then SEC-02 and SEC-06 together; today set a real INSIDEOUT_JWT_SECRET anywhere .env.example was copied, scope the GITHUB_TOKEN, and keep the Railway server service and compose server port off public interfaces."
- **Testing** ([23](23-testing.md)): "Add a go test step to release.yml and a schedule trigger with -shuffle=on to ci.yml."
- **Reliability and ops** ([24](24-reliability-ops.md)): "Set the two limits before the next release: a per-user/per-project cap on concurrent SSE streams sized under the 512m memory limit, and a per-request deadline (including a 2-3 s /healthz Ping) paired with stop_grace_period: 35s."
- **Closure audit** ([25](25-closure-audit.md)): "Wire the Python check_*.py truth scripts into CI."
- **Public release** ([26](26-public-release.md)): "Fix PUB-01 today: five doc edits plus a guard rule, then check the Supabase Data API exposure (PUB-05), then lock the release path before the next tag."
- **Data and RLS** ([27](27-data-rls.md)): "Write the failing test: as a user who is neither the workspace creator nor holding app.join_code, a bare INSERT INTO insideout.workspace_memberships for that workspace must fail, and role='admin' must never be insertable by a non-creator; then fix the policy."

## Effort

| Wave | What | Register IDs | Estimate |
|---|---|---|---|
| [0](40-remediation-plan.md) | Privacy and owner decisions | 6 | 0.5 day (owner time for 0.2 and 0.4) |
| [1](40-remediation-plan.md) | One-commit fixes (front door, gates, release) | 33 | 1.5 days |
| [2](40-remediation-plan.md) | Structural fixes with a red test first | 20 | 5 days |
| [3](40-remediation-plan.md) | Trust, reliability, release hardening, docs truth | 56 | 8 to 10 days |
| not in scope | features, rewrites, Lows | 30 | batch after Wave 3 |
| **Total** | | **145** (some IDs sit in two waves) | **about 3 engineer-weeks** |

## What not to do yet

- **Do not tag another release** before row 1.5 of the plan (`release.yml` runs no test) and row 1.2 (the binary has no version).
- **Do not add more guard scripts or docs claims** before the five existing `check_*.py` guards run in CI (row 1.6); an unwired guard is a claim, not a gate.
- **Do not widen the MCP tool surface** (FEAT-03 CLI and MCP verbs for observation and validation) before an agent token scope exists (row 2.5); every new tool inherits the owner token today.
- **Do not rewrite git history** (PUB-06) without a recorded owner decision (row 0.4); it needs a force-push of `main`.
- **Do not trust** "Fixed" ledger rows that are docs-only commits, the "zero broken doc links" claim, or "the head run is green" until rows 1.4 and 3.6 land.
- **Do not start multi-replica work** (ARCH-03, CLOSE-05, REL-04); the single-replica contract is documented and holds.

Next: [40-remediation-plan.md](40-remediation-plan.md), row 0.1 and row 1.1.
