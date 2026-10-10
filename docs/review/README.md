# Reviews

Dated whole-tree review reports: multirole audits that fan out role and
dimension reviewers over one finding schema, verify every High with skeptics,
reproduce the top findings by running the code, and close with an ordered
remediation plan.

## Convention

- One dated folder per review, `YYYY-MM-DD-slug/`.
- `README.md` is the index: bottom line, score table, confirmed Highs,
  reproduction table, methodology, reading order.
- Numbered files in reading order: `00-executive-summary.md`,
  `01-findings-register.md`, `1x-` role files, `2x-` dimension files,
  `30-completeness-critic.md`, `40-remediation-plan.md`.
- Every finding carries `file:line`, a verbatim quote, and a fix or a
  direction with a first check. Deployment identifiers and personal emails
  are redacted (`<railway-host>`, `<prod-domain>`, `<email>`).

## Records

- [2026-10-10 — Multirole review 3 (whole tree)](2026-10-10-multirole-3/README.md)
  — main @ 754c502, after the close-open-gaps wave; 10 reviewers + 3 skeptic
  clusters + critic in one dynamic workflow; 83 findings (0 Critical, 17 High —
  16 confirmed, 0 refuted; buckets FIX 22 / HARDEN 22 / IMPROVE 25 / ADD 8).
  Verdict: hardened core, nothing shipped — the server image cannot build
  (go.mod 1.26 vs golang:1.25 base), the stage-confirm door has no caller,
  the critique gate is model-rewritable, and the email funnel lies about its
  own TTL. Four workflow-executable waves in
  [40-remediation-plan.md](2026-10-10-multirole-3/40-remediation-plan.md).
- [2026-10-10 — Review-register disposition ledger](2026-10-10-disposition-ledger/README.md)
  — what happened to every finding of the three registers below that no issue
  record, plan, or changelog traced: 187 findings, each fixed (with the
  closing commit), re-proved not a defect, parked, or left to the owner;
  zero unresolved. Companion changelogs:
  [server](../changelogs/2026-10-10-close-open-gaps/review-sweep-server.md),
  [client](../changelogs/2026-10-10-close-open-gaps/review-sweep-client.md).
- [2026-10-04 — Multirole review (whole tree)](2026-10-04-multirole/README.md)
  — main @ 877ad03; 10 reviewers + 5 skeptics + critic; 75 findings
  (12 High confirmed, 2 downgraded, 0 refuted). Verdict: sound engine,
  broken doors — quick start dead-ends, no auth rate limiting, RLS tests
  never run, records stale. ~1 week of ordered remediation in
  [40-remediation-plan.md](2026-10-04-multirole/40-remediation-plan.md).
- [2026-10-04 — Multirole review 2 (whole tree)](2026-10-04-multirole-2/README.md)
  — main @ 26db979, after the five remediation waves; 10 reviewers +
  3 skeptics + critic; 108 findings (7 High confirmed, 8 downgraded,
  0 refuted; one PM sub-claim refuted by execution evidence). Verdict:
  no Criticals; one production-invisible critic bug, a web failure-path
  cluster, and an honest features-left ledger. ~5–7 focused days in
  [40-remediation-plan.md](2026-10-04-multirole-2/40-remediation-plan.md).
- [2026-10-06 — Frontend & Cobalt design-language review](2026-10-06-frontend-cobalt/README.md)
  — main @ f185e94; 10 reviewers + skeptics + critic; 114 findings
  (7 Critical, 22 High; 28 confirmed, 1 downgraded, 0 refuted). Verdict:
  sound engineering core, unfinished visual layer — dock renders at
  viewport center, chrome fails contrast, zero of four Cobalt-Neo
  mechanics implemented. ~11–12 engineer-days in
  [40-remediation-plan.md](2026-10-06-frontend-cobalt/40-remediation-plan.md).

