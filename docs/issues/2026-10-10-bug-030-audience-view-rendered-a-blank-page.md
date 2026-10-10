# BUG-030 — the audience view rendered a blank page against the real server

## Symptom

Opening a PRD's audience view (Decision, Management, Delivery or Validation,
`/prds/{id}/view?audience=…`) showed an empty page instead of the sections that
audience reads. The widget tests for the page were green. A release web build
shows a grey page when a widget's build throws, which is what the member saw.

## Root cause

`GET /api/v1/prds/{id}/view` sends `projection.sections` as an **ordered list**
of `{section, why}` picks (`audienceview.SectionPick`; the order is the order the
audience reads the PRD in). The page cast it to `Map<String, dynamic>` and threw
`TypeError: List<dynamic> is not a subtype of Map<String, dynamic>` while
building, so every audience failed the same way.

The page (`8346684`, 2026-08-22) was written the same day as the server shape
(`66859c7`) and assumed a map from `section` to `why`. Its test fixtures were
written to match that parse, not copied from the wire, so the test and the page
agreed with each other and with nothing else. No test ran the page against the
server's real answer, and the server had no test that pinned the shape the
client depends on.

A second defect sat in the same file (review r3 `full-audience-view-request-waste`):
each section card fetched the whole PRD in its own `initState`, so a full page
sent three to five identical `GET /prds/{id}` calls next to the view call, and
a failure showed one error banner per card.

Found 2026-10-10 by the review-register sweep while it checked the
request-waste finding. Reproduced against a running server (the API answered
with a list; the browser console showed the TypeError) before any change was
made.

## Fix

| Step | Commit |
| --- | --- |
| Server pin: a real-PostgreSQL API test that each audience's `projection.sections` is a list of `{section, why}` in the audience's reading order | `09fef7e` |
| Red test: fixtures use the wire shape, the page throws the TypeError | `af91180` |
| Fix: the page reads the list of picks | `47be023` |
| Red test: two cards made two identical PRD requests | `6dc66de` |
| Fix: the page fetches the view and the PRD together, once, and hands each card its text; a failed fetch fails the page with a retry | `c467c63` |

## Verification

- `prd_view_test.go` `TestPrdViewSendsSectionsAsOrderedPicks` passes against
  PostgreSQL 18; changing the handler's list to a map fails it.
- `audience_view_page_test.dart`: the end-to-end test loads one audience from
  list-shaped fixtures and counts exactly one `GET /prds/{id}`. It failed with
  the TypeError at `af91180` and on the request count at `6dc66de`; it passes
  after `47be023` and `c467c63`.

Known ceiling: the wire shape is pinned on the Go side and the Dart fixtures
point at that test in a comment, but nothing compares the two automatically; a
future change to the shape is caught by whichever side's test is updated last,
not by both.

## Status

**Fixed 2026-10-10.**
