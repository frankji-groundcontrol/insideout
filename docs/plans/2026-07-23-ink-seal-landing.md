# 2026-07-23 — Ink & Seal whole-product + landing rethink

Live checklist. Drives the work; check items off as they land.

## Direction (confirmed with the user 2026-07-23)

- **Visual world:** 国风留白 / **Ink & Seal** (celadon ground, sumi-ink, vermilion
  seal) — the committed brand, **not** the uncommitted "Prisma cinematic"
  black+cream detour from 2026-07-22.
- **Scope:** **whole-product** — Ink & Seal is coherent across the app (tokens,
  dashboard, roadmap tree, coach, board). The landing page is the vanguard.
- **Ambition:** **rethink the landing concept** — new narrative + signature
  interaction + generated hero imagery. Keep only the accurate journey facts:
  idea → PRD → branched AI roadmap (key feature) → track-to-shipped (GitHub sync).
  Bilingual EN/zh-CN. Landing = **Persuade** mode.

## Phase A — theme reconciliation (revert Prisma → Ink & Seal)

Yesterday's real product work (roadmap tree, GitHub sync, AI MVP builder, landing
copy, semantic-token component migration) is KEPT. Only the theme detour reverts.

- [x] `tokens.css` → restore HEAD Ink & Seal values (`git checkout HEAD`). This one
      revert re-themes every semantic-token consumer.
- [x] `tailwind.config.js` → serif stack leads with Noto Serif SC (Ink & Seal
      display); keep self-hosted PuHuiTi for sans.
- [x] `style.css` → keep PuHuiTi `@font-face`; drop Prisma noise utilities + stale
      comments.
- [x] `nuxt.config.ts` → revert forced-dark FOUC default to `prefers-color-scheme`;
      keep the real Go-proxy / tokens.css-load / InsideOut-title work; fix stale
      comments; add Noto Serif SC display link.
- [x] Verify: no unresolved semantic classes; `pnpm build`; `pnpm test`; visual
      check a page renders Ink & Seal.

## Phase B — record the world

- [x] `PRODUCT.md` Brand Commitments → Ink & Seal confirmed as the binding
      whole-product identity (replace the stale "open decision" line).
- [x] `DESIGN.md` → document the Ink & Seal world (per impeccable `document.md`),
      whole-product scope.

## Phase C — landing rethink (Persuade)

- [x] Load `craft-floor.md`; run `concept-seed.mjs --scope surface --mode persuade`.
- [x] Derive structures, dress challengers in Ink & Seal, pick ONE committed
      direction; write the direction contract (THESIS / OWN-WORLD / STORY / FIRST
      VIEWPORT / FORM) and confirm it with the user before the big build.
      → **「The Assembly」/「一步步」** confirmed 2026-07-23.
- [x] Build `app/src/pages/index.vue`; hero imagery is the ink-drawn
      AssemblyDiagram (SVG signature device), not a raster render.
- [x] `detect.mjs --json` once; inspect desktop + mobile; fix material gaps.
      → Ultracode adversarial critique: 3 confirmed majors fixed (reduced-motion
      gate, nested `<a><button>` CTA, in_progress chip contrast); chop 4.48:1
      adjudicated intentional (aria-hidden, decorative).

## Phase D — record + close

- [x] Changelog entry under `docs/changelogs/`; update `docs/HANDOFF.md`; bump
      `docs/design-system/CHANGELOG.md` for the token/font reconciliation; close
      this plan.
      → `docs/changelogs/2026-07-23-ink-seal-landing/`, design-system `0.4.0`,
      HANDOFF updated. **Status: complete 2026-07-23.**
