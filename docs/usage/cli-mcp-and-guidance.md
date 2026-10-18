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
insideout whoami                      # authentication required until sign-in
```

`./scripts/setup.sh` remains the pre-binary fallback (before
`install.sh` has run) and covers the same local-server ground.
Manual fallback (the CLI ships inside the server binary):

```bash
cd server && go build -o /usr/local/bin/insideout ./cmd/insideout
go build -o /usr/local/bin/insideout-mcp ./cmd/insideout-mcp
```

To register the MCP server with an agent, `scripts/mcp-config.sh` emits
the standard `mcpServers` JSON (`INSIDEOUT_TOKEN` included only when
that variable is already set; `-q` for JSON only) plus placement hints
for common clients — the same snippet ports across Claude Code, Cursor,
and anything speaking the standard shape.

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
| Auth | `register [--invite code] <email> [display-name]`, `login [--code\|--magic] <email>`, `forgot-password <email>`, `reset-password` |
| Authoring | `prd update [--title] [--section k=v]… [--status] <id>`, `evidence <node-id>`, `workspaces create --title [--desc]`, `join <code>`, `projects create --title [--desc] <ws-id>` |
| Read surface | `whoami`, `workspaces`, `projects <ws>`, `prd <id>`, `revisions <prd>`, `view [--audience] [--export F] <prd>` |
| Versions | `commit --name --audience [--summary --unresolved… --note] <prd>`, `versions <prd>`, `readiness <prd>` |
| Ideas | `ideas <ws>`, `idea create --title [--content] <ws>`, `idea update --title [--content] <id>`, `idea delete <id>` (soft drop), `idea convert <id>` |
| Roadmap | `roadmap list/add/update/move/delete`, `roadmap update --deadline RFC3339\|clear`, `roadmap progress <proj>`, `roadmap presence <proj>` |
| Agent | `agent-context [--mode M] [--focus node] <proj>`, `checkpoint [--node] <proj> <summary>`, `propose --kind K [--item Title[@Parent]]… <proj> <summary>`, `idea proposal-decide --accept\|--reject [--apply] <update-id>` |
| Admin | `invite [--ttl hours] <email>`, `invite revoke <id>`, `invites`, `signup-mode [invitation_only\|open\|closed]` (admin token required) |
| Server operator | `migrate`, `seed`, `promote-admin <email>` (creates the account, password on stdin, when absent); these need the server config (`DATABASE_URL`, `INSIDEOUT_JWT_SECRET`), and `promote-admin` also the owner DSN (`DATABASE_OWNER_URL`) |
| Front door | `help` (also `-h`, `--help`) and `version` (also `--version`) answer before any server config, so they work with no database; an unknown verb exits 2 with a pointer to `help` |
| Guidance | `guide [--out insideout.yaml] <proj>`, `repo set <proj> <repo-url>`, `sync <proj>`, `snapshot [--note] <prd>` |

Flags precede positional ids (Go convention). Statuses are
`locked|pending|in_progress|done`; audiences are
`decision|management|delivery|validation`.

## MCP server

`server/cmd/insideout-mcp` is a stdio MCP server exposing the same
verbs as tools (ask `tools/list` for the current set — a count written
here drifted twice), token via `INSIDEOUT_TOKEN`:

```json
{ "mcpServers": { "insideout": {
    "command": "/usr/local/bin/insideout-mcp",
    "env": { "INSIDEOUT_API": "https://<your-domain>/api/v1",
             "INSIDEOUT_TOKEN": "…" } } } }
```

(Equivalent to what `scripts/mcp-config.sh` prints, with the install
path resolved for you.)

Signup policy: registration is invitation-only by default
(`GET /auth/config`); admins mint per-email invitations (`invite` /
`invite_create`, revoke with `invite revoke` / `invite_revoke`) and
switch the mode (`signup-mode` / `signup_mode`, the web admin panel
on the profile page). `invite` and `invite_create` print the whole
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
every later tool call rides that session. The code flow is the
agent-friendly one: the human gets the email and supplies the digits, so
no password ever appears in a transcript. (Until 2026-08-24 there was
deliberately no login tool; the all-surfaces-agree product call reversed
that — a login tool's token is process state, not conversation state.)

## The guidance system (`insideout.yaml`)

Connecting a repository so pushes and PRs feed your roadmap:

1. **Bind the repo**: `insideout repo set <project-id>
   https://github.com/owner/repo` (web: the project's repo field). Accepted:
   `owner/repo` or a github.com URL (the host must be exactly `github.com`);
   the server stores `https://github.com/<owner>/<repo>`.
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
