# 2026-10-08 — `client/lib/l10n/{en,zh}.dart` over the ~350-line budget

`en.dart` is 414 lines and `zh.dart` 372: one flat `const` key→string
table each (`enUS`, `zhCN`) plus the coach-suggestion lists. They were
398/357 at HEAD and grew through the block-craft waves (Wave 0 l10n
pre-seed, Wave 6 adding `common.moreActions` and `admin.revokeConfirm`).
Every user-visible string in the app lands in these two tables, so they
only ever grow.

## Why not split now

The flat-table convention is load-bearing: `l10n_test.dart` asserts the
two catalogs share one key set, and `lookup()` reads `enUS`/`zhCN`
directly. A namespace split must keep both invariants and land alone,
not inside a verification gate.

## Target structure

Keep `en.dart`/`zh.dart` as barrels exporting one key set, composed
from namespace parts (each side must mirror the other exactly):

- `en_common.dart` / `zh_common.dart` — `common.*`, `state.*`, `nav.*`,
  `error.*`, `time.*`, `write.*`
- `en_auth.dart` / `zh_auth.dart` — `landing.*`, `login.*`, `register.*`,
  `forgot.*`, `reset.*`, `magic.*`, `admin.*`
- `en_product.dart` / `zh_product.dart` — `workspace.*`, `project.*`,
  `roadmap.*`, `github.*`, `idea.*`, `evidence.*`, `proposal.*`
- `en_prd.dart` / `zh_prd.dart` — `prd.*`, `versions.*`, `view.*`,
  `coach.*`, `profile.*`, `theme.*`, `lang.*`, `dock.*`, `home.*`

The tables reassemble with a spread (`{...enCommon, ...enAuth, ...}`) so
`lookup()` and the parity test keep working unchanged.

## Bounded fix prompt

"Split `client/lib/l10n/en.dart` (414 lines) and `zh.dart` (372) into
the namespace files above, reassembling `enUS`/`zhCN` by spread so
`lookup()` and every call site are untouched, per
[docs/issues/2026-10-08-l10n-tables-line-budget.md](2026-10-08-l10n-tables-line-budget.md).
Gate: `cd client && flutter analyze --no-fatal-infos && flutter test`;
`l10n_test.dart` (key-set parity, interpolation) passes unmodified.
Every file ends under ~350 lines. Record the split in a changelog entry
and close this issue."
