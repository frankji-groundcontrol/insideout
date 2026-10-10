# RoadmapCanvas.vue is a chunky mixed-responsibility file

> **Superseded 2026-10-10.** The file this record is about, `app/src/components/roadmap/RoadmapCanvas.vue`,
> no longer exists: the Nuxt app was deleted in the Flutter rewrite and no `.vue`
> file is tracked. The Flutter canvas had the same problem and was split on its
> own record ([roadmap_canvas.dart line budget](2026-10-08-roadmap-canvas-line-budget.md)).
>
> The original record follows, unedited.

**Date:** 2026-07-25
**File:** `app/src/components/roadmap/RoadmapCanvas.vue` (~407 lines)
**Status:** open — deferred split; behavior is correct and tested.

## Problem

`RoadmapCanvas.vue` holds five responsibilities in one file and has grown past
the ~350-line modularity guideline as the collab-plan tasks (T4 partial
payloads, T5 focus-refresh) landed on it:

1. data fetching + silent focus-refresh (latest-wins seq guard)
2. pan/zoom wiring + refit/toolbar-inset
3. drag-to-reparent gesture (pointer handlers, hit-test, cycle guard, ghost)
4. add-goal / toolbar state
5. the full template (world, edges, cards, toolbar, empty/error/skeleton states)

## Target structure

- `useRoadmapTree(projectId)` composable — `nodes`, `loading`, `loadError`,
  `load()`, `refresh()`, the `fetchSeq` guard, and the `visibilitychange` /
  window `focus` listeners (returns cleanup). Pure script, unit-testable
  without the DOM-heavy canvas.
- `useReparentDrag(layoutSource, { onMoved })` composable — the
  pointerdown/move/up/cancel handlers, `hitTest`, ghost state, `isDescendant`
  guard.
- `RoadmapCanvas.vue` shrinks to composition + template (~180 lines); toolbar
  markup may move to a `RoadmapToolbar.vue` child if it still reads heavy.

## Bounded fix prompt

> Extract `useRoadmapTree` and `useReparentDrag` from
> `app/src/components/roadmap/RoadmapCanvas.vue` per the target structure in
> `docs/issues/2026-07-25-roadmap-canvas-modularity.md`. Keep behavior
> identical: latest-wins refresh, silent-focus no-toast, interacted-guarded
> refit, 5px drag slop, descendant-drop guard. Verify: `pnpm test`,
> `npx nuxi typecheck`, `pnpm build`, and a live reparent + two-tab
> focus-converge walk.
