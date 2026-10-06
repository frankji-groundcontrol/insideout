## Accessibility & Visual Review — Cobalt Frontend

I reviewed the theme system and its application across the product: `client/lib/theme/cobalt.dart` (palette tokens) and `cobalt_theme.dart` (ThemeData wiring, type roles) read in full; every other widget file cross-referenced through the pre-built code map; all 20 in-app screenshots in `docs/review/2026-10-06-frontend-cobalt/shots/` inspected at native resolution; and WCAG 2.x contrast ratios computed with a python3 script over every meaningful foreground/background pair in both themes. I also grepped `client/lib` for `fontSize:`, `Focus`/`focusColor`/`focusNode`/`autofocus`, `Color(0x` literals, `Semantics`, and `textScaler` overrides. Findings below are mine alone; the two lead-confirmed defects (auth-modal header collision, dock overlaying body content) are cited only where they intersect accessibility.

### What is genuinely good

The type-and-color foundation is better than most design systems I review. Every body-text token pair passes WCAG AA in both themes, computed: fgPrimary/surface 18.29:1 (both themes), fgSecondary 9.93:1 light / 11.81:1 dark, fgMuted 4.55:1 light / 6.28:1 dark, button text on brand 5.09:1 light / 4.79:1 dark, dark fgBrand 13.03:1. Tooltips are on every chrome icon button. The motion layer honors `MediaQuery.disableAnimations` (`ink_motion.dart:6`). Nothing overrides text scaling. CJK fallback chains are declared so zh-CN never tofu. Only six hardcoded color literals exist outside `theme/`, most of them the locked brand hue. And the focused input border is explicitly styled (`cobalt_theme.dart:100-103`) — the one place keyboard-adjacent states got real design attention.

### Critical

**Dock "unfinished" cue is blue-on-cobalt (1.32:1 light / 3.41:1 dark).** `product_dock.dart:166,172,177` paints the icon, the 11px label, and a 1.5px border in `statusWarnFg` on the brand-colored pill. In light theme both are blues from the same ramp — `#1D4ED8` on `#3B5BFF` — and the result is invisible in `10-roadmap-dark.png` and would be worse in light. This is the product's only "you skipped the human commit step" signal, on its primary navigation. It needs an on-brand warn token (amber clears 7:1 on hue500) or a shape/position treatment instead of color.

**Top-bar icons float with no backdrop (1.42:1 on the PRD page).** `app.dart:90-130` positions the language/theme/profile/logout `IconButton`s in a bare `Positioned` row; the coach panel fills the right column full-bleed with navy `hue950` even in light theme (`coach_panel.dart:112`), so in light mode those icons render `#0B0D12` on `#1E2A5A`. Shot `15-prd-desk-light.png` shows all four effectively gone — theme toggle, profile, and logout unreachable for light-theme users on `/prd/:id`. Give the strip a surface background pill or a real AppBar.

### High

**Every stroke token fails WCAG 1.4.11 (1.40–2.26:1).** `strokeSubtle`/`strokeStrong` (`cobalt.dart:109-110,136-137`) are used for input borders, card borders, outlined buttons, chip borders, and dividers — so form boundaries are 1.49:1 on white (see the login modal's near-outline-less fields in `02-login-modal.png`) and card edges are 1.40–2.26:1 everywhere. Low-vision users will read the light UI as borderless. Retune to ≥3:1.

**Roadmap minimap: color-only, mostly invisible status.** `roadmap_canvas.dart:172-183` draws unlabeled 4px bars per node status using outline tokens — locked 1.40:1, pending 2.18:1 (light), done 2.90:1 (dark). The control encodes state purely by color, and the colors fail the 3:1 graphics floor; on a project with locked/pending nodes the minimap shows nothing at all. Raise contrast and add a non-color channel (filled/hollow/hatched) plus tooltips.

**Admin error red bypasses the theme (2.72:1 dark).** `admin_section.dart:167` hardcodes `Color(0xFFB3261E)` where the theme already ships a dark-safe `fgDanger` (13.03:1). The literal is unreadable on dark `surfaceRaised`.

### Medium

**No focus strategy exists.** Grep finds zero `focusColor`, zero `FocusNode`, no focus styling anywhere; only `Focus(autofocus: true)` for Esc handling in the auth modal and one autofocus on the login field. Material's default ~12% primary-tint focus overlay on the cobalt dock computes to ~1.3:1, and the dock's selected state is a 1.31:1 white alpha wash (`product_dock.dart:165`) — so neither focus location nor current route is perceivable keyboard-only.

**Dark-theme status pills collapse into one color.** `cobalt.dart:142-153` sets dark `fgDanger`, `statusInfoFg`, `statusWarnFg`, `statusSuccessFg` all to hue200 on identical hue950 backgrounds — danger text is hex-identical to informational accents. Light warn pill is 4.49:1, one-hundredth under AA at its 12px semibold size.

**The brightness sniff is fragile and a no-op.** `coach_panel.dart:230-232` defines `brightnessIsDark => surfaceBase == Cobalt.darkGround` — a palette identity check — and the ternary it feeds resolves both branches to identical colors (`#F7F8FA`/`#1E2A5A` either way). It works today by accident and will silently break the first time someone retunes the dark palette. `Theme.of(context).brightness` is right there in `Cobalt.of`.

**Text ships at 9–11px.** Remote-cursor names at 10px (`roadmap_canvas.dart:100`), minimap labels at 9px clipped to six characters (`:226`), and every dock button's only label at 11px (`product_dock.dart:178`).

### Low

Brand hue500 on dark surfaces reaches only 3.82:1 as large display text (the ideas-wall title, `ideas_page.dart:191`) and 2.69:1 as graphics (presence-cursor icons, coach accent bar) — a dark-theme parity break. The coach input hint at 50% alpha computes 4.37:1. Both are near-misses with cheap token fixes.

### What I would do first

Fix the tokens before any widget work: retune `strokeSubtle/strokeStrong`, add a true dark danger red, an on-brand warn color, and a `brandOnDark`, all verified by a checked-in Dart contrast test that recomputes these ratios in CI. That single file (`cobalt.dart`) plus `product_dock.dart`'s warn/selected colors clears the widest-reaching failures — borders, the unfinished cue, the selected wash — everywhere at once, and converts this system's genuinely strong text contrast into chrome that low-vision and keyboard users can actually operate.
