# 2026-08-22 — Audience view projections

Status: **complete** (2026-08-22). Implements PRODUCT.md's "One PRD
core, multiple audience views": the Decision / Management / Delivery /
Validation views as projections of the same sections — never separately
maintained documents.

## Checklist

- [x] `server/internal/audienceview`: four projections, each with
      ordered section picks + why-this-reader notes + purpose
- [x] `GET /api/v1/prds/{id}/view?audience=…` — projection, readiness
      gaps for that audience, latest Commit
- [x] Export integration: `format=markdown&audience=…` renders the
      projected document (disclosure header, why annotations, blanks
      carried as open questions); print+audience → clear 400
- [x] CLI `insideout view [--audience] [--export]` + MCP `view` (18
      tools)
- [x] Web: export page audience dropdown (full document default), en/zh
- [x] Unit tests + live verification through the domain; deployed to
      both Railway services
      ([changelog](../changelogs/2026-08-22-audience-views.md))

## Follow-ons

- Coach weaves the same gap explanations into conversation (principle 7)
- Per-audience readiness can preselect the commit dialog's audience
- Print/HTML rendering of audience views when needed

## Sources

- PRODUCT.md "One PRD core, multiple audience views"
- `internal/readiness` (gap priorities and reasons)
