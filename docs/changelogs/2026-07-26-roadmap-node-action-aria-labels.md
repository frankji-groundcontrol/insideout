# 2026-07-26 — Roadmap node actions get explicit `aria-label`s

A picky frontend audit found the roadmap canvas's five per-node action buttons
(status-cycle, Break down with AI, Add sub-task, Edit, Delete in
`app/src/components/roadmap/RoadmapCanvasNode.vue`) relied on `title` alone for
their accessible name. `title` is only a last-resort source in the ARIA AccName
algorithm and its announcement is inconsistent across screen-reader
configurations; the canvas's own toolbar buttons already pair `:title` with
`:aria-label`.

## Change

Added `:aria-label` mirroring the existing `:title` on all five node action
buttons, matching the toolbar pattern (`RoadmapCanvas.vue` zoom/fit/open
buttons). The i18n keys were already present and shared.

## Verification

- `pnpm vitest run src/components/__tests__/RoadmapCanvasNode.test.ts` — 4/4 pass.
- Live browser (EN + dark): every node's five action buttons now expose an
  English `aria-label` (To do / Break down with AI / Add sub-task / Edit /
  Delete / In progress) identical to their `title`.
