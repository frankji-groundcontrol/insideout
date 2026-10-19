# Wave 4, repo binding — review-4 remediation

Branch `rv4/w4-rv-github-binding`, `c04dbb28..45e037fb` (eight commits: four red
tests and four fixes), then this fragment. Cluster `w4-rv-github-binding` of the
whole-change review of `git diff d53035a..rv4/integration`. Findings are cited as
`rv4-<ID>` because review IDs restart per review; `FR` is that review's prefix.
The rules are in the
[execution contract](../../review/2026-10-18-multirole-4/41-remediation-contract.md);
the code these findings sit on is described in
[w3-github-evidence.md](w3-github-evidence.md), and the owner question below is
the one [w4-rv-github-client.md](w4-rv-github-client.md) reproduced a step
earlier.

Server only: one forward migration (`20261018222000_repo_holder_ownerless.sql`),
four source files (`store/github_webhook.go`, `store/project_repo.go`,
`api/github.go`, `api/github_webhook.go`), two new test files and one existing
test file edited. No route, no dependency, no environment variable, no client
change. Three answers change shape, all under "What changed": the webhook reply
gains a `skipped` count, a GitHub refusal while linking or syncing is a 400
`GITHUB_ACCESS_REFUSED` (it was a 502), and linking a repo held by a project the
caller cannot read is a 400 `GITHUB_REPO_NOT_VISIBLE` (it was a 409). Ten rows: 3
`FIXED`, 4 `PARTIAL`, 3 `DEFERRED_OWNER`. The only owner decision is the
entitlement question (F119, F32, F27 and the squatting half of F199); it is
reproduced and recorded, not coded. No address, host, secret or token is spelled
in this fragment.

## Dispositions

| Finding | Sev | Status | Red | Fix |
| --- | --- | --- | --- | --- |
| `rv4-FR-F193` | Med | FIXED | `360f5b89` | `1dfe3382` |
| `rv4-FR-F205` | Med | FIXED (same root and tests as F193) | `360f5b89` | `1dfe3382` |
| `rv4-FR-F200` | Med | PARTIAL: the holder rule is done; the `ownerId` tri-state and the member check on `PATCH /projects/{pid}` sit in files this cluster does not own | `360f5b89` | `1dfe3382` |
| `rv4-FR-F121` | Med | PARTIAL: a holder with no owner, or whose owner left, keeps its repo and a delivery no longer acts as a non-member; what `RemoveMember` does to that member's projects, and the PATCH, are outside | `360f5b89` | `1dfe3382` |
| `rv4-FR-F199` | Med | PARTIAL: the cross-workspace oracle is closed (answered like a repo GitHub does not show); the squatting half is the entitlement decision | `8bb63b9f`, `f786ed5b` | `222b062e`, `45e037fb` |
| `rv4-FR-F170` (API half) | Low | FIXED: `*github.AccessRefusedError` is mapped, and the link-step question the previous cluster left open is answered | `276a03a6` | `5ac56cd4` |
| `rv4-FR-F119`, `rv4-FR-F32`, `rv4-FR-F27` | High, High, Med | DEFERRED_OWNER: reproduced, nothing coded | none | none |
| `rv4-FR-F128` | Low | PARTIAL, not taken: the contradiction is in a doc and in client copy, not in a file of this cluster | none | none |

## What changed

### A project holds its repo whether or not it has an owner (F193, F205, F200, F121)

- **Before.** `insideout._projects_by_repo(url)` is the one question three callers
  ask: the bind check (`claimRepo`), the webhook (`Store.ProjectsByRepo`) and the
  installation-token gate (`commitToken`: a sync gets the token only when exactly
  one project holds the repo and it is the project syncing). Migration
  `20261018180000` ended it with `AND p.owner_id IS NOT NULL`. `PATCH
  /projects/{pid}` writes `owner_id = NULL` whenever the body has no `ownerId`
  (null, omitted and `""` read alike), and `RemoveMember` deletes the membership
  and leaves `owner_id` pointing at somebody who is no longer a member. So a
  project that kept its `repo_url` and lost its owner stopped being the holder:
  a second project could link the same repo (two holders, no error), a signed push
  went to the second project and wrote its commit lines there, and an admin's
  by-hand sync of the first lost its installation token (404 for a private repo).
- **Now.** A forward migration replaces the function (same name, signature and
  grant; `CREATE OR REPLACE`, so a database that took the old one and one that did
  not end up alike). The holder is the earliest-created project carrying the
  repo, `id` breaking a tie, owned or not. The second column is no longer
  `projects.owner_id`; it is the user a delivery acts as: the owner while still a
  member of the project's workspace (a `LEFT JOIN` on `workspace_memberships`),
  NULL otherwise. The migration goes forward because `20261018180000` may already
  be applied; it is not edited.
- **Go side.** `store.ProjectOwner.OwnerID` is a `*uuid.UUID` (nil: nobody to act
  as). The webhook syncs only holders that have an actor. A holder without one is
  counted in a new `skipped` field of the reply, logged at Warn with the repo and
  the project id, and the delivery is not handed to another project; the
  guide-evidence step gets the same acting list. `claimRepo` accepts the holder
  itself, so an admin can repair the link.
- **What a holder with nobody to act as can still do.** An admin (or the owner,
  while a member) can run `POST /projects/{pid}/sync-github`: the project is still
  the holder, so it keeps the installation token. An admin can release the link
  with `{"repoUrl": ""}`, after which another project may bind. An owner who
  rejoins the workspace is the actor again with no further step (by the join; not
  separately tested).
- **Conflict resolved.** F121 offers "have the holder lookup ignore owners who are
  not members"; F193, F200 and F205 want an ownerless project to keep holding.
  Both cannot be had from one column. The rule separates the two questions it
  answered: who holds the repo (any project carrying it) and who a delivery acts
  as (a current member who owns the project).

### A cross-workspace link is answered like a repo GitHub does not show (F199, oracle half)

- **Before.** The 409 `GITHUB_REPO_ALREADY_LINKED` came from a SECURITY DEFINER
  lookup that sees every workspace, so any signed-in project owner could ask
  whether a repo is linked anywhere on the instance, a private one the App covers
  included.
- **Now.** When the holder is not a project the caller can read (one `SELECT
  EXISTS` under the caller's own RLS), `claimRepo` returns
  `store.ErrRepoHeldOutOfReach`. It wraps `ErrRepoTaken`, so `errors.Is` still
  holds, and the API answers it through `errRepoNotVisible`: the status, code and
  bytes of a repo GitHub does not show to the App. The test demands byte equality.
  A caller who can read the holder (the admin's sibling project, a plain member's
  own project, both in the holder's workspace) still gets the specific 409.
- **The sentence.** That one text now reads "GitHub does not show this repository
  to InsideOut, or another project already holds it — install the InsideOut GitHub
  App on it, or link a public repository". The old sentence would have sent the
  squatted real owner, who has installed the App, round in a circle; on an
  instance with no App a holder is the only possible cause. Same code and same
  text for both causes, so nothing new is revealed.
- Two assertions of the existing `github_binding_unique_test.go` moved with it (a
  second link from another workspace was a 409; it is a 400 `GITHUB_REPO_NOT_VISIBLE`
  now, and the concurrent-links test counts a 400 as refused). Nothing else in it
  was weakened.

### A refusal from GitHub is a 400 (F170, API half)

- The previous cluster made the client return `*github.AccessRefusedError` for a
  403 that is not a rate limit. `writeGithubError` sent it down the default branch:
  502 `GITHUB_UPSTREAM`, "GitHub sync failed". Now it is a 400
  `GITHUB_ACCESS_REFUSED`, "GitHub refused access to this repository:" followed by
  GitHub's own words (the client already cleans and caps them), and a Warn line.
  The link step and the sync step answer alike. It is not `GITHUB_REPO_NOT_VISIBLE`
  (the repo is not hidden, GitHub refused) and not a 502 (retrying does not help).
- A rate limit keeps its 429 and its retry hint; a test pins that a 403 with the
  quota spent is still `GITHUB_RATE_LIMITED`.

## Owner decision (DEFERRED_OWNER): rv4-FR-F119, F32, F27 and the squatting half of F199

**Question: who may bind a repo that the GitHub App can see, and what proves it?**
It is the open decision 2 of [w3-github-evidence.md](w3-github-evidence.md) and
F171/F174 of [w4-rv-github-client.md](w4-rv-github-client.md), now named from four
more sides. The link check proves that this server can read the repo, not that the
member linking it may; the first binder wins. Nothing is changed until it is
decided.

**Reproduction.** A scratch test, deleted afterwards (not committed); real
PostgreSQL, real handlers, real RLS, an in-process fake GitHub as the HTTP
transport with one installation covering tenant X's private repo. Y is another
tenant with no relation to X. Run once, at the base; the output is kept as
`/private/tmp/insideout-rv4/reports/w4-rv-github-binding-F119-F32-F27-F199-repro.txt`.
The right-hand column is what this branch's tests pin (the 400) or code it did not
touch (the rest); the scratch test was not run again.

| Step | At the base | At this branch |
| --- | --- | --- |
| Y links X's installed private repo | 200, stored | unchanged |
| Y runs a sync | "added 2 commit line(s)"; Y's timeline holds both commit subjects | unchanged |
| What the fake GitHub saw | `GET /repos/{o}/{r}/commits` carrying X's installation token | unchanged |
| A signed push for X's repo | delivered to Y: Y's timeline gains the new commit | unchanged |
| X links its own repo | 409 `GITHUB_REPO_ALREADY_LINKED` (X is locked out) | 400 `GITHUB_REPO_NOT_VISIBLE` (the oracle fix) |
| Y links a public repo X's team has not linked yet | 200 | unchanged |
| X links that public repo afterwards | 409 | 400 `GITHUB_REPO_NOT_VISIBLE` |

So any member who can name a repo the App can see, an installed private one or
any public one, binds it before its owner. For an installed private repo that is
also a read of its commit subjects, authors and dates into the binder's timeline
and every later push. The real owner is locked out, and nobody can reassign the
claim: only the holder (or an admin of the holder's workspace) can release it. A
200 against a 400 on a private repo still tells any member which private repos
have the App installed (F27's oracle).

**Options** (those of the earlier clusters, plus F199's own):

1. A proof inside the repo that names the project id (a file such as
   `insideout.yaml`, or a topic), read with the installation token. Proves write
   access to the repo, needs no new identity system, asks the repo owner to
   commit. The same proof could let a rightful owner take over a squat.
2. A GitHub OAuth identity for each member, checked for push or admin on the repo.
   The strongest, and the heaviest: an OAuth app, token storage, a new sign-in path.
3. Tie each installation to the workspace that installed it (recorded through the
   App's setup flow) and accept only that workspace's repos. Clean for tenants,
   needs the install flow built, does nothing for two members of one workspace.
4. Approval by an instance administrator. No code to speak of, an operator task
   per link.
5. F199's own: scope uniqueness to the workspace and route deliveries by
   installation. It undoes the one-holder rule of rv4-COMP-05 and needs option 3's
   mapping to route anything.

None is decision-free: each refuses a legitimate owner's link until the proof is
given, which changes the private-repo flow for everyone. What an operator can do
today: on an instance with more than one workspace, install the App only on
repositories whose commit subjects every member of the instance may read, or run
one instance per trust boundary. A single-workspace instance is not exposed.

**What changes with the answer.** `handleSetProjectRepo`, `requireRepoVisible` and
`writeRepoStoreError` (`api/github.go`) and `claimRepo` (`store/project_repo.go`);
for option 3 also the webhook path that learns the installation. The holder rule
of this cluster stays.

## Not taken, and why

- **F128 (Low, optional).** The contradiction is in
  `docs/usage/cli-mcp-and-guidance.md` (step 1 "Bind the repo", step 4 "Install
  the GitHub App": for a private repo the App has to come first) and in the client
  copy; neither is a file of this cluster. The server's answer already carries the
  hint ("install the InsideOut GitHub App on it, or link a public repository"), so
  the finding's "no install hint" half is overstated; what is open is the client
  mapping of the code (R3-28 in `docs/issues/`). The finding's other way out, a
  pending bind the webhook later confirms, is a claim with no proof and belongs to
  the entitlement decision above.
- **F177** (a per-user limit on `PUT /projects/{pid}/repo`) is not in this
  cluster's list.
- **The `PATCH` tri-state and the `RemoveMember` rule** of F121 and F200 are in
  `projects.go`, `api/projects.go` and `memberships.go`, none of them owned here.

## Known ceilings, recorded on purpose

- **The held half of the oracle is closed for repos a stranger cannot show
  elsewhere.** For an installed private repo, "held" and "App not installed" now
  read alike. For a repo the caller can see is public, a 400 still means "held by
  a project you cannot read", because public-ness is checkable on github.com. F27's
  200-against-400 split ("App installed here") stays with the entitlement
  decision.
- **No App configured: recoded, not closed.** `requireRepoVisible` never refuses
  without the App, so there a 400 `GITHUB_REPO_NOT_VISIBLE` can only mean a holder
  out of reach. No credential is handed out in that configuration; what leaks is
  that some project on the instance linked a public repo, as the 409 did before.
- **A holder with nobody to act as still blocks the repo.** That is F193's
  point, and it leaves F121's "still blocks every other project" true for the
  link. The way out is the admin's: give the project an owner, or release it. Its
  deliveries are skipped, not synced.
- **Rows that predate the one-project rule** keep their order (earliest created
  holds), now whatever the owner state: a holder does not change when somebody's
  ownership does.
- **The store-level companion of the oracle fix** (`TestRvGhBindARefusedLink...`)
  was written with the fix, not before it, because it names the new sentinel and
  would not compile without it. The red of record for F199 is the API test.

## Left for the integrator

INTEGRATOR TODO, none in files this cluster owns:

- INTEGRATOR TODO: `docs/usage/cli-mcp-and-guidance.md` (lines 195-222, "Connecting
  a repository"): for a private repo the App goes before step 1 (F128); a second
  link inside the workspace is a 409 `GITHUB_REPO_ALREADY_LINKED`, from another
  workspace a 400 `GITHUB_REPO_NOT_VISIBLE`; add the 400 `GITHUB_ACCESS_REFUSED`
  and the webhook reply's `skipped` count.
- INTEGRATOR TODO: `docs/architecture/product-subsystems.md` (lines 94-107):
  "the earliest-created one with an owner" becomes "the earliest-created one,
  owned or not; a delivery acts as its owner while that owner is a member and is
  skipped otherwise"; the second link from another workspace answers
  `GITHUB_REPO_NOT_VISIBLE`; list migration `20261018222000`.
- INTEGRATOR TODO: `server/internal/store/projects.go` and
  `server/internal/api/projects.go`: make `ownerId` tri-state on `PATCH` (absent
  keeps; null clears, for an admin only; a UUID must be a current workspace
  member, else 400) and map the RLS failure of a bad owner to 403 (F200, F121).
- INTEGRATOR TODO: `server/internal/store/memberships.go`: `RemoveMember` leaves
  `owner_id` alone; decide whether a member's projects are reassigned, cleared or
  lose their repo link when they leave (a product question for the owner; this
  cluster only stopped deliveries from acting as them).
- INTEGRATOR TODO: `server/internal/api/agent_scope.go`: optionally `humanOnly` for
  `PUT /projects/{pid}/repo` while the entitlement gap is open (F119 "at
  minimum"); it changes what agents can do, so it is the owner's call.
- INTEGRATOR TODO: `server/internal/api/github.go`: the per-user limit on
  `PUT /projects/{pid}/repo` (F177, the previous cluster's to-do) is not among this
  cluster's findings and stays open; assign it to whoever works on the file next.
- INTEGRATOR TODO: `client/lib/api/errors_l10n.dart` and
  `client/lib/features/project/project_github_card.dart`: map by code, not only by
  status, so `GITHUB_ACCESS_REFUSED`, `GITHUB_REPO_NOT_VISIBLE` and
  `GITHUB_REPO_ALREADY_LINKED` each get a sentence in English and Chinese (R3-28).
- INTEGRATOR TODO: `docs/issues/` (the rv4-COMP-05 open owner question): the
  entitlement question now has F119, F32, F27, F171, F174 and the squatting half of
  F199 behind it.
- INTEGRATOR TODO: shared indexes (`docs/HANDOFF.md`, `docs/plans/README.md`,
  `docs/changelogs/README.md`, `docs/changelogs/index/2026-10.md`, the index of this
  folder, `docs/issues/README.md`, `docs/learning/README.md`,
  `docs/review/README.md`): add this cluster; it is wave 4, `w4-rv-github-binding`.

## Evidence

The first three reds were run on the base plus their test file, against a
database migrated to the base, so the old function answered; the sentence red
(`f786ed5b`) was run on `5ac56cd4` plus its test. Outputs are kept under
`/private/tmp/insideout-rv4/reports/w4-rv-github-binding-*`.

- **F193, F205, F200, F121, red `360f5b89`.** Store: "B linked the repo that
  ownerless project ... still carries (err <nil>), want ErrRepoTaken: two projects
  now hold one repo"; "lookup named 1 project(s) [...], want only ..., the one
  created first (it has lost its owner, not its repo)"; "after the owner left, a
  delivery would act as ...: it can read nothing of the project, so the sync can
  only fail; want nobody to act as". API: "a second project links the repo an
  ownerless project holds: status 200, want 409"; "delivery answered
  {"commits":1,"event":"push","evidence":0,"projects":1,...}" with "the other
  project's timeline has 1 line(s), want none"; "sync-github: status 404, want
  200; body {"code":"GITHUB_NOT_FOUND",...}".
- **F199 oracle, red `8bb63b9f`.** "a stranger can tell a repo another workspace
  holds (409 {"code":"GITHUB_REPO_ALREADY_LINKED",...}) from one GitHub does not
  show (400 {"code":"GITHUB_REPO_NOT_VISIBLE",...})".
- **F199 sentence, red `f786ed5b`.** For each cause: "the answer "GitHub does not
  show this repository to InsideOut — install the InsideOut GitHub App on it, or
  link a public repository" does not say that another project may hold the repo".
- **F170 API, red `276a03a6`.** "link while GitHub refuses the App: status 502,
  want 400; body {"code":"GITHUB_UPSTREAM",...}" and "sync while GitHub refuses
  the commits read: status 502, want 400". The spent-quota test passes at the base
  and at the head and stays as a pin.
- **Green.** Twelve new tests (store 4, API 8) pass under `-race`, and so do all of
  `internal/store` and `internal/api` (see Verification).
- **Upgrade replay.** `scripts/upgrade-replay.sh` replays the deployed migration
  files and then the pending ones against seeded data: "UPGRADE REPLAY GREEN: 31
  deployed files replayed, 14 pending migrations met the seeded data" (run at
  `5ac56cd4`; the two commits after it touch no migration).

Mutation checks, each restored and compared with `cmp` afterwards:

| Mutation | Result |
| --- | --- |
| the function back to the `20261018180000` text (owner filter) | 6 fail (the three holder tests in each package) |
| the actor is the stored `owner_id`, no membership join | 1 fails (owner left) |
| the webhook runs a holder with nobody to act as instead of skipping it | 1 fails |
| a holder is always treated as readable (the 409 for everyone) | 4 fail (1 store, 3 API) |
| a holder is never treated as readable (the cover answer for everyone) | 2 fail (1 store, 1 API) |
| the `AccessRefusedError` case removed from `writeGithubError` | 2 fail (link, sync) |

## Verification

Gates at `45e037fb`; only documentation follows it (this fragment). Private
PostgreSQL 18 with the migrations applied, a skip would count as red:

| Gate | Result |
| --- | --- |
| `go build ./...` and `go vet ./...` in `server` | clean |
| `gofmt -l internal cmd` | empty |
| `go test ./internal/store ./internal/api -race -count=1 -v` | 367 top-level tests (793 with subtests) pass, 0 fail, 0 skip; store 122 (128), API 245 (665) |
| the whole server suite, once, `-p 4` | 28 packages ok, 0 failed, 3 without tests |
| changed `.go` files over 350 lines | none; largest is `api/github_webhook.go`, 252 |
| the 14 `scripts/tests/check_*.py` guards, with this fragment staged | 14 pass (the link guard: 0 broken relative links over 618 tracked `.md` files) |
| `sh scripts/check-redaction.sh` | exit 0 |
| `sh scripts/check-docs-recorded.sh` | exit 0 |

The whole-suite run was not verbose, so it does not list skips; the `-v` run above
shows none. No client file changed, so Flutter was not run. Nothing was pushed and
no CI run is claimed.
