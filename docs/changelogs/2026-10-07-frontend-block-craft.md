# 2026-10-07 — Frontend block craft: the component layer landed

**Scope:** the whole
[plan](../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md),
run as background workflow `wf-67acfb81-b5e` (run `wf_6b6a6481-1a2`):
Wave 0 alone, waves 1–5 as five concurrent disjoint partitions, waves
6–8 alone. Companion to the
[Cobalt-Neo token pass](2026-10-06-frontend-cobalt-neo.md) — the tokens
stayed frozen (`client/test/cobalt_test.dart` untouched and green); this
ported the reference app's **component and interaction** layer onto them
and removed the nine recorded drifts D1–D9. Nothing is committed or
deployed; closure finished 2026-10-08.

## What changed

- **Wave 0 primitives** — new files under `client/lib/theme/`:
  `block_button.dart` (`BlockButton` with fixed outer heights sm 32 /
  md 40 / lg 52 and four variants; primary and danger edged in
  `Cobalt.lightInk` in **both** themes; `BlockChoice` selection block),
  `block_field.dart` (`BlockField` / `BlockTextarea` / `BlockSelect` —
  label above, raised, fixed 44 with a dense 40, `helper` and error
  slots), `block_modal.dart` (`showBlockModal` / `showBlockConfirm` —
  the one 512-max chrome), `block_chrome.dart` (`BlockBanner`,
  `BlockIconButton`, `BlockBadge`, `BlockTag`, `BlockIconWell`, square
  `BlockLoader`, `BlockTone`), `block_state.dart` (`BlockEmpty` /
  `BlockLoading` / `BlockError`, localized through `api/errors_l10n.dart`
  with a retry), `block_form.dart` (`showBlockForm`). `field_card.dart`
  now presses via `PressableBlock` instead of rippling via `InkWell`
  (D7). Additive `cobalt.dart` tokens only: `space1`–`space10`,
  breakpoints 600/900/1200, `contentMaxWidth` 1120, `controlHeight`
  44/40, `warnAmber` `#FFD166` (the dock warn-dot precedent widened
  into the warning tone channel — a DESIGN.md amendment, now recorded
  there).
- **Waves 1–5 + the Wave 6 sweep** — shell (`app.dart` utility row,
  dock), dashboard, workspace, ideas, settings, project, roadmap, PRD,
  coach, versions, revisions, export, audience, profile, admin, auth
  door, and landing all moved onto the primitives. End state in
  `lib/`: zero raw `AlertDialog`, zero bare `showDialog` outside
  `showBlockModal`, zero `InkWell`, zero floating `labelText` fields,
  zero round spinners except the plan-protected boot gate, and
  `contentMaxWidth` on the read surfaces (D9). **Honest note:** the
  Wave 3 agent (project partition) produced no diff at all — the seven
  raw dialogs and three spinners of the D2/D4 evidence tables were
  still standing when Wave 6 started. Wave 6, as sole owner of the
  tree, ran that sweep itself.
- **Wave 6 shared work** — gate finders updated to the new widget types
  (no assertion weakened); `showBlockForm` promoted from
  `features/workspace/` to `theme/` (killing a cross-feature import);
  `eyebrow(context)` added to `cobalt_theme.dart`, collapsing three
  duplicate copies; two app-shell tests relocated verbatim from
  `product_dock_test.dart` into a new `app_shell_test.dart`; l10n fixed
  a real bug (a confirm used the **missing** key `common.post` and
  rendered the raw key — now `project.post`) and added
  `common.moreActions` + `admin.revokeConfirm`; en/zh parity exact at
  366 keys.
- **Identity exceptions preserved, verified against HEAD:** brand mark
  discs, `InkReveal` / `InkStamp` / `inkEase`, the web-font first-frame
  gate, `session/boot.dart`, `theme/web_fonts_web.dart`, `web/index.html`,
  and every frozen assertion in `test/cobalt_test.dart`. No new
  `Color(0xFF…)` literals outside the plan-mandated `0xB3000000` modal
  barrier.

## Wave 7 fidelity review — 7.5/10, recorded honestly

The review confirmed all nine drifts substantively resolved, the exact
rest-cast / hover-castSm+(2,2) / active-flush+(4,4) choreography at
100ms (asserted in `test/block_primitives_test.dart`), WCAG 3:1/4.5:1
tone pairs, and the identity exceptions. It did **not** score 9, and
the gap is real debt, filed the same day:

- **[high] No focus ring or keyboard activation on any custom block.**
  `BlockButton`, `BlockIconButton`, `PressableBlock`, and `BlockChoice`
  are `MouseRegion` + `GestureDetector` only. The sweep replaced
  keyboard-focusable Material buttons with them, so every primary
  action, dock item, and icon control is unreachable by Tab and shows
  no focus indicator — an app-wide a11y regression against the
  pre-sweep baseline and against DESIGN.md's "Focus is a cobalt ring"
  ([issue](../issues/2026-10-08-block-focus-ring-keyboard.md)).
- **[medium] residue** — four raw `PopupMenuButton`s still open stock
  M3 menus; the workspace quick-capture FAB neither casts nor presses;
  the coach panel stores the raw server message and collapses every
  API failure to the generic localized error (D5 residue); dark-mode
  `cobalt-950` (`brandSubtle`) paints slab-sized band/poster surfaces
  the decree reserves for status chips; plus six low findings
  (borderless progress cards, raw enum/ISO strings on the updates
  feed, 9–10px unlocalized minimap labels, `RefreshIndicator` and
  date-picker round chrome, an empty deadline label after clear,
  a possibly crowded 390px utility cluster). All are in the
  [fidelity follow-ups issue](../issues/2026-10-08-block-craft-fidelity-followups.md).
- **Modularity** — `project_page.dart` ended at 675 lines
  ([issue](../issues/2026-10-08-project-page-line-budget.md)), the l10n
  tables at 414/372
  ([issue](../issues/2026-10-08-l10n-tables-line-budget.md)), and
  `roadmap_canvas.dart` at 400
  ([issue](../issues/2026-10-08-roadmap-canvas-line-budget.md)).

## Verification

Local gate in `client/`, 2026-10-08 (verbatim tails):

- `flutter analyze --no-fatal-infos` — `16 issues found. (ran in 0.9s)`;
  0 errors, 0 warnings, 16 pre-existing style infos in `lib/api/` and
  two test files (baseline was 20; the sweep removed 4).
- `flutter test` — `00:09 +160: All tests passed!` (143 at the token
  pass; +17 primitive, shell, and sweep tests).
- `dart format --output=none --set-exit-if-changed lib test` —
  `Formatted 133 files (0 changed) in 0.29 seconds.`, exit 0.

**Not run:** the plan's web walk at 1440×900 and 390×844 in both
themes (verification item 4). The review wave was read-only and could
not execute it; it stays a user-owned remainder, together with the
390px crowding check the review flagged.

## Operator note

No migration, no server change. The whole slice is **uncommitted** in
the worktree (client sweep + new theme files + these records), and a
deployed web client does not pick it up until the client is rebuilt and
shipped. `DESIGN.md` was amended to match what shipped; the design-QA
thread is recorded at
[design-qa/2026-10-07-block-craft.md](../design-qa/2026-10-07-block-craft.md).
