# 2026-08-19 — Restore Ink & Seal on Flutter

Status: **in flight**. Opened after user QA: an infra/client migration is
not a visual rewrite.

## Why this exists

The 2026-08-17 Flutter plan locked **Material 3, not Ink & Seal**. That was
a product-visual decision bundled into a Nuxt → Flutter cutover. It was
wrong: Flutter is the new widget kit; the committed world remains
**国风留白 / Ink & Seal** in [`DESIGN.md`](../../DESIGN.md) and the recovered
token file `app/src/assets/tokens.css` (last live at `f897fb4`).

## Locked (corrected)

| Decision | Choice |
| --- | --- |
| Client | Flutter (`client/`), web + iOS + Android |
| Visual language | Ink & Seal. Material widgets may implement it; they must not replace it |
| Token source | `DESIGN.md` + recovered `tokens.css` (celadon, sumi ink, vermilion seal, paper carve) |
| Primary action | Ink fill, not vermilion (One Seal Rule) |
| Auth chrome | Paper panel + 印 seal over celadon (AuthDoor), not a blank Material form |
| Not in this slice | Collaborative canvas (sibling bands, minimap); bundled native fonts |

## Checklist

- [x] Reverse the Material 3 visual lock in the Flutter plan and board
- [x] Record the QA thread
- [x] Port semantic tokens into Flutter `ThemeData` (light + dark)
- [x] Restore `yin.webp` and an AuthDoor-shaped login/register shell
- [x] Landing uses celadon ground, Song display, ink primary, seal accent
- [x] Assembly landing (idea → PRD → roadmap → seal) with recovered chops
- [x] Roadmap node chops use seal / locked-chop language
- [x] Restore Nuxt motion (seal stamp, diagram click-in, in-view reveals)
- [x] Login/register as a prompt overlay on the landing (not a redirect)
- [x] Step mini-maps play the click-in transition to the current step
- [x] Noto Serif SC / Noto Sans SC bundled for native targets (2026-08-20):
      variable TTFs as lazy assets + startup `FontLoader` on non-web; web
      stays on index.html CDN. Tests 43/43; web and iOS bundles verified
      ([changelog](../changelogs/2026-08-20-native-fonts-bundling.md))
- [x] Visual font sign-off (2026-08-20, iPhone 16 Pro simulator): headline
      renders in Song-style serif, zero tofu boxes, celadon ground and
      vermilion seal accent — the full Ink & Seal language on a native
      target
- [x] Canvas v1 (2026-08-22): sibling bands + status minimap + list
      toggle, interaction-tested and deployed
      ([changelog](../changelogs/2026-08-22-product-follow-ons.md));
      real-time multi-user presence remains a follow-on

## Source of truth

- [`DESIGN.md`](../../DESIGN.md)
- [`docs/design-system/CHANGELOG.md`](../design-system/CHANGELOG.md) `0.2.0` / `0.4.0`
- Git: `f897fb4:app/src/assets/tokens.css`, `app/src/components/auth/AuthDoor.vue`, `app/public/seals/yin.webp`
- QA: [`docs/design-qa/2026-08-19-restore-ink-seal.md`](../design-qa/2026-08-19-restore-ink-seal.md)
