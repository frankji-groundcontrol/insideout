# Wave 1, client polish — review-4 remediation

Branch `rv4/w1-client-polish`, `1dba3fe..a29792d` (four commits: two red/fix
pairs), then this fragment. Findings are cited as `rv4-<ID>` because review
IDs restart per review. Row 1.16 and the dialog half of row 1.12 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md);
the bootstrap half of 1.12 belongs to `w1-ops-docs`. Client only: no server,
schema or API change and no new dependency.

## What changed

- **The Edit project description field has its own label** (rv4-USER-11, red
  `80e12d0`, fix `e11bcab`). The dialog labelled the description field with
  `project.updatePlaceholder` ("What's new?"), the prompt of the status-post
  dialog. A new key, `project.descPlaceholder` ("Project description"), is
  read at that one line in `project_dialogs.dart`, with the matching
  simplified-Chinese text in the Chinese table. The Post-an-update dialog and
  the updates feed keep the old key, so their wording is unchanged. The red
  test renders both dialogs in both locales and asserts on the text on screen,
  not on the table; its two Post-an-update tests pass on the base and pin what
  must not move.
- **A picked deadline reads as the picked day in every zone** (rv4-COMP-02, red
  `65778c3`, fix `a29792d`). The picker sent the picked day's local midnight as
  a UTC instant (`picked.toUtc()`), and every deadline label (`isoDate`, the
  first ten characters) reads the UTC date. In a zone ahead of UTC that is the
  evening before, so a picked day showed as the day before in the edit
  dialog, on the node card and in the progress list. The picker now sends the
  picked calendar day at noon UTC (`DateTime.utc(y, m, d, 12)`): still
  RFC3339, still accepted by `server/internal/api/roadmap.go`, and the UTC
  date is the picked day for every viewer. `isoDate` keeps its behavior; only
  its doc comment says what it reads. The red test drives the real edit dialog
  and its real date picker: pick the day, read the button, save, read the
  PATCH body, hand that string to the node card. It is red only where the bug
  lives: Asia/Shanghai, Asia/Tokyo and Pacific/Auckland fail on the base, UTC
  and America/New_York pass there, which is why a UTC CI run never saw it.

## Design spike (row 1.16, marked `dir`)

Row 1.16 names two ways out, and a third turned out better.

1. *Send a date (`yyyy-MM-dd`).* The PATCH handler parses RFC3339 only and the
   server is outside this cluster, so this is not available.
2. *Render with `.toLocal()`.* Local midnight stays the stored instant and the
   label formats it in the viewer's zone. Not taken: teammates in different
   zones would read different dates for one deadline; the display fixtures
   that spell `...T00:00:00Z` would read the previous day under any zone behind
   UTC, so the suite would depend on the developer's zone; and `isoDate` is
   also `timeAgo`'s date fallback, so the change would reach every timestamp in
   the app.
3. *Write the calendar day, not a local instant (taken).* A deadline is a day
   in the picker, so the stored instant is a fixed anchor on that day in UTC
   and the unchanged UTC-date label reads it back.

Why noon and not midnight: `DateTime.utc(y, m, d)` reads back the same, but its
instant falls before the picked day's local midnight in every zone behind UTC
(up to 12 hours), and `pressure` is computed from the instant, so a node due on
D would turn near, high risk or overdue the evening before D. Noon UTC is
inside the picked day for every zone from UTC-12 to UTC+11. It never lands
before the picked day starts, and a server host that formats the instant in its
own zone (someone running the server locally) still prints the picked date.
The cost is one literal.

## Known effects and limits

- Deadlines saved before this change keep the instant the old picker sent, so
  in a zone ahead of UTC they still read one day early until they are picked
  again. The server stores an instant, so nothing can recover the intended day.
- `pressure` follows the new instant: later than before by 12 hours plus the
  zone's offset (20 hours in Asia/Shanghai, 7 to 8 in America/New_York), never
  earlier. A node due on D turns overdue later on D itself (20:00 local in
  Asia/Shanghai, about 08:00 in New York) instead of as D starts.
- Only the writer changed. `isoDate` still reads the date as written, which is
  the UTC date for the server's `Z` timestamps, so `timeAgo`'s date fallback
  (items older than 14 days) shows the UTC date as before. Showing timestamps
  in the viewer's zone is a wider display decision and was left alone.
- Ceiling: a server host in UTC+12 to UTC+14 that formats in its own zone
  prints the next day. The shipped image sets no zone.
- The deadline test is meaningful only in a process zone ahead of UTC. CI runs
  `flutter test` on a UTC runner, where the bug cannot show, so there the test
  still guards the wire shape and the label but not the bug. See the first
  item below.
- The test picks the picker's opening day (a week from now) instead of the
  literal 2026-10-18 the plan names: the picker's earliest day is yesterday, so
  a fixed past date becomes unpickable within two days. The expected text comes
  from the same clock expression.

## Left for the integrator

- **CI zone step** (`.github/workflows/ci.yml`, owned by `w1-ci-gates`): after
  the `flutter test` step add
  `TZ=Asia/Shanghai flutter test test/roadmap_deadline_tz_test.dart`, or the
  new test passes in CI without exercising the bug.
- **Optional architecture note** (`docs/architecture/product-subsystems.md`,
  the roadmap time-first bullet where PATCH `deadline` is described): one
  sentence saying the web picker sends the picked day at noon UTC, so every
  member reads the same date. No existing sentence became untrue.
- **No usage-doc change**: the only deadline wording under `docs/usage/` is the
  CLI's `--deadline RFC3339|clear`, which this change does not touch.
- **No design-QA entry**: no owner visual comments came with this work.
- The shared indexes (this folder's `index.md`, the changelog README and month
  index, the plans board, the handoff) are the integrator's.

**Closed (integrator follow-up, wave 1).** The noon-UTC sentence is in
`docs/architecture/product-subsystems.md`. The CI zone step is residue R1-01 in
the [open-findings record](../../issues/2026-10-18-review-4-open-findings.md); it waits on the `ci.yml` split
([issue](../../issues/2026-10-18-ci-yml-split.md)). The two optional product
questions are R1-15.

## Verification

Each red commit was replayed on its own parent and fails for the stated
reason; each fix commit turns it green. The outputs are kept as
`w1-client-polish-red-rv4-<ID>.txt` and `w1-client-polish-green-rv4-<ID>.txt`
under `/private/tmp/insideout-rv4/reports/`.

- rv4-USER-11 red: 2 of the 4 new tests fail, both Edit project tests ("Found 1
  widget with text" the status-post prompt, in each locale); the two
  Post-an-update tests pass. Green: the new file plus the project flow, page
  and updates-display tests, 30 passed.
- rv4-COMP-02 red under Asia/Shanghai: expected `2026-10-25`, actual
  `2026-10-24` (run on 2026-10-18; the day is a week out from the clock). Green
  under Asia/Shanghai, UTC, America/New_York, Asia/Tokyo, Pacific/Auckland,
  Pacific/Kiritimati (UTC+14) and Pacific/Pago_Pago (UTC-11).
- Mutation check a verifier can run: put `picked.toUtc().toIso8601String()`
  back in `roadmap_node_menu.dart` and the test is red again under
  Asia/Shanghai (and still green under UTC and America/New_York); switch the
  description field back to `project.updatePlaceholder` and both Edit project
  tests fail again.
- The `isoDate` and `timeAgo` callers (deadline label, prd timestamps, date
  picker theme, roadmap page, bands, progress, narrow and surfaces tests,
  project flows, Material locale): 43 passed under Asia/Shanghai, UTC and
  America/New_York.

Gates: `flutter pub get --enforce-lockfile` and `flutter analyze
--no-fatal-infos` (14 infos, the same set as the base, none in a touched
file); the full `flutter test` (463 passed) in the default zone and under
Asia/Shanghai, UTC and America/New_York; the dedicated test under
Asia/Shanghai; `dart format --output=none --set-exit-if-changed lib test`;
`git diff --stat -- client/pubspec.lock` empty; `sh scripts/check-redaction.sh`
and every `scripts/tests/check_*.py`. No database was used.
