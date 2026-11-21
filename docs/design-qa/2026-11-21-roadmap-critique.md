# 2026-11-21 — Roadmap views: impeccable critique

## Comment

> "use https://github.com/pbakaus/impeccable to review"

The owner asked for an impeccable critique after the
[ragged band rows](./2026-11-21-roadmap-band-rows.md) fix. It covered the
Progress, Bands, Canvas and Mine views and an opened branch, on the Spark
demo data ([demo data](../usage/demo-data.md)). Viewports were 1345x853 and
390x844, in both themes.

## Method

Two isolated reviewers ran, as the critique requires:

- **A design review:** Nielsen heuristics, cognitive load and personas.
- **Detector and browser evidence.** The impeccable detector reads no `.dart`
  files. In the live page, CanvasKit draws everything into one canvas, so
  its DOM rules found only two false positives on Flutter engine elements.
  The evidence came from the Flutter semantics tree, screenshots and sampled
  pixel contrast instead.

**Score:** 18/40 (Poor). There are no P0 findings and three P1 findings.

## Findings

| Severity | Finding | Proposed fix |
| --- | --- | --- |
| P1 | **Emphasis is backwards.** The cobalt slab fills Next in Progress and Later in Bands, while Now stays grey. The [design system](../../DESIGN.md) gives cobalt-500 to "Primary action, focus, Now". | Neutral slabs for every band, with Now marked alone. Show pressure as a shade chip. |
| P1 | **Bands' Later is a dump.** It holds 20+ cards: roots, branches, leaves, and done and validated work. | Move finished work to a Done band. Group the rest by top-level branch, or show branches only. |
| P1 | **Canvas with one root is one column.** A built project has one root, so Canvas is a 300x448 box in a 1345px field: 76.5% of the width is empty and cards are cut off at the box edge. | With one root, its children become the bands and the root becomes a header strip. |
| P2 | **Small reading window.** The pinned header takes 190px on desktop and 317px (37.6%) on a phone. The dock covers the Next row at rest. | Scroll the header with the content. Give the dock a backing plate. |
| P2 | **The grid ignores the 1120 content width.** 213px is left dead to the right of every band. On a phone, the band label takes 26% of the lane. | Constrain the band and let the cards fill each row. Put the label above the cards on a phone. |
| P3 | **Light-theme muted text fails AA on lanes.** `fgMuted` `#6B7280` on `#EEF2FF` measures 4.32:1, below 4.5. | Use a darker muted token on tinted lanes. |
| P3 | **Repeated and empty chrome.** Presence chips repeat one user per session. An empty "Needs deadline" band takes 60px. Progress prints "normal" pressure, which Bands hides. An opened branch is still titled "ROADMAP". | Dedupe presence by user, hide empty bands, hide normal pressure everywhere, title an opened branch with its name. |

Two findings were already fixed on the same branch, `fix/spark-walk`: done
cards no longer show "overdue", and a build no longer seeds a branch as in
progress.

## Status

The findings above wait on the owner's choice of which to take.

## Files

- [`client/lib/features/project/roadmap_progress_view.dart`](../../client/lib/features/project/roadmap_progress_view.dart)
- [`client/lib/features/project/roadmap_bands.dart`](../../client/lib/features/project/roadmap_bands.dart)
- [`client/lib/features/project/roadmap_canvas.dart`](../../client/lib/features/project/roadmap_canvas.dart)
- [`client/lib/features/project/roadmap_band_scaffold.dart`](../../client/lib/features/project/roadmap_band_scaffold.dart)
