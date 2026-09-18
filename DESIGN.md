---
name: InsideOut
description: Cobalt card lock — track the work, refine the ideas, ship better PRDs.
colors:
  cobalt-50: "#EEF2FF"
  cobalt-200: "#C7D2FE"
  cobalt-500: "#3B5BFF"
  cobalt-700: "#1D4ED8"
  cobalt-950: "#1E2A5A"
  light-ground: "#F7F8FA"
  light-ink: "#0B0D12"
  dark-ground: "#0B0D12"
  dark-type: "#F7F8FA"
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
  control: "9999px"
  card: "1.25rem"
  hero: "1.75rem"
spacing:
  unit: "4px"
components:
  button-primary:
    backgroundColor: "{colors.cobalt-500}"
    textColor: "#FFFFFF"
    rounded: "{rounded.control}"
    padding: "14px 24px"
  chip-shade:
    backgroundColor: "{colors.cobalt-50}"
    textColor: "{colors.cobalt-700}"
    rounded: "{rounded.control}"
  card:
    backgroundColor: "#FFFFFF"
    rounded: "{rounded.card}"
---

# Design System: InsideOut

## Overview

**Creative North Star: Cobalt cards on a field.**

One hue. Neutrals carry the page. Light and dark are required. The product
loop — Idea → version → human Commit → timed Roadmap → Git evidence — is
read as rounded cards on a ground, large type, and a bottom dock. Not a
full-bleed slab, not a single floating card on a blank page, not type-rows,
not a left rail.

Lock screens live in
[`docs/design-system/cobalt/`](docs/design-system/cobalt/README.md).
The Flutter tokens are `Cobalt` in
[`client/lib/theme/cobalt.dart`](client/lib/theme/cobalt.dart).

**Key Characteristics:**

- One accent: cobalt-500. Shades do hover, Now, focus, and chips.
- Light ground `#F7F8FA` / ink `#0B0D12`. Dark ground `#0B0D12` / type `#F7F8FA`.
- Body **17.5**. Titles **26+**. Display **48–58**. Meta **12**.
- Ideas wall display `clamp(64px, 10vw, 320px)` on wide web.
- UI sans: Archivo (Latin) + Noto Sans SC (CJK). zh-CN is the default locale.
- Coach-only serif: Source Serif 4 + Noto Serif SC.
- Primary action is **cobalt**, not ink.
- Status is a shade pill, not a chop.

## Colors

| Token | Hex | Use |
| --- | --- | --- |
| cobalt-50 | `#EEF2FF` | Light wash / raised wash |
| cobalt-200 | `#C7D2FE` | Hairline, chip, poster edge |
| cobalt-500 | `#3B5BFF` | Primary action, focus, Now |
| cobalt-700 | `#1D4ED8` | Pressed / dark primary |
| cobalt-950 | `#1E2A5A` | Dark-mode field / deep band |

No magenta, lime, orange, cream, or extra brand hues.

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

Rounded cards on a field. Home is an index card plus a giant numeral with
workspace cards beside it. Ideas is a type wall and poster cards. PRD + Coach
is a split desk. Commit is a centered fanned stack. Roadmap is Now / Next /
Later shade bands that still hold item cards. Signed-in chrome is a bottom
dock. Locale and theme stay in the top-right cluster.

## Components

### Buttons

Primary is a cobalt pill. Hover/press uses cobalt-700. Focus is a cobalt ring.

### Chips

Cobalt-shade pills for lifecycle. Not chops.

### Cards

20px radius, hairline cobalt-200, raised white (light) or lifted dark (dark).

### Navigation

Bottom dock for signed-in product surfaces. No Linear-style left rail at any
breakpoint.

## Do's and Don'ts

### Do

- Do keep one hue and let neutrals carry the page.
- Do put the primary action on cobalt-500.
- Do use the serif only in the Coach column.
- Do keep zh-CN and en-US as separate strings.

### Don't

- Don't bring back chops, wax stamps, or ink-as-primary buttons.
- Don't use full-bleed slabs, 20px type-rows, or one small card on a blank field.
- Don't add a left rail.
- Don't mash Chinese and English in one string.
