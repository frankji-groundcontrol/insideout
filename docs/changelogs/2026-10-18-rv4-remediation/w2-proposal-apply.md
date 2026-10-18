# Wave 2, proposal apply once and the expand contract — review-4 remediation

Branch `rv4/w2-proposal-apply`, `4b2a1db..e0cf4cf` (eight commits: four red/fix
pairs), then this fragment. Findings are cited as `rv4-<ID>` because review IDs
restart per review. Rows 2.3 and 2.4 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md),
plus the carry-forward lead `LEAD-PROPOSE-VALIDATE` (it has its own report row).
This cluster is the dependency root of `w2-agent-scope`: both edit
`agent_routes.go` and `proposal_decisions.go`, and that cluster adds only the
human-only decision gate on top. One facet is `PARTIAL` for ownership (the web
client's own re-apply of an accepted proposal, see below). No address, host,
secret or one-time code is spelled in this fragment.

## What changed

- **A proposal's items are applied once, in one transaction** (rv4-FEAT-02,
  row 2.3, red `46f2b45`, fix `44a69b5`). Accepting with `apply` created every
  `add_node` item as a roadmap node and remembered nothing, so a second accept
  (a double click, a retry, a second tab, the CLI after the web) built every
  node again, N racing accepts built N copies, and the handler swallowed an
  apply error: a failed apply answered 200 with the decision recorded and a
  half-built plan. Now:
  - migration `20261018120000_proposal_items_applied_at.sql` adds
    `proposal_items.applied_at` (NULL = not applied, which is every row that
    existed before), narrows the app role's UPDATE on the table to that one
    column, and adds an UPDATE policy for the project owner or a workspace
    admin;
  - the apply claims an item with `UPDATE ... SET applied_at = now() WHERE id =
    $1 AND applied_at IS NULL` in the transaction that creates its node, and a
    claim that matches no row means a racing accept took it, so it is skipped;
  - the decision and the apply share one transaction
    (`DecideAndApplyProposal`). An apply that fails rolls the decision back and
    the route answers 500 `internal error`, so the proposal stays open to accept
    again. `appliedNodes` lists only the nodes this call created and is `[]` for
    a repeat. The parent hint is resolved inside the transaction, with the
    roadmap list's own ordering, so nodes created earlier in the same apply can
    be a parent exactly as before.
- **The Decision Log doc no longer says the row is immutable** (rv4-FEAT-09,
  row 2.3, red `c667c14`, fix `aa51d5f`). `ProposalDecision` called the entry
  "immutable; re-deciding appends", but `DecideProposal` upserts: the row holds
  the latest decision and the project timeline keeps one `[proposal accepted]`
  or `[proposal rejected]` note per decide. The comment now says that, and the
  red test pins the behaviour (accept then reject leaves `rejected` and two
  notes) as well as the wording.
- **A refused propose writes nothing** (rv4-LEAD-PROPOSE-VALIDATE, red
  `53185f7`, fix `85ef49d`). `handleAgentPropose` filed the `agent_proposal` row
  first and checked the items afterwards, so a 400 (blank title, title over 200
  characters, an action other than `add_node`, an eleventh item) left an orphan
  proposal in the timeline that a human could still accept. Every item is now
  validated before the first write, and the proposal and its items are filed by
  one `AddProposal` transaction, so a database refusal part-way leaves neither.
- **Web "Break down with AI" works** (rv4-FULL-01, row 2.4, red `c9c3142`, fix
  `e0cf4cf`). The server half was already right: `POST /roadmap/{id}/expand`
  drafts and writes nothing (200 `{"children":[{title, description}]}`), and
  only `POST /roadmap/{id}/expand/confirm` creates the nodes (201, a JSON list
  of roadmap nodes). The client cast the draft map to a `List`
  (`type '_Map<String, dynamic>' is not a subtype of type 'List<dynamic>'`),
  never called the confirm route, and the generic write-failed snackbar hid it.
  Now `expandRoadmapNode` reads `children` into `DraftSubtask`s, the new
  `confirmExpandRoadmapNode` posts the draft and parses the 201 list, and the
  new `roadmap_expand_draft.dart` shows the draft in a review dialog (each
  title and description, "Add N sub-tasks" or Cancel), confirms only on the
  button, then reloads. An empty draft says so instead of offering a confirm.
  The client creates no node of its own in this flow, so the web cannot
  double-apply a breakdown. Three keys (`roadmap.expandReview`,
  `roadmap.expandConfirm`, `roadmap.expandEmpty`) were added to each of the
  English and Chinese tables.

## The proposal-apply design (row 2.3)

Row 2.3 is not marked `dir`; this records why the obvious shapes lost.

- **Claim, then create, in one transaction (chosen).** Stamping the item first
  and creating the node second means a rollback undoes both, and a racing
  transaction blocks on the item's row lock, then re-checks `applied_at IS
  NULL` against the committed row (READ COMMITTED re-evaluates the `WHERE`
  after the lock is released), finds it stamped and skips it. Claiming before
  creating also fixes the lock order: a racer waits on the item before it can
  take the per-project advisory lock `createRoadmapNodeTx` uses for the
  position counter, so two accepts cannot deadlock on the two locks.
- **Create the nodes, then stamp the items.** Two racers both read the items as
  unapplied, both create, and the second stamp matches nothing too late.
- **An in-process mutex or a `FOR UPDATE` only.** A mutex protects one process
  and the database is where every replica meets. `FOR UPDATE` on the proposal
  row (which `DecideAndApplyProposal` also takes) serialises the decide path,
  but the store-level `ApplyProposalItems` does not go through it, so the claim
  is the guard every caller shares. Three guards overlap on the decide path
  (the proposal row lock, the decision upsert's conflict, the claim); the
  mutation checks record what was observed with them removed.
- **Authority before the claim.** The claim is an RLS-filtered `UPDATE`; for a
  member who may not apply it would match no row and read as "already applied".
  `applyProposalItemsTx` therefore runs `requirePlanWriter` first and a plain
  member gets 403, not a silent empty success.
- **Decide and apply atomic.** If they were two transactions a failed apply
  would leave a decision with nothing built, and a retry would find the proposal
  already decided. One transaction makes the failure a 500 with nothing changed.

### The grant observation

Migration `20260819190000` set default privileges that already gave the app role
a table-level UPDATE and DELETE on `proposal_items`; only the missing policies
stopped them. This migration revokes the table-level UPDATE and grants UPDATE on
`applied_at` alone, plus the one UPDATE policy, so an item stays immutable apart
from its stamp. The REVOKE comes first on purpose: a table-level REVOKE also
drops column grants, so the pair is re-runnable and ends in one state (the SQL
was executed three times as the owner with identical results). The table-level
DELETE is still granted with no DELETE policy, so RLS denies it; that belongs to
`w2-rls` and is listed for the integrator.

## The expand contract (row 2.4)

Both sides read the same two files, captured from a running server:
`server/internal/api/testdata/roadmap_expand_draft.json` and
`roadmap_expand_confirm.json`. The Go test `TestExpandDraftAndConfirmKeepTheirCapturedShapes`
calls the real routes over real PostgreSQL and compares the response structure
(keys and kinds of every leaf, never values, which come from the planner) with
the fixtures; the Dart tests feed the real client those same bytes through a
fixture HTTP adapter. If either side changes the shape, one of the two suites
goes red.

## The `PARTIAL` facet: the web feed re-applies an accepted proposal

Plan row 2.4 says to remove the client-side apply so the web cannot
double-apply. The expand flow is done. The other client-side apply is in
`client/lib/features/project/project_updates_feed.dart`: after
`decideProposal` it re-creates every `add_node` item through `listRoadmap` and
`createRoadmapNode` (the review's R10 row cites it at lines 44-45). With the
server now applying once, that loop doubles every node on a web accept. The file
and its test `client/test/project_page_test.dart` are outside this cluster's
files, so they are untouched here and the facet is reported `PARTIAL`.

The staged patch is `worker-w2-proposal-apply-r0-client-apply.patch` in the
worker's reports directory. It removes `_applyAcceptedItems` and its call
(`_decide` becomes one `decideProposal` write), drops the `api/client.dart`
import that only it used, and turns the test 'accepting a proposal with one
item adds that roadmap node' into 'accepting a proposal leaves applying its
items to the server': the fake records `(id, accept, apply)` and the test
expects `[(up1, true, true)]` and no client-side node. Verified in a scratch
copy: the changed test is red against the old feed and green with the patch,
the whole client suite passes (474), `dart format` and the analyzer are clean,
`git apply --check` passes on this branch and on the integration tree. Until it
lands the web accept shows the doubled nodes.

## Known ceilings, recorded on purpose

- **`ExpandRoadmapNode` (`roadmap_plan.go`, not in this cluster's files) takes
  no per-project lock** around `MAX(position) + 1`, which `createRoadmapNodeTx`
  does. Two concurrent confirms under one parent can get the same position;
  siblings with equal positions are still ordered by `created_at, id`.
- **`ExpandRoadmapNode` and the plan build check workspace membership only**,
  while `requirePlanWriter` (owner or admin) gates every other roadmap write
  (BUG-020). Read from the code, not exercised live, and not part of any review
  finding; it may be a deliberate choice for the AI expand, so it is recorded
  rather than changed.
- **`ApplyProposalItems` and `ApplyProposalItemsForUpdate` have no production
  caller now** (the route uses `DecideAndApplyProposal`). They stay as the
  store-level seam the claim tests drive.
- **Older migrations still say the decision row is immutable** (historical
  comments; migrations are not edited after the fact).
- **A fallback-served draft is the planner's built-in template.** With no AI key
  configured the draft is the offline template, which the live proof used.

## Left for the integrator

No code is waiting except the patch above.

- Apply the `PARTIAL` patch (pre-gate: a web accept double-applies until it
  lands, and no current test notices).
- `docs/architecture/database-and-rls.md` (owned by `w2-rls`): one sentence that
  `proposal_items` keeps a table-level DELETE from the default privileges with
  no DELETE policy, and that UPDATE is narrowed to `applied_at`.
- `docs/architecture/product-subsystems.md` lines 38 to 43: say that `apply:
  true` runs in the decision's own transaction, once (`applied_at`), and that a
  failed apply is a 500 with no decision recorded.
- `docs/issues/2026-10-11-product-branch-merge-decision-log.md` line 29 cites
  `proposal_decisions.go:13`; the type now starts at line 17.
- The open-findings record: per-ID dispositions for rv4-FEAT-02, rv4-FEAT-09,
  rv4-FULL-01 (the expand flow `FIXED`, the feed re-apply `PARTIAL`) and
  rv4-LEAD-PROPOSE-VALIDATE; this folder's `index.md` wave-2 row; the changelog
  index files.

**Closed (integrator follow-up, wave 2).** The two architecture sentences
(`docs/architecture/database-and-rls.md`, `docs/architecture/product-subsystems.md`)
and the stale line citation are done, and the dispositions are in the
[open-findings record](../../issues/2026-10-18-review-4-open-findings.md#wave-2-open-residue-integrator-2026-10-19).
Still open there: the web feed's own re-apply, the `PARTIAL` patch above (R2-01,
`w3-client-apply`), the three store functions with no production caller (R2-14)
and the expand route's lock and gate (R2-04, a final-review lead).

## Verification

Run against a private PostgreSQL 18 cluster on a port of its own, migrated from
this worktree, with role passwords that exist only for the run, under a scrubbed
environment. A skipped database test counts as red; none skipped.

| Finding | Red, at the red commit | Green, at the fix commit |
| --- | --- | --- |
| rv4-FEAT-02 | store: second apply "created 3 nodes, want 0"; 5 concurrent applies "created 15 nodes in total, want 3"; a failing item 2 left `["Intake"]` instead of `[]`. API: a second accept applied 2 nodes; 4 racing accepts applied 4 nodes, want 1; an accept while the database refused item 2 answered 200 with the decision recorded, want 500 | all six pass under `-race`; the two concurrency tests passed 30 of 30 runs. Guards that came with the fix: a member who may not apply is refused (`ErrForbidden`), a workspace admin who does not own the project applies the proposal and the owner's later accept applies nothing, and through the app role only the plan writer can stamp `applied_at` while no other column of an item changes |
| rv4-FEAT-09 | "proposal_decisions.go still calls the decision row immutable" | passes: accept then reject leaves `rejected` and two timeline notes, and the comment says "latest" |
| rv4-LEAD-PROPOSE-VALIDATE | "blank title: the 400 left 1 agent_proposal row(s) in the project timeline, want none" | passes for all five refusals (blank title, 201 characters, unknown action, a bad item after a good one, eleven items); `TestAddProposal_AnItemTheDatabaseRefusesLeavesNoProposal` is a green-only atomicity guard |
| rv4-FULL-01 | the contract test fails with the `_Map` is not a `List` cast error at `client_project.dart:231`; the flow tests find no draft text, no empty-draft message and no zh-CN confirm label; the confirm tests do not compile (`confirmExpandRoadmapNode` not found). The Go shape test is green at the red commit by design: the server was right, and it keeps the fixtures honest | eight expand tests pass; the 19 tests across the five touched Dart test files pass |

Mutation checks a verifier can run; each was undone afterwards:

- rv4-FEAT-02: drop `AND applied_at IS NULL` from the claim. The store test
  `TestProposalApply_ConcurrentAppliesCreateEachNodeOnce` goes red; the API
  tests stay green because the decision upsert and the proposal row lock
  serialise accepts first. Removing only the proposal row's `FOR UPDATE` keeps
  everything green (three repeats under `-race`), and removing the row lock and
  the claim guard together still leaves the API accept tests green, because the
  decision upsert's conflict serialises the accepts. That is why the store-level
  test is the one that pins the claim for every caller that skips the decide
  path.
- rv4-FEAT-02: drop the UPDATE policy (`DROP POLICY item_applier_update`) and ten
  tests go red, six in the store and four in the API, among them the plain
  owner accept: the claim matches no row, so nothing is applied. In a live
  server that would be a quiet `appliedNodes: []`, which is why the policy lives
  in the same migration file as the column and the grant, and why the tests
  listed above drive the owner's own accept.
- rv4-FEAT-02: re-create that policy with the owner branch only (the
  `_is_admin` arm dropped). Across the whole store and API suites exactly one
  test goes red,
  `TestProposalApply_ANonOwnerWorkspaceAdminAppliesOnceAcrossActors`
  ("admin accept: 0 nodes, <nil>, want both items applied"): a workspace admin
  passes the authority check, the claim matches no row for them, and the accept
  answers 200 with nothing built. Every other apply test accepts as the project
  owner, so without this guard that edit would pass unnoticed.
- rv4-FULL-01: make the dialog skip the confirm call and
  'the draft is shown, and confirming adds exactly the draft' goes red
  ("Expected: expand, expand/confirm; Actual: expand").

Live proof on the real binaries: two server processes on loopback ports against
one private cluster, 20 checks, driven by a script that logs in as a throwaway
admin and never prints a secret. The binary built from this branch passes 20 of
20; the base binary passes 9 of 20 and shows each defect:

| Check | Base binary | This branch |
| --- | --- | --- |
| three refused proposes (400) | left 1, 2, then 3 orphan proposals | left none |
| 8 concurrent accepts of a 3-item proposal, split across both processes | 24 nodes created (8 copies of each) | 3 nodes, each item once; every response 200 |
| a sequential repeat accept | applied 3 more nodes | `appliedNodes` is `[]` |
| a database refusal during the apply (a temporary CHECK on the node title) | 200, decision recorded, one node created | 500 `internal error`, no decision, no node, no item stamped; after the fault is lifted the retry applies both items once |
| expand then confirm | draft writes nothing, confirm adds the children | same, and both response shapes equal the committed fixtures |

Gates, all green on this branch: `gofmt -l` prints nothing; `go build ./...` and
`go vet ./...`; `go test ./... -race -count=1` (27 packages ok, 3 without test
files, 0 failed); `go test ./internal/store ./internal/api -race -count=1 -v`
(230 top-level tests passed, 0 failed, 0 skipped);
`flutter pub get --enforce-lockfile` with `client/pubspec.lock` byte-identical;
`dart format --output=none --set-exit-if-changed lib test` (269 files, 0
changed); `flutter analyze --no-fatal-infos` (exit 0; the same 14 infos as the
base, none in these files); `flutter test` (474 passed) and the same suite under
`TZ=Asia/Shanghai` (474 passed); every touched file at most 350 lines. The
migration SQL was re-executed as the owner three times (the runner's second
`migrate` reports "nothing to do") with the same resulting column, grant and
policy.
