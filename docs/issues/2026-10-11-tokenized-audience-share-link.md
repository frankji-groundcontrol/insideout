# No way to share an audience view with an outside reader

Open, recorded 2026-10-11 (review-3 wave 4, finding USER-04; slice 3 of the
[promise-gap set](2026-10-11-product-promise-gaps.md)).

## What PRODUCT.md promises

> "Early users and advisers can receive a scoped feedback invitation without
> joining the Workspace." (PRODUCT.md:254-255)

The validation leg of the product loop (PRODUCT.md:26) assumes an outside
reader can inspect a version's view.

## What exists today

- The four audience projections are auth-only app routes:
  `client/lib/router.dart:48` maps `'/prd/:id/view'` to the in-app
  `AudienceViewPage`; server side `GET /api/v1/prds/{id}/view` is
  requireAuth-wrapped (`server/internal/api/prds.go:19`).
- No tokenized public route exists (no `share`/`token` route in the API
  inventory, checked 2026-10-11); the only outward path is the manual
  export page — markdown download or print
  (`client/lib/features/prd/export_page.dart:44,71`).

## Failure

A Driver finishes a PRD and wants their boss to read the Decision view: the
only paths are exporting markdown and emailing it, or creating the boss an
account — the validation leg has no outside-reader surface.

## Slice order

3 of the set — the read-only outward leg, after the feedback invitations
(the write leg) whose privacy boundary sets the pattern for outward tokens.

## Fix prompt

Red first: a server test that a GET by share token returns the audience
projection for one committed version without a session fails. Smallest
slice: a `share_tokens` row (PRD version + audience + expiry) generated from
the versions page; a public GET that reuses the existing server-side
`/prds/{id}/view` projection assembly, scoped to that one version and
audience; read-only — no feedback write, no section edits.
