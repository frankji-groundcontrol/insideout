# Wave 3, the coach trust layer — review-4 remediation

Branch `rv4/w3-coach-trust`, `8c74218..89efd58` (fourteen commits: seven red/fix
pairs), then this fragment. Findings are cited as `rv4-<ID>`. Row 3.1 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md),
plus the carry-forward lead `LEAD-FENCE-DEDUP` (its own report row). Six rows are
`FIXED`; `rv4-AIDEV-07` is `PARTIAL` (its other half is in `coach.go`, not this
cluster's file). One choice departs from the cluster note: a skipped critic still
opens the finalize gate and is announced instead (see "The gate decision"). No
migration, no new dependency; no address, host or secret is spelled here.

## What changed

- **A skipped critic is announced, and its marker no longer outlives a later
  round** (rv4-AIDEV-02, red `5deff34`, fix `f94017d`). When the critic could not
  run (`contention`: another request held the conversation's critic slot;
  `critic-output-failure`: two attempts, no usable `report_findings`),
  `maybeRunCritic` stored `critic_state.skipped`, the finalize gate accepted it
  in place of a round, and nobody was told; a later round that did run left the
  marker standing. Now:
  - `recordCriticSkip` (`trust_critic.go`) stores the reason and sends
    `critic_skipped {reason, rounds}` before `message_end` (`rounds` is how many
    critic rounds ran; 0 means nothing was reviewed). It replaces the ignored
    `UpdateConversationMeta` error at the old `critic.go:46`: a failed write is
    logged and no frame goes out for a marker that was not stored. A skip while
    the marker is already set is not announced twice;
  - a critic round that ran clears the marker, and the conversation view gains
    `criticSkipped` and `criticRounds`;
  - the web client says it twice, in English and Chinese: a notice when the frame
    arrives, and a line under the finalize proposal ("No critic review ran (the
    server was busy)."; when only the revision round was skipped, "The critic did
    not re-review your latest changes (...)."). A reason this build does not know
    prints "unknown reason", never a raw token. The page re-reads the
    conversation when a turn ends, so the card carries the stored marker. The
    gate itself is unchanged, on purpose (see "The gate decision").
- **The confirm card lists the finding resolutions the coach parked**
  (rv4-AIDEV-03, red `2a58569`, fix `68cdba5`). `resolve_finding` only parks
  `{id, status}` entries in `meta.pendingFindingResolutions`; confirming applies
  them, and the card showed nothing but the stage name. The view now carries
  `pendingResolutions`: each parked entry joined to its finding, filtered as
  `ApplyStageConfirmation` filters (a finding that is gone, or a status other
  than `resolved` or `overridden`, is skipped there and so here;
  `TestWhatTheViewShowsIsWhatConfirmApplies` compares the two). The card prints
  one line per entry, "Confirming marks this finding resolved (severity /
  section): issue" or "Confirming overrides ...", and the Chinese pair; a
  severity or section this build cannot name is left out of the parenthesis.
  Confirm and Decline are held while a turn streams.
- **Declining a proposed stage is a server-side write** (rv4-CLOSE-03, red
  `efc9f0b`, fix `9f2ccd9`). Decline only dropped the card from the page's state,
  so a reload brought it back and the resolutions parked with it stayed parked
  for the next confirm. The new `POST /api/v1/conversations/{id}/stage/decline`
  (body `{"stage": "..."}`) calls `DeclineConversationStage`: one `UPDATE` that
  removes `pendingStage` and `pendingFindingResolutions` from the actor's own
  conversation when the named stage is the pending one (a plain jsonb key
  removal, so the `Coach` interface is untouched; stage and findings stay). It is
  `humanOnly` in `agentRoutePolicy`, so an agent token gets 403. The client posts
  it, keeps the card up if the server did not record the decline, and ignores a
  second tap while a write is in flight; the widget test that asserted "decline
  is local" was changed on purpose.
- **A provider stream is a whole turn only when the provider says so**
  (rv4-AIDEV-05, red `3108e42`, fix `fe1c7f6`). Three defects in the two stream
  parsers, fixed there and not in the coach loop: a tool call cut by the token
  cap ran with half a JSON object; a stream that just stopped came back as a
  short answer; `response.failed` was ignored (table below). The Responses parser
  moved to the new `trust_responses.go` (182 lines), as `responses.go` was at 330
  of 350. The fix commit message says "half a object"; history is not rewritten.
- **Parallel Responses tool calls stay apart, and are not asked for**
  (rv4-AIDEV-06, red `7c49523`, fix `1fb1224`). The parser kept one
  `callID/name/args` and appended every argument delta to it, so two calls in one
  response became one (the second name, both argument strings glued, or spliced
  mid-JSON when interleaved). Arguments are now filed by the provider's item id
  (`respCalls`); the first announced call runs and later ones are dropped, never
  merged; a request that offers tools carries `parallel_tool_calls: false`.
- **A fact needs a quote long enough to attest it and text the quote supports**
  (rv4-AIDEV-07, `PARTIAL`, red `0c421b2`, fix `35a0873`). `record_fact` stored
  `attested` for any quote the user had said (a four-letter word sits in some
  message) and never compared the fact's text with the quote. `checkFactEvidence`
  (`trust_facts.go`) now refuses a quote under `minAttestedQuoteRunes` and text
  whose share of n-grams found in the stored quote is under `minTextSupport`,
  each with a message that tells the model what to do (see "The quote rule").
- **The coach fences through `agentcontext.FenceData`** (LEAD-FENCE-DEDUP, red
  `994b76d`, fix `89efd58`). `prompt_fence.go` held a copy of the MCP fence's
  regexp, kept in step by a parity test over nine fixed strings, so a rule change
  in one copy would split the fences on any other input, unnoticed. `fenceData`
  now delegates. The behaviour was already identical, so the red test is a source
  scan (`TestFenceRuleHasOneSource`: no non-test file may compile its own
  `<prd_data>` pattern); identity is shown by a throwaway differential run (not
  committed): the old `fenceData` against `agentcontext.FenceData` on 200,000
  random strings, no difference. The `w2-agent-scope` fragment left this edit for
  the integrator.

## Wire format

| Where | Shape |
| --- | --- |
| SSE `critic_skipped` | `{"reason": "contention" or "critic-output-failure", "rounds": n}`, sent when a pass did not run, before `message_end`; the view fields below tell a client that reloads later |
| conversation view: `GET /conversations/{id}`, `GET` and `POST /prds/{id}/conversation`, and the confirm and decline answers | `criticSkipped` (string, absent when none), `criticRounds` (integer, always), `pendingResolutions` (array of `{id, section, severity, issue, status}`, absent when none) |
| `POST /api/v1/conversations/{id}/stage/decline` | body `{"stage": "draft"}`; 200 with the view, 400 `stage is required`, 409 `that stage is not waiting for confirmation` (nothing proposed, another stage, a repeat, or someone else's conversation); human-only |

## The gate decision (rv4-AIDEV-02)

The cluster note asked that the finalize gate stop passing on a critic skip. Row
3.1 and the register's own fix ask for the skip to stop being silent: emit
`critic_skipped`, add an API field, and have the finalize confirmation say that
no critic review ran. That is what was built. The gate (`tools.go:305`,
`cs.RoundCount == 0 && cs.Skipped == ""`) is unchanged, and
`TestStageGate_CritiqueToFinalize_RequiresCriticPassOrSkipMarker` still pins it.

- **A, announce and keep the gate (chosen).** Plan section 5.1 made a skip count
  as a pass on purpose: a busy critic, or a provider that returns nothing usable,
  must not hold a PRD hostage. The person decides at the confirm card, which now
  says that no review ran.
- **B, refuse while a skip is set.** `critic-output-failure` is not transient: a
  misconfigured critic model fails every round, so finalize would stay
  unreachable. That is the hostage the plan avoids.
- **C, refuse only on `contention`.** It clears by itself, so refusing is cheap to
  recover from, but it adds a second rule to explain ("why is Finalize
  refused?") for a case the card already announces.

To flip it, change that one condition and its test together: for B,
`cs.RoundCount == 0 || cs.Skipped != ""`; for C, add
`|| cs.Skipped == "contention"`. The owner should confirm A (listed below).

## What a parser now refuses (rv4-AIDEV-05, rv4-AIDEV-06)

| Provider stream | Before | Now |
| --- | --- | --- |
| Anthropic `tool_use` with `stop_reason: max_tokens` | the half-written JSON ran as the call's arguments | the call is dropped; the turn is `Truncated` with the text so far; the loop's existing marker tells the person; nothing runs |
| Anthropic content but no `message_stop` | returned as a whole turn | `ErrProviderTransient`, "anthropic stream ended before message_stop" |
| Anthropic stream with no content block at all | "empty response from provider" | unchanged |
| Responses `response.incomplete`, or `response.completed` with status `incomplete`, holding a `function_call` | the half-written JSON ran | dropped; `Truncated` |
| Responses without `response.completed` or `response.incomplete` (end of stream, or a bare `[DONE]`) | returned as a whole turn | `ErrProviderTransient`, "<provider> stream ended before response.completed" |
| Responses `response.failed` | ignored; with no text it read as a refusal | its error code goes through the same taxonomy as the `error` event (rate limit, server error, config); no code means transient |
| two `function_call` items in one response | arguments appended into one call | filed by item id; the first runs with its own arguments |
| the Responses request | no flag | `parallel_tool_calls: false` when tools are offered |

`retryStreamChat` is unchanged and decides what a transient error costs: one
retry (about two seconds) while nothing has streamed to the person; an `error`
frame, no retry, once text has streamed (a retry would repeat it).

Strictness has a price. A gateway that never sends `message_stop` (Anthropic
lane) or `response.completed` (Responses lane) worked before and now fails every
turn with the transient error above. The repo's real-capture fixtures and the
live proof send those events, as the providers' documented streams do, but no
real provider was driven. The escape hatch is one condition per parser:
`!sawStop && len(order) > 0` in `anthropic_parse.go`, `!finished` in
`trust_responses.go`. A Responses gateway that rejects `parallel_tool_calls`
would answer 400, surfacing as a provider-config error; the field is set in one
place in `responses.go`.

## The quote rule (rv4-AIDEV-07)

Both numbers are tunables I chose, not measured limits. They are named constants
in `trust_facts.go`; `TestCheckFactEvidence_TunablesArePinned` fails when either
moves, so a change comes with a fresh look at the pairs below.

- **`minAttestedQuoteRunes = 12`.** The quote is normalized as grounding does
  (lower case, spaces and punctuation dropped) and counted in Latin-letter
  equivalents: a Han, kana or hangul rune counts twice. "we want it now" is 11
  and refused; "we want it soon" is 12 and passes; five Han runes (10) are
  refused, six (12) pass.
- **`minTextSupport = 0.2`.** The share of the fact text's distinct character
  n-grams (trigrams; bigrams once the text has any CJK rune) that the stored
  quote, cut at `maxQuoteLen`, also holds. Text too short for one n-gram is not
  judged.

Calibration, by hand, on 39 pairs I wrote (26 honest restatements, 13 texts about
something else; English, Chinese, one mixed): unrelated texts scored 0.00 to 0.16
(the highest a short claim against a long quote) and honest restatements 0.27 to
1.00, except one heavy paraphrase at 0.14, so 0.2 sits in the gap. With both
floors, 24 of the 26 honest pairs pass and all 13 unrelated ones are refused; the
two honest misses are that paraphrase and the 11-letter "we want it now".
Eighteen pairs are pinned in `trust_facts_bounds_test.go` (ten honest, eight
unrelated); the full list was a throwaway. A refused honest fact costs the coach
one retry with the message, so the floor errs towards refusing.

What this does not do:

- Support is lexical, as far as a server without a model call can go: it catches
  text about something else, not text that merely sounds like the quote. The
  model can pass it by copying the quote into the text or padding it with
  repeated filler (the share is over distinct n-grams). A restatement in another
  language than the quote shares no n-grams and is refused.
- The stored status is still `attested`, a legacy value kept byte-compatible with
  stored meta. The UI already reads "Recorded fact" and the gate copy says
  recorded (`TestStageGate_CopySaysRecordedNotAttested`), so the rename needed
  nothing.
- **Not done, so `PARTIAL`:** deriving "conversation complete" from server state
  instead of a model-written line. The finalize report still completes the
  conversation when its last line is `NO OPEN QUESTIONS` (`coach.go:312`,
  `finalizeDoneMarker`, with the instruction in `prompts.go:89` and
  `prompts_en.go:49`), all outside this cluster's files.

Four existing fixtures (in `gates_test.go`, `ledger_caps_test.go` and
`tools_fence_test.go`) recorded an 11-letter quote or text unrelated to its quote;
the fix commit lengthened those quotes and gave the texts a prefix their quote
supports. No assertion changed.

## Known ceilings, recorded on purpose

- **One announcement per run of skips.** While `critic_state.skipped` is set, a
  further skip is not announced again; a round that ran clears it.
- **Only the first of two parallel calls runs.** `parallel_tool_calls: false`
  should keep a second from arriving; if one does, it is dropped silently.
- **An Anthropic stream with no content block at all** keeps its old plain
  "empty response" error and is not retried (an existing test pins it); the
  Responses parser treats the same case as transient.
- **A decline reaches the page only through its answer.** A second tab keeps its
  card until it refreshes, and its confirm then meets the 409.

## Left for the integrator

Owner decision: is announcing a skipped critic enough, or should the finalize
gate refuse? A is built; B and C are one condition each ("The gate decision").

Code edits outside this cluster's files (INTEGRATOR TODO, all small):

- `server/internal/agent/coach.go` and `prompts.go` (the `PARTIAL` remainder of
  rv4-AIDEV-07): complete the conversation from server state (all sections
  non-empty, no open blocking finding) instead of the `NO OPEN QUESTIONS` line.
- `server/internal/agent/llm_factory_test.go`: its Responses stub stream has no
  `response.completed`, so `TestNewLLMStreamerMessagesVsResponsesPaths` now waits
  about two seconds for the retry. It still passes; add the closing event.
- `client/lib/api/requests.dart` lines 142 to 144: the comment on
  `postConversationStage` says "Declining never calls this"; declining now calls
  `declineConversationStage` (built in `client_prd.dart`, as `requests.dart` is
  not this cluster's file); optionally move it beside its sibling.
- `server/internal/agentcontext/agentcontext.go` lines 21 to 23: the comment on
  `fenceTagLike` says the coach applies the rule from its own copy; the coach now
  delegates and `TestFenceRuleHasOneSource` keeps it so.

Doc edits (INTEGRATOR TODO):

- `docs/architecture/prd-coach-agent.md`: the provider-schema paragraph (lines 23
  to 29) and "Anthropic wire format" (31 to 47): a turn is whole only after
  `message_stop` or `response.completed`/`response.incomplete`, a cut tool call
  never runs, `response.failed` is mapped, calls are filed by item id, the
  request sends `parallel_tool_calls: false`. "Tools" (71 to 77) says only
  `parseAnthropicStream` surfaces one call; both parsers do. "Streaming contract"
  (79 to 90): add `critic_skipped`. The stage section: confirm and decline are
  both writes, a skipped critic opens the finalize gate and is announced, and the
  `record_fact` evidence rule.
- `docs/architecture/backend.md` near line 122 (handler list): the decline route
  and the view fields `criticSkipped`, `criticRounds`, `pendingResolutions`.
- `docs/usage/keys/llm/llm-schema.md`, section 4, "If it fails": a gateway whose
  stream ends without `message_stop` or `response.completed` fails the turn with
  "stream ended before ..."; one that rejects `parallel_tool_calls` answers 400.
- This folder's `index.md` (wave-3 row, fragment row for `w3-coach-trust`, plan
  row 3.1), `docs/plans/README.md`, `docs/HANDOFF.md`, the changelog index files,
  and the open-findings record: rv4-AIDEV-02, -03, -05, -06, rv4-CLOSE-03 and
  LEAD-FENCE-DEDUP `FIXED`; rv4-AIDEV-07 `PARTIAL` with the remainder above; the
  gate decision as an open owner question. Optional `docs/learning/` note: a
  parser that accepts a stream that never said it finished turns a dropped
  connection into a short answer, and a source scan is a fair red test for a
  pure refactor.

No migration (the reserved slot `20261018150000` is unused); `go.mod`, `go.sum`
and `client/pubspec.lock` are unchanged.

## Verification

Run against a private PostgreSQL 18 cluster on a port of its own, migrated from
this worktree, with role passwords that exist only for the run, under a scrubbed
environment. A skipped database test counts as red; none skipped.

| Finding | Red, at the red commit | Green, at the fix commit |
| --- | --- | --- |
| rv4-AIDEV-02 | agent: "a skipped critic must tell the person on the stream; the stream carried """ for both skip reasons; "after a round that ran, state = {RoundCount:1 Skipped:contention}, want roundCount 1 and no skip marker". api: `criticSkipped = ""` and `criticRounds = 0` against the stored state. client: 3 of 4 new tests fail (the finalize card, the skipped revision round, the live frame; the reviewed-PRD guard passes) | all five Go tests and the four client tests pass |
| rv4-AIDEV-03 | api: `pendingResolutions = [], want only cf1`; "shown map[], applied map[cf1:resolved cf3:overridden]: want the same two findings in both". client: the card lines are absent in English and Chinese (the nothing-parked guard passes) | the three API tests and the three client tests pass |
| rv4-CLOSE-03 | api: the route sweep reports the classified decline route as unregistered; an agent token on it gets 404, want 403; each of the five decline tests gets 404 where it expects 200, 409 or 400. client: a decline posted nothing (`Expected: [(c1, draft)] Actual: []`) in both decline tests | route sweep and agent-token tests pass with the route registered `humanOnly`; the five decline tests and the file's five widget tests pass |
| rv4-AIDEV-05 | all 7 new parser tests fail (14 failing cases): a cut call is returned as a call (three cases), a tool-only cut turn is not truncated, seven streams without an end return a `nil` error, `response.failed` returns `nil` (two cases), `response.incomplete` text is not flagged truncated | all seven pass with the existing parser tests; the real-capture fixtures pass unchanged |
| rv4-AIDEV-06 | two calls come back as `call_2/set_section/{"kind":"problem"}{"text":"x"}` (spliced mid-JSON when interleaved); `parallel_tool_calls = <nil>` on both request paths | all three pass; a no-tools request carries no flag |
| rv4-AIDEV-07 | four cases accepted as attested: a four-letter quote, a long grounded quote under unrelated text, a four-character Chinese quote, a long Chinese quote under unrelated text; the honest-restatement guard passes | both pass; the bounds tests pin the constants and the 18 pairs |
| LEAD-FENCE-DEDUP | "prompt_fence.go:15 compiles its own <prd_data> fence pattern; fence through agentcontext.FenceData so there is one rule" | passes; the eleven existing fence tests pass unchanged |

Mutation checks a verifier can run; each was undone afterwards (the working tree
was clean after the last). Each Go check ran the whole package and each Dart
check the named test file; the tests named are all that went red.

- rv4-AIDEV-02: remove the `sse.criticSkipped` call (the two
  `...SkipIsAnnouncedOnTheStream` tests); remove `cs.Skipped = ""` in `critic.go`
  (`...ARoundThatRunsClearsTheSkipMarker`); drop the note from `coachTrustNotes`
  (the finalize-card and revision-round widget tests).
- rv4-AIDEV-03: drop either filter in `applyConversationMeta`
  (`TestGetConversationExposesTheParkedFindingResolutions` and
  `TestDeclineStageOnlyDeclinesTheStageThatIsPending`;
  `TestWhatTheViewShowsIsWhatConfirmApplies` stays green, as its set is
  consistent); drop the lines from the card (both widget tests).
- rv4-CLOSE-03: leave `pendingFindingResolutions` out of the removal (two decline
  tests); drop the pending-stage condition
  (`TestDeclineStageOnlyDeclinesTheStageThatIsPending`); make Decline post
  nothing (both decline widget tests).
- rv4-AIDEV-05: let a cut Anthropic call run (two truncation tests); accept a
  stream without `message_stop`, a Responses stream without `response.completed`,
  or a cut Responses call (one test each).
- rv4-AIDEV-06: file argument fragments under the last announced call
  (`...TwoCallsAreNeverMerged`); stop sending the flag
  (`...RequestForbidsParallelToolCalls`).
- rv4-AIDEV-07: floor 4 (three tests); support 0.0 (four); support measured
  against the uncut quote (`...MeasuresSupportAgainstTheStoredQuote`).
- LEAD-FENCE-DEDUP: compile a fence pattern in `prompt_fence.go` again
  (`TestFenceRuleHasOneSource`).

Live proof on the real binary, with a stub in place of the model provider. A
throwaway Python stub (not committed) speaks the Anthropic Messages and OpenAI
Responses streams, scripted per request (one queue for the coach, one for the
critic, told apart by the forced tool). The server built from this branch ran
against the private cluster on a loopback port of its own; a driver logged in as
a throwaway user and read the database as the owner role to assert what was
stored. The branch passes 33 of 33 checks on the Anthropic lane and 9 of 9 on the
Responses lane; the base binary under the same driver passes 13 of 33 and 2 of 9:

| Check | Base binary | This branch |
| --- | --- | --- |
| a `tool_use` cut at `max_tokens` | the half JSON is replayed into the next request: `json: error calling MarshalJSON for type json.RawMessage: unexpected end of JSON input`, an `internal error` frame, the stored answer ends `[interrupted]` | nothing runs, no `fact_recorded`, one provider request, the stored answer ends with the truncation marker |
| `tool_use` started, then the stream ends (nothing streamed) | no retry | retried once (about two seconds), the turn recovers (two requests) |
| text streamed, then the stream dies | accepted as a whole turn, `message_end` sent | `error` frame, one request, no retry |
| `record_fact` with an 11-letter quote; with text unrelated to its quote | both recorded | both refused with their messages; the ledger holds only the four honest facts |
| the critic returns nothing usable twice; a later round that runs | silent | `critic_skipped {"reason":"critic-output-failure","rounds":0}` before `message_end`, the view says `criticSkipped` and `criticRounds: 0`, two forced critic calls; after a round that runs, `criticSkipped` is absent and `criticRounds` is 1 |
| parked resolutions | not in the view | `pendingResolutions` lists the parked entry, status and severity, while the finding is still open |
| decline | route 404; a confirm after "decline" still applied | 200, stage unchanged, proposal and parked resolutions gone from the stored meta, the finding still open; a second decline 409; a confirm after decline 409; a fresh proposal confirms and resolves the finding |
| Responses request with tools | no `parallel_tool_calls` | `false` |
| two `function_call` items | arguments merged: `invalid character '{' after top-level value` | only the first runs, with its own arguments |
| `response.failed` `server_error`; `invalid_prompt` | both read as `CONTENT_REFUSAL`, not retried | retried once and recovered; `PROVIDER_CONFIG` after one request |
| Responses stream ends after text, no `response.completed` | accepted as a short answer | `error` frame, no retry |
| `response.incomplete` holding a `function_call` | `internal error`, `[interrupted]` | not executed, answer marked truncated |

Two Responses checks (the tool result going back, an empty stream retried) pass
on both binaries and do not discriminate. Not exercised: any real model provider
(the stub is the only one), the Responses forced-tool critic path and its flag
(unit tests only), the Flutter client against a live server in a browser (widget
tests only), and the agent-token refusal of the decline route (API route-sweep
tests only).

Gates, all green on this branch; the fragment's own commit changes documentation
only. `gofmt -l internal cmd` prints nothing; `go build ./...`; `go vet` over the
three touched packages; the same three packages with `go test -count=1 -v` (410
top-level tests passed, 0 failed, 0 skipped); the agent package under `-race`;
the whole server suite once, `go test ./... -count=1` (27 packages ok, 3 without
test files, 0 failed); `flutter pub get --enforce-lockfile` (`client/pubspec.lock`
unchanged); `dart format --set-exit-if-changed lib test` (275 files, 0 changed);
`flutter analyze --no-fatal-infos` (exit 0, the same 14 infos as the base, none
in added lines); `flutter test` (491 passed) and again under `TZ=Asia/Shanghai`
(491 passed); the nine `scripts/tests/check_*.py` guards under `python3 -I`;
`scripts/check-redaction.sh`; `scripts/check-docs-recorded.sh`. Every touched Go
and Dart file is at most 350 lines (largest: `coach_panel.dart` 347, `critic.go`
342, `prd_page.dart` 336). The branch touches 44 files, +1959 and -179 lines
before this fragment.
