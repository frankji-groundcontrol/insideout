# 2026-07-24 — AI-generated baiwen seals for landing + navbar

Replaced the landing's flat/hand-drawn `bg-seal` chops and the text-only
navbar brand with authentic AI-generated Chinese seal (baiwen 白文)
impressions — vermillion red blocks with the characters carved out as bare
paper, so each mark is self-contained ink that reads on any background.

## What changed

- **Generated 6 seals with the codex CLI's image generation.** Each is a
  traditional baiwen chop: a square of cinnabar red with the characters in
  white negative space, carved bold and legible (clerical-influenced, not
  ancient flowing zhuanshu). The two-character seals are stacked vertically.
  - `app/public/seals/luomo.webp` — 落墨 (step 1)
  - `app/public/seals/chengwen.webp` — 成文 (step 2)
  - `app/public/seals/fenzhi.webp` — 分枝 (step 3)
  - `app/public/seals/gaiyin.webp` — 盖印 (step 4)
  - `app/public/seals/yin.webp` — 印 (CTA close)
  - `app/public/seals/neiwei.webp` — 内外 (navbar brand mark)
- **Processed for the design system.** Each raw generation went through a
  white-margin knockout that flood-fills only the border-connected paper
  margin to transparency — the enclosed white characters stay opaque, so the
  red seal floats on any background — plus a color normalization that re-hues
  the red toward the `--color-seal` token (`200 64 47`) while preserving the
  carved luminance texture. Exported as 320px WebP with alpha (~22–40 KB
  each, ~172 KB total).
- **Wired into three touchpoints.**
  - The four [`AssemblyStep`](../../app/src/components/landing/AssemblyStep.vue)
    chops, via a new `seal` prop fed from
    [`index.vue`](../../app/src/pages/index.vue).
  - The CTA-close seal in [`index.vue`](../../app/src/pages/index.vue).
  - The navbar brand in [`NavBar.vue`](../../app/src/components/layout/NavBar.vue),
    which now pairs the 内外 mark with the "InsideOut" wordmark (the brand
    string is "InsideOut" in both locales, so there is no duplicated text).

Because a baiwen seal is self-contained red ink, one asset at the light
`--color-seal` value reads correctly on both light and dark themes — no
separate dark variant is needed.

## Generation method (for regeneration)

The qwenimage MCP (Wan2.6) route was failing (`-32602 Invalid request
parameters`), so the seals were generated with the codex CLI's built-in image
tool, invoked with a low-reasoning-effort override (the default reasoning
tier stalls on this task) and a prompt that pins the baiwen style, the exact
characters, and vertical stacking. The raw PNGs and the one-off
margin-knockout script were scratch under `/tmp`, not committed; the method
is recorded here so a seal can be regenerated or extended later.

## Verification

- Each seal visually confirmed to carry the correct, legible characters.
- Knockout composites checked on both light and dark backgrounds — no white
  halo, and the enclosed white characters stay opaque.
- Browser-verified live across light + dark themes: all four step chops, the
  CTA 印 seal, and the navbar 内外 mark render crisply; all six WebP assets
  return 200/304 and there are no seal-related console errors.

## Operator note

No action required. These are static assets under `app/public/seals/` plus
three component/page edits; a normal frontend build/deploy picks them up.
