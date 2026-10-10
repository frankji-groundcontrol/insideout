# 2026-10-08 — Block-craft fidelity follow-ups (Wave 7 medium and low findings)

> **Resolved 2026-10-10.** Every item is done or was reviewed and kept. Popup menus: one
> `popupMenuTheme` (`fd7e59e`). Quick-capture FAB: a cast-and-press block with a
> name (`9cf00cb`). Coach errors map by status and code (`c2fd281`, and
> `0fd2547` for a failure landing after the panel is gone). Cobalt-950 slabs:
> bands and the progress band (`de817fe`), idea poster and version peeks
> (`47f9734`). Low items: progress cards (`de817fe`); update kinds and ages
> (`ad7a688`, `c435fd8`); minimap marks and the 12px floor (`53e509d`); square
> pull-to-refresh (`f2a77e4`) and date picker (`ff34e00`); the cleared-deadline
> label (`44e7003`). The 390px crowding risk was real: the web walk found the
> roadmap header and the PRD page broken at that width and both are fixed (BUG-025).
> See [client fidelity fixes](../changelogs/2026-10-10-close-open-gaps/client-fidelity-fixes.md) and
> [web walk](../changelogs/2026-10-10-close-open-gaps/web-walk-and-phone-layouts.md).
>
> The original record follows, unedited.

The Wave 7 fidelity review of the
[block-craft plan](../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md)
scored the component pass 7.5/10. The high finding (focus ring /
keyboard) is filed separately
([issue](2026-10-08-block-focus-ring-keyboard.md)); this issue holds the
medium and low residue, each a bounded fix. Line numbers are against
the uncommitted worktree of 2026-10-08.

## Medium

- **Four raw `PopupMenuButton`s still open stock M3 menus** (rounded
  4px, blurred elevation, item ripples) — no `popupMenuTheme` exists in
  `lib/`: `features/workspace/idea_poster.dart:79`,
  `features/workspace/settings_page.dart:233`,
  `features/project/roadmap_node_card.dart:126`,
  `features/project/project_page.dart:441`. Only
  `features/prd/prd_desk_actions.dart:67-101` restyles its menu
  (NoSplash, square, 2px border, elevation 0). Fix: hoist that
  treatment into a `popupMenuTheme` in `theme/cobalt_theme.dart` (or a
  `showBlockMenu` helper in `theme/block_modal.dart`) and adopt it at
  the four sites.
- **Workspace quick-capture FAB is flat** — a stock
  `FloatingActionButton` (squared, zero elevation) that cannot cast or
  press, with no tooltip/semanticLabel:
  `features/workspace/workspace_page.dart:155-158`. Fix: replace with a
  `PressableBlock`-based control (or a `lg` icon `BlockButton`) that
  carries the (4,4) cast and the 100ms travel.
- **Coach panel defeats `localizedError`** — the `ApiException` branch
  stores `error = e.message` (the raw server string,
  `features/prd/coach_panel.dart:105`), so line 226's `localizedError`
  sees a plain `String` and every coach API failure collapses to
  `error.generic`; line 240 then language-sniffs the raw English
  message for rate-limit copy. The sibling catch (108-110) documents
  the correct keep-the-object rule. Fix: store the exception object and
  let `localizedError` map it (D5 residue).
- **Dark-mode cobalt-950 on slab surfaces** — DESIGN.md reserves
  cobalt-950 (`brandSubtle` in dark) for status-chip backgrounds, but
  it paints the whole Next band (`features/project/roadmap_bands.dart:79`),
  the needsDeadline band (`features/project/roadmap_progress_view.dart:81`),
  the idea-poster gradient end (`features/workspace/idea_poster.dart:44`),
  and the versions peek cards (`features/prd/versions_stack.dart:145`).
  Fix: move band/poster surfaces to `surfaceSunken`/`surfaceRaised`, or
  amend DESIGN.md deliberately if the bands are meant to be brand-tinted.

## Low

- Progress-view item cards are bare `Container(color: surfaceRaised)` —
  no 2px border, no cast (`features/project/roadmap_progress_view.dart:131-136`),
  inconsistent with the card grammar.
- Raw server enum + raw ISO timestamps: `features/project/project_page.dart:403`
  prints `u.kind` (e.g. `agent_proposal`) where `board_project_card.dart:92`
  localizes via `project.updateKind.*`; `:412` prints raw `createdAt`
  where the app uses `timeAgo`. Same raw-ISO pattern at
  `features/prd/prd_desk.dart:105` and `features/prd/revisions_table.dart:196`.
- Off-scale 9px/10px type and hardcoded-English minimap labels under
  zh-CN (`features/project/roadmap_canvas.dart:248`, `:124`, `:343-354`)
  — below the 12px meta floor; acknowledged in the file comment but
  still unlocalized user-visible strings.
- Round chrome returns via `RefreshIndicator`
  (`features/workspace/workspace_page.dart:163`) and `showDatePicker`
  (`features/project/roadmap_node_card.dart:275`) against the
  square-loaders decree. (The boot splash spinner is plan-protected and
  not part of this finding.)
- Clearing a deadline leaves an empty button label:
  `features/project/roadmap_node_card.dart:297` sets `deadline = ''`,
  and `:268-270` renders `isoDate('')` = '' instead of falling back to
  "Set deadline".
- 390px crowding risk: `AppScaffold` lays page actions plus 2–4
  `BlockIconButton`s in one unwrapping top-right row (`app.dart:112-168`)
  and RoadmapPage injects three labeled `BlockChoice`s
  (`features/project/roadmap_page.dart:191-202`) — borderline at 390px
  with English labels; check in the pending 390×844 web walk and add
  wrapping/collapse if it overflows.

## Bounded fix prompt

"Work the medium items (then any low items) in
[docs/issues/2026-10-08-block-craft-fidelity-followups.md](2026-10-08-block-craft-fidelity-followups.md),
one commit per finding or one commit per medium cluster. No new hues;
keep en/zh l10n parity (`l10n_test.dart` green). Gate per fix:
`cd client && flutter analyze --no-fatal-infos && flutter test && dart
format --output=none --set-exit-if-changed lib test`. Tick items off
here as they land; close the issue when the mediums are done or
explicitly re-deferred."
