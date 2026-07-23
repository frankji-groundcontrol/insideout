# Verification — what was actually checked

Design/frontend-only change; no DB or API surface touched, so the no-mocks
database gate does not apply here. Verified on the Nuxt dev server (real SSR)
in a real browser engine.

## Build / static

- `npx nuxi typecheck` — 0 errors (exit 0).
- `pnpm test` — **20/20 passed** across 10 files (vitest, happy-dom + nuxt envs).
- `pnpm build` — green (exit 0).
- impeccable `detect.mjs --json` over `index.vue`, `components/landing/`,
  `useReducedMotion.ts`, `BaseButton.vue` — clean (`[]`). (One detector flag,
  a redundant overshoot easing token in `AssemblyDiagram.vue`, was fixed during
  the build; the keyframes carry the overshoot in their values.)

## Adversarial critique (Ultracode workflow)

4 critics (craft-floor, a11y-motion, responsive-layout, copy-i18n) → per-finding
adversarial verify. 3 of 4 critics returned zero findings; copy-i18n surfaced 4
issues, of which **3 confirmed real** and fixed (see summary.md): the
reduced-motion gate, the nested `<a><button>` CTA, and the status-chip contrast.
The 4th (step chop 4.48:1) verified **not a defect** — `aria-hidden` decorative.

- The reduced-motion finding was verified by the critic both in the
  `motion-v@2.3.0` source (`resolveInitialValues` runs once at construction) and
  by a live Playwright `reducedMotion:'reduce'` reproduction showing the hero
  animating before the fix.

## Browser (live DOM + screenshots)

- **Nested-interactive fix**: `document.querySelectorAll('a button').length === 0`
  — every CTA is now a single `<a>` carrying the button classes (hero ×2, close
  ×1). Previously `<a href><button>`.
- **Status-chip fix**: computed styles confirm the design-system pair in both
  themes — light `bg-status-info-bg`/`text-status-info-fg`, dark
  `rgb(58,32,26)` bg / `rgb(232,144,126)` fg (~5.0:1 / ~6.2:1, AA-passing).
- **Renders**: hero title at `opacity 1`, all four step chops
  (落墨/成文/分枝/盖印) and the close CTA present. Hero screenshotted in **light
  and dark** — celadon ground / ink-night ground, Song-serif display, the
  Assembly diagram with the vermilion shipped seal, and the primary button
  correctly inverting ink→paper on dark.

## Notes / limitations

- The only console error is the known **BUG-011** locale SSR hydration mismatch
  (server `zh-CN` → client `en-US`); pre-existing and app-wide, recorded in
  `docs/issues/`, out of scope here.
- The reduced-motion fix is verified by construction (synchronous client init)
  and typecheck; the AssemblyDiagram's separate CSS keyframes were already gated
  by a `@media (prefers-reduced-motion)` query.
- Headless screenshots below the fold desync from scroll position in this
  environment, so below-fold steps were verified via the live DOM/a11y tree
  rather than pixels.
