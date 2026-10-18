# 2026-10-18 — Review-4 remediation: waves 0 to 2 recorded

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
| 3 | 11 clusters (rows 3.1 to 3.11), plus `w3-client-apply` and `w3-users-read-grant` added after wave 2 | not started | |

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

## Where the open items are

- Residue with an ID, an owner and a one-line fix prompt: R1-01 to R1-17 (wave
  1) and R2-01 to R2-15 (wave 2) in the
  [open-findings record](../../issues/2026-10-18-review-4-open-findings.md).
- Owner decisions (rotations, the version policy, whether an agent token may
  decide its own proposal, the release variable name) are the last table of
  that record; the privacy halves are in
  [the owner record](../../issues/2026-10-18-rv4-owner-privacy-decisions.md).
- Wave 3 starts after a pause; its carry-forward notes are at the end of
  [wave-2.md](wave-2.md#carry-forward-into-wave-3).
