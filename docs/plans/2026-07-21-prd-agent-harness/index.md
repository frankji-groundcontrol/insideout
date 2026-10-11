# 2026-07-21 — PRD agent harness redesign

Status: complete — plan reviewed and revised, ready to implement (pending the §9 user decisions).
*Corrected 2026-10-11: "ready to implement" understates history — the
harness's core mechanics (fact ledger, critic, streaming, run lifecycle)
largely shipped, and the [2026-07-27 hardening pass](../2026-07-27-hardening/README.md)
was already fixing bugs in its live components. Do not reimplement them;
see the board's superseded note
([plans/README.md](../README.md)).*

PRD writing is a subtle task: the failure mode of a naive agent is confident
fabrication — inventing users, requirements, and metrics the human never
validated. This plan researches how to design the coaching harness (possibly
multi-agent) so the agent elicits rather than invents, and reviews the
resulting design before any implementation.

## Files

- [research.md](research.md) — digested online research with sources:
  PRD-quality literature, agent-harness design patterns, multi-agent
  writing/critique systems, existing PRD-AI products.
- [plan.md](plan.md) — the harness design plan (the deliverable).
- [reviews.md](reviews.md) — plan-ceo-review and plan-eng-review findings
  and their resolutions.

## Checklist

- [x] Plan record opened, linked from `docs/plans/README.md`.
- [x] Online research fan-out completed and digested into `research.md`.
- [x] `plan.md` written, grounded in the current architecture
      (`docs/architecture/prd-coach-agent.md`) and the research.
- [x] plan-ceo-review run against `plan.md`; findings recorded.
- [x] plan-eng-review run against `plan.md`; findings recorded.
- [x] Plan revised to resolve review findings; resolutions recorded in
      `reviews.md`.
- [x] Indexes updated; links verified; plan marked complete.

## Parallelization

Research is a 6-lane workflow fan-out (independent reads): PRD-quality
literature, agent-harness patterns, academic multi-agent writing systems,
existing PRD-AI products, and the two gstack review skill definitions
(fetched so the reviews can be emulated faithfully). Plan writing is
coordinator-only (needs repo grounding). Reviews are a second 2-lane
fan-out against the written plan. Revision is coordinator-only.
