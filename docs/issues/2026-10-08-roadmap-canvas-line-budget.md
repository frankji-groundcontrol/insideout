# 2026-10-08 — `client/lib/features/project/roadmap_canvas.dart` over the ~350-line budget

> **Resolved 2026-10-10.** `roadmap_canvas.dart` is 200 lines; the minimap is `roadmap_minimap.dart` (184)
> and the dash painter `roadmap_painters.dart` (48). The split was a pure move
> (`a2dfa9d`); the minimap's open fidelity fixes landed on top of it (`53e509d`:
> localized marks, labels lifted to the 12px floor). See
> [client file splits](../changelogs/2026-10-10-close-open-gaps/client-file-splits.md).
>
> The original record follows, unedited.

`roadmap_canvas.dart` is 400 lines: the pan/zoom canvas shell, band
layout, the pointer-only minimap, and the dash painter in one file. The
oversize predates the
[block-craft plan](../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md)
(400 lines at HEAD `a762ca8` as well); the Wave 6 sweep touched it
(minimap `InkWell` → `PressableBlock`, D7) without changing the budget,
and the Wave 7 review flagged that no modularity issue had been filed —
`project_page.dart` and the l10n tables both have one
([project_page](2026-10-08-project-page-line-budget.md),
[l10n](2026-10-08-l10n-tables-line-budget.md)); this is the third of
the trio.

## Why not split now

The plan explicitly deferred structural splits out of the verification
gate, and the file is cohesive (one route, one paint pipeline). It also
carries three open fidelity findings (9px/10px minimap type,
hardcoded-English minimap labels) that will touch the same code — the
split and those fixes should land in one pass, not collide.

## Target structure

- `roadmap_canvas.dart` — the canvas shell: pan/zoom state, layout, hit
  testing (under ~250 lines).
- `roadmap_minimap.dart` — `_Minimap`, `_MinimapMark`, cursor labels;
  the localization and 12px-type fixes from the
  [fidelity follow-ups](2026-10-08-block-craft-fidelity-followups.md)
  land here.
- `roadmap_painters.dart` — the dash/edge painters.

## Bounded fix prompt

"Split `client/lib/features/project/roadmap_canvas.dart` (400 lines)
into the shell plus `roadmap_minimap.dart` and `roadmap_painters.dart`
per
[docs/issues/2026-10-08-roadmap-canvas-line-budget.md](2026-10-08-roadmap-canvas-line-budget.md).
No behavior changes: the roadmap canvas tests (paint fingerprints,
minimap shape channel) pass unmodified. Gate: `cd client && flutter
analyze --no-fatal-infos && flutter test && dart format --output=none
--set-exit-if-changed lib test`. Every touched file ends under ~350
lines. Record the split in a changelog entry and close this issue."
