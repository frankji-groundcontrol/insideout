# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Two equally weighted primary audiences (confirmed 2026-07-23):

- **Team / group lead** — continuously tracks projects other people are building
  or maintaining; needs at-a-glance visibility into the freshest progress,
  blockers, and notes, with stale projects flagged so nothing goes quiet
  unnoticed.
- **Workshop / cohort setting** — time-boxed collaborative groups (classes,
  hackathons, workshops) whose members capture ideas and are guided toward a
  solid PRD.

Two roles in every workspace (invite-code membership):

- **Workspace admin** — creates the workspace, generates the invite code,
  manages member roles, approves/rejects PRDs.
- **Workspace member** — joins a workspace, tracks projects, records and
  converts ideas, converses with the PRD coach, submits PRDs for review.

## Product Purpose

InsideOut walks a team's idea all the way to shipped software. It captures the
spark, coaches it into a solid PRD, branches that PRD into a parallel roadmap
(the product's key feature), and tracks real progress — including synced GitHub
commits — until the idea is shipped. Success looks like: a lead sees every
project's status at a glance without chasing people, and any member can carry a
one-line idea through PRD, roadmap, and build to done.

The journey (idea → PRD → roadmap → shipped):

1. **Record ideas** — a frictionless per-member idea inbox (title + free text);
   ideas move inbox → refining → converted / dropped.
2. **Refine into PRDs via agents** — converting an idea creates a structured PRD
   (8 fixed sections) plus a coaching conversation; the PRD Coach interviews
   across four stages (clarify → draft → critique → finalize), writing sections
   directly via tool calls, with a review lifecycle (submit → approve/reject →
   revise & resubmit) and versioned snapshots.
3. **Branch a roadmap (key feature, confirmed 2026-07-23)** — the PRD becomes a
   project with an AI-generated branched roadmap: a tree of goals broken into
   executable tasks across parallel tracks. Nodes carry a status lifecycle
   (locked → pending → in_progress → done) and the AI can expand a node into
   sub-tasks. (The retired item in `docs/INIT.md` is the old organizer-authored
   questionnaire flow, not this roadmap.)
4. **Track to shipped** — members log progress on a timeline and sync real
   GitHub commits onto it; the group-leader board surfaces the freshest updates
   across the whole workspace and flags stale projects.

## Positioning

The integrated whole path (confirmed 2026-07-23): idea capture + an AI PRD
coach + a branched AI roadmap + progress/GitHub tracking, unified in one
invite-code workspace. A neighboring tool (Notion, Linear, a PRD template, or
one-shot ChatGPT) can copy any single step but cannot truthfully claim the
connected journey from a one-line idea to shipped software.

## Operating Context

- Distributed and run as a **hosted product (SaaS)** — users sign up and the
  service is operated for them (confirmed 2026-07-23; overrides the repo's
  self-hosted framing).
- The current build is also runnable self-hosted: Go backend + PostgreSQL +
  Nuxt 4 SSR via docker-compose, configured through `.env`; without an
  Anthropic token the coach falls back to an offline template reply for local
  development.
- Used in collaborative team and workshop rhythms: board reviews for tracking,
  in-the-moment idea capture, and dedicated refinement sessions with the coach.

## Capabilities and Constraints

- PRD has 8 fixed sections; the coach conversation runs clarify → draft →
  critique → finalize; the coach writes sections via tool calls streamed over
  SSE.
- A PRD converts to a project with an AI-generated branched roadmap
  (a `RoadmapNode` tree: locked / pending / in_progress / done; AI `expand`
  breaks a goal into sub-tasks; nodes can be created, moved, and reordered).
- Projects sync real GitHub commits onto the progress timeline (`syncGithub`,
  a repo URL per project).
- Review lifecycle: submit → approve/reject → revise & resubmit; every snapshot
  is saved as a version.
- Export any PRD to Markdown, or print-to-PDF via the browser.
- Bilingual UI: English (en-US) and Simplified Chinese (zh-CN).
- Invite-code workspace model carried over from the prior workshop product
  (JuanLeMe).

## Brand Commitments

- **Name:** InsideOut.
- **Tagline:** "Track the work. Refine the ideas. Ship better PRDs." /
  「跟踪进展、打磨想法、写出更好的 PRD」.
- **Voice:** bilingual EN/中文 — warm, confident, concrete.
- **Visual identity (confirmed 2026-07-23; binding across the whole product):**
  国风留白 / "Ink & Seal" — a soft celadon (青) ground, sumi-ink text, and a
  single vermilion 印泥 (cinnabar-seal) accent; a Song/Mincho display serif
  (Noto Serif SC) over a self-hosted PuHuiTi sans; light (celadon) + dark
  (ink-night) themes. Applies across landing, dashboard, roadmap tree, coach,
  and board. Documented in `DESIGN.md`. (The uncommitted "Prisma cinematic"
  black+cream detour from 2026-07-22 was reverted 2026-07-23.)

## Evidence on Hand

- Product docs: `README.md`, `docs/INIT.md`, `docs/PLAN.md`,
  `docs/plans/2026-07-20-go-rewrite/`.
- The `prisma-*.png` files at repo root (`prisma-hero-1.png`, `prisma-full.png`,
  etc.) are the *design reference* for the "Prisma cinematic" look (a third-party
  artists' studio site), NOT InsideOut product screenshots — do not reuse them
  as product evidence.
- Real, screenshottable product surfaces for proof: the branched roadmap tree,
  the PRD coach conversation, the GitHub-synced project timeline, the group
  board.
- No testimonials, customer logos, benchmarks, or pricing exist on hand —
  future work must not fabricate them.

## Product Principles

1. **Visibility without chasing** — a lead knows project status at a glance;
   stale work surfaces itself.
2. **Zero-friction capture** — recording an idea costs almost nothing;
   refinement is opt-in and later.
3. **Guidance over blank pages** — the coach interviews and writes; the user
   never faces an empty PRD template.
4. **One workspace, whole path** — idea, PRD, roadmap, and shipped-tracking live
   together; no exporting to a separate tool to finish the job.
5. **Craft in both languages** — EN and 中文 are first-class; design and copy
   must hold up in each.
