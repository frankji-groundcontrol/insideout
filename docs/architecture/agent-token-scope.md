# Agent token scope

Design for rv4-FEAT-01 and rv4-AIDEV-04 (review 4, plan row 2.5), with the output
fencing of rv4-AIDEV-01 and rv4-AIDEV-08. Written before the code; the tests and
the changelog fragment (`docs/changelogs/2026-10-18-rv4-remediation/w2-agent-scope.md`)
record what shipped.

## Problem

PRODUCT.md says agents "cannot directly apply strategic changes, Commit a product
version, or Merge a Product Branch". Nothing enforces it. `insideout-mcp` calls the
same `/api/v1` routes as the web with the user's own access token, the server
accepts any valid token on every route, and the MCP registry lists all 52 tools.
One session can `propose`, `proposal_decide` with `apply`, then `commit`. The
"agent vocabulary" is prose in `SKILL.md`.

## Scope model

A token has one of two scopes.

- `full`: the claim is absent. Every token issued before this change, every browser
  cookie and every saved CLI login is `full`, so nothing already issued breaks and
  no migration is needed.
- `agent`: the JWT carries `"scope":"agent"`. Same secret, same `sub`, same expiry
  rules. Any other `scope` value is rejected (401), so a scope added later never
  falls back to full power on an older server.

An agent token is minted by `POST /api/v1/agent/token`: authenticated, deliberately
not under `/auth` (it needs no throttle table), and answered `{accessToken,
expiresAt, scope}` with no cookie and no refresh token. Its lifetime is the
access-token lifetime (`INSIDEOUT_ACCESS_TTL`, default 15 minutes). The route is
itself human-only, so an agent token can never mint another one or widen itself.

## Where it is stored and checked

Stored in the signed token only: no table, no migration (the reserved slot
`20261018130000_agent_token_scope.sql` stays unused).

Checked in one place. `requireAuth` is the only code that verifies an access
token, and every authenticated route, including `requireAdmin`, goes through it.
After verifying, it looks up `r.Pattern` ("METHOD /path", set by the mux) in a
central table, `agentRoutePolicy` in `server/internal/api/agent_scope.go`:

- `full` tokens pass unchanged.
- `agent` tokens pass only on a route marked agent-allowed. Everything else is 403
  with code `agent_scope_denied`. A pattern missing from the table is treated as
  human-only (fail closed), and a route-table test fails the build when a
  registered route has no entry, or an entry has no route.
- The scope rides on the request context, and `withLogging` records `scope=agent`.

Curl, the web client and any other client meet the same wall: it is the server's
rule, not the MCP binary's.

## What each scope may do

The agent scope is exactly the routes the MCP tools call, minus the human-only
ones below, nothing wider: 36 of the 72 authenticated routes (the 71 that exist
today plus the mint route). A route joins the agent list only when a tool needs it.

Agent-allowed: `GET /me`; workspaces (list, create, join); projects (list, create);
ideas (list, create, update, convert); PRDs (read, update, view, readiness, export,
commits list, revisions list and create, and `POST /prds/{id}/status` to `draft` or
`reviewing` only); roadmap (list, add, update, move, guide, progress, evidence
read) and the four AI build/expand routes; repo bind and GitHub sync;
`agent/context`, `agent/checkpoint`, `agent/propose`; the presence snapshot.

Human-only (the other 36): proposal decision (`POST /agent/proposals/{uid}/decision`);
Commit (`POST /prds/{id}/commit`); all six `/admin/*` routes (`signup_mode`,
`invite_*`, the mail probe); every `DELETE` (ideas, roadmap nodes, projects,
updates, workspaces, members); member roles; workspace and project edits and
timeline updates; `PATCH /me`; the coach conversation routes; evidence entry and
validation; the SSE streams and cursor; the single-record reads no tool calls
(`GET /ideas/{iid}`, `GET /projects/{pid}`, `GET /workspaces/{id}` and its member
list); `POST /prds/{id}/status` to `approved` or `rejected` (a workspace-admin
review gate, so an agent holding an admin's token cannot approve its own
submission); and `POST /agent/token`.

Password flows are the public `/auth/*` routes. They take no access token, so a
scope cannot gate them, and an agent token gives no extra reach there (a reset
needs the emailed token, a refresh needs a refresh token, which agent tokens do not
have). The MCP profile drops the password tools so a password never enters the
conversation.

## A proposal's author cannot decide it

An agent proposes through an agent-allowed route, so the agent actor is the author
of every proposal it files. The decision route is closed to that whole actor kind.
This is stricter than "not the same actor" and needs no per-proposal bookkeeping:
the store sees user ids only, and the agent runs under the user's id, so comparing
ids could never tell them apart. `store/proposal_decisions.go` is unchanged.

## MCP profile and exchange

`insideout-mcp` keeps holding the user's credential only to trade it. For every
tool call it resolves credentials as before (INSIDEOUT_TOKEN, the saved login, an
in-process sign-in), and if the token is a full-scope JWT it calls
`POST /agent/token` through the credential's own client (so the saved session
renews as usual) and sends the tool call with the agent token alone. The agent token
is cached until it is about a third of its life from expiry, then minted again. A
failed exchange is a tool error: it never falls back to the full token.

The registry follows the credential. When the credential in use is a JWT (at
startup, or right after an in-process sign-in), the profile removes `commit`,
`proposal_decide`, `signup_mode`, `invite_create`, `invite_list`, `invite_revoke`,
`idea_delete`, `roadmap_delete` and the password tools (`auth_register`,
`auth_login`, `auth_forgot_password`, `auth_reset_password`). The passwordless
sign-in tools stay so an unauthenticated process can still be signed in. A token
that is not a JWT cannot carry a scope and no InsideOut server accepts it, so it is
passed through unchanged and the registry stays whole. Removal is advice to the
model; the 403 above is the guarantee.

## Fencing and labelling

User-writable text in tool output is data, not instructions.

- `get_prd` (`agent/tools.go`) returns its JSON inside the existing `<prd_data>`
  fence (`fenceData`), the clause the system prompts already carry.
- `agentcontext.Assemble` keeps its field values (callers and tests read them) and
  adds an `untrusted` object naming the fields that hold user or GitHub text, with
  the data-not-instructions notice. It also exports `FenceData`, the one fence
  implementation reachable from outside package `agent`; a parity test pins it to
  `agent.fenceData`, whose file this cluster does not own.
- `mcpserver/fence.go` wraps every product tool result: a one-line untrusted-data
  label, then the answer inside `FenceData`, with tag-like text broken so stored
  text cannot close the fence early.

## Limits, said plainly

- An agent that can read the user's saved config file or environment holds the full
  credential and can use it directly. The scope protects against an agent confined
  to MCP tools, against prompt injection steering such an agent, and against a
  leaked MCP-process token. Running the agent without the user's home directory is
  the hardening beyond this.
- The agent acts under the user's id, so audit rows still carry the user. Logs add
  `scope=agent`; per-row actor attribution is out of scope here.
- Single-user workspaces: the owner is also the author. The strict default
  (an agent token never decides) holds; DEFERRED_OWNER: should an agent token be
  allowed to decide a proposal when the human owner is also the author in a
  single-user workspace, or must decisions always require an interactive human
  session?

## Verification

Real PostgreSQL through the whole handler chain, with agent tokens built by the test
itself so the red run exercises today's server: an agent token is refused on
decision, commit, `PUT /admin/settings`, invitations, deletes and the mint route,
passes on allowed reads, and the author-actor case (agent proposes, agent decides:
403, human decides: 200). Then a live server, `curl`, and the built `insideout-mcp`
over stdio with `INSIDEOUT_ACCESS_TTL=3s` to prove re-minting.
