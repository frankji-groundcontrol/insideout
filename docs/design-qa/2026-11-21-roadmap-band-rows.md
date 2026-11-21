# 2026-11-21 — Roadmap bands: ragged card rows

## Comment

> "so messed up, optimize the visual"

The owner's screenshot shows the Bands view's Now band, with a red box around
the bottom edge of its first row of cards.

## What was wrong

1. **Cards in one row ended at different heights.** The band laid its cards
   out with a `Wrap`, so each card was as tall as its own content. A
   two-line title or a longer "Done when" made one card taller than the
   next, and the row's bottom edge and the gap below it read as ragged.
   Progress shares the same band scaffold and had the same fault.
2. **A compact card printed two names.** The card showed the last editor's
   name, unlabelled, under the title, and then "Owner: …" below it. With
   the same person in both, it read as one name printed twice.

## Resolution

- **Rows:** the band scaffold lays out the cards in rows of as many as fit.
  Each row is as tall as its tallest card. Each card keeps a tight width, so
  the row measures it at the width it is laid out at. On a phone the card
  narrows to the band.
- **Names:** compact (band) cards drop the unlabelled editor line. The full
  card on Canvas keeps it.
- **Pinned by:**
  [`roadmap_band_scaffold_test.dart`](../../client/test/features/project/roadmap_band_scaffold_test.dart).
  It failed first on unequal heights.

## Follow-up review

An impeccable critique of the roadmap views ran after this fix. Its findings
are in the [roadmap critique record](./2026-11-21-roadmap-critique.md).

## Files

- [`client/lib/features/project/roadmap_band_scaffold.dart`](../../client/lib/features/project/roadmap_band_scaffold.dart)
- [`client/lib/features/project/roadmap_node_card.dart`](../../client/lib/features/project/roadmap_node_card.dart)
- [`client/test/features/project/roadmap_band_scaffold_test.dart`](../../client/test/features/project/roadmap_band_scaffold_test.dart)
