# Secondary audiences cannot be attached to a working version

Open, recorded 2026-10-18 (review-4 wave 3, finding COMP-08; row 50 of the
[promise register](../review/2026-10-10-multirole-3/50-promise-register.md)).
A PARTIAL promise with no dated record until now; context in the
[promise-gap enumeration](2026-10-11-product-promise-gaps.md).

## What PRODUCT.md promises

> "Each working version has one human-confirmed primary audience. The Coach may
> recommend it, and secondary audiences may be attached."
> ([PRODUCT.md](../../PRODUCT.md):157-158)

## What exists today

- The primary audience is stored on every Commit: `primary_audience text NOT
  NULL` (`server/db/migrations/20260821150000_prd_commits.sql:11`), written by
  `CreatePrdCommit` (`server/internal/store/prd_commits.go:34`) and chosen in
  the client by `client/lib/features/prd/audience_selector.dart`.
- Four audience keys exist (`server/internal/audienceview/audienceview.go:72`:
  decision, management, delivery, validation).
- Nothing models a second audience: no column, no table, no client control
  (a case-insensitive search for "secondary" over `server/` and `client/lib/`
  finds only UI button and theme-token names).

## Fix prompt

Red first: a store test that attaches a secondary audience to a Commit and
reads it back fails today. Smallest slice: a nullable `secondary_audiences
text[]` on `prd_commits`, validated against `audienceview.Valid`, never equal to
the primary; the commit dialog offers optional extra chips; the audience views
stay one-primary-at-a-time. Attaching a secondary grants no permission (an
audience view already grants none), and the Coach may recommend only the
primary.
