# 2026-07-25 — Roadmap canvas: single-player hardening pass

The hand-rolled pan/zoom roadmap canvas (plan
[2026-07-24 — Roadmap: tree on a canvas](../plans/2026-07-24-roadmap-canvas.md))
shipped as a polished single-player editor. This pass closes it out: an
adversarial review (correctness / interaction / a11y / design-system lenses,
findings independently verified) confirmed eight defects, all fixed here; the
single-player canvas is now **implemented + hardened**. The collaborative
follow-up is tracked separately in
[2026-07-24 — Roadmap canvas: collaborative model](../plans/2026-07-24-roadmap-canvas-collab.md).

## What changed

Eight review findings, all verified real and fixed:

- **H1 — fit left the top node under the toolbar.** `fitTo` took a single
  symmetric pad, so a fitted tree's top row landed beneath the floating
  toolbar band and its hover actions were unreachable. `fitTo` now takes
  per-side insets (`app/src/composables/usePanZoom.ts`); the canvas passes a
  96px top inset (`TOOLBAR_INSET`) measured against the live toolbar bottom.
- **M1 — a failed load looked like an empty roadmap.** `load()` had no `catch`,
  so a network failure fell through to the empty state and a silent
  `addGoal()`. A `loadError` ref now drives a distinct error state with a
  Retry (`RoadmapCanvas.vue`); `addGoal()` failure keeps the draft and
  surfaces the banner.
- **M2 — the drag-reparent gesture ignored `pointercancel`.** An OS-cancelled
  pointer (palm rejection, system gesture) could commit a reparent. `endDrag()`
  now also detaches on `pointercancel` and drops the gesture without moving.
- **M3 — mid-stream Anthropic rate-limit never started the backoff countdown.**
  `ANTHROPIC_RATE_LIMIT` only ever arrives as a mid-stream SSE `error` event
  whose HTTP status is already gone (`undefined`), but `toApiError` gated it on
  `status === 429`, so the coach's retry countdown never fired and Send stayed
  enabled into an upstream limit. It now matches on code alone
  (`app/src/services/api/http.ts`).
- **M4 — edit / add-child popovers clipped off the bottom edge.** Bottom-half
  cards opened downward into the `overflow-hidden` viewport. The node now
  receives `openUpward` (flipped when the card sits in the layout's bottom
  half) and renders the popover above (`RoadmapCanvasNode.vue`).
- **M5 — node actions were invisible to keyboard and touch.** The hover action
  bar was `opacity-0` until `group-hover`, which never fires for keyboard
  focus or touch. It now also reveals on `group-focus-within` and, via
  `[@media(hover:none)]`, always on touch devices.
- **M6 — focus dropped to `<body>` after edit / delete.** Saving or cancelling
  an edit now returns focus to the card; deleting (which unmounts the card)
  moves focus into the `role="tree"` viewport first. The card and viewport are
  `tabindex="-1"` focus targets with a visible focus ring.
- **L1 — node mutations swallowed their failures.** All four CRUD handlers now
  `catch` and `emit('error', msg)` to the canvas banner (the `// ponytail:`
  minimal surface for collab A3's failure feedback — see the deferred-toast
  issue below), so no mutation fails silently.

Supporting i18n: `common.retry` added to both `en-US` and `zh-CN` (locale
parity test stays green).

## What was *not* actioned (and where it lives)

The same review raised interaction-polish suggestions that overlap the
collaborative plan's later workstreams and were intentionally deferred there
rather than patched into the single-player canvas: spatial arrow-key
navigation and animated layout transitions / minimap are Workstream D
([collab plan](../plans/2026-07-24-roadmap-canvas-collab.md)), and the
canvas's own design rules already earmark arrow-key nav as a follow-up. Two
real deferred items are recorded as dated issues rather than fixed here:

- the build-from-PRD handler runs the LLM `PlanMVP` call before the live-count
  409 guard, wasting the call on a non-empty roadmap —
  [2026-07-25 issue](../issues/2026-07-25-build-from-prd-runs-llm-before-conflict-guard.md);
- collab A3's specced stacked toast + delete descendant-count confirm shipped
  as a simpler in-canvas error banner + static confirm this pass —
  [2026-07-25 issue](../issues/2026-07-25-canvas-failure-feedback-is-a-banner-not-the-specced-toast.md).

## Verification

- `cd app && pnpm test` — 44/44 (includes the D9 sparse-PATCH contract in both
  happy-dom and nuxt envs, and the locale-parity check for `common.retry`).
- `npx nuxi typecheck` clean; `pnpm build` clean.
- Live (real backend + DB, random high ports, proxy in sync): authenticated on
  the 14-node demo project and measured the fit geometry — viewport 560px,
  toolbar bottom 89px, fitted top node at 97px (8px clearance; before the fix
  it sat near 48px, buried under the toolbar). The Fit button recovers the
  same geometry after manual pan/zoom. Light theme verified in-browser; locale
  parity covered by the test.

## Operator action

None. This pass is frontend-only with no schema or migration change. (The
collaborative plan's B3 attribution migration remains a future operator step
under its T6.)
