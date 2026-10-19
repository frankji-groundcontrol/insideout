# Review-4 remediation, wave 2: credentials, apply once, agent scope and RLS writes

The wave-2 record of the [folder index](index.md): the outcome, the per-ID
dispositions, the gates, the live check, the carry-forward and the operator
notes. Each cluster's own account (what changed, the evidence, the ceilings) is
its fragment, linked below.

## Wave 2

Four clusters, plan rows 2.1 to 2.6. The integrator cherry-picked 55 commits
with `-x` onto the wave-1 records head `4b2a1db` (`38aafed` through `e6bc637`,
148 above `d53035a`; no conflicts, no fix-ups, no baseline failure). The records
follow-ups are `7696b08`; this section and the per-ID state are the commit
titled `docs(rv4): wave 2 dispositions`. All four verifiers accepted in round 1.
SHAs are on `rv4/integration`; the fragments cite the worker-branch SHAs, and
each integration commit is patch-identical to its worker commit (its body names
the original in the `cherry-pick -x` line). The wave changes 123 files (8,970
insertions, 807 deletions), adds two migrations
(`20261018120000_proposal_items_applied_at.sql` and
`20261018140000_rls_membership_and_user_grants.sql`; the slot `20261018130000`
stays unused) and no dependency.

Outcome lists (23 rows): `FIXED` 21; `PARTIAL` 2 (`FEAT-02-CLIENT-APPLY`,
rv4-DATA-03; wave 3 closed both: [w3-client-apply](w3-client-apply.md),
[w3-users-read-grant](w3-users-read-grant.md)); `NOT_REAL` none; `BLOCKED` none.
rv4-SEC-05 was `PARTIAL` in its own cluster (the client half) and is `FIXED` here
because rv4-AIDEV-04 closed the admin-tools half in the same wave.
`FEAT-02-CLIENT-APPLY` is the worker's id for the web feed that re-applies an
accepted proposal (rows 2.3 and 2.4); R1-03 and R1-09 are residue rows of wave 1
that this wave closed, and R1-09 is the `LEAD-PROPOSE-VALIDATE` row.
`DEFERRED_OWNER` is two owner calls (may an agent token ever decide a proposal;
the repository variable that feeds the `DefaultBase` link flag) and two operator
actions (confirm the hosted database role, deploy the server before the MCP
binary), all in the last table of the
[open-findings record](../../issues/2026-10-18-review-4-open-findings.md#owner-decisions-no-engineering-action-until-decided).

### Clusters

| Cluster | Rows | What landed | Verifier | Fragment |
| --- | --- | --- | --- | --- |
| `w2-client-credentials` | 2.1, 2.2 | one `clientconfig.Resolve(flagAPI)` for the CLI, every `--api` flag and `insideout-mcp`; a saved token goes only to the base it was saved for and an unreadable config is an error; `DefaultBase` is a `var` that `-ldflags -X` retargets; setup keeps the refresh token and a saved session renews once on 401 (a loser of a concurrent renewal waits for the winner's pair); `login` and `register` save the session; `setup --check` fails a rejected login; the web session ends only on a 401 or 403 refresh; `scripts/mcp-config.sh` is gone | round 1 | [w2-client-credentials.md](w2-client-credentials.md) |
| `w2-proposal-apply` | 2.3, 2.4 | `proposal_items.applied_at`: an item is applied once, in one transaction with the decision, and a failed apply is a 500; propose validates before its first write; the Decision Log comment tells the truth; web "Break down with AI" drafts, shows the draft and confirms it | round 1 | [w2-proposal-apply.md](w2-proposal-apply.md) |
| `w2-agent-scope` | 2.5 | a signed `scope` claim, `POST /api/v1/agent/token` and one fail-closed `agentRoutePolicy`; the MCP binary runs on an agent token and its registry follows the login (52 to 40 tools); coach and MCP results are labelled and fenced; the `invite_create` description names `mailSent` | round 1 | [w2-agent-scope.md](w2-agent-scope.md) |
| `w2-rls` | 2.6 | membership `INSERT` needs the creator or the join code; `users` `INSERT` is cut by column grants and `UPDATE` is pinned by a trigger; `serve` refuses a `BYPASSRLS` or `SUPERUSER` pool role; a write-side RLS walk | round 1 | [w2-rls.md](w2-rls.md) |

### Dispositions

Status is the engineering status. Every `FIXED` has a red test that failed
before its fix and passes after it (the verifiers replayed each pair at the
worker SHAs; the integrator ran the whole suite at the head). No row here is a
CI-config property, so none says "pending first CI run": no workflow file
changed. The R2 IDs are the rows of the wave-2 residue table in the
[open-findings record](../../issues/2026-10-18-review-4-open-findings.md#wave-2-open-residue-integrator-2026-10-19);
the one-line dispositions are in the
[dispositions record](../../issues/2026-10-18-review-4-dispositions.md#wave-2-dispositions-integrator-2026-10-19).

| Finding | Cluster | Row | Status | Red | Fix | Left open |
| --- | --- | --- | --- | --- | --- | --- |
| rv4-ARCH-01 | client-credentials | 2.1 | FIXED | `38aafed` | `c5af0e9`; chore `8899c73` | `scripts/setup.sh:53` still teaches `export INSIDEOUT_TOKEN=...` (R2-06). The red message says 38 verb forms; the exact count is 37 plus 7 saved-base verbs |
| rv4-ARCH-02 | client-credentials | 2.1 | FIXED | `3976879` | `ca1c06a` | A request that cannot start is wrapped as `Get "/me": ...` (R2-13, cosmetic) |
| rv4-ARCH-05 | client-credentials | 2.1 | FIXED | `acdc088` | `1df2a47`; chore `3645f0d` | `release.yml` does not pass `-X ...DefaultBase=` yet (R2-07, owner: the variable name) |
| rv4-SEC-05 | client-credentials, agent-scope | 2.1, 2.5 | FIXED | `f312ab6`, `a308076` | `6aa5760`, `5e4968e` | Client half: the MCP process fixes its base once. Admin half: a JWT login's registry no longer offers `signup_mode` or `invite_*`, and `/admin/*` answers an agent token 403 (rv4-AIDEV-04) |
| rv4-FULL-02 | client-credentials | 2.2 | FIXED | `ef1af83` | `96748fc` | none |
| rv4-USER-04 | client-credentials | 2.2 | FIXED | `37da942`, `a5a790f` | `12b6a00`, `90bab89` | The renewal wait is a poll, not a lock (R2-12); an MCP `auth_*` session cannot renew (R2-10); `Client.Guide` prints the plain 401 (R2-11) |
| rv4-USER-03 | client-credentials | 2.2 | FIXED | `8bed85d` | `5111740` | none |
| rv4-USER-05 | client-credentials | 2.2 | FIXED | `719b780` | `76b55a2` | An unreachable server stays a `skip`, on purpose: the check also runs offline right after install |
| rv4-FULL-05 | client-credentials | 2.2 | FIXED | `f3d51af` | `1d9d1ff` | none |
| rv4-FEAT-02 | proposal-apply | 2.3 | FIXED | `f624efb` | `a512e0a`; guard `70d3136` | Server half. The web feed's own re-apply is the next row. `AddProposalItems`, `ApplyProposalItems` and `ApplyProposalItemsForUpdate` have no production caller (R2-14) |
| FEAT-02-CLIENT-APPLY | proposal-apply | 2.3, 2.4 | PARTIAL (FIXED in wave 3) | the changed test in the staged patch (red on the old feed, scratch copy) | not applied | `project_updates_feed.dart` re-creates every accepted `add_node` item after the server already did; the patch was not in the cluster's files and the integrator took no product code (R2-01). Closed in wave 3 by `w3-client-apply`: red `6096862`, fix `0d33077` ([record](w3-client-apply.md)) |
| rv4-FEAT-09 | proposal-apply | 2.3 | FIXED | `3891f08` | `f8c9959` | Older migrations keep their historical "immutable" comments (migrations are not edited) |
| rv4-LEAD-PROPOSE-VALIDATE (R1-09) | proposal-apply | 2.3 | FIXED | `6d8e957` | `bfe019e` | none; R1-09 is closed |
| rv4-FULL-01 | proposal-apply | 2.4 | FIXED | `2046fb5` | `29c0956` | The expand route takes no per-project position lock and gates on membership only (R2-04) |
| rv4-FEAT-01 | agent-scope | 2.5 | FIXED | `62e68f4` | `3a22813`; design `9b3cdb9` | `DEFERRED_OWNER`: may an agent token ever decide a proposal (strict default shipped). Accepted limits R2-15; audit rows still carry the user id (R2-05) |
| rv4-AIDEV-04 | agent-scope | 2.5 | FIXED | `a308076`, `c4087ec` | `5e4968e`, `8def9d6` | A process with no credential keeps 52 tools until its first sign-in (R2-15); deploy the server before the MCP binary (operator notes) |
| rv4-AIDEV-01 | agent-scope | 2.5 | FIXED | `883daa1` | `6e6622f` | `agent/prompt_fence.go` still holds its own copy of the fence rule (R2-08) |
| rv4-AIDEV-08 | agent-scope | 2.5 | FIXED | `53ddb09` | `7aaf8ad` | The CLI `context` and `evidence` verbs print the API JSON unfenced (R2-09) |
| R1-03 | agent-scope | 2.5 | FIXED | `0b2e9fd` | `f9e9894` | none; a JWT login no longer offers `invite_create`, so the sentence serves a process with no credential or an opaque token |
| rv4-DATA-01 | rls | 2.6 | FIXED | `91098a7` | `f696db4` | A seat can still be re-pointed by its workspace's admin (R2-03) |
| rv4-DATA-02 | rls | 2.6 | FIXED | `91098a7` | `f108e64` | The write walk shares its red commit with DATA-01 and DATA-03; its third test is green on the base on purpose |
| rv4-DATA-03 | rls | 2.6 | PARTIAL (read half FIXED in wave 3) | `91098a7` | `f108e64` | Write half done. Read half: a co-member's `password_hash` is readable by anything that holds the app role (no API response carries it) (R2-02). Closed in wave 3 by `w3-users-read-grant`: red `8f57d77`, fix `c57073b` ([record](w3-users-read-grant.md)) |
| rv4-DATA-04 | rls | 2.6 | FIXED | `ad7ce63` | `e9d8a79` | Hosted `DATABASE_URL` must be the app role before the first deploy (owner decisions); `SUPERUSER` is covered by a table test only |

Why `FEAT-02-CLIENT-APPLY` was still open at the end of wave 2: the pre-gate rule
keeps product code and tests out of the records pass, and the worker's staged
patch edits `client/lib/features/project/project_updates_feed.dart` and
`client/test/project_page_test.dart`. Until `w3-client-apply` landed, an accept
through the web feed made the server's nodes and then the feed's own copy. The
server-side once-only guarantee held for every other caller (API, CLI, MCP).
Wave 3 closed it: the feed now only asks the server to decide and apply
(`6096862` red, `0d33077` fix; [w3-client-apply](w3-client-apply.md)).

### Gates (integrator, at `e6bc637`)

On a fresh private PostgreSQL 18 that was torn down afterwards:

- `gofmt -l .` prints nothing; `go build ./...`, `go vet ./...` and
  `go mod tidy -diff` (in `server/`) are clean.
- The migrations apply from empty, including `20261018120000` and
  `20261018140000`; a second `migrate` does nothing.
- `go test ./... -race -count=1 -v`: every package ok (`cmd/insideout` takes
  227 s), 1,563 `PASS` lines, the only skip `TestReadSecretTerminal`, no data
  race.
- Flutter: `pub get --enforce-lockfile` ok (`pubspec.lock` unchanged); `analyze`
  0 errors or warnings and 14 infos (equals `INFO_BASELINE` 14); 483 tests pass,
  also under `TZ=Asia/Shanghai`.
- All nine `scripts/tests/check_*.py` guards exit 0 (docs truth: 52 MCP tools,
  23 internal packages); `sh scripts/check-redaction.sh` and
  `sh scripts/check-docs-recorded.sh` exit 0.
- No workflow file changed in this wave, so R1-17 (the first CI run) is
  unchanged.

After the two records commits the build, vet, `gofmt`, `go mod tidy -diff`, the
guards and the link check of the touched files were run again and are green; the
records commits touch no Go, Dart, script, workflow or migration file, and the
one non-document edit is a comment in `.env.example`.

### Live check (integrator)

Real binaries, two servers on loopback ports (A with `INSIDEOUT_ACCESS_TTL=3s`,
B with the default) sharing one private database:

| # | Observed |
| --- | --- |
| 1 | A login saved for A, five seconds later `whoami` succeeds (A's log: `/me` 401, `/auth/refresh` 200, `/me` 200). `roadmap list --api B` reaches B and sends none of A's token (B's log shows the request with no token and a 401 unauthorized) |
| 2 | A propose with bad items answers 400 and leaves 0 `project_updates` rows |
| 3 | Accept with `apply` twice: the first answers `appliedNodes` with 2 ids, the second `[]`; the roadmap holds 2 nodes |
| 4 | The real expand draft `{children:[{title,description}]}` and the confirm 201 list have the shape the client contract tests pin |
| 5 | psql as the app role: a self-insert as admin, and as member without the join code, is refused by RLS; a self-promotion of `system_role` is refused by the pin trigger; an `INSERT` into `users` that sets `system_role` is "permission denied" |
| 6 | `serve` with the owner DSN exits 1 with "refusing to serve", names `BYPASSRLS` and prints no DSN |
| 7 | A token from `/agent/token`: the proposal decision, Commit, `PUT` admin settings, admin invitations, `DELETE` of an idea, a roadmap node, a project, an update and a workspace, and `/agent/token` itself are all 403 `agent_scope_denied`; the same decision with the full login is 200. The password routes are unauthenticated reset flows and are not scope-gated, by design |

The servers were stopped and the port was confirmed down.

### Follow-ups closed in `7696b08`

Residue from the worker hand-overs that is only a document, a comment or an
index edit (16 files). No product code, test, script, workflow or migration
changed; each item was read back and the guards pass.

| Closed | Where | Finding |
| --- | --- | --- |
| `agent-token-scope.md` is listed in the architecture index | `docs/architecture/index.md` | rv4-FEAT-01 |
| The layout and request flow: one `clientconfig.Resolve`, renewal on 401, the agent-token exchange, the `scope` claim and `agentRoutePolicy`, the fenced `agentcontext` and MCP results, twelve tools withheld from an agent profile | `docs/architecture/backend.md` | rv4-ARCH-01, rv4-USER-04, rv4-FEAT-01, rv4-AIDEV-04, rv4-AIDEV-08 |
| The proposal-apply contract: decision and apply in one transaction, once per item, a failed apply is a 500 and records no decision | `docs/architecture/product-subsystems.md` | rv4-FEAT-02 |
| The proposal-item stamp: `applied_at`, `UPDATE` narrowed to it, the table-level `DELETE` kept with no policy | `docs/architecture/database-and-rls.md` | rv4-FEAT-02 |
| The server enforces the agent boundary; user-written text returns inside `<prd_data>`; the registry is 52 before a sign-in and 40 after; `idea_delete` and `roadmap_delete` are CLI-only for a signed-in MCP; deploy the server before the binary; a stale "no API deletes" claim is gone | `PRODUCT.md`, `SKILL.md`, `docs/usage/cli-mcp-and-guidance.md` | rv4-FEAT-01, rv4-AIDEV-04, rv4-AIDEV-08 |
| `setup --check` goes through `Resolve`; `serve` exits 1 on a `BYPASSRLS` or `SUPERUSER` pool role; `DATABASE_URL` must be the app role | `docs/usage/environment.md`, `docs/SETENV.md`, `docs/usage/deployment.md`, `.env.example` (a comment only) | rv4-USER-05, rv4-DATA-04 |
| Four stale line citations (`cli_version.go`, `product_tools.go`, `proposal_decisions.go`, `agentcontext.go`) | `docs/issues/2026-10-11-focus-surface.md`, `docs/issues/2026-10-11-product-branch-merge-decision-log.md` | none |
| New: two reusable notes, a route policy that keys on the pattern and fails closed, and a column grant turns zero rows into permission denied | [learning](../../learning/README.md) | rv4-FEAT-01, rv4-DATA-03 |

Also: the four wave-2 fragments are listed in the [index](index.md); the
`w1-auth-atomic` and `w1-clock-and-runes` fragments and [wave 1](wave-1.md) now
mark R1-03 (`f9e9894`) and R1-09 (`bfe019e`) closed; `docs/architecture/` and
`docs/usage/` carry the new contract; and the rest became the residue table
R2-01 to R2-15 in the open-findings record, each with an owner and a one-line
fix prompt (product code, tests, executing scripts and workflows are outside a
records pass). `PRODUCT.md` (462 lines) and `docs/SETENV.md` (368) were already
over the line budget and each grew by a few lines, so their split is recorded in
[product-and-setenv-over-line-budget](../../issues/2026-10-18-product-and-setenv-over-line-budget.md).

## Carry forward into wave 3

1. **Two clusters and two optional rows were added.** `w3-client-apply` (R2-01)
   and `w3-users-read-grant` (R2-02, after `w3-auth-network`) are the 12th and
   13th wave-3 clusters. `LEAD-FENCE-DEDUP` (R2-08) rides with `w3-coach-trust`
   (row 3.1) and `LEAD-SETUP-BLURB` (R2-06) with `w3-first-run-copy` (row 3.8). A
   worker on those rows reads its R2 line first.
2. **A new authenticated route is a decision.** Classify it in `agentRoutePolicy`
   (`server/internal/api/agent_scope.go`) and in the two lists of
   `agent_scope_routes_test.go`, or `TestEveryRouteHasAnAgentScopeDecision` fails.
   A tool that must not reach an agent goes in `withheldFromAgents`
   (`mcpserver/server.go`); every other successful non-auth tool result is
   fenced by `textResult`, and the sweep in `fence_test.go` fails on a gap.
3. **The write side of RLS has a walk.** A migration that adds an RLS table, or a
   `users` column, extends `rls_write_walk_test.go` (a new `users` column is
   refused until it is classified). A fixture that `INSERT`s a privileged `users`
   column, or `UPDATE`s another user's, as `insideout_app` now fails with 42501
   by design: go through the owner pool or a store method. A live proof that
   opens signup must put `signup_mode` back to `invitation_only`, or
   `TestSignupModeSetting` fails on the shared row.
4. **`clientconfig` behaviour changed.** `Load` returns only the file and
   `Resolve` owns the environment; an unreadable config is fatal for every
   client verb, the MCP tools and `setup --check`. Row 3.9 (context threading in
   `apiclient`) and any new verb build their client with `clientFromFlag`, not by
   hand.
5. **Four out-of-ownership edits were disclosed and kept:**
   `client/test/sse_test.dart`, `scripts/default-api.sh`,
   `scripts/check-redaction.sh` and `server/internal/apiclient/escape_test.go`. A
   worker who edits them keeps the `DefaultBase` reader that accepts a `const`
   or a `var` and the `RefreshToken` and `SetRenewal` accessor names.
6. **Leads for the final review (not findings):** `ExpandRoadmapNode` (R2-04);
   a workspace admin re-pointing a seat (R2-03); the body-dependent agent check
   on `POST /prds/{id}/status` (the check reads the whole capped body and
   survived no mutation); version skew (a new MCP binary against an older
   server fails every tool); the `clientconfig` behaviour changes in item 4.
7. **HANDOFF and the plans board.** The "Everything engineering is closed"
   paragraph of `docs/HANDOFF.md` and the docs-order row of `docs/plans/README.md`
   stay as [wave 1](wave-1.md) left them for `w3-docs-truth` (row 3.6). The
   coordinator's lessons from the run are not recorded yet; they become one
   `docs/learning/` note in the final records commit.

### Operator notes

No new dependency. Two migrations; both re-run to the same state (a `REVOKE`
comes before each `GRANT`). The startup migrations apply them after the serve
check below, so the order of a first deploy is: confirm the role, then deploy the
server, then release the MCP binary. What a running install sees:

- **Before the first deploy that contains this wave:** `serve` exits 1, before
  migrating, when the `DATABASE_URL` role has `BYPASSRLS` or is a superuser. The
  hosted environment was not inspected from here. Confirm `DATABASE_URL` is the
  `insideout_app` role and `DATABASE_OWNER_URL` the owner; rollback is
  redeploying the previous image. `migrate`, `seed` and `promote-admin` never
  reach the check.
- **Deploy the server before the MCP binary.** A new `insideout-mcp` against a
  server without `POST /api/v1/agent/token` fails every tool with a hint to
  update the server. Tokens issued before the change have no `scope` claim and
  stay full sessions, so nobody has to sign in again.
- **`proposal_items` gets `applied_at`.** Every row that existed before the
  migration reads as not applied, because the old code remembered nothing and
  nothing can say which were. Accepting such an already-applied proposal again
  with `apply` builds its nodes once more, and only once.
- **Accepting a proposal:** `appliedNodes` lists the nodes this call created and
  is `[]` for a repeat. A failed apply is a 500 `internal error` with no decision
  recorded, so the proposal stays open to accept again.
- **The CLI:** `login` and `register` write the 0600 config and print its path on
  stderr (stdout is still the bare token). A saved token goes only to the base it
  was saved for, so `--api B` after a login to A is unauthenticated. A saved
  session renews once on a 401. An unreadable config fails every verb with the
  file name. An `INSIDEOUT_TOKEN` in the environment still wins, is never renewed
  and hides the saved login; the 401 message says so.
- **MCP registration** is the static
  `{"mcpServers":{"insideout":{"command":"insideout-mcp"}}}` with no `env` and no
  token; `scripts/mcp-config.sh` is removed. A JWT login sees 40 tools, not 52;
  its tool results come back labelled and inside a `<prd_data>` fence.
- **The release binaries** keep the compiled-in hosted default until
  `release.yml` passes `-X` (R2-07); a source build can retarget it with
  `-ldflags "-X ...apiclient.DefaultBase=<url>"`.
- **Web:** "Break down with AI" shows its draft and adds the sub-tasks only on
  the confirm button; a dropped connection no longer signs the user out. Until
  R2-01 lands, accepting a proposal in the project feed still adds each node a
  second time.

Nothing is pushed, merged or deployed.
