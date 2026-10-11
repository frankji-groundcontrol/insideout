# Remediation plan — 2026-10-06 frontend cobalt review

Ordered by impact over effort. Each wave ends with one runnable check; do not
start a wave until the previous wave's check passes. Register IDs refer to
[01-findings-register.md](01-findings-register.md). All paths under
`client/` unless noted.

*Corrected 2026-10-11: many File-column paths above never existed at the
reviewed tree (`f185e94`) — there is no `lib/widgets/`, no
`lib/features/ideas|admin|settings|coach|roadmap/`, and no `test/widgets/`.
Real locations (verified via `git ls-tree` at `f185e94` and HEAD):
`product_dock.dart` → `lib/chrome/`, `ink_motion.dart` → `lib/theme/`,
`ideas_page.dart` → `lib/features/workspace/`, `admin_section.dart` →
`lib/features/profile/`, `appearance.dart` → `lib/session/`, `coach_panel.dart`
→ `lib/features/prd/`, `roadmap_canvas.dart` → `lib/features/project/`,
`product_dock_test.dart` → `client/test/`. The register cites the correct
paths.*

## Wave 0 — safety

Nothing to do: no committed secrets found, `main` is the product branch,
license present, ownership clear. (Critic probed the never-assigned security
dimension read-only: no token logging, tokens only in `Authorization`
headers; one Low filed as `critic-auth-secrets-in-url-query`, fixed in
Wave 7.)

## Wave 1 — one-line & token-value fixes (single commit, ~1 day)

| # | Register ID | File | Change | Size |
|---|---|---|---|---|
| 1.1 | dock-renders-at-viewport-center, dock-pill-mid-screen-root-cause, pm-dock-mid-viewport | `lib/widgets/product_dock.dart:26-28` | `Center` → `Align(alignment: bottomCenter, heightFactor: 1.0)` | ~3 lines |
| 1.2 | missing-viewport-meta | `web/index.html` | Add `<meta name="viewport" content="width=device-width, initial-scale=1.0">` | 1 line |
| 1.3 | feat-idea-status-raw-enum | `lib/features/ideas/ideas_page.dart:326` | Replace raw enum with `l10n.t('idea.status.${idea.status}')` | 1 line |
| 1.4 | admin-hardcoded-red-dark-unreadable | `lib/features/admin/admin_section.dart:167` | `Color(0xFFB3261E)` → `colorScheme.error` | 1 line |
| 1.5 | light-mode-default | `lib/features/settings/appearance.dart:36` | Default to dark (or follow system); note the new default in DESIGN.md | 1 line + doc |
| 1.6 | rel-hydrate-wipes-tokens | `lib/session/session.dart:58` | Only `clear()` stored tokens on 401/403, never on transport errors | ~4 lines |
| 1.7 | rel-no-network-timeouts | `lib/api/client.dart:30` | Dio `connectTimeout: 10s`, `receiveTimeout: 30s`; hydrate 15s overall | ~3 lines |
| 1.8 | stroke-tokens-fail-non-text-contrast | `lib/theme/cobalt.dart` | Retune: light strokeStrong ≈ #6E7FD6, light strokeSubtle ≈ #94A3E8, dark strokeSubtle ≈ #4A5680 | 3 lines |

**Check:** `cd client && flutter analyze && flutter test`; launch the web
build at 1440×900 and confirm the dock sits at the bottom of the viewport on
landing, projects, and roadmap pages.

## Wave 2 — dock geometry, clearance & header collisions (~2 days)

| # | Register ID | File | Change | Size |
|---|---|---|---|---|
| 2.1 | no-dock-clearance-coach-and-canvas | coach panel + roadmap canvas root Columns | Bottom padding = `ProductDock.height + 16 + safeArea.bottom` (or drop `extendBody`) | ~10 lines ×2 |
| 2.2 | create-project-fab-invisible | `lib/features/workspace/workspace_page.dart:115` | After 1.1 the FAB returns; also add an explicit new-project button in the projects section | ~15 lines |
| 2.3 | topbar-icons-invisible-over-navy | `lib/app.dart:90` | Wrap the utility icon row in a surfaceRaised (95%) pill, or use a real AppBar; add a ≥3:1 golden assertion | ~10 lines + test |
| 2.4 | pm-authdoor-header-collision | `lib/features/auth/auth_door.dart:50` | Move dot/wordmark/greeting inside the modal card (or make the barrier fully opaque); re-shoot screens 02/03 | ~15 lines |
| 2.5 | project-page-evidence-heading | `lib/features/project/project_page.dart:150` | Project name becomes the display heading; "证据" demoted to eyebrow/section label | ~6 lines |
| 2.6 | full-dock-crosses-workspaces | `lib/widgets/product_dock.dart:91` | Route-first focus (`^/projects/` path wins); focus keyed per workspace | ~12 lines |

**Check:** re-shoot all pages at 1440×900 and 390×844; confirm dock at
bottom, no content hidden behind it, top-bar icons readable, AuthDoor header
inside the card.

## Wave 3 — Cobalt-Neo token system + DESIGN.md (~3 days)

Gate: draft the token table as the dated plan
(`docs/plans/2026-10-06-frontend-cobalt-neo/2026-10-06-frontend-cobalt-neo.md`) and review it before code —
all values below are blocked on that sign-off (design-fidelity's first move).

| # | Register ID | File | Change | Size |
|---|---|---|---|---|
| 3.1 | radius-decree-inverse | `lib/theme/cobalt.dart:72-75` | All four radius tokens → 0.0 (or delete the group, migrate to `radiusZero`); sweep features for hardcoded `BorderRadius` | 4 lines + sweep |
| 3.2 | no-shadow-system | `lib/theme/cobalt_theme.dart:54-61` | Add `cast` (4px 4px 0 0) and `castSm` (2px 2px 0 0) tokens — dark: solid ink; light: ~15% alpha; wire into FieldCard, cardTheme, AuthDoor, ProductDock, dialogs, toasts | ~30 lines |
| 3.3 | hairline-border-weight | `lib/theme/cobalt.dart` + call sites | Add `borderWidth` tokens (container 2.0, divider 1.0); wire into FieldCard, cardTheme, ShadeChip, AuthDoor, ProductDock, inputs, dialogs | ~25 lines |
| 3.4 | no-press-the-block | `lib/widgets/ink_motion.dart:4` | Motion tokens (pressDuration 100ms, hoverOffset (2,2), activeOffset (4,4)) + shared InkWell/ButtonStyle extension | ~40 lines |
| 3.5 | dock-unfinished-blue-on-blue | `lib/widgets/product_dock.dart:166` | Amber warn token (#FFD166, ≈7:1 on dock brand) or white dot + tooltip | ~6 lines |
| 3.6 | blue-tinted-dark-ramp | `lib/theme/cobalt.dart:133-137` | New neutral dark gray ramp; hue950 demoted to brandSubtle-only; statusNeutralBg remapped to #1A1F2E | ~12 lines |
| 3.7 | critic-design-md-chipshade-textcolor, design-md-contract-stale | `DESIGN.md` | Refresh DESIGN.md to the shipped/approved tokens; pick one authority for chip text color | doc |

**Check:** WCAG contrast script over every new/changed token pair (target
≥3:1 non-text, ≥4.5:1 text); visual pass on shots of every page in both
themes against the Cobalt-Neo decree (0 radius, 2px borders, cast shadows,
press feedback).

## Wave 4 — write reliability & stream timeouts (~2 days)

| # | Register ID | File | Change | Size |
|---|---|---|---|---|
| 4.1 | full-write-actions-fail-silently | 6 write sites (ideas/workspace/project pages) | Guarded-action helper: try/catch → localized-error SnackBar, success → reload; model on `project_page.dart` `_decide` | ~50 lines |
| 4.2 | feat-capture-convert-no-error-handling | `lib/features/ideas/ideas_page.dart:60` | Same helper; on failure keep the input text | ~8 lines |
| 4.3 | rel-coach-stream-no-timeout | `lib/features/coach/coach_panel.dart:212` | 75s idle-timeout Timer on the SSE stream | ~10 lines |
| 4.4 | dock-dead-clicks-evidence-version | `lib/widgets/product_dock.dart` | Evidence: scroll-highlight when already on `/projects/<id>`; Version: go to `/prd/<id>/versions` or disable the tile | ~15 lines |

**Check:** for each of the six writes, kill the network mid-action and
confirm a localized error appears and state is preserved; coach stream idles
75s → clean timeout, retry works.

## Wave 5 — non-color accessibility channels (~1 day)

| # | Register ID | File | Change | Size |
|---|---|---|---|---|
| 5.1 | minimap-color-only-invisible-marks | `lib/features/roadmap/roadmap_canvas.dart:172` | Marks ≥3:1 plus a non-color channel: filled = done, hollow = pending, dashed = locked (or Semantics + tooltip) | ~20 lines |
| 5.2 | (verify) | theme | Re-run WCAG script across all retuned tokens from 1.8/3.5/3.6 | script |

**Check:** WCAG script passes; minimap states distinguishable in grayscale
screenshot and via screen-reader labels.

## Wave 6 — test harness, goldens & format gate (~2 days)

| # | Register ID | File | Change | Size |
|---|---|---|---|---|
| 6.1 | test-dock-harness-hides-overlap | `test/widgets/product_dock_test.dart:17` | Mount via AppScaffold (or replicate `extendBody: true` + real body); at 1400×900 assert dock rect does not overlap content | ~25 lines |
| 6.2 | test-no-goldens-modal-collision | `test/widget_test.dart:12` | 1440×900, both themes, zh-CN golden of LandingPage + AuthDoor (or geometric assertion: wordmark inside modal card) | ~30 lines |
| 6.3 | (format drift, 55/68 files) | `client/lib` | `dart format` sweep as its own commit; add `--set-exit-if-changed` to CI | sweep + CI line |
| 6.4 | pm-flutter-version-unpinned | CI / `client/Dockerfile` | Pin Flutter 3.44.0 everywhere; fix `docs/usage/local-development.md:11` | 2 lines + doc |

**Check:** CI green with the new format gate and golden tests; deleting the
1.1 fix makes the new goldens fail (prove the harness can see the bug).

## Wave 7 — bundle & URL hygiene (~0.5 day)

| # | Register ID | File | Change | Size |
|---|---|---|---|---|
| 7.1 | critic-web-bundle-ships-41mb-unused-fonts | `client/pubspec.yaml:66`, `client/Dockerfile` | Exclude the two CJK variable fonts (24M + 17M) from web builds; web loads faces via CDN already | ~6 lines |
| 7.2 | critic-auth-secrets-in-url-query | `lib/router.dart:28,30,31` | Consume invite/reset/magic-link secrets, then replace the history entry | ~10 lines |

**Check:** web bundle `assets/fonts/` no longer ships the CJK fonts and
first paint is unchanged; after using a magic link, the secret is absent
from `window.history`.

## Not in scope for this release

- Server-side changes; responsive redesign below 390px; performance work
  beyond 7.1 (no other hotspot measured); full localization audit; any
  re-architecture of the theme file structure.
- Score-calibration mismatches noted by the critic (a11y 4.5, feature 7.2,
  PM 6.0) — record-keeping for the next review, not work items.
