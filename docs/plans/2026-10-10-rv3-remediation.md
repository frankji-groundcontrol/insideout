# 2026-10-10 — Execute the review-3 remediation plan (waves 1–4)

Execute every workflow-executable cluster of
[40-remediation-plan.md](../review/2026-10-10-multirole-3/40-remediation-plan.md)
— 4 sequential waves, 30 file-disjoint clusters — until the only rows left
open are the owner-blocked ones (deploy, keys, first `v*` tag). Each wave: one
dynamic workflow (worker → adversarial verifier → repair ≤3 per cluster, in
per-cluster git worktrees), then integrator cherry-pick `-x`, full gates,
and on to the next wave.

## Checklist

- [x] Wave 1 — 7 clusters (AIDEV-01/02/07, FULL-01/06/08, TEST-02, ARCH-01/02,
      REL-01, USER-01, FEAT-02): all verifier-accepted round 1 (27 commits),
      cherry-picked, plus 2 integrator seam commits (pendingStage on the GET,
      confirm applies parked resolutions); server suite green (one allowed
      skip), client 427 green, analyze 14; pushed f822873.
- [x] Wave 2 — 11 clusters (AIDEV-03/06, CQ-04, AIDEV-05/11, CQ-07, USER-08,
      SEC-02/06, FEAT-07, REL-02/03, SEC-08, FULL-02, CQ-03, USER-03, PM-04,
      SEC-07, ARCH-03, SEC-03, REL-05).
- [x] Wave 3 — 6 clusters (docs truth, env catalog, client polish x2, prompts
      contract, test hygiene): all verifier-accepted round 1 — 31 commits
      cherry-picked; TEST-06 closed by the integrator (parameterized circuit
      functions, migration 20261011100000 — its overload bug was caught and fixed
      on the integration DB before any push); server 26 pkgs green (one allowed
      skip), client 449 green, analyze 14.
- [ ] Wave 4 — 8 clusters (promise register, locale threading, upgrade lane,
      hosted smoke, attestation honesty, validation slice, contribution
      surface, gap issues).
- [ ] Whole-change adversarial review + full gates on the integrated tree.
- [ ] Records: register dispositions per finding, changelog folder, board,
      HANDOFF; push, CI green, MBP synced.): all verifier-accepted round 1 — 38 commits cherry-picked; the integrator closed the
      reported NOT_DONEs (CQ-03 shared snapshot flow, USER-08 localized marker; CQ-04
      parked by the brief's pure-move gate); server + client gates green; in bce47c4.

## Status

In flight 2026-10-11. Wave 4 workflow running over base `2c5631b` (waves 1-3 integrated): worktrees under `/tmp/insideout-gap/rv3-w1-*`, branches
`rv3/w1-*`, briefs/reports under `/tmp/insideout-gap/rv3/work/`, per-cluster
databases on the disposable cluster (`rv3w1*`). Owner-blocked rows are out of
scope by design and stay in [HANDOFF](../HANDOFF.md).
