# Wave 4, project hand-over 403 — review-4 remediation

Branch `rv4/w4-project-handover-403`, base `582a4bd`, five commits
`60b6ed6..40fa986` (one refactor, then two red/fix pairs), then this fragment.
Wave 4, cluster `w4-project-handover-403`: the defect that `rv4-TEST-03` found
while raising API coverage (recorded under "Found on the way" in the
[wave 3 fragment](w3-test-hardening.md)) and the `rv4-FR-F194` finding of the
final review. Findings are cited as `rv4-<ID>` because review IDs restart per
review. No host, address, secret or token is spelled here; every database below
was a private throwaway cluster on loopback, torn down at the end. Nothing was
pushed, merged or deployed, and no server was started (the tests drive the real
router in-process).

## What was decided, and what was not changed

`PATCH /api/v1/projects/{pid}` lets a project's owner or a workspace admin edit
a project. The `projects_update` policy has a `USING` clause and no `WITH
CHECK`, so Postgres judges the new row by `USING` too: an owner who is not a
workspace admin and changes `owner_id` (to another member, or to nobody by
leaving `ownerId` out, which clears it) is refused with SQLSTATE 42501, and the
handler answered `500 {"error":"internal error"}`.

The policy is kept: an owner who is not an admin may edit the project but not
give it away; an admin may do both. The code now says so with a 403. **No
migration was added**, no route, no dependency, and **the PATCH semantics are
unchanged**: `ownerId` absent, `null` or `""` still clears the owner
(`TestProjectEditsFollowTheirRoles` pins it). Whether the policy is the right
one is the owner's question (Q1 below).

## Dispositions

| ID | Status | Red | Fix | Note |
| --- | --- | --- | --- | --- |
| rv4-TEST-03 | `FIXED` | `d4a7d5e` | `c970b31` | the defect it found: a non-admin owner's hand-over is a 403, decided before the UPDATE; the policy's 42501 is mapped to 403 as a backstop |
| rv4-FR-F194 | `PARTIAL` | `d4a7d5e`, `a5d2ef1` | `c970b31`, `40fa986` | three claims. The owner's raw 42501 (500) is fixed with the row above. A new owner must be a workspace member (400), its own red/fix pair. "Omitted `ownerId` clears the owner" is kept as the decided contract and left as Q2 |
| PROJECT-HANDOVER-POLICY | `DEFERRED_OWNER` | | | may an owner who is not a workspace admin give the project away? Q1 below |

## What changed

- **`store.UpdateProject`** (`server/internal/store/projects.go`). The project's
  owner needs no other standing for an edit that keeps the owner. Any change of
  owner, `nil` included, needs an admin: an owner who is not one gets the new
  `ErrHandoverNeedsAdmin` (an `ErrForbidden`) before the UPDATE is tried. A
  member who owns nothing keeps the plain `ErrForbidden`. A new owner who is not
  a member of the project's workspace is `ErrValidation`, judged after the
  caller's standing (so a caller who may not change the owner learns nothing
  about who is in the workspace) and only for a change of owner (so a project
  whose owner has since left the workspace stays editable by an admin with the
  owner the client sends back). The roadmap already holds a node's owner to the
  same rule.
- **`updateProjectRow`** (same file): the one UPDATE, split out in its own
  behavior-free commit (`60b6ed6`) so the backstop can be driven by itself.
  SQLSTATE 42501 from it is `ErrForbidden`, for a caller whose standing changed
  between the check and the write (the store takes no row lock here, BUG-007).
- **`handleUpdateProject`** (`server/internal/api/projects.go`):
  `ErrHandoverNeedsAdmin` is 403 "only a workspace admin can give the project to
  someone else or leave it without an owner (send the current ownerId to keep
  it)"; `ErrValidation` is 400 "ownerId must be a member of this workspace". The
  older 403 text stays for everything else.
- **Tests.** `coverage_projects_test.go`: the known-defect pin is flipped to the
  new truth and renamed `TestProjectOwnerWhoIsNotAnAdminCannotGiveTheProjectAway`
  (403, and the project untouched, for "to another member" and "to nobody"); its
  comment no longer says "known defect". Four new files, `project_update_handover_test.go`
  and `project_update_owner_member_test.go`, each in `internal/api` and
  `internal/store`: 14 tests on real PostgreSQL and its row-level security.

## Evidence

Output files, one per step, are under
`/private/tmp/insideout-rv4/reports/w4-project-handover-403/`.

**rv4-TEST-03, red (`d4a7d5e`, on the unfixed code).** The flipped pin and the
refusal-message test answer `500 {"error":"internal error"}` where 403 is
wanted. The store test gets the raw `new row violates row-level security policy
for table "projects" (SQLSTATE 42501)` instead of `ErrForbidden`. The lock test
(below) waits out its 5 s deadline instead of refusing at once. The backstop
test's premise holds (the bare UPDATE answers 42501) and `updateProjectRow`
returns it raw. **Already green at the red commit, and kept as guards** (the
"stays allowed" cases): a plain edit by a non-admin owner (title, description,
status, and its own id in capitals) stays 200; an admin's hand-over, and its
clearing with `""`, `null` or the key left out, stay 200; the same at the store.

**rv4-TEST-03, green (`c970b31`).** All of the above pass, nothing else in
`internal/api` or `internal/store` changed.

**rv4-FR-F194, red (`a5d2ef1`).** An admin naming an existing user outside the
workspace gets `200` with that user as owner (a project owned by someone who
cannot open it); naming a user id that does not exist gets `500` (the
`projects_owner_id_fkey` foreign key, SQLSTATE 23503). At the store the same two
cases return `<nil>` and the raw 23503. **Already green, kept as guards:** a
caller who may not change the owner gets 403 / `ErrHandoverNeedsAdmin` before
the new owner is judged; an admin's edit that keeps an owner who has left the
workspace still applies. Green: `40fa986`.

**Mutations, each restored afterwards.**

| Mutation | What goes red |
| --- | --- |
| the pre-check removed, the backstop kept | the lock test (the owner's call reaches the UPDATE and waits out 5 s), the `ErrHandoverNeedsAdmin` assertions and the message test; the flipped pin still passes, the backstop answers 403 |
| the backstop removed, the pre-check kept | the backstop test only |
| the new-owner check removed | the member tests, API and store, both cases |
| the new owner judged on every edit, not only on a change | the departed-owner guards, API and store |
| the whole fix reverted | everything in the two red lists above (that is the red run) |

**Why a lock test.** Through the API the pre-check and the backstop answer the
same 403, so no API test can tell them apart. The store test holds an
uncommitted admin UPDATE on the project row from a second connection: a call
that reaches the UPDATE waits for the lock and runs out its deadline, a refused
one answers at once. The admin's hand-over in the same spot is the control; it
must reach the UPDATE and does wait (1.5 s), so the lock is shown to hold the
UPDATE up. The backstop is driven through `updateProjectRow` inside the owner's
own transaction, with a premise assertion that the bare UPDATE really answers
42501 from real Postgres: no mock, no DDL, no race.

## Owner questions

**Q1, PROJECT-HANDOVER-POLICY (`DEFERRED_OWNER`): may a project owner who is not
a workspace admin give the project away?** Today no, by the policy as written
and kept. If yes: a migration giving `projects_update` a `WITH CHECK` that
requires the new owner to be a workspace member (the rule would then also live in
the database), and the `ErrHandoverNeedsAdmin` branch of `store.UpdateProject`
relaxed (hand-over to a member allowed, to nobody still the admin's) or removed.
The tests that change with the answer: the flipped pin,
`TestRvHandoverANonAdminOwnerCannotGiveTheProjectAway`,
`TestRvHandoverTheRefusalComesBeforeTheUpdate`, and the backstop test (its
premise is that the bare UPDATE says 42501).

**Q2 (contract; the unfixed part of `rv4-FR-F194`): should `PATCH` keep the
owner when `ownerId` is absent?** Today absent, `null` and `""` all clear it,
for an admin without a word. The review asks that an omitted owner be kept; the
decided scope here kept replace semantics, and `TestProjectEditsFollowTheirRoles`
pins it, so a change is a contract change for every client.

## Observations

- **No in-repo client omits `ownerId` for an owned project.** `patchProject` in
  `client/lib/api/requests.dart` always sends the key; the only caller of
  `updateProject` is `_edit` in `client/lib/features/project/project_page.dart`,
  which sends back `data['ownerId']`, the owner the page was loaded with (`null`
  only for an unowned project). The CLI, the MCP tools and the Go API client have
  no project-update path at all, and the route is `humanOnly` in
  `server/internal/api/agent_scope.go`. The exposure is an external HTTP caller
  that leaves the key out: as an admin it silently unowns an owned project (200);
  as a non-admin owner it gets the 403 above, which says to send the current
  `ownerId`.
- **A project can keep an owner who left the workspace.** `RemoveMember` deletes
  the membership and leaves `projects.owner_id`, so the owner cannot open the
  project. Nothing clears or reassigns it; the new-owner rule judges only a
  change for that reason, and the departed-owner guards pin that an admin's edit
  still applies.
- `R3-43`'s second half is untouched: `RemoveMember` and `UpdateMemberRole` still
  have no last-admin guard.

## Known ceilings

- The lock test depends on Postgres row-lock behavior and two deadlines (1.5 s
  for the control, 5 s for the refusal, 1 s allowed for the answer); it adds
  about 1.5 s. If it ever flakes, widen the allowed answer time before touching
  the logic.
- The pre-check reads the current owner and the caller's role without a row lock,
  so a change between the check and the UPDATE is possible; the backstop turns
  the policy's refusal into a 403, but a role gained in that window is not seen.
- The new-owner rule is in the store only. The database still accepts a
  non-member owner from any other writer (`owner_id` has a foreign key to users,
  not a membership check); Q1's migration would close that.

## INTEGRATOR TODO

INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md` (the `R3-43`
row, line ~168): the pinned defect is fixed, so the test is now
`TestProjectOwnerWhoIsNotAnAdminCannotGiveTheProjectAway` and answers 403 (this
fragment); keep the last-admin-guard half of the row open. The `rv4-TEST-03`
owner-question row (line ~299, "who may give a project away") becomes
`PROJECT-HANDOVER-POLICY`, `DEFERRED_OWNER`, with Q1 above, and add Q2.

INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-dispositions.md` (line ~171,
the `rv4-TEST-03` row): add that the defect it found is fixed in
`w4-project-handover-403` (red `d4a7d5e`, fix `c970b31`), and add the
`rv4-FR-F194` row as `PARTIAL` (red `d4a7d5e`, `a5d2ef1`; fix `c970b31`,
`40fa986`; Q2 open).

INTEGRATOR TODO: `docs/changelogs/2026-10-18-rv4-remediation/w3-test-hardening.md`
(lines ~201 to 207, "Found on the way", and the to-do at ~288): the "Pinned, not
fixed" text names `..._KnownDefect`, which is renamed; point it at this fragment.

INTEGRATOR TODO: indexes: the `w4-project-handover-403` row in this folder's
`index.md`; `docs/changelogs/README.md` and `docs/changelogs/index/2026-10.md`;
`docs/issues/README.md` for the dispositions above.

INTEGRATOR TODO: `docs/learning/`: a policy with `USING` and no `WITH CHECK`
judges the new row by `USING`, so a rule about who may change a column needs
either a `WITH CHECK` or a decision in the store before the write; map 42501 as
a backstop and prove the two separately (a held row lock for "refused before the
UPDATE", a bare UPDATE as the backstop's premise). No document under `docs/`
names this route or the `projects_update` policy, so no architecture or usage
page changes.

## Verification

All database-backed runs used a private PostgreSQL cluster on loopback, torn
down at the end; a skipped `DATABASE_URL` test counts as red. Nothing was
pushed, merged or deployed.

| Gate | Result |
| --- | --- |
| `gofmt -l internal cmd` | empty |
| `go build ./... && go vet ./...` | clean |
| `go test ./internal/store/... ./internal/api/... -race -count=1` | ok (store 23 s, api 38 s) |
| the same two packages, `-v`, once, counted | 814 `PASS` lines, 0 `FAIL`, 0 `SKIP` |
| files over 350 lines, `.go` and `.dart` changed since the merge base with `rv4/integration` | none (7 files) |
| `scripts/tests/check_*.py` | all fourteen pass |
| `go test ./... -race -count=1`, the whole suite, once, last | exit 0: 28 packages `ok`, none `FAIL` (store 24 s, api 40 s, `cmd/insideout` 259 s) |
| `scripts/check-redaction.sh`, `scripts/check-docs-recorded.sh --worktree`, `scripts/tests/check_doc_links.py` | all three exit 0 with this fragment in the tree |
