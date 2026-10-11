# BUG-020 — a workspace member can write the plan

## Symptom

A stakeholder joined a workspace as `member` and, without any further
grant, did both of these:

- `insideout roadmap add` created a node on someone else's project.
  The node landed with that member as creator. Confirmed by a following
  `roadmap list` (4 nodes, was 3).
- `insideout propose --kind scope` recorded an `agent_proposal` on the
  same project. The project list then showed it as the latest update.

The product's oversight role is a reader
([PRODUCT.md](../../PRODUCT.md), "Collaboration and authority": a
Viewer reads; a Reviewer comments and requests changes; only a
Contributor adds proposals). Workspace membership today is the only
gate on roadmap create/update/delete and on
`POST /projects/{id}/updates` — the handlers answer 403 only when the
caller is not a member at all. Proposal *decision* is correctly
limited to the project owner or a workspace admin.

Seen 2026-10-06 during the live user simulation, persona C (a reader)
against persona A's project. CLI and MCP both allow the writes. The
web project page also still offers Accept and Reject on a proposal
that has already been rejected.

## Status

Fixed 2026-10-06. `requirePlanWriter` allows the project owner or a
workspace admin, and roadmap create, update, move, delete, and
`agent_proposal` all use it. A member can still read and leave a note.
`TestRoadmap_MemberCannotWrite` pins the refusal. Live, Ravi's add and
propose both returned 403 naming the owner-or-admin rule, and Mira's
three nodes were unchanged.

The project page also stops offering Accept and Reject once a proposal
carries a decision. The list query joins `proposal_decisions`, and the
live project response shows the rejected proposal with
`decision: rejected`. The release bundle on :5174 predates that client
change.

*Superseded 2026-10-11: the app image was rebuilt and deployed with
`v0.1.0` (see the
[owner-deploy changelog](../changelogs/2026-10-11-owner-deploy-and-mail-truth.md));
the hosted client no longer predates that change.*
