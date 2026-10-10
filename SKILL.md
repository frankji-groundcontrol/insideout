---
name: insideout
description: Use when driving InsideOut — the ideas → PRDs → roadmaps product with GitHub-tracked evidence — through its CLI (`insideout`), MCP server (`insideout-mcp`, 50 tools), or /api/v1; covers auth, the idea inbox, PRD authoring, roadmap CRUD, versions/Commits, repo binding and evidence, and the agent vocabulary (checkpoint / propose / humans decide).
---

# InsideOut

InsideOut walks an idea to shipped: an **idea inbox** → **PRD** (eight
sections) → **roadmap** (project tree with deadlines) → **versions**
(human Commits), with **GitHub activity attached as evidence** to roadmap
nodes. Two agent modes exist: the built-in web coach (conversations), and
**you** — an external coding agent driving the same product through this
CLI/MCP surface. This skill is the contract for the second mode.

## Setup

From a checkout of this repository (needs Go 1.25+, see
`server/go.mod`). Setting up a machine from scratch — installing the
toolchains, choosing hosted vs own server, the key store, and the
installation record — is its own skill:
[skills/insideout-setup/SKILL.md](skills/insideout-setup/SKILL.md).

```bash
./scripts/install.sh          # binaries into ~/.local/bin; hosted API, no env vars
insideout setup               # the front door: interactive remote/local, or flags —
                              #   --hosted | --local --postgres bundled|own [--no-start] | --check
insideout whoami              # authentication required until you sign in
```

`insideout setup` signs in (password / emailed code / magic link,
hidden input) and saves the token to a 0600 config file every client
surface reads (env vars still win); `--local` writes `.env`, migrates,
and can boot the server; `--check` re-runs the health probes.

A server on this machine is `./scripts/setup.sh --local` (bundled
Docker Postgres, or your own URL on stdin). Then set
`INSIDEOUT_API=http://127.0.0.1:8080/api/v1` or the CLI keeps calling
the hosted API. Full steps, including the questions an agent should
ask: [docs/usage/installation.md](docs/usage/installation.md).

To register the MCP server, emit the snippet and merge it. With no
token the `INSIDEOUT_TOKEN` field is omitted; sign in with
`insideout login` or the `auth_*` tools:

```bash
./scripts/mcp-config.sh
./scripts/mcp-config.sh -q
```

Manual fallback (no script): `cd server && go build -o /usr/local/bin/insideout
./cmd/insideout` and the same for `./cmd/insideout-mcp`.

Signup is invitation-only by default: an admin mints a per-email code
(`invite_create` / `insideout invite <email>`, revoke with
`invite_revoke` / `insideout invite revoke <id>`) and registration
consumes it (`register --invite`, `auth_register` with `invite_code`).
Join an existing workspace with `workspace_join` / `insideout join
<code>`.

## The working loop (in order)

1. **Bootstrap** (only if nothing exists): `workspaces create --title T`
   → `projects create <ws-id> --title T` → `roadmap_add` for the first
   nodes. Otherwise start from `workspaces` → `projects <ws-id>`.
2. **Triage your inbox**: `ideas <ws-id>` (your own ideas; they are private
   to their author) → `idea_update` to sharpen, `idea_delete` to drop (soft —
   the row is kept for audit but `ideas` no longer lists it), `idea_convert`
   to open a PRD from an idea.
3. **Author the PRD directly**: `prd <id>` to read, `prd update <id>
   --section goals=…` to edit. Section keys: background, users, goals,
   nonGoals, stories, requirements, constraints, risks. `snapshot`
   records a working revision; `prd_export` renders markdown (optionally
   per audience).
4. **Grow the roadmap**: `roadmap_add/update/move/delete`,
   `roadmap update --deadline` for time pressure, `progress <project>`
   for the time-first view. `expand` grows a node via the planner.
5. **Track the repo**: `repo_set <project> <repo-url>` → commit the
   generated `guide` (`insideout.yaml`) at the repo root → `sync` on
   demand → **`evidence <node-id>`** to read what actually attached.
   Evidence never auto-proves outcomes — you judge it.
6. **Report like an agent, decide like a human** (the vocabulary — see
   below).

## The agent vocabulary (non-negotiable)

- `checkpoint <project> "<what was done>"` — record agent work; never
  applies changes.
- `propose <project> --kind structure|scope|priority "<summary>"` —
  propose changes; a **human** accepts/rejects
  (`proposal_decide` is theirs, owner/admin only).
- **Commits are human-only**: never call `commit` unless the human
  explicitly asked for it in this conversation. `readiness <prd-id>`
  tells you what a Commit would carry.
- `agent_context <project>` is your compact briefing; re-read it when
  context runs thin.

## Surface parity

Every verb exists 1:1 as CLI command ↔ MCP tool ↔ API route (50 tools:
auth ×9, admin ×4, ideas/PRD/roadmap/evidence/workspace/project CRUD,
versions, agent vocabulary, GitHub tracking, planner). Prefer the
narrowest tool; reads return raw JSON — trust it over your assumptions.

## Guardrails

- No API deletes PRDs/workspaces/projects — that is a human, web-side
  decision; `idea_delete`/`roadmap_delete` are the agent-scale deletes.
- The web coach's conversations are not exposed here; when a user wants
  the coached experience, point them to the web app.
- Full command/tool reference:
  [docs/usage/cli-mcp-and-guidance.md](docs/usage/cli-mcp-and-guidance.md).
