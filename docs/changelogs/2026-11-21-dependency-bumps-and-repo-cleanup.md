# 2026-11-21 — Dependency bumps and repository cleanup

After the review 5 work merged, the owner asked to clean the repository's
branches and open pull requests.

## Dependencies

Five low-risk bumps were applied on top of the current `main` in #27. They
replace Dependabot PRs #6, #21, #19, #14 and #18, which were cut from an
older `main`.

| Module | From | To |
| --- | --- | --- |
| `golang.org/x/crypto` | 0.54.0 | 0.58.0 |
| `golang.org/x/term` | 0.46.0 | 0.48.0 |
| `github.com/jackc/pgx/v5` | 5.10.0 | 5.13.0 |
| `dio` (client) | 5.11.0 | 5.11.1 |
| `web` (client) | 1.1.1 | 1.1.2 |

The Go bumps also pull in `x/sys` 0.50.0 and `x/text` 0.43.0, which the new
versions require. Verified locally, because CI is disabled in the repository:

- every gate passes, including 667 Flutter tests and no skipped database
  tests;
- `govulncheck ./...` reports no vulnerability reachable from the code.

## Closed without merging

Each PR was closed with a comment giving the reason.

- **Four major-version bumps**, which need planned work:
  - `mark3labs/mcp-go` 0.32 to 1.5 (#20);
  - `go_router` 16 to 18 (#22);
  - `flutter_secure_storage` 9 to 11 (#13);
  - `cupertino_icons` 2.0 (#15).
- **Five GitHub Actions bumps** (#4, #5, #8, #10, #12), which only touch
  workflows that are disabled.

Dependabot stays on, at the owner's choice. Its new PRs are not tested
automatically; they are verified and merged locally when asked.

## Branches and local data

- **Remote:** the merged branches `rv5/remediation`,
  `rv5/proposals-in-subtree`, `rv5/close-gaps` and
  `cursor/vibrant-ink-seal-198b` are deleted, along with every Dependabot
  branch. The remote now holds only `main`. All 247 commit hashes cited in
  the review 5 changelogs are on `main`, because those PRs were merged with
  merge commits.
- **Local:** the 32 worker branches, the remediation stash and backups, and
  1.6 GB of scratch are removed. The private PostgreSQL 18 test cluster that
  development uses is kept.
