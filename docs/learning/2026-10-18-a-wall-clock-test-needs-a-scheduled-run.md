# A wall-clock test turns red by date, and a push-only CI never sees it

**Date:** 2026-10-18
**Source:** [multirole review 4](../changelogs/2026-10-18-multirole-review-4.md), lead L1 and findings PM-02, ARCH-04, QUAL-05
**Fixed in:** [wave 1, clock and runes](../changelogs/2026-10-18-rv4-remediation/w1-clock-and-runes.md)
(L1, ARCH-04, QUAL-05: the clock seam and the guard) and
[wave 1, CI gates](../changelogs/2026-10-18-rv4-remediation/w1-ci-gates.md)
(PM-02: the weekly shuffled run, pending first CI run after push)

## The lesson

`TestProjectQuietFlagged` pins its own clock (`now = 2026-10-04`) but
asserts through `projectResponse`, which calls `time.Now()`
(`server/internal/api/projects.go:84` at `9f0b1c7`). The board's quiet window is 14 days,
so the two `view.Flagged` subtests went red once the real date passed
2026-10-16 without any commit changing. `ci.yml` has no `schedule:` trigger and
the last CI run was 2026-10-11, so nothing ran: `main` sat red for days while
`docs/HANDOFF.md` still said the head run was green. The first run to see it
was a review that ran the whole suite on a clean checkout.

## Why it happens

- A function that reads the wall clock directly has no seam; the test can only
  pin the half it controls, and the other half drifts.
- CI that runs only on push and pull request measures the tree, not the tree
  plus the date. Quiet weeks make the gap invisible.
- "The head run is green" is a statement about the last run, not about today.

## When to apply

- Any view builder, TTL, expiry or "quiet since" rule: inject the clock and
  have the test pin that one. Here that is a `Server.clock` field read through
  `s.now()`, with the instant passed to the builder as a parameter, and a guard
  test (`view_clock_guard_test.go`) that fails on a raw `time.Now`,
  `time.Since` or `time.Until` in the package.
- Any repository whose tests depend on time, even once: give CI a weekly
  `schedule:` run with `-shuffle=on`, and cite a dated SHA, not "the head
  run", in HANDOFF.
- Reading a claim of "green": look for the date of the run behind it.

## Evidence

Coordinator run, clean worktree at `9f0b1c7`, private PostgreSQL 18.4,
`go test ./... -race -count=1`: 25 packages ok, 855 pass lines, one failure:
`--- FAIL: TestProjectQuietFlagged` (`view.Flagged = true, want false`,
`projects_test.go:51`). Remediation rows 1.3 and 1.4 of the
[review-4 plan](../review/2026-10-18-multirole-4/40-remediation-plan.md).
