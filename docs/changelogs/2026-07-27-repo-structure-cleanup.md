# 2026-07-27 — Repository structure cleanup

Housekeeping that clears scattered scratch out of the working tree and the
legacy coding-tool directories out of git. Follows the
[2026-07-23 scratch-cleanup precedent](2026-07-23-woff2-fonts-and-scratch-cleanup.md)
and closes the
[tracked-tool-scratch-dirs issue](../issues/2026-07-21-tracked-tool-scratch-dirs.md).

## What changed

- **Deleted 12 unreferenced verification screenshots from the repo root**
  (`dark-landing-*.png`, `en-dark-*.png`, `t9-*.png`) — throwaway browser
  captures from the collaborative-canvas and landing verification sessions. A
  `git grep` confirmed no source or doc references any of them. Same category
  as the `prisma-*.png` removed on 2026-07-23.
- **Gitignored `.claude/`** (local agent/tool config carrying machine-specific
  absolute paths). It stays on disk and untracked — same treatment as the
  already-ignored `.trae/` and `.playwright-mcp/` — so it no longer shows as
  `git status` noise and cannot be committed by accident.
- **Untracked the JuanLeMe-era tool scratch and purged it from history**
  (user-approved): `git rm -r --cached .sisyphus .trae` then
  `git filter-repo --invert-paths --path .sisyphus --path .trae --path review`.
  This removes `.sisyphus/` (session plans/notepads plus 15 old-UI evidence
  PNGs, ~1.8 MB — the largest tracked binaries in the repo),
  `.trae/rules/project_rules.md`, and the long-gone `review/` workshop notes
  from *every* commit. The directories remain on disk (regenerable tool
  scratch), now gitignored. Because this rewrites history, every commit from
  the point those paths existed forward got a new SHA and `main` was
  force-pushed.

## Verification

- `git grep` before deletion: none of the 12 screenshots referenced by any
  source or doc; none of `.sisyphus` / `.trae` / `review/` referenced outside
  the two records that discuss removing them.
- A full-history backup bundle was taken before the rewrite
  (`git bundle create … --all`, then `git bundle verify`) so the pre-purge
  history is restorable.
- After the rewrite: `git status` clean; the three paths absent from
  `git log --all` and from `HEAD`'s tree yet still present-and-ignored on disk;
  `go build ./... && go vet ./...` still green.

## Operator notes

- **Anyone with a clone must re-clone or hard-reset** — the rewrite changed
  commit SHAs: `git fetch origin && git reset --hard origin/main` (after
  setting aside any local commits), or a fresh clone.
- The pre-purge objects linger on GitHub until its garbage collection runs; a
  fork or open PR would keep them reachable. This is a solo repo with no forks,
  so nothing else references the old history.
- The on-disk `.sisyphus/`, `.trae/`, and `.claude/` directories are kept
  intentionally (local tool state) and are all gitignored.
