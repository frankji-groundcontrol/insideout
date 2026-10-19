# Wave 3, the web feed leaves applying a proposal to the server — review-4 remediation

Branch `rv4/w3-client-apply`, `8c74218..6348267` (two commits: one red/fix
pair), then this fragment. Findings are cited as `rv4-<ID>` because review IDs
restart per review. One row: `FEAT-02-CLIENT-APPLY`, the worker's id for the
register row rv4-FEAT-02 (plan rows 2.3 and 2.4, residue row R2-01 of the
[open-findings record](../../issues/2026-10-18-review-4-open-findings.md)). It
closes the `PARTIAL` facet that [w2-proposal-apply](w2-proposal-apply.md) left
open. No address, host, secret or one-time code is spelled in this fragment.

## What changed

- **A web accept no longer builds the roadmap nodes a second time**
  (`FEAT-02-CLIENT-APPLY`, red `4fd6027`, fix `6348267`). Since wave 2 the
  server applies an accepted proposal's items inside the decision's own
  transaction, once (`proposal_items.applied_at`). The updates feed in
  `client/lib/features/project/project_updates_feed.dart` still ran
  `_applyAcceptedItems` after `decideProposal`: it listed the roadmap and called
  `createRoadmapNode` for every `add_node` item, so a web accept made the
  server's nodes and then a second copy of each. Now `_decide` is one write,
  `decideProposal(update.id, accept: accept)` under `guardWrite`, followed by
  `onChanged()`. `_applyAcceptedItems` is deleted together with the
  `api/client.dart` import that only it used.
- **The feed still asks for the apply and still refreshes.** `decideProposal`
  defaults `apply` to `true`, so the request body carries `apply: true` and the
  server does the work. `onChanged` is the page's `_reload`, which re-reads the
  project and the roadmap, so the nodes the server just made appear without the
  client creating anything.
- **The test says so.** In `client/test/project_page_test.dart` the fake
  `ApiClient` now records each decision as `(id, accept, apply)` and counts
  roadmap reads, and 'accepting a proposal with one item adds that roadmap node'
  became 'accepting a proposal leaves applying its items to the server'. It
  expects exactly the decision `(up1, true, true)`, no node created by the
  client, and a roadmap read after the decision.

The wave-2 worker's patch (`worker-w2-proposal-apply-r0-client-apply.patch`) was
a lead, not applied: the change was re-derived and committed red first. It
differs in two ways. The test also asserts that the page re-reads the roadmap
after the decision, because that refresh replaces the removed code and the other
two assertions do not notice if it goes (mutation check A below). And
`projectId` stays on the widget (see the ceilings).

## Other callers and strings

- `createRoadmapNode` has two other callers in `client/lib`:
  `roadmap_node_menu.dart` (the add-child menu action) and `roadmap_page.dart`
  (the add-node action). Both run on a person's tap and neither follows a
  decision, so neither changed. The updates feed no longer calls it, and
  `decideProposal` has one caller, the feed.
- The removed code used no localized string of its own. `proposal.accept` and
  `proposal.reject` are still read by the two buttons and the
  `l10n_unread_keys_test` meta-test passes, so no key was orphaned and no table
  changed.
- `ProposalItem` and `ProjectUpdate.items` (`models_project.dart`) are now only
  parsed; no widget reads them. They stay (not in this cluster's files, and
  harmless).

## Evidence

| Check | Red at `4fd6027` (feed unchanged) | Green at `6348267` |
| --- | --- | --- |
| `flutter test test/project_page_test.dart` | `accepting a proposal leaves applying its items to the server` fails at `project_page_test.dart:202`: `Expected: empty`, `Actual: [({Null parentId, String projectId, String title}):(parentId: null, projectId: p1, title: Login throttling)]`, reason `the server creates the nodes, once`; the file ends `+4 -1: Some tests failed.` | `+5: All tests passed!` |
| the whole client suite | not run at the red commit | 483 passed; `TZ=Asia/Shanghai` 483 passed; `TZ=UTC` 483 passed |

Mutation checks on the fix (each undone afterwards, the tree is clean):

- **A, no refresh.** Delete the `await onChanged()` line from `_decide`. The new
  test goes red with `Expected: a value greater than <1>`, `Actual: <1>` and the
  reason `the page re-reads the roadmap after the decision`; the decision and
  no-node assertions stay green, which is why the read count is asserted.
- **B, wrong request.** Pass `apply: false` to `decideProposal`. The decision
  assertion goes red (the recorded list is not `[(up1, true, true)]`).
- **C, the old loop back.** The red commit is this check: with the feed as it
  was, the test fails on `api.created`.

## Live proof

A private PostgreSQL 18 cluster on a port of its own, migrated from this
worktree with the two roles (`insideout_owner`, `insideout_app`); the real
server binary built from this worktree, serving as the app role on a loopback
port; the built CLI under a scrubbed environment, logged in as an admin created
for the run. The proposal was filed through the real API (`insideout propose`
with one `add_node` item, "Login throttling"), accepted through the decision
route with the web client's body (`{"decision":"accepted","reason":"","apply":true}`),
and the roadmap listed through the real API. Output is in
`w3-client-apply-live-FEAT-02-CLIENT-APPLY.txt` in the reports directory.

| Step | Result |
| --- | --- |
| A. one accept with `apply: true`, what the fixed feed sends | `appliedNodes` lists 1 node; the roadmap has exactly 1 node, "Login throttling" |
| A. the same accept again (a double click, a retry) | `appliedNodes` is empty; the roadmap still has 1 node |
| B. the removed feed code replayed: accept, list the roadmap, create each item with `roadmap add` | the roadmap has 2 nodes, both "Login throttling" |
| C. database cross-check of both projects | A: 1 node and 1 item stamped applied; B: 2 nodes and 1 item stamped applied |

What this proves and what it does not. It proves the contract the fixed feed
relies on (one decision request with `apply: true` leaves exactly one node and a
repeat adds none) and that the removed behaviour doubles the roadmap. It does
not drive the web UI. **The web widget path is covered by the widget test with
the repository's fake `ApiClient`, not by a browser**: no Flutter web build was
opened in a browser in this cluster. The curl request and the CLI are other
clients of the same endpoint, not the feed.

## Gates

| Command | Result |
| --- | --- |
| `(cd client && flutter pub get --enforce-lockfile)` | exit 0; `client/pubspec.lock` unchanged |
| `flutter analyze --no-fatal-infos` | exit 0; 14 infos, all in `lib/api/*` (untouched), none in the files of this cluster |
| `flutter test`, `TZ=Asia/Shanghai flutter test` | 483 passed each; a `TZ=UTC` run (the machine zone is CST) also 483 |
| `dart format` on the two Dart files | 0 changed |
| 350-line gate on the changed `*.go` and `*.dart` files | nothing over (`project_updates_feed.dart` 162 lines, `project_page_test.dart` 279) |
| `scripts/tests/check_*.py`, `sh scripts/check-redaction.sh`, `sh scripts/check-docs-recorded.sh` | all exit 0 |

## Known ceilings, recorded on purpose

- **`ProjectUpdatesFeed.projectId` is unread.** Dropping it needs the
  `projectId: widget.id` argument in `project_page.dart`, which is outside this
  cluster's files, so the field stays with a doc comment saying why. The display
  test (`project_updates_display_test.dart`) still passes it and was not edited.
- **No "N nodes added" message.** The decision response's `appliedNodes` is not
  read; the page shows the result by re-reading the roadmap. A message would
  need a new key (for example `proposal.applied`) in the English and Chinese
  tables, and localization is outside this cluster's files.
- **A failed apply** is the server's 500 with no decision recorded (wave 2).
  `guardWrite` reports it as a failed write, `onChanged` is skipped, and the
  proposal stays open to accept again. Not changed here.

## Left for the integrator

No code is waiting. The records say `PARTIAL` for this row and should now say
`FIXED` (red `4fd6027`, fix `6348267`):

- `docs/changelogs/2026-10-18-rv4-remediation/wave-2.md`: the outcome list at
  lines 23 to 27 (`PARTIAL` 2 becomes 1, rv4-DATA-03 only), the
  `FEAT-02-CLIENT-APPLY` table row at line 66, and the paragraph "Why
  `FEAT-02-CLIENT-APPLY` stays open" at lines 80 to 85.
- `docs/issues/2026-10-18-review-4-open-findings.md`: the rv4-FEAT-02 row at
  line 36, the `FEAT-02-CLIENT-APPLY` row at line 148, and R2-01 at line 174.
- `docs/plans/2026-10-18-multirole-4.md` (the wave-2 bullet around line 42) and
  `docs/plans/2026-10-18-multirole-4.track.yaml` (the wave-2 step and the
  history): the web feed's re-apply is no longer open.
- `docs/architecture/product-subsystems.md` lines 50 to 53: after "always a list
  (empty on a repeat)." add that the web updates feed asks for the apply once
  and creates no node of its own.
- This folder's `index.md` (a row for this fragment and the wave-3 state),
  `docs/changelogs/README.md` and `docs/changelogs/index/2026-10.md`.
