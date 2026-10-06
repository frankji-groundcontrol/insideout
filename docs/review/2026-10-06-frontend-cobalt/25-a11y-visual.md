# 25 · A11y Visual

**Lens:** Contrast ratios, non-color channels, semantics, touch targets
**Score:** **4.5 / 10**
**Bottom line:** The Cobalt token system is fundamentally sound for body text — every foreground/background pair I computed passes WCAG AA in both light and dark (4.55:1 to 18.29:1) — but the chrome around it fails badly: the dock's "unfinished step" state is 1.32:1 blue-on-blue in light theme, the floating top-bar icons are 1.42:1 over the navy coach panel on the PRD page, every stroke/border token sits at 1.40–2.26:1 against WCAG 1.4.11's 3:1, and the roadmap minimap encodes status as unlabeled 4px color marks at 1.40–2.90:1. There is no focus strategy anywhere in client/lib (zero focusColor/FocusNode usages), text ships at 9–11px in three places, and the known brightness-sniff in coach_panel.dart is both fragile (color identity) and currently a no-op ternary whose two branches resolve to identical colors. A strong type/contrast foundation undermined by systemic non-text contrast failures on primary navigation.

Reviewed branch `main` at `f185e94`; a skeptic pass re-checked every Critical/High finding — corrections marked [Correction]. Screenshot evidence lives in `./shots/`.

---

## Scope

The theme system and its application across the product: `client/lib/theme/cobalt.dart` (palette tokens) and `cobalt_theme.dart` (ThemeData wiring, type roles) read in full; every other widget file cross-referenced through the pre-built code map; all 20 in-app screenshots in `./shots/` inspected at native resolution; and WCAG 2.x contrast ratios computed with a python3 script over every meaningful foreground/background pair in both themes. Also grepped `client/lib` for `fontSize:`, `Focus`/`focusColor`/`focusNode`/`autofocus`, `Color(0x` literals, `Semantics`, and `textScaler` overrides. The two lead-confirmed defects (auth-modal header collision, dock overlaying body content) are cited only where they intersect accessibility.

## What is good

- All body-text token pairs pass WCAG AA in both themes (computed): `fgPrimary` 18.29:1 both themes, `fgSecondary` 9.93:1 light / 11.81:1 dark, `fgMuted` 4.55:1 light / 6.28:1 dark, button text on brand 5.09:1 light / 4.79:1 dark — `client/lib/theme/cobalt.dart:102-154`.
- The focused input border is explicitly styled (hue500, 1.5px) — the one place focus got real design attention — `client/lib/theme/cobalt_theme.dart:100-103`.
- Tooltips on every chrome icon: `client/lib/app.dart:101-124`, `client/lib/chrome/product_dock.dart:156`, `client/lib/features/prd/coach_panel.dart:117,157`, `client/lib/features/prd/prd_desk.dart:153`, `client/lib/features/project/roadmap_node_card.dart:219`.
- The motion layer honors `MediaQuery.disableAnimations` (reduced-motion support) — `client/lib/theme/ink_motion.dart:6`.
- No `textScaler`/`textScaleFactor` overrides anywhere, so OS text scaling flows through; the CJK font fallback chain prevents zh-CN tofu — `client/lib/theme/cobalt.dart:88-93`.
- Token discipline is otherwise excellent: only 6 hardcoded `Color(0x...)` literals outside `theme/` across ~10k lines (grep-verified), most of them the locked brand hue.

## Findings, worst first

### dock-unfinished-blue-on-blue — Critical

The dock's "unfinished step" cue paints the icon, the 11px label, and a 1.5px border in the warn token directly on the brand-colored pill — 1.32:1 in light theme (blue on cobalt from the same ramp), 3.41:1 in dark. This is the product's only "you skipped the human commit step" signal, sitting on its primary navigation, and it is effectively invisible in light theme.

Evidence: `client/lib/chrome/product_dock.dart:166`

```
border: unfinished ? Border.all(color: palette.statusWarnFg, width: 1.5) : null,
...
Icon(_icon(item), size: 20, color: unfinished ? palette.statusWarnFg : palette.btnFg),
// cobalt.dart light: statusWarnFg: hue700 (#1D4ED8) drawn on palette.brand = hue500 (#3B5BFF)
// computed: light #1D4ED8/#3B5BFF = 1.32:1; dark #C7D2FE/#3B5BFF = 3.41:1 — shot 10-roadmap-dark.png shows the periwinkle '未完成；提交' label+icon on the cobalt pill
```

**Fix:** Introduce an on-brand warn token (e.g. #FFD166 amber: ~7:1 on hue500) or drop the warn color on the pill entirely and mark unfinished steps with a white dot + tooltip; recompute with the same WCAG script until ≥4.5:1.

**Verdict:** CONFIRMED

### topbar-icons-invisible-over-navy — Critical

The floating utility icons (language/theme/profile/logout) are positioned in a bare `Positioned` row with no backdrop; wherever a page's dark content reaches the top strip — e.g. the coach panel, which fills the right column full-bleed with navy hue950 even in light theme — the icons render dark-on-navy at ~1.3–1.42:1. In `15-prd-desk-light.png` all four are effectively gone, so theme toggle, profile, and logout are unreachable for light-theme users on `/prd/:id`.

Evidence: `client/lib/app.dart:90`

```
Positioned(
  top: 0,
  right: 0,
  child: SafeArea(
    child: Padding(
      padding: const EdgeInsets.fromLTRB(8, 8, 12, 8),
      child: Row( // IconButtons at app.dart:100-125, no background
// coach_panel.dart:112: final field = palette.brightnessIsDark ? palette.surfaceSunken : Cobalt.hue950;
// lightInk #0B0D12 on hue950 #1E2A5A = 1.42:1 — shot 15-prd-desk-light.png ≈(1287-1410,28): four icons nearly invisible on the navy panel
```

**Fix:** Give the utilities Row a surfaceRaised/95% background pill (or hoist a real AppBar with backgroundColor) so icon color always pairs with a known surface; add a golden screenshot asserting ≥3:1 for those icons.

**Verdict:** CONFIRMED

### stroke-tokens-fail-non-text-contrast — High

All four stroke tokens are 1.40–2.26:1 against their surfaces, and they outline everything: input borders, card borders, outlined buttons, chip borders, and dividers. Form field boundaries are 1.49:1 on white (the login modal's fields in `02-login-modal.png` are nearly outline-less), so low-vision users will read the light UI as borderless. Every one of these fails WCAG 1.4.11's 3:1 non-text minimum.

Evidence: `client/lib/theme/cobalt.dart:109`

```
strokeSubtle: hue200,   // #C7D2FE: 1.40:1 on surfaceBase, 1.49:1 on surfaceRaised (light)
strokeStrong: Color(0xFF9AA7E8),  // 2.18:1 (light); dark strokeSubtle #2A3358 = 1.59:1, strokeStrong #3B4A7A = 2.26:1
// used for input borders cobalt_theme.dart:94,98, cards :58, outlined buttons :74, chips :110, dividerColor :112
// shot 02-login-modal.png: 邮箱/密码 field outlines are barely perceptible on the white card
```

**Fix:** Retune stroke tokens to ≥3:1 (light strokeStrong ≈ #6E7FD6, light strokeSubtle ≈ #94A3E8; dark strokeSubtle ≈ #4A5680); the WCAG python script already in this review becomes the regression check.

**Verdict:** CONFIRMED

### minimap-color-only-invisible-marks — High

The roadmap minimap encodes node status as unlabeled 4px color bars drawn from outline tokens — locked 1.40:1, pending 2.18:1 (light), done 2.90:1 (dark). State is conveyed by color alone, the colors fail the 3:1 graphics floor, and on a project with locked/pending nodes the minimap shows nothing at all against the page ground. Colorblind users can't distinguish done vs in-progress even where marks are visible.

Evidence: `client/lib/features/project/roadmap_canvas.dart:172`

```
Color mark(String status) {
  switch (status) {
    case 'done': return theme.colorScheme.tertiary;      // hue700: 2.90:1 dark
    case 'in_progress': return theme.colorScheme.secondary; // hue500: 3.82:1 dark
    case 'locked': return theme.colorScheme.outline;       // strokeSubtle: 1.40:1 light / 1.59:1 dark
    default: return theme.colorScheme.outlineVariant;      // strokeStrong: 2.18:1 light / 2.26:1 dark
  }
} // bars are height 4, no label, no tooltip (roadmap_canvas.dart:211-218)
```

**Fix:** Raise mark colors to ≥3:1 vs surface and add a non-color channel: filled bar = done, hollow (bordered) = pending, hatched/dashed = locked, or an aria-style Semantics label + tooltip per column.

**Verdict:** CONFIRMED

### admin-hardcoded-red-dark-unreadable — High

Error text in the admin section uses a literal `Color(0xFFB3261E)` instead of the theme's `fgDanger`, giving 2.72:1 in dark mode against `surfaceRaised` — where the existing token would give 13.03:1. Admin-facing failure messages (invite/revoke errors) are unreadable in dark theme exactly when the operator needs them most.

Evidence: `client/lib/features/profile/admin_section.dart:167`

```
child: Text(localizedError(error!, lang.t), style: const TextStyle(color: Color(0xFFB3261E))))
// computed: #B3261E / #FFFFFF = 6.54:1 (light OK); #B3261E / #141820 = 2.72:1 (dark FAIL, needs 4.5:1)
// theme already ships fgDanger: hue700 (light) / hue200 (dark) at cobalt.dart:116,143
```

**Fix:** Replace the literal with `Theme.of(context).colorScheme.error` (or `palette.fgDanger`) and add a lint banning `Color(0x` outside `theme/`.

**Verdict:** CONFIRMED

### no-focus-indicators — Medium

There is no focus strategy anywhere in `client/lib`: grep finds zero `focusColor`, zero `FocusNode`, and no focus-highlight customization — only `Focus(autofocus: true)` for Esc handling in the auth modal and one autofocus on the login field. Keyboard focus therefore relies on Material defaults, which on the cobalt dock compute to ~1.3:1, and the dock's selected state is a 1.31:1 white alpha wash. Neither "where is focus" nor "which page am I on" is perceivable keyboard-only.

Evidence: `client/lib/theme/cobalt_theme.dart:37`

```
grep 'FocusNode|focusColor|Focus\b|focusNode|autofocus' over client/lib → only matches: auth_door.dart:30-31 Focus(autofocus: true) (Esc handling), auth_form.dart:159 autofocus: true, plus LoopFocus (unrelated nav state)
// product_dock.dart:165: color: selected ? palette.btnFg.withValues(alpha: 0.16) : Colors.transparent  // computed selected-wash vs brand = 1.31:1
```

**Fix:** Set `ThemeData.focusColor`/focus highlights with a 2px offset ring at ≥3:1, and redesign the dock selected state (solid white icon+text on cobalt, or a white outline) to ≥3:1.

### dark-status-pills-collapse — Medium

In dark mode `fgDanger`, `statusInfoFg`, `statusWarnFg`, and `statusSuccessFg` are all the same hue200 periwinkle on identical hue950 backgrounds — danger text is visually indistinguishable from informational accents (they are literally the same hex), so error semantics are lost. The light warn pill also hairline-fails at 4.49:1, one hundredth under AA for its 12px semibold label.

Evidence: `client/lib/theme/cobalt.dart:143`

```
fgDanger: hue200,
...
statusInfoBg: hue950, statusInfoFg: hue200,
statusWarnBg: hue950,  statusWarnFg: hue200,
statusSuccessBg: hue950, statusSuccessFg: hue200,  // dark: all three pills = #C7D2FE on #1E2A5A
// light warn pill computed: statusWarnFg hue700 / statusWarnBg hue200 = 4.49:1 (fails 4.5 by 0.01 at 12px semibold)
```

**Fix:** Give dark `fgDanger` a true red ramp (e.g. #F2B8B5 / #FF8A80, both >4.5 on the dark surfaces) and differentiate warn with an amber; nudge light warn bg or fg to clear 4.5:1.

### brightness-sniff-noop-ternary — Medium

Dark mode is detected by palette color identity (`surfaceBase == Cobalt.darkGround`), and the ternary it feeds currently resolves both branches to identical colors (`#F7F8FA` / `#1E2A5A` either way) — so the check is both fragile and a no-op. It works today by accident and will silently break the first time someone retunes the dark palette (e.g. to fix dark-status-pills-collapse), flipping the light-theme coach panel to mixed colors with no test to catch it.

Evidence: `client/lib/features/prd/coach_panel.dart:230`

```
extension on Cobalt {
  bool get brightnessIsDark => surfaceBase == Cobalt.darkGround;
}
// coach_panel.dart:111-112:
final ink = palette.brightnessIsDark ? palette.fgPrimary : Cobalt.darkType;   // #F7F8FA both ways
final field = palette.brightnessIsDark ? palette.surfaceSunken : Cobalt.hue950; // #1E2A5A both ways
// Cobalt.of (cobalt.dart:100) already exposes Theme.of(context).brightness
```

**Fix:** Delete the extension; use `Theme.of(context).brightness == Brightness.dark` (or pass brightness into the palette lookup) and replace the ternaries with a single `field`/`ink` constant pair; add a widget test asserting coach panel colors identical pre/post palette tweak.

### sub-12px-text — Medium

Production UI ships 9–11px text in three places: 10px remote-cursor names, 9px minimap band labels clipped to six characters, and 11px labels on every primary-nav dock button. At 1440px this is hard to read for many users and becomes unreadable under browser text zoom steps; the dock — the app's main navigation — has no larger-than-11px label anywhere.

Evidence: `client/lib/features/project/roadmap_canvas.dart:100`

```
roadmap_canvas.dart:100: Text(c.name, style: Theme.of(context).textTheme.bodySmall?.copyWith(fontSize: 10))
roadmap_canvas.dart:226: style: theme.textTheme.bodySmall?.copyWith(fontSize: 9)  // label clipped: _clip() -> s.substring(0, 6)
chrome/product_dock.dart:178: fontSize: 11,  // the only text label on each dock button
```

**Fix:** Set 12px as a hard floor (grep lint on `fontSize:`), raise dock labels to 12–13px, and give minimap bands a tooltip with the full title instead of 9px truncated text.

### brand-on-dark-accents-fail — Low

The locked brand hue500 is used directly on dark surfaces where it reaches only 3.82:1 as large display text (the ideas-wall title) and 2.69:1 as graphics (remote-cursor icons, coach accent bar) — below the 3:1 graphics bar for the presence cursors, which are the only indicator of another live user. Dark-theme users see a dimmer, muddier brand than light-theme users: a parity break.

Evidence: `client/lib/features/workspace/ideas_page.dart:191`

```
ideas_page.dart:188-192: style: theme.textTheme.displayLarge?.copyWith(fontSize: display, height: 0.9, color: palette.brand, // hue500 on darkGround = 3.82:1 (shot 08-ideas-wall-dark.png — title noticeably dimmer than in light)
roadmap_canvas.dart:99: Icon(Icons.navigation, size: 14, color: Theme.of(context).colorScheme.secondary) // hue500 on hue950 = 2.69:1 — non-text FAIL
coach_panel.dart:152: Container(width: 36, height: 3, ..., color: Cobalt.hue500) // 2.69:1 on the navy panel
```

**Fix:** Add a `brandOnDark` token (hue200) chosen via `Cobalt.of` for dark grounds; use it for the cursor icon and any brand graphic on dark.

### coach-alpha-text-below-aa — Low

Coach panel hint text at 50% ink over the navy field computes 4.37:1 — a hairline fail for its 17.5px body size (the 60%/70% blends elsewhere pass at 5.64/7.08). The placeholder prompting the user how to reply to the AI coach is the single faintest text in the product and just under AA.

Evidence: `client/lib/features/prd/coach_panel.dart:215`

```
hintStyle: TextStyle(color: ink.withValues(alpha: 0.5)),  // coach_panel.dart:215; also :143 alpha .7 (OK) and _Quote role :254 alpha .6 (OK)
// computed: #F7F8FA @50% over #1E2A5A → #8A91AA, 4.37:1 vs 4.5 required
```

**Fix:** Bump the hint alpha to 0.55+ (≈4.75:1) or introduce a dedicated `mutedOnNavy` token so alpha math doesn't get re-derived ad hoc.

### light-warn-pill-borderline — Low

The light-theme warn/danger pill (hue700 text on hue200 background) computes 4.49:1 — 0.01 under the 4.5:1 AA bar for its 12px semibold label. It sits within measurement noise of failing: a font-weight or anti-aliasing change could tip it over. The dock's unfinished border uses `statusWarnFg` too, so this token matters beyond the pill.

Evidence: `client/lib/theme/cobalt_chip.dart:33`

```
ShadeTone.danger => (palette.statusWarnBg, palette.fgDanger),  // cobalt_chip.dart:33
// cobalt.dart:123-124 light: statusWarnBg: hue200 (#C7D2FE), statusWarnFg: hue700 (#1D4ED8) → 4.49:1; label is labelSmall 12px semibold = normal text threshold
```

**Fix:** Darken the light warn fg one step (e.g. #1E40AF) or lighten the bg (hue50), re-run the ratio script; while there, add the planned shape differentiation from the minimap finding to pills.

## Recommendations

- Fix the tokens first (`strokeSubtle`/`strokeStrong`, dark `fgDanger`, an on-brand warn color, `brandOnDark`) in `cobalt.dart` — findings 1, 3, 4, 7, 10, 12 are all token-level and inherit everywhere at once.
- Land a Dart contrast test that recomputes WCAG ratios from `Cobalt.of` values for both themes, so palette tweaks fail CI instead of shipping silent 1.3:1 chrome (the python3 script from this review ports in ~30 lines).
- Add an a11y lint pass: no `Color(0x` outside `theme/`, no `fontSize < 12`, no text colors applied via `withValues(alpha:)` without a ratio assertion, no palette-color identity comparisons.
- Design and implement a focus system: `focusColor` with a visible ring, a ≥3:1 dock selected state, and a golden screenshot with an injected focused element.
- Add `Semantics` (status labels on minimap columns, selected state on dock) — only one `Semantics` widget exists in the whole app (`assembly_diagram.dart:80`).

## What this role would do first

Retune the stroke tokens and the dock's warn/selected colors in `client/lib/theme/cobalt.dart` + `client/lib/chrome/product_dock.dart`, verifying with the WCAG ratio script — this single change clears the widest-reaching failures (input/card borders system-wide, the 1.32:1 unfinished cue, and the 1.31:1 selected wash) before any widget-level work.
