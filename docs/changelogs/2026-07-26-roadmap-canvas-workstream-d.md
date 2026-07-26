# 2026-07-26 — Roadmap canvas: Workstream D (transitions + sibling bands + minimap) — collab T9

Closes the collaborative-canvas plan
([2026-07-24](../plans/2026-07-24-roadmap-canvas-collab.md)): Workstream D was
its last open workstream. Four frontend changes, all in
[`RoadmapCanvas.vue`](../../app/src/components/roadmap/RoadmapCanvas.vue) and
its neighborhood, plus one routing correctness fix they depended on and one
stacking fix the adversarial pass surfaced.

## What changed

- **Cards glide, don't teleport.** Wrappers are keyed by node id
  (`:key="p.node.id"`), so each card keeps its DOM node across a data reload and
  a CSS `transform` transition (`260ms cubic-bezier(0.22,1,0.36,1)` on
  `.rm-card`) fires only on genuine repositions — fresh mounts appear in place.
  Gated on the native `@media (prefers-reduced-motion: reduce)` query (a CSS
  concern, unlike the motion-v entrance swaps that `useReducedMotion` gates;
  the `CoachPanel` is the in-repo precedent).
- **Sibling bands.** `siblingBands(placed)` groups cards by `parentId` (skips
  roots and single-child groups) and draws a whisper of a panel behind each
  parallel track — neutral `surface-raised` fill + `stroke-strong` hairline,
  light and dark. **One Seal Rule holds:** bands carry no vermilion; only status
  seals are colored.
- **Minimap** ([`RoadmapMinimap.vue`](../../app/src/components/roadmap/RoadmapMinimap.vue)).
  Full-route only (`v-if="!embedded"`), a 176×120 overview that renders the whole
  tree, frames the current viewport, and click/drags to `centerOn`. Honest
  `role="img"` + localized `aria-label` (小地图 / Minimap); deliberately
  pointer-only — see follow-ups.
- **Routing correctness fix (load-bearing).** The full `/projects/:id/roadmap`
  route was silently rendering the *embedded* project page instead of the canvas,
  because `pages/projects/[id].vue` is the **parent** of the `pages/projects/[id]/`
  folder and, without a `<NuxtPage />` outlet, swallowed its children. Fix: `[id].vue`
  is now a bare outlet; the project-detail page moved to `[id]/index.vue`
  (layout `default`), the canvas route keeps `canvas`. Per-child layouts still
  resolve — the child's `definePageMeta({ layout })` overrides the parent's.
  See the [learning note](../learning/2026-07-26-nuxt-dynamic-route-parent-shadowing.md).
- **Popover stacking fix (surfaced by the adversarial pass, applied).** Each
  wrapper's inline `transform` makes it its own stacking context, so a card's
  add-child/edit popover could not rise above the sibling painted just below it:
  the form overflowed under that sibling, its Add button was unclickable, and a
  press-slip there armed a reparent-drag — a **silent data mutation**. One line,
  `.rm-card:focus-within { z-index: 1; }`, lifts the focused card above its
  `z-auto` siblings (popovers autofocus and hold focus while open). Covers
  add-child and edit, both directions, no JS. Pre-existing canvas debt, not a T9
  regression — recorded in the
  [adversarial follow-ups issue](../issues/2026-07-26-roadmap-canvas-adversarial-followups.md).

## Verification

Live, real backend + DB, random high ports (proxy in sync), light + dark ×
embedded + full:

- **Routing:** full route → canvas + minimap, no embedded-only "open full" link;
  index route → project page with the embedded canvas and no minimap. No regression.
- **Glide mechanism, proven by its two premises:** (1) a keyed wrapper's DOM node
  survives a full nodes reload; (2) the `260ms` transform transition is present
  and reduced-motion-gated. A persistent element with a transform transition
  animates on transform change — deterministic browser behavior.
  **Honest limitation:** a clean frame-by-frame capture of a card mid-glide was
  *not* obtained — the attempts were blocked by the popover-overlap defect (now
  fixed), one transient backend `502` (port verified listening; not a code
  defect), and a structural edit (a grandchild under a leaf) that correctly does
  not reposition siblings. The glide is verified by mechanism, not photographed.
- **Minimap:** renders the full tree, frames the viewport, click re-centers; the
  frame tracks zoom/pan (prior pass).
- **CRUD on the canvas:** add-child created nodes (real, non-force click after the
  stacking fix) and delete-branch removed them; the demo project was restored to
  its pristine 3 nodes.
- `pnpm test` **52/52** and `npx nuxi typecheck` clean (`EXIT=0`).

An adversarial-verify workflow (6 dimensions → per-finding refutation →
synthesis) confirmed three findings and refuted one: the routing fix
(*passthrough `[id].vue` is redundant with Nuxt folder-promotion but correct and
documented — not a defect*), the stacking defect (fixed here), and two
pre-existing out-of-scope items tracked in the
[follow-ups issue](../issues/2026-07-26-roadmap-canvas-adversarial-followups.md).
Two of the six reviewers (perf, canvas-correctness) died on inference-gateway
`524` timeouts; the four surviving dimensions plus synthesis read the same files
and covered that ground.

## Operator notes

Frontend-only; no migration, no env change. Existing deployments pick it up on
rebuild. The routing restructure moves the project-detail page to
`projects/[id]/index.vue` — the URL is unchanged.

## Follow-ups

Tracked in the
[adversarial follow-ups issue](../issues/2026-07-26-roadmap-canvas-adversarial-followups.md):
keyboard directional pan (arrow-key `panBy()` on the already-focusable viewport)
and reduced-motion gating for the pre-existing skeleton pulse + progress-bar
transition. The `RoadmapCanvas.vue` over-350-line split is already tracked in the
[modularity issue](../issues/2026-07-25-roadmap-canvas-modularity.md).
