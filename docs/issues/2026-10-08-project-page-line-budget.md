# 2026-10-08 — `client/lib/features/project/project_page.dart` over the ~350-line budget

> **Resolved 2026-10-10.** `project_page.dart` is 297 lines; the GitHub card, updates feed, modals and
> the done-with-evidence block are `project_github_card.dart` (85),
> `project_updates_feed.dart` (180), `project_dialogs.dart` (144) and
> `project_leaf_evidence.dart` (68) (`78bf90a`). `project_flows_test.dart`
> (`d2321aa`) pins every Evidence write flow and passes unchanged before and after
> the split. See [client file splits](../changelogs/2026-10-10-close-open-gaps/client-file-splits.md).
>
> The original record follows, unedited.

`project_page.dart` is 675 lines: the Evidence page shell (load/reload,
dock evidence-pulse flash, header field), the GitHub sync card, the
updates feed (agent-proposal Accept/Reject loop plus the note edit/delete
menu), and three block modals (edit project, delete project, post an
update). It was 640 lines of stock-Material dialogs before the
[frontend block craft plan](../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md)
declared its split out of scope; the Wave 6 sweep onto the block
primitives (`showBlockModal`, `BlockField`, `BlockSelect`, `BlockButton`)
added ~34 lines of named-argument chrome. The plan's Status section calls
this debt "already filed" — this file is that filing.

## Why not split now

Wave 6 is the verification gate: a four-way restructure of the largest
feature file lands worse there than alone. The page is cohesive (one
route, one data load) — the oversize is a sectioning problem, not a
tangle.

## Target structure

Keep `project_page.dart` as the page shell (state, load/reload, evidence
pulse, header, layout — under ~300 lines), and extract:

- `project_github_card.dart` — the GitHub sync `FieldCard` (repo
  `BlockField`, save + sync buttons, synced SnackBar).
- `project_updates_feed.dart` — the updates list: proposal decision loop
  (Accept/Reject), the note popup menu, and the edit-update
  `showBlockForm` call.
- `project_dialogs.dart` — `showProjectEdit`, `showProjectDelete`,
  `showProjectAddUpdate` (the three modal builders now inline as
  `_edit`/`_delete`/`_addUpdate`).

Neighbor to consider in the same pass: `roadmap_canvas.dart` (400 lines,
pre-existing at HEAD, canvas + bands + minimap + dash painter).

## Bounded fix prompt

"Split `client/lib/features/project/project_page.dart` (675 lines) into
the shell plus `project_github_card.dart`, `project_updates_feed.dart`,
and `project_dialogs.dart` per
[docs/issues/2026-10-08-project-page-line-budget.md](2026-10-08-project-page-line-budget.md).
No behavior changes: `project_page_test.dart` (Accept/Reject, more_vert
menu, repo save error, evidence pulse) and `feature_wiring_test.dart`
(`updateProject`, `deleteProject`, `editUpdate`, `removeUpdate`
substrings) must pass unmodified. Gate: `cd client && flutter analyze
--no-fatal-infos && flutter test && dart format --output=none
--set-exit-if-changed lib test`. Every touched file ends under ~350
lines. Record the split in a changelog entry and close this issue."
