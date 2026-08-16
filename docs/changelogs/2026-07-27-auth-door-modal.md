# 2026-07-27 — Auth pages become one shared prompted floating modal

`/login` and `/register` are rebuilt from page-level cards into thin pages
that render a single shared shell — `components/auth/AuthDoor.vue` — a paper
panel floating over a dimmed scrim with real dialog semantics and a
reduced-motion-aware entrance. The dialog behavior is extracted from the
existing `BaseModal` into a shared composable and reused by `BaseModal`, so
the app now has one dialog implementation. Frontend-only; the auth flow, copy,
i18n keys, and routes are unchanged. The driving feedback is recorded in
[docs/design-qa/2026-07-27-auth-door.md](../design-qa/2026-07-27-auth-door.md).

## Why

Two design-QA comments from the user: *"the login page is ugly, is it possible
for you to refine it? for example, a motion to prompt a modal, and the seal
logo is not applied"* and *"can you make it a prompted floating modal with
motion?"*. The earlier same-day
[login refinement](2026-07-27-login-page-refinement.md) added the real seal and
a card that *surfaces like* a modal; this change makes it a *real* modal, and
brings `register` to parity (it still carried the fake bordered-glyph seal
placeholder).

## What changed

- **`components/auth/AuthDoor.vue` (new).** Teleported floating shell that
  paints its **own** world-ground (`bg-surface-base` on the fixed container, so
  the door always floats over celadon / ink-night — never a void), an **ambient
  ink vignette** (`radial-gradient`, transparent over the seal+panel, dimming
  only the periphery — *not* a flat overlay, which would black out the Ink &
  Seal ground in light mode), the faint token-driven vermilion wash
  (`rgb(var(--color-seal)/0.06)`), the real `/seals/yin.webp` seal stamping in,
  the serif wordmark as the `aria-labelledby` target, and a raised paper panel
  carrying the slotted form. The `role="dialog"` + `aria-modal="true"` landmark
  wraps the header **and** the panel (see the corrections below for why). The
  container's `@click.self` close works because both backdrop layers are
  `pointer-events-none`. Motion (seal stamp + panel surfacing, staggered) is
  gated by the landing's shared `useReducedMotion()`. No leave animation —
  closing navigates away, so exit motion would be dead code.
- **`composables/useDialogA11y.ts` (new).** The modal accessibility behavior
  extracted verbatim from `BaseModal`: focusable scan, Escape → close, Tab
  wrap, open-watch that saves/restores focus, locks body scroll, focuses the
  first focusable (panel fallback), and cleans up on unmount. `ponytail:` a
  `ref` on a `motion.div` yields motion-v's component proxy (no
  `querySelectorAll`), which crashed the trap on entry-page hydration; the fix
  keeps the composable byte-identical and puts `panelRef` on a plain inner
  `<div tabindex="-1">` instead.
- **`components/common/BaseModal.vue`.** Now delegates to `useDialogA11y`;
  its local duplicates are deleted. One latent visual bug fixed in passing: its
  scrim was `pointer-events-none`-less, so the opaque overlay child swallowed
  the container's `@click.self` (backdrop-click close was dead; only Escape and
  the X worked). The scrim is now `pointer-events-none`, matching `AuthDoor`.
  Everything else (markup, props, emits, transitions, aria) is byte-for-byte
  identical — regression-guarded by the suite below.
- **`pages/login.vue` / `pages/register.vue`.** Reduced to thin pages that
  slot their form into `AuthDoor` and close via `navigateTo('/')`. Register
  loses the fake bordered-glyph seal; both now share the real 印 seal.

No i18n keys were added, removed, or renamed. No new dependencies. No backend
change.

## Corrections after adversarial + visual verification

An adversarial pass over the first cut surfaced three issues, all fixed at the
root before sign-off:

1. **`aria-modal` was on the wrong element (accessibility blocker).** The
   `role="dialog"` + `aria-modal="true"` pair first landed on the decorative
   header `<div>` only, leaving the form panel and the close button as
   *siblings outside* the landmark. Because `aria-modal="true"` tells the UA to
   hide everything outside the dialog from the accessibility tree, every form
   control and the focus-trap root were being pruned for screen-reader users.
   Fixed by moving the landmark onto the shared wrapper that contains the
   header, the panel, and the `panelRef` focus root.
2. **Dead backdrop click (major).** The opaque scrim child intercepted pointer
   events, so the container's `@click.self` never matched — backdrop-click
   close was dead on both `AuthDoor` and (latently) `BaseModal`. Fixed with
   `pointer-events-none` on the backdrop layers.
3. **Black-void backdrop in light mode (visual).** The first cut used the
   generic `bg-surface-overlay` scrim, whose token is `0 0 0` with no alpha — a
   fully opaque black wall that erased the celadon ground entirely in light
   mode (the exact "near-black background, single accent" anti-pattern). Fixed
   by giving the modal its own `bg-surface-base` ground and replacing the flat
   veil with the ambient vignette described above, so the door floats within
   the Ink & Seal world in both themes.

A side benefit of the `open = ref(false)` + `onMounted(() => open = true)`
pattern (SSR and first client paint both render nothing; the entrance plays
post-mount): the auth surfaces no longer emit the product-wide
`Hydration completed but contains mismatches` warning that the motion elements
produced elsewhere — confirmed by a clean console in a fresh browser below.

## Verification

- `cd app && npx nuxi typecheck` — exit 0.
- `cd app && pnpm build` — green (the arbitrary `radial-gradient` vignette
  class compiles).
- `cd app && pnpm test` — 16 files / 60 tests passed (regression guard for the
  `BaseModal` refactor).
- Live, fresh browser (real backend on `:54363`, `/login` + landing loads),
  light **and** dark: `role="dialog"` wraps the close button + form + wordmark,
  `aria-modal="true"`, `aria-labelledby` → wordmark, the scrim/vignette are
  `pointer-events: none`, the real 印 seal renders, and the container paints
  the world ground (`rgb(228,234,228)` light / `rgb(22,26,23)` dark). Close
  paths all navigate to `/`: Escape, backdrop click (`elementFromPoint` in the
  corner hits the container → `@click.self`), and the X button. `/register`
  renders the same shell with the real seal (placeholder gone) and
  `aria-modal="true"`. With `prefers-reduced-motion: reduce`, the seal's first
  paint is `opacity:1 / transform:none` (vs. mid-stamp `opacity≈0.77 /
  matrix(0.88…)` when motion is on) — entrances fully suppressed. **Console:
  zero errors/warnings across every load** (no hydration mismatch on these
  surfaces). Screenshots confirmed the ambient backdrop in both themes.

## Operator notes

- Frontend-only: no migrations, no env changes, no re-clone.
- The `/login` and `/register` routes still exist; the auth middleware's
  redirect target is unchanged.
- Closing the modal (Escape or backdrop) navigates to `/`.
