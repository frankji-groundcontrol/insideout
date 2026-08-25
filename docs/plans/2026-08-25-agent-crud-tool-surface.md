# 2026-08-25 — Agent CRUD tool surface (complete the matrix for external agents)

Status: **Finished** — all items verified live; see the
[changelog](../changelogs/2026-08-25-agent-crud-tool-surface.md).

## Why

The product's agent story has two modes: (1) the built-in web agent (PRD
Coach conversations), and (2) the user's **own coding agent** driving
InsideOut via skills/MCP/CLI. An audit of the 39 current MCP tools
against the API found the external-agent loop broken in four places:
the idea inbox is write-only (`idea_create` with no list/update/delete),
PRD sections cannot be edited (PATCH route exists, no projection),
GitHub tracking can bind+sync but **cannot read the evidence collected**
(`GET /roadmap/{nid}/evidence` unprojected), and an agent cannot
bootstrap a workspace/project without the planner. Design agreed with
the user: complete the CRUD matrix wisely — full lifecycle on ideas and
PRD authoring, read the evidence, allow bootstrapping; keep destructive
deletes and the coach's conversation channel web-only.

## Scope (7 tools → 46 total, 1:1 across API ↔ CLI ↔ MCP)

- `idea_list` — workspace inbox with full content (no separate get)
- `idea_update` — title/content/status triage
- `idea_delete` — inbox hygiene
- `prd_update` — patch PRD sections; folds the status transition in as
  a field (one orthogonal tool over the two routes)
- `roadmap_evidence` — read a node's GitHub evidence (closes
  bind → sync → **read** → checkpoint)
- `workspace_create`, `project_create` — agent bootstrapping

Plus the **agent skill file** (`SKILL.md` at the repo root): teaches any
coding agent the workflow and vocabulary — auth → triage ideas → author
PRD (edit sections, snapshot) → human Commits → roadmap CRUD → bind
repo → sync → read evidence → checkpoint work / propose changes; humans
decide; only humans Commit.

## Deliberately excluded (the "wisely")

- DELETE for PRD/workspace/project — destructive; humans decide in web.
- Conversations/messages tools — the web coach's channel; external
  agents author directly, and SSE coach streams don't fit stdio MCP.
- Member management, project-updates CRUD — human surfaces;
  `checkpoint` already covers agent reporting.

## Flow (progressive-development)

Single main track, sequential (all steps touch shared files):
apiclient verbs (tests first) → CLI verbs → MCP tools → skill file →
suites green → live matrix on a local server (throwaway workspace/
project/idea/PRD, cleanup after) → docs closure → commit/push/deploy/
verify. Smallest end-to-end check first: one idea's full lifecycle
(create → list → update → delete) through the CLI against a live server
before building the rest.

## Checklist

- [x] Plan + board row + YAML/visible track
- [x] apiclient: 7 verbs, httptest first
- [x] CLI: 7 verbs (ideas, idea-update, idea-delete, prd-update,
      evidence, workspace-create, project-create)
- [x] MCP: 7 tools (+ fix the stale tool count in the usage doc)
- [x] SKILL.md agent skill
- [x] go suites green; live matrix end-to-end
- [x] Changelog + indexes; usage doc; board/HANDOFF closure
- [ ] Commit, push, deploy server, verify
