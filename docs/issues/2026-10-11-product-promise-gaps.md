# PRODUCT.md promise gaps — the parked product slices

Open, recorded 2026-10-11 (review-3 wave 4, finding FULL-03; the records
for findings CQ-02, SEC-04, USER-04, FEAT-05 and FEAT-06 hang off this one).
This is the enumeration of what
[PRODUCT.md](../../PRODUCT.md) — "Authority: canonical target product
experience" (line 3) — promises that the running system does not implement,
with the recommended delivery order. One dated record per slice carries the
verbatim promise, today's tree state and a bounded fix prompt;
`scripts/tests/check_product_promise_gaps.py` fails while a slice is both
unimplemented and untracked.

## The delta, in recommended slice order

1. [Product Branch, Merge and the Decision Log](2026-10-11-product-branch-merge-decision-log.md)
   — the keystone: Commit/Merge language and the persona hypotheses hang
   off Branch (CQ-02, SEC-04).
2. [Scoped external feedback invitations](2026-10-11-scoped-feedback-invitations.md)
   — the only promise with an explicit privacy boundary; its security
   design comes first among the outward slices (CQ-02, SEC-04).
3. [Tokenized audience-share link](2026-10-11-tokenized-audience-share-link.md)
   — the read-only outward leg, reusing the /prds/{id}/view projection
   (USER-04).
4. [Persona grounding labels](2026-10-11-persona-grounding-labels.md)
   — core/key/later priority and grounded/synthesized/assumed labels (CQ-02).
5. [Facts trust layer](2026-10-11-facts-trust-layer.md) — the
   confirm/correct/remove surface parked from AIDEV-09's full path; it
   governs what the outward slices may expose (FULL-03).
6. [Roadmap lenses](2026-10-11-roadmap-lenses.md) — the five lenses as
   projections over the one graph (CQ-02, SEC-04).
7. [Roadmap dependency edges](2026-10-11-roadmap-dependency-edges.md) —
   depends_on edges, blocked surfacing, then the time-Diff rebaseline
   (FEAT-05).
8. [Focus surface](2026-10-11-focus-surface.md) — the per-session focus
   pointer and team Now, reusing the presence TTL (FEAT-06).
9. [Working-version autosave](2026-10-11-prd-autosave.md) — the autosaved
   mutable draft that never auto-Commits (FULL-03).

## Already disclosed in PRODUCT.md itself (not re-tracked here)

- Per-Project roles — PRODUCT.md:429-436 concedes the running v1 authority
  model is workspace-scoped and the per-Project roles are the target.
- The eight fixed PRD sections — PRODUCT.md:419-427 keeps them as the
  data-model scaffold "until flexible sections are designed".

These stay disclosed where they are so this enumeration does not create a
second truth; when a slice above ships, delete its record and this row.

## Cross-check note

Rows match the review-3 findings register
([01-findings-register.md](../review/2026-10-10-multirole-3/01-findings-register.md):
CQ-02, FULL-03, SEC-04, USER-04, FEAT-05, FEAT-06, AIDEV-09). The
consolidated 50-promise register (the `w4-promise-register` cluster) was
not yet in this branch's base; when it lands, its rows and these records
must be reconciled to one truth — every row here cites its PRODUCT.md
line for exactly that.

*Corrected 2026-10-11: the register has landed — the
[50-promise register](../review/2026-10-10-multirole-3/50-promise-register.md)
(stamped 2026-10-11, base `2c5631b`). The reconciliation above has not
been recorded yet.*
