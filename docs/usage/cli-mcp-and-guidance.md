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
The full guide is [installation.md](installation.md).

```bash
./scripts/install.sh
```

Then `insideout setup` is the front door — an interactive form on a
terminal (remote sign-in or a local PostgreSQL server, secrets
hidden as you type, the result saved to a 0600 config file the CLI
reads), or flags for agents:

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

## Verbs (parity with web and MCP)

| Area | Verbs |
| --- | --- |
| Auth | `register [--invite <invitation-code>] <email> [display-name]`, `login [--code\|--magic] <email>`, `forgot-password <email>`, `reset-password` |
| Authoring | `prd update [--title] [--section k=v]… [--status] <id>`, `evidence <node-id>`, `workspaces create --title [--desc]`, `join <code>`, `projects create --title [--desc] <ws-id>` |
| Read surface | `whoami`, `workspaces`, `projects <ws>`, `prd <id>`, `revisions <prd>`, `view [--audience] [--export F] <prd>` |
| Versions | `commit --name --audience [--summary --unresolved… --note] <prd>`, `versions <prd>`, `readiness <prd>` |
| Ideas | `ideas <ws>`, `idea create --title [--content] <ws>`, `idea update --title [--content] <id>`, `idea delete <id>` (soft drop), `idea convert [--locale en\|zh] <id>` (the coach's language; default from `LC_ALL`, `LC_MESSAGES` or `LANG`, else `en`; the MCP tool `idea_convert` takes the same `locale`) |
| Roadmap | `roadmap list/add/update/move/delete`, `roadmap update --deadline RFC3339\|clear`, `roadmap progress <proj>`, `roadmap presence <proj>` |
| Agent | `agent-context [--mode M] [--focus node] <proj>`, `checkpoint [--node] <proj> <summary>`, `propose --kind K [--item Title[@Parent]]… <proj> <summary>`, `idea proposal-decide --accept\|--reject [--apply] <update-id>` |
| Admin | `invite [--ttl hours] <email>`, `invite revoke <id>`, `invites`, `signup-mode [invitation_only\|open\|closed]` (admin token required) |
| Server operator | `migrate`, `seed`, `promote-admin <email>` (creates the account, password on stdin, when absent); these need the server config (`DATABASE_URL`, `INSIDEOUT_JWT_SECRET`), and `promote-admin` also the owner DSN (`DATABASE_OWNER_URL`) |
| Front door | `help` (also `-h`, `--help`) and `version` (also `--version`) answer before any server config, so they work with no database; an unknown verb exits 2 with a pointer to `help` |
| Guidance | `guide [--out insideout.yaml] <proj>`, `repo set <proj> <repo-url>`, `sync <proj>`, `snapshot [--note] <prd>` |

Flags precede positional ids (Go convention). Statuses are
`locked|pending|in_progress|done`; audiences are
`decision|management|delivery|validation`.

Every request has a deadline: 5 minutes for the two calls that wait on a model
(`POST /prds/{id}/build` and `POST /roadmap/{id}/expand`), 20 seconds for the
rest, and an MCP tool call is bound by the same numbers. Ctrl-C (or SIGTERM)
ends the request in flight and the command with status 130 and
`insideout: interrupted`; a command still running 400 ms later, or a second
signal, ends the process at once (rv4-QUAL-04, rv4-REL-06).

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
admin routes and every delete), so the limit holds for any client, not just
this binary ([design](../architecture/agent-token-scope.md)). The registry
follows the login: twelve tools are not offered while the process holds a JWT
login — `commit`, `proposal_decide`, `signup_mode`, `invite_create`,
`invite_list`, `invite_revoke`, `idea_delete`, `roadmap_delete` and the four
password tools (`auth_register`, `auth_login`, `auth_forgot_password`,
`auth_reset_password`, so a password never enters the conversation). Run
those from the CLI or the web. Before any sign-in the whole registry is
listed, so a process can still sign in; the first sign-in narrows it.

- A failed exchange is a tool error and never falls back to the full token.
  An `INSIDEOUT_TOKEN` that already carries a scope is sent as it is (the
  server decides); one that is not a JWT is passed through and the registry
  stays whole.
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
(`GET /auth/config`); admins mint per-email invitations (`invite` /
`invite_create`, revoke with `invite revoke` / `invite_revoke`) and
switch the mode (`signup-mode` / `signup_mode`, the web admin panel
on the profile page). The four MCP admin tools (`invite_create`,
`invite_list`, `invite_revoke`, `signup_mode`) are CLI and web only once
the MCP holds a JWT login (see "Agent scope"): the server refuses an agent
token on `/admin/*`. `invite` and `invite_create` print the whole
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
every later tool call rides that session. Once the process holds a JWT
login, the four password tools are no longer offered (see "Agent scope");
the code and magic-link tools stay. The code flow is the
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
   `GITHUB_REPO_ALREADY_LINKED`. When the GitHub App is configured, the
   repository must be public or covered by an installation of the App, else the
   answer is 400 `GITHUB_REPO_NOT_VISIBLE` (rv4-COMP-05).
2. **Scaffold the guide**: `insideout guide --out insideout.yaml
   <project-id>` — generated from your roadmap: every leaf node with
   editable matchers, branch nodes documented as comments.
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
stable, so committed matchers survive regeneration.

## What agents may and may not do

`agent-context` embeds the vocabulary contract: agents **checkpoint**
work and **propose** structure/scope/priority changes (optionally with
structured items a human can apply on acceptance); a human **decides**
— and only humans Commit versions. See
[architecture/product-subsystems.md](../architecture/product-subsystems.md).
