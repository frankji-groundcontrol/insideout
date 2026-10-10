# 2026-10-04 — `client/lib/api/models.dart` over the ~350-line budget

> **Resolved 2026-10-10.** `client/lib/api/models.dart` is a 4-line barrel over four domain files:
> `models_session.dart` (91), `models_project.dart` (123), `models_roadmap.dart`
> (138), `models_prd.dart` (222). The file had grown to 577 lines before the split
> (`2bb07f8`); see [client file splits](../changelogs/2026-10-10-close-open-gaps/client-file-splits.md).
>
> The original record follows, unedited.

`models.dart` is 478 lines holding every API model as hand-written
`fromJson`/constructor pairs: session/user, workspace, project, idea,
PRD (revisions/commits/readiness), roadmap (nodes/progress), presence,
and conversation types. It crossed the budget through ordinary feature
growth (the 2026-10-04 Wave 3 board fields were the latest additions).

## Why not split now

The file is uniform (one model per class, no logic to speak of), so the
oversize is a paging problem, not a tangle — but every feature slice
touches it, and a split should land alone to keep those diffs readable.

## Target structure

Keep `models.dart` as the barrel the app imports (re-export), with the
classes grouped by domain:

- `models_session.dart` — `UserProfile`, `Workspace`, `WorkspaceMember`
- `models_project.dart` — `Project`, `ProjectUpdate`, `Idea`
- `models_prd.dart` — `Prd`, `PrdRevision`, `PrdCommit`, `ReadinessGap`,
  `AudienceReadiness`, `PrdReadiness`
- `models_roadmap.dart` — `RoadmapNode`, `ProgressItem`,
  `RoadmapProgress`, `PresenceEntry`, `Conversation`, `ChatMessage`

## Bounded fix prompt

"Split `client/lib/api/models.dart` (~478 lines) into the four domain
files above, with `models.dart` kept as a barrel that re-exports them so
no import sites change, per
[docs/issues/2026-10-04-models-dart-line-budget.md](2026-10-04-models-dart-line-budget.md).
Gate: `cd client && flutter analyze && flutter test` green with zero
import-path changes outside the barrel. Every file ends under ~350
lines. Record the split in a changelog entry and close this issue."
