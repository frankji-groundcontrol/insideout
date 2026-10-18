# Multirole review 4 — 2026-10-18

A whole-tree review of InsideOut at `main` @ `9f0b1c7` (a clean detached worktree; the reviewers and the critic did not read the dirty in-flight checkout). 13 reviewers (5 roles, 8 dimensions) filed on one schema, 8 skeptic clusters tried to refute every one of the 23 Highs, the coordinator reproduced the top findings by running the code on private PostgreSQL 18.4 clusters, and a completeness critic closed the review. Review 3 is [here](../2026-10-10-multirole-3/README.md); this one reads the product after it went public (2026-10-11) and shipped `v0.1.0`.

> **Bottom line.** The hardened core held: no Critical, 14 of 17 review-3 Highs verified closed in the code, `FORCE` RLS on 16 tables, and the real-database suite is green on a clean checkout except one test. What is wrong is the edge and the gates. A self-hoster cannot create the first account; the binary's `--help` and `version` fail with a server-config error; a saved CLI login dies after 15 minutes; four resolvers disagree about which server a client talks to; the web "Expand with AI" button never works; re-accepting a proposal duplicates roadmap nodes; and MCP gives an agent the human-only Commit and decision tools. The critic adds that, for the zh-first product, a picked deadline displays a day early in UTC+ time zones and Chinese titles are limited by bytes, not characters. `main` has been red since 2026-10-16 because a test reads the wall clock and CI has no schedule, and `release.yml` publishes on any `v*` tag without running a test. About three engineer-weeks close the register; the first day closes what a stranger would hit. [Start at the plan](40-remediation-plan.md).

## Findings at a glance

**0 Critical · 10 High · 88 Medium · 47 Low = 145** after dedup. Of the 23 Highs filed, the skeptics **confirmed 9, downgraded 14 to Medium, refuted 0**; the critic added 8 rows, one of them High (COMP-02, reproduced by the coordinator instead of a skeptic). Cite IDs as `rv4-<ID>` outside this folder (review IDs restart each review).

| File | Lens | Score | Reconciled | Highlights |
|---|---|---|---|---|
| [10](10-architect.md) | Architect | 7 | 6.5 | four base resolvers, hosted URL baked in, wall clock in view builders |
| [11](11-ai-developer.md) | AI developer | 6 | 6 | MCP policy is prose; unfenced `get_prd`; skipped critic opens the gate |
| [12](12-fullstack.md) | Fullstack | 6.5 | 6 | Expand with AI always broken; 15-minute CLI token |
| [13](13-project-manager.md) | Project manager | 6 | 6 | five guards unwired, no CI schedule, no `--help` |
| [14](14-user.md) | User | 5.8 | 5.8 | no first account; `login` persists nothing; `setup --check` cannot fail |
| [20](20-feature-implementation.md) | Feature implementation | 6.9 | 6.3 | human boundary unenforced; apply twice duplicates nodes |
| [21](21-code-quality.md) | Code quality | 8 | 7 | no High; divergent client-IP copy, unescaped apiclient URLs |
| [22](22-security.md) | Security | 6.8 | 6.5 | login-code burst bypass; XFF tail trusted; unsigned supply chain |
| [23](23-testing.md) | Testing | 6.6 | 6.2 | release ungated, no schedule, global `signup_mode` race |
| [24](24-reliability-ops.md) | Reliability and ops | 7 | 6.5 | no SSE cap, no deadline on `/healthz`, 10 s SIGKILL |
| [25](25-closure-audit.md) | Closure audit | 7.4 | 6.8 | 14 of 17 closed; docs-only "Fixed" rows; guards unwired |
| [26](26-public-release.md) | Public release | 6 | 5.5 | real emails in docs, author identities in history, no privacy notice |
| [27](27-data-rls.md) | Data and RLS | 7 | 6 | write-side RLS weak and untested; no erasure path |
| [30](30-completeness-critic.md) | Completeness critic | 7.0 | n/a | 8 missed findings, 10 clusters, reconciled scores |
| [00](00-executive-summary.md) | Executive summary | n/a | n/a | top ten, effort, what not to do yet |
| [01](01-findings-register.md) | Findings register | n/a | n/a | 145 rows, 0 rejected |
| [40](40-remediation-plan.md) | Remediation plan | n/a | n/a | 4 waves, a check per wave |

Mean of the 13 scores: **6.7** reported, **6.2** after the critic's reconciliation.

## The Criticals and Highs

No Critical. Ten Highs: nine survived the skeptics and one (COMP-02) was added by the critic and reproduced by the coordinator:

1. **ARCH-01** — Client API base is resolved four different ways; the MCP binary ignores the config file's api_base while reading its token, and the CLI --api flag is dead or reset for roadmap, presence, version and guide verbs. (`server/cmd/insideout-mcp/main.go:22`)
2. **AIDEV-04** — All MCP tools share one flat registry and one process token; the human-only agent vocabulary exists only in SKILL.md, so a connected agent can accept its own proposal, commit, change signup mode and manage invites. (`server/internal/mcpserver/product_tools.go:176`)
3. **FULL-01** — Web Expand with AI casts the server's {children:[...]} draft to a List and never calls confirm, so it always fails. (`client/lib/api/client_project.dart:231`)
4. **FULL-02** — CLI setup persists only a 15-minute access token; clientconfig has no refresh token, and login only prints the token. (`server/internal/setup/flow/flow.go:151`)
5. **USER-01** — promote-admin only updates an existing user, so on a fresh invitation_only database no first account or admin can be created by the documented steps. (`server/internal/store/invitations.go:193`)
6. **USER-04** — The CLI/MCP never uses the refresh token, so the token the wizard saves to config expires after 15 minutes and every verb then fails with an env-var hint. (`server/internal/apiclient/apiclient.go:39`)
7. **FEAT-01** — The 'humans only' Commit and proposal decision are not enforced; the MCP server exposes them with the user's own token. (`server/internal/mcpserver/product_tools.go:120`)
8. **FEAT-02** — Accepting a proposal with apply twice creates its nodes twice; apply is non-atomic and its error is swallowed. (`server/internal/api/agent_routes.go:254`)
9. **TEST-01** — release.yml publishes unsigned release artifacts on any v* tag with no test step and no dependency on CI, and main is red now (L1). (`.github/workflows/release.yml:5`)
10. **COMP-02** — The deadline picker sends the local midnight as UTC, and isoDate() then shows the first 10 characters of the UTC string. For every UTC+ user the displayed deadline is one day before the picked date. Measured with TZ=Asia/Shanghai: picked 2026-10-18 renders 2026-10-17. (`client/lib/chrome/time_ago.dart:3`)

Reproduced Mediums worth reading first: USER-02 (help and version fail), DATA-01 (SQL-level self-insert as admin), PM-02 (the red test went unseen).

<a id="reproduced-by-the-coordinator"></a>

## Bugs reproduced by running the code

Clean worktree at the SHA above; private PostgreSQL 18.4 clusters with CI-parity roles (`insideout_owner` BYPASSRLS, `insideout_app` NOBYPASSRLS) on ports 55440 to 55499; commands under `env -i` with a sandboxed `HOME`; localhost only.

| # | Finding | Command | Result |
|---|---|---|---|
| R1 | L1, ARCH-04 | `go test ./internal/api -run TestProjectQuietFlagged -count=1`; full gate: `go test ./... -race -count=1 -v` after `insideout migrate` | FAIL `view.Flagged = true, want false` (`projects_test.go:51`); 25 packages ok, 855 pass lines, 1 allowed skip; `gofmt` clean |
| R2 | USER-01 | fresh migrated DB: `insideout promote-admin <email>`; `POST /auth/register` without an invite; `PUT /admin/settings` without a token | `not found`; 403 "an invitation is required to register"; 401. A fresh install has no documented way in. |
| R3 | DATA-01 | as `insideout_app` with `app.user_id` set: bare `INSERT INTO workspace_memberships` (role `admin`) for another user's workspace; then `UPDATE workspaces SET title` | visible workspaces 0 to 1; the `UPDATE` succeeds; policy `WITH CHECK (user_id = insideout.current_user_id())`. Skeptic: Medium, no API path. |
| R4 | ARCH-01 | config `api_base=A`; `insideout roadmap list --api B <project>` and `versions --api B <project>` | server A received both requests with the config token; server B saw nothing; `insideout-mcp` reads only `INSIDEOUT_API`/default |
| R5 | AIDEV-04, FEAT-01 | `insideout-mcp` over stdio, `tools/list` | 52 tools incl. `commit`, `proposal_decide`, `propose`, `signup_mode`, `invite_create`, `invite_revoke`, `auth_login` |
| R6 | USER-02, FULL-03, PM-04 | `insideout --help`, `insideout version`, `insideout frobnicate` with no `DATABASE_URL` | `config: DATABASE_URL is required`, exit 1, for all three |
| R7 | FULL-02, USER-04 | read `apiclient.go:39`, `auth.go:14`, `config.go:89`; skeptics ran `INSIDEOUT_ACCESS_TTL=4s` | refresh token documented as unused; default access TTL 15 m; expiry reproduced end to end |
| R8 | TEST-01, PM-05 | grep `release.yml` for `needs:`, `go test`, `workflow_run` | 0 matches; trigger is `push` of `v*` tags; publish is `gh release create ... \|\| gh release upload --clobber` |
| R9 | FULL-01 | read `roadmap_ai.go:233` and `client_project.dart:231`; skeptic ran `flutter test` | server returns `{"children": views}`, client casts `(d as List)`: `TypeError` |
| R10 | FEAT-02 | skeptic, private cluster: accept a proposal with apply twice | 2 roadmap nodes; the web client also applies client-side (`project_updates_feed.dart:44-45`). Not re-run by the coordinator. |
| R11 | PUB-01 | count non-placeholder addresses in tracked docs (counts only, values not printed) | 3 personal or owner addresses plus fragments; `check-redaction.sh` matches one literal address |
| R12 | PM-06 | relative-link scan of 528 markdown files (three parsers) | 74, 75 and 76 broken; 4 point at `.md` files, the rest at moved source paths; the last commit says zero |
| R13 | L4 | `flutter analyze --no-fatal-infos && flutter test` in `client/` | 0 issues, 458 tests pass (HANDOFF says 456: small doc drift) |
| R14 | COMP-02 | `TZ=Asia/Shanghai dart run` of the picker line (`picked.toUtc().toIso8601String()`) and `isoDate` (`substring(0, 10)`), no Flutter dependencies; UTC and New York as controls | Shanghai: sent `2026-10-17T16:00:00.000Z`, shown `2026-10-17` for a picked 2026-10-18; UTC and New York show 2026-10-18 |
| R15 | COMP-01 | count a 67-character Chinese title in Go; grep the `len(req.X) > N` sites in `internal/api` (16 on strings) | 67 characters = 201 bytes, rejected by `len(...) > 200` whose message says "max 200 characters"; the in-tree rune pattern is `project_updates.go:49` |

Not run: the hosted deployment beyond one read-only health probe during scouting, a real mail provider, and a real LLM turn (no keys were read or used). See [30](30-completeness-critic.md) for what the critic says was not covered.

## Methodology

1. **Find the product.** `main` is the product; every reviewer worked from a clean detached worktree at `9f0b1c7` so the dirty in-flight checkout could not leak into any finding.
2. **Scout and run leads.** 18 leads were written and every cheap one run before the fan-out: 15 run-confirmed, 3 forwarded as not-run (L7 machine manifests, L17 review-3 ledger claim, L18 schema confinement), none listed as refuted. One lead (L1, the red test) became the first reproduction.
3. **Fan out.** 13 agents in parallel on one schema: 5 roles (architect, AI developer, fullstack, project manager, user) and 8 dimensions (feature implementation, code quality, security, testing, reliability and ops, closure audit, public release, data and RLS). Cap of 12 findings each, `file:line` plus a verbatim quote or dropped, a fix or a direction plus first check on every row.
4. **Dedup, then refute.** 137 unique rows before the critic, 145 after (dedup key `(file, summary[:60])`). The 23 Highs went to 8 skeptic clusters told to refute by default and given the coordinator's run results: 9 CONFIRMED, 14 DOWNGRADED, 0 REFUTED, 0 UNVERIFIED.
5. **Reproduce.** The coordinator ran R1 to R6, R8, R11 to R13 and, after the critic, R14 and R15 above; skeptics reproduced R7, R9 and R10 (the coordinator read the cited lines for R7 and R9).
6. **Completeness critic.** One agent answered the four questions in [30](30-completeness-critic.md): 8 missed findings (COMP-01 to COMP-08), a score reconciliation (mean 6.69 to 6.24), three verdict-versus-measurement notes and 10 root-cause clusters. It changed no severity.
7. **Redact, then write.** Key-shaped scan and identifier scan over the whole folder, below.

Fleet: 22 agents (13 reviewers, 8 skeptic clusters, 1 critic), about 8.1 M tokens across the reviewers and skeptics plus 0.34 M for the critic. No finding was rejected, so the register has no "Rejected" rows.

**Redaction scan (run after the last edit, over this folder and the records written with it).**

```text
key-shaped patterns (sk-, fc-, AKIA, ghp_, xox, *_SECRET/TOKEN/PASSWORD/API_KEY assignments,
  postgres URLs with credentials, JWTs, publishable-key prefix, private-key headers): 0 matching lines
hosted API host (read from default_base.go): 0 matching lines
provider app hosts (railway, supabase, vercel, netlify, fly, heroku, ngrok): 0 matching lines
home path, machine hostnames, ssh alias, institution and owner-domain names: 0 matching lines
email addresses other than example.*, noreply, resend.dev and insideout.local: 0
not counted as identifiers: the public repo path, the key name ssh_alias, and the documented placeholder domains above
```

## Reading order for a fixer

1. [40-remediation-plan.md](40-remediation-plan.md), rows 0.1 and 1.1 first.
2. [00-executive-summary.md](00-executive-summary.md), the top ten and "what not to do yet".
3. [01-findings-register.md](01-findings-register.md), then the role file for the area you are touching (10 to 14, 20 to 27).
4. [30-completeness-critic.md](30-completeness-critic.md), for the root-cause clusters and what was not covered.

## Related

- Prior review: [2026-10-10 multirole review 3](../2026-10-10-multirole-3/README.md); the remediation that followed: [rv3 plan](../../plans/2026-10-10-rv3-remediation.md) and the [disposition ledger](../2026-10-10-disposition-ledger/README.md), whose 17 Highs the closure audit ([25](25-closure-audit.md)) re-checked in code.
- Records: [changelog](../../changelogs/2026-10-18-multirole-review-4.md), [plan](../../plans/2026-10-18-multirole-4.md), [issue tracker](../../issues/2026-10-18-review-4-open-findings.md), [learning note](../../learning/2026-10-18-a-wall-clock-test-needs-a-scheduled-run.md).
- [Review index](../README.md).
