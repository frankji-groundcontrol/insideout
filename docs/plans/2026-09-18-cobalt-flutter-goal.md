# Goal prompt — Cobalt Flutter restyle

Paste the block below into a new agent (or Cursor Goal). It is
self-contained. Do not invent a new visual language.

---

Restyle the live Flutter client from **Ink & Seal** to the accepted
**Cobalt card lock**. Implement the lock. Do not explore, vote, or
redesign.

## Success

A signed-in user on Flutter web (light and dark, zh-CN and en-US) sees
the Cobalt card grammar — rounded cards on a field, large type, bottom
dock — matching the lock screens. Product loop and APIs stay the same.
No celadon, cream, vermilion, chops, wax stamps, or left rail remain
on live surfaces.

## Source of truth (in order)

1. This prompt and
   [`docs/plans/2026-09-18-cobalt-flutter.md`](2026-09-18-cobalt-flutter.md).
2. Lock screens in
   [`docs/design-system/cobalt/`](../design-system/cobalt/README.md)
   (`LOCK-home-light.jpg`, `LOCK-home-dark.jpg`, `LOCK-ideas.jpg`,
   `LOCK-prd.jpg`, `LOCK-commit.jpg`, `LOCK-roadmap.jpg`).
3. Verbatim QA:
   [`docs/design-qa/2026-09-17-cobalt-layouts.md`](../design-qa/2026-09-17-cobalt-layouts.md),
   [`docs/design-qa/2026-09-17-cobalt-shade.md`](../design-qa/2026-09-17-cobalt-shade.md).
4. HTML prototype (if present on the host): OpenDesign project
   `<project-id>`, files
   `DESIGN_LANGUAGE.md`, `brand-spec-cobalt.md`,
   `cobalt-language.html`, `insideout-cobalt.html`.

If the HTML is missing, the lock JPEGs + this prompt are enough.

## Visual lock

One hue. Neutrals carry the page. Light and dark required.

| Token | Hex | Use |
| --- | --- | --- |
| cobalt-50 | `#EEF2FF` | Light wash / raised wash |
| cobalt-200 | `#C7D2FE` | Hairline, chip, poster edge |
| cobalt-500 | `#3B5BFF` | Primary action, focus, Now |
| cobalt-700 | `#1D4ED8` | Pressed / dark primary |
| cobalt-950 | `#1E2A5A` | Dark-mode field / deep band |
| light ground | `#F7F8FA` | Scaffold |
| light ink | `#0B0D12` | Type |
| dark ground | `#0B0D12` | Scaffold |
| dark type | `#F7F8FA` | Type |

No magenta, lime, orange, cream, celadon, vermilion, sumi-as-brand.

**Type.** Body **17.5**. Titles **26+**. Display **48–58**. Meta **12**.
Ideas wall display `clamp(64px, 10vw, 320px)` on wide web. Giant Home
index numeral sits in flow (do not clip it behind the dock).
UI sans: **Archivo** for Latin, **Noto Sans SC** for CJK (zh-CN is the
default locale — never tofu). Coach-only serif: **Source Serif 4** for
Latin, **Noto Serif SC** for CJK. Keep the web first-frame font gate
([`client/lib/theme/web_fonts_web.dart`](../../client/lib/theme/web_fonts_web.dart));
extend `_faces` to the new families. Native path stays lazy assets, not
eager pubspec fonts.

**Grammar.** Rounded cards on a field — not full-bleed slabs, not a
single floating card on a blank page, not type-rows. Poster cards, not
list tiles, for Ideas. Centered fanned Commit stack. Roadmap
Now / Next / Later as shade bands that still hold item cards. Bottom
dock chrome. No Linear-style left rail at any breakpoint.

## Layout map

| Surface | Lock | Flutter today | Target |
| --- | --- | --- | --- |
| Home | F Index + giant `03` | [`dashboard_page.dart`](../../client/lib/features/dashboard/dashboard_page.dart) (`/dashboard`) | Index card + giant numeral; workspace cards on the field |
| Ideas | A Type Wall + three posters | [`ideas_page.dart`](../../client/lib/features/workspace/ideas_page.dart) | Giant type + poster cards; keep one-step capture |
| PRD + Coach | D Split Desk | [`prd_page.dart`](../../client/lib/features/prd/prd_page.dart), [`coach_panel.dart`](../../client/lib/features/prd/coach_panel.dart) | Split desk; Coach is the serif column; SSE stays live |
| Commit | C Soft Stack | [`versions_page.dart`](../../client/lib/features/prd/versions_page.dart) | Centered fanned stack — not a 300px list + detail row |
| Roadmap | B Shade Bands | [`roadmap_page.dart`](../../client/lib/features/project/roadmap_page.dart) | Default = Now/Next/Later bands with cards. Canvas may remain a secondary view |
| Evidence | dock view in the HTML proto | no dedicated page; project updates live on [`project_page.dart`](../../client/lib/features/project/project_page.dart) | Restyle existing Git / project-update evidence. Add a route only if the dock needs one. No new backend |
| Chrome | E Dock | [`app.dart`](../../client/lib/app.dart) `AppScaffold` AppBar | Signed-in product chrome = bottom dock. Locale + theme stay reachable |

Landing, AuthDoor, profile, workspace, settings, export, revisions,
audience view: retoken to Cobalt and strip seals. They do not need a
full A–F layout if they are not in the lock set.

## Forbidden

- Ink & Seal leftovers on live surfaces: celadon, cream, vermilion,
  印 chops, wax stamps, `assets/seals/*.webp` in chrome/landing/chips/
  roadmap nodes, Song-as-default display, “primary button is ink”.
- The rejected density rewrite (full-bleed, no cards, 20px type-rows).
- One-card / small-text / lots-of-blank.
- Extra hues. Mashed zh+en in one string. English-only prototype copy
  in the running app — use `l10n` (zh-CN default, en-US).
- Mocks. Writes outside the `insideout` schema. Reading or printing
  `.env` / `.env.local`.
- Inventing product behavior. The loop stays
  Idea → version → human Commit → timed Roadmap → Git evidence.

## Current code to replace

Theme is still Ink & Seal:

- [`client/lib/theme/ink_seal.dart`](../../client/lib/theme/ink_seal.dart)
- [`client/lib/theme/ink_seal_theme.dart`](../../client/lib/theme/ink_seal_theme.dart)
- [`client/lib/theme/seal_chip.dart`](../../client/lib/theme/seal_chip.dart)
- wired from [`client/lib/app.dart`](../../client/lib/app.dart)
- asserted by [`client/test/ink_seal_test.dart`](../../client/test/ink_seal_test.dart)
  (expects vermilion `#C8402F` and celadon ground)

Rename the token type to **Cobalt**. Do not keep celadon/seal values
under old names. Status chips become cobalt-shade pills, not chops.
Rewrite [`DESIGN.md`](../../DESIGN.md) to Cobalt. Bump
[`docs/design-system/CHANGELOG.md`](../design-system/CHANGELOG.md)
(major if semantic keys change). Update
[`docs/architecture/frontend.md`](../architecture/frontend.md).

## How to work

1. Write failing token/theme tests first (TDD), then the smallest
   passing theme swap, then chrome (dock), then the six lock surfaces.
2. Stay on `cursor/vibrant-ink-seal-198b` (PR #2). Do not start a
   parallel visual exploration.
3. Files under ~350 lines, single responsibility. If a file cannot be
   split in this task, file a dated modularity issue.
4. Record frontend comments in `docs/design-qa/`. Keep
   `docs/plans/README.md` and `docs/HANDOFF.md` current. Changelog the
   restyle.
5. Engineering review of *this* plan is skipped: the user already
   accepted the lock. Implement pixels; do not reopen color or layout.

## Verification (required before done)

```bash
(cd client && flutter analyze --no-fatal-infos && flutter test)
```

Then a real browser walk (not a single screenshot):

- light and dark
- zh-CN and en-US
- Home, Ideas (capture), PRD + Coach (live SSE if the API is up),
  Commit, Roadmap (add a Now item), Evidence, theme toggle, dock
  navigation
- empty / error states on the pages you touch
- confirm no seal assets or vermilion/celadon remain on those routes

If Coach SSE is reachable, exercise it live
([practice](../practices/2026-07-21-live-exercise-streaming-endpoints.md)).
If `DATABASE_URL` is set and you touch store code (you should not),
run the real-DB authz tests. Never print secrets.

Repo contract: [`AGENTS.md`](../../AGENTS.md),
[`docs/index.md`](../index.md).

*Superseded 2026-10-11: the restyle this prompt drove is long merged —
PR #2 (this goal prompt, docs) and PR #3 (the Cobalt restyle itself)
both merged 2026-09-18, so "Stay on `cursor/vibrant-ink-seal-198b` (PR
#2)" is dead guidance; the restyle record is
[2026-09-18-cobalt-flutter.md](2026-09-18-cobalt-flutter.md).*
