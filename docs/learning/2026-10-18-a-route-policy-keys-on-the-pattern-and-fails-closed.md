# A route policy keys on the mux pattern and treats the zero value as denied

**Date:** 2026-10-18
**Source:** [multirole review 4](../changelogs/2026-10-18-multirole-review-4.md), finding FEAT-01 (agents were never kept from deciding, Committing or administering)
**Fixed in:** [wave 2, agent scope](../changelogs/2026-10-18-rv4-remediation/w2-agent-scope.md)
([design](../architecture/agent-token-scope.md))

## The lesson

PRODUCT.md said an agent may not decide a proposal, Commit a version or switch
signup mode. Nothing enforced it: `insideout-mcp` sent the user's own token, the
server accepted any valid token on every route, and the rule lived in prose. A
check inside each handler would have been one more line to forget on the next
route.

The fix classifies every authenticated route once, in a table keyed by the mux
pattern (`"METHOD /path"`, which `net/http` sets on `r.Pattern`), and reads it in
the one place that verifies a token (`requireAuth`). The access type's zero value
is `humanOnly`, so a route nobody classified is closed to an agent token, and a
test fails the build when a registered route has no entry or an entry names no
route. Adding a route is therefore a decision, taken in review, not a default.

## Why it works

- A deny list ("these routes are human-only") fails open: the route added next
  month is open until someone remembers it. An allow list whose zero value is
  "denied" fails closed.
- The pattern identifies method plus route, so `GET` and `DELETE` on one path
  are two entries and path parameters do not matter. A handler the mux did not
  serve has no pattern and is denied too.
- An unknown claim value is a 401, not "full power", so a scope added later
  cannot fall back to full access on an older server.
- Defence in depth is cheap and testable: the decision handler also refuses the
  agent scope on its own, and only turning both off together turns the test red.

## When to apply

- Any capability that one kind of credential holds and another does not (an
  agent token, a read-only key, a service account): one table, one enforcement
  point, a zero value that means no.
- Any list that must stay complete as the code grows: make the build fail on the
  gap instead of relying on a reviewer's memory.

## Evidence

At `e6bc637` the table holds 72 authenticated routes: 35 agent-allowed, 1
submit-only (the PRD status route, whose body decides), 36 human-only. The
route-table test is `TestEveryRouteHasAnAgentScopeDecision`; thirteen mutations
(deny disabled, unknown-scope check dropped, submit always allowed, and others)
each turned a test red. Live: an agent token got 403 `agent_scope_denied` on the
decision, Commit, admin, delete and mint routes and 200 on propose and reads.
