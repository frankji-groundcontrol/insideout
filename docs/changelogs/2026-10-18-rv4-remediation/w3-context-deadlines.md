# Wave 3, context and deadlines — review-4 remediation

Branch `rv4/w3-context-deadlines`, `8c74218..d86ca54` (eight commits: four red
tests and four fixes), then this fragment. Row 3.9 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md)
(a design row); findings `rv4-QUAL-04`, `rv4-QUAL-06`, `rv4-REL-06` and
`rv4-FULL-07`. Findings are cited as `rv4-<ID>` because review IDs restart per
review. No value, name or address is spelled in this fragment.

## What changed

- **A cancelled context ends the request in flight** (rv4-QUAL-04, red
  `b5febe1`, fix `f05e708`). `apiclient.Client` has `SetContext(ctx)` and
  `Context()`; `newRequest`, the one place a request is built (`do`,
  `getBytes`, `authPost`, `Guide`), puts the client's context on it. The renewal
  of an expired session and the agent-token trade already ran under
  `req.Context()`, so they end with it. Before, every request was built without
  a context, the MCP tool handlers dropped the one the SDK hands them, and the
  CLI took the default signal action: a Ctrl-C killed the process wherever it
  stood, even between the server rotating a refresh token and the CLI saving
  the new pair. Now each MCP tool call builds its client on its own context
  (`clientCtx`; the agent-scope trade rides it too) and the CLI runs under
  `runInterruptible`: the first Ctrl-C or SIGTERM cancels `cliCtx`, the command
  prints its error and the process exits 130, the shell's code for an
  interrupted command. A command still running after `interruptGrace`, or a
  second signal, ends the process at once. `insideout setup` binds its sign-in
  client and its interactive flow to the same context.
- **A lost terminal-state write leaves a log line** (rv4-QUAL-06, red
  `a54a533`, fix `6dc6e64`). Eleven `_ = c.store.*(...)` writes in the coach and
  its gate (four in `coach.go`, one in `coach_loop.go`, two in `coach_fail.go`,
  four in `gate.go`) threw their errors away: the run's terminal status, the
  circuit-breaker result, the text that had streamed, the heartbeat. A lost
  `MarkAIRunFailed` leaves the run "running" until the reaper; a lost
  `RecordCircuitResult` leaves the breaker blind to failures. `logErr` (new
  `log_err.go`) logs each failure at Error with the store method and the run id,
  and nothing for a nil error or `context.Canceled`: a client that left is
  routine, and every terminal write runs detached from the request, so a
  cancelled context can only be the heartbeat. The red test loses the database
  for real (it closes the pool at the point each write runs) and asserts the log
  names each lost write with the pool's error; a control keeps a healthy turn
  silent.
- **A model call is not cut at 20 seconds** (rv4-REL-06, red `aa62283`, fix
  `ed759b5`). Every request shared one `http.Client{Timeout: 20s}`, so
  `POST /prds/{id}/build`, which drafts a whole roadmap with the planner's
  model, was cut off mid-draft whenever the provider took longer. The client has
  no timeout now. `newRequest` wraps the client's context in
  `context.WithTimeout` for the deadline `deadlineFor(method, route)` picks (new
  `deadline.go`): 20 s for a call the server answers from its database, 5 min
  for a POST whose handler waits on the model (`aiRoutes`). The four call sites
  defer the cancel after the body is read, so the deadline covers the body: a
  server that sends its headers and stalls is still cut, and no request is left
  without a deadline. The red test reads the deadline each request is sent with,
  through a recording transport on a real server, and fails on the base for the
  right reason (the build "was allowed only 20s"). The fix adds clock tests with
  both deadlines shrunk to milliseconds.
- **A node expand is not cut either** (rv4-FULL-07, red `c720db5`, fix
  `d86ca54`). `POST /roadmap/{id}/expand` drafts a node's sub-tasks with the same
  planner model, so the same slow provider cut it off at 20 s. This finding and
  REL-06 describe one defect on two routes; it is two commit pairs so each row
  has its own red and fix. The fix is one entry in `aiRoutes` and three cases in
  the route table; `/expand/confirm` and every GET stay on the quick deadline.

## Design spike (row 3.9)

Four questions were settled before the code. *Where the context lives:*
`SetContext` sets it in place instead of returning a copy, because a `Client`
serves one caller at a time (an MCP call builds one, a CLI verb builds one) and a
copy would be a second client whose session could fall out of step with the
first. *What a deadline is:* a client-wide timeout can not fit a read and a model
call, so the deadline is per call and comes from the route; it is a context
deadline, not a second `http.Client`, because the route is known only when the
request is built, the renewal round trip and its replay run under the request's
context and share the one deadline, and the caller's own cancel composes with it
(the earlier one wins). An environment override was left out: nobody has asked
for one, and the two durations are package variables in one file. *Which routes
wait on the model:* only `build` and `expand`. The planner is the one model call
an `apiclient` method reaches; idea convert is database-only (its handler calls
`ConvertIdeaWithLocale` and nothing else), and the coach's SSE turn has no
`apiclient` method at all (the web client streams it, and the server bounds it
with its own turn deadline), so neither needs the long class. *What Ctrl-C
means:* cancel, do not kill. The first signal cancels the command's context so
its request aborts and its error prints, and the process exits 130; the second
signal, and a 400 ms grace for a command that is not in a request (a prompt),
keep a stuck command from trapping the user.

## Decisions and deviations

- **Tunables.** `quickDeadline` 20 s (the old number, so no quick call fails
  later than before); `aiDeadline` 5 min (a draft is one forced-tool call capped
  at 4096 output tokens, retried once on a transient failure, with a 90 s idle
  watchdog on the stream, so it ends in minutes on a slow provider);
  `interruptGrace` 400 ms. Both deadlines are unexported package variables so a
  test can shrink them.
- **Error text is raw.** A cut call now reads "context deadline exceeded"; the
  old "Client.Timeout exceeded while awaiting headers" is gone. A friendlier
  sentence is a later copy change.
- **Renewal is untouched.** Single-flight, the grace wait and the replay are as
  wave 2 left them; the renewal now runs under the route's deadline, as it did
  under `Client.Timeout`.
- **`AgentScoped` copies the login's context**, and `client(api)` stays as
  `clientCtx` with a background context for callers that have none.
- **`IdeaConvert(ideaID string)` is unchanged.** The deadline tests never call
  it, so the first-run-copy cluster can add its locale argument without touching
  them.

## Evidence

Under `/private/tmp/insideout-rv4/reports/w3-context-deadlines-evidence/`: a red
and a green file per finding (`qual04-*`, `qual06-*`, `rel06-*`, `full07-*`),
the DB-gated agent tests having run against a private PostgreSQL cluster (a
skipped test counts as red). Removing the build route from `aiRoutes` turns the
REL-06 red test, the draft subtest and the route table red again.

Live proof with the real binaries: the CLI built at the REL-06 red commit
("before", still on the 20 s timeout) and at the head of this branch ("after"),
a real server on loopback against the private database, and a provider stub on
loopback that takes 25 s to answer (`live-proof.txt`):

| Call | Before | After |
| --- | --- | --- |
| `insideout build <prd>` | exit 1 after 20.0 s, "Client.Timeout exceeded while awaiting headers" | exit 0 after 25.0 s, the drafted tree |
| `insideout expand <node>` | exit 1 after 20.0 s, same error | exit 0 after 25.0 s, the drafted sub-tasks |
| Ctrl-C 3 s into a build | not run: the "before" binary already carries the QUAL-04 fix | exit 130, 0.00 s after the signal, "insideout: interrupted" |

## Known ceilings and a finding for follow-up

- **The server does not notice a caller that left, on exactly these two
  routes.** `handleBuildFromPrd` and `handleExpandNode` never read the request
  body, so net/http never starts its background read of the connection and
  `r.Context()` is not cancelled when the client goes. A Ctrl-C frees the user,
  but the planner call, its dispatch permit and its `ai_runs` row run to the end.
  Measured live (`live-disconnect-probe.txt`): the provider stub was asked at
  0 s, the CLI was interrupted at 3 s, and the stub was still answering at 25 s
  with no sign that the server had cancelled. By the same mechanism a call the
  client cut at its deadline leaves the server working too (inferred, not
  measured). `handleConvertIdea` decodes its body and does not have the problem. Fix, in `server/internal/api/roadmap_ai.go` (not this
  cluster's file): decode or drain the body first in both handlers, and add a
  test that closes the connection mid-call and asserts the planner's context is
  cancelled.
- **`aiRoutes` is a hand-kept list.** A new route whose handler calls the model
  has to be added, or it gets the quick deadline (marked in `deadline.go`).
- **MCP cancellation** is proved through the real transport in
  `ctx_cancel_test.go`; it was not driven through a live MCP host.
- **The deadline is per exchange, not per byte.** A model that streams a draft
  slowly but steadily is held to `aiDeadline` in total.

## Integrator notes (wave 3)

- `docs/changelogs/2026-10-18-rv4-remediation/index.md`: list this fragment.
- `docs/usage/`: the CLI page should say that Ctrl-C ends a command with status
  130 and that a model call (`build`, `expand`) is allowed five minutes while
  everything else is allowed twenty seconds.
- The server-side disconnect finding above is not in the register; give it an id
  and an owner.
