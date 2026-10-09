# Frontend block craft — component-layer implementation plan

- **Date:** 2026-10-07 · **Branch:** `main` @ `a762ca8`
- **Source:** user request — optimize the frontend against the design language of
  the reference reimbursement app, keeping the cobalt blue hue family.
- **Predecessor:** [2026-10-06 Frontend Cobalt-Neo remediation](../2026-10-06-frontend-cobalt-neo/2026-10-06-frontend-cobalt-neo.md)
  ported the reference's **token** layer (0px radius, 2px borders, cast shadows,
  press-the-block motion, dark-first). That plan is Finished.
- **This plan** ports the reference's **component and interaction** layer, which
  the token pass did not reach.

## Why

The decree is in the tokens and the tokens are tested
(`client/test/cobalt_test.dart`), but the surfaces still build themselves out of
stock Material. Verified drift, with evidence:

| # | Drift | Evidence | Reference equivalent |
|---|---|---|---|
| D1 | Buttons do not cast or press | `theme/cobalt_theme.dart:98-132` sets squared shape + `elevation: 0` but no cast shadow and no translate; `PressableBlock` only wraps the dock | `BrutalButton` — rest `cast` → hover `castSm`+translate(2,2) → active no shadow+translate(4,4), measured heights 32/40/52 |
| D2 | ~20 raw `AlertDialog` sites, no shared modal chrome | `features/dashboard/dashboard_page.dart:41`, `features/prd/prd_page.dart:104,149`, `features/project/project_page.dart:407,499,561,592`, `features/workspace/workspace_page.dart:74,335,390`, `features/workspace/ideas_page.dart:87,116`, `features/workspace/settings_page.dart:104`, `features/project/roadmap_page.dart:167`, `features/project/roadmap_node_card.dart:143,231`, `features/prd/revisions_page.dart:42`, `features/prd/versions_commit_dialog.dart:53`, `features/profile/admin_section.dart:82`, `features/dashboard/dashboard_page.dart:79` | `showBrutalModal` / `showBrutalConfirm` — 512 max-width, header, 2px divider, scroll body, footer bar |
| D3 | Material floating labels instead of raised fixed-height fields | `features/dashboard/dashboard_page.dart:43-47` | `BrutalInput` — label above, fixed measured height, raised with cast, focus border → accent |
| D4 | Round spinner in a square-block language | `features/dashboard/dashboard_page.dart:134` `CircularProgressIndicator`; DESIGN.md: "status dots and loaders are square blocks" | busy state inside the button |
| D5 | Raw error strings dumped on the field, unlocalized | `features/dashboard/dashboard_page.dart:132` `Center(child: Text('${snap.error}'))` — `api/errors_l10n.dart` exists and is unused on read paths | `BrutalBanner` tone strip |
| D6 | Borderless bare `IconButton` chrome | `app.dart:99-142` — a `Material` with no border and no cast | `BrutalIconButton` — 2px border, hover accent, square badge |
| D7 | An approved primitive uses Material ripple | `theme/field_card.dart:42-46` — `onTap` wraps in `InkWell`, not press-the-block | `_PressShift` |
| D8 | No uppercase/letter-spacing badge, tag, or icon well | absent | `BrutalBadge`, `BrutalTag`, `AccentIconBox` |
| D9 | No content max-width on wide viewports | `features/dashboard/dashboard_page.dart:173-174` — the `ListView` spans the full 1440+ | constrained content column |

## Scope

**In:** `client/lib/theme/`, `client/lib/widgets/`, `client/lib/chrome/`,
`client/lib/app.dart`, and every folder under `client/lib/features/` including
`landing/` and `auth/`.

**Preserved untouched (identity + signed-off behaviour):**

- the brand mark disc — `landing_page.dart:81,198`, `step_peek.dart:95`,
  `ideas_page.dart:378`, `auth_door.dart:91` (DESIGN.md "Identity exception")
- `InkReveal` / `InkStamp` / `inkEase` and the landing + AuthDoor motion
- the web-font first-frame gate (`theme/web_fonts_web.dart`, `session/boot.dart`)
- every assertion in `client/test/cobalt_test.dart` — tokens are frozen
- `server/`, `scripts/`, the CLI, and MCP surfaces

## Wave 0 — primitive contract

New files under `client/lib/theme/`. All square, all 2px `borderContainer`
unless noted, all cast shadows via `cast`/`castSm`, all motion
`Cobalt.pressDuration`.

### `block_button.dart` — `BlockButton`

Sizes are **fixed outer heights, border included**, on the 4px grid:

| Size | Padding (h) | Font | Height |
|---|---|---|---|
| `sm` | 12 | 12 | 32 |
| `md` | 16 | 14 | 40 |
| `lg` | 24 | 16 | 52 |

Choreography (the signature; reproduced exactly):

| State | Shadow | Translate |
|---|---|---|
| rest | `cast` (4,4) | 0 |
| hover | `castSm` (2,2) | (2,2) |
| active | none | (4,4) |

Variants — **one hue only**, no new brand colors:

| Variant | bg | fg | border |
|---|---|---|---|
| `primary` | `btn` (cobalt-500) | `btnFg` | `Cobalt.lightInk` in **both** themes — the hard ink edge against cobalt is the reference's defining detail |
| `secondary` | `surfaceRaised` | `fgPrimary` | `strokeSubtle` |
| `danger` | `brandHover` (cobalt-700) | `btnFg` | `Cobalt.lightInk` — distinguished by weight, not by hue |
| `ghost` | transparent | `fgPrimary` | none; flat, no travel, no shadow |

Label: `Cobalt.sans` + `sansFallback`, w600. `busy` swaps the icon for a 14px
`BlockLoader`. `expand` fills width and ellipsizes. Disabled → `Opacity(0.5)`,
no travel, no shadow, `SystemMouseCursors.basic`. Wrapped in
`Semantics(button: true, enabled: …)`.

### `block_field.dart` — `BlockField`, `BlockTextarea`, `BlockSelect`

Label above (12, w600, `fgSecondary`), 4px gap, then a **fixed-height** raised
box: bg `surfaceRaised`, 2px border (`strokeSubtle` rest / `brand` focus /
`fgDanger` error), `cast` shadow, `TextField` with `InputBorder.none` and
`contentPadding` horizontal 12.

Heights: **44** for `BlockField` and `BlockSelect` (fits the mandated 17.5 body
plus 2px borders on the 4px grid), **40** for a dense field. `BlockTextarea`
uses `minLines`/`maxLines` instead of a fixed height. Error text below at 12 in
`fgDanger`. Disabled → `Opacity(0.5)`.

### `block_modal.dart` — `showBlockModal<T>`, `showBlockConfirm`

`barrierColor` `0xB3000000`, `insetPadding` h16 v24, `maxWidth` 512, transparent
`Dialog`, `Material(color: surfaceRaised)` inside a `Container` with a 2px
`strokeSubtle` border and a `cast` shadow. Column: header (16 padding,
`titleMedium` title + close `BlockIconButton`) → `Divider(height: 2, thickness: 2,
color: strokeSubtle)` → `Flexible(SingleChildScrollView(padding 16, body))` →
optional footer (bg `surfaceSunken`, 2px top border, 16 padding, `Row` end-aligned
with 12px gaps). `showBlockConfirm` returns `bool` and takes `danger`.

### `block_chrome.dart` — `BlockBanner`, `BlockIconButton`, `BlockBadge`, `BlockTag`, `BlockIconWell`, `BlockLoader`

- `BlockBanner(message, tone)` — full width, padding h12 v10, bg `surfaceSunken`,
  **4px left border** in the tone color, text 14 `fgPrimary`.
- `BlockIconButton(icon, onPressed, tooltip, badge, label, hoverAccent)` —
  padding h8 (h12 with a label or badge) v8, bg `surfaceRaised`, 2px border
  (`strokeSubtle` rest → `brand` hover), 16px icon, optional 12/w700 label,
  optional square badge (bg `brand`, fg `btnFg`).
- `BlockBadge(label, tone)` — uppercase, 12, w600, letterSpacing 0.6, padding
  h8 v2, **1px** border, tone-tinted bg.
- `BlockTag(label, color)` — uppercase, 10, w700, letterSpacing 1.0, padding
  h6 v2, **1px** border.
- `BlockIconWell(icon, size 48, iconSize 24)` — square well, bg `brand` @ 10%,
  2px `brand` border, icon in `brand`.
- `BlockLoader(size, color)` — **square**, per the decree. A 2px square outline
  with an inner filled square pulsing opacity 0.25 ↔ 1.0 over 900ms; static when
  `MediaQuery.disableAnimationsOf(context)`.

Tones stay in-hue (`error` → `fgDanger`, `success` → `statusSuccessFg`, `info` →
`brand`). **Amber `#FFD166` is used for `warning` only** — the dock warn dot
already established amber as the warn channel; this widens that precedent and is
recorded as a DESIGN.md amendment in Wave 5.

### `block_state.dart` — `BlockEmpty`, `BlockLoading`, `BlockError`

The read-path trio, centered on the field, content constrained to 480.
`BlockError` localizes through `localizedError(e, t)` from
`api/errors_l10n.dart` and offers a retry; it replaces every raw
`Text('${snap.error}')`.

### Token additions — `cobalt.dart`

Additive only; no existing value changes.

| Token | Value |
|---|---|
| `space1/2/3/4/6/8/10` | 4 / 8 / 12 / 16 / 24 / 32 / 40 |
| `bpNarrow/bpWide/bpDesk` | 600 / 900 / 1200 |
| `contentMaxWidth` | 1120 |
| `controlHeight` / `controlHeightDense` | 44 / 40 |

### `field_card.dart` fix (D7)

When `onTap != null`, wrap in `PressableBlock` and drop `FieldCard`'s own cast
shadow (`PressableBlock` supplies it) — no double shadow, no `InkWell` ripple.
Keep `clipBehavior: Clip.antiAlias`.

## Waves

| Wave | Scope | Files | Status |
|---|---|---|---|
| 0 | Primitive contract + widget tests, analyze/test green | new `theme/block_*.dart`, `theme/field_card.dart`, `theme/cobalt.dart`, `test/block_primitives_test.dart` | ✔ done |
| 1 | Shell — utility icon row → `BlockIconButton` on a bordered cast block; dock on the new primitives | `app.dart`, `chrome/product_dock.dart` | ✔ done |
| 2 | Dashboard + workspace + ideas + settings — modals, fields, states, buttons, max-width | `features/dashboard/**`, `features/workspace/**` | ✔ done |
| 3 | Project + roadmap — modals, row/table treatment, node cards, bands | `features/project/**` | ✔ done — **by Wave 6**: the Wave 3 agent produced no diff; Wave 6 ran this sweep itself (see Status) |
| 4 | PRD + coach + versions + revisions + export + audience | `features/prd/**` | ✔ done |
| 5 | Profile + auth door + landing — block grammar, brand disc and Ink motion preserved | `features/profile/**`, `features/auth/**`, `features/landing/**` | ✔ done; identity exceptions verified against HEAD |
| 6 | Verify — `flutter analyze`, `flutter test`, `dart format`, fix breakage | whole `client/` | ✔ done — all three gates green (evidence in Status) |
| 7 | Fidelity review — re-score every surface against DESIGN.md + the reference | read-only | ✔ done — 7.5/10, findings filed (see Status) |
| 8 | Closure — DESIGN.md amendments, changelog, design-qa, issues, indexes, HANDOFF, board row | `docs/**`, `DESIGN.md` | ✔ done 2026-10-08 |

Waves 2–5 run on **strictly disjoint file sets** so they can run concurrently.
Each wave's gate: `cd client && flutter analyze --no-fatal-infos && flutter test`.

## Verification

1. `cd client && flutter analyze --no-fatal-infos` — no errors, no warnings.
2. `cd client && flutter test` — all existing tests pass, `cobalt_test.dart`
   unmodified and green, plus new primitive tests.
3. `cd client && dart format --output=none --set-exit-if-changed lib test`.
4. Web walk at 1440×900 and 390×844, both themes: buttons cast and press,
   modals share one chrome, no round spinner anywhere, errors localized in a
   banner, content column constrained on wide.
5. WCAG: the ink-on-cobalt primary border clears ≥3:1 non-text; any new tone
   pair clears ≥4.5:1 text / ≥3:1 non-text — asserted in the primitive test.

## Out of scope

Server changes; a redesign below 390px; the deferred W3-N2 minified console
errors ([issue](../../issues/2026-10-06-wave3-minified-console.md)); splitting
`project_page.dart` (640 lines, already filed as modularity debt) unless a wave
naturally lands it; new features; new brand hues.

## Status

**In flight — code complete, one verification item open** 2026-10-08.
Not marked Finished: AGENTS.md closes a plan only when every item is
resolved, and verification item **4 (the web walk at 1440×900 and
390×844, both themes) has never been run**. It needs a live
server/browser and stays user-owned. Items 1–3 and 5 are green.

Current gate, re-run after the focus fix below:
`dart format` 135 files / 0 changed, exit 0; `flutter analyze
--no-fatal-infos` 16 issues, **0 errors / 0 warnings**, exit 0;
`flutter test` **167/167** passed, exit 0; `client/test/cobalt_test.dart`
still unmodified.

Records: [changelog](../../changelogs/2026-10-07-frontend-block-craft.md),
[focus-ring changelog](../../changelogs/2026-10-08-block-focus-ring.md),
and the [design-QA thread](../../design-qa/2026-10-07-block-craft.md);
DESIGN.md was amended to match what shipped. The whole slice is
**uncommitted**, and the worktree also carries the app-icon slice —
stage one task at a time.

**Wave 7's one high finding is fixed (2026-10-08).** The sweep had
replaced keyboard-focusable Material buttons with mouse-only blocks;
`BlockFocus` now gives `BlockButton`, `BlockIconButton`, and
`PressableBlock` Tab traversal, Enter/Space activation, and the cobalt
ring DESIGN.md mandates, painted outside the block bounds so no measured
control height moved
([issue resolved](../../issues/2026-10-08-block-focus-ring-keyboard.md),
7 new tests in `client/test/block_focus_test.dart`). The medium/low
residue stays filed, not fixed:
[fidelity follow-ups](../../issues/2026-10-08-block-craft-fidelity-followups.md).
The regression class is recorded as
[replacing a platform widget drops the affordances no test asserts](../../learning/2026-10-08-custom-primitives-drop-unasserted-affordances.md)
— the gate was green at 160/160 with the hole open, because no test
pressed Tab.

**Two of the five sweep agents failed and the run still produced a
correct tree.** `wave3:project` died on a model-service error and
`wave4:prd` hit the 150-turn limit; Wave 6 absorbed the project sweep.
The PRD partition was in fact swept (9 files carry the block primitives)
despite its agent returning nothing — verified by grep, not by trusting
the summaries. Two lessons for the next orchestration: a per-agent
return value is not evidence that its partition was edited, and the
script's own `sweep_summaries` mislabeled results because it indexed a
filtered array (`sweeps.filter(Boolean)[i]`) against the unfiltered
surface list, so the failing partitions reported as succeeding.
Recorded as a reusable lesson:
[an agent's self-report is not evidence](../../learning/2026-10-08-agent-self-report-is-not-evidence.md).

Execution was handed to a background workflow rather than run inline:

| | |
|---|---|
| Task ID | `wf-67acfb81-b5e` |
| Run ID | `wf_6b6a6481-1a2` |
| Script | `.qoder/sessions/<session-id>/workflows/scripts/frontend-block-craft-wf_6b6a6481-1a2.js` (local only, not committed) |
| Journal | `.qoder/sessions/<session-id>/workflows/runs/wf_6b6a6481-1a2/journal.jsonl` (local only, not committed) |
| Live view | `/workflows` |

Phase map: Wave 0 ran **alone** (one agent owned the whole primitive
contract so the public API stayed coherent); Waves 1–5 then ran **five
agents concurrently** on strictly disjoint partitions; Waves 6–8 ran
alone again in order.

Two concurrency rules were load-bearing and were written into every sweep
prompt — see [the dirty-worktree lesson](../../learning/2026-10-07-parallel-slots-dirty-worktree.md)
for why:

1. **Partition by shared-file ownership, not just by feature.** Sweeps
   were forbidden from `lib/theme/**`, `lib/widgets/**`, and
   `lib/l10n/**`. l10n was pre-seeded in Wave 0 and then owned solely by
   Wave 6, because five agents appending keys to the same two tables
   would collide on every run.
2. **Each sweep ran `flutter analyze` but fixed only diagnostics inside
   its own partition**, and did not run `flutter test` at all. Other
   partitions were mid-edit, so their diagnostics were transient noise;
   five concurrent suites also thrash the machine. Wave 6 owned the gate.

The rules worked except in one respect the journal cannot show: **the
Wave 3 agent (`features/project/**`) produced no diff at all** — zero
changed files, so all seven raw `AlertDialog`s and three round spinners
from the D2/D4 evidence tables were still standing when Wave 6 started.
Wave 6, as sole owner of the tree by then, ran that sweep itself. A
silent no-op partition is a failure mode the concurrency design did not
guard against; a per-wave diff check at the join would have caught it
earlier.

### Verification evidence (Wave 6, run in `client/` 2026-10-08)

- `flutter analyze --no-fatal-infos` — `16 issues found. (ran in 0.9s)`;
  0 errors, 0 warnings, 16 pre-existing style infos in `lib/api/` and
  two test files (baseline 20; the sweep removed 4).
- `flutter test` — `00:09 +160: All tests passed!`;
  `test/cobalt_test.dart` unmodified and green (items 1–2).
- `dart format --output=none --set-exit-if-changed lib test` —
  `Formatted 133 files (0 changed) in 0.29 seconds.`, exit 0 (item 3).
- Item 4 (web walk at 1440×900 and 390×844, both themes) — **not run**;
  the review wave was read-only and the workflow had no browser. Open,
  user-owned, together with the 390px utility-cluster crowding check.
- Item 5 (WCAG) — asserted in `test/block_primitives_test.dart`
  ("every block tone clears 3:1 accent and 4.5:1 ink"), green.

### Wave 7 fidelity review — 7.5/10, filed not buried

All nine drifts D1–D9 are substantively resolved (zero raw
`AlertDialog`/`showDialog`/`InkWell`/floating-label/round-spinner sites
outside the protected boot gate; `contentMaxWidth` on read surfaces;
exact press choreography at 100ms; en/zh parity at 366 keys; identity
exceptions byte-identical to HEAD). What kept it off 9, filed the same
day:

- **[high]** no focus ring or keyboard activation on any custom block —
  an a11y regression against the Material buttons the sweep replaced
  ([issue](../../issues/2026-10-08-block-focus-ring-keyboard.md)).
- **[medium ×4 / low ×6]** raw M3 popup menus, flat workspace FAB,
  coach panel collapsing `ApiException`s to the generic error,
  dark-mode cobalt-950 on band/poster slabs, and six low findings
  ([follow-ups issue](../../issues/2026-10-08-block-craft-fidelity-followups.md)).
- **Modularity** — `project_page.dart` 675 lines
  ([issue](../../issues/2026-10-08-project-page-line-budget.md)), l10n
  tables 414/372
  ([issue](../../issues/2026-10-08-l10n-tables-line-budget.md)),
  `roadmap_canvas.dart` 400
  ([issue](../../issues/2026-10-08-roadmap-canvas-line-budget.md)).
