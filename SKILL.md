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

From a checkout of this repository (needs Go 1.22+):

```bash
./scripts/install.sh          # builds + installs both binaries into ~/.local/bin
                              # (override: first arg or INSIDEOUT_INSTALL_DIR),
                              # and adds that dir to PATH via the shell rc
export INSIDEOUT_API=https://<your-domain>/api/v1   # your deployment's domain
export INSIDEOUT_TOKEN="$(insideout login you@example.com)"   # or --code / --magic
insideout whoami
```

To register the MCP server in your own (or the user's) agent config, emit
the standard snippet and merge it — the same JSON ports across MCP clients
(Claude Code `.mcp.json`, Cursor `~/.cursor/mcp.json`, other
`mcpServers` blocks):

```bash
./scripts/mcp-config.sh                    # prints hints + JSON (token from
                                           # INSIDEOUT_TOKEN, else a placeholder)
./scripts/mcp-config.sh -q --api "$INSIDEOUT_API"   # JSON only
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
2. **Triage the inbox**: `ideas <ws-id>` → `idea_update` to sharpen,
   `idea_delete` to drop (soft — it stays in `ideas` with status `dropped`; filter client-side), `idea_convert` to open a PRD from an idea.
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
