# 2026-07-26 — Roadmap canvas: adversarial-verify follow-ups (Workstream D)

Findings from the adversarial-verify workflow run for collab
[T9 / Workstream D](../changelogs/2026-07-26-roadmap-canvas-workstream-d.md)
(6 dimensions → per-finding refutation → synthesis; 3 confirmed, 1 refuted).
One confirmed finding was fixed in-line; two pre-existing, out-of-scope items
are tracked here.

## FIXED — popover trapped under the sibling below (silent reparent)

Each `.rm-card` wrapper's inline `transform` makes it its own stacking context,
so a card's add-child/edit popover (`z-20`) could not escape it. The wrappers are
`z-auto` siblings painted in DOM order (`tidyTree` post-order), so the sibling
below painted over the upper card's overflowing form: its Add button was
unclickable and — worse — a press-slip there armed a reparent-drag on the lower
card (the `closest()` button/input guard walks the *lower* card's DOM and can't
see the upper card's form), a **silent data mutation**. Keyboard Enter still
submitted (autofocus) and bottom-half cards open upward and work, which is why it
was medium, not high.

Pre-existing (the placement transform predates T9; the glide's
`transition: transform` does not change stacking) — **not** a T9 regression.

**Fix (applied, one line):** `.rm-card:focus-within { z-index: 1; }` in
`RoadmapCanvas.vue`. Popovers autofocus and hold focus while open, so the hosting
card is lifted above its `z-auto` siblings for add-child and edit, both
directions, with no JS. Verified live: `elementFromPoint` at the Add button's
center now returns the button (was the sibling), a real non-force click created a
node, and delete restored the pristine tree.
**Ceiling:** tabbing fully out of the popover sinks it — acceptable, it is no
longer the target; if it ever bites, key `z-index` off an emitted open state.

## OPEN — no keyboard path to directional pan (low)

The viewport (`role="tree"`, `tabindex="-1"`) is focusable but inert; `usePanZoom`
wires only pointer + wheel. A keyboard-only user on the full route can fit/zoom
(center-anchored) but cannot steer to off-center regions; the minimap is honestly
pointer-only (`role="img"`). Node *controls* stay Tab-reachable (cards have real
buttons with a group-focus-within reveal), so this is visual-navigation only —
not functional access. Pre-existing canvas gap the minimap merely mirrors.

**Fix prompt:** add an arrow-key handler on the focused viewport calling
`panBy()` (already exists for wheel); that gives keyboard users directional
navigation and makes a keyboard-operable minimap unnecessary. Leave
`role="img"` as-is.

## OPEN — pre-existing motions not reduced-motion-gated (low)

The Workstream-D glide is correctly gated, but two older motions in the same file
are not: the loading skeleton's `animate-pulse` (~L373) and the progress fill's
`transition-all` (~L467). Tailwind (repo is ^3.4) does not auto-disable these, and
there is no global reduced-motion reset — so a reduce-motion user gets glide-off
but pulse/width motion still on (WCAG 2.3.3 inconsistency). Pre-existing
Workstream A/B code.

**Fix prompt:** `motion-reduce:animate-none` on the skeleton div and
`motion-reduce:transition-none` on the progress fill. Two native utilities, no
Workstream-D code touched.

## Refuted (for the record)

The passthrough `pages/projects/[id].vue` is redundant with Nuxt folder-promotion
(a `[id]/` folder with no matching `[id].vue` auto-generates the same parent) —
factually true, but routing works today (verified live) and the only failure
scenario is a future dev ignoring the comment that forbids exactly that. A
documented design choice, not a defect. See the
[routing learning note](../learning/2026-07-26-nuxt-dynamic-route-parent-shadowing.md).

Related: the `RoadmapCanvas.vue` over-350-line split is tracked separately in
[2026-07-25-roadmap-canvas-modularity.md](2026-07-25-roadmap-canvas-modularity.md).
