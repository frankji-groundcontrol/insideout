# 00 · Executive summary

**Repository:** InsideOut
**Branch / commit:** `main` @ `4d186da`, read with its 57 uncommitted files; 0 ahead of and 2 documentation commits behind `origin/main` @ `c2900fd`
**Date:** 2026-11-21
**Mean of the ten lens scores:** **5.3 / 10** (5.3 after the critic's reconciliation)

## Verdict

One person can take an idea to a committed PRD and a roadmap with evidence,
and that path is carefully built: commits are immutable and closed to agent
tokens, the completion rule lives in the store so every surface meets it,
and GitHub evidence is mapped conservatively. The product stops being
complete where a second person arrives. A teammate cannot find a PRD they did
not write, cannot change a node they were handed, and has no tree of their
own; a manager cannot open a branch; an agent cannot be given one. The task
tree for roles, which the owner calls crucial, has its first column
(`owner_user_id`) and little else, and it is written down in no product or
architecture record. Separately, the build-from-PRD path writes before it
checks, and three defects on it were reproduced on a real database.

## Answers to the five questions

| Question | Answer | Score |
| --- | --- | --- |
| 1. Are all functional properties implemented? | No. Of 50 distinct promises in PRODUCT.md, 16 run fully, 19 in part and 15 have no code. Nine of the missing ones are tracked as parked slices; per-Project roles and the task tree are tracked nowhere. The Flutter client covers the single-author loop with complete English and Chinese strings and both themes | 5.0 and 6.2 |
| 2. Are the web app, API, MCP and CLI consistent? | Partly. Sign-in and PRD verbs agree. Projects beyond list and create, members, workspace edits, account deletion, validation, and the node owner and acceptance fields exist on some surfaces only. The published claim of 1:1 parity is false, and no test checks fields across surfaces | 6.2, 5.6, 6.3 |
| 3. Does each role have its own task tree, with delegation and a manager's view? | No. One of eleven properties is implemented on each side. Assigning a node grants nothing (reproduced), no node opens as a tree, there is no "my tree", and delegation is web-only | 2.5 and 2.5 |
| 4. Is the code modular, with long files split? | By file size, yes: one file is over the 350-line rule and it is uncommitted. By responsibility, less so: 33 files sit at 330 to 350 lines, one function is 315 lines, package `main` holds three programs, and setup is implemented three to four times. The line rule itself produces the shaved files | 6.3 and 6.2 |
| 5. Is the documentation index appropriate? | The structure is: 657 of 659 markdown files are reachable from the routers and no link is broken. The content of the routers is stale: the board has no row for the work in flight and the architecture index fails its own gate. Marksman's 319 warnings have two causes and one small cure | 6.2 |

## Scores

| File | Lens | Score | Critic | Why |
| --- | --- | --- | --- | --- |
| [10](./10-feature-product.md) | Completeness: product definition | 5.0 | 5.0 | 51 percent of promises run in some form; three reproduced build-path defects |
| [11](./11-feature-frontend.md) | Completeness: Flutter client | 6.2 | 5.5 | Disciplined states and strings; three confirmed Highs on the team path |
| [12](./12-parity-identity.md) | Parity: identity and workspace | 6.2 | 6.2 | Consistent sign-in; one data-losing difference |
| [13](./13-parity-prd.md) | Parity: project and PRD | 5.6 | 5.0 | All four Highs confirmed, two of them authorization defects |
| [14](./14-parity-roadmap.md) | Parity: roadmap and delivery | 6.3 | 5.5 | The delegation field is unreachable on two of four surfaces |
| [15](./15-tree-model.md) | Task tree: backend | 2.5 | 2.5 | 1 implemented, 4 partial, 6 missing of 11 |
| [16](./16-tree-experience.md) | Task tree: experience | 2.5 | 3.0 | 1 implemented, 6 partial, 4 missing of 11 |
| [20](./20-structure-server.md) | Structure: Go server | 6.3 | 7.0 | No High; debt is in functions and responsibilities, not files |
| [21](./21-structure-client-scripts.md) | Structure: client, scripts, layout | 6.2 | 6.5 | Both Highs downgraded; duplicated setup tooling |
| [22](./22-docs-index.md) | Documentation index | 6.2 | 6.5 | Sound graph, stale routers |

## Top ten by user impact

| # | Finding | Where | Severity | Fix size |
| --- | --- | --- | --- | --- |
| 1 | The person a node is assigned to cannot change it or anything under it (`TREEB-02`; reproduced) | `server/internal/store/memberships.go:236` | High | about 70 lines of store, 1 migration, 150 lines of tests |
| 2 | A node cannot be opened as its own tree, and there is no "my tree" (`TREEX-02`, `TREEX-03`, `TREEB-04`) | `client/lib/router.dart:43`, `server/internal/api/roadmap.go:17` | High | about 500 lines across server, clients and web |
| 3 | The Build draft call creates a project; a non-author member's builds create unlinked projects (`FEAT-01`; reproduced) | `server/internal/store/roadmap_plan.go:242` | High | about 40 lines plus a test |
| 4 | A rebuild erases every evidence row and validation record (`FEAT-03`; reproduced) | `server/internal/store/roadmap_plan.go:58` | High | guard about 30 lines |
| 5 | A title-only idea update from CLI or MCP erases the idea body (`PARA-02`; reproduced) | `server/internal/apiclient/crud.go:24` | High | about 25 lines |
| 6 | The coach transcript shows one reply at a time until the page is reloaded (`FE-01`) | `client/lib/features/prd/coach_panel.dart:112` | High | about 25 lines plus a test |
| 7 | Nobody can find a PRD they did not create, including the admin who must approve it (`FE-02`, `PARB-03`) | `server/internal/api/projects.go:26` | High | about 250 lines |
| 8 | The node owner can be set only from the web; MCP and CLI cannot delegate (`PARC-01`; reproduced) | `server/internal/apiclient/roadmap.go:82` | High | about 180 lines |
| 9 | An agent token may replace the whole roadmap, and may build from a rejected PRD (`FEAT-02`, `PARB-01`) | `server/internal/api/agent_scope.go:79`, `roadmap_draft.go:50` | High | about 45 lines |
| 10 | The pushed tip is red in three CI jobs (`IDX-02`, `COMP-01`; reproduced) | `docs/architecture/backend.md:147`, `testdata/roadmap_expand_confirm.json` | High | under 20 lines |

## What each lens would do first

Shortened from each reviewer's closing paragraph; the full text ends each lens file.

- **Completeness, product:** Write three real-PostgreSQL red tests on the build path, then fix `EnsureProjectForPrd` and the replace guard until they pass.
- **Completeness, client:** Fix the coach transcript: re-read the conversation's messages when a turn ends and show the sent message at once.
- **Parity, identity:** Change the terminal check at `cli_auth.go:55`; today a detached `insideout forgot-password` reports success without sending anything.
- **Parity, PRD:** Add the PRD status check and remove the project insert from the Build draft path.
- **Parity, roadmap:** Write the field-level parity test first, with owner and acceptance failing it, then make it pass.
- **Task tree, backend:** Add the `root={nid}` subtree read; every other property needs a server-defined answer to 'what is this node's tree'.
- **Task tree, experience:** Write the ownership rule down and implement `requireNodeWriter` with a failing-first real-database test.
- **Structure, server:** One registration helper and per-resource MCP tool files; that 315-line function is where the node-owner and subtree tools must be added next.
- **Structure, client and scripts:** Fix the three red contract gates so the uncommitted work can be pushed without turning CI red.
- **Index:** Add the In flight board row and name the missing files in `backend.md`.

## Effort

Estimates are the reviewers' line counts turned into working days for one
person who knows the tree; they are not measured.

| Wave | Rows | Estimate |
| --- | --- | --- |
| 0 · Green tip, slice closed, concept written | 7 | 1 to 2 days, plus the owner's four decisions |
| 1 · Data loss and misplaced writes | 8 | 2 to 3 days |
| 2 · The task tree | 9 | 10 to 15 days |
| 3 · Surface parity | 7 | 6 to 9 days |
| 4 · Structure | 7 | 5 to 8 days, mostly moves |
| 5 · Documentation index | 6 | 2 to 3 days |
| **Total** | **44** | **26 to 40 days** |

## What not to do yet

- Do not build the per-session Focus pointer (parked slice 8) as a stand-in
  for the task tree. It is a pointer, not ownership, and it gives a delegate
  no rights and no tree.
- Do not give a delegate write access (row 2.2) before the rebuild and agent
  guards (rows 1.4 and 1.5) are in: both can delete a delegate's whole
  subtree today.
- Do not add the owner field to MCP and CLI (row 2.3) before deciding whether
  an agent token may change an owner (row 0.6).
- Do not tighten the line budget. Both structure reviewers proposed it and
  the critic showed the count is what produces files trimmed to 349 lines.
- Do not commit the uncommitted slice as it stands: it turns five tests and
  two gates red.
