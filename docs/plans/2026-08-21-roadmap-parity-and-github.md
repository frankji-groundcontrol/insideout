# 2026-08-21 — Roadmap parity (CLI/agent) and GitHub-driven progress

Status: **in flight**. The user called out the ignored gap: the roadmap
API is nearly complete — agent planning (`POST /prds/{pid}/build`,
`POST /roadmap/{nid}/expand`), node CRUD + move, repo binding + pull
sync — but only the web client can reach it, and GitHub progress is
poll-based with a server-level token instead of user-granted access.
Extends [cli-mcp-parity](2026-08-21-cli-mcp-parity.md).

## Stage A — roadmap parity on the CLI (this increment)

- [x] `apiclient`: build-from-PRD, expand-node, roadmap list/add/update/
      move/delete, repo set, github sync (delete discards JSON bodies —
      regression-locked by test)
- [x] CLI verbs: `insideout build <prd-id>`, `insideout expand <nid>`,
      `insideout roadmap list|add|update|move|delete`, `insideout repo
      set <project-id> <repo-url>`, `insideout sync <project-id>`
      (flags precede positional ids, Go convention)
- [x] Unit tests for the new client methods
- [x] Live verification against the hosted API (2026-08-21, scratch user
      `cli-roadmap-check@example.com`): idea → convert → **build**
      (agent produced a 16-node branched roadmap) → branch add (root +
      child) → update (rename + status `in_progress`) → move to root →
      delete → **expand** (agent grew a planner node) → repo set →
      **sync pulled 20 evidence updates** from the real insideout repo
- [x] Plan/board/changelog; checkpoint pushed

## Stage B — GitHub user access, webhook, matching guide

Design (implementation next; requires a user-registered GitHub App):

- **Matching guide**: each repo carries `insideout.yaml` at its root —
  a map from GitHub signals to roadmap nodes, e.g.
  `branches: {feature/*: <node-id>, ...}`, `labels: {roadmap/<slug>:
  <node-id>}`, `paths: {server/...: <node-id>}`. Commit/PR/deployment
  events advance `implementation activity`/`review`/`release` evidence
  on the matched leaf (PRODUCT.md: never above leaf level, never
  auto-proving outcomes; unmatched stays visible).
- **User access**: GitHub App (user registers it, grants repo scope);
  server exchanges the installation token per project and stores it
  encrypted. Replaces the server-level `GITHUB_TOKEN` for user repos.
- **Webhook**: `https://insideout.yalotein.net/api/v1/hooks/github` —
  HMAC-SHA256 signature check against a per-installation secret, events
  `push`, `pull_request` (opened/merged), `deployment_status`; handler
  loads the matching guide from the repo at the event's ref, resolves
  node ids, and appends evidence rows.
- [x] Webhook endpoint + signature verification + tests (2026-08-21:
      live via the domain; ping/bad-sig/push verified, 62 backlog
      commits synced, [changelog](../changelogs/2026-08-21-github-webhook.md))
- [ ] Install the GitHub App on the repo (user action) — real deliveries
- [x] Guide scaffolding for users on every surface (2026-08-21):
      `GET /projects/{id}/guide`, `insideout guide [--out F]`, MCP `guide`
      tool; format v1 node-keyed with leaf-only matchers
      ([changelog](../changelogs/2026-08-21-guide-and-mcp.md))
- [x] Matching-guide loader (2026-08-21): parse a committed
      `insideout.yaml`, match deliveries, append idempotent leaf
      evidence; `GET /roadmap/{nid}/evidence` exposes it
      ([changelog](../changelogs/2026-08-21-evidence-loop.md))
- [x] Installation tokens: App JWT + cached installation access tokens
      wired into guide fetching (live-proven so far only via the
      public-repo unauthenticated fallback; installing the app
      activates the token path)
- [ ] Install the GitHub App on the repo (user action) and confirm one
      real delivery in the app's delivery log

## Stage C — projection

- [ ] All Stage A/B verbs become MCP tools 1:1 (with the parity plan's
      frozen list extended)

## Sources

- User direction (roadmap ignored; agent+CLI creation, branch CRUD,
  GitHub-hook progress with a repo-side matching guide and user-granted
  access)
- PRODUCT.md "Git, CLI, MCP, and Agents" evidence rules
- Routes: `server/internal/api/roadmap.go`, `roadmap_ai.go`, `github.go`
