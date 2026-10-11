# The facts trust layer (confirm/correct/remove) is promised but not built

Open, recorded 2026-10-11 (review-3 wave 4, finding FULL-03; the full path
parked from
[AIDEV-09](../review/2026-10-10-multirole-3/01-findings-register.md);
slice 5 of the
[promise-gap set](2026-10-11-product-promise-gaps.md)).

## What PRODUCT.md promises

> "The internal evidence experience shows:
> - the Coach's interpretation;
> - the supporting user words or source;
> - whether it is confirmed, synthesized, assumed, or still to validate;
> - small actions to confirm, correct, remove, or discuss it."
> ([PRODUCT.md](../../PRODUCT.md):182-187)

## What exists today

- Recording is agent-side only: facts live in the conversation's meta JSON
  (`server/internal/agent/ledger.go:62,81` — `lm.Facts`) and are re-derived
  into every system prompt (`ledger.go:184-193`).
- No route or client surface touches facts: the API inventory
  (`server/internal/api/server.go:98-110`) has no facts route; the PRD desk
  shows none. AIDEV-09 additionally found the stored status is set
  `attested` by the model's own `record_fact` (`ledger.go:239-244`) — the
  parallel [`w4-attestation-honesty`](../review/2026-10-10-multirole-3/40-remediation-plan.md)
  slice makes user-facing copy say
  "model-recorded"; the confirm/correct/remove API itself lands here.

## Slice order

5 of the set — governs what the outward slices (2 and 3) may expose:
"Raw quotes and private conversation remain internal by default"
(PRODUCT.md:189-190).

## Fix prompt

Red first: an API test that a Driver can list a PRD's candidate facts and
confirm, correct or remove one fails. Smallest slice: a read-only facts
surface first (list the recorded facts with their status on the PRD desk,
straight from the conversation meta), then the three writes that flip or
withdraw a status — each write recording who and when. Nothing silently
re-enters prompts or views without its status.
