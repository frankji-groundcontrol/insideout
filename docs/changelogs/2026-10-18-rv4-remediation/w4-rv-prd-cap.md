# Wave 4, whole-change review: the PRD section cap — review-4 remediation

Branch `rv4/w4-rv-prd-cap`, `582a4bd..a881cff` (five commits: three red tests,
two fixes), then this fragment. Findings are cited as `rv4-<ID>`. Three Medium
findings of the whole-change review that follows wave 3 (the "After wave 3"
section of the [execution contract](../../review/2026-10-18-multirole-4/41-remediation-contract.md)),
`rv4-FR-F85`, `rv4-FR-F95` and `rv4-FR-F127`, share one root cause. All three are
`FIXED`. No migration, route or dependency; no address, host or secret is
spelled here.

## Root cause

The 10,000-character PRD section cap (rv4-COMP-04, see
[w3-stream-bounds](w3-stream-bounds.md)) was enforced by one of the two callers
of `store.UpdateSections`, the only writer of `prds.sections`. The human
`PATCH /prds/{id}` handler checked each section before calling the store; the
coach's `update_prd_section` tool did not. Three symptoms followed:

- **F85**: the coach could store a section of any length.
- **F95**: the coach, critic and roadmap planner prompts clip a section at
  10,000 characters, so a long section the coach had written reached its next
  prompt cut (`...[truncated: 500 more characters]`) while `get_prd` returned it
  whole. The comment on the clip said it only bites rows stored before the cap;
  that was false.
- **F127**: the PRD page sends all eight sections back on every save. One
  section over the cap, however it got there, made every save of that PRD answer
  400 (`section background is too long (max 10000 characters)`) and lose the
  user's edit.

## What changed

- **One cap, in the one writer** (rv4-FR-F85, red `a602548`, fix `a18fdfe`).
  `store/prds.go` gains `MaxSectionRunes` (10,000, counted in characters: 10,000
  three-byte CJK characters are 30,000 bytes and pass) and the typed
  `SectionTooLongError{Section}`, which is an `ErrValidation`, so a caller that
  maps only the class still answers 400. `UpdateSections` checks the patch after
  the CAS check and before the merge, keys in sorted order so the section named
  does not depend on map order. A refused call stores nothing: not the other
  sections, not the title. The coach tool turns the error into the text the
  model reads (the coach loop hands a tool error to the model as `error: <text>`):
  "agent: section background is too long (max 10000 characters); you sent 10001
  characters. Shorten the section, moving detail that belongs elsewhere into the
  section it fits, then call update_prd_section again".
- **The prompt clip follows the constant, and its comment is true**
  (rv4-FR-F95, red `d5b86d7`, fix `a18fdfe`, the same commit as F85: the write
  cap is the fix). `maxPromptSectionRunes` is `store.MaxSectionRunes`, so the two
  cannot drift. The clip stays for rows stored before the cap; the comment now
  says that, and why `get_prd` returns such a row whole. Red: a 10,500-character
  section written through the tool reached the coach, critic and planner prompts
  cut. After the fix the write is refused, so no prompt meets a cut section the
  coach wrote.
- **A save checks only the text it changes** (rv4-FR-F127, red `0f03907`, fix
  `a881cff`). The store compares each patched section with the text it holds,
  read in the same transaction, and applies the cap only to a section whose text
  differs. The page's whole-PRD save therefore works around a long section it did
  not touch. The handler's own check (`sectionLengthError`, `maxPrdSectionRunes`,
  the whole of `api/prd_section_cap.go`) is deleted; the handler maps
  `*store.SectionTooLongError` to the same 400 body, so the web client is
  unchanged. One constant, one check.

## Design notes

- **Where the check lives.** In the store, because every writer passes there. A
  second copy in `agent/tools.go` would have closed F85 and left the next writer
  (a new tool, a new route) to forget it, which is how the cap came to bind one
  caller only.
- **Unchanged text is exempt.** The first version of the store check (`a18fdfe`)
  refused any patch that carried an over-cap section. That closed F85 and F95 and
  moved F127's failure from the handler into the store. `a881cff` makes the rule
  exact string equality with the stored text: no legacy flag, no migration, no
  client change. Changing a long section by one character, longer or shorter, is
  a write like any other and has to land at or under the cap, after which the row
  is clean for good.
- **`get_prd` is not clipped (not taken).** The coach can shorten a legacy
  section faithfully only if it reads all of it; a clipped read would invite it
  to rewrite the visible prefix and drop the rest. The prompts still show the cut
  with its marker, so the model knows the section is long.
  `TestRvPrdCapCoachCanShortenAnOversizeStoredSection` pins the path: `get_prd`
  returns a 15,000-character row whole, a rewrite to 12,000 is refused, a rewrite
  to 9,000 lands, and the prompt then carries it uncut. This is the second half of
  R3-05 and it stays unfixed on purpose.
- **The tool description is unchanged.** The refusal carries the limit and the
  way out; naming the limit in the description would add tokens to every turn for
  a refusal the model meets rarely.

| Constant | Value | Where |
| --- | --- | --- |
| `store.MaxSectionRunes` | 10,000 characters | `store/prds.go`, enforced by `UpdateSections` |
| `maxPromptSectionRunes` | `store.MaxSectionRunes` | `agent/prompts.go`, the clip for rows stored before the cap |
| `maxPrdSectionRunes` | removed | was in `api/prd_section_cap.go`, now deleted |

## Behaviour to know

- **Authorization answers before the length.** The handler used to refuse an
  over-cap body with 400 before it touched the store, so any signed-in user got
  the length error. The store now authorizes first: a stranger gets the 403 or
  404 that any other save gets, and only an author or admin reaches the 400. An
  over-cap save costs one read transaction before the refusal.
- **The 400 text is unchanged**: `section <key> is too long (max 10000
  characters)`, as the live run below shows. With two sections over the cap the
  first in sorted order is named, the same on every call.

## Evidence

Red outputs, verbatim, from a private PostgreSQL cluster:

- F85, `a602548`: store, for each of the eight section keys, `background at
  10001 characters: err = <nil>, want a validation error: the store accepted it`;
  `the refused call stored goals="a fine goal" risks=10001 characters
  title="renamed in the same call", want none of it`; `a 10001-character write
  with a fresh revision: err = <nil>, want a validation error`. Agent: `a
  10001-character section was written: the coach can store text its own prompts
  then clip`.
- F95, `d5b86d7`: `coach: the section the coach wrote reaches its own prompt cut
  ("[truncated: 500 more characters]...")`, and the same line for the critic and
  the planner.
- F127, `0f03907`: `save of every section with the 15000-character one
  unchanged: status 400, want 200: {"error":"section background is too long (max
  10000 characters)"}` and `the edit was lost: goals = ""`.

Green: at `a18fdfe` the four F85 tests and the F95 test pass, with the typed-error,
constant and shorten-a-legacy-row tests that arrive with it; at `a881cff` the F127
test passes and so do the two tests that arrive with it (the exemption, and a trim
that stays over the cap). At the head all eleven `TestRvPrdCap*` tests pass (store
6, agent 3, api 2), with the existing `TestPrdSectionCap_*` and
`TestUpdateSections_*` unchanged and green. The tests spell the number 10,000
out on purpose: it is the documented contract, and raising it is a decision for
the docs.

Mutations, each applied, run, and restored (the tree was checked clean after);
the first three ran against `a18fdfe`, the last three against `a881cff`:

| Mutation | Tests that went red |
| --- | --- |
| `>` became `>=` in the store check | store `...RefusesAnOverlongSection`, agent `...CoachWriteOverTheCap...` |
| the store check removed | store 4 (refuse, refused-patch, coach call shape, typed), agent 3 (tool, prompt, shorten) |
| the tool's shorten-it message removed | agent `...CoachWriteOverTheCapIsRefusedAndNamesTheWayOut` |
| the exemption dropped | store `...UnchangedTextIsExemptButAChangeIsNot`, api `...SavingAroundAnUnchangedLongSectionSucceeds` |
| the exemption widened to any long stored section | store `...UnchangedTextIsExemptButAChangeIsNot`, api `...TrimmingALongSectionIsHeldToTheCap` |
| the handler no longer maps the typed error | api `TestPrdSectionCap_*` (3) and `...TrimmingALongSectionIsHeldToTheCap` |

Live check, on the head binary against the private database, on a loopback port,
as the seeded demo user, with a 15,000-character `background` planted by the
owner role (the state a pre-cap row is in). Five `PATCH /api/v1/prds/{id}` calls:

| Call | Answer | Stored afterwards |
| --- | --- | --- |
| every section sent, `background` untouched at 15,000, `goals` edited | 200 | goals saved, background still 15,000 |
| `background` trimmed to 12,000 (still over), `goals` also changed | 400 `section background is too long (max 10000 characters)` | nothing changed |
| `background` trimmed to 9,000 | 200 | 9,000 |
| `background` grown from 9,000 to 10,001 | 400, same text | 9,000 |
| `background` at exactly 10,000 | 200 | 10,000 |

The server log held five PATCH lines (200, 400, 200, 400, 200) and no WARN or
ERROR. The one process was stopped by its recorded pid; the port answered
nothing afterwards.

Not exercised: a live coach turn against a model provider (the tool path, the
refusal text and the rewrite-after-refusal path run through `Execute` against
real PostgreSQL), and the Flutter PRD page.

## Known limits

- **Rows stored before the cap stay stored.** Prompts still cut them at 10,000
  characters with the marker. No migration trims them; whether to is an owner
  decision, and an edit under the cap cleans a row by itself.
- **The comparison takes no row lock**, like the CAS check above it (the SELECT
  in `requirePrdAuthorOrAdmin` has no `FOR UPDATE`, although its comment says it
  locks). Two writers still resolve last-write-wins, as before. A race can put
  back a long text another writer just shortened; it cannot store a new text over
  the cap, because a text over the cap passes only when it equals what the row
  held at the read.
- **The web page learns of the cap from the 400.** The section editors in
  `client/lib/features/prd/prd_desk.dart` still have no `maxLength` or counter
  (R3-06); the client is outside this cluster's files.

## Left for the integrator

- INTEGRATOR TODO: docs/changelogs/2026-10-18-rv4-remediation/index.md: add a wave 4 row linking `w4-rv-prd-cap.md` (rv4-FR-F85, rv4-FR-F95, rv4-FR-F127, all `FIXED`).
- INTEGRATOR TODO: docs/changelogs/README.md: list this fragment.
- INTEGRATOR TODO: docs/changelogs/index/2026-10.md: list this fragment.
- INTEGRATOR TODO: docs/plans/README.md: record the three Mediums of `w4-rv-prd-cap` as `FIXED` on the integration branch.
- INTEGRATOR TODO: docs/issues/2026-10-18-review-4-open-findings.md: R3-05 first half (cap `update_prd_section`) is closed here; the `get_prd` clip is not taken (reasons above); R3-06 (editor `maxLength` and counter) stays open.
- INTEGRATOR TODO: docs/issues/2026-10-18-review-4-dispositions.md: the rv4-COMP-04 note "`update_prd_section` through the agent has no cap" is stale for the agent half; add the three rows of this cluster.
- INTEGRATOR TODO: docs/changelogs/2026-10-18-rv4-remediation/w3-stream-bounds.md: the constants table (line 104) still names `maxPrdSectionRunes` and `prd_section_cap.go`, both gone; the cap is `store.MaxSectionRunes`. A dated record, so add a pointer to this fragment rather than rewrite it.
- INTEGRATOR TODO: docs/architecture/prd-coach-agent.md: say that `update_prd_section` refuses a section over 10,000 characters with the shorten-it text, and that the prompt clip is the backstop for rows stored before the cap.
- INTEGRATOR TODO: docs/architecture/backend.md: name `store.MaxSectionRunes` as the one section cap and `UpdateSections` as where it is enforced.
- INTEGRATOR TODO: docs/usage/deployment.md: the constants list: the section cap is `store.MaxSectionRunes`; no variable, nothing to configure.
- INTEGRATOR TODO: docs/usage/cli-mcp-and-guidance.md: the `prd update --section k=v` row (line 78): add the 10,000-character limit and the 400 text.
- INTEGRATOR TODO: client/lib/features/prd/prd_desk.dart: R3-06: give the section editors `maxLength` 10,000 with a counter, plus one widget test.

## Verification

Run on the head of the branch against a private PostgreSQL cluster.

- `go build ./...` and `go vet ./...` on the server module: clean. `gofmt -l
  internal cmd`: clean. No Go or Dart file the branch touches is over 350 lines
  (the largest is `api/prds.go` at 349; `api/prd_section_cap.go` is deleted).
- `go test ./internal/agent/... ./internal/api/... ./internal/store/... -race
  -count=1`: all three packages `ok`. The same run with `-v` counted 1,049 PASS
  lines, no FAIL and no SKIP.
- The whole server suite ran once, last, with `-count=1 -v` and no `-race`:
  28 packages `ok`, 3 without test files, 2,007 PASS lines, no FAIL. One test
  skipped, `TestReadSecretTerminal` ("stdin is not a terminal"), which is not
  database-gated; no `DATABASE_URL`-gated test skipped anywhere.
- The 14 `scripts/tests/check_*.py` scripts exit 0 (`check_doc_links.py`: 0
  broken relative links across 617 tracked Markdown files, this fragment
  included); `scripts/check-redaction.sh` and `scripts/check-docs-recorded.sh`
  exit 0.
- Each red commit fails for the behaviour in its message and each fix commit turns
  it green; six mutations of the fixes turned the new tests red (table above).
