# Wave 4, github client — review-4 remediation

Branch `rv4/w4-rv-github-client`, `582a4bd..65d2764` (six commits: two red tests,
two fixes and two comment-only commits), then this fragment. Cluster
`w4-rv-github-client` of the whole-change review of
`git diff d53035a..rv4/integration`. Findings are cited as `rv4-<ID>` because
review IDs restart per review; `FR` is that review's prefix. The rules are in the
[execution contract](../../review/2026-10-18-multirole-4/41-remediation-contract.md);
the code these findings sit on is described in
[w3-github-evidence.md](w3-github-evidence.md).

Server only: two source files (`server/internal/github/github.go` and
`appauth.go`), two new test files and two existing test files edited by a helper
call. No route, no migration (slot `20261018220000` stays unused), no dependency,
no environment variable, no client change, no change to a response shape. Eight
rows: 3 `FIXED`, 3 `PARTIAL`, 2 `DEFERRED_OWNER`. The only owner decision is
F171 with F174; it is recorded, not coded. No address, host, secret or token is
spelled in this fragment.

## Dispositions

| Finding | Sev | Status | Red | Fix |
| --- | --- | --- | --- | --- |
| `rv4-FR-F170`, `rv4-FR-F186`, `rv4-FR-F130` | Low | FIXED (one root, one red test; the API sentence for a refusal is the next cluster's, see "Left for the integrator") | `278f47d2` | `31e2f4b3` |
| `rv4-FR-F177` | Med | PARTIAL: retry hint and the per-repo memory done; the per-user limit on the link route (the finding's loop over distinct names) is outside this package | `278f47d2` (hint), `87204543` (memory) | `31e2f4b3`, `48b5015c` |
| `rv4-FR-F185` | Low | PARTIAL: the memory done; the startup warning and the doc line are in files this cluster does not own | `87204543` | `48b5015c` |
| `rv4-FR-F183` | Low | PARTIAL, not taken: the Go half is blocked by the SQL twin and needs a migration; the coupling is now written on `ParseRepoURL` | none | `65d27646` (comment only) |
| `rv4-FR-F171`, `rv4-FR-F174` | Med, High | DEFERRED_OWNER: reproduced, nothing coded; `RepoVisible` accepts what it accepted | none | `98c78226` (comment only) |

## What changed

### A 403 is a rate limit only when GitHub says so (F170, F186, F130)

- **Before.** Three places asked GitHub something and mapped every 403 and 429 to
  `RateLimitError` (`fetchCommitPage`, `InstallationForRepo`, `RepoIsPublic`;
  `RepoVisible` and `RepoToken` call into the last two). `retryAfterSeconds` read
  `Retry-After` and otherwise said 60. So a repository GitHub has blocked, an App
  that was not granted Contents read ("Resource not accessible by integration")
  and a suspended installation all became "GitHub rate limit hit - try again
  shortly", 429 `GITHUB_RATE_LIMITED`, retry in 60 seconds, for ever. A spent
  primary quota, which can be an hour away, was also told 60.
- **Now.** One helper, `limitOrRefusal(resp)` in `appauth.go`, is what each of the
  three places returns for a 403 or 429. A 429 is a rate limit. A 403 is one only
  with `Retry-After` (a secondary limit) or `X-RateLimit-Remaining` of 0 (the
  quota is spent). Any other 403 is a new `*AccessRefusedError{Message}`: GitHub's
  own `message`, read from at most 4 KiB of the body, control characters turned to
  spaces, 200 runes at most, "GitHub refused the request" when the body says
  nothing. It is not `ErrRepoNotFound` or `ErrNotInstalled`, so a blocked repo
  cannot read as a missing one, and `RepoVisible` still returns an error, never
  `false`, for it.
- **The hint.** `retryAfterSeconds(h, now)` takes `Retry-After` when it is a
  positive whole number; else, for a spent quota, the seconds until
  `X-RateLimit-Reset` (epoch seconds), never less than 1; else 60. A 429 with quota
  left has no refill time worth waiting for, so it stays 60.
- The helper is in `appauth.go` because `github.go` is at 349 of the 350 lines.

### The "is it public?" answer is remembered (F177, F185)

- **Before.** With no server-wide token (rv4-FEAT-04) every link of a public or
  uninstalled repo made one anonymous `GET /repos/{o}/{r}`, and the whole
  deployment shares one address's 60 requests an hour. Linking the same repo
  three times asked three times.
- **Now.** `RepoIsPublic` keeps the answer per repo, keyed by the lower-cased
  `owner/repo` (GitHub reads names without regard to case): 5 minutes for yes,
  1 minute for no, so a repo just made public is not refused for long. Only an
  answer is kept. A limit, a refusal or an outage is returned and forgotten, so
  the next link sees the real state. The installation lookup is never kept, so an
  App installed a minute ago counts at once. The map holds at most 512 answers
  and is emptied when full. One mutex, no dependency, no goroutine.
- The existing `TestRepoIsPublic` and `TestRepoVisible` flip GitHub's answer for
  one repo name. `point()` now clears the memory on entry and in its cleanup, and
  the flipping sub-cases call the new `forgetPublicAnswers()` (in
  `installation_repo_test.go`). No existing helper changed its signature.

### Comments only (F171, F174, F183)

- `RepoVisible` (`appauth.go`) now says what it proves and what it does not: it
  takes no caller, so it cannot tell whether the person linking may read the
  repo. The text points at the owner decision below.
- `ParseRepoURL` (`github.go`) says that `www.github.com` and a login before the
  host are refused on purpose because `insideout._canonical_repo_url` refuses
  them too, so both are widened or neither.

## Owner decision (DEFERRED_OWNER): rv4-FR-F171 and rv4-FR-F174

**Question: who may bind a private repo that the GitHub App can see?** Today the
link check proves that this server can read the repo, not that the member linking
it may. It is the open decision 2 of
[w3-github-evidence.md](w3-github-evidence.md) ("Owner decision"), seen again by
the whole-change review and now reproduced.

**Reproduction.** A scratch test, deleted afterwards (not committed, nothing in
the tree); real PostgreSQL, real handlers, real RLS, an in-process fake GitHub as
the HTTP transport with one App installation that covers tenant X's private repo.
Y is another tenant with no relation to X. Output kept under the reports folder
named below.

| Step | Answer |
| --- | --- |
| Y links X's installed private repo | 200, stored |
| Y runs a sync | "added 2 commit line(s)"; Y's timeline holds both commit subjects |
| What the fake GitHub saw | `GET /repos/{o}/{r}/commits` carrying X's installation token |
| X links its own repo | 409 `GITHUB_REPO_ALREADY_LINKED` (first come wins; X is locked out) |
| Y links an installed private repo / an uninstalled private repo / a name GitHub never heard of | 200 / 400 `GITHUB_REPO_NOT_VISIBLE` / the same 400 |

So any member can name an installed private repo, bind it before its owner, and
read its commit subjects; and the 200, 400 and 409 answers tell any member which
private repos have the App installed and which are taken. A repo that is not
installed and a name that does not exist cannot be told apart, on purpose.

**Options** (the four of the cluster brief; [w3-github-evidence.md](w3-github-evidence.md)
lists the first, second and fourth):

1. A proof inside the repo that names the project id (a file such as
   `insideout.yaml`, or a topic), read with the installation token. Proves write
   access to the repo, needs no new identity system, asks the repo owner to commit.
2. A GitHub OAuth identity for each member, checked for push or admin on the repo.
   The strongest, and the heaviest: an OAuth app, token storage, a new sign-in path.
3. Tie each installation to the workspace that installed it (recorded through the
   App's setup flow) and accept only that workspace's repos. Clean for tenants,
   needs the install flow built; it does nothing for two members of one workspace.
4. Approval by an instance administrator. No code to speak of, an operator task per
   link.

No option is decision-free: each one refuses the link of a legitimate owner until
it is proven, which is the whole private-repo feature. What an operator can do
today: on an instance with more than one workspace, install the App only on
repositories whose commit subjects every member of the instance may read, or run
one instance per trust boundary. A single-workspace instance is not exposed.

**What changes with the answer.** `requireRepoVisible` and `claimRepo`
(`server/internal/api/github.go`) and, for option 3, the webhook path that learns
the installation. In this package only `RepoVisible` would gain a caller argument.

## Not taken, and why (rv4-FR-F183)

A Low, optional in the brief: "accept `www.github.com` ... if that is all it
takes". It is not all it takes. `TestCanonicalRepoURLInSQLMatchesGo`
(`server/internal/store/project_repo_unique_canonical_test.go`) runs one corpus
through `ParseRepoURL` and `insideout._canonical_repo_url` and demands the same
answer, and it pins `www.github.com/own/repo` as refused on both sides. Accepting
the host in Go alone breaks that test; accepting it in SQL needs a new migration
(the brief forbids one), and the same migration would rewrite the stored
`www.github.com` rows that the finding says now fail every sync and miss the
webhook. A login before the host (an `ssh://` clone URL) is the same coupling.
The existing pin `https://www.github.com/a/b` in `github_segments_test.go` is the
Go-side statement of the same decision. Nothing changed but the comment.

## Known ceilings, recorded on purpose

- **A loop over distinct names still spends the quota.** The memory bounds repeats
  of one repo, not 60 different names from one member, which is the central
  scenario of F177. The package has no caller to count; a per-user limit on
  `PUT /projects/{pid}/repo` belongs in the API layer (to-do below). A loop of
  more than 512 names also empties everyone's remembered answers, so the others
  must ask again (a `ponytail:` comment on the map says so).
- **A secondary limit that arrives as a 403 with neither header** reads as a
  refusal. GitHub's rate-limit guide says a secondary limit is a 403 or 429 with an
  error message that names it, `Retry-After` "if present", and otherwise "wait for
  at least one minute". The brief's rule is the headers, and matching GitHub's
  prose was refused as fragile. The user still reads GitHub's own sentence; only
  the retry hint is lost. If it shows up in logs, matching the message is the cheap
  extension.
- **Answers can be stale.** A repo that went private is "public" for up to 5
  minutes; the link stored then reads nothing (a sync answers 404), the same as a
  link made a minute before the flip. A repo made public is refused for up to 1.
- **The memory is per process.** Several replicas each keep one and each ask
  from their own address. Two first asks for one repo at once both reach GitHub.
- **Public-repo commit reads are still anonymous.** A sync of a repo the App is not
  installed on reads its commits (up to five pages) from the shared address; the
  memory does not touch that. Reading public repos with an installation token
  (any installation) or an optional, explicitly named public-read token is the
  finding's own suggestion and a policy choice (which credential reads what); not
  taken.
- **Not changed, outside the three places the brief names:** `Token()` (the
  installation-token POST) and `FetchGuideFile` return a plain error with the
  status for a 403 and never called it a rate limit.
- **Interim answer for a refusal.** Until the next cluster, `writeGithubError`
  (`server/internal/api/github.go`) sends `*AccessRefusedError` down its default
  branch: 502 `GITHUB_UPSTREAM` with the generic upstream sentence, and an error
  line in the server log that carries GitHub's words. The wrong advice (wait a
  minute) is gone; the real reason reaches the user with the mapping.

## Left for the integrator

INTEGRATOR TODO, none in files this cluster owns:

- `server/internal/api/github.go`: map `*github.AccessRefusedError` in
  `writeGithubError` so the user reads GitHub's reason and "check the App's
  permissions" instead of the generic 502. The brief assigns this to the next
  cluster, `w4-rv-github-binding`; its test can reuse the 403 body in
  `rv4_review_github_client_test.go`. Decide there whether a refusal at the link
  step is 400 `GITHUB_REPO_NOT_VISIBLE` or 502.
- `server/internal/api/github.go`: a per-user limit on `PUT /projects/{pid}/repo`
  (F177). The memory in this package does not bound a loop over distinct names.
- `server/internal/api/server.go` (near line 100, where `ghTokens` is built) or
  `server/cmd/insideout`: one startup warning when `GITHUB_TOKEN` is set but
  ignored, and one when no App is configured (F185). The skip text at
  `server/cmd/insideout/cli_setup_check.go:120` ("evidence sync runs
  unauthenticated") should say the same.
- `docs/usage/environment.md` (the `GITHUB_TOKEN` row, line 113) and
  `docs/usage/keys/github/github-token.md` (line 7): name the cost, 60 anonymous
  requests an hour for the whole deployment, and that a link's public check is
  remembered for 5 minutes (1 for a no) (F185).
- F183: a new migration that mirrors `www.github.com` (and a login before the host,
  if wanted) in `insideout._canonical_repo_url` and rewrites the stored rows; then
  move `"www.github.com/own/repo"` out of the refused part of the corpus in
  `server/internal/store/project_repo_unique_canonical_test.go`, flip the pin in
  `server/internal/github/github_segments_test.go`, and widen `ParseRepoURL` in the
  same change. The one place to widen in Go is the `github.com` comparison in
  `ParseRepoURL`.
- F171, F174: when the owner decides, one sentence in
  `docs/architecture/product-subsystems.md`, "GitHub evidence loop": on an instance
  with several workspaces, any member can read the commit subjects of any
  repository the App is installed on until a proof of control is required.
- Shared indexes (`docs/changelogs/README.md`, `docs/changelogs/index/2026-10.md`,
  the index of this folder, `docs/HANDOFF.md`, `docs/plans/README.md`,
  `docs/issues/README.md`): add this cluster. The folder index lists fragments
  under their wave and cluster; this one is wave 4, `w4-rv-github-client`.

## Evidence

Reds are replays from `git archive <sha>` into an empty directory, so they are the
tree at that commit and nothing else; outputs are kept under
`/private/tmp/insideout-rv4/reports/w4-rv-github-client-*`.

- **F170, F186, F130, red `278f47d2`.** `TestRvGhClient403And429AreToldApartAtEveryCallSite`
  runs 12 header shapes against 5 call sites (`RepoIsPublic`, `InstallationForRepo`,
  `RepoVisible` on its installation leg and on its public leg, `FetchCommitsSince`):
  60 subtests, 30 fail. Ten say "reported as a rate limit (retry in 60 s), want a
  refusal that keeps GitHub's words" (a bare 403 and a 403 with quota left, 5 sites
  each); twenty say "retry hint = 60 s, want 2990 to 3001 s" / "25 to 31 s" /
  "115 to 121 s" / "1 to 5 s" (5 sites each). The 30 that pass at the base pin the
  behaviour that must not change (a 429, a `Retry-After`, a 403 with
  `Retry-After`).
- **F177, F185, red `87204543`.** "GitHub was asked 3 times about one repo, want 1"
  (public and not public), "anonymous questions = 3, want 1" (the link route's
  own call), "one repo in three spellings: asked 3 times, want 1", "a second repo:
  asked 4 times in all, want 2". `TestRvGhClientAnErrorIsNotRemembered` passes at
  the base and stays as a pin: 403, 429, 500 and 502 are each asked again.
- **Real GitHub, once, anonymously, no App.** `GET /rate_limit` answers with
  `x-ratelimit-limit: 60`, `x-ratelimit-remaining`, `x-ratelimit-used`,
  `x-ratelimit-resource: core` and `x-ratelimit-reset` as epoch seconds, 54
  minutes ahead when asked. That is the shape the hint reads, and the 60 an hour
  is the shared ceiling of F177 and F185. A real 403 for a blocked repo, a
  secondary limit and a suspended installation were not obtained; those bodies and
  headers are the stub's.
- **Interplay, with the real database.** The 24 API tests that touch the changed
  paths (repo link and its visibility, one repo one project, holder and token,
  private-repo sync, webhook, deployment evidence, agent scope) pass with the
  memory in place, none skipped.

Mutation checks, each restored from git (the tree was clean after each):

| Mutation | Result |
| --- | --- |
| drop the spent-quota clause of `limitOrRefusal` | 20 subtests fail |
| drop the `Retry-After` clause | 10 fail |
| every 403 is a rate limit (the old behaviour) | 24 fail |
| ignore `X-RateLimit-Reset` in the hint | 24 fail |
| no one-second floor | 7 fail |
| keep control characters / no 200-rune cap | 1 fails each |
| never store an answer | 10 fail |
| keys not lower-cased | 1 fails |
| never empty a full map / never expire / yes kept as long as no | 1 / 1 / 2 fail |
| store an error as an answer | 40 fail |

## Verification

Gates at `65d27646` (this fragment is the only commit after it), private
PostgreSQL 18, a skip would count as red:

| Gate | Result |
| --- | --- |
| `go build ./...` and `go vet ./...` in `server` | clean |
| `gofmt -l internal cmd` | empty |
| `go test ./internal/github/... -race -count=1 -v` | 43 top-level tests (158 with subtests) pass, 0 fail, 0 skip |
| the whole server suite, once, `-p 4` | 28 packages ok, 0 failed, 3 without tests |
| changed `.go` files over 350 lines (`rv4/integration...HEAD`) | none; largest is `github.go`, 349 |

The whole-suite run was not verbose, so it does not list skips; the `-v` runs above
show none. With this fragment staged, the 14 `scripts/tests/check_*.py` guards,
`sh scripts/check-redaction.sh` and `sh scripts/check-docs-recorded.sh` exit 0 (the
link guard: 0 broken relative links over 617 tracked `.md` files). No client file
changed, so Flutter was not run. Nothing was pushed and no CI run is claimed.
