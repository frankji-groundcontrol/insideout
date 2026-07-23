---
name: InsideOut
description: 国风留白 / Ink & Seal — track the work, refine the ideas, ship better PRDs.
colors:
  seal: "#C8402F"
  seal-strong: "#A83426"
  seal-locked: "#8C8A80"
  seal-wash: "#F2DDD7"
  celadon-ground: "#E4EAE4"
  celadon-raised: "#EEF1EB"
  celadon-sunken: "#D8E0D8"
  ink-night: "#161A17"
  sumi-ink: "#211E1A"
  rice-paper: "#E6E9E1"
  carve: "#F2F4EE"
  sage: "#3E6B4A"
  amber: "#8A6A20"
typography:
  display:
    fontFamily: '"Noto Serif SC", "Songti SC", "Alibaba PuHuiTi", Georgia, serif'
    fontWeight: 700
    lineHeight: 1.1
  body:
    fontFamily: '"Alibaba PuHuiTi", "Noto Sans SC", system-ui, sans-serif'
    fontWeight: 400
    lineHeight: 1.5
rounded:
  control: "0.625rem"
  card: "1rem"
  pill: "9999px"
  hero: "1.5rem"
spacing:
  unit: "4px"
components:
  button-primary:
    backgroundColor: "{colors.sumi-ink}"
    textColor: "{colors.rice-paper}"
    rounded: "{rounded.control}"
    padding: "12px 24px"
  button-primary-hover:
    backgroundColor: "{colors.seal}"
    textColor: "{colors.carve}"
  chip-seal:
    backgroundColor: "{colors.seal}"
    textColor: "{colors.carve}"
    rounded: "{rounded.control}"
  card:
    backgroundColor: "{colors.celadon-raised}"
    rounded: "{rounded.card}"
---

# Design System: InsideOut

## Overview

**Creative North Star: "The Sealed Scroll" (卷与印).**

Every project is a scroll. Work lays down ink; completion presses a vermilion
seal. InsideOut's whole visual world grows from that one image — the 国风留白
(Chinese-art negative-space) tradition, where the empty ground is not blank but
*reserved*, and a single cinnabar chop carries all the authority on the page.

The system is deliberately quiet. A soft celadon (青) ground does the work a
white page does in Western UI, sumi-ink carries the text, and one vermilion 印泥
accent — used sparingly — marks identity, status, and the rare moment that
matters. Restraint is the brand: the seal means something *because* it is rare.
This is not the warm-cream-and-serif SaaS default, nor the near-black-neon AI
default; it is an ink painting that happens to be software.

The product's own mechanics already speak this language: roadmap nodes render
their lifecycle as seals (a grey locked chop, a vermilion in-progress chop), and
the brand mark is a scroll (卷). The design system simply codifies what the
product already believes — that progress is ink laid down and a shipped idea is
a seal pressed.

**Key Characteristics:**

- One vermilion accent on a celadon field; the seal is rare on purpose.
- Generous negative space (留白) — density is earned, never default.
- Sumi-ink text; the primary action is **ink**, not the accent color.
- A Song/Mincho display serif (Noto Serif SC) for headings, over a self-hosted
  PuHuiTi sans for body and UI.
- Light (celadon) and dark (ink-night) themes; the seal lifts slightly on dark
  so chops keep their glow.
- Bilingual by construction: EN and 中文 are first-class in every component.

## Colors

The palette is one warm accent held against a cool, quiet ground — cinnabar on
celadon.

### Primary
- **Cinnabar Seal 印泥** (#C8402F): the single signature accent. Identity
  wordmark, status chops, links, focus ring, and the moments of commitment
  (a shipped milestone, an approved PRD). Never the primary button fill.
- **Pressed Seal** (#A83426): hover/active state of the seal.
- **Seal Wash** (#F2DDD7): a pale vermilion tint for subtle selected/active
  backgrounds where a full chop would shout.

### Neutral
- **Celadon Ground 青** (#E4EAE4): the page background (light). The reserved
  "silk" the ink sits on.
- **Raised Celadon** (#EEF1EB) and **Sunken Celadon** (#D8E0D8): tonal steps for
  cards, panels, and input wells — depth comes from tone, not shadow.
- **Sumi Ink 墨** (#211E1A): primary text **and** the primary-action surface
  (the ink button).
- **Rice Paper** (#E6E9E1): text on ink or seal fills; on dark it becomes the
  primary text color.
- **Ink Night** (#161A17): the dark page ground; the seal lifts to #D84A31 here.

### Status
- **Sage** (#3E6B4A): done / approved — a cool green that stays inside the
  celadon family.
- **Amber** (#8A6A20): in-progress / needs attention — aged-paper amber.
- **Locked Chop** (#8C8A80): neutral grey — the locked / not-yet state.

### Named Rules

**The One Seal Rule.** Vermilion appears on at most a few elements per screen —
a wordmark, a status chop, a link. Its rarity is its authority; a screen full of
seals has none.

**The Ink-Not-Vermilion Rule.** The primary action is **sumi ink** (`bg-btn`),
never the seal. Vermilion is the accent and the signature; ink is the verb.

## Typography

**Display Font:** Noto Serif SC (Song/Mincho serif), falling back to Songti SC
then PuHuiTi.
**Body Font:** Alibaba PuHuiTi (self-hosted, covers Latin + full CJK), falling
back to Noto Sans SC then system-ui.

**Character:** A Song serif does the talking at headline scale — it carries the
brush-adjacent, carved 国风 voice — while a clean modern CJK sans keeps body and
UI legible and neutral. The pairing is "carved heading, brushed-aside body."

### Hierarchy
- **Display** (700, clamp up to ~3.75rem, 1.1): hero and page titles — always
  the serif.
- **Headline** (600, ~1.5rem, 1.25): section titles — serif.
- **Title** (600, ~1.125rem): card and panel headings — sans, semibold.
- **Body** (400, ~0.875–1rem, 1.5): running text — sans; keep measure to ~65–75ch.
- **Label / Chop** (500–600, ~0.75rem, tracked): seals, status chips, eyebrows.

### Named Rules

**The Serif-Only-For-Headings Rule.** The Song serif is reserved for display and
headline. Body, UI, and form text never use it — legibility is the sans's job.

## Layout

A single centered measure with generous margins; 留白 (negative space) is the
default state, not the absence of content. Content is grouped into tonal panels
rather than boxed cards where possible. Vertical rhythm gives more space above a
heading than below it, so sections read as exhale–inhale. On mobile the single
column holds; panels stack and the seal accents stay small.

## Elevation & Depth

Depth is conveyed by **tonal layering**, not shadow. Raised panels step lighter
(#EEF1EB), sunken wells step darker (#D8E0D8) against the celadon ground; hairline
ink strokes (#C2CABF) separate where tone alone is ambiguous.

### Shadow Vocabulary
- **Card** (`0 1px 2px rgb(0 0 0 / 0.05)`): a whisper, only on floating cards.
- **Modal** (`0 20px 25px -5px …`): reserved for true overlays (dialogs, popovers).

### Named Rules

**The Flat-By-Default Rule.** Surfaces are flat at rest and separate by tone.
Shadow appears only as a response to elevation or overlay — never as decoration.

## Shapes

Corners are softly rounded, never sharp and never pill-except-for-tags. Controls
use a gentle 10px radius (`rounded-control`), cards a 1rem sweep (`rounded-card`),
and the rare hero shell a 1.5rem curve. The recurring silhouette is the **seal
chop**: a small rounded square or circle of vermilion bearing carved (paper)
glyphs.

## Components

### Buttons
- **Shape:** gently rounded (10px, `rounded-control`).
- **Primary:** sumi-ink fill with rice-paper text (`bg-btn text-btn-fg`),
  padding ~12px 24px. On dark the button inverts to a paper fill.
- **Hover / Focus:** hover warms toward the seal; focus shows a vermilion ring.
- **Secondary / Ghost:** celadon tonal fill or an ink hairline outline; text in
  sumi ink. Never a vermilion fill for the default action.

### Chips / Status Seals
- **Style:** a small rounded seal bearing carved text; tone follows lifecycle —
  grey locked, vermilion in-progress, sage done, amber attention.
- **State:** the seal is the status — no separate colored dot needed.

### Cards / Containers
- **Corner Style:** 1rem (`rounded-card`).
- **Background:** raised celadon (#EEF1EB) on the celadon ground.
- **Border:** optional thin ink hairline; tone usually suffices.
- **Internal Padding:** generous — let the 留白 breathe.

### Inputs / Fields
- **Style:** sunken celadon well, ink hairline stroke, control radius.
- **Focus:** vermilion focus ring (`--color-stroke-focus`).
- **Error:** vermilion-danger text associated via `aria-describedby`.

### Navigation
- Top bar on a raised celadon strip with a hairline bottom stroke. The wordmark
  is the only vermilion element; links are muted ink warming to full ink on
  hover; the primary entry is the ink button.

## Do's and Don'ts

### Do:
- **Do** keep vermilion rare — one seal per view is a feature, not a bug.
- **Do** use the ink button for the primary action and reserve the seal for
  accent, status, and identity.
- **Do** let tone, not shadow, separate surfaces.
- **Do** set headings in the Song serif and body in the sans.
- **Do** hold the 留白 — when a layout feels full, remove before you resize.

### Don't:
- **Don't** fill the primary button with vermilion — that inverts the One Seal
  Rule and spends the accent.
- **Don't** introduce a second accent hue; the system is one seal on celadon.
- **Don't** add drop shadows to resting cards or text.
- **Don't** use the display serif for body copy or UI labels.
- **Don't** let the layout get dense to look "productive" — empty ground is the
  brand.
