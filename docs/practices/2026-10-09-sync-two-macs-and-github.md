# 2026-10-09 — Sync two Macs and GitHub onto one `main`

## Trigger

Work happens on two Macs and either may hold unpushed commits or
uncommitted slices (the board says "MBP sync still open"). The goal is one
commit and one tree on GitHub and both clones, with nothing private
published.

## Sequence

1. **Inspect all three before changing any.** On each Mac: `git status -sb`,
   `git branch -vv`, `git stash list`, `git worktree list`, then
   `git fetch` and `git rev-list --left-right --count main...origin/main`.
   Ask GitHub directly with `git ls-remote origin`; a clone's `origin/main`
   can be stale.
2. **Back up uncommitted work outside the repo**: `git diff HEAD --binary`
   plus a tar of the committable untracked files.
3. **Audit read-only first.** Privacy scan of everything that will be
   published: the uncommitted work **and** the added lines of every local
   commit GitHub lacks (`git diff origin/main..HEAD -U0`); a second reader
   re-checks each finding. Also a link and index check on new docs,
   `client/` analyze and tests, `server/` build/vet/test, and a proposed
   grouping into commits.
4. **Hold out, never ignore.** Tool scratch and raw agent journals
   (`.grok/`, `.qoder/`) and OS junk stay untracked. Record each file in the
   local manifest `.git/info/untracked-manifest.tsv` (path, size, mtime,
   md5, purpose).
5. **Commit one slice at a time from explicit path lists**
   (`git add --pathspec-from-file`), checking the staged set equals the plan
   and holds no held-out path. Never `git add .`. If the clone has not
   enabled the hooks, run commits with `-c core.hooksPath=config/git-hooks`.
6. **Push fast-forward only.** Re-read GitHub's tip, confirm it is an
   ancestor of `HEAD`, then a plain `git push origin main`. No force, no tags
   (tags fire the release workflow).
7. **Fast-forward the other Mac** on a clean tree: `git fetch origin`, then
   `git merge --ff-only origin/main`. If its route to GitHub is flaky, retry
   with a bounded timeout; fall back to `git push` over SSH into that clone.
8. **Record**: changelog, board row, HANDOFF; push the record commit the same
   way.

## Verify

For GitHub, MBP2, and MBP compare the commit, `git rev-parse HEAD:` (the
tree), and a digest of every tracked file as it sits on disk
(`git ls-files -z | xargs -0 shasum -a 1 | shasum`). Both clones must have no
tracked changes and `origin/main` equal to GitHub's tip. Then repeat the
privacy scan over what was actually pushed (`git diff <old-tip>..HEAD -U0`:
home paths, tokens, DSNs, signed URLs); it catches what the pre-push scan
did not cover.

## Failure signals

- `ssh` reports `Connection timed out during banner exchange` while the TCP
  port is open: the host is overloaded, not down. Raise `ConnectTimeout` and
  reuse one multiplexed connection (`ControlMaster auto`, `ControlPersist`).
- A clone's `origin/main` differs from `git ls-remote`: its fetch is stale,
  and its "ahead 0 / behind 0" proves nothing.
- A lockfile diff on one clone: see the
  [lesson](../learning/2026-10-09-lockfile-diff-in-sdk-pins-is-toolchain-drift.md).
- A clone that should be clean after the fast-forward is dirty: a tool wrote
  a tracked file. Diagnose before committing it.
- CI turns red after a push that carried older unpushed commits: they are
  being judged for the first time. Compare with the last green run before
  blaming the newest slice
  ([BUG-021](../issues/2026-10-09-bug-021-go-tests-share-one-config-file-on-linux.md)).
