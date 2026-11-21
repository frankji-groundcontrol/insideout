# 2026-11-21 — Roadmap canvas: the band around a lone card

## Comment

> "the visual is problematic"

The owner's screenshot shows the roadmap's Canvas view with one node.

## What was wrong

1. **The card sat offset inside its band.** The band reserved a 40px fold
   gutter on every row, even when nothing in it could fold. A lone leaf
   card was pushed right, and its block shadow ran past the band's edge.
2. **The band ran about 90px past its last card.** Its `ListView` had no
   padding of its own, so it inherited the page's `MediaQuery` bottom
   padding (the dock's height).
3. **A minimap of one column sat in the header,** with a full-width rule
   cutting under it. With a single band there is nothing for the minimap to
   jump between.

## Resolution

- **The band:** reserves the fold gutter only when it holds a branch, has
  explicit 8px padding, and spaces cards from the top.
- **The minimap:** shows only when there are at least two bands.
- **The rule:** removed.
- **Pinned by:**
  [`roadmap_canvas_band_layout_test.dart`](../../client/test/features/project/roadmap_canvas_band_layout_test.dart).
  It failed first at 40px and 90px.

## Files

- [`client/lib/features/project/roadmap_canvas.dart`](../../client/lib/features/project/roadmap_canvas.dart)
- [`client/lib/features/project/roadmap_page_views.dart`](../../client/lib/features/project/roadmap_page_views.dart)
- [`client/test/features/project/roadmap_canvas_band_layout_test.dart`](../../client/test/features/project/roadmap_canvas_band_layout_test.dart)
