# 2026-07-24 — Roadmap: tree on a canvas

Status: implemented + hardened (single-player). An adversarial review landed
eight fixes on 2026-07-25 — see the
[changelog](../changelogs/2026-07-25-roadmap-canvas-hardening.md); the
collaborative follow-up is
[2026-07-24-roadmap-canvas-collab.md](2026-07-24-roadmap-canvas-collab.md).
Owner directive: "roadmap is an important feature, it should be
a tree in a canvas."

## Context

The roadmap (branched-tree per project: `locked → pending → in_progress →
done`, AI expand, `move` reparent API) currently renders as an indented nested
list (`RoadmapTree.vue` + `RoadmapNodeItem.vue` with CSS hairline connectors).
It is the product's central object — "every project is a scroll; work branches"
— and it deserves a spatial stage: a real tree laid out on a pan/zoom canvas.

## Decisions

- **Hand-rolled canvas, no new dependency.** SVG edges + absolutely-positioned
  HTML node cards inside a transformed world container. VueFlow/d3 would fight
  the Ink & Seal token system for marginal gain; layout + pan/zoom is ~300
  lines of owned code that stays fully on-brand.
- **Horizontal tidy tree** (roots at left, branches flow right) — reads as
  progression, matches the 分枝 metaphor, suits a wide canvas.
- **One component, two placements.** `RoadmapCanvas` fills the project page's
  roadmap section (fixed ~560px shell) and a new full-viewport route
  `projects/[id]/roadmap.vue`. The indented list is deleted, not kept.
- **Drag to reparent** on the existing `move(nodeId, parentId, position)` API;
  drop on empty canvas = move to root. Cycle-guard: never onto own descendant.
- **Wheel = pan, ctrl/cmd+wheel = zoom at cursor** (map convention), so an
  embedded canvas never hijacks page scroll unexpectedly.

## Files

- NEW `app/src/utils/tidyTree.ts` — pure layout: flat nodes → `{id,x,y}` +
  edge pairs. Unit-tested.
- NEW `app/src/composables/usePanZoom.ts` — pan/zoom state machine.
- NEW `app/src/components/roadmap/RoadmapCanvas.vue` — viewport, edges,
  toolbar (zoom/fit/add-goal/progress), drag-reparent, CRUD wiring.
- NEW `app/src/components/roadmap/RoadmapCanvasNode.vue` — node card (status
  seal cycle, title/desc, hover actions, inline edit, add-child form).
- NEW `app/src/pages/projects/[id]/roadmap.vue` — full-viewport canvas route.
- EDIT `app/src/pages/projects/[id].vue` — swap list for canvas + full-view link.
- EDIT i18n `en-US.ts` / `zh-CN.ts` — `roadmap.canvas.*` keys.
- DEL `RoadmapTree.vue`, `RoadmapNodeItem.vue`.
- NEW `app/tests/tidyTree.test.ts`.

## Design rules (Ink & Seal)

- Edges: cubic bezier, hairline; tinted by **child** status — neutral for
  locked/pending, seal for in_progress, sage for done. Meaning, not decoration.
- Node card: raised celadon, hairline stroke, `rounded-card`, flat at rest
  (Flat-By-Default); the status chop is the only seal element per card.
- Canvas ground: page ground with a faint sunken well; no dot-grid costume.
- One Seal Rule holds — the wordmark/toolbar adds no new vermilion fills.
- Type: card titles in the sans (serif is headings-only); no eyebrow spam.
- a11y: `role="tree"`/`treeitem` semantics, real buttons inside cards, visible
  focus rings, `prefers-reduced-motion` kills transitions, keyboard tab order
  follows layout order. Spatial arrow-key nav: follow-up, not this pass.
- Bilingual EN/中文 for every new string.

## Verification

- `app/tests/tidyTree.test.ts` green under `pnpm test` (with the existing 20).
- `npx nuxi typecheck` clean; `pnpm build` clean.
- Live (real backend + DB, random high ports, proxy in sync): render tree,
  pan/zoom/fit, add goal/child, cycle status, inline edit, delete, drag
  reparent (to node + to root), AI expand, light + dark, EN + 中文.
- Adversarial review workflow (correctness / design-system / interaction);
  fix findings; then changelog + index updates.

## Checklist

- [x] Plan filed (this doc)
- [x] tidyTree + usePanZoom + unit test
- [x] RoadmapCanvas + RoadmapCanvasNode
- [x] Route + project-page swap + i18n + delete old list
- [x] tests / typecheck / build green
- [x] live browser verification (both themes, both locales)
- [x] adversarial review + fixes
- [x] changelog + doc indexes
