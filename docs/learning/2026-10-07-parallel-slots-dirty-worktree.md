# 2026-10-07 — Parallel slots on a dirty worktree can revert unrelated uncommitted lines

## What was learned

Running parallel workflow slots over a worktree that holds other
people's uncommitted changes invites silent reverts: during the
setup-TUI fan-out, some slot (or its review/fix round) restored
`server/cmd/insideout/cli.go`'s `defaultAPIBase` from
`apiclient.DefaultBase` — a line belonging to the *uncommitted*
hosted-default slice — back to HEAD's `http://127.0.0.1:8080/api/v1`
literal. A `git diff` at the join showed line 19 "unchanged", which
looked healthy precisely because reverting *removes* the diff. The
damage surfaced only as wrong runtime behavior (`setup --hosted`
printing localhost; `TestJoinCommand` dialing it).

## Evidence

[Setup TUI slice-1 changelog](../changelogs/2026-10-07-setup-tui-slice-1.md),
defect 2: the join read the line as `apiclient.DefaultBase` before
the slots ran; after the run, worktree and HEAD agreed on the
localhost literal; `git show HEAD:…` confirmed HEAD never had the
hosted line, so the "no diff" state was the revert, not the original.

## Scope

Any parallel-agent or multi-worker run in a repository whose
worktree carries uncommitted in-flight work — this repo's standing
state (HANDOFF "worktree to preserve").

## When to apply

1. Before fanning out: snapshot the pre-existing uncommitted diff of
   shared files (`git diff -- <file> > /tmp/preserve.diff`) so a
   revert is detectable and recoverable.
2. After the run: for every file a slot touched, compare against the
   snapshot, not just against HEAD — "matches HEAD" is exactly what
   a revert of uncommitted work looks like.
3. Put "do not revert lines you did not write" in every slot's
   contract; scope discipline stops at *not editing*, which does not
   cover restoring-from-HEAD.
