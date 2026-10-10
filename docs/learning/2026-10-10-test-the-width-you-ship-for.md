# 2026-10-10 — Test the width you ship for, and assert rectangles

## Context

The roadmap header and the PRD page were broken at 390px (toggles cut off at
the left edge, headings breaking mid-word, a title field squeezed to nothing)
while a green suite of widget tests covered both pages
([BUG-025](../issues/2026-10-10-bug-025-phone-layouts-and-copy-found-by-the-web-walk.md)).
Writing the tests that should have caught it ran into three traps:

- `tester.binding.setSurfaceSize(Size(390, 844))` resizes the layout surface
  but leaves `MediaQuery` at 800x600. A page that branches on
  `MediaQuery.sizeOf(context).width` kept rendering its wide layout, so the
  "narrow" test passed against the broken page.
- A `Stack` clips instead of raising a `RenderFlex` overflow. The toggles sat
  119px left of the screen and no error was thrown; only their rectangles
  showed it.
- The test font (Ahem) is one em per glyph, wider than the real font, so a
  layout that fits at 390px in a browser can overflow by a few pixels in the
  test.

## Lesson

- Resize the view, not the surface:
  `tester.view.devicePixelRatio = 1; tester.view.physicalSize = size;
  addTearDown(tester.view.reset);`.
- Assert geometry the user would see: `tester.getRect(finder)` against the
  viewport (`left >= 0`, `right <= width`), field width against a floor, one
  line of text via `getBoxesForSelection`. "No overflow exception" proves
  nothing when the parent clips.
- If the test font forces a wider phone width than the real one (420 for 390),
  say so in a comment, keep it below the breakpoint, and still look at the
  real width in a real browser once: the test proves the layout rule, the
  browser proves the pixels.
- Red-check every such test against the page before the fix, with the number
  that fails in the commit message (here: toggle at left -119.4, field width
  0, coach header still at y = 704 after the drag).

## When to apply

- Any page that branches on a breakpoint or reads `MediaQuery` for layout.
- Any "does it fit" claim for a control row, heading, or form field.
- Any fix for a scroll or gesture problem: assert the scroll offset or the
  position of something that should have moved, not that the call returned.
