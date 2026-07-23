# 2026-07-21 — Docs-recording guardrail

## Trigger

Any commit that changes source outside `docs/`, and any milestone commit
another agent could take over from.

## The guardrail

Installed from the clean-repo-org skill's installer on 2026-07-21. One shared
check script, three triggers:

- `scripts/check-docs-recorded.sh` — the single source of truth
  (`--staged` for commit time, `--worktree --json` for session time,
  `--checkpoint` for the gate).
- `config/git-hooks/pre-commit` — **warns (never blocks)** when source
  changed but nothing under `docs/changelogs/` is staged. Fires for every
  actor — Claude, Codex, humans — since all commit through git. Activated
  via `git config core.hooksPath config/git-hooks` (already set for this
  clone; re-run once per fresh clone).
- `config/git-hooks/commit-msg` — the **checkpoint gate**: a commit whose
  message is tagged `[checkpoint]` is blocked unless the handoff trio is
  staged — a `docs/changelogs/` entry, the active `docs/plans/` file, and
  [`docs/HANDOFF.md`](../HANDOFF.md). Bypass with `git commit --no-verify`.
- `.claude/settings.json` `Stop` hook — in-session nudge for Claude.

## Repeatable sequence

1. Make the change.
2. Before committing, add/refresh the dated `docs/changelogs/` record and
   any affected `docs/` surfaces (architecture, usage, issues, learning,
   practices) and their indexes.
3. For a handoff-worthy milestone: update `docs/HANDOFF.md` and the active
   plan, tag the commit `[checkpoint]`.

## Verification

`sh scripts/check-docs-recorded.sh --staged` with a source file staged and
no changelog entry prints the reminder and exits 0; with a changelog entry
staged it is silent. (Verified at install time.)

## Failure signals

- The pre-commit warning appearing repeatedly on the same topic — the
  change is going unrecorded.
- A `[checkpoint]` commit being blocked — the handoff trio is stale.

## Related

- Skill reference: the clean-repo-org docs-recording-guardrail reference
  (external to this repo).
- [Reorg plan that installed it](../plans/2026-07-21-clean-repo-org.md).
