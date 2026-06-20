# Design System Changelog

> Tracks the **design system** — token foundation, primitives, icon/identity maps, and these docs — not app features. Source plan: [`docs/plans/2026-06-19-frontend-design-system-deliverable.md`](../plans/2026-06-19-frontend-design-system-deliverable.md). For _why_ a value is what it is, cite the relevant doc ([`README.md`](./README.md)); this file records _what changed and when_.

Format follows [Keep a Changelog](https://keepachangelog.com/). Entries are grouped under a version (`## [x.y.z] — YYYY-MM-DD`). Until the token layer ships in code, the system is **pre-`0.1.0`** — the spec is authored, the code substrate is not.

**When to bump (the token contract is the version):**

- **Patch** (`0.0.x`) — doc-only edits, clarifications, new design-system docs; no code/token change.
- **Minor** (`0.x.0`) — add a semantic token, primitive, icon-map entry, or new themeable surface; backward-compatible.
- **Major** (`x.0.0`) — rename/remove a semantic token key or break the `:root`/`.dark` triplet contract (forces a find-replace across consumers).

Every token/primitive entry **must** name the file it lands in (e.g. `app/src/assets/tokens.css`, `app/tailwind.config.js`, `app/src/lib/icons.ts`) so the diff is traceable.

---

## [0.2.0] — 2026-06-20

Re-themed the semantic token palette to **国风留白 / Ink & Seal** — a soft celadon (青) ground with sumi-ink text and a single vermilion 印泥 (cinnabar-seal) accent. Source of truth: the Figma matrix page **`Ink&Seal · 全部`** (iPhone / iPad / Web × light / dark, all 10 screens + the presets sheet) in file `lACR32RXL0D1xrh6p48FcC`.

### Changed — palette re-theme (token _keys_ unchanged; only values + a few additions)

- [`app/src/assets/tokens.css`](../../app/src/assets/tokens.css) — both `:root` and `.dark` re-valued: `brand` indigo→vermilion; `surface-*` gray→celadon / ink-night; `stroke-*`→ink hairlines; `fg-*`→ink / body / meta on light, rice-paper on dark; the five `status-*` `-bg`/`-fg` ramps recolored to seal/grey/amber/sage washes (dark `-bg` pre-composited solid). `--radius-control` 0.5rem→0.625rem.

### Added — Ink & Seal signature tokens

- `app/src/assets/tokens.css` — `--color-seal` / `-strong` / `-locked` (the 印泥 chop + its hover + the grey locked chop), `--color-carve` (paper color drawn on a seal/ink fill), and `--color-btn` / `--color-btn-fg` (the **ink** primary-action surface that inverts to **paper** on dark).
- [`app/tailwind.config.js`](../../app/tailwind.config.js) — `colors.seal.{DEFAULT,strong,locked}`, `colors.carve`, `colors.btn.{DEFAULT,fg}`; `fontFamily.serif` (Noto Serif SC display). Ma Shan Zheng brush face is still an open self-hosting item (§11).

### Adoption note (not yet codemodded)

- The design keeps the **primary action ink, not vermilion**. To match it, switch `BaseButton` primary from `bg-brand` to `bg-btn text-btn-fg`; `seal`/`brand` stays the accent (chips, status chops, links, focus ring). Existing `bg-brand` consumers now render vermilion automatically — correct for accents, intentionally not for the primary button.

## [0.1.0] — 2026-06-19

The token substrate, the full primitive set, the a11y floor, and the brand assets **landed in code**. Verified: `pnpm test` **592 green**, `eslint .` clean, dark var-flip + modal focus-trap + the cover-contrast invariant confirmed live via gstack `/browse`.

### Added — token foundation (Phase 1)

- `app/src/assets/tokens.css` — `:root` + `.dark` semantic tokens as **channel triplets** (surface/stroke/fg/brand/status `-bg`/`-fg`, radius). Dark status `-bg` values are pre-composited so `bg-status-*-bg` flips cleanly with no alpha.
- [`app/tailwind.config.js`](../../app/tailwind.config.js) — `theme.extend` maps the semantic keys via `rgb(var(--x) / <alpha-value>)` (colors `brand/surface/stroke/fg/status`, `borderRadius` control/card/pill/hero, `boxShadow` card/popover/modal, `zIndex` ladder, `fontFamily.sans`). Stale `./src/views/**` content glob removed.
- [`app/src/style.css`](../../app/src/style.css) — `font-family` → Inter + Noto Sans SC + system-ui fallback (self-hosting via `@nuxt/fonts` is the open delivery decision §11).
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
