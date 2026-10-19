# A handler that never reads its body never sees the client leave

**Date:** 2026-10-19
**Source:** [multirole review 4](../changelogs/2026-10-18-multirole-review-4.md), findings QUAL-04, QUAL-06 and REL-06 (context and deadlines)
**Found in:** [wave 3, context and deadlines](../changelogs/2026-10-18-rv4-remediation/w3-context-deadlines.md)

## The lesson

Go's `net/http` learns that a client went away from a background read of the
connection, and it starts that read only once the handler has read the request
body to its end (a request with no body starts it at once). A handler that
never touches a non-empty body therefore never has `r.Context()` cancelled when
the caller disconnects. The caller is free (a Ctrl-C ends the CLI in 0.00 s with
exit 130), but the server goes on: the planner call, its dispatch permit and its
`ai_runs` row run to the end.

A client-side test cannot show this. It proves the CLI stopped, not that the
server noticed. The check has to close the connection mid-call and assert on the
server side: the planner's context is cancelled and the run row ends `failed`.

## Where it is still open

`handleBuildFromPrd` and `handleExpandNode` in
`server/internal/api/roadmap_ai.go` never read the body. It is open residue
R3-25 of the [open-findings record](../issues/2026-10-18-review-4-open-findings.md):
decode or drain the body first in both handlers, and add the disconnect test.
`handleConvertIdea` decodes its body and does not have the problem. The cost is
bounded by the limit of 10 AI runs a minute and 60 an hour.

## When to apply

- Any long handler (an LLM call, a slow query) that should stop when its caller
  does: read the body before the work, and test the disconnect on the server.
- Any time a deadline or Ctrl-C change is called done on client evidence alone.

## Evidence

Measured live against a provider stub: the stub was asked at 0 s, the CLI was
interrupted at 3 s, and the stub was still answering at 25.0 s with no sign that
the server had cancelled. The same mechanism lets a call the client cut at its
own deadline keep the server busy (inferred, not measured).
