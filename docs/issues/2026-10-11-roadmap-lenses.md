# Per-role Roadmap Lenses are promised but not built

Open, recorded 2026-10-11 (review-3 wave 4, findings CQ-02 and SEC-04;
slice 6 of the [promise-gap set](2026-10-11-product-promise-gaps.md)).

## What PRODUCT.md promises

> "Different people use different Lenses over that graph:
> - **Decision:** outcomes, risk, time, and decisions required.
> - **Product:** persona paths, priorities, dependencies, blockers, and next
>   milestones.
> - **Delivery:** acceptance, work items, code review, tests, and deployments.
> - **Validation:** demos, hypotheses, requested feedback, and evidence.
> - **Agent context:** the current objective, Why, baseline, acceptance,
>   constraints, dependencies, and latest checkpoint."
> (PRODUCT.md:277-285)

## What exists today

- The roadmap UI offers exactly three views:
  `client/lib/features/project/roadmap_page.dart:36`
  `enum _RoadmapView { progress, bands, canvas }`.
- The API registers no lens parameter
  (`server/internal/api/roadmap.go:17-25`).
- The five-lens language exists today only as the PRD audience projections
  (`server/internal/audienceview/audienceview.go:72` — decision,
  management, delivery, validation), which project a PRD, not the roadmap
  graph.

## Slice order

6 of the set — a projection over the existing graph; lands after the
product-side slices it draws on (personas, dependencies).

## Fix prompt

Red first: a client test that the roadmap page exposes a Lens selector with
the five lenses fails. Smallest slice: derive each lens client-side from the
existing graph payload — Decision filters to outcomes + deadline pressure;
Product adds persona paths (PRD link) and blocked items (dependency edges,
slice 7); Delivery to acceptance + evidence; Validation to evidence kinds;
Agent context reuses the agentcontext assembly. One graph, no second store;
switching a Lens grants no permission (PRODUCT.md:287-288).
