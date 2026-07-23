# 2026-07-22 — Idea → Reality: branched roadmap, MVP build, GitHub sync, UI prettify

Live plan. Drive this task from it as a checklist. English-only per repo rule.

## Goal (user directive)

The point is not the PRD — it is putting an idea into reality. The PRD is one
step. After it, help users: (1) build their MVP, (2) record progress / sync
GitHub, (3) plan with a **branched-tree** roadmap (parallel, hierarchical).
Also: the current UI/UX is ugly — prettify it.

## Product spine

```
idea → PRD (coach) → PROJECT with a branched ROADMAP TREE → progress (manual + GitHub) → MVP → reality
```

A project is the "make it real" container. Its roadmap is a tree: a root (the
product/MVP) whose children are parallel workstreams, each branchable further.
Progress (manual updates + GitHub commits) feeds the project timeline.

## Work breakdown

- [x] **M1 — schema**: `insideout.roadmap_nodes` (tree: `parent_id` self-FK,
      `status`, `position` among siblings) + `projects.repo_url` + RLS policy
      (mirror 20260720150000 pattern, member-of-workspace read/write).
- [x] **M2 — store+API**: `roadmap.go` — create / list-tree / update /
      move (reparent w/ cycle guard) / delete (subtree CASCADE). Routes under
      `/projects/{pid}/roadmap` and `/roadmap/{nid}`. Integration test vs real DB.
- [x] **M3 — frontend roadmap**: `roadmapService` + `RoadmapNode` type +
      recursive `RoadmapTree`/`RoadmapNode` components (branched-tree visual,
      status chips, add-child/edit/delete, Ink & Seal). Wire into project page.
- [x] **M4 — GitHub sync**: `projects.repo_url` + `POST /projects/{pid}/sync-github`
      pulling recent public commits into `project_updates` (dedupe via meta).
      UI: set repo, sync button.
- [x] **M5 — MVP build**: `POST /prds/{id}/build` — create/link project +
      AI-generate a starter roadmap tree from PRD sections (real Anthropic call;
      deterministic template fallback when token unset). "Start building" UI on PRD.
- [x] **M6 — AI expand node**: `POST /roadmap/{nid}/expand` — break a node into
      subtasks via the coach's Anthropic client (template fallback).
- [x] **M7 — prettify**: landing hero, dashboard, project page (Roadmap/Progress),
      roadmap tree visual. Light + dark browser-verified.
- [x] **M8 — docs+i18n**: EN/CN keys, changelog, TODO, HANDOFF, this plan checked off.

## Follow-on (same session): Prisma re-theme

After the core landed, the UI was re-themed wholesale to the user's Prisma
reference (dark cinematic, warm cream, Almarai + Instrument Serif, noise,
motion-v): token layer re-valued (keys unchanged), dark made default, landing
rebuilt with motion reveals. Recorded in
[`docs/design-system/CHANGELOG.md`](../design-system/CHANGELOG.md) `0.3.0` and
the [changelog](../changelogs/2026-07-22-idea-to-reality/index.md).

## Conventions (verified by reading the code)

- Store: `withUserContext(ctx, actorID, fn)` + `requireMember`, tables `insideout.x`,
  `ErrNotFound`/`ErrForbidden`. RLS: `FORCE`, keyed on `insideout.current_user_id()`.
- API: `registerXRoutes`, `pathUUID`, `UserID`, `httpx.*`, camelCase views, `timeLayout`.
- Frontend: `createApiXService(): IXService` over `apiFetch`, registered in
  `registry.ts`, types in `types/services.ts`, views in `types/index.ts`.
- Design: Ink & Seal semantic tokens only (no raw colors); `font-serif` headings;
  status `-bg`/`-fg` pairs; whole-literal Tailwind classes (no dynamic interpolation).

## Verification (real, no mocks)

Each layer verified before the next: `go build/vet/test`, migrate against real
`DATABASE_URL`, integration test (authz incl. deny paths), live HTTP via the
running server, frontend `typecheck`/`test`/`build`, browser screenshots light+dark.
