# Summary — what changed, subsystem by subsystem

The pivot: **idea → PRD → project with a branched roadmap → progress (manual +
GitHub) → shipped MVP.** Three new capabilities, plus a full UI/UX re-theme to
the Prisma reference.

## 1. Branched-tree roadmap (the core)

- **Schema** (`server/db/migrations/20260722120000_roadmap_nodes.sql`):
  `insideout.roadmap_nodes` — a tree on a project: `parent_id` self-FK (NULL =
  root) forms the hierarchy, siblings share a parent and are ordered by
  `position`, `status` ∈ `locked/pending/in_progress/done`. RLS mirrors
  `project_updates` (workspace member read/write). Also `projects.repo_url`.
- **Store** (`server/internal/store/roadmap.go`, `roadmap_plan.go`): create /
  list-tree / update / move (reparent with a recursive **cycle guard**) /
  delete (subtree via `ON DELETE CASCADE`); `ReplaceRoadmapTree` (atomic
  regenerate), `GetRoadmapNode`, `EnsureProjectForPrd`.
- **API** (`server/internal/api/roadmap.go`): `GET/POST /projects/{pid}/roadmap`,
  `PATCH/DELETE /roadmap/{nid}`, `POST /roadmap/{nid}/move`.
- **Frontend**: `roadmapService` + `RoadmapNode`/`RoadmapTreeNode` types +
  recursive `roadmap/RoadmapNodeItem.vue` + `roadmap/RoadmapTree.vue`
  (trunk-and-stub connectors, cream status seals, progress bar, add-child /
  edit / delete / status-cycle). The project page gains **Roadmap / Progress
  tabs**.

## 2. Progress + GitHub sync

- `server/internal/github/github.go` — parse a repo URL, fetch recent commits
  from the public GitHub API (no auth needed; optional `GITHUB_TOKEN`).
- Store (`project_repo.go`): `SetProjectRepo`, `ProjectRepoSync`,
  `RecordRepoSyncSHA` — a per-project sync cursor in `projects.meta`
  (`github_last_sha`) so repeat syncs only add new commits.
- API (`api/github.go`): `PUT /projects/{pid}/repo`, `POST
  /projects/{pid}/sync-github` — commits land in the existing
  `project_updates` timeline (owner/admin).
- Frontend: `project/GithubSync.vue` in the Progress tab (link repo, sync,
  result message).

## 3. Build the MVP (PRD → reality, AI)

- `server/internal/agent/roadmap_planner.go` — a `RoadmapPlanner` that turns a
  PRD (or one node) into a branched tree via `StreamChatForcingTool` (the
  critic's schema-validated forced-tool trick, so output is real JSON — no
  fence-scraping). A **deterministic template** is the offline/fallback path,
  so the feature never hard-fails on LLM flakiness.
- API (`api/roadmap_ai.go`): `POST /prds/{pid}/build` (create/link a project +
  generate the starter roadmap) and `POST /roadmap/{nid}/expand` (break a node
  into subtasks). Wired through `api.NewServer` + `cmd/insideout/main.go`
  (real planner when `ANTHROPIC_AUTH_TOKEN` is set, template otherwise).
- Frontend: a vermilion **"Build the MVP"** button on the PRD page (→ navigates
  to the generated project) and a per-node **"break down with AI"** action in
  the roadmap tree.

## 4. UI/UX re-theme to the Prisma reference

- Whole app re-themed to **Prisma cinematic** (pure black + warm cream ink,
  dark default) via the token layer — see
  [`docs/design-system/CHANGELOG.md`](../../design-system/CHANGELOG.md) `0.3.0`.
- Fonts: **Almarai** (Latin) + **Instrument Serif** (italic accent), CJK
  fallback to Noto faces; noise-texture utilities; `motion-v` cinematic
  reveals on the landing; the landing hero/pillars rebuilt.
- The landing message re-framed to "idea → reality" (the PRD is one step) with
  a 4-pillar journey: capture → refine → roadmap → track-to-shipped.
