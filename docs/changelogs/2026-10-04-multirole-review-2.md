# 2026-10-04 — Multirole review 2 (whole tree)

**Scope:** whole repository, `main` @ `26db979` (post five-wave remediation).
**Type:** review and records only — no product code changed. Mandate: what to
fix, harden, improve, and the key features left to implement.

## What was done

- Ran the second multirole review: 10 reviewers (5 roles + 5 dimensions),
  3 skeptic verifiers (5 High findings each, refute-by-default), 1 completeness
  critic — 14 agents on one finding schema. The code-quality reviewer died on
  an account rate limit during the fan-out and was re-run serially.
- Coordinator scouting first, with leads verified by hand before the fan-out
  (notably the `package:web` `external Window? open(` declaration behind FS-01
  and the critic `nil`-messages call site behind AIDEV-09).
- Skeptics re-verified all 15 filed Highs: **7 confirmed, 8 downgraded to
  Medium, 0 refuted.** One sub-claim was refuted by execution evidence: PM-13's
  "all five commits undeployed" — both services were deployed after `26db979`
  and the domain answered 200/401; recorded as a [Correction] in the PM file.
- Reproduced every confirmed High at code level (grep/read; read-only review):
  the seven commands and outputs are in the report README.
- Report: [`docs/review/2026-10-04-multirole-2/`](../review/2026-10-04-multirole-2/README.md)
  — 108 unique findings (0 Critical · 7 High · 53 Medium · 46 Low), register
  with verdicts, executive summary with the features-left effort table,
  completeness critic, and a four-wave remediation plan.
- Closed the records: this changelog, the review index row, a finished plan
  file for the board row, and a HANDOFF pointer (which also records the
  `26db979` deploy — closing PM-13's record-hygiene remainder).

## What was found (headline)

No Criticals; the five-wave cleanup held. Seven confirmed Highs:

1. **AIDEV-09** — the fresh-context critic passes `nil` messages; with no
   `omitempty` the wire body is `"messages": []`, so every real critic call
   400s and silently skips. Production-invisible; unit tests use fakes.
2. **REL-13/SEC-10** — rightmost-XFF parsing keys per-IP throttles on the
   Railway edge IP shared by every user: one attacker trips the fail-closed
   backstop sitewide.
3. **USER-10** — the web coach SSE is a raw fetch that never refreshes on 401:
   expired token mid-coaching = permanent "stream failed" until reload.
4. **USER-09** — Drop never visibly drops (no dialog, dead `dropConfirm`
   string, no status filter, raw 'dropped' rendered).
5. **USER-11** — raw English server errors rendered as page content across
   12+ sites in a zh-CN UI (carried from review 1).
6. **FS-01** — popup-blocked print export silently no-ops; the fallback is
   dead code because `window.open`'s null return is ignored.
7. **FEAT-03** — agent proposals have no web accept/reject loop (route + MCP
   exist; client has zero proposal handling).

The features-left ledger (the fourth mandate ask) is filed as Mediums:
Branches/Merge, Decision Log, and per-project authority are real, unbuilt, and
explicitly end-state per PRODUCT.md's own header — slice-ordered in wave 4 of
the plan.

## What was not done

- No product code changed and nothing deployed (docs-only change; precedent
  `58dc2ea`). The plan's wave 1 is the next code work.
- No live signed-in UI walk, no real provider call, no live Railway canary
  (REL-13's XFF chain observation), no `-race` run, no fresh DB — the critic
  file lists every method limit.
- The CLI, observability, mail-delivery, and backup/restore surfaces got no
  dedicated review this round (critic's coverage holes); backup/restore is
  filed as CC-03 and enters wave 3.

## Verification

- `register.py self-test` passed; register generated from the fan-out JSON
  with verdicts joined on `(file, id)`.
- Redaction scans over the report folder before writing, both clean:
  - key-shaped: `rg -n -e 'sk-[A-Za-z0-9_-]{16,}' -e 'fc-[0-9a-f]{20,}' -e 'AKIA[0-9A-Z]{16}' -e 'ghp_[A-Za-z0-9]{30,}' -e 'xox[abp]-[A-Za-z0-9-]{20,}' -e '(SECRET|TOKEN|PASSWORD|API_KEY)[A-Z0-9_]*\s*[=:]\s*["'"'"']?[A-Za-z0-9/+_-]{16,}' -e 'postgres(ql)?://[^:]+:[^@]+@'` → no matches
  - deployment identifiers + local paths: `rg -n -e '[a-z0-9-]+\.up\.railway\.app' -e '(vercel|netlify|fly)\.(app|dev)' -e 'herokuapp\.com' -e 'ngrok\.(io|app)' -e 'yalotein' -e '/Users/'` → no matches
- Link check: every relative link target in the report folder exists.
