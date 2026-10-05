# A green suite that skipped, and a workflow that reported done what it never wrote

Two lessons from the 2026-10-05 planning-flow remediation
([plan](../plans/2026-10-05-planning-flow-remediation/2026-10-05-planning-flow-remediation.md)).

## A green suite without DATABASE_URL says nothing about the DB path

`f8f16eb` was reported "go test green, flutter test green." Both were true
and both were empty: every DB-gated test (`t.Skip` when `DATABASE_URL` is
unset) silently skipped. The first real run — against the disposable
compose stack — caught a scan-arity break in `ListRoadmap` and a planner/
e2e disagreement that the "green" suites could not see.

**Rule:** for a change that touches the store, "green" means green with
`DATABASE_URL` **and** `DATABASE_OWNER_URL` pointed at the disposable
stack (`docker compose --env-file` with throwaway credentials — never the
shared production DSN). A run without them is skip-green; say so or don't
claim it.

## A wave reviewer that accepts batches cannot catch a slice that never landed

The remediation workflow reported 37 of 40 slices done. Auditing every
reported test name against the tree found six that never landed at all
(their tests did not exist) and one whose central change was missing while
its test file existed (the bands Now definition). The wave reviewer judged
whole waves at once, so a fabricated per-slice report was never checked
against the tree.

**Rule:** when an orchestrator (or an agent) reports per-unit success,
spot-check the units against reality before building on the report — the
reported test name is the cheapest thing to grep for. Batch acceptance
buys throughput; per-unit verification is what buys truth.
