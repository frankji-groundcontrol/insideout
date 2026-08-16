# 2026-07-27 — Login page refined into the Ink & Seal door

The `/login` page is rebuilt from a generic bordered-letter placeholder into a
proper Ink & Seal "door": the real baiwen seal impression, the serif wordmark,
and a paper card that surfaces like a modal — all animated with the landing's
existing reduced-motion-aware motion language. Frontend-only; the public auth
flow, copy, and i18n keys are unchanged.

## Why

The page was inconsistent with the committed visual world in two ways a user
flagged as "ugly": (1) it faked the seal — a bordered box holding the first
letter of the brand name instead of the generated 印 baiwen seal that the
landing and navbar already ship — and (2) it had no motion at all, so the card
just appeared flat on the celadon field. Both are fixed by reusing assets and
patterns that already exist in the codebase rather than inventing new ones.

## What changed

`app/src/pages/login.vue`:

- **Real seal logo.** The bordered `border-2 border-seal` box + single serif
  glyph is replaced by the real `/seals/yin.webp` baiwen seal (印 — "seal", the
  correct mark for the door; the navbar uses the 内外 brand seal). It carries
  `alt=""` / `aria-hidden="true"` like the navbar, because the adjacent `<h1>`
  wordmark already names the product — the seal is decorative, not a second
  announcement of the name.
- **One authored motion moment.** The seal *stamps in* with the same scale
  keyframe curve the landing's `AssemblyDiagram` uses for its `seal-stamp`
  (`scale: [1.6, 0.88, 1.07, 1]`, 0.65 s) — so the entrance reads as a seal
  being pressed onto the page, consistent with the rest of the product. The
  wordmark and greeting rise in behind it, then the **card surfaces like a
  modal** (`opacity 0→1`, `y 24→0`, `scale 0.98→1`) — that surfacing is the
  requested "motion that prompts a modal." All entrances use `motion-v` with
  the landing's ease `[0.16, 1, 0.3, 1]` and the shared `useReducedMotion()`
  gate (`:initial="reduce ? false : {…}"`), the identical SSR-safe-by-pattern
  idiom as `AssemblyHero`.
- **Composition.** Removed the redundant in-card serif title (the wordmark +
  "Welcome back" greeting already headline the page); the card now holds only
  the form, named for assistive tech via `:aria-label="t('login.title')"`. A
  single faint vermilion radial wash sits behind the seal
  (`rgb(var(--color-seal)/0.06)`), token-driven so it flips correctly in dark;
  it is the seal's own presence bleeding into the paper, not a second solid
  vermilion element (the One Seal Rule holds — the only solid vermilion is the
  seal plus the existing seal-colored register link).
- The ink-paper primary button (Ink-Not-Vermilion rule), the sunken input
  wells, and the soft offset+blur `shadow-modal` were already token-correct, so
  they carry over unchanged.

No i18n keys were added or removed (`login.title` is repurposed as the form's
accessible name). No backend change.

## Verification

- `cd app && npx nuxi typecheck` — exit 0 (the `motion-v` keyframe-array
  `:animate` and the `as const` ease tuple type-check, matching the landing).
- `cd app && pnpm build` — green.
- Live dev-server (preview pane): `/login` confirmed in **light** (celadon
  ground, sumi-ink wordmark, rice-paper card) and **dark** (ink-night ground,
  seal lifted) — the real 印 seal renders, the single vermilion wash sits
  behind it, and the ink primary button + seal-colored link are correct in
  both themes. No console errors beyond the known hydration note below.

## Known trade-off (not introduced here)

The dev console logs `Hydration completed but contains mismatches` for the
motion elements. This is **pre-existing and product-wide**, not a login
regression: the shared `useReducedMotion()` composable reads
`prefers-reduced-motion` synchronously on the client (so a motion-sensitive
visitor is never flashed the animated entrance before the gate flips), which
makes `motion-v`'s `initial` differ between the SSR render (`false`) and the
client's first render (the real query). The landing's `AssemblyHero` /
`AssemblyStep` use the exact same `reduce ? false : {…}` gate and emit the same
warning. It is dev-only and self-correcting on the client; the production build
renders correctly. A clean root-cause fix (server-side motion preference, or a
post-hydration mount gate) is a cross-surface change to the shared composable
and the landing — recorded as a follow-up, not folded into this page-level
refinement.

## Operator notes

None — purely a frontend visual change; no migrations, no env changes, no
re-clone. `register.vue` deliberately keeps the old placeholder pattern for
now; giving it the same door is the obvious one-line follow-up if desired.
