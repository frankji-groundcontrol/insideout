# CLI, MCP, and the guidance system

How to drive InsideOut from the terminal, from any MCP-capable agent,
and how to connect a GitHub repository so its activity feeds your
roadmap. Everything here is a projection of the same `/api/v1`
contract the web client uses.

## Setup

From a checkout of this repository (needs Go 1.26+, see
`server/go.mod`), the installer builds and installs both binaries
into `~/.local/bin` (override with the first argument or
`INSIDEOUT_INSTALL_DIR`) and adds that directory to PATH via your
shell rc when missing. The binaries call the hosted API unless
`INSIDEOUT_API` is set. `./scripts/default-api.sh` prints the URL.
Since the `v0.1.0` release (2026-10-11),
`curl -fsSL https://github.com/frankji-groundcontrol/insideout/releases/latest/download/bootstrap.sh | bash`
(or `scripts/bootstrap.sh` from a checkout) installs prebuilt,
checksum-verified binaries with no checkout and no Go toolchain.
The full guide is [installation.md](./installation.md).

```bash
./scripts/install.sh
```

Then `insideout setup` is the front door — numbered questions on a
terminal (connect to the hosted service, or a PostgreSQL server on
this machine; secrets hidden as you type; the login saved to a 0600
file the CLI reads), or flags for agents:

```bash
insideout setup                       # interactive (needs a terminal)
insideout setup --hosted              # confirm the hosted path
insideout setup --local --postgres bundled|own [--no-start]
insideout setup --check [--llm]       # re-run the health probes now
insideout whoami                      # "not signed in" until sign-in
```

`./scripts/setup.sh` remains the pre-binary fallback (before
`install.sh` has run) and covers the same local-server ground.
Manual fallback (the CLI ships inside the server binary):

```bash
cd server && go build -o /usr/local/bin/insideout ./cmd/insideout
go build -o /usr/local/bin/insideout-mcp ./cmd/insideout-mcp
```

To register the MCP server with an agent, merge the standard
`mcpServers` entry shown under [MCP server](#mcp-server) — the same
snippet ports across Claude Code, Cursor, and anything speaking the
standard shape. It needs no `env` or token.

`login` prompts for the password on stderr; the token is the only
stdout. A successful `login` or `register` also saves the session
(server, access token, refresh token) to the 0600 config file and
prints its path on stderr, so the next verb is signed in; a saved
session renews itself once when its access token expires. Without a
saved session or `INSIDEOUT_TOKEN`, read verbs fail with `not signed in`
and say how to sign in. Signup on the hosted service takes an invitation code
from the server's administrator:
`insideout register --invite <invitation-code> you@example.com`.
`insideout logout` ends the saved login: it revokes that session on the
server, then deletes `<OS config dir>/insideout/config.json` when the file
holds nothing else, or clears only the token fields when the file holds
more. With no saved login it prints `not signed in` and exits 0. If the
server cannot be reached, the file is still removed and the command says
the server session could not be revoked and expires by itself.
`insideout reset-password` clears a saved login for that server too, because
the reset revokes every session and the file would otherwise keep a dead pair.

Every way the web can sign in works here too (prompts on stderr, token
on stdout, session saved):

```bash
insideout register you@example.com "Display Name"   # password prompt
insideout login you@example.com                     # password prompt (default)
insideout login --code you@example.com              # emailed 6-digit code prompt
insideout login --magic you@example.com             # paste the emailed link (or token)
insideout forgot-password you@example.com           # emails a reset link
insideout reset-password                            # paste link/token + new password
```

`login --code`, `login --magic`, `forgot-password` (and MCP
`auth_login_code_request`, `auth_magic_link_request`) print the server's email
status: `accepted for delivery`, or `logged on the server, not delivered` when
the server has no mail provider (ask its operator for the code or link).

## Verbs

Every row runs on the CLI. MCP has a tool for each verb an agent token may
call and none for a human-only route (see [Agent scope](#agent-scope)).

| Area | Verbs |
| --- | --- |
| Auth | `register [--invite <invitation-code>] <email> [display-name]`, `login [--code\|--magic] <email>`, `logout`, `forgot-password <email>`, `reset-password`, `account delete` (the current password from the hidden prompt; see [account deletion](./account-deletion.md)) |
| Authoring | `prd update [--title] [--section k=v]… [--status] <id>`, `evidence <node-id>`, `evidence add --detail D [--source URL] [--observed YYYY-MM-DD] <node-id>` (a validation observation on a leaf; the day defaults to today), `workspaces create --title [--desc]`, `workspace update [--title] [--description D\|clear] [--cover URL\|clear] <ws>`, `join <code>`, `projects create --title [--desc] <ws-id>`, `project update [--title] [--description D\|clear] [--status planning\|active\|paused\|done\|archived] [--owner user-id\|me\|clear] <proj>`, `project delete --yes <proj>` (workspace admins), `workspace delete --yes <ws>` (its creator), `update add --kind progress\|blocker\|note --content C <proj>`, `update edit --content C <update>`, `update delete <update>` (author or admin), `member role --role admin\|member <ws> <user>`, `member remove <ws> <user>` (admins), `profile update [--username U] [--bio B\|clear] [--keywords a,b\|clear] [--avatar URL\|clear]` |
| Read surface | `whoami`, `workspaces`, `workspace <ws>`, `projects <ws>`, `project <proj>`, `updates [--before c] [--limit n] <proj>`, `prds <ws>`, `prd <id>`, `revisions <prd>`, `view [--audience] [--export F] <prd>`, `export [--format markdown\|print] [--audience A] [--out FILE] <prd>` (the whole PRD on stdout or in a file, as MCP `prd_export` gives it) |
| Versions | `commit --name --audience [--summary --unresolved… --note] <prd>`, `versions <prd>`, `readiness <prd>` |
| Ideas | `ideas <ws>`, `idea create --title [--content] <ws>`, `idea update --title [--content C\|clear] <id>` (an absent or empty `--content` keeps the body), `idea delete <id>` (soft drop), `idea convert [--locale en\|zh] <id>` (the coach's language; default from `LC_ALL`, `LC_MESSAGES` or `LANG`, else `en`; the MCP tool `idea_convert` takes the same `locale`) |
| Roadmap | `roadmap list [--root node] <proj>` (with `--root`, that node and the nodes below it), `roadmap add`, `roadmap update [--title] [--description D\|clear] [--status] [--deadline RFC3339\|clear] [--acceptance A\|clear] [--owner user-id\|me\|clear] [--completion-reason S\|clear] <node>`, `roadmap move [--parent id \| --root] [--position n] <node>` (no `--parent` keeps the parent; the MCP `roadmap_move` takes `parent_id` `"root"` for a root), `roadmap delete [--force] <node>` (a branch holding evidence or done or validated nodes is refused with the counts it would erase; `--force` deletes it anyway), `roadmap validate --reason R <node>` (confirm a done node as validated), `roadmap progress [--root node] <proj>`, `roadmap presence <proj>` |
| Task tree | `roadmap show <node>` (the node, its ancestors root first, its project), `roadmap mine <ws>` (your topmost owned nodes), `members <ws>` (user ids for `--to` and `--owner`), `roadmap delegate --to <user-id> [--note N] <node>`, `roadmap handback [--note N] <node>`, `roadmap assignments <node>` (who held it, newest first). MCP: `roadmap_node`, `roadmap_mine`, `roadmap_assignments` (no MCP tool for `members`, delegate or hand-back: every MCP tool acts through an agent-scope token, which those routes refuse); `roadmap_update` takes `acceptance` (`"clear"` clears it) but no `owner_id` or `completion_reason`, which an agent token is refused, `roadmap_list` and `progress` take `root`. Node views carry `canWrite` and `rollup` ([roadmap tree](../architecture/roadmap-tree.md)) |
| Build | `build <prd>` drafts a roadmap from an approved PRD and writes nothing (409 `prd_not_approved` otherwise); `build-confirm [-expected-count N] [-force] -tree '<json>' <prd>` writes it, creating and linking the project on the first build (PRD author or workspace admin only); over a roadmap with evidence or done or validated nodes it answers 409 `roadmap_replace_has_history` with the counts until `-force`; `expand <node>`, `expand-confirm -children '<json>' <node>` |
| Agent | `agent-context [--mode M] [--focus node] <proj>`, `checkpoint [--node] <proj> <summary>`, `propose --kind K [--item Title[@Parent]]… <proj> <summary>`, `proposal decide --accept\|--reject [--no-apply] [--reason R] <update-id>` (acceptance applies the proposal's items unless `--no-apply`, as on the web; `idea proposal-decide` is the old spelling) |
| Admin | `invite [--ttl hours] <email>` (1 to 720 hours, default 168; `POST /admin/invitations` answers 400 for another `ttlHours`), `invite revoke <id>`, `invites`, `signup-mode [invitation_only\|open\|closed]` (admin token required) |
| Server operator | `migrate`, `seed`, `promote-admin <email>` (creates the account, password on stdin, when absent); these need the server config (`DATABASE_URL`, `INSIDEOUT_JWT_SECRET`), and `promote-admin` also the owner DSN (`DATABASE_OWNER_URL`) |
| Front door | `help` (also `-h`, `--help`) and `version` (also `--version`) answer before any server config, so they work with no database; an unknown verb exits 2 with a pointer to `help` |
| Guidance | `guide [--out insideout.yaml] <proj>`, `repo set <proj> <repo-url>`, `sync <proj>`, `snapshot [--note] <prd>` |

Where a CLI verb and its MCP tool are named differently:

| CLI | MCP |
| --- | --- |
| `ideas`, `idea create`, `idea update`, `idea convert` | `idea_list`, `idea_create`, `idea_update`, `idea_convert` |
| `evidence <node>` | `roadmap_evidence` |
| `join <code>`, `workspaces create` | `workspace_join`, `workspace_create` |
| `projects create` | `project_create` |
| `prds <ws>`, `updates <proj>`, `export <prd>` | `prd_list`, `project_updates`, `prd_export` |
| `project update`, `update add`, `update edit`, `workspace update` | `project_update` (no `owner`), `update_add`, `update_edit`, `workspace_update` |
| `roadmap list`, `roadmap show`, `roadmap mine`, `roadmap add`, `roadmap update`, `roadmap move`, `roadmap assignments` | `roadmap_list`, `roadmap_node`, `roadmap_mine`, `roadmap_add`, `roadmap_update`, `roadmap_move`, `roadmap_assignments` |
| `roadmap progress`, `roadmap presence` | `progress`, `presence` |
| `repo set <proj> <url>` | `repo_set` |
| `agent-context`, `build-confirm`, `expand-confirm` | `agent_context`, `build_confirm`, `expand_confirm` |
| `signup-mode` (no argument) | `signup_mode` |

Flags precede positional ids (Go convention). Statuses are
`locked|pending|in_progress|done`; audiences are
`decision|management|delivery|validation`.

Every request has a deadline: 5 minutes for the two calls that wait on a model
(`POST /prds/{id}/build` and `POST /roadmap/{id}/expand`), 20 seconds for the
rest, and an MCP tool call is bound by the same numbers. Ctrl-C (or SIGTERM)
ends the request in flight and the command with status 130 and
`insideout: interrupted`; a command still running 400 ms later, or a second
signal, ends the process at once (rv4-QUAL-04, rv4-REL-06).

### Absent, empty and clear

Every edit verb follows one rule on the CLI and on MCP: a flag or argument you
leave out, or pass empty (`--description ""`, `"description": ""`), leaves that
field as it is; the word `clear` clears a field that can be empty
(description, deadline, acceptance, owner, completion reason, cover, an
idea's body, bio, avatar); a title or status cannot be cleared. Both fill
one request struct in `server/internal/apiclient` (`apiclient.PatchText`).
PRD sections are the exception: `prd update --section key=` and
`sections_json` write the text you pass, an empty one included.

### Lists, paging and the rest of the contract

- `prds <ws>` (MCP `prd_list`, `GET /api/v1/workspaces/{id}/prds`) lists
  every PRD of a workspace to its members, newest first: `id`, `title`,
  `status`, `authorId`, `authorName`, `projectId` (null until built) and
  `updatedAt`. Every project view carries `prdId`, the PRD it was built from,
  or null.
- `project <id>` (`GET /api/v1/projects/{pid}`) embeds the newest 50 updates
  and a `nextCursor` (null when no older update exists). `updates [--before
  <nextCursor>] [--limit 1..100] <proj>` (MCP `project_updates` with `before`
  and `limit`, `GET /api/v1/projects/{pid}/updates`) answers
  `{"updates": [...], "nextCursor": ...}` with the next page; without
  `before` it starts from the newest, and `limit` defaults to 50. A `before`
  that is not a UUID answers 400; a cursor whose update was deleted between
  pages (or that is no update of this project) answers 404, not an empty page.
- `project update`, `workspace update`, `profile update` and their MCP
  tools read the row first, because the server rewrites title, description,
  status, username and bio on every PATCH; what you leave out is sent as read.
- `GET /api/v1/roadmap/{nid}/evidence` (the `evidence` verb) answers 404
  for a node that does not exist or that you cannot see. `createdAt` is a
  timestamp like every other, and a validation observation's `observedAt`
  is the date as entered (`YYYY-MM-DD`).
- `PATCH /api/v1/me` and `PATCH /api/v1/workspaces/{id}`: an absent
  `avatarUrl` or `coverUrl` keeps the picture; `null` or `""` clears it.

### Errors

Every error body is `{"error": "<message>", "code": "<code>"}`. The code is
lowercase snake_case: a specific one where a client acts on it
(`agent_scope_denied`, `app_throttle`, `github_repo_not_visible`), otherwise
the status's name (`not_found`, `bad_request`). Extra keys are camelCase
(`retryAfterSeconds`, `liveCount`).

A proposal's `detail`, a decision's `reason`, a commit's `changeSummary`,
`decisionNote` and each `unresolved` item, a revision's `note` and a project's
`description` hold at most 2000 characters; a longer one answers 400
`field_too_long`. A checkpoint's `nodeId` must be the id of a node of that
project: 400 for a value that is no id, 404 for a node it cannot find there.

## MCP server

`server/cmd/insideout-mcp` is a stdio MCP server exposing the same
verbs as tools (ask `tools/list` for the current set — a count written
here drifted twice, and the set now depends on the login, see "Agent
scope" below). It resolves the server and the token exactly like
the CLI (`--api`, `INSIDEOUT_API`, the saved `insideout login`, then the
hosted API; `INSIDEOUT_TOKEN` or the saved token), so the entry needs
no `env`:

```json
{ "mcpServers": { "insideout": { "command": "insideout-mcp" } } }
```

- Claude Code: `claude mcp add insideout -- insideout-mcp`, or
  `.mcp.json` in the repo root. Cursor: `~/.cursor/mcp.json` (global)
  or `.cursor/mcp.json` (project). Other agents: their `mcpServers`
  block. Reload MCP servers afterwards.
- A client that does not inherit your shell `PATH` needs the absolute
  `command`: `command -v insideout-mcp` (`~/.local/bin/insideout-mcp`
  after `scripts/install.sh`).
- The server is fixed when the process starts: the saved login's
  tokens are only ever sent to the server they were issued by. After
  `insideout login --api <other server>`, restart the MCP server. Add
  `"env": { "INSIDEOUT_API": "http://127.0.0.1:8080/api/v1" }` only to
  pin a server other than the saved login's; an `INSIDEOUT_TOKEN` in
  `env` is an explicit credential and is never renewed.

### Agent scope

An MCP process is an agent, and an agent must not be able to do what only
the human should. When the credential it holds is a JWT (the saved login,
`INSIDEOUT_TOKEN` or an in-process sign-in), `insideout-mcp` trades it for a
short-lived **agent-scope token** (`POST /agent/token`, one access-token
life, cached and minted again before it expires) and sends every tool call
with that token alone. The server refuses an agent token on the human-only
routes with 403 `agent_scope_denied` (the proposal decision, Commit, the
admin routes, every delete, delegate, hand-back and the member list), so the limit holds for any client, not just
this binary ([design](../architecture/agent-token-scope.md)). It also refuses
an agent's `build_confirm` over a roadmap that is not empty, and an agent's
`completionReason` or `ownerId` on `PATCH /roadmap/{id}` and any `ownerId` on
`PATCH /projects/{pid}` (an agent suggests an owner through `propose`, which
only a plan writer's agent may file).

MCP registers no tool for a human-only route, since no token the process
sends can reach one. These are CLI and web only, on purpose: Commit
(`commit`), the proposal decision (`proposal decide`), account deletion
(`account delete`), validation (`evidence add`, `roadmap validate`), every
delete (`idea delete`, `roadmap delete`, `project delete`, `workspace
delete`, `update delete`, `member remove`), member roles and the profile
(`member role`, `profile update`), a project's owner (`project update
--owner`), delegation, hand-back and the member list
(`roadmap delegate`, `roadmap handback`, `members`), and the admin verbs
(`invite`, `invites`, `invite revoke`, `signup-mode <mode>`). An agent reads
and edits a project or workspace (`project`, `project_update`, `workspace`,
`workspace_update`), posts and corrects updates (`update_add`,
`update_edit`), and pages the feed and the PRD list (`project_updates`,
`prd_list`). The registry also follows the login:
the four password tools (`auth_register`, `auth_login`,
`auth_forgot_password`, `auth_reset_password`) are not offered while the
process holds a JWT login, so a password never enters the conversation.
Before any sign-in they are listed, so a process can still sign in.

- `"env": { "INSIDEOUT_AGENT_ROOT": "<node-id>" }` confines the agent token to
  one node's subtree: node writes outside it answer 403 `agent_scope_denied`,
  reads stay workspace-wide, and `agent_context` focuses on that node by
  default.
- A failed exchange is a tool error and never falls back to the full token.
  An `INSIDEOUT_TOKEN` that already carries a scope is sent as it is (the
  server decides), so `INSIDEOUT_AGENT_ROOT` beside it does nothing and the
  process prints one warning line on stderr at startup; a token that is not a
  JWT is passed through and the registry stays whole.
- Deploy the server before the MCP binary: a new binary against a server
  without `POST /agent/token` fails every tool with a hint to update the
  server.
- Tool results arrive under a one-line label and inside one `<prd_data>`
  fence: text people wrote (PRD sections, titles, notes) is data, not
  instructions. The `agent-context` response carries an `untrusted` block
  naming those fields; the CLI's `context` and `evidence` verbs print the
  JSON as it is and do not fence.
- The scope defends an agent confined to MCP tools. It does not protect
  against an agent that can read your saved config file or environment,
  which holds the full credential.

Signup policy: registration is invitation-only by default
(`GET /auth/config`, MCP `signup_mode` reads it); admins mint per-email
invitations (`invite`, revoke with `invite revoke`) and switch the mode
(`signup-mode <mode>`, the web admin panel on the profile page). These are
CLI and web only (see "Agent scope"): the server refuses an agent token on
`/admin/*`. `invite` prints the whole
response, including `mailSent`: true only when the provider accepted the
e-mail, so when it is false (dev mode, or a provider refusal) nothing was
e-mailed and you send the `signupUrl` to the invitee yourself. The first
admin is created out-of-band with `insideout promote-admin <email>`
(server subcommand, owner DSN): when the address has no account it
creates one, reading the new password from stdin (never from argv), and an
existing account is promoted with its password untouched.

Auth can also happen at runtime: the `auth_*` tools
(`auth_register`, `auth_login`, `auth_login_code_request` +
`auth_login_code_verify`, `auth_magic_link_request` +
`auth_magic_link_consume`, `auth_forgot_password`,
`auth_reset_password`, `auth_logout`) sign the server **process** in —
the token is kept in memory and never echoed into the conversation, and
every later tool call rides that session. `auth_logout` drops that
in-memory session only; the saved login file stays until `insideout logout`. Once the process holds a JWT
login, the four password tools are no longer offered (see "Agent scope"),
and neither are `auth_login_code_verify`, `auth_magic_link_consume` and
`auth_logout`: restart `insideout-mcp` to change identity, which is also
what its errors say. The code flow is the
agent-friendly one: the human gets the email and supplies the digits, so
no password ever appears in a transcript. (Until 2026-08-24 there was
deliberately no login tool; the all-surfaces-agree product call reversed
that — a login tool's token is process state, not conversation state.)

## The guidance system (`insideout.yaml`)

Connecting a repository so pushes and PRs feed your roadmap:

1. **Bind the repo**: `insideout repo set <project-id>
   https://github.com/owner/repo` (web: the project's repo field). Accepted:
   `owner/repo` or a github.com URL (the host must be exactly `github.com`);
   the server stores `https://github.com/<owner>/<repo>`. A repository belongs
   to one project: a second project linking it, in any letter case, gets 409
   `github_repo_already_linked`. When the GitHub App is configured, the
   repository must be public or covered by an installation of the App, else the
   answer is 400 `github_repo_not_visible` (rv4-COMP-05).
2. **Scaffold the guide**: `insideout guide --out insideout.yaml
   <project-id>` — generated from your roadmap: every leaf node with
   editable matchers, branch nodes documented as comments. On the web, the
   project's GitHub card has **Get insideout.yaml**, which saves the same
   file (or copies it where no file can be saved).
3. **Edit and commit** the file at the repo root:

   ```yaml
   version: 1
   nodes:
     <node-uuid>:
       title: "交付 MVP"
       branches: [main, "feature/*"]   # exact or prefix ending in /*
       labels:   [roadmap/mvp]        # exact PR label
       paths:    ["server/"]           # prefix of any touched file
   ```

4. **Install the GitHub App** on the repository (one-time; the app's
   webhook is already configured).

From the next push, GitHub delivers to
`https://<your-domain>/api/v1/hooks/github` (your deployment's domain); the
delivery is
HMAC-verified, the guide is fetched (installation token for private
repos), and matched **leaf** nodes gain evidence rows — commits show
activity, opened PRs review, merged PRs implementation, and a successful
deployment status release (a deployment merely created is activity; the App
must deliver `deployment_status` events). Unmatched activity stays visible but
attaches to nothing;
evidence never auto-proves outcomes. Redeliveries are idempotent.
Re-generate the guide any time with `insideout guide`; node ids are
stable, so committed matchers survive regeneration. A rebuild from the PRD
(`build-confirm`) is the exception: it replaces every node with new ids and,
with `-force`, erases their evidence, so regenerate and recommit the guide
after one.

## What agents may and may not do

`agent-context` embeds the vocabulary contract: agents **checkpoint**
work and **propose** structure/scope/priority changes (optionally with
structured items a human can apply on acceptance); a human **decides**
— and only humans Commit versions. See
[architecture/product-subsystems.md](../architecture/product-subsystems.md).
