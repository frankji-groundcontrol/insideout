# Persona grounding labels are promised but not built

Open, recorded 2026-10-11 (review-3 wave 4, finding CQ-02; slice 4 of the
[promise-gap set](2026-10-11-product-promise-gaps.md)).

## What PRODUCT.md promises

> "Personas are prioritized as core users, key participants, and later
> users." ([PRODUCT.md](../../PRODUCT.md):124-125)
>
> "Every persona is labeled as grounded in real evidence, synthesized from
> evidence, or an assumption. A plausible persona must never masquerade as a
> researched one." (PRODUCT.md:132-134)

## What exists today

- Personas are the free-text `users` section: `client/lib/api/
  models_prd.dart:39` lists the eight fixed section keys — no priority
  field, no grounding label.
- "persona" appears in code only as section labels and audience prose
  (`server/internal/export/render.go:20`,
  `server/internal/audienceview/audienceview.go:39`,
  `client/lib/l10n/en_prd.dart:11`); no structured persona machinery in the
  store or API (checked 2026-10-11).

## Slice order

4 of the set — persona hypotheses hang off Product Branch (slice 1), so the
labeling lands once branches can hold competing persona sets.

## Fix prompt

Red first: a model round-trip test that a persona carries a priority
(core/key/later) and a grounding label (grounded/synthesized/assumption)
fails. Smallest slice: an optional structured persona list on the PRD
version (name, one-line path, priority, label) beside the free-text
section — sections JSON first, a child table only if the JSON stops
scaling; render the labels in the management audience view. The label is
human- or coach-set with the evidence pointer; a plausible persona must
render visibly un-researched.
