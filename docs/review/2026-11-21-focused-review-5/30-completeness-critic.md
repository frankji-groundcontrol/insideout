# 30 · Completeness critic

**Bottom line:** The ten reviews cover the five requests well in breadth, but three things fell through. First, commit 0a87f91 left two more CI jobs red on the pushed tip (the server response-shape test and the client analyzer baseline of 14 against 15 infos); the coordinator measured both and one verdict mentions them, yet no finding exists. Second, nobody examined the uncommitted migration 20261021140000_reset_watches.sql: it stays inside the insideout schema, but its rows are never purged, it adds an undocumented entry to the no-RLS list, and no test would notice a table created without RLS. Third, the two top delegation fixes (give a node owner write rights; add the owner field to CLI and MCP) collide with the agent-token wall, which already lets an agent token write ownerId through PATCH and closes the member list to agents. On structure, the 350-line rule is itself the cause of the shaved files, and both structure reviewers proposed tightening the count. Scores cluster between 5.6 and 6.3 for seven reviewers whether they hold zero or four confirmed Highs, so they do not rank the areas. I changed nothing in the repository, read no .env file, and ran no tests myself: test results quoted here are the coordinator's recorded output, and the remaining claims come from reading files and git history.

The critic read every reviewer's bottom line and finding list, every skeptic verdict and the coordinator's measurements. Its added findings carry `COMP-` ids in the [findings register](./01-findings-register.md).

---

This section checks the ten reviews against the five requests, the skeptics' verdicts and the coordinator's measurements. Nothing in the repository was changed. No test was run for this section; test results are the coordinator's recorded output, and the rest comes from reading files and git history.

## What the reviews missed

**The pushed tip is red in three CI jobs, and only one has a finding.** IDX-02 covers the docs truth gate. Commit 0a87f91 also broke two others. The server response-shape test fails because the node view gained `completionReason` and the shared fixture did not. The client analyze step fails because the same commit added a fifteenth analyzer info against a ceiling of 14 in `ci.yml`. Measurements R3 and R4 show both, and one verdict mentions them in passing. Filed here as COMP-01.

**Nobody read the uncommitted migration.** `20261021140000_reset_watches.sql` stays inside the `insideout` schema. Its rows are created by an unauthenticated route and never purged. It joins the tables without row-level security but is not added to the documented list, and no test checks that a new table is either protected or exempted. It has no down path, like the other 49 migrations. Filed as COMP-04.

**Delegation and the agent-token wall collide.** An agent token may already write `ownerId` through the allowed PATCH route, and the member list is closed to agents. Giving a node owner write rights turns that field into a grant of authority, and adding the owner field to the MCP tool cannot work without a member read. This needs a decision before the delegation wave starts (COMP-03).

**No test would notice a response field drifting.** The one shared fixture was captured from nodes with no owner and no deadline, the Dart model reads every field leniently, and the CLI and MCP pass raw JSON through (COMP-02).

**The 350-line rule causes the shaved files.** 33 of 917 source files sit between 330 and 350 lines and one sits above. A security comment was removed from `ci.yml` to keep it at the limit, and 15 of 84 issue records are about line counts, some for prose. Both structure reviewers proposed tightening the count; the rule should instead measure responsibility and function length, for source code only (COMP-05).

Two smaller items: the `dropped` mail word can reveal that an account exists (COMP-06), and the untracked-file manifest that HANDOFF.md cites does not exist in this clone (COMP-07).

Four claims contradict the tree and were not filed: README.md:35-37 ("every verb in the terminal"), HANDOFF.md:111-112 (the manifest), SKILL.md:82 ("Evidence never auto-proves outcomes", while any evidence row closes a node), and the real-verification rule against a tip that has been red since 2026-10-20.

## Scores

Seven reviewers scored between 5.6 and 6.3 whether they held no High or four confirmed Highs, so the scores do not rank the areas. Suggested values follow each reviewer's own findings after verification.

| Reviewer | Given | Suggested | Reason |
|---|---|---|---|
| feature-product | 5.0 | 5.0 | 51 percent of 50 promises; three Highs reproduced |
| feature-frontend | 6.2 | 5.5 | three confirmed Highs in core flows |
| parity-identity | 6.2 | 6.2 | one High stands, one downgraded |
| parity-prd | 5.6 | 5.0 | four confirmed Highs, two shared with feature-product |
| parity-roadmap | 6.3 | 5.5 | delegation field unreachable on two surfaces; PARC-05 under-rated |
| tree-model | 2.5 | 2.5 | 27 percent of eleven properties |
| tree-experience | 2.5 | 3.0 | its own checks give 36 percent |
| structure-server | 6.3 | 7.0 | no High; clean package graph |
| structure-client-scripts | 6.2 | 6.5 | both Highs downgraded |
| docs-index | 6.2 | 6.5 | structure sound; one High stands |

## Verdicts that disagree

- Four pairs of findings name each other as the duplicate (FEAT-01 and PARB-02, TREEB-02 and TREEX-01, TREEX-04 and PARC-01, FE-02 and PARB-03). Keep FEAT-01, TREEB-02, PARC-01 and PARB-03.
- TREEB-03 was downgraded because a client can cut a subtree, while TREEX-05 and TREEB-04 are High for the same missing server function.
- STRC-01 is marked a duplicate of IDX-02, but two of its three gates fail only in the uncommitted slice.
- FE-04 calls project hand-over not required, yet measurement R1 shows it is the only delegation that grants write today.

## Root causes, in remediation order

1. The pushed tip is red and nothing blocks on it (IDX-02, COMP-01).
2. The build-from-PRD path writes before it checks (9 findings).
3. Plan-write authority has no node owner in it (15).
4. A PRD has no address a second person can find (3).
5. Every verb is written three times with no field-level contract test (27).
6. The tree has one read and one route (13).
7. The reset and install slice is uncommitted, unplanned and red (9).
8. Flutter pages hold fetch, stream and layout in one class (6).
9. Validation and error mapping are repeated per handler (5).
10. Open, promised and done records are kept by hand with no check (19).
11. Setup tooling does the same job several times (7).
12. Modularity is measured as a line count (6).

Cluster 3 must land before cluster 6, and the owner-field part of cluster 5 waits on the COMP-03 decision.

## Not covered by any reviewer

- The GitHub evidence loop, mail, export and presence packages as features.
- The Flutter app run live, and the iOS, Android and web shells.
- Response shapes across surfaces.
- Row-level security on roadmap nodes, probed rather than read.
- The migrations folder and the uncommitted setup packages (`flow`, `probe`).
- Deployment files, git hooks and the four workflows other than `ci.yml`.
- The indexes of `docs/learning`, `docs/practices`, `docs/history` and `docs/design-system`.

## What no reviewer covered

- Request 1 (feature completeness): the GitHub evidence loop, mail delivery, export and live collaboration packages were never read as features or run. server/internal/mail, export, presence, progresshub, prdcommit and config have zero mentions in the full reviewer output; github has 5 and guide 1. First read: server/internal/github/github.go and server/internal/api/github.go (webhook and sync), then server/internal/mail/mail.go.
- Request 1: the mobile and web shells (client/ios, client/android, client/web, 943 tracked files) and the Flutter app itself were not run by any reviewer; every web walk is from code. First step: flutter run -d chrome against a local server and walk capture, coach, build, roadmap as two users.
- Request 2 (interface consistency): only request bodies were compared across surfaces. Nobody compared response shapes (Go view structs in server/internal/api against client/lib/api/models_*.dart and what the CLI and MCP pass through). First read: roadmapNodeView at server/internal/api/roadmap.go:30 beside client/lib/api/models_roadmap.dart:50-65.
- Request 2: the insideout-mcp binary entry point (server/cmd/insideout-mcp, zero mentions) and the SSE streams were not exercised live on any surface, although CLAUDE.md requires streaming endpoints to be exercised live. First read: server/cmd/insideout-mcp/main.go, then curl the progress stream on a local server.
- Request 3 (task tree): row-level security on roadmap_nodes was read from migrations and never probed (TREEB-07 has no verdict). First check: add a plain-member UPDATE and DELETE on roadmap_nodes to the existing RLS write walk and run it on the private PostgreSQL cluster.
- Request 3: no reviewer looked for an existing design of the tree or delegation in DESIGN.md, docs/design-system (25 files, zero mentions) or docs/design-qa. First read: docs/design-system/README.md and docs/design-qa/README.md.
- Request 4 (code and file structure): server/db/migrations (50 files, forward-only, no down path in any file) and the uncommitted migration 20261021140000_reset_watches.sql were not reviewed by anyone. First read: that file, then docs/architecture/database-and-rls.md:72-79.
- Request 4: the uncommitted setup packages server/internal/setup/flow and server/internal/setup/probe have zero mentions, and setup/ui only its line count. First read: server/internal/setup/flow/flow.go (261 lines) and its two test files.
- Request 4: deployment and guardrail files were not reviewed: docker-compose.yml, server/Dockerfile, client/Dockerfile, client/nginx.conf, docker/postgres-init, config/git-hooks, and the workflows other than ci.yml (release, security, smoke, ubuntu-install). First read: config/git-hooks/pre-commit and commit-msg, since CLAUDE.md calls that guardrail active.
- Request 4: the 351 Go test files are exempt from the server line guard and no reviewer measured them. I measured: none over 350 lines, six between 330 and 350. Function length inside tests was not measured by anyone.
- Request 5 (documentation index): reachability was computed by script, but the indexes of docs/learning (70 files, one mention), docs/practices, docs/history and docs/design-system were not read for whether their grouping suits a reader. Fragment anchors were not checked with a GitHub-exact slugger. First read: docs/learning/README.md.

## Published claims that contradict the code and that nobody filed

- README.md:35-37 says the CLI offers 'every verb in the terminal' and the MCP server 'the same verbs as MCP tools'. The Go client has no method for reading, editing or deleting a project, for project updates, workspace edits or members, and the CLI answers 'unknown command' to each (the PARB-04 verdict ran them). Reviewers filed this claim against SKILL.md:107 only; the README carries the same promise on the repository's front page, in both languages (the Chinese line is README.md:31).
- docs/HANDOFF.md:111-112 says untracked '.grok/ and .qoder/' are 'listed in the local manifest .git/info/untracked-manifest.tsv'. In this clone .git/info holds only 'exclude' and 'refs'; the manifest file does not exist, .qoder/ does not exist, and 23 untracked paths are present. CLAUDE.md makes the manifest a hard privacy rule, and docs/practices/2026-10-09-sync-two-macs-and-github.md:27 repeats it.
- SKILL.md:82 (and docs/usage/cli-mcp-and-guidance.md:243) says 'Evidence never auto-proves outcomes — you judge it.' server/internal/store/roadmap_completion.go:35 reads 'if evidenced && reason == ""' and returns nil: any single evidence row, including a failed deployment, lets a node move to done with no reason. FEAT-07 filed the behaviour but nobody filed the sentence agents are told to rely on.
- CLAUDE.md hard rule 'Real verification' and AGENTS.md:33-35 say DB-dependent work is not done until the DATABASE_URL-gated tests pass. Commit 0a87f91 is on origin/main with TestExpandDraftAndConfirmKeepTheirCapturedShapes red (coordinator measurement R3) and with the analyzer at 15 infos against the CI ceiling of 14 at .github/workflows/ci.yml:260. docs/HANDOFF.md still describes the last red run as 'red on env-suites only'.

## Score reconciliation

| Lens | Given | Suggested | Why |
| --- | --- | --- | --- |
| feature-product | 5 | 5 | Agrees. Its concept checks give 16 implemented, 19 partial and 15 missing of 50, which is 51 percent, and three of its four Highs were reproduced on the private database. |
| feature-frontend | 6.2 | 5.5 | Too high for its own findings. Three Highs were confirmed (coach transcript does not update, an approving admin cannot reach a PRD, no per-person tree), and the one concept row it marks missing outright is the capability the owner calls crucial. The 73 percent from its concept checks weighs a locale toggle the same as the coach chat. |
| parity-identity | 6.2 | 6.2 | Agrees. One High stands (title-only idea update erases the body), the other was downgraded to Medium, and most of its 14 matrix gaps are CLI absences without data loss. |
| parity-prd | 5.6 | 5 | Slightly high. All four Highs were confirmed, two of them authorization defects that feature-product also reported and scored at 5. The two reviewers should not differ by 0.6 on the same defects. |
| parity-roadmap | 6.3 | 5.5 | Too high. It is the highest score in the review for the domain where the delegation field is unreachable on two of four surfaces (PARC-01 confirmed High, measurement R2) and where it found no drift test. PARC-05 is rated Medium, but reading roadmap.go:291-294 and apiclient/roadmap.go:108-112 confirms that a position-only move lifts a nested node and its subtree to the root, which is wrong behaviour on an advertised path. Nothing was run. |
| tree-model | 2.5 | 2.5 | Agrees. One implemented, four partial and six missing of eleven is 27 percent. Two of its Highs were downgraded on severity, not on fact. |
| tree-experience | 2.5 | 3 | Close, but its own concept checks (one implemented, six partial, four missing) give 36 percent, not the 'one quarter' in its bottom line. Either the MCP and CLI rows should read missing, since they can only show the owner, or the score is 3. All five Highs were confirmed. |
| structure-server | 6.3 | 7 | Too low against its own findings and against its peers. It has no High, nine Medium and three Low, an acyclic package graph and every committed file under the guard, yet it scores the same as parity-roadmap with a confirmed High. |
| structure-client-scripts | 6.2 | 6.5 | Slightly low after verification. Both of its Highs were downgraded to Medium: two of the three red gates fail only against uncommitted files, and the setup.sh quoting defect fails closed. |
| docs-index | 6.2 | 6.5 | Slightly low for the question asked. The index structure is sound (657 of 659 files reachable, no broken link, and measurement R6 shows 319 of 321 Marksman diagnostics are link ambiguity). One High stands (the truth gate is red on the pushed tip); the other was downgraded. |

## Verdicts that disagree with a measurement or with each other

- Measurement R4 against the IDX-02 verdict: R4 records 'flutter analyze: 15 infos' as a neutral fact, while the IDX-02 verdict says the CI client analyze step is red on c2900fd. Both are true and they are the same thing: .github/workflows/ci.yml:260 sets INFO_BASELINE to 14, and the fifteenth info is client/lib/api/requests.dart:101, added by commit 0a87f91. No finding covers it. The same verdict mentions the red server test step, which is R3's committed-main failure, also unfiled (see COMP-01).
- Circular duplicate pointers: FEAT-01 and PARB-02 name each other as the canonical finding, as do TREEB-02 and TREEX-01, TREEX-04 and PARC-01, and FE-02 and PARB-03. A de-duplication that drops every finding with duplicate_of set drops all eight. Keep FEAT-01, TREEB-02, PARC-01 and PARB-03 as canonical.
- PARC-02 (downgraded to Medium, duplicate of TREEX-04) against TREEX-04 (confirmed High, duplicate of PARC-01): one chain of three findings closed by the same one-line client change carries two severities, and measurement R2 shows owner and acceptance missing in exactly the same way. Report one High for the owner field and note acceptance as its Medium rider.
- TREEB-03 (downgraded to Medium because a client can cut a subtree from the full list) against TREEX-05 and TREEB-04 (both confirmed High): all three are the same missing server function, the descendants of one node. TREEX-05 is High precisely because the server does not scope by subtree for an agent, and CLI and MCP callers have no client to do the cut. Measurement R1 shows the delegate's read returns the sibling branch.
- STRC-01 is marked duplicate of IDX-02, but only one of its three red gates (check_docs_truth_rv4.py) shares that cause. The other two (check_first_run_copy.py, check_supply_chain.py) fail only against the uncommitted installer and go.mod and belong with PARA-01 and PARA-08. Following the pointer would lose them from the slice's closing checklist.
- The TREEB-01 downgrade reasons that a delegate with write rights 'could pick the manager from the member list'. The PARC-01 verdict establishes that no member listing exists in the Go client, the CLI or the MCP, and server/internal/api/agent_scope.go:122 makes the member list human-only. Hand-back without a stored delegator works on the web only.
- FE-04 (downgraded to Medium, project hand-over 'not owner-required') against TREEB-02 and measurement R1: R1 shows that being project owner or workspace admin is the only thing that grants roadmap write. Until node authority exists, handing over the project is the only delegation that works, and FE-04's own probe (steps B and C) shows exactly that. The Medium is defensible only once TREEB-02 is fixed.
- The IDX-01 verdict says of the uncommitted slice 'No runtime behaviour is wrong'. The PARA-01 verdict reproduced wrong behaviour in the same slice (forgot-password with a detached stdin sends nothing and exits 0), and R3 and R4 list five red tests in it. The sentence is about the missing board row and should not be carried into the report as a statement about the slice.
- Severity differs on one root cause between a verdict and an unverified sibling: FEAT-03 is confirmed High for the rebuild cascade while TREEB-10 reports the same cascade at Medium; TREEX-05 is confirmed High while TREEB-06 reports the same unscoped agent context at Medium. The report should carry the verified severity for both.

## Root-cause clusters

| Cluster | Findings | The one change |
| --- | --- | --- |
| 1. The pushed tip is red and nothing blocks on it | IDX-02 | One small commit that makes origin/main green: name the three files in docs/architecture/backend.md, add completionReason to server/internal/api/testdata/roadmap_expand_confirm.json, and clear the fifteenth analyzer info. Then record in HANDOFF that a red ci run blocks the next checkpoint. Closes COMP-01 as well. Smallest effort, and every later wave needs a green baseline to verify against. |
| 2. The build-from-PRD path writes before it checks | FEAT-01, PARB-02, PARB-01, FEAT-02, FEAT-03, TREEB-10, FEAT-06, FEAT-09, FE-09 | Move every write of the build path into the confirm call, in one transaction that first checks author-or-admin and the PRD status, creates the project only there, refuses an agent token on a replace, and reconciles nodes by id so evidence survives. The web shows the draft before confirm. |
| 3. Plan-write authority is one project-level rule with no node owner in it | TREEB-02, TREEX-01, TREEB-01, TREEB-05, TREEB-07, TREEB-09, TREEB-12, TREEX-07, TREEX-08, FEAT-05, FEAT-12, FE-04, FE-05, PARB-08, STRS-05 | One authorization file in the store with a single node-write predicate (project owner, workspace admin, or owner of the node or an ancestor), used by all ten roadmap and proposal writes, mirrored in the row-level-security policies, and returned to clients as a canWrite flag so controls are hidden. Decide COMP-03 (who may change ownerId) in the same change. |
| 4. A PRD and a feed page have no address a second person can find | FE-02, PARB-03, PARB-07 | Add a workspace PRD list route (and prdId on the project view), accept the cursor the updates feed already emits, and surface both on web, CLI and MCP. |
| 5. Every verb is written three times with no shared definition and no field-level contract test | PARC-01, TREEX-04, PARC-02, PARC-03, PARC-04, PARC-05, PARC-06, PARC-07, PARC-09, PARC-11, PARC-12, PARA-02, PARA-03, PARA-04, PARA-06, PARA-07, PARA-09, PARA-11, PARB-04, PARB-05, PARB-06, PARB-09, PARB-11, PARB-12, FE-07, STRS-02, STRS-06 | One request struct per verb, shared by the Go client, the CLI flags and the MCP schema, with absent-versus-empty semantics stated once, plus one test that walks the JSON tags of every API request struct and fails unless each is reachable on each surface or listed as a deliberate omission. Then correct README.md:35 and SKILL.md:107 to what the test proves. |
| 6. The tree has one read (the whole project) and one route | TREEB-03, TREEB-04, TREEB-06, TREEB-08, TREEX-02, TREEX-03, TREEX-05, TREEX-06, TREEX-09, TREEX-10, TREEX-11, FE-03, FE-11 | One server subtree function (root node id, or owner equals caller) with roll-up computed in it, exposed as a node read and a my-nodes read, reused by agent context for focus, and one web route that renders the same roadmap page rooted at a node with a breadcrumb. Depends on cluster 3. |
| 7. The reset and install slice is uncommitted, unplanned and red | PARA-01, PARA-05, PARA-08, PARA-10, FE-06, IDX-01, IDX-04, IDX-09, STRC-01 | Open one dated plan and board row for the slice, fast-forward to origin/main first, then close its checklist: the five red tests, the two red gates, the reset-page test, and COMP-04 and COMP-06, before the checkpoint commit. |
| 8. Flutter pages hold fetch, stream, state and layout in one State class | FE-01, FE-08, FE-10, STRC-05, STRC-08, STRC-10 | A controller per page between the State class and the API client, starting with the coach transcript (one message list that send and stream both append to), with typed models in place of untyped maps. |
| 9. Validation and error mapping are re-implemented in each handler | PARC-08, PARC-10, PARB-10, PARA-12, STRS-11 | One store-error-to-status helper with one code style, and one field-bounds helper, applied across server/internal/api so every route answers the same error for the same cause. |
| 10. The records that say what is open, promised or done are kept by hand with no committed check | FEAT-04, FEAT-07, FEAT-08, FEAT-10, FEAT-11, TREEB-11, TREEX-12, IDX-03, IDX-05, IDX-06, IDX-07, IDX-08, IDX-10, IDX-11, IDX-12, FE-12, STRS-07, STRS-12, STRC-07 | One committed docs-state gate in CI: orphan and index-against-folder checks, a .marksman.toml with the leading ./ link rule, an open-issues table generated from record front matter, and a rule that every promise-register row not marked implemented links a dated issue. Write the task-tree concept into PRODUCT.md and one architecture page as the first record it tracks. |
| 11. Setup and environment tooling implements the same job several times | STRC-02, STRC-03, STRC-04, STRC-09, STRC-11, STRC-12, STRS-03 | Make the Go wizard and server/internal/setup the only .env writer and placeholder rule; move the cli_setup engine out of package main into internal/setup; reduce scripts/setup.sh and the installers to callers of the binary. |
| 12. Modularity is measured as a line count | STRS-01, STRS-04, STRS-08, STRS-09, STRS-10, STRC-06 | Rewrite the rule in CLAUDE.md and AGENTS.md as a responsibility rule scoped to source code with a function-length guard (about 80 lines) beside one file threshold, and stop applying it to YAML and prose (see COMP-05). Then split the sixteen long functions by concept. |
