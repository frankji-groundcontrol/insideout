# 2026-08-22 — Audience view projections (one PRD core, many views)

PRODUCT.md "One PRD core, multiple audience views": the same fact core
projects as Decision / Management / Delivery / Validation views —
projections, never separately maintained documents. This increment made
the projections real on every surface.

## What changed

- `server/internal/audienceview`: the four projections — each
  audience's ordered section picks with a why-this-reader note, plus
  the audience's purpose statement from PRODUCT.md.
- `GET /api/v1/prds/{id}/view?audience=…`: the projection with that
  audience's readiness gaps and the latest Commit for context.
- Export: `GET /prds/{id}/export?format=markdown&audience=…` renders
  the projected document — audience title, purpose header with a
  "projected from the working version" disclosure, per-section why
  annotations, and blank sections carried as explicit open questions.
  Print+audience is a clear 400 for now.
- CLI: `insideout view [--audience A] [--export FILE] <prd-id>`; MCP
  tool `view` (18 tools). Web: the export page gained an audience
  dropdown (full document default), en/zh.

## Verification

- Unit: projections cover all audiences with complete picks (all keys
  real), unknown audiences rejected; audience markdown keeps order,
  leaks no non-projected section, annotates whys, carries blanks as
  open questions. Full server and client suites green (44/44), vet,
  gofmt.
- Live through the domain: `view --audience decision` returned the
  ordered projection with integrated readiness; the decision markdown
  export rendered the projected document with disclosure and open
  questions; `?audience=boss` → 400. Deployed on both services
  (server + app).

## Next product depth

- Coach weaves the same gap explanations into conversation
  (principle 7), and per-audience readiness can drive the "form a
  version now" dialog's default audience.
