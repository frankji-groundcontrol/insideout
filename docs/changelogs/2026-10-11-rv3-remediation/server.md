# Server changes — review-3 remediation (waves 1, 2, 4)

Commits: wave 1 `5c8f54f..c89faea`, wave 2 `c89faea..ce8e173`,
wave 4 `b056e30..4464330`. Every fix below landed with its red test committed
first; the register rows each change closes are named.

## Agent (coach, critic, planner)

- **Critic findings cannot be silently rewritten** (AIDEV-01): a later ranked
  restatement merges into stored findings instead of erasing them
  (`88e5da4`), and the findings parsers drop sections outside the canonical
  eight (AIDEV-12, `22a37d2`).
- **`resolve_finding` no longer flips statuses on the model's say-so**
  (AIDEV-07, wave 1 + integrator seam): the request is parked in
  `meta.pendingFindingResolutions` and applied atomically only when a human
  confirms the stage — `ConfirmConversationStage` takes a `MetaTransform`,
  `agent.ApplyStageConfirmation` resolves by ID, keeps only
  resolved/overridden statuses and appends a `findingResolutionsApplied`
  decision trail.
- **Untrusted text is fenced in every prompt** (AIDEV-02, `d6e1fef`);
  the clarify prompt follows PRODUCT.md's coaching contract (AIDEV-08,
  `598c73f`); the draft-gate copy says *recorded*, not *attested* (FEAT-05
  half, `f0f5b26`).
- **Turn and stream discipline** (wave 2): a coach turn has a total deadline a
  trickling stream cannot outlive (AIDEV-11, `f65b211`); completion keys on
  the marker as the report's final line, not a substring (AIDEV-05,
  `84ec698`); Responses output tokens are capped at 4096 like the Anthropic
  streamer (AIDEV-06, `628508f`); residual 4xx and unknown stream codes
  classify as provider config, not breaker food (REL-03, `2b7ba1c`).
- **Ledger caps** (AIDEV-10, `69e59ad`): ledger text and assumptions are
  capped per conversation so context cannot grow without bound.
- **The offline template coach discloses in-band** (honest degradation,
  `1168164`): the streamer echoes plainly and says it is offline instead of
  impersonating a live model.
- **Locale threading** (FEAT-01, wave 4): the conversation remembers the
  locale it was created with (`0a781c5`, migration `20261011110000`), the
  coach and critic speak it (`48b437b`), and the planner drafts the roadmap
  in it (`ef582ba`).

## Store and migrations

- **Circuit-breaker rows are named** (TEST-06, `2c5631b`): migration
  `20261011100000` drops the zero-arg `ai_check_circuit` /
  `ai_record_circuit_result` and recreates them with a row-id parameter, so
  rules tests pin their own row instead of the production global one.
- **App-pool statements are bounded** by a 15s server-side timeout
  (REL-05, `f63d878`); **a 30-day reaper** purges `ai_runs` and expired auth
  rows within retention (REL-02, `7b08881`); the dead
  `SharesWorkspaceWith` (whose query failed closed) is deleted
  (`e9441b3`).
- **Validation leg** (FEAT-04, wave 4): migration `20261011140000` plus the
  store path (`acaa5a1`) — a human observation is evidence, and a milestone
  reaches `validated` only through that human confirm.

## API

- **Stage confirmation is a real door** (USER-01, `f822873`): the
  conversation GET exposes `meta.pendingStage` so a reload shows the
  proposal, and the confirm endpoint drives the coach's `ConfirmStage`.
- **The native coach stream survives 30 silent seconds** (REL-01, `6210703`):
  the coach POST turns off the shared receive timeout so long thinking turns
  are not cut mid-stream.
- **A second api-server process against one database announces itself**
  (singleton guard, `a49936a`) — the single-replica contract is loud instead
  of silently split-brained.
- **Admin mail probe** (`3b9552f`) gives the funnel a delivery signal;
  **avatar/cover URL schemes are validated at the door** (SEC-04, `b2479c4`);
  `deployment_status` deliveries append deploy-failed evidence instead of
  vanishing (FULL-02, `a82d82d`).
- **Auth emails are testable and honest** (wave 1): invitation HTML expiry
  copy matches the text at day TTLs (`c346b25`), the shared button is cobalt
  squared per DESIGN.md (`7c7fc90`), the footer phrases the link in the
  link's own units (`d3eea51`), and the templates live in their own file.
- **GitHub surface** (wave 2): repo segments are validated and the
  guide-fetch URL escapes raw refs (SEC-02, `4fd1615`); the guide-file size
  cap is enforced while the body streams (SEC-03, `40d2431`).
- **MCP parity** (AIDEV-04, `efb107e`): the commit tool's schema declares
  its `unresolved` argument.
- **ConvertIdea carries the idea body** (FEAT-02, `f172d76`): the body seeds
  the coach conversation's first message instead of being dropped.

## Verifier-visible gates

`go test ./...` over 26 packages with exactly one allowed skip
(`TestReadSecretTerminal`) against a real PostgreSQL clone —
[verification.md](verification.md).
