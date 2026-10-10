# Scoped external feedback invitations are promised but not built

Open, recorded 2026-10-11 (review-3 wave 4, findings CQ-02 and SEC-04;
slice 2 of the [promise-gap set](2026-10-11-product-promise-gaps.md)).

## What PRODUCT.md promises

> "Early users and advisers can receive a scoped feedback invitation without
> joining the Workspace. Feedback retains author, time, and context and
> becomes a suggestion or proposed branch rather than a silent mainline edit.
> External respondents see their own feedback by default, not everyone
> else's; the team sees the complete set, and the Decision Owner can publish
> a deliberate synthesis." (PRODUCT.md:253-259)

## What exists today

- No feedback route in the API inventory: every `register*` call in
  `server/internal/api/server.go:98-110` is auth/account/work/admin — none
  handles feedback (checked 2026-10-11).
- The only invitations are workspace-signup invitations
  (`server/internal/api/admin.go:24-26`) — a different concept that grants
  membership, not a scoped read.
- The closest live mechanism is workspace membership, which grants far more
  than a scoped invitation: an early adviser must be given full membership
  or nothing.

## Slice order

2 of the set — the only parked promise carrying an explicit privacy boundary
("External respondents see their own feedback by default"), so its security
design comes before the read-only share link and the other outward slices
(SEC-04: "scoped feedback tokens next").

## Fix prompt

Red first: an authz test that a feedback token reaches exactly one committed
PRD version's view and its own feedback rows — and nothing else — fails (no
table, no route). Smallest slice: `feedback_tokens` (one committed version,
expiry) and `feedback` rows (author label, time, context, body); token
holders create feedback and read only their own; the team reads the complete
set. No account creation, no workspace membership. Feedback becomes a
suggestion/proposal, never a mainline edit.
