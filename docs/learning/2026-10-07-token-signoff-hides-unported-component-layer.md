# A signed-off token layer hides an unported component layer

## What was learned

A design-system port that lands as a **token layer first** can be
declared Finished — with green tests and a signed-off table — while
almost nothing on screen actually consumes it. Unit assertions over
constants prove the decree was *declared*, not that any surface
*obeys* it. The completion metric for a design system is not "do the
tokens exist and pass their tests"; it is "how many stock primitives
are still standing in the surfaces the tokens were meant to replace".
That number is only visible by grepping the feature tree, and no token
test will ever report it.

## Evidence

The [2026-10-06 Cobalt-Neo remediation](../plans/2026-10-06-frontend-cobalt-neo/2026-10-06-frontend-cobalt-neo.md)
ported the reference app's token layer and closed with a real gate:
`flutter analyze` no errors or warnings, `flutter test` 143 passed,
`dart format` clean, plus a WCAG contrast script and a six-shot visual
pass. [`client/test/cobalt_test.dart`](../../client/test/cobalt_test.dart)
asserts every radius token is `0.0`, `borderContainer` 2.0 /
`borderDivider` 1.0, `cast`/`castSm` offsets (4,4)/(2,2) with
`blurRadius` and `spreadRadius` both 0, `pressDuration` 100ms,
`hoverOffset` (2,2), `activeOffset` (4,4), and squared card, dialog,
snackbar, input, and chip shapes.

Auditing the same tree on 2026-10-07 for the follow-on component pass
([plan](../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md))
found the decree largely unconsumed:

- **~20 raw `showDialog` + `AlertDialog` sites**, no shared modal
  chrome: `dashboard_page.dart:41,79`, `prd_page.dart:104,149`,
  `project_page.dart:407,499,561,592`, `workspace_page.dart:74,335,390`,
  `ideas_page.dart:87,116`, `settings_page.dart:104`,
  `roadmap_page.dart:167`, `roadmap_node_card.dart:143,231`,
  `revisions_page.dart:42`, `versions_commit_dialog.dart:53`,
  `admin_section.dart:82`.
- `dashboard_page.dart:134` — a round `CircularProgressIndicator`,
  while DESIGN.md states "status dots and loaders are square blocks".
- `dashboard_page.dart:132` — `Center(child: Text('${snap.error}'))`,
  a raw unlocalized error dumped on the field, although
  `lib/api/errors_l10n.dart` exists for exactly this.
- `app.dart:106-139` — four bare `IconButton`s in a borderless
  `Material`: no 2px border, no cast shadow.
- `field_card.dart:42-46` — the approved card primitive itself wrapped
  `onTap` in an `InkWell` (Material ripple) rather than press-the-block.
- `cobalt_theme.dart:98-132` — the button themes carry the squared
  shape, `elevation: 0`, and `animationDuration: Cobalt.pressDuration`,
  but **no cast shadow and no translate**. `PressableBlock` existed and
  was wired into exactly one widget, the dock.

So the press-the-block motion was a tested token, a built wrapper, and
one call site. Every test passed; the most-seen interaction in the
product did not exist.

## Scope

Any convention ported in layers — design tokens, config schemas, shared
constants, error taxonomies — where the first layer is pure values and
a later layer is the components or call sites that use them. Also any
"signed off" or "Finished" claim whose evidence is unit assertions over
constants plus a visual pass over a handful of screens.

## When to apply it again

Close a token-layer plan with a **consumption census**, not only a
value census: grep the feature tree for the stock primitives the tokens
were meant to replace (here: raw `AlertDialog`, Material floating
`labelText`, `CircularProgressIndicator`, bare `IconButton`, `InkWell`
on a block) and record the counts in the plan. Non-zero means the
system is declared, not shipped — either finish the component layer in
the same plan or file the remainder explicitly, so "Finished" cannot
read as "done" to the next agent.

Then make the tests assert consumption as well as declaration. A scan
test that fails when a banned primitive reappears under `lib/features/`
catches this class of drift permanently; `client/test/leftover_scan_test.dart`
is the natural home for that kind of check. Value assertions and a
contrast script cannot substitute for it — they only ever see the
tokens.

Related: [Port a reference scheme structurally, not by approximation](2026-10-05-port-reference-schemes-structurally.md)
— the same failure mode one level down, where an approximation
contained every fact but not the shape.
