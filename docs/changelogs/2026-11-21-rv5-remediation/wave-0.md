# Wave 0 · Green tip, the slice finished, the concept written

Rows 0.1 to 0.7 of the
[remediation plan](../../review/2026-11-21-focused-review-5/40-remediation-plan.md).
One worker finished the slice and the green-tip fixes in the main checkout, a
second wrote the task-tree concept, and an adversarial verifier re-ran every
gate and accepted in round 1. The coordinator applied the verifier's three
documentation fixes and one comment move.

## Rows

| Row | IDs | Disposition | Commit |
| --- | --- | --- | --- |
| 0.1 | IDX-02, COMP-01 | FIXED: `backend.md` names the four files; the expand-confirm fixture carries `completionReason`; the analyzer reports 8 infos and `INFO_BASELINE` is 8 | `50b7437` |
| 0.2 | IDX-04 | FIXED: `main` fast-forwarded to `origin/main` @ `c2900fd`; the branch `rv5/remediation` was cut from it with the uncommitted state carried over | branch base |
| 0.3 | IDX-01 | FIXED: the slice is covered by the [remediation plan](../../plans/2026-11-21-rv5-remediation/2026-11-21-rv5-remediation.md) and its board row | `226c1fb` |
| 0.4 | PARA-01, PARA-08, PARA-10, STRC-01, FE-06, IDX-09, COMP-04, COMP-06 | FIXED: the slice is finished and committed; detail in the [reset-watch record](../2026-10-21-reset-watch.md#wave-0-fixes-to-the-slice) | `454ab0b` |
| 0.5 | TREEB-11, TREEX-12, IDX-07 | FIXED: `PRODUCT.md` subsection and [roadmap-tree.md](../../architecture/roadmap-tree.md) | `99b4bf3` |
| 0.6 | COMP-03 and the four task-tree decisions | DECIDED by the owner, recorded in the [open-findings record](../../issues/2026-11-21-review-5-open-findings.md) | `226c1fb` |
| 0.7 | COMP-07 | FIXED: `.git/info/untracked-manifest.tsv` lists the untracked `.grok/` files; `HANDOFF.md` names only folders that exist | this record |

## Gates

Run by the verifier on the final tree, and the fast gates again by the
coordinator after the last edits:

- `go build ./... && go vet ./...`, `gofmt -l` empty.
- `go test ./... -count=1` without a database and on a freshly migrated
  private PostgreSQL 18 database: every package ok. The new purge test was
  shown to fail with the purge removed.
- `flutter analyze --no-fatal-infos`: 8 infos; `flutter test`: 581 passed.
- Every `scripts/tests/check_*.py` and `scripts/check-redaction.sh`: exit 0.
- The verify skill's selected tests: 497 tests in 127 files passed; it
  reports `INCOMPLETE` because the repository has no pregate policy and
  the skill cannot run the database-gated Go tests on its own.

## Left open

- `INFO_BASELINE` 8 was measured with the local Flutter 3.47; CI pins 3.44.
  The first CI run on the pull request settles it.
- CI's `dart format` step may reformat `client/lib/l10n/en_workspace.dart`
  with the pinned SDK; the file is unchanged by this wave.
- `PRODUCT.md` grew by 15 lines and stays over its line budget (tracked in
  [the line-budget record](../../issues/2026-10-18-product-and-setenv-over-line-budget.md)).
