# InsideOut Product Definition

> **Status**: since 2026-07-20, this project pivoted from 卷了么 (JuanLeMe), a
> workshop companion app, to InsideOut. The original brief remains in git
> history (and the JuanLeMe-era docs in [`docs/history/`](history/README.md));
> this document describes the current product. Full technical plan:
> [`docs/plans/2026-07-20-go-rewrite/`](plans/2026-07-20-go-rewrite/README.md).

## What InsideOut is

InsideOut helps users keep track of projects being developed or maintained by
others — e.g., a group leader managing their team's progress. It's built for
collaborative work or workshop environments: everyone can record their ideas,
get help turning them into a PRD, and be guided toward a better PRD through
agent-led interactive conversations.

## Three pillars

1. **Track others' projects** — a workspace (team, joined via invite code)
   contains projects, each with an owner, a status, and a timeline of
   progress/blocker/note updates. The group-leader view is a board surfacing
   the freshest updates and flagging stale ones.
2. **Record ideas** — every member has a frictionless idea inbox per
   workspace (title + free text, quick capture). Ideas move through a
   lifecycle: inbox → refining → converted (to a PRD) / dropped.
3. **Refine into PRDs via agents** — converting an idea creates a structured
   PRD (8 fixed sections) and a coaching conversation. The PRD Coach agent
   interviews the user across four stages (clarify → draft → critique →
   finalize), writing sections directly via tool calls, with a full review
   lifecycle (submit → approve/reject → revise & resubmit).

## Roles and collaboration model

The invite-code + workspace model proven in the workshop use case is kept
(carried over from JuanLeMe, since it's a natural fit for collaborative and
workshop settings):

- **Workspace admin** — creates the workspace, generates the invite code,
  manages member roles, approves/rejects PRDs.
- **Workspace member** — joins a workspace, tracks projects, records and
  converts ideas, converses with the PRD coach, submits PRDs for review.

Retired: the original roadmap/questionnaire-style guided flow
(organizer-authored tasks, participants completing them step by step) —
ideas and the PRD coach replace it as the new guided-content model.

## Technology decisions

- **Backend**: Go (was TypeScript/Supabase Edge Functions), talking directly
  to PostgreSQL (Supabase's platform dropped), a single `insideout_app` role
  (owning the whole database on a dedicated instance, or just the
  `insideout` schema on a shared instance), nothing ever created in the
  `public` schema.
- **Agent**: the PRD coach talks to the Anthropic Messages API through a
  direct client (`server/internal/agent/anthropic.go`), streamed over SSE,
  with tool calls writing PRD sections directly. (An earlier langchaingo
  implementation was dropped during the build — see decision D6 and
  [BUG-009](issues/2026-07-21-bug-009-langchaingo-removed.md).)
- **Frontend**: Nuxt 4 Universal SSR (kept from the JuanLeMe Nuxt
  migration), prettified with the 国风留白 / "Ink & Seal" visual language
  (celadon ground, sumi-ink text, a single vermilion seal accent).

The full decision record (D1–D10) with rationale is in
[`docs/plans/2026-07-20-go-rewrite/README.md`](plans/2026-07-20-go-rewrite/README.md)
§3 and §6; the current system shape is in
[`docs/architecture/`](architecture/index.md).
