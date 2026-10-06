# 24 · Design Fidelity

**Lens:** Match to the target neo-brutalist design language (radius, borders, shadows, press motion, dark-first ramp)
**Score:** **2.5 / 10**
**Bottom line:** The current UI is a coherent, well-disciplined execution of the *rounded* Cobalt card lock — and that is precisely the problem: against the Cobalt-Neo target it fails all four structural pillars. There is no shadow system at all (elevation is explicitly zeroed and the only two shadows in the app are soft Material blurs), every radius token (12/20/28/999) is the exact inverse of the 0px decree, container borders are 1px periwinkle hairlines instead of 2px, and no press-the-block interaction or 100ms motion token exists anywhere. The cobalt palette itself is faithfully and cleanly tokenized, and dark/light parity is real — so the migration is a token-layer rebuild, not a rewrite. Score 2.5/10 against the Cobalt-Neo target.

Reviewed branch `main` at `f185e94`; a skeptic pass re-checked every Critical/High finding — corrections marked [Correction]. Screenshot evidence lives in `./shots/`.

---

## Scope

The shipped Flutter client was compared against the Cobalt-Neo target: 0px radius, 2px container borders, hard `4px 4px 0 0` offset shadows, press-the-block interactions at 100ms, a dark-first neutral gray ramp, and the cobalt `#3B5BFF` palette retained. Evidence came from a full read of `client/lib/theme/cobalt.dart`, `cobalt_theme.dart`, `field_card.dart`, `cobalt_chip.dart`, `ink_motion.dart`, `chrome/product_dock.dart`, `features/auth/auth_door.dart`, `session/appearance.dart`, and `features/prd/coach_panel.dart`; `DESIGN.md`, `docs/design-system/cobalt/README.md`, and `docs/design-system/CHANGELOG.md`; the reference neo-brutalist brief; the pre-built code map; nine screenshots across both themes ([01](./shots/01-landing-logged-out.png), [02](./shots/02-login-modal.png), [04](./shots/04-dashboard-light.png), [05](./shots/05-dashboard-dark.png), [08](./shots/08-ideas-wall-dark.png), [09](./shots/09-prd-desk-dark.png), [14](./shots/14-ideas-wall-light.png), [16](./shots/16-roadmap-light.png) plus the capture index); and computed WCAG contrast ratios for every relevant token pair. A repo-wide grep confirmed there is no existing neo/brutalist mention anywhere — the migration spec is genuinely greenfield.

## What is good

- **One-hue cobalt palette, centrally tokenized.** `client/lib/theme/cobalt.dart:62-66` defines the hue50–hue950 ramp once, and the code map's token-discipline pass found zero `withOpacity`/`fromRGBO` and only five hardcoded color lines across all of `client/lib`.
- **True dark/light parity from a single class.** `client/lib/theme/cobalt.dart:102-154` mirrors every semantic key across both themes; [05-dashboard-dark](./shots/05-dashboard-dark.png) and [09-prd-desk-dark](./shots/09-prd-desk-dark.png) render consistently with their light twins 04/15.
- **Ideas-wall display type matches the lock spec.** `client/lib/features/workspace/ideas_page.dart:181-192` implements `clamp(64px,10vw,320px)` exactly, visible in shots [08](./shots/08-ideas-wall-dark.png)/[14](./shots/14-ideas-wall-light.png).
- **Primary actions pass WCAG AA.** Cobalt-500 on white measures 5.09:1 (computed); the btn/btnFg pairing is disciplined in both themes (`client/lib/theme/cobalt.dart:117-118`, `:144-145`).
- **Chrome consistency.** The capture-to-evidence bottom dock renders identically on every signed-in surface (shots 04–11, 16–17), matching `DESIGN.md`'s loop-navigation intent.

The failure is not execution quality — it is that the executed design is the wrong design.

## Findings, worst first

### no-shadow-system — Critical

The `Cobalt` token class defines zero shadow or elevation tokens, and the theme actively suppresses the Material defaults, so the neo-brutalist signature — an unblurred `4px 4px 0 0` cast on every container — has no substrate in the codebase. The only two shadows in the entire app are soft, blurred Material elevations (auth modal at `auth_door.dart:79-80`, elevation 6; dock at `product_dock.dart:31`, elevation 8). This matters beyond aesthetics: with 0px radius and 2px borders but no cast shadow, boxes go flat. Current dark-mode surface separation computes to raised-on-base 1.09:1, sunken-on-base 1.42:1, stroke-on-raised 1.45:1 — cards already barely separate from the page, and the reference system solves that with shadows plus border weight, not contrast.

**Evidence:** `client/lib/theme/cobalt_theme.dart:54-61`

```dart
    cardTheme: CardThemeData(
      color: c.surfaceRaised,
      elevation: 0,
      shape: shapeCard.copyWith(
        side: BorderSide(color: c.strokeSubtle),
      ),
      margin: EdgeInsets.zero,
    ),
```

**Fix:** Add shadow tokens to `Cobalt`: cast (`4px 4px 0 0`), castSm (`2px 2px 0 0`), each with a per-theme shadowColor (solid ink in dark; ~15% alpha ink in light per the reference brief); wire them into FieldCard, cardTheme, AuthDoor, ProductDock, dialogs, and toasts; remove Material elevation from those surfaces.
**Verdict:** CONFIRMED

### radius-decree-inverse — Critical

The four radius tokens are 12/20/28/999 — every control class (buttons, FilledButtons, text inputs, chips, the FAB, the dock) ships rounded by theme wiring (`cobalt_theme.dart:28-33`, `:87-111`), FieldCard is 20px everywhere, the auth card is 28px, and the capture bar is a literal `StadiumBorder` (`ideas_page.dart:200`). Features add hardcoded radii on top: `circular(99)` twice, top-radius 10 on the versions stack, and 12/4/3/1px corners on the roadmap canvas. Shots [01](./shots/01-landing-logged-out.png), [02](./shots/02-login-modal.png), [08](./shots/08-ideas-wall-dark.png), [14](./shots/14-ideas-wall-light.png), and [16](./shots/16-roadmap-light.png) are a sea of pills. Against a 0px-by-decree target this is the single widest-scoped change.

**Evidence:** `client/lib/theme/cobalt.dart:72-75`

```dart
  static const radiusControl = 12.0;
  static const radiusCard = 20.0;
  static const radiusHero = 28.0;
  static const radiusPill = 999.0;
```

**Fix:** Set all four constants to 0.0 (or delete the group and migrate call sites to a single `radiusZero`), then sweep the hardcoded `BorderRadius` values in features; keep `BoxShape.circle` brand marks only if the spec explicitly exempts identity marks.
**Verdict:** CONFIRMED

### no-press-the-block — Critical

There is no press-the-block interaction anywhere: no hover state, no shadow-shrink-plus-translate, no 100ms motion token. The only `MouseRegion` in the whole client is the roadmap canvas's remote-cursor tracker; tappables get Material's default ink splash (`field_card.dart:35-39`) plus one unrelated 180ms selection fade on the dock (`product_dock.dart:162`). The motion file is a mount-reveal system — a 650ms expo ease, philosophically opposite to fast mechanical press motion — and there are no duration/curve/offset tokens to implement the target with. Without a shared token, each developer will invent different durations and curves.

**Evidence:** `client/lib/theme/ink_motion.dart:4`

```dart
const inkEase = Cubic(0.16, 1.0, 0.3, 1.0);
```

**Fix:** Define motion tokens in `Cobalt` (pressDuration 100ms, pressCurve, hoverShadow 2px, hoverOffset (2,2), activeOffset (4,4), disabled rule) and ship one shared InkWell/ButtonStyle extension implementing the shadow+translate swap; convert FieldCard, buttons, dock items, and node cards to it.
**Verdict:** CONFIRMED

### hairline-border-weight — High

Every container border in the app is a default-width 1.0px `BorderSide` in cobalt-200 (`#C7D2FE` light) — the "border-everything at 2px" rule is halved, and the light hairline measures only 1.49:1 on white (computed). In light shots the card borders are barely-there pastel hairlines, so cards read as floating whitespace rather than bordered blocks; meanwhile the focused input is already 1.5px (`cobalt_theme.dart:102`) and the dock's unfinished outline is 1.5px (`product_dock.dart:166`), so border widths are already drifting and will get worse when someone "ups the border" screen by screen.

**Evidence:** `client/lib/theme/field_card.dart:29`

```dart
        side: BorderSide(color: borderColor ?? palette.strokeSubtle),
```

**Fix:** Introduce borderWidth tokens (container 2.0, divider 1.0) in `Cobalt` and apply `BorderSide(width: Cobalt.borderContainer)` in FieldCard, cardTheme, ShadeChip, AuthDoor, ProductDock, inputs, and dialogs; reserve 1px strictly for inner dividers.
**Verdict:** CONFIRMED

### blue-tinted-dark-ramp — High

The dark "elevation ladder" is blue-tinted — ground `#0B0D12`, raised `#141820`, sunken literally `hue950` (`#1E2A5A`, the brand hue at 15% lightness) — not the dark-first neutral gray ramp (`#000`/`#111`/`#1A1A1A`/`#222` + `#333` border) the target requires. The three steps sit only 1.09–1.42:1 apart (computed): shot [05](./shots/05-dashboard-dark.png) shows the workspace chapter card, index cards, and page ground as three near-identical navy blacks, so "depth from layered grays" is imperceptible. Under Cobalt-Neo, elevation must be carried by a neutral ramp plus cast shadows, and there is no token slot for the four-step ladder (page/secondary/tertiary/surface) the reference uses for nav, wells, table headers, and cards.

**Evidence:** `client/lib/theme/cobalt.dart:133-137`

```dart
    surfaceBase: darkGround,
    surfaceRaised: Color(0xFF141820),
    surfaceSunken: hue950,
    strokeSubtle: Color(0xFF2A3358),
    strokeStrong: Color(0xFF3B4A7A),
```

**Fix:** The spec must define a neutral dark gray ramp (page/secondary/tertiary/surface + border) as new `Cobalt` keys, decide what happens to hue950 (demote to brandSubtle-only), and re-map statusNeutralBg `#1A1F2E` onto the ramp.
**Verdict:** CONFIRMED

### light-mode-default — High

The app boots light-first: `Appearance` defaults `dark = false` and `hydrate()` only turns dark on an explicit stored 'dark' value, with no `prefers-color-scheme` fallback — the inverse of the target's dark-first requirement. New visitors and cleared-storage sessions always see light mode; a dark-first Cobalt-Neo must flip the default, decide the system-brightness fallback (currently ignored), and define hydrate behavior for the no-stored-preference case, otherwise the "dark is the hard default" rule silently never happens.

**Evidence:** `client/lib/session/appearance.dart:36`

```dart
  Appearance({SettingsStore? store, this.locale = defaultLocale, this.dark = false})
```

**Fix:** Default `dark = true` (or system-aware via `PlatformDispatcher` brightness), keep the stored override, and record the new default in `DESIGN.md`; re-shoot all lock screens dark-first.
**Verdict:** CONFIRMED

### semantic-hues-absent — Medium

All status semantics are cobalt washes — success background is hue50, warning is hue200, and fgDanger is cobalt-700 in light mode — so "done", "warning", and "danger" are indistinguishable shades of blue; the only true red in the app is a hardcoded `#B3261E` in `admin_section.dart:167`. ShadeChip (`cobalt_chip.dart:27-34`) renders done/pending/locked in near-identical cobalt tints, so lifecycle state is invisible at a glance on the ideas wall and versions pages. The neo reference keeps error/success/warning hues at 15–20% alpha tint behind 2px same-hue borders — the migration spec must either adopt semantic hues or explicitly codify the one-hue monochrome rule with distinct enough cobalt steps.

**Evidence:** `client/lib/theme/cobalt.dart:116-126`

```dart
    fgDanger: hue700,
    ...
    statusWarnBg: hue200,
    statusWarnFg: hue700,
    statusSuccessBg: hue50,
    statusSuccessFg: hue700,
```

**Fix:** Define semantic tokens (success/warning/error bg+fg+2px border hue) in the spec; if the one-hue brand rule is kept, codify minimum separation (e.g. done=cobalt-950 well, warning=cobalt-200 fill, danger=borrowed red) and delete the hardcoded `#B3261E` in favor of the token.

### coach-panel-off-ladder — Medium

The PRD coach panel is an always-dark navy surface reached by hardcoding `Cobalt.hue950` and `Cobalt.darkType` in light mode, with dark mode detected by comparing `surfaceBase` identity — a surface that exists entirely outside the token ladder. Shot 15 (light PRD desk) shows the panel staying navy while the rest of the page is light, and the feature had to invent an identity-based brightness hack (`coach_panel.dart:231`, `surfaceBase == Cobalt.darkGround`) that breaks the moment the dark ground value changes — exactly the kind of shadow dependency a neo token system (inverse/always-dark well role) must eliminate.

**Evidence:** `client/lib/features/prd/coach_panel.dart:111-112`

```dart
    final ink = palette.brightnessIsDark ? palette.fgPrimary : Cobalt.darkType;
    final field = palette.brightnessIsDark ? palette.surfaceSunken : Cobalt.hue950;
```

**Fix:** Add an explicit inverse-panel token pair to `Cobalt` (panelBg/panelFg in both themes, dark-navy values in light mode) and replace the identity check with `Theme.brightness`; the spec must decide whether inverse wells survive migration or join the gray ramp.

### light-sunken-blue-wells — Medium

Light-mode wells are brand-tinted, not neutral: `surfaceSunken` is hue50 (`#EEF2FF`) and the ground is a cool `#F7F8FA`, so "sunken" reads as "cobalt wash" everywhere it appears (chips, status backgrounds, search areas). Under the neo light theme the reference uses neutral warm off-whites with a translucent shadow twin; if the spec does not decide whether Cobalt-Neo light keeps blue-tinted wells or moves to neutral grays, feature authors will keep reaching for brandSubtle (hue50) as a gray substitute and the light theme will never form a legible ladder.

**Evidence:** `client/lib/theme/cobalt.dart:106-108`

```dart
    surfaceBase: lightGround,
    surfaceRaised: Color(0xFFFFFFFF),
    surfaceSunken: hue50,
```

**Fix:** Spec decision: define the light ramp explicitly (e.g. ground `#F7F8FA` / secondary `#FFFFFF` / tertiary `#F0F1F4` / surface `#FFFFFF`, or a warm variant), give brandSubtle a single documented role, and stop using it as the sunken step.

### type-spacing-unscaled — Medium

There is no 4px spacing scale and the type ramp is coarse: both body roles are 17.5px, labels are 12px, and spacing is ad hoc per widget (FieldCard padding 24, buttons h24/v14, chips h8/v3, bands 16, dock h10/v8). The reference vocabulary depends on a compact scale (body 14, meta 12, weight-driven hierarchy) and 4px-rhythm padding (p-4/p-6, gap-3); InsideOut's 17.5px body makes dense neo tables, badges, and dropdowns physically impossible at 1440×900 without a spec decision, and ad-hoc padding means the 2px-border boxes will each have their own rhythm.

**Evidence:** `client/lib/theme/cobalt_theme.dart:139-144`

```dart
    bodyLarge: sans(17.5, FontWeight.w400, color: c.fgSecondary),
    bodyMedium: sans(17.5, FontWeight.w400, color: c.fgSecondary),
    bodySmall: sans(12, FontWeight.w400, color: c.fgMuted, height: 1.4),
    labelLarge: sans(14, FontWeight.w600, color: c.btnFg, height: 1.2),
```

**Fix:** The spec must fix the Cobalt-Neo type scale (keep Archivo, but define body/title/display sizes and whether 17.5 survives) and add spacing tokens (unit 4, controlGap 12, cardPadding 16/24, sectionBreak 32) to `Cobalt`; then normalize FieldCard/button/chip/dock paddings onto them.

### dark-shadow-color-undefined — Low

The only dark-mode shadow in the app is a hand-tuned 18%-alpha lightInk blur on the dock, and the token class has no shadowColor slot. Hard `4px 4px 0 0` shadows in pure black crush detail on a near-black dark ground, and without a defined color each implementer will pick a different one (black vs `#0B0D12` vs cobalt-950), so the app will subtly disagree with itself across surfaces.

**Evidence:** `client/lib/chrome/product_dock.dart:31-33`

```dart
          elevation: 8,
          shadowColor: Cobalt.lightInk.withValues(alpha: 0.18),
          borderRadius: BorderRadius.circular(Cobalt.radiusPill),
```

**Fix:** Define per-theme shadow colors in the spec: e.g. light = ink at ~15% alpha twin of solid, dark = solid near-black (or a slightly-lightened ink so the cast is visible on `#000`–`#222`); encode as `Cobalt.shadowCast`/`shadowCastSm` colors used by all shadow tokens.

### design-md-contract-stale — Low

`DESIGN.md` still prescribes the rounded lock (rounded.control 9999px, card 1.25rem, hero 1.75rem, hairline cobalt-200 cards) and even disagrees with code (doc says control radius 9999px; `cobalt.dart` `radiusControl` is 12.0), so the documented contract will actively fight the migration. Anyone migrating a screen against `DESIGN.md` will re-implement pills and hairlines, and the design-system CHANGELOG (`docs/design-system/CHANGELOG.md`) requires a major 2.0.0 bump "when the token contract breaks" — which this migration does — so without that, the guardrails and review checklists keep pointing at the old system.

**Evidence:** `DESIGN.md:27-45`

```
rounded:
  control: "9999px"
  card: "1.25rem"
  hero: "1.75rem"
```

**Fix:** Rewrite `DESIGN.md` to the Cobalt-Neo contract (0px radius, 2px borders, cast shadows, press states, gray ramp, dark-first) in the same migration PR that lands the new tokens, and add the 2.0.0 entry naming every touched file per the CHANGELOG rules.

## Recommendations

- Write the Cobalt-Neo token spec before touching widgets: new gray ramps (dark 4-step + light 4-step), borderWidth tokens, cast/pressed shadow tokens with per-theme shadow colors, press-motion tokens, semantic-hue policy, dark-first default, and the type/spacing scale decision.
- Land tokens as Cobalt 2.0.0 in `client/lib/theme/cobalt.dart` + `cobalt_theme.dart` with the design-system CHANGELOG major bump and `DESIGN.md` rewrite in the same commit set.
- Migrate theme-level widgets first (FieldCard, ShadeChip, AuthDoor, ProductDock, button/input themes) so features inherit the neo grammar for free, then sweep feature hardcodes (radii 99/12/10/4/3/1, StadiumBorder capture bar, segmented control).
- Build one shared pressable (hover shrink+translate, active press, 100ms, reduce-motion safe) and convert every InkWell/tappable card to it in a single pass.
- Flip the default to dark-first only after the dark gray ramp lands, and re-shoot the six lock screens in `docs/design-system/cobalt/` as the new acceptance baseline.
- Convert the coach panel onto an explicit inverse-panel token and delete the surfaceBase-identity brightness hack.

## What this role would do first

Draft the Cobalt-Neo token table — dark and light gray ramps, border widths (2px container / 1px divider), cast and pressed shadows with an explicit dark-mode shadow color, press-motion tokens (100ms, offsets, curve), the semantic-hue policy, the type/spacing scale decision, and the dark-first default — as a dated, reviewed plan in `docs/plans/`. Every code change above is blocked on those values being decided; once reviewed, land the table as `Cobalt` 2.0.0 in `client/lib/theme/cobalt.dart` and `client/lib/theme/cobalt_theme.dart`, converting the theme-level widgets (`field_card.dart`, `cobalt_chip.dart`, `auth_door.dart`, `product_dock.dart`, button/input themes) before any feature screen is touched.
