# 2026-07-21 — Tracked coding-tool scratch directories from the JuanLeMe era

Status: open (needs a user decision).

## Problem

Three directories from earlier coding-tool sessions are committed to git but
are not part of this repository's product, code, or documentation system:

- `.sisyphus/` — a session-planning tool's plans, drafts, notepads, and ~15
  PNG evidence screenshots of the old JuanLeMe UI (`boulder.json`,
  `plans/juanleme-*.md`, `evidence/task-10-*.png`). All JuanLeMe-era.
- `.trae/rules/project_rules.md` — Trae editor project rules. The directory
  is gitignored (`.gitignore` line ~143) but this one file was committed
  before the ignore rule was added, so it is still tracked.
- `review/class_01_review.md`, `review/class_02.md` — workshop class review
  notes from the JuanLeMe workshop era.

## Risk

- Confuses future agents: `.sisyphus/plans/` looks like a second plans
  system competing with `docs/plans/`; the JuanLeMe evidence screenshots
  imply UI states that no longer exist.
- Dead weight: the PNGs are the largest tracked binary content in the repo.

## Proposed resolution

`git rm -r --cached .sisyphus .trae review` (keep on disk, untrack), add
`.sisyphus/` and `review/` to `.gitignore` alongside the existing `.trae/`
entry — these are regenerable tool scratch, the gitignore-is-fine category,
not sensitive content requiring the untracked-manifest treatment. If any of
the class-review notes have durable value, move that content into
`docs/learning/` first.

## Why deferred

Removing tracked files is destructive from git's perspective and touches
directories this repo's user may still be using with those tools — a
user-level judgment call, deliberately not made unilaterally during the
2026-07-21 docs reorganization ([plan](../plans/2026-07-21-clean-repo-org.md)).

## Fix prompt for a future agent

> After the user confirms: untrack `.sisyphus/`, `.trae/`, and `review/`
> (`git rm -r --cached`), extend `.gitignore`, verify `git status` is clean
> and nothing else references those paths (`grep -rn "sisyphus\|\.trae/\|review/" docs/ README.md`),
> and record the removal in `docs/changelogs/`.

## Acceptance checks

- The three directories are untracked and ignored; working tree preserved.
- No doc links reference them.
- A dated changelog entry records the removal.
