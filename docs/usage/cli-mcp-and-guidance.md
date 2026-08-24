# CLI, MCP, and the guidance system

How to drive InsideOut from the terminal, from any MCP-capable agent,
and how to connect a GitHub repository so its activity feeds your
roadmap. Everything here is a projection of the same `/api/v1`
contract the web client uses.

## Setup

The CLI ships inside the server binary:

```bash
cd server && go build -o /usr/local/bin/insideout ./cmd/insideout
export INSIDEOUT_API=https://insideout.yalotein.net/api/v1   # or your local server
export INSIDEOUT_TOKEN="$(insideout login you@example.com)"   # prints only the token
insideout whoami
```

`login` prompts for the password on stderr; the token is the only
stdout. Without `INSIDEOUT_TOKEN`, read verbs answer 401 with a hint.

Every way the web can sign in works here too (prompts on stderr, token
on stdout):

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
| Auth | `register <email> [display-name]`, `login [--code\|--magic] <email>`, `forgot-password <email>`, `reset-password` |
| Read surface | `whoami`, `workspaces`, `projects <ws>`, `prd <id>`, `revisions <prd>`, `view [--audience] [--export F] <prd>` |
| Versions | `commit --name --audience [--summary --unresolved… --note] <prd>`, `versions <prd>`, `readiness <prd>` |
| Ideas | `idea create --title [--content] <ws>`, `idea convert <id>` |
| Roadmap | `roadmap list/add/update/move/delete`, `roadmap update --deadline RFC3339\|clear`, `roadmap progress <proj>`, `roadmap presence <proj>` |
| Agent | `agent-context [--mode M] [--focus node] <proj>`, `checkpoint [--node] <proj> <summary>`, `propose --kind K [--item Title[@Parent]]… <proj> <summary>`, `idea proposal-decide --accept\|--reject [--apply] <update-id>` |
| Guidance | `guide [--out insideout.yaml] <proj>`, `repo set <proj> <repo-url>`, `sync <proj>`, `snapshot [--note] <prd>` |

Flags precede positional ids (Go convention). Statuses are
`locked|pending|in_progress|done`; audiences are
`decision|management|delivery|validation`.

## MCP server

`server/cmd/insideout-mcp` is a stdio MCP server exposing the same
verbs as tools (36 today), token via `INSIDEOUT_TOKEN`:

```json
{ "mcpServers": { "insideout": {
    "command": "/usr/local/bin/insideout-mcp",
    "env": { "INSIDEOUT_API": "https://insideout.yalotein.net/api/v1",
             "INSIDEOUT_TOKEN": "…" } } } }
```

Auth can also happen at runtime: the `auth_*` tools
(`auth_register`, `auth_login`, `auth_login_code_request` +
`auth_login_code_verify`, `auth_magic_link_request` +
`auth_magic_link_consume`, `auth_forgot_password`,
`auth_reset_password`, `auth_logout`) sign the server **process** in —
the token is kept in memory and never echoed into the conversation, and
every later tool call rides that session. The code flow is the
agent-friendly one: the human gets the email and supplies the digits, so
no password ever appears in a transcript. (Until 2026-08-24 there was
deliberately no login tool; the all-surfaces-agree product call reversed
that — a login tool's token is process state, not conversation state.)

## The guidance system (`insideout.yaml`)

Connecting a repository so pushes and PRs feed your roadmap:

1. **Bind the repo**: `insideout repo set <project-id>
   https://github.com/owner/repo` (web: the project's repo field).
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
`https://insideout.yalotein.net/api/v1/hooks/github`; the delivery is
HMAC-verified, the guide is fetched (installation token for private
repos), and matched **leaf** nodes gain evidence rows — commits show
activity, opened PRs review, merged PRs implementation, deployments
release. Unmatched activity stays visible but attaches to nothing;
evidence never auto-proves outcomes. Redeliveries are idempotent.
Re-generate the guide any time with `insideout guide`; node ids are
stable, so committed matchers survive regeneration.

## What agents may and may not do

`agent-context` embeds the vocabulary contract: agents **checkpoint**
work and **propose** structure/scope/priority changes (optionally with
structured items a human can apply on acceptance); a human **decides**
— and only humans Commit versions. See
[architecture/product-subsystems.md](../architecture/product-subsystems.md).
