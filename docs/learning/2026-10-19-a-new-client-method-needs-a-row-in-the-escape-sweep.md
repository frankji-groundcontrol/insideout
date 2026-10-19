# A new exported client method needs a row in the escape sweep

**Date:** 2026-10-19
**Source:** [multirole review 4](../changelogs/2026-10-18-multirole-review-4.md), finding CLOSE-02 (`idea convert` and the language of its output)
**Found in:** [wave 3, first-run copy](../changelogs/2026-10-18-rv4-remediation/w3-first-run-copy.md)

## The lesson

`TestSweepCoversEveryClientMethod` in `server/internal/apiclient/escape_test.go`
lists the exported methods of `apiclient.Client` by reflection and requires a
row for each one in the escape sweep (the table that feeds hostile values to
every URL shape). A new exported method with no row fails that test with
`Client.<Name> has no row in the escape sweep: add one`. The sweep exists so that
a path segment or a query value built from user text cannot grow a second
segment or a second parameter; a method nobody swept is the one that will.

The fix for `idea convert` first added a second exported method,
`IdeaConvertWithLocale`, and the sweep failed. The shipped shape is one method
with a variadic, `IdeaConvert(ideaID string, locale ...string)`: it keeps the
one row per URL shape and every existing caller compiling. The test file
belongs to another cluster, so adding a row there was not an option.

The second half of the lesson is how it was found: the new tests passed, and
only the whole package run showed the failure. Run the whole package, not only
the tests you wrote, before calling a client change done.

## When to apply

- Adding or renaming an exported method on `apiclient.Client`: prefer extending
  an existing method (an optional argument) over a sibling; if a sibling is
  right, add its sweep row in the same commit.
- Any guard that enumerates a surface by reflection: expect it to fail the first
  time someone adds to the surface, and treat that failure as the guard working.

## Evidence

At the repair commit the full package run printed
`escape_test.go:257: Client.IdeaConvertWithLocale has no row in the escape sweep: add one`;
with the variadic form both `TestSweepCoversEveryClientMethod` and
`TestHostileValuesStayOneSegmentOrOneQueryValue` pass.
