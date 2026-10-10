# BUG-022 — "Build MVP" drafted a roadmap and wrote nothing

## Symptom

On the PRD page, **Build MVP** looked like it worked: the spinner ran, the
app navigated to the project's roadmap, and the roadmap was empty. For a
project that already had a roadmap the replace prompt ("This project already
has a roadmap with N nodes…") could never appear, so a second build never
asked and never replaced anything either.

Found 2026-10-10 while closing the
[build-from-PRD record](2026-07-25-build-from-prd-runs-llm-before-conflict-guard.md),
and confirmed against a real server in a browser (below). The CLI
(`build` then `build-confirm`) and the MCP `build` / `build_confirm` tools
were never affected.

## Root cause

`POST /prds/{id}/build` has only drafted a tree since the draft/confirm split
of 2026-10-05 (`f8f16eb`): it returns `{projectId, tree}` and touches nothing.
The roadmap is written by `POST /prds/{id}/build/confirm`, which also owns the
non-empty guard and the 409 `replace_conflict` with `liveCount`.

The Flutter client was written against the earlier one-call contract. `_build`
in `prd_page.dart` called `buildFromPrd`, treated the draft as the result, and
navigated to `/projects/<projectId>/roadmap`. The client had no request
builder or verb for the confirm call at all, and the replace-conflict branch
sat on the draft call, where the server can never raise it. The page also sent
`expectedCount` on the draft, which the server never reads.

Why the tests stayed green: `prd_actions_test.dart` answered the draft
endpoint with a 409 on the first call and a success on the retry, which is the
old contract. The fake encoded the same misreading as the page, so the test
pinned the page's consistency, not the server's behavior (the same shape as
[BUG-015](2026-09-03-bug-015-signup-mode-wrong-method.md)).

## Fix

Three commits, test first:

- `94b2521` adds the missing client call, `postPrdBuildConfirm` /
  `confirmBuildFromPrd(tree, {expectedCount})`, with a request-shape test.
- `c216a5a` rewrites the page test around the real contract. The fake answers
  the draft shape and records the confirm. Four tests, all red against the old
  page: the tap sends status, build, then build/confirm with the tree; a 409 on
  confirm asks and then retries with `expectedCount`; declining sends no second
  confirm; a failing confirm shows the error SnackBar and stays on the PRD.
- `87f8194` makes the page draft, confirm, and navigate; prompts on
  `replace_conflict`; and moves `expectedCount` off the draft call onto
  confirm only.

## Status

**Fixed 2026-10-10.** Verified end to end in a real browser against a local
server, not only in widget tests: tapping Build MVP on a PRD wrote the nodes
and landed on the populated roadmap; a second tap showed the replace dialog
with the live count, and confirming replaced the tree. The pre-existing
singular wording was fixed on the same pass
([BUG-025](2026-10-10-bug-025-phone-layouts-and-copy-found-by-the-web-walk.md)).
See the [web walk changelog](../changelogs/2026-10-10-close-open-gaps/web-walk-and-phone-layouts.md)
and [client fidelity fixes](../changelogs/2026-10-10-close-open-gaps/client-fidelity-fixes.md).

Lesson: [a fake copies the page's misreading of the contract](../learning/2026-10-10-a-fake-copies-the-pages-misreading.md).
When a verb's behavior changes server-side, the client's fake is the first
thing to re-derive from the server handler, and a flow that writes needs one
live call before it is called done.
