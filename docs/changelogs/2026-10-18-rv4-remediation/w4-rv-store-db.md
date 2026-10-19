# Wave 4, membership RLS, last admin, applied_at backfill and re-decision — review-4 remediation

Branch `rv4/w4-rv-store-db`, `582a4bd4..1f200493` (nine commits: five red, four
fix), then this fragment. Cluster `w4-rv-store-db` of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md),
under its [execution contract](../../review/2026-10-18-multirole-4/41-remediation-contract.md):
the store and database findings of the whole-change review. They are cited as
`rv4-FR-<ID>` because review IDs restart per review. Hashes are the worker
branch's; the integrator's cherry-picks carry their own, with a `-x` trailer.
No address, host, secret or one-time code is spelled in this fragment.

**Migrations: FORWARD, no existing migration edited.** Two new files:
`20261018220000_membership_rls_hardening.sql` and
`20261018221000_proposal_items_applied_backfill.sql`. The files they correct
(`20261018140000`, `20261018120000`) already ran wherever the earlier rv4 build
was applied, and the runner records file names, not checksums, so an edited file
would never run again there. Both new files run as `insideout_owner` (the
runner's role guard), are idempotent, and add no foreign key.

## What changed

### 1. A membership row cannot be re-pointed, and the creator is a fact (rv4-FR-F37, F22, F34, F23, F33)

One root, two doors. `20261018140000` made a membership INSERT need a reason
(the join code for a member, the workspace creator for an admin) but left two
ways to reach the same state without one.

- **UPDATE (F37, F22, F34).** `workspace_memberships_update` is an admin-only
  `USING` policy with no column limit and `insideout_app` holds a table-wide
  UPDATE, so an admin could rewrite `user_id` (or `workspace_id`) of an existing
  row and put any user into the workspace with no code and no consent, or
  promote them in the same statement. Red, on real PostgreSQL as an admin:
  re-point at a user who never joined, re-point and promote in one statement,
  move a row into another workspace the admin administers, rewrite the row id,
  rewrite `created_at`. All five answered `want a refusal (SQLSTATE 42501), got
  <nil>`.
- **The creator (F23, F33).** The admin branch of `_may_insert_membership`
  trusted `workspaces.creator_id` for the life of the workspace. That column
  outlives the creator's membership, and `workspaces_update` let any admin
  rewrite it. Red: a creator removed by the other admin, and one who left,
  inserted themselves back as admin with no join code; an admin who was not the
  creator named themselves creator; the creator handed the role to another
  admin; and the three-step chain (rewrite `creator_id`, get removed, insert back
  as admin) worked.

Fix, migration `20261018220000_membership_rls_hardening.sql`:

1. `_pin_membership_columns` and the `workspace_memberships_pin_columns` BEFORE
   UPDATE trigger: everything but `role` and `updated_at` (the `set_updated_at`
   trigger writes it) must come out unchanged, else SQLSTATE 42501. It compares
   `to_jsonb(NEW)` with `to_jsonb(OLD)` minus those two keys, so a column added
   later is pinned by default. A role RLS does not apply to (`insideout_owner`, a
   superuser) is trusted with every column, the line RLS itself draws. A trigger
   and not a column grant, for the reason `20261018140000` records for `users`: a
   grant is checked before the statement reads a row, so an UPDATE that matches
   no visible row would turn into "permission denied".
2. `_pin_workspace_creator` and the `workspaces_pin_creator` trigger (BEFORE
   UPDATE OF `creator_id`): `creator_id` cannot change. No flow changes it
   (`DeleteWorkspace` and the admin branch only read it); an UPDATE that does not
   name the column never fires the trigger, and writing the same value back
   passes.
3. `_may_insert_membership`, same signature and attributes so the policy keeps
   pointing at it: the admin branch also needs the workspace to have no member
   rows yet. That is the state `createWorkspaceTx` inserts into, in the
   transaction that creates the workspace. A workspace that has people gets new
   admins by promotion (`UpdateMemberRole`), never by insert. The member (join
   code) branch is unchanged. A branch for "a current admin may insert an admin"
   would be dead code, since promotion is an UPDATE, so it is absent.

No Go change for this root: `UpdateMemberRole` writes `role` and nothing else.

### 2. The last admin cannot step down or leave (rv4-FR-F134, F141, F146)

Hypothesis: `UpdateMemberRole` and `RemoveMember` have no last-admin guard, so
the only admin can orphan a workspace through the ordinary members API, and a
workspace with no admin has nobody left who can promote. Confirmed. Red, store:
the only admin demoting themselves returned nil and the roster showed 0 admins;
the only member, an admin, leaving succeeded; two admins demoting each other
concurrently both went through (each transaction read "two admins") and left 0.
Red, router: `PATCH` of the sole admin to `member` answered `200 {"ok":true}`.

Fix, no migration:

- `store.ErrLastAdmin`, an `ErrConflict` that carries "a workspace needs at least
  one admin". `handleUpdateMemberRole` and `handleRemoveMember` map it to `409`
  with that message.
- Both store methods take `pg_advisory_xact_lock(hashtext('insideout.workspace_admins'),
  hashtext(<workspace id>))` first, then check the actor, then call
  `requireAnotherAdmin` (one query: is the target an admin, and how many other
  admins are there) and refuse when the target is the sole admin and the
  statement would take the role (demotion to `member`) or the membership
  (removal, self-leave) away. Promotion to `admin` is not guarded. A plain member
  leaving, and an admin leaving or being demoted while another admin remains,
  behave as before.
- An advisory lock rather than `FOR UPDATE` on the admin rows: row locks under
  FORCE RLS silently return zero rows when a policy calls another table
  (BUG-007). The lock comes before the actor check so a race loser re-reads who
  is still an admin. The key is per workspace, so unrelated workspaces never
  wait; a 32-bit hash collision only adds waiting, never a wrong answer.
- Workspace deletion removes memberships through the foreign key cascade, not
  through `RemoveMember`, and there is no account-deletion flow, so nothing else
  needed the guard.

### 3. Proposal items applied before `applied_at` existed are stamped (rv4-FR-F38, F25)

Hypothesis: `20261018120000` added `proposal_items.applied_at` with no data step,
its comment reads "NULL = not applied yet (every pre-migration row)", and the
first accept with apply after the upgrade builds every historic item again.
Confirmed in the shape that matters: a proposal accepted with apply by the old
build, its stamps wiped, then accepted again, builds its nodes a second time.
The first red commit (`963556fd`) fails on the missing upgrade step; mutation
checking then showed its test too weak, and a second red commit (`def92ff3`)
added the "stamped twin" case. Both are kept (no history rewrite).

Fix, migration `20261018221000_proposal_items_applied_backfill.sql`. An item is
stamped only when its node demonstrably exists:

- its proposal carries an accepted decision (nothing in a pending or a rejected
  proposal was applied by that proposal), and
- its project holds a node whose description is `'proposed: ' || title`, the text
  the apply wrote on every node it built (a node made by hand has a description
  of its own), and
- each marked node is credited to one item. The items of a project that share a
  title are ranked, the already stamped first (they have used their node), then
  by decision time, position, `created_at`, id; only the first N are stamped, N
  being the number of marked nodes with that description. Two proposals with one
  title and one node get one stamp, never two.

The stamp is the decision time, the nearest honest instant there is. A re-run
changes nothing: a stamped item keeps its rank and nothing new qualifies.

Deliberately not done: stamping every item of an accepted proposal (it would
also skip the items an owner accepted without apply, a worse mistake than one
repeated node) and matching on titles alone.

The test, `TestRvStoreDbBackfill_AppliedBeforeTheColumnReadsAsApplied`, makes the
old shape the way it came to be (a real accept with apply, then the stamps
wiped as the owner) and runs the migration's own file text from the embedded
`db.Migrations`, so a copy cannot drift from the file. It spans three projects
and six proposals: an applied proposal (3 of 3 stamped) beside a rejected and a
later-accepted proposal with the same title (0 stamped each, the one marked node
is credited once); a proposal accepted without apply beside a hand-made node and
a stamped twin (0 of 3); a proposal the new build applied (kept, 1 of 1); and
another project's applied proposal with a title the waiting one also uses (1 of
1, and its mark must not credit the other project). It then asserts the
decision-time value, that a second run moves nothing, and the point of it:
re-accepting builds 0 nodes for the upgraded proposals, 1 for the late one, 3
for the waiting one.

### 4. Deciding what is already decided is not an event (server half of rv4-FR-F02, F08, F15)

Hypothesis: `decideProposal` upserted the decision and appended a timeline note
on every call, so a double tap (a retry, a second tab, the CLI after the web)
duplicated the public `[proposal accepted]` note and rewrote `decided_at`,
although the nodes are applied once. Confirmed. Red, store: a repeated accept
left 2 notes and moved the record; four taps at once left 4 notes; "accept again
to apply what is left" appended a note. Red, router: two accepts left 2 notes.

Fix, `recordDecisionTx` in `proposal_decisions.go`: it reads the stored decision
inside the transaction that already holds the proposal row's lock; when it equals
the requested decision and reason, nothing is written and no note is appended.
The apply step still runs, so accepting again still builds whatever is left (an
accept without apply, then an accept with it, works as before). A change of
decision or of reason is an event: the upsert and one note, exactly as before.
The `ProposalDecision` and `DecideProposal` comments say so.

The client half (an in-flight guard in `project_updates_feed.dart`, with a
double-tap widget test) is the client cluster's; no Dart was touched here.

**Owner question, the opposite tap.** F02, F08 and F15 also name accept-then-reject.
Today the record flips to `rejected` (one more note, as `w2-proposal-apply`
documented and pinned) while the nodes the accept built stay. This cluster does
not choose. Options: (a) refuse a reversal with `409` once any item of the
proposal has been applied; (b) keep it as it is, the record says what the owner
last decided and the plan keeps what was built; (c) allow it and un-apply, which
deletes the nodes that accept created and clears `applied_at`, destructive if
they were edited meanwhile. Question: which one? Recommendation only: (a), one
condition in `decideProposal` and a `409` mapping in the handler.

## Evidence

All on a private PostgreSQL 18 cluster (roles `insideout_owner` with BYPASSRLS
and `insideout_app` without), `DATABASE_URL`-gated tests, no mocks.

| Root | Red (worker commit) | The red output, abridged | Green |
| --- | --- | --- | --- |
| 1 | `419a24ea` | 5 refusals missing in `CannotRepointARow`; 2 sub-cases of `RemovedCreatorCannotReAdmin`; 4 steps of `AdminCannotBecomeCreator`, each `got <nil>` | `70c5ea08` |
| 2 | `cb423f00` | `the only admin demoting themselves: want a conflict ... got <nil>`, `0 admins after the refused demotion, want 1`; `both demotions went through, the workspace has 0 admins`; router `status 200, want 409` | `5ccb6cf5` |
| 3 | `963556fd`, `def92ff3` | `no upgrade step stamps the items applied before applied_at existed: open migrations/20261018221000_...: file does not exist` | `3eea8590` |
| 4 | `1eec82ac` | `a repeated accept left 2 decision notes, want 1`; `4 taps at once left 4 decision notes, want 1`; router `two taps on Accept left 2 decision notes` | `1f200493` |

The red of root 3 is a missing file; its behaviour is pinned by the mutation
checks below, not by the red alone.

Mutation checks (each change made on the fix, the named test run, then undone):

- Root 1: each of the three migration pieces removed in turn (the column pin, the
  creator pin, the first-member rule); the matching test went red each time.
- Root 2: no advisory lock, the two-admins race went red 3 runs of 3; no guard,
  the self-demote, leave and router tests went red.
- Root 3, seven variants of the SQL: M1 ignore the decision; M2 drop the
  one-to-one credit; M3 match nodes across projects; M4 match plain titles
  instead of the mark; M5 blanket stamp; M6 unstamped first in the ranking; M7
  stamp `now()` instead of the decision time. All seven red.
- Root 4: always write the note; compare the decision but not the reason. Both
  red.

## Row dispositions

| ID | Status | Red | Fix | Note |
| --- | --- | --- | --- | --- |
| rv4-FR-F37 | `FIXED` | `419a24ea` | `70c5ea08` | UPDATE of a membership row is pinned to `role` |
| rv4-FR-F22 | `FIXED` | `419a24ea` | `70c5ea08` | same root as F37 |
| rv4-FR-F34 | `FIXED` | `419a24ea` | `70c5ea08` | same root as F37 |
| rv4-FR-F23 | `FIXED` | `419a24ea` | `70c5ea08` | admin insert only for the first member |
| rv4-FR-F33 | `FIXED` | `419a24ea` | `70c5ea08` | `creator_id` immutable; same root as F23 |
| rv4-FR-F134 | `FIXED` | `cb423f00` | `5ccb6cf5` | `409`, advisory lock |
| rv4-FR-F141 | `FIXED` | `cb423f00` | `5ccb6cf5` | same root as F134 |
| rv4-FR-F146 | `FIXED` | `cb423f00` | `5ccb6cf5` | the issue-book entry it asks for is an INTEGRATOR TODO below |
| rv4-FR-F38 | `FIXED` | `963556fd`, `def92ff3` | `3eea8590` | conservative backfill, residue below |
| rv4-FR-F25 | `FIXED` | `963556fd`, `def92ff3` | `3eea8590` | replay on the upgrade seed not taken: INTEGRATOR TODO |
| rv4-FR-F02 | `PARTIAL` | `1eec82ac` | `1f200493` | server half fixed; client guard and the owner question remain |
| rv4-FR-F08 | `PARTIAL` | `1eec82ac` | `1f200493` | same |
| rv4-FR-F15 | `PARTIAL` | `1eec82ac` | `1f200493` | same |

## Known ceilings, recorded on purpose

- **The backfill cannot see everything.** An item whose node left no mark stays
  NULL, and a repeat accept builds that node once more, as it did before this
  file: nodes the old web feed created itself after an accept (it wrote no
  description, so they look like hand-made nodes) and nodes deleted or whose
  description was edited since. The stamp is the decision time, not the apply
  time, which was never recorded.
- **A creator who left keeps workspace rights (observation, not changed).**
  `workspaces_update` names `creator_id` as well as admins and `workspaces_delete`
  names only `creator_id`, so a creator who left or was removed can still update
  or delete the workspace. The pins stop `creator_id` being rewritten and stop the
  creator re-entering as admin; whether leaving should also give up those rights
  is a policy decision. Fix hook: have both policies require a current membership
  for the creator.
- **Who may remove the last admin is now decided in code.** The zero-admin
  refusal was treated as decision-free (a workspace nobody can administer helps
  no one). If the owner wants a sole admin to be able to leave and take the
  workspace with them, `requireAnotherAdmin` is the one place to change; the
  project hand-over half of R3-43 is `w4-project-handover-403`'s.
- **No new case in `TestRLSWriteWalk`.** F22 suggests one; that file is not this
  cluster's. The dedicated tests in `rv4_review_store_db_test.go` cover the same
  statements.

## Gates

Private cluster, `env -i` plus only the database and JWT variables the tests
read.

- `go build ./... && go vet ./...`: clean. `gofmt -l internal cmd`: empty.
- `go test ./internal/store/... ./internal/api/... -race -count=1 -v`: both
  packages `ok`; 0 `FAIL`, 0 `SKIP`, no data race; the 12 new tests ran and passed.
- Whole server suite, once, last (`go test ./... -race -count=1 -v` on a freshly
  migrated database): every package `ok`; 0 `FAIL`, no data race, no panic. The
  one `SKIP` is `TestReadSecretTerminal`, which needs a pty; it is not a
  database-gated test.
- `scripts/tests/check_*.py` (14): all pass. `scripts/check-redaction.sh`: clean.
  `scripts/check-docs-recorded.sh`: clean. No `.go` file in the diff is over 350
  lines.
- `server/scripts/upgrade-replay.sh` on an emptied database: `UPGRADE REPLAY
  GREEN: 31 deployed files replayed, 15 pending migrations met the seeded data`,
  both new migrations among the 15.
- One-off, not committed (the seed is not this cluster's file): the same replay
  with an accepted-and-applied proposal and one accepted without apply seeded at
  the deployed schema, where `applied_at` does not exist yet. After the pending
  set, exactly the two items with a marked node were stamped, at the decision
  time, and the other stayed NULL.

## Integrator's to-do

INTEGRATOR TODO: `docs/changelogs/2026-10-18-rv4-remediation/index.md`: add the
`w4-rv-store-db` row to the cluster-fragments table (rows: the whole-change
findings F134, F141, F146, F22, F23, F33, F37, F38, F25, F34, and the server half
of F02, F08, F15) and a wave-4 row to the wave table.

INTEGRATOR TODO: `docs/changelogs/README.md` and `docs/changelogs/index/2026-10.md`:
the index lines for this fragment.

INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md` (R3-43 at
line 168, rv4-TEST-03 at line 299) and `docs/issues/2026-10-18-review-4-dispositions.md`
(rv4-TEST-03 at line 171): the last-admin half of R3-43 is closed here (`409`);
only the project hand-over half remains, and `w4-project-handover-403` owns it.
Decide whether "may the last admin be removed or demoted?" still stands as an
owner question (see the ceilings).

INTEGRATOR TODO: `docs/changelogs/2026-10-18-rv4-remediation/w3-test-hardening.md`
(the "Observation only" bullet at line 208, and the issue-book to-do at line 289)
and `wave-3-dispositions.md` (line 79): point the missing last-admin guard at this
fragment as closed.

INTEGRATOR TODO: `docs/issues/README.md`: file the dated entries this fragment
leaves open: (1) F02/F08/F15 remainder, the client in-flight guard and the
owner's reversal question; (2) the creator who left keeps update and delete
rights on the workspace; (3) the backfill residue (old-web-feed and deleted
nodes). F146 asks for the entry; the index line is the integrator's.

INTEGRATOR TODO: `docs/plans/2026-07-20-go-rewrite/02-backend-go.md` (member
routes): `PATCH` and `DELETE /workspaces/{id}/members/{userId}` answer `409` with
`a workspace needs at least one admin` when the target is the sole admin.

INTEGRATOR TODO: `server/scripts/upgrade-seed.sql` and `upgrade-replay.sh`: F25
asks for a replay test on the upgrade seed. Seed, at the deployed schema, one
`agent_proposal` update with two `add_node` items, an `accepted` decision, and two
roadmap nodes described `proposed: <title>`; plus a second accepted proposal with
no node. Assert after the pending set that exactly two items have
`applied_at IS NOT NULL`, equal to the decision time. `proposal_items` and
`proposal_decisions` both exist at the cutoff, and the one-off run above used
exactly these rows.

INTEGRATOR TODO: `docs/architecture/database-and-rls.md`, the paragraph "A
membership row needs a reason" (around line 149): the admin row is now only for
the creator while nobody belongs yet; add the UPDATE pin
(`workspace_memberships_pin_columns`, `role` only) and the immutable
`creator_id` (`workspaces_pin_creator`), and the 409 last-admin rule.

INTEGRATOR TODO: indexes: `docs/HANDOFF.md` and `docs/plans/README.md` need no
line from this cluster beyond the wave-4 summary; `docs/review/README.md` if it
lists remediation fragments.
