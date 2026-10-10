---
name: InsideOut
description: Cobalt-Neo block lock — track the work, refine the ideas, ship better PRDs.
colors:
  cobalt-50: "#EEF2FF"
  cobalt-200: "#C7D2FE"
  cobalt-500: "#3B5BFF"
  cobalt-700: "#1D4ED8"
  cobalt-950: "#1E2A5A"
  light-ground: "#F7F8FA"
  light-ink: "#0B0D12"
  light-stroke-subtle: "#7B8CE0"
  light-stroke-strong: "#5A6FE0"
  dark-ground: "#0B0D12"
  dark-type: "#F7F8FA"
  dark-raised: "#14171D"
  dark-sunken: "#16181E"
  dark-stroke-subtle: "#646A78"
  dark-stroke-strong: "#788098"
  cast-dark: "#05070A"
  warn-amber: "#FFD166"
typography:
  display:
    fontFamily: '"Archivo", "Noto Sans SC", system-ui, sans-serif'
    fontWeight: 700
    lineHeight: 1.0
  body:
    fontFamily: '"Archivo", "Noto Sans SC", system-ui, sans-serif'
    fontWeight: 400
    lineHeight: 1.45
  coach:
    fontFamily: '"Source Serif 4", "Noto Serif SC", Georgia, serif'
    fontWeight: 600
    lineHeight: 1.35
rounded:
  control: "0"
  card: "0"
  hero: "0"
  pill: "0"
borders:
  container: "2px solid"
  divider: "1px solid"
shadows:
  cast: "4px 4px 0 0"
  castSm: "2px 2px 0 0"
motion:
  pressDuration: "100ms"
  hoverOffset: "2px 2px"
  activeOffset: "4px 4px"
spacing:
  unit: "4px"
  scale: "4 / 8 / 12 / 16 / 24 / 32 / 40"
layout:
  contentMaxWidth: "1120"
  breakpoints: "600 / 900 / 1200"
components:
  heights:
    buttonSm: "32"
    buttonMd: "40"
    buttonLg: "52"
    field: "44"
    fieldDense: "40"
    modalMaxWidth: "512"
  button-primary:
    backgroundColor: "{colors.cobalt-500}"
    textColor: "#FFFFFF"
    rounded: "{rounded.control}"
    border: "{borders.container}"
    shadow: "{shadows.cast}"
  chip-shade:
    backgroundColor: "{colors.cobalt-50}"
    textColor: "{colors.cobalt-700}"
    rounded: "{rounded.control}"
    border: "{borders.container}"
  card:
    backgroundColor: "#FFFFFF"
    rounded: "{rounded.card}"
    border: "{borders.container}"
    shadow: "{shadows.cast}"
---

# Design System: InsideOut

## Overview

**Creative North Star: Cobalt-Neo — cobalt blocks on a field.**

One hue. Neutrals carry the page. Light and dark are required. The product
loop — Idea → version → human Commit → timed Roadmap → Git evidence — is
read as **squared blocks on a ground**: 0px corners, 2px solid borders,
hard 4px 4px zero-blur cast shadows, and a 100ms press-the-block motion.
Not a full-bleed slab, not rounded cards, not type-rows, not a left rail.

Lock screens live in
[`docs/design-system/cobalt/`](docs/design-system/cobalt/README.md).
The Flutter tokens are `Cobalt` in
[`client/lib/theme/cobalt.dart`](client/lib/theme/cobalt.dart); cast shadows
and theme wiring in
[`client/lib/theme/cobalt_theme.dart`](client/lib/theme/cobalt_theme.dart);
press motion in
[`client/lib/widgets/pressable_block.dart`](client/lib/widgets/pressable_block.dart).
The component layer — buttons, fields, the one modal chrome, chrome
pieces, and read states — lives in `client/lib/theme/block_*.dart`
(see [Block primitives](#block-primitives)).
The signed-off token table and its verification record live in
[`docs/plans/2026-10-06-frontend-cobalt-neo/2026-10-06-frontend-cobalt-neo.md`](docs/plans/2026-10-06-frontend-cobalt-neo/2026-10-06-frontend-cobalt-neo.md).

**Key Characteristics:**

- One accent: cobalt-500. Shades do hover, Now, focus, and chips.
- **Every corner is 0px.** The radius token names are kept (`radiusControl`,
  `radiusCard`, `radiusHero`, `radiusPill`) but pinned to 0 — the decree is
  semantic, not a per-call-site choice.
- **Borders come in two weights:** 2px `borderContainer` for blocks (cards,
  chips, modals, dock, inputs), 1px `borderDivider` for hairlines (table
  rows, separators, underlines).
- **Shadows are cast, not blurred:** `cast` = offset (4,4), blur 0, spread 0;
  `castSm` = offset (2,2). Dark theme: solid ink `#05070A`. Light theme:
  black at 15% (`#26000000`).
- **Motion is press-the-block, 100ms:** rest sits on `cast`; hover translates
  (2,2) and shrinks to `castSm`; active translates (4,4) with the shadow
  gone — the block presses into the page.
- **Controls have fixed measured heights, border included:** buttons
  32 / 40 / 52 (`sm` / `md` / `lg`), fields and selects 44, dense fields
  40, the shared modal chrome caps at 512 wide.
- **Spacing is the named 4px scale** (`space1`–`space10` = 4 / 8 / 12 / 16 /
  24 / 32 / 40); breakpoints are 600 / 900 / 1200 (`bpNarrow` / `bpWide` /
  `bpDesk`); signed-in read surfaces cap their content column at **1120**
  (`contentMaxWidth`) on wide viewports.
- Light ground `#F7F8FA` / ink `#0B0D12`. Dark ground `#0B0D12` / type `#F7F8FA`.
- Dark surfaces are a **neutral ramp** (raised `#14171D`, sunken `#16181E`);
  cobalt-950 is demoted to `brandSubtle` and dark status-chip backgrounds —
  never a page surface.
- Body **17.5**. Titles **26+**. Display **48–58**. Meta **12**.
- Ideas wall display `clamp(64px, 10vw, 320px)` on wide web.
- UI sans: Archivo (Latin) + Noto Sans SC (CJK). zh-CN is the default locale.
- Dark is the default theme; a saved explicit choice still wins.
- Coach-only serif: Source Serif 4 + Noto Serif SC.
- Primary action is **cobalt**, not ink.
- Status is a squared shade chip, not a chop; status dots and loaders are
  square blocks.

## Colors

| Token | Hex | Use |
| --- | --- | --- |
| cobalt-50 | `#EEF2FF` | Light wash / chip bg |
| cobalt-200 | `#C7D2FE` | Chip edge, poster edge, dark fgBrand |
| cobalt-500 | `#3B5BFF` | Primary action, focus, Now |
| cobalt-700 | `#1D4ED8` | Pressed / chip text (light) |
| cobalt-950 | `#1E2A5A` | `brandSubtle` (dark), dark status-chip bg only |

Strokes (verified against the WCAG script in `client/test/cobalt_test.dart`):

| Token | Theme | Hex | Measured contrast |
| --- | --- | --- | --- |
| strokeSubtle | light | `#7B8CE0` | 3.15:1 on white / lightGround |
| strokeStrong | light | `#5A6FE0` | 4.39:1 on white / lightGround |
| strokeSubtle | dark | `#646A78` | 3.27:1 on sunken, 3.58:1 on ground |
| strokeStrong | dark | `#788098` | 4.47:1 on sunken, 4.89:1 on ground |

Accent exceptions:

| Item | Hex | Note |
| --- | --- | --- |
| Warn channel | `#FFD166` | Amber is the warning channel only: the dock warn dot (square 8×8 block; measured 3.53:1 on the cobalt dock — ≥3:1 non-text pass; the plan's "≈7:1" estimate was wrong, value stands) and warning-tone block chrome (`BlockBanner` strips, badges) in dark theme. On light grounds the warning tone darkens to cobalt-700, because pure amber cannot clear 3:1 against a pale surface. |
| Cast shadow (dark) | `#05070A` | Solid ink |
| Cast shadow (light) | `#26000000` | Black 15% |

Cast shadows are decorative depth, not informational graphics: they sit at
~1.2:1 against their own ground by construction (any ≥3:1 "shadow" would be
a gray glow). The WCAG script asserts they are strictly darker than the
ground instead (WCAG 1.4.11 exemption).

No magenta, lime, orange, cream, or extra brand hues. Amber appears only as
the warn channel.

## Typography

**UI:** Archivo + Noto Sans SC. **Coach:** Source Serif 4 + Noto Serif SC.
Web loads faces from `client/web/index.html` and gates the first frame in
`client/lib/theme/web_fonts_web.dart`. Native keeps lazy Noto assets.

### Hierarchy

- **Display** (700, 48–58, Ideas wall up to 320): page names and the Home numeral.
- **Title** (700, 26+): card and section headings.
- **Body** (400, 17.5): reading text.
- **Meta** (600, 12): chips, eyebrows, dock labels.

## Layout

Squared blocks on a field. Home is an index block plus a giant numeral with
workspace blocks beside it. Ideas is a type wall and poster blocks. PRD +
Coach is a split desk. Commit is a centered fanned stack. Roadmap is Now /
Next / Later shade bands that still hold item blocks. Signed-in chrome is a
bottom dock. Locale and theme stay in the top-right cluster. Signed-in read
surfaces constrain their content column to 1120 (`contentMaxWidth`) on wide
viewports instead of spanning the full window.

## Components

### Block primitives

The component layer lives in `client/lib/theme/` and is the mandated
grammar for interactive surfaces — no stock Material dialogs, default-
chrome buttons, floating-label fields, or round spinners:

- [`block_button.dart`](client/lib/theme/block_button.dart) — `BlockButton`
  (fixed outer heights 32 / 40 / 52 for `sm` / `md` / `lg`; variants
  `primary`, `secondary`, `danger`, `ghost`) and `BlockChoice`, the
  pressable pick-one selection block (brand fill + ink edge selected,
  raised block unselected).
- [`block_field.dart`](client/lib/theme/block_field.dart) — `BlockField`,
  `BlockTextarea`, `BlockSelect`: label above (12, w600), raised
  fixed-height box (44; dense 40), 2px border `strokeSubtle` at rest →
  `brand` on focus → `fgDanger` on error, `helper` and error slots below.
- [`block_modal.dart`](client/lib/theme/block_modal.dart) — `showBlockModal`
  / `showBlockConfirm`: the one modal chrome (512 max width, header +
  close block, 2px divider, scroll body, sunken footer bar).
- [`block_chrome.dart`](client/lib/theme/block_chrome.dart) — `BlockBanner`
  (4px tone strip), `BlockIconButton`, `BlockBadge`, `BlockTag`,
  `BlockIconWell`, and the square `BlockLoader`.
- [`block_state.dart`](client/lib/theme/block_state.dart) — `BlockEmpty` /
  `BlockLoading` / `BlockError`; `BlockError` localizes through
  `api/errors_l10n.dart` and offers a retry — never a raw error string.
- [`block_form.dart`](client/lib/theme/block_form.dart) — `showBlockForm`
  builds a labeled form dialog on the shared chrome.

### Buttons

Primary is a cobalt block: 0 radius, 2px border, `cast` shadow, 100ms
press-the-block. `BlockButton` implements the choreography directly — rest
`cast` → hover `castSm` + translate (2,2) → active flush + translate (4,4)
— because Material's `ButtonStyle` cannot translate its shell;
`PressableBlock` supplies the same motion for custom interactive blocks
like the dock, and the theme button styles keep the squared geometry,
duration, and zero elevation for any Material button that survives.
Primary and danger are bordered in ink (`lightInk`) in **both** themes —
the hard ink edge against cobalt is the defining detail; danger is
cobalt-700 fill, distinguished by weight, never by hue. Hover/press color
is cobalt-700. Focus is a cobalt ring: `BlockFocus`
(`client/lib/theme/block_focus.dart`) gives every interactive block Tab
traversal and Enter/Space activation, and paints the 2px `brand` ring
outside the block's own bounds so the fixed control heights never move.
A disabled block is not wrapped, so it leaves focus traversal.

### Chips

Cobalt-shade blocks for lifecycle: 0 radius, 2px border. Chip text is
cobalt-700 on cobalt-50 in light (5.99:1 — DESIGN.md is the authority for
chip text; code aligns to it). The light danger chip (overdue, rejected,
gone quiet) sits on cobalt-200, where cobalt-700 text would be 4.49:1, so
its label is ink (13.03:1): still in-palette, no new hue. Dark chips keep
cobalt-200 on cobalt-950 (9.18:1); done and danger share it there by the
one-hue decree, so the label, not a second colour, says which is which.
Every chip label owes 4.5:1 and `client/test/cobalt_test.dart` pins each
tone in both themes. Status marks are square blocks, never dots. Not chops.

### Cards

0 radius, 2px `strokeSubtle` border, `cast` shadow, raised white (light) or
neutral raised `#14171D` (dark).

### Navigation

Bottom dock for signed-in product surfaces: a 0-radius block with a 2px ink
border and `cast` shadow. Every dock item is a `PressableBlock`. The current
item is outlined 2px in `btnFg` and announced as selected, and the dock's
focus ring is `btnFg` too: the dock is filled with `brand`, where the usual
brand ring would vanish. An unfinished item carries an amber `#FFD166` square
at its top-right corner. No Linear-style left rail at any breakpoint.

### Identity exception

The brand mark (landing logo, auth-modal avatar) stays a disc — it is
identity, not a status indicator or container. Everything interactive or
informational is a block.

## Do's and Don'ts

### Do

- Do keep one hue and let neutrals carry the page.
- Do put the primary action on cobalt-500.
- Do square every corner — reach for the radius tokens, never a raw
  `BorderRadius.circular`.
- Do give new blocks a 2px border and a `cast` shadow, and new interactive
  blocks the press-the-block motion via `PressableBlock`.
- Do build dialogs, buttons, fields, banners, and read states from the
  `theme/block_*` primitives — never stock `AlertDialog`, floating-label
  `TextField`s, bare `IconButton`s, or round spinners.
- Do use the serif only in the Coach column.
- Do keep zh-CN and en-US as separate strings.

### Don't

- Don't round corners, blur shadows, or add elevation glow — the decree is
  0 / 2px / (4,4) zero-blur.
- Don't bring back chops, wax stamps, or ink-as-primary buttons.
- Don't use full-bleed slabs, 20px type-rows, or one small card on a blank field.
- Don't add a left rail.
- Don't put cobalt-950 on a page surface in dark mode — it is brandSubtle
  and status-chip backgrounds only.
- Don't mash Chinese and English in one string.
