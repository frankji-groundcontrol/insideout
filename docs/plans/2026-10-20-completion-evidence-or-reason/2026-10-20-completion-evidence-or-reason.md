# Completion needs evidence or a sentence (2026-10-20)

Goal: moving a roadmap node from `in_progress` to `done` requires linked
evidence or a one-sentence completion reason stored on the node. The
store enforces it for the API, CLI and MCP. The web status control asks
for the reason when the node has no evidence. Other transitions stay as
they are.

## Execution

Live coordination state: [track YAML](2026-10-20-completion-evidence-or-reason.track.yaml).

- [x] Red store test: `in_progress` to `done` with no evidence and no
      reason is refused.
- [x] Migration `completion_reason`, store check in a new file, smallest
      edit to the status update, API/CLI/MCP field, Flutter prompt.
- [x] Green: evidence still completes; a one-sentence reason still
      completes and is stored. Related tests updated.
- [x] Record the decision on the issue, changelog, and architecture note.

## Acceptance

`TestRoadmapCompletion_RefusesDoneWithoutEvidenceOrReason` failed on the
red commit (`err = <nil>`) and passed after the rule, on a private
Postgres 18. The cluster was torn down.
