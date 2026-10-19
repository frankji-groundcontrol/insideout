# 2026-10-18 — Review-4 remediation: waves 0 to 3 recorded

Execution of the [review-4 remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md)
under its [execution contract](../../review/2026-10-18-multirole-4/41-remediation-contract.md):
four waves on the local branch `rv4/integration` (base `main` @ `d53035a`),
one worker per cluster, a red test before every fix, and a fresh adversarial
verifier before any cherry-pick. Findings are cited as `rv4-<ID>` because
review IDs restart per review. Nothing is pushed, merged or deployed; landing
the branch on `main` is the owner's step. Per-ID state for the Highs, the red
test and the owner decisions is in the
[open-findings record](../../issues/2026-10-18-review-4-open-findings.md).
Each wave has a file of its own here (outcome lists, per-ID dispositions,
gates, carry-forward and operator notes) so that every record stays under the
line budget; this page is the way in.

| Wave | Clusters | State | Record |
| --- | --- | --- | --- |
| 0 | `w0-privacy` (rows 0.1 to 0.4) | integrated; one gate failure, proven to be on the base | [wave-0.md](wave-0.md) |
| 1 | 9 clusters (rows 1.1 to 1.18) | integrated; gates green; 30 of 34 findings `FIXED`, 4 `PARTIAL` | [wave-1.md](wave-1.md) |
| 2 | 4 clusters (rows 2.1 to 2.6) | integrated at `e6bc637`; gates and a live check green; 23 rows: 21 `FIXED`, 2 `PARTIAL` | [wave-2.md](wave-2.md) |
| 3 | 13 clusters (rows 3.1 to 3.11, plus `w3-client-apply` and `w3-users-read-grant` added after wave 2) | integrated at `3a1c4ac` (171 commits above the wave-2 records head); gates green, workflow properties pending the first CI run; 64 rows: 48 `FIXED`, 13 `PARTIAL`, 3 `DEFERRED_OWNER` | [wave-3.md](wave-3.md), [wave-3-dispositions.md](wave-3-dispositions.md) |
| 4 | 21 clusters (carry-forward plus the whole-change review) | integrated at `97cf4c06` (174 cherry-picks, two pre-gate fixups); gates green on a re-run; not pushed | [wave-4.md](wave-4.md) |

## Cluster fragments

One fragment per cluster, written by its worker: what changed, the evidence,
the ceilings and the integrator's to-do. The wave files hold the dispositions.

| Wave | Cluster | Plan rows | Fragment |
| --- | --- | --- | --- |
| 0 | `w0-privacy` | 0.1 to 0.4 | [w0-privacy.md](w0-privacy.md) |
| 1 | `w1-cli-front-door` | 1.1, 1.2, 1.5 | [w1-cli-front-door.md](w1-cli-front-door.md) |
| 1 | `w1-ci-gates` | 1.4, 1.6, 1.11, CI half of 1.15 | [w1-ci-gates.md](w1-ci-gates.md) |
| 1 | `w1-clock-and-runes` | 1.3, 1.17 | [w1-clock-and-runes.md](w1-clock-and-runes.md) |
| 1 | `w1-auth-atomic` | 1.9, 1.10, cooldown half of 1.13 | [w1-auth-atomic.md](w1-auth-atomic.md) |
| 1 | `w1-apiclient-escape` | apiclient half of 1.13 | [w1-apiclient-escape.md](w1-apiclient-escape.md) |
| 1 | `w1-github-repo` | 1.14 | [w1-github-repo.md](w1-github-repo.md) |
| 1 | `w1-client-polish` | 1.16, dialog half of 1.12 | [w1-client-polish.md](w1-client-polish.md) |
| 1 | `w1-ops-docs` | 1.7, 1.8, 1.18, bootstrap half of 1.12 | [w1-ops-docs.md](w1-ops-docs.md) |
| 1 | `w1-misc-correctness` | 1.15 without its CI step | [w1-misc-correctness.md](w1-misc-correctness.md) |
| 2 | `w2-client-credentials` | 2.1, 2.2 | [w2-client-credentials.md](w2-client-credentials.md) |
| 2 | `w2-proposal-apply` | 2.3, 2.4 | [w2-proposal-apply.md](w2-proposal-apply.md) |
| 2 | `w2-agent-scope` | 2.5 | [w2-agent-scope.md](w2-agent-scope.md) |
| 2 | `w2-rls` | 2.6 | [w2-rls.md](w2-rls.md) |
| 3 | `w3-coach-trust` | 3.1 | [w3-coach-trust.md](w3-coach-trust.md) |
| 3 | `w3-stream-bounds` | 3.2 | [w3-stream-bounds.md](w3-stream-bounds.md) |
| 3 | `w3-auth-network` | 3.3 | [w3-auth-network.md](w3-auth-network.md) |
| 3 | `w3-release-supply` | 3.4 | [w3-release-supply.md](w3-release-supply.md) |
| 3 | `w3-privacy` | 3.5 | [w3-privacy.md](w3-privacy.md) |
| 3 | `w3-docs-truth` | 3.6, README hunks of 3.8 | [w3-docs-truth.md](w3-docs-truth.md) |
| 3 | `w3-test-hardening` | 3.7 | [w3-test-hardening.md](w3-test-hardening.md) |
| 3 | `w3-first-run-copy` | 3.8 | [w3-first-run-copy.md](w3-first-run-copy.md) |
| 3 | `w3-context-deadlines` | 3.9 | [w3-context-deadlines.md](w3-context-deadlines.md) |
| 3 | `w3-github-evidence` | 3.10 | [w3-github-evidence.md](w3-github-evidence.md) |
| 3 | `w3-client-assets` | 3.11 | [w3-client-assets.md](w3-client-assets.md) |
| 3 | `w3-client-apply` | 2.3, 2.4 (R2-01) | [w3-client-apply.md](w3-client-apply.md) |
| 3 | `w3-users-read-grant` | 2.6 (R2-02) | [w3-users-read-grant.md](w3-users-read-grant.md) |
| 4 | `w4-supply-chain` | PUB-07 | [w4-supply-chain.md](w4-supply-chain.md) |
| 4 | `w4-release-version` | PM-04, PM-05, PUB-03, SEC-07 | [w4-release-version.md](w4-release-version.md) |
| 4 | `w4-privacy-opener` | PUB-02 | [w4-privacy-opener.md](w4-privacy-opener.md) |
| 4 | `w4-project-handover-403` | TEST-03 | [w4-project-handover-403.md](w4-project-handover-403.md) |
| 4 | `w4-cli-setup-check` | QUAL-04 | [w4-cli-setup-check.md](w4-cli-setup-check.md) |
| 4 | `w4-env-truth` | FEAT-04 | [w4-env-truth.md](w4-env-truth.md) |
| 4 | `w4-copy-sweep` | PM-08, USER-10 | [w4-copy-sweep.md](w4-copy-sweep.md) |
| 4 | `w4-rv-client-boot` | whole-change client boot | [w4-rv-client-boot.md](w4-rv-client-boot.md) |
| 4 | `w4-rv-client-api` | whole-change client API | [w4-rv-client-api.md](w4-rv-client-api.md) |
| 4 | `w4-rv-redaction-guard` | whole-change redaction | [w4-rv-redaction-guard.md](w4-rv-redaction-guard.md) |
| 4 | `w4-rv-store-db` | whole-change store | [w4-rv-store-db.md](w4-rv-store-db.md) |
| 4 | `w4-rv-github-client` | whole-change GitHub client | [w4-rv-github-client.md](w4-rv-github-client.md) |
| 4 | `w4-rv-github-binding` | whole-change GitHub binding | [w4-rv-github-binding.md](w4-rv-github-binding.md) |
| 4 | `w4-rv-agent` | whole-change agent | [w4-rv-agent.md](w4-rv-agent.md) |
| 4 | `w4-rv-prd-cap` | whole-change PRD cap | [w4-rv-prd-cap.md](w4-rv-prd-cap.md) |
| 4 | `w4-rv-roadmap-expand` | whole-change roadmap expand | [w4-rv-roadmap-expand.md](w4-rv-roadmap-expand.md) |
| 4 | `w4-rv-auth-refresh` | whole-change refresh | [w4-rv-auth-refresh.md](w4-rv-auth-refresh.md) |
| 4 | `w4-rv-api-guards` | whole-change API guards | [w4-rv-api-guards.md](w4-rv-api-guards.md) |
| 4 | `w4-rv-apiclient` | whole-change API client | [w4-rv-apiclient.md](w4-rv-apiclient.md) |
| 4 | `w4-rv-mcpserver` | whole-change MCP session | [w4-rv-mcpserver.md](w4-rv-mcpserver.md) |
| 4 | `w4-rv-deploy-truth` | whole-change deploy truth | [w4-rv-deploy-truth.md](w4-rv-deploy-truth.md) |

## Where the open items are

- Residue with an ID, an owner and a one-line fix prompt: R1-01 to R1-17 (wave
  1), R2-01 to R2-15 (wave 2) and R3-01 to R3-50 (wave 3) in the
  [open-findings record](../../issues/2026-10-18-review-4-open-findings.md);
  the closed rows of every wave are in its
  [dispositions sibling](../../issues/2026-10-18-review-4-dispositions.md).
- Owner decisions (rotations, the version policy, whether an agent token may
  decide its own proposal, the release variable name, the retention and erasure
  rules, whether a skipped critic may reach finalize, what proves a linker
  controls a repository) are the last table of that record; the privacy halves
  are in [the owner record](../../issues/2026-10-18-rv4-owner-privacy-decisions.md).
- Seven wave-3 items were applied by the integrator before the gates, in
  `4060d64` (1 to 6) and `3a1c4ac` (7), and are not redone: the
  `INSIDEOUT_TRUSTED_PROXY_HOPS` entry in `.env.example`, 42 broken dated links
  (ceilings to 0), PostgreSQL 18 in the unowned files, the review-3 header, the
  `testlock` line and the new api file names in `docs/architecture/backend.md`,
  two rows of `docs/usage/environment.md`, and the `go.mod` toolchain and
  `x/text` bump. See
  [the coordinator note](wave-3.md#applied-before-the-gates-coordinator-note).
- Wave 4 is integrated. What is still open (prose to-dos, the wave-4b
  carry-forward, owner questions) is in [wave-4.md](wave-4.md). The first
  CI run after a push is still pending. Nothing is pushed.
