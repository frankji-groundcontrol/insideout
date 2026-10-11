# Rerun the gate after a workflow's last repair round

**Date:** 2026-10-11
**Source:** [docs-order plan, C1](../plans/2026-10-11-docs-marksman-order.md)
([changelog](../changelogs/2026-10-11-docs-order-marksman.md))

## The lesson

A gated repair workflow whose gate runs **before** an adversarial-verifier
stage reports a reading that its own final edits invalidate. In C1 the
script ordered: clusters → deterministic gate (passed, `md_broken=0`) →
adversarial diff review → verifier-mandated repair (which reverted 5
"fixes" as verbatim evidence quotes). The run's conclusion truthfully
reported the gate it had run — but the tree it handed over no longer
matched it. The coordinator's independent re-run read `md_broken=5` and
caught what the run could not see about itself.

## Why it happens

The gate verifies a state, not a property. Any stage that edits after the
last gate execution invalidates the reading, no matter how honest the
report. Verifier-driven repairs are exactly such a stage: they are the
workflow's own mechanism for overturning earlier edits.

## When to apply

- Any workflow with gate → review → repair ordering: the coordinator re-runs
  every deterministic gate on the settled tree before believing the run's
  `verified` claims — one command per gate, cheaper than one wrong merge.
- When writing such a script, put a final gate execution **after** the last
  repair round, so the run's own report carries the final reading (the
  docs-order C2/D scripts do this).

## Evidence

C1 (dwfrun-b3fb78a0) concluded `orphans=0 md_broken=0`; the coordinator's
post-settlement re-run of the same script read `md_broken=5` — the five
verifier-reverted quote-evidence links. Resolution was not to re-"fix" the
links (quotes must stay verbatim) but to teach the gate the policy
(`QUOTED_EVIDENCE` exemption in `graph.py`), after which both gates read
green on the same tree.
