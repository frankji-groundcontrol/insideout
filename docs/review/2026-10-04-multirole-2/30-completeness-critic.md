# 30 · Completeness Critic

**Lens:** coverage holes, unfiled contradictions, score coherence
**Score:** n/a (meta-review)

Read every bottom line, finding list and skeptic verdict from the ten reviewers,
then selectively verified in the tree. Adds three findings (CC-01..CC-03), all Low,
all personally verified before filing.

---

## Coverage holes

All four mandate asks received findings (fix: FS/USER/AIDEV items; harden: SEC/REL;
improve: CQ/ARCH; features-left: PM/FEAT), but several advertised surfaces got nobody:

- **CLI (`server/cmd/insideout/`, 12 files + `internal/apiclient`)** — zero findings. Only passing citations (FEAT-02 names `cli_version.go:125`). Nobody checked CLI error handling, arg validation, or its parity with the API contract.
- **Observability** — no findings on logs/metrics/alerting as a surface. Only REL-23 (email in logs) touches it. Verified: the sole liveness surface is `healthz` (DB ping); there is no metrics endpoint anywhere in `server/internal/api`.
- **Data export/backup/restore** — zero findings, zero story. Verified: `grep pg_dump|backup` over `docs/`, `scripts/`, `server/scripts/`, `docker-compose.yml` returns nothing. All member work (PRDs, revisions, facts, roadmaps) has no documented recovery path. → CC-03.
- **Mail delivery (`internal/mail`)** — unreviewed this round (auth email flows have tests, but sender config, From/spoofing posture, deliverability untouched).
- **GitHub evidence loop** — partially covered (SEC-17 body limit, FEAT-04 evidence read, FEAT-12 unmatched queue). Token storage is fine (verified `appauth.go`: short-lived installation tokens cached in memory, nothing persisted). The guide-matcher behavior end-to-end and the GitHub App private-key handling got no review.
- **Invitation/admin flows and MCP** — adequately covered (admin tests, USER-15/16; feature-implementation verified all 50 MCP tools). Migrations/RLS — well covered (ARCH-03/04, REL-14, SEC-11/13).

## Unfiled contradictions

1. **PRODUCT.md:312-313** "a deployment may show release" — but `github_webhook.go` handles only `ping`/`push`/`pull_request`; deployments are 200-acknowledged and dropped. The *generated* `insideout.yaml` (`guide.go:31`) even promises "An event (push, pull request, deployment) attaches evidence". Nobody filed. → CC-01.
2. **PRODUCT.md:416-418** lists "a Workspace admin approves all PRDs" as a *superseded assumption*, yet `store/prds.go:192-194` still implements exactly that model ("only a workspace admin approves or rejects (never the author acting alone)") and the README advertises it. PM-15 filed the sibling superseded assumption (rigid coach); this one is unfiled. → CC-02.
3. **Backup silence** — no doc claims a backup story, but a product whose value is stored user work has no restore narrative at all; no reviewer noticed the absence. → CC-03.

## Score coherence

| Role | Score | Own Highs | Confirmed Highs | Verdict |
|---|---|---|---|---|
| architect | 7.6 | 1 (downgraded) | 0 | Coherent (all Medium/Low post-downgrade) |
| ai-developer | 6.8 | 1 | 1 | Coherent |
| fullstack | 7.2 | 1 | 1 | Coherent, mildly generous (5 Mediums incl. SSE-401 dup of USER-10's High) |
| project-manager | 6.5 | 4 (all downgraded) | 0 | Coherent after downgrades; score rides on Medium breadth |
| user | 6.5 | 3 | 3 | Coherent (low score, most confirmed Highs) |
| feature-implementation | 7.5 | 3 | 1 | Slightly generous: confirmed High (FEAT-03) + 8 Mediums |
| code-quality | 8.4 | 0 | 0 | Coherent |
| security | 7.5 | 0 | 0 | **Incoherent in severity, not score**: SEC-10 is the same XFF defect REL-13 filed High and the skeptic CONFIRMED High (sitewide fail-closed) — security under-severitied its own finding |
| testing | 7.0 | 1 (downgraded) | 0 | Coherent |
| reliability-ops | 7.5 | 1 | 1 | Mildly generous: confirmed sitewide High + 6 Mediums |

## Missed findings

| ID | Sev | File:line | Summary (verified) |
|---|---|---|---|
| CC-01 | Low | `server/internal/api/github_webhook.go:50` | Deployment webhooks acknowledged+ignored; PRODUCT.md and generated guide promise deployment→release evidence |
| CC-02 | Low | `server/internal/store/prds.go:192` | Admin-approves-all-PRDs still the only approval path though PRODUCT.md declares it superseded |
| CC-03 | Low | `docker-compose.yml` (absence) | No backup/restore story exists anywhere for user data |

## What this review could not cover

- No live signed-in UI walk — every client finding (FS-01, USER-09..20) is code-reading; popup blocking, SSE reconnection, and l10n rendering were never observed in a browser.
- No real provider call — AIDEV-09 (empty messages 400) was confirmed by wire-shape reasoning against the SDK, not by hitting Anthropic.
- No live Railway probe — REL-13's XFF collapse still has its canary curl unrun; the two-proxy topology is inferred from docs.
- No concurrency/load verification — `-race` was never executed by anyone (TEST-12/REL-22 file it as unrun); no load test touched throttle/presence/SSE.
- No database was stood up — read-only review; RLS and migration findings rely on prior DB-gated tests, not fresh runs.
- Deploy state unverified — PM-13's signed-in hosted page load remains human-only; no reviewer saw production.
- Mobile/desktop builds of the Flutter client untouched; email deliverability (SPF/DKIM/From) untested.

## Bottom line

The register is broad and mostly self-consistent: all four mandate asks are populated, migrations/RLS and the MCP surface are well covered, and the seven confirmed Highs are genuine. The real holes are quiet surfaces — CLI, observability, backup/restore, mail — plus two PRODUCT.md contradictions (deployment evidence promised but dropped; admin approval "superseded" but still load-bearing) that the 105-finding register missed. The one score-side correction worth making: security's SEC-10 should be re-registered at High alongside REL-13, since the skeptic confirmed the same defect as a sitewide fail-closed backstop failure.

*Corrected 2026-10-11: recounted, the register's severity tables contain 106
rows (the folder README headlines 108, its methodology says 105 unique) — see
the correction note in [01-findings-register.md](01-findings-register.md).*
