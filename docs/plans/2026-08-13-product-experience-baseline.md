# 2026-08-13 — Product-experience baseline

Status: complete.

## Goal

Turn the completed product interview into one canonical, implementation-neutral
product baseline. The baseline must distinguish confirmed decisions from
derived delivery defaults, replace stale product assumptions without rewriting
historical plans, and define the smallest coherent end-to-end experience.

## Scope

- Rewrite `PRODUCT.md` as the canonical target product experience.
- Reduce `docs/INIT.md` to a pointer so it cannot compete as a second product
  definition.
- Link the canonical baseline from the documentation map.
- Record the documentation change in the changelog. Let the concise handoff
  link to the next implementation step without copying this task's history.

No source code, schema, API, or UI implementation changes are part of this
task. Historical plans remain unchanged as records of what was proposed or
built at the time.

## Checklist

- [x] Audit the current product, plan, architecture, and handoff documents.
- [x] Separate confirmed decisions from derived delivery defaults.
- [x] Write the canonical product-experience baseline.
- [x] Replace the competing product-definition document with a thin pointer.
- [x] Update the minimum indexes and records.
- [x] Review links, contradictions, and the working-tree diff.

## Review

Product and documentation reviews were run before editing. Engineering review
is not applicable: this task changes no runtime behavior or technical design.
