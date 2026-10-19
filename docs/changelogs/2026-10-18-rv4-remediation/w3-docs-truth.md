# Wave 3, docs truth — review-4 remediation

Branch `rv4/w3-docs-truth`, base `8c74218`: twelve red/fix pairs, two touch-ups
and the fragment commits (the last pair is repair round 1). Findings are cited as
`rv4-<ID>` because review IDs restart per review. Row 3.6 and the README hunks
of row 3.8 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md).
Documents and guard scripts only: no Go, Dart, YAML or compose file changed.
No secret, DSN, address or hosted hostname is spelled here.

## Dispositions

| Finding | Status | Red | Fix | In one line |
| --- | --- | --- | --- | --- |
| rv4-PM-06 | PARTIAL | `8dcda81` | `a1dda3b` | 76 broken links to 42; the 42 sit in unowned dated changelogs, capped per prefix; a patch takes them to 0 |
| rv4-COMP-08 | FIXED | `33ad0f8` | `e34a476` | every PARTIAL promise row links a dated issue; "PostgreSQL 18" in nine sentences |
| COMP-08-DECISION | DEFERRED_OWNER | | | does in_progress to done need evidence or a reason? Owner question below |
| rv4-FULL-09 | FIXED | `b7b6700` | `8e5e039` | five unlisted `INSIDEOUT_*` variables documented |
| rv4-USER-08 | FIXED | `663a48a` | `6db74b9` | the offline coach is said to echo only; the client banner is `w3-first-run-copy`'s |
| rv4-FEAT-08 | FIXED | `3de49d3` | `31703f8` | README names the `insideout.yaml` guide and the GitHub App webhook |
| rv4-PM-08 | PARTIAL | `a5c560e` | `e62b765` | README half done; script and client copy are `w3-first-run-copy`'s |
| rv4-ARCH-06 | FIXED | `f31fe4c` | `2820ba0` | the CLI/MCP/client-config architecture doc; `backend.md` corrected |
| rv4-PM-03 | FIXED | `5ed0652` | `217739a` | HANDOFF and the board show the nine open promise slices |
| rv4-PM-09 | PARTIAL | `df8ba8e` | `30ade18` | the Flutter contradiction is gone and guarded; archiving terminal board rows is outside the owned files |
| rv4-CLOSE-04 | FIXED | `769fa38` | `355ff5a` | ten docs-only ledger rows relabelled Recorded, counts stated and checked |
| rv4-CLOSE-10 | PARTIAL | | | no commit: the sentence lives in an unowned file; patch only |
| R1-06 (rv4-USER-09, README half) | FIXED | `e4dde71` | `791374c` | a residue the open-findings record assigns to this cluster |
| rv4-CLOSE-04 (board, repair 1) | FIXED | `e661895` | `15ed0d8` | the board and the review-3 README restate 63 by code and 10 recorded; a guard arm keeps them equal |

## What changed

- **A relative-link scan, and the 34 live breaks repaired** (rv4-PM-06, red
  `8dcda81`, fix `a1dda3b`). `scripts/tests/check_doc_links.py` reads every
  tracked `.md` file and fails on a relative link whose target is missing. The
  base held 76: 34 outside the two prefixes below (the guard calls them
  "living"; they include dated review records) and 42 inside them. The 34 are
  fixed in ten files: the auth-door QA note, one learning record, two plans,
  the board and five review records. A link to a web-client file that no
  longer exists became a code span, a wrong depth was retargeted, and the
  relative-links learning record's literal broken example was reworded. The
  board's closure claim now says what the scan covers. The 42 are in
  `docs/changelogs/` (24 links, eight files) and `docs/design-system/CHANGELOG.md`
  (18), which this cluster does not own: the guard holds each prefix under a
  ceiling (24, 18), and the dated-links patch repairs them and lowers both to 0.
  That is the only reason this row is PARTIAL.
- **Every PARTIAL promise is a dated issue** (rv4-COMP-08, red `33ad0f8`, fix
  `e34a476`, touch-up `ecc9303`). The
  [promise register](../../review/2026-10-10-multirole-3/50-promise-register.md)
  had 17 PARTIAL rows with no dated issue behind them. `check_product_promise_gaps.py`
  now fails a PARTIAL row whose issue link is missing or names a file that does
  not exist (a `Superseded` row is exempt). Six records are new,
  `docs/issues/2026-10-18-promise-partial-*.md` (change proposal, commit
  baseline tie, completion override reason, outcome time ranges, overdue
  resolution, secondary audiences), each with the gap, the code it touches and
  a bounded fix prompt; eleven rows now point at existing records. The
  PRODUCT.md sentence about the register says the same and stays in its
  original lines (`ecc9303`; the file is already over the line budget, see
  `docs/issues/2026-10-18-product-and-setenv-over-line-budget.md`). The same
  finding's second half: nine sentences in six living docs (README, SETENV, the
  deployment architecture doc, `environment.md`, `installation.md`,
  `local-development.md`) promised "PostgreSQL 14+" while CI starts PostgreSQL
  18 only; they say 18 now. `check_docs_truth.py` reads the tested floor from
  the workflow (`image: postgres:N`) and fails a living doc that claims a lower
  one.
- **Five variables the code reads are listed** (rv4-FULL-09, red `b7b6700`,
  fix `8e5e039`). `INSIDEOUT_AI_TURN_DEADLINE`, `INSIDEOUT_API`,
  `INSIDEOUT_ENV_FILE`, `INSIDEOUT_ROLELESS_URL` and
  `INSIDEOUT_SETUP_REPO_ROOT` had no row in `docs/usage/environment.md`, and
  the server-read one no entry in `.env.example`. Now: 23 lines of rows in the
  reference, a commented entry in `.env.example`, and a paragraph in
  `docs/usage/keys/llm/README.md` naming the turn deadline as the third coach
  knob that has no key-store kind. The guard collects every quoted
  `INSIDEOUT_*` name in non-test Go code and every one a workflow sets (27
  today) and requires each in the reference. `docker-compose.yml` does not pass
  the turn deadline through; the row says so (see Left open).
- **The offline coach is described as it is** (rv4-USER-08, red `663a48a`, fix
  `6db74b9`). The offline streamer (`templateStreamer.StreamChat`) echoes
  the member's last message and does nothing else, yet four living docs
  (`SETENV.md`, `.env.example`, `environment.md`, `local-development.md`)
  called it "end to end" or the "whole PRD flow". Those four, `README.md` and
  `docs/architecture/prd-coach-agent.md` now say it echoes, and
  `local-development.md`'s Offline AI mode section says what real coaching
  needs (a set `INSIDEOUT_LLM_API_KEY`). The fix commit also widens one guard
  match to `set\s+` so a wrapped line still counts. The matching banner in the
  client belongs to `w3-first-run-copy`.
- **GitHub evidence is scoped in the README** (rv4-FEAT-08, red `3de49d3`, fix
  `31703f8`). Evidence attaches only to a roadmap leaf that a committed
  `insideout.yaml` matches, and only once the GitHub App's webhook delivers.
  The README's English and Chinese paragraphs promised it unconditionally; both
  now name the guide and the webhook. One file, 12 lines.
- **A stranger's first run does not dead-end** (rv4-PM-08, red `a5c560e`, fix
  `e62b765`; README half). The CLI install leads with the prebuilt one-liner
  now that `v0.1.0` exists, says "macOS and Linux only" and "Windows is not
  supported" (what `scripts/bootstrap.sh` enforces), names a way to ask for an
  invite (an issue titled "Invite request", without an email address in it, or
  the owner through GitHub as `SECURITY.md` lists), and has a "Try it without
  an account" line that says it needs the checkout and Docker. The hosted
  instance is pointed at through `./scripts/default-api.sh`, so no host is
  spelled. The script and client copy of the same row are `w3-first-run-copy`'s.
- **The README no longer offers `setup --local` to a prebuilt user** (R1-06,
  the README half of rv4-USER-09; red `e4dde71`, fix `791374c`). The command
  block under "After installing" now says the local server needs a repository
  checkout and Docker, which the prebuilt install lacks, and the
  invitation-only note says `promote-admin` creates the account first when the
  address has none (password on stdin) and then promotes it. The open-findings
  record assigns both to "the row's one README commit"; they came as their own
  pair because the PM-08 pair had already landed.
- **The client surface has an architecture doc** (rv4-ARCH-06, red `f31fe4c`,
  fix `2820ba0`).
  [`cli-mcp-and-client-config.md`](../../architecture/cli-mcp-and-client-config.md)
  (166 lines) covers the surfaces, how a CLI verb, an MCP tool and an
  `apiclient` call stay one contract, `clientconfig.Resolve` (base, token and
  refresh token, first set wins), the 401 renewal, the MCP process's two
  clients (`humanClient` and the agent-scoped `client`) and its profile, the
  `<prd_data>` fence, install and release, and the single-replica advisory
  lock. Every statement was read off the code, not off older prose.
  `backend.md` gained 29 API and 18 store file names it never listed, the
  enforced line budget (400, not ~350) and the Bearer-first authentication
  paragraph (it said cookie-only). Linked from `docs/architecture/index.md`,
  `docs/index.md` and `backend.md`.
- **HANDOFF and the board stop saying engineering is closed** (rv4-PM-03, red
  `5ed0652`, fix `217739a`). HANDOFF opens with "Engineering is not closed: nine
  product-promise slices are open", links the
  [recommended order](../../issues/2026-10-11-product-promise-gaps.md) and says
  to open a dated plan first; the [board](../../plans/README.md) has one P1
  Pending row that links the same enumeration.
- **HANDOFF agrees with itself about Flutter** (rv4-PM-09, red `df8ba8e`, fix
  `30ade18`). One section warned that a host runs 3.47.0 against the 3.44.x
  pin while the next-actions section recorded 3.44.8 on the same machine. The
  warning is now a pin statement (3.44.x; any other SDK rewrites
  `client/pubspec.lock`, so never commit it). The register's second half,
  archiving terminal board rows to a dated archive, edits the shared board and
  stays open.
- **The ledger says what it counts** (rv4-CLOSE-04, red `769fa38`, fix `355ff5a`,
  touch-up `ff3d747`). The review-3
  [disposition ledger](../../review/2026-10-10-disposition-ledger/04-multirole-3.md)
  labelled 78 rows Fixed; ten of them had fix commits that changed only `docs/`
  (CQ-02, FEAT-05, FEAT-06, FULL-03, SEC-04, USER-04, COMP-01, COMP-04, PM-03,
  REL-04). They are now Recorded, a correction paragraph says why, and a
  `Counts` line states the tally: 83 rows = 68 Fixed + 1 Fixed (funnel
  rehearsed to the provider) + 10 Recorded + 3 Satisfied by the report + 1
  Parked. REL-05 is the exception to the register's list of eleven: its row
  cited the docs commit `787c8ee`, but the singleton guard itself is code
  commit `a49936a` (`single_replica.go`, `auth_throttle.go`, a test), so the
  row cites that first, with `787c8ee` as the docs half and red `242ef57`, and
  stays Fixed. The rv3 plan record's restated total follows (`ff3d747`).
- **The board and the review-3 README restate the ledger** (rv4-CLOSE-04, repair
  round 1, red `e661895`, fix `15ed0d8`). The verifier found the rv3 row of the
  [board](../../plans/README.md) and the review-3 README's outcome line still
  saying 73 rows closed by code after the ledger said 63 and ten Recorded. Both
  now say 63 by code, 10 recorded, 3 satisfied by the report, 1 parked, 6
  owner-blocked. `check_ledger_honesty.py` gained a third arm: a document it
  lists may not claim more rows closed by code than the table labels Fixed (69)
  and must name the Recorded rows; a superseded figure may be quoted as "first
  count N closed by code" (the plan record does). The same figure in the
  unowned changelog index and the rv3 track file is in the rv3-header patch,
  which also lists both under the arm.
- **The rv3 header counts and the locale default** (rv4-CLOSE-10). The rv3
  changelog index says existing rows "default to `en`"; the migration
  `20261011110000_conversation_locale.sql` says `DEFAULT 'zh'`. The sentence is
  in `docs/changelogs/2026-10-11-rv3-remediation/index.md`, which this cluster
  does not own, so there is no commit: the rv3-header patch fixes it together
  with the index's own "73 closed by code" header.

## Design spikes

The register's two "Direction" rows, and the three calls its rows leave open.

- **PM-03: one board row, not nine.** The direction is a Pending row per
  slice. The enumeration already carries each slice's order and status, and
  nine rows would copy it to a second place that drifts. One P1 Pending row
  links it, HANDOFF links it and states the count, and the guard derives the
  count from the enumeration: a tenth slice fails HANDOFF until it says ten,
  and the demand ends if the enumeration is retired. A slice gets its own row
  when its plan opens.
- **PM-06: scope the claim and add the scan.** Only the first leaves the next
  broken link invisible; only the second leaves "zero broken links"
  unqualified. Both are done. Dated records get a shrinking allowlist kept as a
  per-prefix count (24 and 18), not a list of links, because nobody should edit
  history link by link; a ceiling leaves headroom, so the patch sets both to 0.
- **COMP-08: tracking is checked on PARTIAL rows only.** A DOCS-ONLY or
  Superseded row can be accurate without a dated issue. "PostgreSQL 14+"
  becomes "18", not a range: CI tests 18.6 alone.
- **PM-09: delete the contradiction, defer the archive.** The guard compares
  every Flutter version HANDOFF states with the `ci.yml` pin. Moving terminal
  rows out of the board rewrites a shared file the integrator owns.
- **CLOSE-04: the label must match the diff both ways.** A Fixed row whose
  commits changed only `docs/` fails, and so does a Recorded row whose fix
  changed anything outside it. "Docs-only" means paths under `docs/` (see Left
  open for the root documents that misses).

## The guards

Each is a standalone stdlib script run as `python3 -I scripts/tests/<name>`;
`-I` forbids sibling imports, so each carries its own few-line helpers. They
derive the premise from the tree first and go dormant when it disappears (an
echoing streamer, a guide matcher, a release tag, the enumeration, the CLI's
checkout refusal), so a later change to the code retires a demand instead of
leaving a stale failure. They match `check_*.py`, which the CI job `env-suites`
runs by glob (a shallow checkout that fetches tags; every guard passes on a
one-commit export of this HEAD that carries the tag).

| Guard | Rules (finding) |
| --- | --- |
| `check_doc_links.py` (new, 68 lines) | relative links resolve; dated-record ceilings (PM-06) |
| `check_docs_truth.py` (+PostgreSQL rule) | living docs' PostgreSQL floor matches CI; ceiling for unowned `docs/PLAN.md` and `skills/` (COMP-08) |
| `check_product_promise_gaps.py` (+1 rule) | a PARTIAL register row links an existing dated issue (COMP-08) |
| `check_docs_truth_rv4.py` (new, 285 lines) | variables documented (FULL-09); offline coach (USER-08); GitHub evidence (FEAT-08); first run (PM-08); prebuilt users (R1-06); client surface and `backend.md` facts (ARCH-06); open slices (PM-03); Flutter versions (PM-09) |
| `check_ledger_honesty.py` (new, 130 lines) | the `Counts` line equals the table's tally; every cited fix commit git can see matches its label; the board and records that restate the tally agree with it (CLOSE-04) |

Notes on how they behave:

- **Tag-conditional.** The README ordering rule (PM-08) and the existing
  quick-start rule (USER-05) read `git tag --list 'v*'`; `v0.1.0` exists, and CI
  fetches tags. A scratch repository needs the tag too.
- **Coverage is by glob, not by list.** `backend.md` must name every non-test
  file in `server/internal/store` and `server/internal/api`, a glob token such
  as `ai_*.go` counting for its matches, so the next file added fails until it
  is named.
- **Three arms in the ledger guard.** The counts and restated arms need no git
  and hold on any clone. The history arm asks git about the 65 fix commits the
  ledger cites; they are pre-integration ids that exist only where the review-3
  branches' objects do, so a commit git cannot see is skipped and counted in the
  summary line, never failed. In CI only the git-free arms run. The restated
  arm cannot derive the six owner-closed rows, so it is a ceiling (69).

Mutation sweep: 82 variants (68, then 14 in repair round 1) on scratch exports
and clones of the tree, never the repository, every one returning the expected
exit code. 63 fail as expected: each rule broken in each direction it guards (a
link, a ceiling, a PARTIAL row unlinked, a variable or architecture fact left
out, an offline overclaim, a README condition dropped, HANDOFF's claim or a
Flutter version drifted, a ledger row relabelled either way, the `Counts` line
edited, the board back to 73 or without "10 recorded"). 19 pass by design:
out-of-scope links, dated records, a `Superseded` row, each demand ending with
its premise (offline echo, guide matcher, release tag, retired enumeration), a
ledger on a clone with no history, a board claiming 69 (the restated arm is a
ceiling, not an equality) and a file the arm does not list.

## Verification

Red at the red commit, green at the fix commit, each run with `python3 -I` on
an export of that commit (a scratch repository carrying the `v0.1.0` tag).

| Finding | Red | Green |
| --- | --- | --- |
| rv4-PM-06 | `76 broken relative links (34 in living docs, 42 in dated records) across 583 tracked .md files`, 34 listed (exit 1) | `42 broken relative links (0 in living docs, 42 in dated records) ...` (exit 0, inside the ceilings) |
| rv4-COMP-08 | nine `promises 'PostgreSQL 14+' but CI only starts PostgreSQL 18`; `9 parked slices with live premises, 17 tracking failures` | both guards exit 0; `... 0 tracking failures` |
| rv4-FULL-09 | six messages, e.g. `docs/usage/environment.md does not list INSIDEOUT_AI_TURN_DEADLINE (used at server/internal/agent/coach.go)` | `docs truth (rv4) ok: 27 INSIDEOUT_* variables documented` |
| rv4-USER-08 | six: four docs `calls the offline coach 'end-to-end' ..., but it only echoes the last message`, plus two missing statements | ok |
| rv4-FEAT-08 | four: the README promises evidence without the `insideout.yaml` guide or the webhook, in each language | ok |
| rv4-PM-08 | five: prebuilt not before source; `macOS and Linux only`; `Windows is not supported`; no invite route; no try-out line | ok |
| R1-06 | two: `README offers 'setup --local' without saying it needs a repository checkout`; `promote-admin ... creates the account` | ok |
| rv4-ARCH-06 | 53: three missing links, the missing doc, 18 store and 29 API files unnamed, the 400 budget, the cookie-only claim | ok |
| rv4-PM-03 | four: `says engineering is closed, but 9 product-promise slices are open`, no order link, no count, no board row | ok |
| rv4-PM-09 | `docs/HANDOFF.md states Flutter 3.47.0, but CI pins 3.44.x` | ok |
| rv4-CLOSE-04 | 12: no `Counts` line (the table has 83 rows) and 11 `... is Fixed but its fix commit(s) ... changed only docs/` | `ledger honesty ok: 83 rows, Counts line matches the table; 65 of 65 cited fix commits visible to git, labels match their diffs` |
| rv4-CLOSE-04 (board) | four: the board and the review-3 README `say 73 rows are closed by code, but the ledger labels only 69 Fixed` and omit the 10 Recorded | `... labels match their diffs; 3 restatements agree` |

The full output of each run is in `w3-docs-truth-red-<ID>.txt` and
`w3-docs-truth-green-<ID>.txt` under `/private/tmp/insideout-rv4/reports/`.

By hand, because the guards read text:

- **Link scan.** The cluster gate's scan over `docs/` and the README: base 76
  (plans 13, review 13, design-qa 7, learning 1, changelogs 24, design-system
  18), this branch 42 (changelogs 24, design-system 18). With the three
  patches applied to a clone of this HEAD (re-run in repair round 1): 0, every
  guard green, `scripts/check-redaction.sh` exit 0.
- **Architecture facts.** Each claim in the new doc was read off the code it
  names (`clientconfig.Resolve`, the `apiclient` renewal, `humanClient` and
  `AgentScoped`, `withheldFromAgents`, the fence, `startSingletonGuard`,
  `release.yml`, `bootstrap.sh`, `DefaultBase`, `linebudget`, the Bearer branch).
- **REL-05.** `git show --stat`: `a49936a` (three Go files under
  `server/internal/api`), `787c8ee` (`docs/usage/deployment.md` only).
- **No database, no network.** `needs_db` is false and no port was opened.

**Pending first CI run after push.** Nothing here has run on a runner. The five
guards run there through the `check_*.py` loop; the ledger guard's history arm
will skip the review-3 commits in a shallow checkout and say so in its summary.

## Left open, with owners

1. **Three patches the integrator applies before the gates** (INTEGRATOR TODO,
   pre-gate), in `/private/tmp/insideout-rv4/reports/`. Each passes
   `git apply --check` on this branch alone and all three together:
   - `w3-docs-truth-dated-links.patch` repairs the 42 breaks in eight changelog
     files and `docs/design-system/CHANGELOG.md` and sets both ceilings in
     `check_doc_links.py` to 0 (rv4-PM-06).
   - `w3-docs-truth-pg18-unowned.patch` says "PostgreSQL 18" in `docs/PLAN.md`
     and two `skills/insideout-setup` files and drops the legacy ceiling
     (1 and 3) from `check_docs_truth.py` (rv4-COMP-08).
   - `w3-docs-truth-rv3-header.patch` recomputes the rv3 changelog index's
     header and the rv3 track file's outcome line, corrects the index's locale
     default to `zh` (rv4-CLOSE-10) and lists both files under the ledger
     guard's restated arm (the unowned half of CLOSE-04).
2. **Edits that depend on other wave-3 clusters** (INTEGRATOR TODO, pre-gate; the
   guards fail without them):
   - `docs/architecture/backend.md`: a Layout bullet for
     `server/internal/testlock` (from `w3-test-hardening`, in the form of the
     `linebudget/` bullet at line 34) and a name or glob for any new non-test
     file under `server/internal/store` or `server/internal/api`.
   - `.env.example` and `docs/usage/environment.md`: a row for the auth-network
     hop-count variable and for any other `INSIDEOUT_*` name a cluster adds.
3. **After those clusters land** (INTEGRATOR TODO):
   - `docker-compose.yml`: pass `INSIDEOUT_AI_TURN_DEADLINE:
     ${INSIDEOUT_AI_TURN_DEADLINE:-}` through, then drop the "does not
     interpolate it yet" sentence from its row in `environment.md`.
   - `docs/usage/deployment.md`: behind a proxy the hop count must be set; the
     SSE caps; the health-check deadline.
   - `README.md`: a link to `PRIVACY.md` once row 3.5 writes it.
   - `docs/usage/releasing.md`: signature and checksum verification.
   - `docs/issues/README.md`: index lines for the six `promise-partial-*`
     records; `docs/changelogs/README.md` and `index/2026-10.md` for this folder.
   - HANDOFF and the board: rebase onto this cluster's claims (HANDOFF lines
     13-27 and board row 12 were left to the integrator); if a cluster closes
     or adds a promise slice, update the count there (the guard derives it).
4. **Owner decision (COMP-08-DECISION).** Should moving a roadmap node from
   `in_progress` to `done` require linked evidence or a short reason, recorded
   on the node and enforced server-side in `UpdateRoadmapNode`? Today
   `legalStatusTransitions` allows it with neither, although a `validated`
   status exists. Recommendation: require either. Recorded, with the code it
   touches and a fix prompt, in
   [the completion-override record](../../issues/2026-10-18-promise-partial-completion-override-reason.md).
5. **Observations for whoever touches the same ground.**
   - Five ledger rows are fixed by root-level documents only (ARCH-05, COMP-02,
     FEAT-08, PM-05, TEST-08) and one in ledger 03; "docs-only" means `docs/`,
     so the guard does not catch them. Extending the rule to root `*.md` is
     one line in `check_ledger_honesty.py`; they were not relabelled because
     the register did not list them.
   - DOCS-ONLY register rows (PRODUCT.md lines 320-321, 348 and 379-380) now
     trail the shipped `validated` state and evidence, and "After release:
     compare outcomes" (lines 352-355) has no dated issue; the tracking rule
     covers PARTIAL rows only, so the guard does not see them.
   - `server/internal/api/server.go:20` still says "api doesn't import agent
     until it's wired in P4"; a Go comment, outside this cluster.
   - The release workflow passes only `-X main.version`, so a fork's binaries
     aim at the upstream hosted API (documented in the new architecture doc);
     wiring `DefaultBase` is the owner's decision, rv4-ARCH-05.
6. **Coordinator.** rv4-PM-08 is split with `w3-first-run-copy` (this cluster
   the README, that one the scripts and client copy): one combined disposition.

## Operator notes

No migration, dependency or configuration change; nothing to deploy. The
README now says macOS and Linux only and that the local server needs a
checkout (already true of the installer and the CLI). Contributors get three
new checks: a relative link must resolve, a new `INSIDEOUT_*` variable needs a
row in `environment.md`, a new store or API file needs a name in `backend.md`.
