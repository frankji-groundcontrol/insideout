# Go test census: `=== RUN` lines double-count subtests

**Lesson.** When census-counting Go test results (pass/skip/fail), count
top-level result lines (`--- PASS:` / `--- SKIP:` / `--- FAIL:`), never
`=== RUN` lines. Since Go 1.21+ subtests also print `=== RUN` (in this repo,
Go 1.25 prints subtest RUN lines flush-left, indistinguishable by leading
whitespace), so `grep -c '=== RUN'` counts parents and children — a census of
290 "tests" was really 247 RUN lines (145 parents + 59 subtests + 43 skips)
and 188 top-level results.

**Evidence.** 2026-10-04 multirole review: coordinator reported "290 results
/ 43 SKIP"; the completeness critic's recount (`go test ./... -count=1 -v`)
showed 145 top-level PASS + 43 SKIP = 188, with subtests like
`TestQuoteIsGrounded_ExactAndCJK/exact_english_substring` printing their own
unindented `=== RUN`. The verdict the census supported survived, but the
published number had to be corrected
([critic §4](../review/2026-10-04-multirole/30-completeness-critic.md)).

**Scope / when to apply.** Any time a test-count figure goes into a record,
review, or CI badge: count `--- (PASS|SKIP|FAIL):` at the top level, or use
`go test -json` and filter `Action == "pass"|"skip"` on package-level events.
Never quote a grep of `=== RUN`.
