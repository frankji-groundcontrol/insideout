# 2026-07-23 — Ink & Seal reconciliation + landing rethink (「The Assembly」)

Large change record. Two moves in one day: **revert** the 2026-07-22 "Prisma
cinematic" theme detour back to the committed **国风留白 / Ink & Seal** world,
and **rethink the public landing** as 「The Assembly」/「一步步」 — the product
story drawn as ink build-instructions.

- [summary.md](summary.md) — what changed, phase by phase.
- [verification.md](verification.md) — the verification actually performed.

## Primary sources

- [Plan](../../plans/2026-07-23-ink-seal-landing.md) (the live checklist this was driven from)
- Direction record: [`PRODUCT.md`](../../../PRODUCT.md) + [`DESIGN.md`](../../../DESIGN.md)
- Token change record: [`docs/design-system/CHANGELOG.md`](../../design-system/CHANGELOG.md) `0.4.0`

## The day in one paragraph

Yesterday's Prisma re-theme was an uncommitted detour. The committed brand is
**Ink & Seal** — celadon ground, sumi-ink text, a single vermilion seal accent,
Song/Mincho display serif — so the theme detour reverted (one `tokens.css`
restore re-themes every semantic-token consumer) while yesterday's real product
work (roadmap tree, GitHub sync, AI MVP builder) stayed. On top of that
foundation the landing page was rebuilt around a new concept: **「The
Assembly」**, a build-instruction narrative in which the four-step journey —
idea spark → PRD → branched roadmap → shipped seal — assembles itself piece by
piece, each step a numbered instruction with a 1:1 call-out box peeking at the
real product artifact. Bilingual EN/zh-CN throughout.
