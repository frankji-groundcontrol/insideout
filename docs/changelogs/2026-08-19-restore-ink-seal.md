# 2026-08-19 — Restore Ink & Seal on Flutter

## What changed

User QA: the Nuxt → Flutter cutover dropped 国风留白 / Ink & Seal. That was
a visual rewrite bundled into an infra/client migration, not a requirement
of Flutter. The Material-3-only lock in the 2026-08-17 plan is reversed.

- Flutter `ThemeData` now uses the recovered Ink & Seal tokens (celadon
  ground, sumi-ink primary action, vermilion seal accent, paper carve)
  in light and dark.
- Login/register use an AuthDoor-shaped paper panel with the recovered
  印 seal (`yin.webp`) over celadon, not a blank Material form.
- Landing uses Song display, ink filled button, seal wash chip.
- Web loads Noto Serif SC / Noto Sans SC.
- Assembly landing restored: spark → PRD → tree → seal diagram, four
  step chops (`luomo` / `chengwen` / `fenzhi` / `gaiyin`), 1:1 peeks,
  close CTA.
- Roadmap nodes use square seal marks + status chops (locked grey,
  in-progress vermilion, done sage) instead of raw status strings.
- Motion restored from the Nuxt landing / AuthDoor: seal stamp, hero
  fade/rise, diagram click-in, in-view step reveals. Reduced-motion
  skips the sequence.
- Login/register is a prompt overlay on the Assembly landing (scrim +
  paper door), not a full-page redirect. `/login` and `/register` deep
  links show the same overlay.
- Step mini-maps animate the click-in up to the current step instead of
  painting the final ghosted state immediately.

Not in this slice: collaborative canvas (sibling bands / minimap),
bundled fonts on iOS/Android.

## Verification

- `cd client && flutter analyze --no-fatal-infos` — no errors (pre-existing infos only)
- `cd client && flutter test` — 43 passed (landing prompt overlay + click-in keyframes)
- Local Chrome at `http://localhost:5173` against the hosted API
  (`API_BASE=https://server-production-9c338.up.railway.app/api/v1`).

## Sources

- [`DESIGN.md`](../../DESIGN.md)
- Plan: [`docs/plans/2026-08-19-restore-ink-seal.md`](../plans/2026-08-19-restore-ink-seal.md)
- QA: [`docs/design-qa/2026-08-19-restore-ink-seal.md`](../design-qa/2026-08-19-restore-ink-seal.md)
