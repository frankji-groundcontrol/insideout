# Wave 1, github repo — review-4 remediation

Branch `rv4/w1-github-repo`, `838e6a5..d5159fa` (2 commits: one red test and one
fix), then this fragment. Findings are cited as `rv4-<ID>` because review IDs
restart per review. Row 1.14 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md);
the rules are in the
[execution contract](../../review/2026-10-18-multirole-4/41-remediation-contract.md).
Two server files changed (`github/github.go` and `api/github.go`) and two test
files were added. No migration, no client change, no new route and no change to
a response shape. The one visible difference is the value of `repoUrl` in the
`PUT /projects/{pid}/repo` response: it is now the canonical URL.

The row has two halves. The parsing and storage half is done. The token half
(one server-wide `GITHUB_TOKEN` for every project member's repo) is a policy
change that overlaps FEAT-04 and plan row 3.10, so this cluster only marks the
spot and reports the row `PARTIAL`.

## What changed

- **A repo segment made only of dots is refused** (rv4-SEC-06, red `028ad93`,
  fix `d5159fa`). `isRepoSegment` allows `[A-Za-z0-9._-]`, and `.`, `..` and
  `...` are built from those characters. So `../..` parsed as owner `..` and
  repo `..`, the commits URL became `/repos/../../commits`, and the request left
  with the server-wide `GITHUB_TOKEN` as its bearer token, aimed at a path the
  caller chose. A segment is now refused when nothing is left after trimming
  dots. A real name that starts with a dot (`owner/.github`) and one that holds
  dots (`a.b/_c-d`) are still accepted.
- **The host must be exactly github.com** (same commits). The old parser looked
  for `github.com/` anywhere in the input, so `https://evil.example/x/github.com/a/b`
  parsed as `a/b` and a link to a foreign host passed validation. A host is now
  whatever follows a scheme, or a first segment that holds a dot (a GitHub login
  never does). It must equal `github.com`, compared without regard to case;
  anything else is refused. A bare `owner/repo` has no host. Segments past
  owner/repo (a pasted `/tree/main`) are still ignored.
- **The canonical URL is what is stored** (same commits). `PUT .../repo` parsed
  the input to validate it and then stored the raw string, so the row that the
  sync and the webhook both read held whatever was typed. `CanonicalRepoURL` is
  one line over the existing `RepoURLFor` and yields
  `https://github.com/<owner>/<repo>`; `handleSetProjectRepo` stores and returns
  that. An empty value still clears the link.
- **The token half is marked, not changed** (see Left for the integrator).
  `fetchCommitPage` still reads `GITHUB_TOKEN`; a `ponytail:` comment there names
  the ceiling and the upgrade (per-installation tokens, plan row 3.10).

## Design notes

- **One parser, two callers.** `ParseRepoURL` has exactly two non-test callers:
  the PUT handler (validates at write) and `FetchCommitsSince` (parses again at
  read). Fixing the parser covers both, and it covers rows stored before the
  fix: a stored `../..` now stops inside `FetchCommitsSince` before a request is
  built. `TestFetchCommitsSinceSendsNothingForAHostileRepo` counts requests at a
  local `httptest` server and requires zero.
- **The case stays as typed.** The webhook lookup
  (`insideout._projects_by_repo`, reached through `Store.ProjectsByRepo`) is an
  exact match on `projects.repo_url` against `https://github.com/<full_name>`
  with GitHub's own casing. Lowercasing the canonical form would make
  `Owner/Repo` unfindable (mutation M6 below). The route test does not assert a
  string and hope: it runs that very lookup against the real database and
  requires exactly the one project back, with a fresh owner and repo name per
  run because the lookup is database-wide.
- **One spelling.** The canonical form reuses `RepoURLFor`, so the writer and
  the webhook cannot drift apart, and a unit test pins that they agree.
- **Scheme and host compare without case**, so `HTTPS://GITHUB.COM/Owner/Repo`
  reads as `Owner/Repo`; the owner and repo keep their case.

## Behaviour that narrowed

The rule is "the host is exactly github.com", and a few inputs that were
accepted by accident fall outside it. Nothing the UI placeholder
(`owner/repo or a github.com URL`) or the usage docs promise is affected.

- `www.github.com/...` and an `ssh://` URL with a user part are now refused;
  they were accepted only because the old parser searched for `github.com/`
  anywhere. A bare dotted owner (`a.b/c`) is now read as a foreign host; it was
  accepted as an owner GitHub cannot have, since logins hold no dots.
- `GitHub.com/Owner/Repo` now reads as `Owner/Repo`; it used to parse the host
  as the owner.
- A row stored before this fix in a rejected form fails at sync with the generic
  502 `GITHUB_UPSTREAM` ("GitHub sync failed"); the parse error is logged. The
  owner fixes it by linking the repo again. It is not mapped to a 400: the value
  can only come from the old code, and the one-line remedy is the same.
- The Evidence page seeds its repo field from `repoUrl` on load, so after a
  reload it shows the canonical URL instead of the typed text. Nothing in the
  client compares the two.

## Observations, not fixed

Found while reading; each is outside the owned files or beyond the row.

- `FetchGuideFile` (`appauth.go`, called from the webhook path with an
  installation token) builds its path from `url.PathEscape(owner)` and
  `url.PathEscape(repo)`, which leaves dots alone, and it takes owner and repo
  by cutting the payload's `full_name` at the first slash; it never calls
  `ParseRepoURL`. A dot-only name in a delivery would climb the request path.
  The destination stays the GitHub API and a delivery needs the webhook secret,
  so this is defence in depth. Candidate for `w3-github-evidence`, which owns
  that file.
- The webhook lookup is case-sensitive and exact, so a project whose stored
  `repo_url` is not the canonical spelling (any row written before this fix, or
  the same repo typed with a different case) never receives its pushes. Making
  the lookup case-insensitive and canonicalising existing rows is a migration;
  see below.

## Left for the integrator

- Add this fragment to the wave-1 section of this folder's `index.md` with the
  disposition below. SHAs are worker-branch SHAs; each integration commit's
  `cherry-pick -x` line names the original.
- `docs/issues/README.md` and the open-findings record: rv4-SEC-06 stays
  `PARTIAL`. The code half is FIXED here; the token half is deferred to
  `w3-github-evidence` (plan row 3.10, with FEAT-04).
- For `w3-github-evidence`: (a) replace the server-wide `GITHUB_TOKEN` in
  `fetchCommitPage` with a per-installation token, the way the webhook path
  already uses `s.ghTokens`; the `ponytail:` comment marks the spot; (b) make
  `_projects_by_repo` case-insensitive and canonicalise existing `repo_url` rows
  in the same migration (`https://github.com/<owner>/<repo>`, as typed); (c)
  refuse dot-only names in `FetchGuideFile`. (b) belongs with COMP-05, which owns
  `store/project_repo.go` and the lookup.
- Optional, no existing text becomes untrue: `docs/usage/cli-mcp-and-guidance.md`
  (the `insideout repo set` step) names no accepted forms. If it should: "Accepted:
  `owner/repo` or a github.com URL; the server stores
  `https://github.com/<owner>/<repo>`."
- Ordering: this cluster runs after `w1-clock-and-runes` and keeps its
  `projectResponse(..., s.now())` line in `handleSetProjectRepo`. No other
  wave-1 cluster owns these files.

| Finding | Row | Status | Red | Fix |
| --- | --- | --- | --- | --- |
| rv4-SEC-06 | 1.14 | PARTIAL: dot segments, host and canonical URL FIXED; token deferred to `w3-github-evidence` | `028ad93` | `d5159fa` |

## Verification

Each commit was replayed from a plain export of its tree against a private
PostgreSQL 18 (the route tests run the real handlers under row-level security,
and a skip counts as red):

| Commit | Test | Result |
| --- | --- | --- |
| `028ad93` | `TestParseRepoURLRejectsDotSegments` | fails, 10 of 10 inputs accepted |
| `028ad93` | `TestParseRepoURLRequiresTheGithubHost` | fails, 7 of 9 accepted (the userinfo and port forms were already refused) |
| `028ad93` | `TestParseRepoURLReadsEveryRealForm` | fails, 2 of 7 (mixed-case host read as the owner) |
| `028ad93` | `TestFetchCommitsSinceSendsNothingForAHostileRepo` | fails, 4 requests reach the server, each with the bearer token (`/repos/../../commits`, `/repos/./../commits`, `/repos/../user/commits`, `/repos/a/b/commits`) |
| `028ad93` | `TestSetProjectRepoStoresTheCanonicalURL` | fails, 4 of 5 subtests (only the already-canonical input passes); the stored text is the raw input and the webhook lookup finds 0 projects |
| `028ad93` | `TestSetProjectRepoRefusesHostileURLsAndStoresNothing` | fails, 6 of 6: status 200 and the hostile text stored |
| `028ad93` | `TestSetProjectRepoStillClears` | passes, and stays: it guards against over-rejection |
| `d5159fa` | all of the above, with `-race` | pass; the existing `TestParseRepoURLRejectsURLShapingCharacters` and `TestParseRepoURLAcceptsRealNames` still pass |

`TestCanonicalRepoURLIsWhatTheWebhookLookupMatches` ships with the fix, not with
the red commit: it calls `CanonicalRepoURL`, so at the red commit it would be a
compile error, and a compile error is not a red test. The route test compiles on
the base and fails on behavior.

The outputs are kept as `w1-github-repo-red-rv4-SEC-06.txt` and
`w1-github-repo-green-rv4-SEC-06.txt` under `/private/tmp/insideout-rv4/reports/`.

Mutation checks a verifier can run, each restored from git; every one turned the
named test red:

| Mutation | Fails |
| --- | --- |
| M1 drop the dot-only check in `isRepoSegment` | `TestParseRepoURLRejectsDotSegments` |
| M2 accept any host | `TestParseRepoURLRequiresTheGithubHost`, `TestFetchCommitsSinceSendsNothingForAHostileRepo` |
| M3 drop the rule for a dotted first segment | the host, dot, real-form and canonical tests, and the two existing parser tests |
| M4 compare the host with case | `TestParseRepoURLReadsEveryRealForm` |
| M5 the handler stores the raw input | `TestSetProjectRepoStoresTheCanonicalURL` |
| M6 lowercase the canonical URL | the canonical unit test and the route test |
| M7 build the canonical URL over http | the canonical unit test and the route test |

M1 is caught by the parser test only: the end-to-end test uses first segments
that hold a dot, which the host rule refuses on its own. The dot-only rule is
what stops `owner/..` and `github.com/owner/..`, and the parser test pins both.

Gates: `gofmt -l` is empty; `go build ./...` and `go vet ./...`;
`go test ./... -race -count=1` over the whole server, run once at `d5159fa`
(this fragment is the only commit after it): 26 packages ok, 3 without test
files, no failure, with `TestProjectQuietFlagged` green; a verbose run of
`internal/api`, `internal/github` and `internal/store` against the real database
shows 231 top-level tests passing and none skipped; every touched file is at
most 350 lines
(largest: `github/github.go`, 237); the six `scripts/tests/check_*.py` guards,
`sh scripts/check-redaction.sh` and `sh scripts/check-docs-recorded.sh` all exit
0. No client file changed, so Flutter was not run. Nothing was pushed, and no CI
run is claimed.
