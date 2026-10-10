# Verification — review-3 remediation (all waves)

## The adversarial contract

Every one of the 32 clusters passed this contract before its commits were
cherry-picked onto `main`:

1. **Red first** — the cluster's first commit per item is the failing test,
   proven failing on the wave base and quoted in the worker report.
2. **Independent adversarial verifier** — a second agent reproduced the red
   at the base, the green at the branch tip, re-ran the full gates in the
   cluster's worktree, and **mutation-checked** the fix (reverted or broke
   the fix and required the test to fail again).
3. **Repair loop** — a reject sent the cluster back with the verifier's
   blocking list; ≤3 rounds. **All 32 clusters were accepted in round 1.**
4. **Integrator gates** — after each wave's cherry-pick (`-x`, per-cluster
   ranges), the full gate set ran on the integrated tree before the next
   wave started.

Three rows the workers could not close inside their file ownership
(CQ-03, TEST-06, USER-08) were closed by the integrator across cluster
boundaries and re-gated. CQ-04 was parked by its brief's own pure-move gate
(the three providers' streaming bodies diverge; consolidation is a
behavior-risk refactor beyond a Low finding).

## Final gates (fresh database `rv3final`, integrated tree @ `4464330`)

The database was cloned from the migrated base **and replayed the tree's
migrations from scratch**, proving all seven pending migrations
(`20261010100000`…`20261011140000`) apply together on top of the deployed
shape:

| Gate | Result |
| --- | --- |
| `go test ./...` (server, 26 packages, real PostgreSQL) | **ok**, exactly one allowed skip (`TestReadSecretTerminal`) |
| `flutter test` (client) | **456 passed** (up from 421 at the review base) |
| `flutter analyze` | 14 infos — the repo baseline, no new findings |
| `dart format --set-exit-if-changed` | clean |
| CI (GitHub) | green on every integration push, including the new `server image (docker build)` and upgrade-replay lanes |
| Privacy scan over `git log -p 5c8f54f..HEAD` | only CI-local placeholder DSNs (`ci-app-password`, `postgres:postgres@localhost`); no provider keys, tokens, or real DSNs |

## Whole-change adversarial review

A final reviewer agent examined the integrated diff (`5c8f54f..4464330`)
across the six cross-cluster seams (prompts rewrite × locale variants, critic
merge × section validation, coach turn-bounds × stream classification,
stage-confirm × locale column, the coach panel's four simultaneous changes,
and the seven-migration chain), re-proved five red/fix pairs in detached
worktrees (AIDEV-01, FULL-01, REL-01, FEAT-01, FEAT-04), and checked the
invariants (file budgets, l10n en/zh parity, no new dependencies, one
allowed skip). Verdict: **accept** — no blocking items
(`/tmp/insideout-gap/rv3/whole-change-review.json`, 2026-10-11). One
non-blocking nuance it recorded: FEAT-01's en-prompt tests co-landed with
their fix rather than a separate red commit, so that half is proven
green-side only (the cluster's red covers the storage and client halves).
