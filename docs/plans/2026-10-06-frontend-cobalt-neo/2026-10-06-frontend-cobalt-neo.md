# Frontend Cobalt-Neo — remediation implementation plan

- **Date:** 2026-10-06 · **Branch:** `main` @ `f185e94`
- **Source:** [2026-10-06 frontend cobalt review](../review/2026-10-06-frontend-cobalt/README.md)
  (114 findings, 7 Critical / 22 High; 28 confirmed, 1 downgraded, 0 refuted).
- **Contract:** execute the review's
  [40-remediation-plan](../review/2026-10-06-frontend-cobalt/40-remediation-plan.md)
  waves 1–7 in order; each wave's runnable check gates the next.

**Design target — Cobalt-Neo:** 0px corner radius, 2px solid borders, 4px4px
zero-blur cast shadows, 100ms press-the-block motion, dark-first default,
neutral dark ramp. The design language follows the user-supplied reference
app; the hue family stays the current blue (cobalt).

## Status

**Finished** 2026-10-06 — waves 1–7 are on `main` as `28e1fb1`.
The local gate in `client/` passed: `flutter analyze --no-fatal-infos`
exit 0, no errors or warnings, 32 infos (`32 issues found. (ran in
0.9s)`); `flutter test` `00:09 +143: All tests passed!` (`TEST_EXIT:0`);
`dart format --output=none --set-exit-if-changed lib test` exit 0.

Waves 1–3 were already complete: Wave 1 gates green (94/94); Wave 2
re-shoot verified A1/A2/A9/A10 fixed and A5 resolved by the 3.5 dock
rewrite (`shots-wave2/` + `shots-wave3/`); Wave 3 WCAG script green
(107/107, analyze clean) and decree visual pass six-for-six
(`shots-wave3/`, both themes + narrow; DESIGN.md refreshed = 3.7).

Waves 4–7 landed in this pass. Wave 5.2 was not a new WCAG run: theme
files were not edited, so the Wave 3 107/107 result still covers the
retuned tokens. The web bundle was not rebuilt, so the font-asset
shrink is not measured. GitHub CI was not watched; the format step is
in `.github/workflows/ci.yml`.

Closed with Wave 4's dock pass: workspace card dates already reserve
space for the dock and the create control, and the 390px dock pill is
capped (the unfinished label is icon-only, and ellipsized when the
text is shown). Still deferred: W3-N2 minified console errors
([issue](../../issues/2026-10-06-wave3-minified-console.md)).
`project_page.dart` is still over the ~350-line budget (640); the
decide edit shortened it and did not split it.

## Cobalt-Neo token table (Wave 3 gate deliverable)

Contrast ratios below are computed with the WCAG relative-luminance formula.
The Wave 3/5 check re-verifies every pair with the WCAG script against the
real surfaces; where a value falls short, take the lightest/darkest adjacent
step that passes and record the final value in DESIGN.md (Wave 3.7).

### Radius — `client/lib/theme/cobalt.dart:72-75`

| Token | Old | New | Note |
|---|---|---|---|
| `radiusControl` | 12.0 | **0.0** | names kept (kills the sweep); semantics now "decree radius" |
| `radiusCard` | 20.0 | **0.0** | |
| `radiusHero` | 28.0 | **0.0** | |
| `radiusPill` | 999.0 | **0.0** | status dots become square blocks; boot loader follows the same block language |

### Border widths — new tokens in `cobalt.dart`

| Token | Value | Used by |
|---|---|---|
| `borderContainer` | **2.0** | FieldCard, cardTheme, ShadeChip, AuthDoor, ProductDock, dialogs |
| `borderDivider` | **1.0** | hairline separators, table rows, input underlines |

### Cast shadows — new tokens in `cobalt_theme.dart:54-61`

| Token | Geometry | Dark color | Light color |
|---|---|---|---|
| `cast` | offset (4,4), blur 0, spread 0 | solid ink `#05070A` | black @ ~15% (`#26000000`) |
| `castSm` | offset (2,2), blur 0, spread 0 | same | same |

Wired into FieldCard, cardTheme, AuthDoor, ProductDock, dialogs, toasts.

### Motion — `client/lib/widgets/pressable_block.dart`

| Token | Value |
|---|---|
| `pressDuration` | **100ms** |
| rest | cast (4,4) |
| hover | translate (2,2), shadow shrinks to castSm (2,2) |
| active | translate (4,4), shadow gone (block pressed into the page) |

Shared InkWell/ButtonStyle extension so every interactive block gets it.

### Strokes (Wave 1.8 + 3.6 final values)

The 40-plan's first-pass candidates `#94A3E8` (2.4:1 on white) and `#4A5680`
(2.7:1 on dark ground) were computed during planning and **fail 3:1** — the
values below are the corrected set this plan signs off:

| Token | Theme | Value | Computed contrast |
|---|---|---|---|
| `strokeSubtle` | light | **`#7B8CE0`** | 3.15:1 on white / lightGround |
| `strokeStrong` | light | **`#5A6FE0`** | 4.39:1 on white / lightGround |
| `strokeSubtle` | dark | **`#646A78`** (neutral) | 3.27:1 on sunken `#16181E`, 3.58:1 on ground `#0B0D12` |
| `strokeStrong` | dark | **`#788098`** (neutral) | 4.47:1 on sunken, 4.89:1 on ground |

Wave 1.8 applies the final values directly (no interim blue-only step); the
neutral dark strokes land together with the Wave 3.6 ramp below.

### Neutral dark ramp (Wave 3.6)

| Token | Old | New | Note |
|---|---|---|---|
| `hue950` `#1E2A5A` | surfaceSunken + brandSubtle | **brandSubtle only** | blue demoted out of surfaces |
| `surfaceSunken` (dark) | `hue950` | **`#16181E`** (candidate) | neutral, slightly above ground |
| `surfaceRaised` (dark) | `#141820` | neutralize (~`#14171D`) | candidate; script verifies |
| `statusNeutralBg` (dark) | `#1A1F2E` | keep | already the target value |

### Status & accent

| Item | Value | Check |
|---|---|---|
| dock warn dot | amber **`#FFD166`** (square block, not a circle) | script: ≥3:1 on dock brand background (computed ≈7:1) |
| chip text (light `statusNeutralFg`) | **`#3A3F4A` → `hue700 #1D4ED8`** | 5.99:1 on `hue50` chip bg — passes ≥4.5:1 text |

**chip-shade authority adjudication** (critic-design-md-chipshade-textcolor):
DESIGN.md:41 is the authority — chip text is `cobalt-700`; `cobalt.dart`
aligns to it. DESIGN.md wins because it is the design contract.

### Theme default (Wave 1.5)

Dark-first: `appearance.dart` defaults to dark (explicit user choice still
wins); DESIGN.md records the new default.

## Wave checklist

Details (register IDs, file:line, diffs) live in the
[40-remediation-plan](../review/2026-10-06-frontend-cobalt/40-remediation-plan.md);
this checklist is the live tracker.

- [x] **Wave 1** — one-line & token-value fixes (1.1 dock Align · 1.2 viewport
  meta · 1.3 idea status l10n · 1.4 admin error color · 1.5 dark-first ·
  1.6 hydrate 401-only clear · 1.7 Dio timeouts · 1.8 stroke values above)
  → check: `flutter analyze && flutter test` **green 2026-10-06 (94/94)**;
  dock-at-bottom @1440×900 visual folded into the Wave 2 re-shoot
- [x] **Wave 2** — dock geometry, clearance, header collisions (2.1–2.6)
  → check: re-shoot all pages @1440×900 & 390×844 **done 2026-10-06 —
  A1/A2/A9/A10 verified fixed** (`shots-wave2/`, 25 shots); **A5 fixed** by
  the 3.5 rewrite (amber square + label swap, `shots-wave3/14/16`)
- [x] **Wave 3** — token system + DESIGN.md (3.1 radius · 3.2 cast shadows ·
  3.3 border widths · 3.4 press motion · 3.5 warn amber · 3.6 neutral ramp ·
  3.7 DESIGN.md refresh) — **gate lifted by this file's sign-off**
  → check: WCAG script over all pairs **green 2026-10-06 (107/107, analyze
  clean; deviations recorded in DESIGN.md: amber measured 3.53:1, cast
  shadows WCAG-exempt directionally, ButtonStyle choreography →
  `PressableBlock`)**; decree visual pass **six-for-six**
  (`shots-wave3/`, 29 shots, both themes + narrow)
- [x] **Wave 4** — write reliability & stream timeouts (4.1–4.4)
  → check: guarded writes and a 75s coach idle timeout **done 2026-10-06**.
  Proposal accept/reject and the version commit use `guardWrite` and
  the localized `write.failed` snackbar (idea, workspace, and the other
  project writes already did). A failed capture or convert keeps the
  card and the field. A failed version commit does not land on the
  stack. `readCoachSse` is 75s. An explicit `Timer` resets on every
  complete frame, including heartbeats (`Stream.timeout` did not
  elapse under the widget-test clock). The coach tests stall through
  that reader. Evidence scroll-highlight, version → `/prd/<id>/versions`
  (or disabled), and cold tiles that do not open `/dashboard` were
  already true. No live kill-network pass; the gate is the widget tests.
- [x] **Wave 5** — non-color a11y channels (5.1 minimap shape channel · 5.2
  WCAG re-run) → check: grayscale-distinguishable minimap **done
  2026-10-06**. States already differed without color (filled, half,
  hollow, dashed) plus semantics labels; marks are `BorderRadius.zero`.
  The widget test fingerprints paint without color and fails if done,
  pending, in-progress, and locked collapse to one signature. Theme
  files were not edited. The closing `flutter test` still re-ran
  `cobalt_test` and it passed, which is the 5.2 WCAG script.
- [x] **Wave 6** — test harness, goldens, format gate (6.1–6.4)
  → check: overlap and geometry tests, CI format step **done 2026-10-06**.
  6.1 mounts `AppScaffold` (`extendBody`) at 1400×900 and asserts the
  dock rect misses the body landmark; the 390px pill stays under 360
  wide. 6.2 is a geometric assertion, not a golden: at 1440×900, light
  and dark, zh-CN, the wordmark sits inside the card (the remediation
  plan allows that in place of a golden). 6.3 adds
  `dart format --output=none --set-exit-if-changed lib test` to CI.
  6.4 docs say Flutter 3.44.x; CI and the Dockerfile already pin 3.44.0.
  GitHub CI was not watched, and deleting the 1.1 fix was not used as
  a golden proof.
- [x] **Wave 7** — bundle & URL hygiene (7.1 CJK fonts out of web bundle ·
  7.2 consume URL secrets) → check: web asset list and history rewrite
  **done 2026-10-06**. `OFL.txt` stays an unconditional asset; both
  variable TTFs are limited to android, ios, linux, macos, and windows
  (native `rootBundle` still loads them; a web build omits them). The
  TTF files stay on disk and the Google Fonts link is unchanged. The
  bundle was not rebuilt, so the byte shrink is not measured. Invite,
  reset, and magic-link tokens are copied into state, then web-only
  `history.replaceState` drops those query keys (`urlWithoutQueryKeys`;
  no `GoRouter.replace`).

## Verification (whole effort)

1. `cd client && flutter analyze && flutter test` after every wave.
2. Screenshots at 1440×900 and 390×844, both themes, against the Cobalt-Neo
   decree (0 radius, 2px borders, cast shadows, press feedback) — compare
   with the review's `shots/` baseline.
3. WCAG script: ≥3:1 non-text, ≥4.5:1 text, every new/changed pair.
4. Web bundle inspection after Wave 7.

## Engineering review record

Per AGENTS.md ("run an engineering review of the plan, or record why not"):
**recorded as not-run.** Every wave row and every token above derives
line-by-line from the 40-remediation-plan, which already passed the full
multirole pipeline (skeptic verdicts on every High/Critical plus a
completeness critic). The only new content in this file is the computed
contrast arithmetic, which is stated next to each value and is re-verified
by the Wave 3/5 WCAG-script gate before any visual sign-off.

## Closing checklist

- [x] Dated changelog in `docs/changelogs/` + README index row
- [x] DESIGN.md refreshed to shipped tokens (Wave 3.7, already signed
  with the Wave 3 gate)
- [x] `docs/HANDOFF.md` updated
- [x] Deferred items filed in `docs/issues/` with register IDs (W3-N2)
- [x] This plan marked Finished; board row updated
