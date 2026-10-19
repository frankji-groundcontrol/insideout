# Design System Changelog

> Tracks the **design system** — token foundation, primitives, icon/identity maps, and these docs — not app features. (The original source plan and companion README predate this repo's current docs layout and were never carried over — see git history if needed.) The live token wiring is described in [`docs/architecture/frontend.md`](../architecture/frontend.md); this file records _what changed and when_.

Format follows [Keep a Changelog](https://keepachangelog.com/). Entries are grouped under a version (`## [x.y.z] — YYYY-MM-DD`). Live Flutter tokens are **`1.1.0` Cobalt-Neo** — the `1.0.0` Cobalt keys (renamed from Ink & Seal), all kept, with squared geometry and a block component layer added.

**When to bump (the token contract is the version):**

- **Patch** (`0.0.x`) — doc-only edits, clarifications, new design-system docs; no code/token change.
- **Minor** (`0.x.0`) — add a semantic token, primitive, icon-map entry, or new themeable surface; backward-compatible.
- **Major** (`x.0.0`) — rename/remove a semantic token key or break the `:root`/`.dark` triplet contract (forces a find-replace across consumers).

Every token/primitive entry **must** name the file it lands in (e.g. `client/lib/theme/cobalt.dart`) so the diff is traceable.

---

## [1.1.0] — 2026-10-08

Minor: **Cobalt-Neo.** The cobalt hue family and the screen layouts stay; the
geometry is squared (0px corners, 2px borders, hard zero-blur cast shadows, a
100ms press-the-block motion) and a block component layer is built on the
tokens. Every `1.0.0` semantic key survives (the radius names are kept, pinned
to `0`), so this is not a major bump. It landed in three dated passes: tokens
([2026-10-06](../changelogs/2026-10-06-frontend-cobalt-neo.md)), components
([2026-10-07](../changelogs/2026-10-07-frontend-block-craft.md)), and keyboard
focus ([2026-10-08](../changelogs/2026-10-08-block-focus-ring.md)). The law is
[`DESIGN.md`](../../DESIGN.md); the [lock screens](cobalt/README.md) predate
this version. Token values stay pinned by
[`client/test/cobalt_test.dart`](../../client/test/cobalt_test.dart) (frozen
assertions plus the WCAG contrast script).

### Changed — token values (keys unchanged)

- [`client/lib/theme/cobalt.dart`](../../client/lib/theme/cobalt.dart) —
  `radiusControl` / `radiusCard` / `radiusHero` / `radiusPill` are `0` (were
  12 / 20 / 28 / 999). Strokes retuned to clear 3:1 non-text contrast: light
  `strokeSubtle` `#7B8CE0` and `strokeStrong` `#5A6FE0`, dark `#646A78` and
  `#788098` (ratios in DESIGN.md). Dark surfaces are a neutral ramp
  (`surfaceRaised` `#14171D`, `surfaceSunken` `#16181E`); in the dark table
  `hue950` fills `brandSubtle` and the status-chip backgrounds (the coach
  column also uses raw `hue950` as its light-theme slab). Light
  `statusNeutralFg` is `hue700`.
- [`client/lib/session/appearance.dart`](../../client/lib/session/appearance.dart) —
  dark is the default theme; a saved explicit choice still wins.

### Added — tokens

- `client/lib/theme/cobalt.dart` — `borderContainer` 2 and `borderDivider` 1;
  the press motion `pressDuration` 100ms, `hoverOffset` (2,2) and
  `activeOffset` (4,4); the 4px grid `space1`–`space10`; breakpoints
  `bpNarrow` / `bpWide` / `bpDesk` 600 / 900 / 1200; `contentMaxWidth` 1120;
  `controlHeight` 44 and `controlHeightDense` 40; `warnAmber` `#FFD166`, the
  warning channel only.
- [`client/lib/theme/cobalt_theme.dart`](../../client/lib/theme/cobalt_theme.dart) —
  `cast` (offset 4,4) and `castSm` (2,2), zero blur and zero spread, coloured
  by `castShadowColor` (dark `#05070A`, light black at 15%); the shared
  `eyebrow(context)`. `ThemeData` now squares every shape, draws containers
  with 2px borders, and sets elevation to zero.

### Added — primitives

- [`client/lib/widgets/pressable_block.dart`](../../client/lib/widgets/pressable_block.dart) —
  `PressableBlock`: rest on `cast`, hover to `castSm` shifted (2,2), press
  flush and shifted (4,4), over `pressDuration`.
- [`client/lib/theme/cobalt_chip.dart`](../../client/lib/theme/cobalt_chip.dart)
  and [`field_card.dart`](../../client/lib/theme/field_card.dart) — chips and
  cards take the 2px container border; the status mark (`ShadeMark`) is a
  square block, not a dot; `FieldCard` rests on `cast` and presses through
  `PressableBlock`.
- `client/lib/theme/block_*.dart` — the component layer: `block_button.dart`
  (`BlockButton`, `BlockChoice`), `block_field.dart` (`BlockField`,
  `BlockTextarea`, `BlockSelect`, on `block_field_frame.dart`),
  `block_modal.dart` (`showBlockModal`, `showBlockConfirm`),
  `block_chrome.dart` (`BlockBanner`, `BlockBadge`, `BlockTag`,
  `BlockIconWell`, `BlockLoader`, `BlockTone`), `block_icon_button.dart`
  (`BlockIconButton`), `block_state.dart` (`BlockEmpty`, `BlockLoading`,
  `BlockError`), `block_form.dart` (`showBlockForm`), and `block_focus.dart`
  (`BlockFocus`, the 2px cobalt focus ring with Tab and Enter/Space).

## [1.0.0] — 2026-09-18

Major: semantic type renamed **InkSeal → Cobalt**. Seal / celadon / vermilion
keys are removed, not aliased. Primary action is cobalt-500. Record:
[`docs/changelogs/2026-09-18-cobalt-flutter-restyle.md`](../changelogs/2026-09-18-cobalt-flutter-restyle.md).

### Changed — token type and values

- [`client/lib/theme/cobalt.dart`](../../client/lib/theme/cobalt.dart) —
  light + dark palettes from the accepted lock (cobalt-50/200/500/700/950,
  `#F7F8FA` / `#0B0D12` neutrals).
- [`client/lib/theme/cobalt_theme.dart`](../../client/lib/theme/cobalt_theme.dart) —
  ThemeData: cobalt primary, body 17.5, titles 26+, display 48–58, meta 12,
  Archivo + Noto Sans SC; coach serif is Source Serif 4 + Noto Serif SC.
- [`client/lib/theme/cobalt_chip.dart`](../../client/lib/theme/cobalt_chip.dart) —
  shade pills replace chops.
- [`DESIGN.md`](../../DESIGN.md) rewritten to the Cobalt card lock.

### Removed

- `client/lib/theme/ink_seal.dart`, `ink_seal_theme.dart`, `seal_chip.dart`
  and live uses of `assets/seals/*.webp`.

## [0.5.0] — 2026-08-19

Token substrate moved onto the Flutter client after the Nuxt `app/` tree
was deleted. Semantic **keys and values** are unchanged from `0.4.0` /
`0.2.0` (celadon, sumi ink, vermilion seal, paper carve). The consumer is
now Dart `ThemeData` rather than CSS custom properties. Record:
[`docs/changelogs/2026-08-19-restore-ink-seal.md`](../changelogs/2026-08-19-restore-ink-seal.md),
plan [`docs/plans/2026-08-19-restore-ink-seal.md`](../plans/2026-08-19-restore-ink-seal.md).

### Changed — token consumer (keys/values unchanged)

- `client/lib/theme/ink_seal.dart` —
  light + dark palettes matching last live `app/src/assets/tokens.css`
  (`f897fb4`).
- `client/lib/theme/ink_seal_theme.dart` —
  ThemeData: celadon scaffold, ink primary button, vermilion focus/accent,
  Song display / Gothic body, control/card/hero radii.
- [`client/assets/seals/yin.webp`](../../client/assets/seals/yin.webp) —
  recovered 印 seal from `app/public/seals/yin.webp`.

The Nuxt `tokens.css` / `tailwind.config.js` paths in older entries are
historical; they no longer exist on disk.

## [0.4.0] — 2026-07-23

Reverted the `0.3.0` Prisma cinematic detour back to the committed **国风留白 / Ink & Seal** world, and rebuilt the public landing on that foundation. Token _keys_ are unchanged (the `0.2.0` values are restored), so this is a backward-compatible minor bump; the new landing is a new themeable surface. Record: [`docs/changelogs/2026-07-23-ink-seal-landing/`](../changelogs/2026-07-23-ink-seal-landing/index.md).

### Changed — palette re-theme, reverted (token _keys_ unchanged; only values)

- `app/src/assets/tokens.css` — restored the HEAD **Ink & Seal** values in both `:root` and `.dark`: celadon/ink ground, vermilion `seal`, paper `carve`, the ink `btn`/`btn-fg` that inverts to paper on dark, and the five `status-*` ramps. The `0.3.0` pure-black + cream re-value is discarded; dark is no longer the forced default.
- `app/nuxt.config.ts` — the FOUC default reverted from forced-dark to `prefers-color-scheme`; added the Noto Serif SC display link.

### Changed — typography back to Ink & Seal

- `app/tailwind.config.js` — `fontFamily.serif` leads with **Noto Serif SC** (the Song/Mincho display face); `fontFamily.sans` stays on self-hosted **Alibaba PuHuiTi**. The `0.3.0` PuHuiTi-for-everything + Impeccable-display mapping is removed.
- `app/src/style.css` — kept the PuHuiTi `@font-face`; dropped the Prisma `.noise-overlay`/`.bg-noise` utilities.

### Added — landing surface (「The Assembly」)

- `app/src/pages/index.vue` + `app/src/components/landing/{AssemblyHero,AssemblyStep,AssemblyDiagram,StepPeek}.vue` — the public landing rebuilt as an ink build-instruction narrative (idea spark → PRD → branched roadmap → shipped seal), fully on semantic tokens in light + dark.
- `app/src/components/common/BaseButton.vue` — new `to` prop renders a single styled `NuxtLink` (so CTA links stay one interactive element, never `<a><button>`).

> **License note retired:** the Impeccable display font (personal-use) is no longer referenced after this revert; the `0.3.0` license caveat applies only to that version's usage.

## [0.3.0] — 2026-07-22

Re-themed the semantic token palette from **国风留白 / Ink & Seal** to **Prisma cinematic** — a pure-black ground with a warm cream ink (`#E1E0CC` / `#DEDBC8`) and the cream itself as the accent (the vermilion 印泥 accent is retired). Reference: the Prisma studio landing page built earlier the same day. Light mode becomes the **warm-paper inversion** (cream ground, ink text). **Dark is now the default theme.**

### Changed — palette re-theme (token _keys_ unchanged; only values)

- `app/src/assets/tokens.css` — both `:root` and `.dark` re-valued: `surface-*` celadon/ink-night→pure black + `#101010`/`#1c1c1c` (dark) and warm cream paper (light); `fg-*`→warm cream ink on dark, ink on paper light; `seal`/`brand` vermilion→cream (dark) / ink (light); `carve`→black on cream fills; `btn`/`btn-fg`→cream button (dark) / ink button (light); the five `status-*` ramps re-harmonized to the warm palette.

### Added — Prisma typography + texture + motion

- `app/src/style.css` — `@font-face` for **Alibaba PuHuiTi** (self-hosted from `app/public/fonts/`, weights 400/500/700/800) — the universal font covering Latin + full CJK, so no external font CDN and no CJK fallback gap. Also `@font-face` for **Impeccable** (Latin-only monoline handwritten script) as the elegant display/accent face. Also `.noise-overlay` + `.bg-noise` SVG fractal-noise (feTurbulence) texture utilities.
- `app/tailwind.config.js` — `fontFamily.sans` and `fontFamily.serif` both → **Alibaba PuHuiTi** (display headings render PuHuiTi heavy weights; no separate serif face); `fontFamily.display` → **Impeccable** (Latin accent only — no Han glyphs, so never on CJK text).
- `app/nuxt.config.ts` — all external font links removed (fully self-hosted); the FOUC script's default theme is now **dark**.
- `app/src/components/layout/NavBar.vue` — brand wordmark uses `font-display` (Impeccable script).
- `motion-v` dependency — cinematic pull-up / fade / scale reveals (landing page so far).
- `app/src/pages/index.vue` — landing hero + journey pillars rebuilt with motion-v reveals and the noise texture.

> **License note:** Impeccable is a **personal-use / demo** font (author: Paily
> Studio). The full/commercial license must be purchased before InsideOut is
> used commercially — see `app/public/fonts/Impeccable.ttf`'s source readme.

## [0.2.0] — 2026-06-20

Re-themed the semantic token palette to **国风留白 / Ink & Seal** — a soft celadon (青) ground with sumi-ink text and a single vermilion 印泥 (cinnabar-seal) accent. Source of truth: the Figma matrix page **`Ink&Seal · 全部`** (iPhone / iPad / Web × light / dark, all 10 screens + the presets sheet) in file `lACR32RXL0D1xrh6p48FcC`.

### Changed — palette re-theme (token _keys_ unchanged; only values + a few additions)

- `app/src/assets/tokens.css` — both `:root` and `.dark` re-valued: `brand` indigo→vermilion; `surface-*` gray→celadon / ink-night; `stroke-*`→ink hairlines; `fg-*`→ink / body / meta on light, rice-paper on dark; the five `status-*` `-bg`/`-fg` ramps recolored to seal/grey/amber/sage washes (dark `-bg` pre-composited solid). `--radius-control` 0.5rem→0.625rem.

### Added — Ink & Seal signature tokens

- `app/src/assets/tokens.css` — `--color-seal` / `-strong` / `-locked` (the 印泥 chop + its hover + the grey locked chop), `--color-carve` (paper color drawn on a seal/ink fill), and `--color-btn` / `--color-btn-fg` (the **ink** primary-action surface that inverts to **paper** on dark).
- `app/tailwind.config.js` — `colors.seal.{DEFAULT,strong,locked}`, `colors.carve`, `colors.btn.{DEFAULT,fg}`; `fontFamily.serif` (Noto Serif SC display). Ma Shan Zheng brush face is still an open self-hosting item (§11).

### Adoption note (not yet codemodded)

- The design keeps the **primary action ink, not vermilion**. To match it, switch `BaseButton` primary from `bg-brand` to `bg-btn text-btn-fg`; `seal`/`brand` stays the accent (chips, status chops, links, focus ring). Existing `bg-brand` consumers now render vermilion automatically — correct for accents, intentionally not for the primary button.

## [0.1.0] — 2026-06-19

The token substrate, the full primitive set, the a11y floor, and the brand assets **landed in code**. Verified: `pnpm test` **592 green**, `eslint .` clean, dark var-flip + modal focus-trap + the cover-contrast invariant confirmed live via gstack `/browse`.

### Added — token foundation (Phase 1)

- `app/src/assets/tokens.css` — `:root` + `.dark` semantic tokens as **channel triplets** (surface/stroke/fg/brand/status `-bg`/`-fg`, radius). Dark status `-bg` values are pre-composited so `bg-status-*-bg` flips cleanly with no alpha.
- `app/tailwind.config.js` — `theme.extend` maps the semantic keys via `rgb(var(--x) / <alpha-value>)` (colors `brand/surface/stroke/fg/status`, `borderRadius` control/card/pill/hero, `boxShadow` card/popover/modal, `zIndex` ladder, `fontFamily.sans`). Stale `./src/views/**` content glob removed.
- `app/src/style.css` — `font-family` → Inter + Noto Sans SC + system-ui fallback (self-hosting via `@nuxt/fonts` is the open delivery decision §11).
- Cleanups: deleted the dead `HelloWorld.vue`; `BaseModal` magic `z-50` → `z-modal`; `<html lang>` follows locale (`app/src/app.vue` reactive `useHead`); document title template `%s · 卷了么`.
- Guard test: i18n key parity (`app/src/i18n/__tests__/localeParity.test.ts`); eslint config now ignores build output (`.nuxt`/`.output`) + Nuxt single-word filenames + `^_` arg convention.

### Added — a11y + exemplars (Phase 2)

- **Cover-contrast invariant** — `app/src/lib/a11y.ts` (WCAG luminance/contrast/composite) + `workshopCover.contrast.test.ts`: the white monogram chip was failing 4.5:1 (worst 3.35:1); the chip is now a **dark frosted glass** (`bg-black/15`) → ≥4.5:1 across the band. Figma cover updated to match (design-first).
- `BaseBadge.vue` (tone-based, status tokens) — `PrdStatusBadge` + the `WorkshopCard` status pill route through it. `BrandSpinner.vue` consolidates the panel spinners.
- `BaseButton`/`BaseInput`/`BaseModal` migrated onto tokens + `focus:` → `focus-visible:`; `BaseInput` error is now `aria-describedby`-associated (+ `aria-required`/`aria-invalid`); `BaseModal` gains focus-trap, return-focus, `aria-labelledby`; toggles bumped to ≥44px with focus-visible rings.

### Added — primitives + identity + assets (Phase 3, dynamic workflow)

- `app/src/lib/icons.ts` — the semantic icon map (`statusIcon`/`actionIcon`/`navIcon`, AI = `SparklesIcon`).
- `app/src/lib/identity.ts` + `GenerativeIdentity.vue` — the generative-identity primitive promoted out of workshop-only scope (card/avatar/tile shapes).
- `BaseCard`, `BaseEmptyState`, `BaseSkeleton`, `BaseToast` + `useToast`, `BaseTooltip`, `BaseAvatar`, `BaseIconButton`, `BaseTextarea`, `BaseSelect` — all TDD, token-based.
- Brand assets: `app/public/brand/{logo.svg, favicon.svg, apple-touch-icon.png, og.png}`; favicon + og/twitter meta wired; `vite.svg` removed.

### Added — adoption (Phase 4)

- `NavBar` avatar → `BaseAvatar` (generative identity); dashboard joined-empty → `BaseEmptyState`.

### Tail (incremental, explicitly allowed by the plan §10/§11)

- **Per-page token codemod** of the remaining raw `indigo-*`/`gray-*` utilities (working surfaces: roadmap/editor/PRD harness, profile, manage, export) — non-breaking, do screen-by-screen Figma-first.
- `BaseButton` `secondary`/`outline` variants still on dark:-paired utilities (primary/danger migrated).
- Repo-wide `prettier --write` of ~109 pre-existing legacy files (new files are clean).
- Component-level `components/*.md` specs + Figma Code Connect for the new primitives.

## [Unreleased]

_Nothing yet._
