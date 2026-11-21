# Focused review 5 — 2026-11-21

A review of `main` @ `4d186da` together with its 57 uncommitted files, on the
five things the owner asked about: feature completeness, consistency across
the web app, REST API, MCP server and CLI, the task tree for different roles,
code and file structure, and the documentation index inspected with Marksman.
Ten reviewers read the tree on one finding schema, skeptics tried to refute
every High, the coordinator reproduced the top findings on a private
PostgreSQL 18 cluster, and a completeness critic closed the review. No source
file was changed.

> **Bottom line.** The single-author loop works: capture an idea, coach it
> into a PRD, commit a version, build a roadmap, attach evidence. The design
> the owner calls crucial is about one quarter built. A roadmap node can carry
> one owner, and that is all: the person a node is handed to is refused on
> every write inside it, no node opens as its own tree, there is no "my tree",
> and the MCP tools and the CLI cannot assign an owner at all. About half of
> PRODUCT.md runs. The four surfaces agree on sign-in and PRD verbs and
> disagree on projects and on the node fields delegation needs. The build
> path has three reproduced defects that lose or misplace data. The pushed
> tip of `main` is red in CI. Code is modular by file size and less so by
> function and responsibility. The documentation graph is sound; the routers
> that say what is open are stale. Wave 0 and wave 1 of the
> [remediation plan](./40-remediation-plan.md) are about fifteen small
> changes; the task tree is nine slices after that.

## Findings at a glance

127 unique findings: **0 Critical, 21 High, 81 Medium, 25 Low** after
verification. Of the 29 findings filed as High, skeptics confirmed 20,
downgraded 9 to Medium and refuted none; the critic added one High. The 21
High rows are 15 distinct defects.

| File | The owner's question | Score | Critic | What it found |
| --- | --- | --- | --- | --- |
| [10](./10-feature-product.md) | 1. Feature completeness: product definition | 5.0 | 5.0 | 16 of 50 promises run fully, 19 in part, 15 not at all; three build-path defects |
| [11](./11-feature-frontend.md) | 1. Feature completeness: Flutter client | 6.2 | 5.5 | States, strings and themes are disciplined; the coach transcript does not update; an approving admin cannot reach a PRD |
| [12](./12-parity-identity.md) | 2. Surface parity: identity and workspace | 6.2 | 6.2 | Sign-in is consistent; a title-only idea update from CLI or MCP erases the body |
| [13](./13-parity-prd.md) | 2. Surface parity: project and PRD | 5.6 | 5.0 | PRD verbs line up; beyond list and create, projects exist only on the web |
| [14](./14-parity-roadmap.md) | 2. Surface parity: roadmap and delivery | 6.3 | 5.5 | Structural verbs are on all four surfaces; node owner and acceptance are web-only; no test would catch the next drift |
| [15](./15-tree-model.md) | 3. Task tree for roles: backend | 2.5 | 2.5 | 1 of 11 properties implemented, 4 partial, 6 missing |
| [16](./16-tree-experience.md) | 3. Task tree for roles: experience | 2.5 | 3.0 | 1 of 11 properties implemented, 6 partial, 4 missing |
| [20](./20-structure-server.md) | 4. Code structure: Go server | 6.3 | 7.0 | Acyclic packages, every file under the guard; a 315-line function, three programs in one `main` package |
| [21](./21-structure-client-scripts.md) | 4. Code structure: client, scripts, layout | 6.2 | 6.5 | Client is in reasonable shape; setup is implemented three to four times and the copies have diverged |
| [22](./22-docs-index.md) | 5. Documentation index and Marksman | 6.2 | 6.5 | 657 of 659 files reachable, no broken link; the board and the architecture index are stale; 319 Marksman warnings have one two-part cure |
| [30](./30-completeness-critic.md) | Completeness critic | n/a | n/a | Seven added findings, score reconciliation, twelve root-cause clusters |
| [41](./41-task-tree-target-design.md) | The task tree, as a target design | n/a | n/a | Requirement by requirement: today, target, slice |

Mean of the ten lens scores: **5.3 / 10** (5.3 after the critic's
reconciliation). The mean hides the spread: the task tree scores 2.5 and
everything else sits between 5 and 7.

## The Highs

Fifteen distinct defects, grouped by the owner's question.

**Task tree for roles**

1. `TREEB-02` / `TREEX-01` — the person a node is assigned to has no write
   right over it or anything under it. `server/internal/store/memberships.go:236`
   admits only the project owner or a workspace admin.
2. `TREEX-02` (and `FE-03`) — a node cannot be opened as its own tree.
   `client/lib/router.dart:43` has one roadmap route per project and a node
   card has no tap action.
3. `TREEB-04` / `TREEX-03` — there is no "my tree": no server read scoped to
   the caller's nodes, no filter, no landing.
4. `PARC-01` / `TREEX-04` — the node owner can be set only from the web.
   `server/internal/apiclient/roadmap.go:82` has no owner parameter, and no
   CLI verb or MCP tool lists workspace members.
5. `TREEX-05` — the agent briefing called focus-scoped returns every leaf of
   the project and ignores an unknown focus id
   (`server/internal/agentcontext/agentcontext.go:111`).

**Feature completeness**

6. `FEAT-01` / `PARB-02` — the Build draft call, documented as writing
   nothing, creates a project; for a member who did not author the PRD it
   creates a new unlinked project on every call
   (`server/internal/store/roadmap_plan.go:242`).
7. `FEAT-03` — rebuilding a roadmap deletes every node and, by cascade, every
   evidence row and validation record (`roadmap_plan.go:58`).
8. `FEAT-02` — an agent token is refused the deletion of one node but may
   replace the whole roadmap (`server/internal/api/agent_scope.go:79`).
9. `PARB-01` — the admin approval gate before Build is a disabled web button
   only; REST, CLI and MCP build from a rejected PRD
   (`server/internal/api/roadmap_draft.go:50`).
10. `FE-01` — the coach transcript does not update during a session
    (`client/lib/features/prd/coach_panel.dart:112`).
11. `FE-02` / `PARB-03` — nobody can find a PRD they did not create; no route
    lists PRDs and the project payload carries no PRD id.

**Surface parity**

12. `PARA-02` — `idea update` without `--content` erases the idea body on CLI
    and MCP (`server/internal/apiclient/crud.go:24`).
13. `PARB-04` — `SKILL.md:107` and `README.md:35` promise every verb on every
    surface; the CLI and MCP can only list and create projects.

**Gates**

14. `IDX-02` — `docs/architecture/backend.md` fails its own truth gate on the
    committed tree, so the CI guard step is red on the pushed tip.
15. `COMP-01` — commit `0a87f91` left two more CI jobs red on the pushed tip:
    the expand response-shape test and the analyzer baseline.

`FE-03` restates items 2 and 3 from the web client's side and stays in the
register for traceability.

## Reproduced by running the code

The coordinator ran these. The database rows used a private PostgreSQL 18
cluster on `127.0.0.1` with all migrations applied and throwaway roles; probe
tests were injected with `go test -overlay`, so nothing was written into the
repository.

| # | Finding | Command | Result |
| --- | --- | --- | --- |
| R1 | `TREEB-02` | probe test `TestRv5DelegateProbe` in `internal/store` | Manager assigns node "Payments" to a plain member: accepted. The member then renames it, starts a leaf under it, adds a child, adds evidence, hands it back: each `forbidden`. The member's list call returns all 4 project nodes |
| R2 | `PARC-01` | read `server/internal/mcpserver/product_tools.go:67-70`; skeptic ran `insideout roadmap update --owner x <id>` | The tool takes title, description, status, deadline, completion_reason. The CLI answers `flag provided but not defined: -owner` |
| R3 | `FEAT-01` | probe test `TestRv5BuildPathProbe` | Non-author member, two calls: 2 projects, PRD still unlinked. The author's build afterwards: a third project |
| R4 | `FEAT-03` | same probe | One evidence row, then a rebuild with the identical tree and the confirmed count: 0 evidence rows, the leaf has a new id |
| R5 | `PARA-02` | same probe | A title-only update as the Go client sends it stores an empty body |
| R6 | working tree | `go build ./... && go vet ./... && go test ./...` without a database | Red in 3 packages (`cmd/insideout`, `internal/api`, `internal/apiclient`), all in the uncommitted password-reset work; 25 packages pass |
| R7 | `COMP-01` | `go test ./... -count=1` with the private database | Two more failures in `internal/api`. `TestExpandDraftAndConfirmKeepTheirCapturedShapes` is unmodified and fails on committed `main` since `0a87f91` |
| R8 | `COMP-01` | `flutter analyze --no-fatal-infos && flutter test` | 15 infos against the CI baseline of 14; 576 tests pass, 1 fails (`test/routes_test.dart`, reset-password deep link) |
| R9 | `IDX-02` | `python3 -I scripts/tests/check_docs_truth_rv4.py` | Exit 1, four files not named in `backend.md`; a skeptic read the CI run on `origin/main` and found the same step red |
| R10 | `IDX-05` | Marksman 2026-02-08 driven headlessly over the tree | 321 diagnostics: 319 ambiguous links, 2 false reference links, 0 broken links |
| R11 | `TREEB-07` | probe test `TestRv5RlsProbe` | With the store's Go check bypassed, a plain member's SQL `UPDATE` and `DELETE` on another person's node each affect 1 row: row-level security does not enforce the write rule |

Skeptics ran their own probes for `FEAT-02` (agent token replaced a
three-node roadmap, evidence gone), `PARB-01` (agent built from a rejected
PRD), `TREEX-05` (focus on "Payments" returned the leaves of "Search" too),
`FE-01` (after two turns only the second reply is on screen), `PARB-03` (an
admin read seven list routes and none named the member's PRD), `PARB-04`
(`insideout project delete x` answers `unknown command`), `STRC-02` (a
database URL with two query parameters is written unquoted and then reads as
not set) and `PARA-01` (`forgot-password` with stdin at `/dev/null` sends
nothing and exits 0). The commands are in the verdict table of each lens file.

**Not run:** the Flutter app in a browser, the streaming endpoints live, the
hosted deployment, and anything that needs a model key. Every web walk is
read from code and widget tests.

## Methodology

1. **Scout.** `origin/main` is the product (2762 files; the other branches
   are dependency updates and one old design branch). 14 leads, each cheap one
   run before fan-out.
2. **Fan out.** Ten reviewers, one schema: two on completeness, three on
   surface parity (each built a capability matrix), two on the task tree (each
   filled a property table against the owner's own sentence), two on
   structure, one on the index with the Marksman output. 120 unique findings,
   29 High.
3. **Verify.** Five skeptic clusters, told to refute and allowed to run
   probes on the private database. 20 confirmed, 9 downgraded, 0 refuted.
4. **Reproduce.** Rows R1 to R11 above, by the coordinator.
5. **Critic.** One agent reconciled scores, grouped the findings into twelve
   root causes and added seven findings.
6. **Redact and write.** Secret scan and identifier scan of this folder:
   clean (result recorded in the [changelog](../../changelogs/2026-11-21-focused-review-5.md)).

Sixteen agents, about 2.9 million tokens, 800 tool calls. One scout lead was
corrected (local `main` is two documentation commits behind `origin/main`,
not level).

## Reading order for a fixer

1. [40-remediation-plan.md](./40-remediation-plan.md) — start at row 0.1.
2. [41-task-tree-target-design.md](./41-task-tree-target-design.md) — the
   owner's concept, requirement by requirement, with four decisions to make.
3. [00-executive-summary.md](./00-executive-summary.md) — the top ten and the
   effort table.
4. [01-findings-register.md](./01-findings-register.md) — every finding.

## Related

- Previous review: [2026-10-18 multirole review 4](../2026-10-18-multirole-4/README.md).
- Plan and board row: [2026-11-21 focused review 5](../../plans/2026-11-21-focused-review-5.md).
- Open records from this review: [issues](../../issues/2026-11-21-review-5-open-findings.md).
- Probe method: [practice](../../practices/2026-11-21-probe-a-real-database-without-touching-the-tree.md).
