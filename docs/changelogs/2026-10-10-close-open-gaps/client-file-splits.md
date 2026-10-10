# Client file splits

Eleven files were over the ~350-line budget in CLAUDE.md: ten in `client/` and
one Go test. Each is now split along a responsibility seam with no behavior
change. Line counts are the base commit (`8ee97fc`) against the tree now, so
the "after" column includes the fidelity fixes that landed on top of the
splits ([client-fidelity-fixes.md](client-fidelity-fixes.md)).

| File | Before | After | New file(s) |
| --- | --- | --- | --- |
| `client/lib/api/client.dart` | 851 | 27 | `client_core.dart` (120), `client_auth.dart` (145), `client_workspace.dart` (126), `client_project.dart` (196), `client_prd.dart` (142), `client_stream.dart` (144) |
| `client/lib/api/models.dart` | 577 | 4 | `models_session.dart` (91), `models_project.dart` (123), `models_roadmap.dart` (138), `models_prd.dart` (222) |
| `client/lib/theme/block_chrome.dart` | 360 | 247 | `block_icon_button.dart` (118) |
| `client/lib/theme/block_field.dart` | 357 | 226 | `block_field_frame.dart` (134) |
| `client/lib/features/project/project_page.dart` | 675 | 297 | `project_dialogs.dart` (144), `project_github_card.dart` (85), `project_updates_feed.dart` (180), `project_leaf_evidence.dart` (68) |
| `client/lib/features/project/roadmap_canvas.dart` | 400 | 200 | `roadmap_minimap.dart` (184), `roadmap_painters.dart` (48) |
| `client/lib/l10n/en.dart` | 414 | 34 | `en_shell.dart` (68), `en_account.dart` (110), `en_workspace.dart` (153), `en_prd.dart` (102) |
| `client/lib/l10n/zh.dart` | 372 | 18 | `zh_shell.dart` (65), `zh_account.dart` (98), `zh_workspace.dart` (145), `zh_prd.dart` (90) |
| `client/test/product_dock_test.dart` | 472 | 184 | `product_dock_visual_test.dart` (226), `_dock_test_helpers.dart` (73) |
| `client/test/block_primitives_test.dart` | 574 | 211 | `block_button_test.dart` (156), `block_field_test.dart` (104), `block_modal_test.dart` (111), `_block_test_helpers.dart` (19) |
| `server/internal/api/auth_throttle_test.go` | 363 | 314 | `auth_client_ip_test.go` (56) |

No tracked source file is over 350 lines now.

## Seams

- **`ApiClient`** is a mixin application: a private core class (construction,
  the Dio interceptors, the single-flight refresh, `_exec`, `_parse`) plus one
  private mixin per resource, each in its own `part` file. The verbs stay
  ordinary instance members, so test subclasses keep overriding them and no
  import site changed. `feature_wiring_test` greps client source text, so its
  concatenation also lists the part files that hold the verbs it asserts on.
- **`models.dart`** is a four-line barrel that re-exports four domain files.
  `Conversation` and `ChatMessage` sit with the PRD because they are the
  coach's per-PRD types.
- **`block_chrome.dart`** re-exports `BlockIconButton`, the one interactive
  control in a file of passive chrome. **`block_field.dart`** keeps its
  private names through `part` / `part of`, so the shared field shell
  (`_inputStyle`, `_fieldDecoration`, `_FieldFrame`) did not have to go public.
- **`project_page.dart`** keeps the page shell. The GitHub card, the updates
  feed, three modals, and the "done, with its evidence" block are widgets that
  take callbacks. The four API verbs stay in `project_page.dart` as thin
  wrappers or injected closures because `feature_wiring_test` reads only that
  file.
- **`roadmap_canvas.dart`** re-exports the minimap and the dash painter, so
  `roadmap_page.dart` and the canvas test keep their imports.
- **l10n** is four namespace files per locale (shell, account, workspace, PRD)
  reassembled by spread behind the same `enUS` / `zhCN` barrels; the coach
  suggestion lists stay in the barrels.
- **Test splits** move tests verbatim. The harness each pair shares becomes a
  public helper in an underscore-led file (`pumpDock`, `host`), because a
  private name cannot cross files and the leading underscore keeps
  `flutter test` from running the helper.

## How it was verified

- **Line-multiset check per split commit** (non-blank, trimmed lines of the
  old file at the parent compared with the union of the new files at the
  commit; `package` / `import` / `export` / `part` lines ignored):
  - Pure move, nothing lost and nothing added: `models.dart` (515 significant
    lines), `block_chrome.dart` (329), `block_field.dart` (333),
    `roadmap_canvas.dart` (363), `auth_throttle_test.go` (312).
  - Moved with scaffolding, by design: `client.dart` loses only the class
    header (now a core class plus mixins) and gains doc comments; every verb
    body is untouched. The l10n pair loses nothing and gains only the barrel
    spreads and one-line comments, and a sorted JSON dump of all four maps was
    byte-identical before and after (369 entries per locale), with
    `l10n_test.dart` untouched. `project_page.dart` changes only the call
    sites that became callbacks (`onEdit(u, …)`, `onRemove(u)`, `setRepo`),
    which is how the extracted widgets reach the page's verbs.
  - The two test splits differ only by the harness rename and one extra
    `main()` wrapper; undoing exactly those edits in temp copies prints
    `PURE MOVE`. Test names are identical before and after.
- **The Flutter suite's test-name set was identical** before and after each
  split (167 tests at that point).
- **Behavior of the largest split, `project_page.dart`:** the Evidence write
  flows test ([`project_flows_test.dart`](../../../client/test/project_flows_test.dart),
  added after the split, 8 tests) passes unchanged on the tree from before
  the split (`78bf90a^`, run in a throwaway worktree) and on the tree after
  it.
- `flutter analyze` kept its baseline of 16 infos, `dart format` stayed clean,
  and CI's `client (flutter)` job is the final gate (see the plan for the run).

## Operator notes

None. No runtime behavior changed.
