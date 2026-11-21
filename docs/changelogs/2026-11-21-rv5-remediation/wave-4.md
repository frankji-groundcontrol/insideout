# Wave 4 · Structure

Rows 4.1 to 4.7 of the
[remediation plan](../../review/2026-11-21-focused-review-5/40-remediation-plan.md),
plus the wave 3 follow-ups that the verifiers left as should-fix. From this
wave on, the coordinator ran the clusters directly instead of through a fixed
build, verify and repair workflow
([plan](../../plans/2026-11-21-rv5-remediation/2026-11-21-rv5-remediation.md)).

- **What could run beside wave 3 started early.** The coordinator mapped each
  cluster's files against wave 3's. Setup-engine and rule-guards started
  beside wave 3. Agent-split started on wave 3's server branch once it was
  accepted, and the server half of the renames started beside wave 3's CLI,
  MCP and web stages.
- **The rest started together.** The MCP split, the CLI split, the web
  controllers and the wave 3 follow-ups started at once from one green base.
- **Review and gates.** The coordinator read each report and diff, fixed small
  findings itself, and ran the full gates once per integration.

## Rows

| Row | IDs | Disposition | Commits |
| --- | --- | --- | --- |
| 4.1 | COMP-05, STRS-08, STRC-06 | FIXED: CLAUDE.md and AGENTS.md set a responsibility rule for source code (one responsibility per file; functions under about 80 lines; one file threshold; no line rule for workflows, YAML or prose). [`config/code-budget.json`](../../../config/code-budget.json) holds both thresholds for the Go and Dart guards (the Go file budget drops from 400 to 350). A new function-length guard lists today's long functions; a listed function may not grow, and an entry fails once its function is cured, so the list only shrinks. After the wave 4 splits, 8 remain ([record](../../issues/2026-11-21-long-functions.md)). Four line-budget records closed as not applicable | `68d284a1`, `a7f1902c`, `ce35aa45`, `8aae8aad`, `23eca01d` |
| 4.2 | STRS-01, STRS-02 | FIXED: one MCP registration helper (`register.go`, the only place a tool takes the call lock) and per-resource `tools_*.go` files, with every function under 80 lines. A golden of the registry, with and without a login, stayed byte-identical through the moves. `roadmap_update` no longer offers `owner_id` or `completion_reason`, which an MCP agent token can never set. Its description sends an owner change to `propose`; a proposal's items can only add nodes, so that proposal is free text a human acts on. The coach turn is split into open, run and close; the Anthropic parser into event handlers | `be69c826`, `daa6f493`, `1263c3db`, `6d234afe`, `b97f3323`, `f54df7ea` |
| 4.3 | STRS-03, STRC-02, STRC-03, STRC-11, STRC-09 | FIXED: the local-setup engine moved from package `main` into `server/internal/setup/local` and `runLocalSetup` was split into steps. `scripts/setup.sh` is a shim for `insideout setup` and no longer writes `.env` itself. `keys.py` takes the placeholder rule and `.env` parser from `env_catalog.py`, so the skeleton's `change_me` JWT secret is no longer copied as a real value. The env suites moved to `scripts/tests/env/` and the nginx checks to `scripts/checks/`. A check keeps the two installers' PATH blocks equal | `91f2f1c2`, `991abcf8`, `67323304`, `03279496`, `6710bd36`, `4a3135a8`, `143640a0` |
| 4.3 | STRC-04 | PARTIAL: a test holds every host's key manifests to the variables `.env.example` declares; moving to one shared manifest set is [recorded](../../issues/2026-11-21-key-manifests-one-set.md) | `1eee288d` |
| 4.4 | STRS-09, STRS-04, STRS-10 | FIXED: the provider clients form the leaf package `server/internal/llm`, which imports nothing internal. The critic is grouped by concern. `store/agent_messages.go` is split into messages, AI runs, operations and retention. `github.go` is split into URL parsing, commit fetching and the visibility probe; `roadmap_planner.go` into the contract, tree assembly and one file per planner. `cli.go` is split into the interrupt runner, dispatcher and saved-login files, every CLI file is named for its verbs, and the long verb functions are split | `6180c695`, `f6465d2c`, `e6e2d598`, `ae07dc5f`, `8c37c90c`, `e92ec8ff`, `a7a94b6f`, `ffe97efd` |
| 4.5 | STRC-05, STRC-08 | FIXED: `RoadmapController` owns the roadmap page's data and live streams for the whole tree, an opened subtree and the Mine view. The roadmap, evidence and validation verbs moved to `client_roadmap.dart`. `getProject` and `listNodeEvidence` return typed models, checked against captured server responses | `be521407`, `46c58a5e` |
| 4.6 | STRS-07, STRC-07, STRC-10 | FIXED: every test file named after review 4, a coverage goal or a workstream is renamed for what it pins, in every Go package and the client, and test files that pinned two unrelated things are split. Test-name lists were compared before and after. The five review-4 gates are named by subject and share `scripts/tests/_gate.py`. `client/test` mirrors `lib/` | `d0500887`, `4fe24af8`, `f4b5e797`, `0058bb99`, `6970fef8`, `700a9e99`, `6df12229`, `a748de49` |
| 4.7 | STRS-12, STRC-12 | FIXED: four functions with no production caller are deleted (`splitCORSOrigins`, `UpdateConversationStage`, `hasOpenBlockingFindings`, `apiclient.Cursor`); `.grok/` and `.zcode/` are ignored | `cc743226`, `dcb789b6`, `de9fc822` |

### Wave 3 follow-ups landed in this wave

| IDs | Change | Commits |
| --- | --- | --- |
| FE-05 | Every workspace view carries `canEdit` and `canDelete`. The web shows Delete only to the creator, and a member sees the settings read-only | `e1774704`, `d1ad8a56`, `fdc92a89`, `8e0b614b` |
| FE-02 | The PRD list re-reads with the workspace page's refresh | `fd679e99`, `27ec5b48` |
| PARA-05 | The forgot-password page says when the server only logged the mail | `f0cdecbc`, `2bf79c8f` |
| PARB-10 | Each unresolved commit item, the revision note and the project description are bounded at 2000 characters | `ffae64b7`, `e6f419aa` |
| PARB-07 | Loaded older updates survive a parent rebuild | `8a6f6ac4`, `5ce10a06` |
| Docs | `PRIVACY.md` names both account-deletion paths; the updates-feed paragraph states the cursor's 400 and 404; `environment.md` drops the stale line-budget reason | `e7aadab9` |

## Gates on the integrated tree

- `go build`, `go vet`, `gofmt`: clean.
- `go test ./... -count=1` without a database and on a freshly migrated private
  PostgreSQL 18 database: every package ok, no skipped tests.
- `flutter analyze`: 8 infos (baseline 8); `dart format`: no change;
  `flutter test`: 666 passed.
- Every `scripts/tests/check_*.py` and `scripts/check-redaction.sh`: exit 0.

## Left open

- Eight functions remain on the function guard's list.
- STRC-04: moving the key manifests to one shared set.
- The write verbs that used to share one function (`ideas`, `idea update`,
  `evidence`, `join`, `workspaces create`, `projects create`, `prd update`)
  still accept the old combined flag set, so existing command lines keep
  parsing. Narrowing each verb to its own flags would be a behaviour change.
- `insideout setup --hosted` prints the Go wizard's text, which does not show
  the API URL the old script printed.
- `parseResponsesStream` (98 lines) is still listed in the function guard.
