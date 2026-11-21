# 2026-11-21 — Wave 4b synced from MBP and pushed to GitHub

The owner directed a sync with the MBP clone (`MBP-local:~/Projects/insideout`)
and a push to `main`. This record covers that operation only; the code it
published is recorded in [wave 4b](./2026-10-18-rv4-remediation/wave-4b.md) and
the [2026-10-20 decision clusters](./2026-10-20-rv4-decisions.md).

## What was done

- On MBP2 (this checkout, clean tracked tree): fetched `origin` and MBP's
  `main` over SSH. MBP was 22 commits ahead of local `main` (`e6a3d140`),
  local had nothing extra — a clean fast-forward, no conflicts, no merge
  commit. `git merge --ff-only FETCH_HEAD` moved `main` to `4d186da9`.
- Confirmed GitHub's tip was still `e6a3d140` (an ancestor), then
  `git push origin main` (`e6a3d140..4d186da9`), per the
  [sync practice](../practices/2026-10-09-sync-two-macs-and-github.md).
  This supersedes the earlier "do not push" hold: the owner directed the
  push in this session.
- MBP's `origin/main` tracking ref was stale (`e6a3d140`, showing a false
  "ahead 22"). Refreshed it ref-only over SSH
  (`git push <ssh-url> 4d186da9:refs/remotes/origin/main`); MBP's checked-out
  `main` was already at `4d186da9` and now reports up to date.
- After the record commit (`c828b9ed`) an attempt to fast-forward MBP onto
  it aborted safely: MBP's uncommitted work also edits `docs/HANDOFF.md`,
  `docs/changelogs/README.md`, and `docs/changelogs/index/2026-10.md`.
  MBP's checkout and its in-progress slice were left untouched; its `main`
  stays at `4d186da9`, behind GitHub by the record commits.

## Verification

- Three-way commit and tree check: GitHub (`git ls-remote origin`), MBP2,
  and MBP all at commit `4d186da9`, tree `a32375a4`.
- On-disk tracked-file digest
  (`git ls-files -z | xargs -0 shasum -a 1 | shasum`): MBP2
  `cf609ef5…`, MBP `f073860e…`. They differ because MBP carries a whole
  uncommitted feature slice — the install front door, the password-reset
  watch, and the reset-code-on-the-page work (35 modified tracked files
  across `server/`, `client/`, `docs/`, plus new files and draft
  changelogs dated 2026-10-20 to 2026-10-22). Left untouched.
- Post-push privacy scan of every added line (`git diff e6a3d140..4d186da9
  -U0`): only the repository's own import paths, loopback/`.invalid` test
  fixture DSNs, and code identifiers. No home paths, real DSNs, tokens, or
  keys.

## Operator notes

- CI judges the wave-4b commits for the first time on this push. Observed
  at recording time: `security` (run 44786716778) is red — gitleaks passed,
  `govulncheck` flags two Go stdlib vulnerabilities
  ([issue record](../issues/2026-11-21-govulncheck-go-toolchain.md)); the
  `ci` run (44786716979) was still in progress. The last known `ci` state
  was red on `env-suites` only at `e6a3d14`; wave 4b carries the
  fetch-depth fix (`460c33cf`). Compare any red run against that history
  before blaming the newest slice.
- Deploy prerequisites are unchanged and remain the owner's: hosted
  `DATABASE_URL` as the app role, `INSIDEOUT_TRUSTED_PROXY_HOPS=1`, the
  migration `20261018200000` read grant, server before MCP binary, and the
  two new migrations `20261020140000` / `20261020150000` (see
  [retention and erasure](./2026-10-20-retention-and-erasure.md) and
  [completion evidence](./2026-10-20-completion-evidence-or-reason.md)).
- MBP holds the uncommitted install/reset slice described above. Whoever
  resumes there should commit and push it (its draft changelogs are
  untracked in that checkout); until then MBP's `main` stays at `4d186da9`
  and the three-way "one commit" check is knowingly open.