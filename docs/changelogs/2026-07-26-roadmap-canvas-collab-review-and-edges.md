# 2026-07-26 — Roadmap canvas: review mode + edge re-semantics (collab T7–T8)

Frontend-only lanes of the [collaborative model
plan](../plans/2026-07-24-roadmap-canvas-collab.md): a truly read-only
review/present mode and localized freshness (Workstream B, minus the schema
migration), and the edge re-semantics (Workstream C). No backend, no schema,
no operator action.

## What changed

**B1 — review/present mode ([RoadmapCanvas.vue](../../app/src/components/roadmap/RoadmapCanvas.vue),
[RoadmapCanvasNode.vue](../../app/src/components/roadmap/RoadmapCanvasNode.vue)).**
A toolbar eye toggle (on both the embedded map and the full route) flips a
`reviewing` ref; `?review=1` seeds it on mount as a shareable view-only
deep-link (a view state, *not* an access boundary — every invite-code member
can toggle it off). Enforcement is real, not cosmetic: the status seal is
`:disabled`, and every other mutation surface — hover edit/delete/AI-expand
buttons, the edit + add-child popovers, the add-goal button/form/empty-state,
and drag-to-reparent (`onCardPointerDown` early-returns) — renders `v-if`-off
so it leaves the DOM and the tab order. A quiet, non-vermilion "Reviewing ·
read-only" chip persists while active (One Seal Rule). Zoom/pan/fit/open-full
stay enabled (read operations).

**B2 — i18n freshness ([useTimeAgo.ts](../../app/src/composables/useTimeAgo.ts),
both locales).** The card shows "updated X ago" via a new locale-aware
`useTimeAgo()` composable backed by `time.*` keys (today / yesterday / Nd /
Nmo / Ny) in `en-US` + `zh-CN`, so the label follows the active locale; the
hard-coded-English `timeAgo()` is retained only for the non-setup workspace
call site. A branch untouched >7 days dims its timestamp ("quiet").

**C1 — prospective edge during drag.** While reparenting, a dashed
`stroke-subtle` hairline draws from the drop target to the drag ghost inside
the world SVG (cursor converted via `toWorld`), previewing the structure the
gesture creates; nothing renders for a move-to-root.

**C2 — neutral hairline edges + hot-branch emphasis.** All four status tints
collapse to a single neutral `stroke-subtle` at 1.5px (status now lives only
on the seals, restoring vermilion rarity). A branch is "hot" when any
descendant was touched within 7 days; hot edges step up to `stroke-strong` —
and, because that token swap is a perceptual near-no-op on the dark ground
(Δ≈13/channel), bump to 2.5px in dark mode.

## How it was verified

- **Deterministic gates:** `pnpm test` → 44/44 pass (14 files); `npx nuxi
  typecheck` clean.
- **Live** (real backend on :54363 behind the :54362 proxy, pairing kept in
  sync): across light/dark × 中文/EN with DOM/CSS assertions — review ON =
  chip + `aria-pressed=true` + add-goal removed + 14/14 seals disabled + 0
  hover-action buttons in the DOM; toggle OFF restored them; freshness
  rendered "昨天更新" (zh) / "updated yesterday" (en); light hot edge =
  `stroke-strong` @1.5px, dark hot edge = @2.5px (the specced bump).
- **Honest gaps:** the C1 prospective edge was verified by code + compiled
  scoped style + build only, not a live drag simulation; the neutral
  (non-hot) edge is unobservable on the seed tree (all 11 nodes recent → all
  hot) but shares the same verified style block.

## Operator notes

None — frontend-only. B3 attribution (the `created_by`/`updated_by`
migration) remains deferred pending an explicit go; pre-migration rows will
render an "unknown" initial once it lands.
