# Product subsystems

The subsystems added in the 2026-08 product arc, layered on
[backend.md](backend.md) and [database-and-rls.md](database-and-rls.md).
Each is a projection of the same `/api/v1` contract — one truth, many
views (PRODUCT.md principle 3).

## Version control (the human Commit)

- `prd_commits` (immutable; FORCE RLS) freeze a working PRD as a named
  version: primary audience, change summary, carried unresolved items,
  decision note, and a recorded section diff versus the previous commit
  (`internal/prdcommit.Diff`). Working versions stay the mutable `prds`
  row + numbered `prd_revisions` snapshots; there are no commit
  update/delete paths.
- Routes: `POST /prds/{id}/commit`, `GET /prds/{id}/commits`.
- Readiness (`internal/readiness`): per-audience gap disclosure with
  priorities (must clarify now / should clarify this version / validate
  later) and reader-facing reasons. Never a completeness score; never a
  Commit blocker — `carryIntoCommit` is the bridge to a "form a version
  now" Commit. `GET /prds/{id}/readiness`.
- Audience views (`internal/audienceview`): the Decision / Management /
  Delivery / Validation projections — ordered section picks with whys,
  rendered read-only in the web `/prd/{id}/view` page and as audience
  markdown via `GET /prds/{id}/export?audience=…`. Projections read the
  core; nothing per-audience is stored, so views cannot drift.

## Agent vocabulary

- `GET /api/v1/agent/context?project_id&mode&focus` — compact,
  focus-scoped context (`internal/agentcontext`): brainstorming /
  implementation / review shapes, never the whole graph; every response
  embeds the vocabulary contract.
- `POST /agent/checkpoint` and `POST /agent/propose` write typed
  timeline records (`agent_checkpoint`, `agent_proposal` kinds).
  Agents never apply strategic changes and never Commit versions —
  `version` is surfaced from commits, human-only.
- Proposals may carry structured `items` (`proposal_items`: add_node
  with optional parent hint). `POST /agent/proposals/{uid}/decision`
  records the human accept/reject (`proposal_decisions`, the decision
  log; reversal allowed, history in the timeline) and `apply: true` on
  acceptance creates the items as real roadmap nodes, executed as the
  deciding human under their RLS context.

## Time-first roadmap

- `roadmap_nodes.deadline` (nullable — deadlines are commitments).
- `internal/roadmaptime`: pressure states (normal → near → high risk →
  overdue) and the Progress assembly — **Now admits deadlined work
  only**; in-progress items without a deadline surface as
  `needsDeadline`; Next caps at three by earliest deadline.
  `GET /projects/{id}/roadmap/progress`; node views carry `deadline` +
  computed `pressure`; PATCH accepts `deadline` (RFC3339 or `""`).

## Real-time collaboration

- `internal/presence`: in-memory per-project session registry (TTL
  pruned, change broadcasts). One server instance makes process-local
  state the honest v1.
- `GET /projects/{id}/presence/stream` (SSE): connecting registers the
  viewer (auth identity + per-tab client id); snapshots on join/leave;
  heartbeats double as TTL refresh; disconnect leaves. Non-members
  403. `GET /projects/{id}/presence` is the snapshot for CLI/MCP.
- Cursors: `POST /projects/{id}/cursor` broadcasts ephemeral
  `event: cursor` frames (session, name, x/y in canvas content space)
  on the same stream; nothing is stored; the web canvas renders other
  sessions as labeled pointers and drops them when presence leaves.

## GitHub evidence loop

- Webhook `POST /api/v1/hooks/github` (`insideout.yalotein.net`):
  HMAC-SHA256 verification against `INSIDEOUT_GH_WEBHOOK_SECRET`
  (constant time; unset → 503). `push`/`pull_request` deliveries
  resolve the repo's projects via the SECURITY DEFINER helper
  `_projects_by_repo` (the webhook has no RLS identity) and re-run the
  per-project commit sync as each project owner.
- The repo's committed `insideout.yaml` matching guide (scaffolded by
  `GET /projects/{id}/guide`, `insideout guide`, or the MCP `guide`
  tool) maps branches/labels/paths to roadmap nodes; matched leaves
  append idempotent evidence rows (`roadmap_evidence`, unique
  (node, kind, detail) so redeliveries write once). Evidence attaches
  to leaves only and never proves outcomes.
- GitHub App identity (`INSIDEOUT_GH_APP_ID` + private key, escaped or
  `_FILE`) mints installation access tokens (`internal/github` JWT +
  cache) to read the guide from private repos; public repos fall back
  to unauthenticated fetch.
