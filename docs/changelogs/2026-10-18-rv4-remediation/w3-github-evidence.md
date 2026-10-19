# Wave 3, github evidence — review-4 remediation

Branch `rv4/w3-github-evidence`, `8c74218..6a40814` (13 commits: six red-test
commits, five fixes and two tests that pin guards a mutation check found
unguarded), then this fragment. Findings are cited as `rv4-<ID>` because review
IDs restart per review; `R1-08` and `R1-12` are rows of the
[open-findings record](../../issues/2026-10-18-review-4-open-findings.md). Row
3.10 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md);
the rules are in the
[execution contract](../../review/2026-10-18-multirole-4/41-remediation-contract.md).
Server only: seven source files, one migration
(`20261018180000_github_repo_binding.sql`), sixteen new test files and four
edited ones. No client change, no new route, no new environment variable, no
`go.mod` or `go.sum` change. Visible to a user or operator: three new error
codes, the evidence kind of a deployment, and where commit sync gets its
credential. It is a `dir` row, so the design spike comes first. One finding is
only partly closed: rv4-COMP-05 carries an owner decision.

## Design spike

The three findings are one problem seen from three sides. Commit sync reads
GitHub with the wrong credential (FEAT-04); the repo link that tells it what to
read is unverified and shared by URL text (COMP-05); one evidence kind is
recorded before its outcome is known (FEAT-07, independent). Once commit sync
can use the App's installation token, a repo link stops being a pointer at
public data and decides whose timeline receives a private repo's commit
subjects. Fixing FEAT-04 alone would turn COMP-05 from a nuisance into a leak,
so the credential and the binding are one design.

Credential: keeping the server-wide `GITHUB_TOKEN` is rejected (it is the
finding: one personal token for every project's repo, and unset, one address's
60 requests an hour); anonymous only is rejected (private repos stay broken).
Chosen: a per-installation token asked of GitHub for the repo itself. The
installation id comes from `GET /repos/{o}/{r}/installation` (App JWT) or from
an HMAC-verified delivery, never from the caller, so a token reads only what its
installation was granted.

Binding: a `UNIQUE` index on the canonical URL is rejected, because hosted rows
may already share a repo and the migration would fail on them; the rule is
enforced in `SetProjectRepo` under a transaction-scoped advisory lock on the
lower-cased URL, with a non-unique lookup index. Proof that the binder controls
the repo (GitHub identity, a file in the repo, administrator approval) is not
guessed: InsideOut has no GitHub user identity and which proof to require is
policy (owner decision below). Chosen: to bind, the caller may link the project,
GitHub must show the repo to the server (App installed, or public), and no other
project holds it; to read, only the holder gets the installation token. "Holder"
is one definition used three times (bind check, webhook delivery, token gate):
all ask `insideout._projects_by_repo`, which returns at most one project, the
earliest-created one that has an owner, so they cannot disagree.

## What changed

### rv4-FEAT-04, commit sync uses the repo's installation token (red `91af972`, fix `d402a96`)

- Before: `fetchCommitPage` took its bearer token from `GITHUB_TOKEN` and
  nothing else; the App, which already minted tokens to read `insideout.yaml`,
  played no part. Unset, every sync was anonymous: a private repo answered 404
  (`GITHUB_NOT_FOUND`), a push to one synced 0 commits. Set, one personal token
  went to every project's repo.
- Now: `FetchCommitsSince(ctx, token, repoURL, ...)` takes the credential as an
  argument; the library reads no environment variable. `commitToken` supplies
  it: no App gives none (anonymous); a webhook delivery passes its authenticated
  `installation.id`; the manual route asks GitHub (`RepoToken`), and a 404 means
  "not installed", which reads a public repo anonymously. The repo is always the
  one stored on the project, never a name from the request.
- The token goes only to GitHub: a `rel="next"` link is followed only while it
  stays on the API origin the walk started on (`onAPIHost`). Before, a bad
  `Link` header was fetched with the bearer token attached (red: one request to
  `evil.example` carrying it).
- Unit tests of the new library functions ship with the fix, not the red commit:
  they call functions that do not exist there, and a compile error is not a red
  test. Two existing test files changed only by the added `token` argument (in
  one the token moved from a `GITHUB_TOKEN` `Setenv` to that argument).

### rv4-FEAT-07, release only after success (red `cb6130a`, fix `4c133bf`)

- Before: the `deployment` event was recorded as `release`. GitHub fires it when
  a deployment is created, before anyone knows whether it worked, so a failed
  deployment was already a release and the `deployment_status` saying "success"
  appended nothing. PRODUCT.md promises only that "a deployment may show
  release".
- Now (`planDeployment`): creation is `activity` "deploy started: <environment>
  @ <sha7>"; `deployment_status` `success` is `release` "deploy succeeded: ...";
  `failure` and `error` stay `activity` "deploy failed: ..."; `queued`,
  `pending`, `in_progress` and `inactive` append nothing. The `(node, kind,
  detail)` idempotency key is untouched, so redeliveries stay free.
- Two old unit tests are deleted, not weakened: one asserted that `success`
  appends nothing, the other that creation is `release`, the behaviour the
  finding calls a bug. `github_webhook_release_test.go` replaces them with the
  opposite expectations and the lifecycle end to end; the failure test is kept.
- Rows recorded before the fix stay. A deployment in flight at upgrade time can
  show its old creation-time `release` row and, once it succeeds, the new
  `deploy succeeded` row.

### rv4-COMP-05, one repo, one project, verified (red `29482b2` and `cf13868`, fix `3b45cfd`)

- Before: any member could store any URL, and the webhook delivered to every
  project holding the same text, so two owners who had typed one repo each
  received the other's commit lines (red: "delivery reached 2 projects, want
  1"; six concurrent links all succeeded).
- Bind (`handleSetProjectRepo`): the caller must be allowed to link this
  project; with the App configured, GitHub must show the repo to the server
  (`RepoVisible`: an installation covers it, or it is public), else 400
  `GITHUB_REPO_NOT_VISIBLE`, or 502 with nothing stored if GitHub cannot be
  asked; then `SetProjectRepo` takes the advisory lock, asks `_projects_by_repo`
  who holds the repo and answers 409 `GITHUB_REPO_ALREADY_LINKED` if another
  project does. The refusal names no project, owner or workspace. The holder may
  re-link in any spelling; clearing frees the repo. With no App configured
  nothing is checked: there is no credential a link could hand out.
- Authorisation comes before GitHub is asked, so a caller who may not link cannot
  spend the App's rate limit or tell "not visible" (400) from "not yours"
  (403/404). GitHub is asked before the transaction opens, so no connection or
  lock is held across a network call.
- Read: `commitToken` gives the installation token only to the project that
  `ProjectsByRepo` names as holder; another project naming the same repo (rows
  that predate the rule) reads anonymously. Its test ships with the fix: the
  installation-token path exists only after the FEAT-04 fix, so it has no red
  state at the base.
- `cf13868` is a second red commit: the first spelled an scp-style remote as one
  literal and `scripts/check-redaction.sh` rejected it, so it is built from
  pieces. One existing test (`github_repo_canonical_test.go`) now links a unique
  repo name per run, because a repo binds to one project database-wide.
- `eb6ce39` and `6a40814` pin two guards the mutation checks below found
  unguarded: the lock (a store test holds a link open and requires the second to
  wait, then be refused) and authorise-before-GitHub (counts the requests that
  reach GitHub for a caller who may not link: none).

### Migration `20261018180000_github_repo_binding.sql`

- `insideout._canonical_repo_url(text)` mirrors `ParseRepoURL` then
  `CanonicalRepoURL` and returns NULL for what the Go side refuses;
  `TestCanonicalRepoURLInSQLMatchesGo` runs one corpus through both (the corpus
  grew in the fix commit once the function existed: Unicode white space, scheme
  and host case).
- A guarded `UPDATE` rewrites only rows whose canonical form is non-NULL and
  differs from the stored text (bare `owner/repo`, `.git`, a trailing slash, no
  scheme, surrounding Unicode white space). A refused value (foreign host,
  dot-only segment) stays as stored and answers 400 `GITHUB_REPO_URL_INVALID` at
  sync. Case is kept as typed.
- `_projects_by_repo` is replaced, same signature and grant: case-insensitive,
  requires an owner, at most one row ordered by `created_at, id`, and never
  matches the empty URL (the old one returned every unlinked project for `''`;
  red: 128 rows). A partial non-unique index on `lower(repo_url)` is added.
- Proven on a database seeded with nine legacy rows (loose spellings, a
  duplicate, an earlier ownerless project, a foreign host, `../..`, an empty
  link): the first run applied, the second said "nothing to do", and running the
  file again by hand as `insideout_owner` gave `UPDATE 0` with every row,
  `updated_at` included, unchanged.

### Carry-forwards from w1-github-repo (R1-08), and R1-12

- (a) the server-wide token: FEAT-04. (b) the case-insensitive lookup and
  canonical rows: COMP-05, one migration.
- (c) rv4-SEC-06, red `5c698c0`, fix `434af6f`: `FetchGuideFile` refuses
  dot-only and empty names before building a request (`PathEscape` leaves `.` and
  `..` alone and an empty segment collapses into `//`; the names come from a
  delivery's `full_name` and the App token rides on the request). A stored link
  `ParseRepoURL` refuses now answers 400 `GITHUB_REPO_URL_INVALID` at sync (it
  was 502), and nothing is asked of GitHub.
- R1-12, red `19bbd50`, fix `fff3d9c`: `firstLine` cut a subject at 120 bytes,
  which could split a multibyte character (PostgreSQL refuses such text:
  `invalid byte sequence for encoding "UTF8"`), and when the text had a newline
  it returned the first line at any length, so the cap held only for text
  without one. It now takes the first line, then cuts at 120 characters. The
  `conversations.go` half of R1-12 is row 3.1.

## How existing duplicate bindings are treated

Nothing is merged, deleted or unlinked. Rows keep their `repo_url`,
canonicalised if it was a loose spelling of a valid repo. For a repo that more
than one project names (compared without regard to case), the holder is the
earliest-created project that has an owner (`created_at`, then `id`; the link
time is not stored). Only the holder receives webhook deliveries and the
installation token. The others keep their link and existing commit lines, read
the repo anonymously on a manual sync (a public repo works, a private one answers
404), silently stop receiving pushes, may clear the link, and cannot re-link it
while it is held (409). To find them, as `insideout_owner` (RLS hides other
workspaces from the app role):

```sql
SELECT lower(repo_url) AS repo, count(*) AS projects
FROM insideout.projects WHERE repo_url <> '' GROUP BY 1 HAVING count(*) > 1;
```

## Operator notes

- The server no longer reads `GITHUB_TOKEN` for commit sync; only `insideout
  setup --check` does. A deployment that relied on a personal token for rate
  limits now reads public repos anonymously unless the GitHub App is configured.
- Private repos need the App installed on them. No new App permission is
  expected (the guide read already needs Contents read; I expect the commits list
  to need the same) but it was not tried against a real installation.
- With the App configured a link makes one App-JWT call and, for a repo the App
  is not installed on, one anonymous call. GitHub down refuses the link (502),
  rate limited is 429 `GITHUB_RATE_LIMITED`; nothing is stored either way.
- The canonicalising `UPDATE` bumps `updated_at` on the rows it rewrites and is
  one-way: the typed spelling is not kept.
- For `release` evidence the App must deliver `deployment_status` events; a
  `deployment` event alone now yields `activity` only.
- The web client maps errors by HTTP status only (`errors_l10n.dart`), so the
  Evidence card shows its generic bad-request and conflict text for the two new
  refusals; the sentence is in the JSON body for any API caller.

## Owner decision (DEFERRED_OWNER)

1. May two projects of different owners link the same repo at all? Default: no.
   A repo links to one project at a time, a second link is refused with 409
   `GITHUB_REPO_ALREADY_LINKED`, clearing a link frees the repo. Sharing needs a
   delivery model (which project gets which commits) that nothing decides today.
2. How should a binder prove they control a private repo the App can see?
   InsideOut has no GitHub user identity. Options: the repo's `insideout.yaml`
   names the project id (proof by write access to the repo); a GitHub OAuth
   identity; approval by an instance administrator. Default: the repo must be
   public or have the App installed, the first successful link wins, and only
   the holder is handed the installation token. Residual risk: the first binder
   who can name an installed private repo, including by its old name after a
   rename, holds it and sees its commit subjects in their timeline.

Decision 2 would change `requireRepoVisible`; decision 1 would touch `claimRepo`,
`_projects_by_repo` and `commitToken`.

## Exercised against what

GitHub's real API is reachable from this machine anonymously; no GitHub App is.

- **Live** (scratch script, not committed: the real server binary on a loopback
  port, a private PostgreSQL, the real api.github.com anonymously, no App
  configured so no App credential was sent): `octocat/Hello-World.git` linked and
  stored as `https://github.com/octocat/Hello-World`; the same repo from a second
  tenant in another case, 409, nothing stored; the holder re-links, 200; two
  tenants racing for a new repo, one 200 and one 409; a real sync added 3 commits
  and the next added 0; a bad signature 401 and a signed push 200 reaching one
  project; with a second project forced onto the same repo, a push still
  reached one project (commit lines: first 3, second 0) and the second could
  clear its link but not re-link it; a stored non-GitHub link, sync 400; no
  error lines in the server log. Checked with curl: an anonymous `GET
  /repos/{o}/{r}` is 200 for that repo and 404 for a name that does not exist,
  the two answers `RepoIsPublic` relies on.
- **Local stub only** (in-process fake GitHub as the HTTP transport, real
  PostgreSQL, real handlers, real RLS): everything credentialed by the App (JWT
  minting, installation lookup, installation tokens, private-repo commit reads,
  `RepoVisible` and `RepoIsPublic` themselves, the guide fetch), the refusals at
  link, rate-limit and 403/429 handling, a `Link` header pointing at another
  host, and GitHub outages.
- **Not exercised at all**: a real App installation, GitHub accepting the JWT,
  real installation tokens and their permission scope, rename and transfer
  redirects, real rate-limit bodies and headers, and FEAT-07 against a real repo
  with an `insideout.yaml` (no public repo with a guide was available and
  nothing may be pushed; the deployment events are fixtures).

## Observations, not fixed

- GitHub redirects the old name of a renamed or transferred repo (not
  exercised here). A link by the old name would pass the visibility check and
  store the old name; the webhook then arrives under the new `full_name`, which
  no longer matches, so that project stops receiving pushes, and the
  one-project rule, which keys on the name text, would let the new name be linked
  as well. Keying the binding on GitHub's numeric repo id would close it (a
  column and a migration beyond this row); it belongs with the owner decision.
- The new lookups treat any 403 as a rate limit, as `fetchCommitPage` already
  does. GitHub can also answer 403 for a refusal that is not a rate limit (I
  expect a suspended installation is one; not exercised), which would read as
  "rate limit hit" with no retry hint.
- `planGuideEvidence` and `planDeployment` cut a sha with `[:7]` on bytes; a sha
  is hex, so it never splits a character.

## Left for the integrator

INTEGRATOR TODO, none in files this cluster owns:

- Add this fragment to the wave-3 section of this folder's `index.md` with the
  table below, and update the shared indexes (`docs/changelogs/README.md`,
  `docs/changelogs/index/2026-10.md`, `docs/HANDOFF.md`, `docs/plans/README.md`,
  `docs/issues/README.md`).
- `docs/issues/2026-10-18-review-4-open-findings.md`: R1-08 (a), (b), (c) and the
  `firstLine` half of R1-12 are closed; rv4-SEC-06 is FIXED; rv4-COMP-05 stays
  open as PARTIAL with the owner decision above.
- `GITHUB_TOKEN` is no longer read by commit sync (only `insideout setup
  --check`): `docs/usage/environment.md` (lines 87, 112 and 155),
  `docs/usage/keys/github/{README.md,github-token.md,gh-app-id.md}`,
  `docs/usage/keys/README.md:40`, `docs/usage/key-testing-tutorial.md:116`,
  `docker-compose.yml:99-106`, `.env.example:130`, `scripts/env.sh:195-197`,
  `scripts/env-write.sh:266-270` and the skip message at
  `server/cmd/insideout/cli_setup_check.go:115-120`. Say instead: private repos
  are read with the App installation token, public ones anonymously.
- `docs/usage/cli-mcp-and-guidance.md` (the `repo set` step, about lines
  189-193): a repo links to one project (second link, 409); with the App
  configured it must be installed or public (400).
- `docs/architecture/product-subsystems.md`, "GitHub evidence loop":
  `_projects_by_repo` names at most one project, the App token also reads commits
  (holder only), a link verifies visibility, a `deployment` creation is activity
  and only a successful `deployment_status` is release. Optionally
  `docs/architecture/backend.md:31` and `docs/usage/keys/github/gh-webhook-secret.md`
  (the App must deliver `deployment_status`).
- Client follow-up: map `GITHUB_REPO_NOT_VISIBLE` and `GITHUB_REPO_ALREADY_LINKED`
  so the Evidence card shows the sentence.

| Finding | Row | Status | Red | Fix |
| --- | --- | --- | --- | --- |
| rv4-FEAT-04 (R1-08 a) | 3.10 | FIXED | `91af972` | `d402a96` |
| rv4-FEAT-07 | 3.10 | FIXED | `cb6130a` | `4c133bf` |
| rv4-COMP-05 (R1-08 b) | 3.10 | PARTIAL: fan-out, verified bind, one project per repo, holder-only token, lookup and migration FIXED; policy and proof of control DEFERRED_OWNER | `29482b2`, `cf13868` | `3b45cfd` |
| rv4-SEC-06 (R1-08 c) | 3.10 | FIXED; with a and b, R1-08 is closed | `5c698c0` | `434af6f` |
| R1-12 (`firstLine`) | 3.10 | FIXED | `19bbd50` | `fff3d9c` |

## Verification

Run against a private PostgreSQL 18 (the route and store tests run the real
handlers under row-level security, and a skip counts as red). Outputs are kept
under `/private/tmp/insideout-rv4/reports/w3-github-evidence-*`.

| Commit | Test | Result at the red commit |
| --- | --- | --- |
| `91af972` | `TestSyncUsesTheInstallationCredentialForAPrivateRepo`, `TestWebhookPushSyncsAPrivateRepoThroughItsInstallation` | fail: status 404, want 200; synced 0 commits, want 2 |
| `91af972` | `TestSyncWithoutAnAppIsAnonymousAndIgnoresTheServerWideToken`, `TestSyncNeverFollowsANextLinkOffTheGitHubHost` | fail: an anonymous sync carried a bearer token; 1 request to another host with the token |
| `cb6130a` | `TestDeploymentCreationIsActivityNotRelease`, `TestDeploymentStatusMapsToEvidenceByOutcome/success`, `TestDeploymentLifecycleRecordsReleaseOnlyAfterSuccess` | fail: kind "release", want activity; no evidence, want release; a release row before any outcome |
| `29482b2` | `TestASecondProjectCannotLinkARepoThatIsAlreadyLinked`, `TestConcurrentLinksOfOneRepoLetExactlyOneProjectWin` | fail: status 200, want 409; 6 succeeded, 0 refused |
| `29482b2` | `TestWebhookDeliversToOneProjectWhenRowsAlreadyShareARepo` | fails: reached 2 projects, want 1 |
| `29482b2` | `TestLinkingARepoRequiresGitHubToShowItToTheServer` | fails 3 of 6 (200 where 400, 400, 502 are wanted); the other 3 pass and guard against over-refusal |
| `29482b2` | `TestCanonicalRepoURLInSQLMatchesGo`, three `TestProjectsByRepo*` | fail: the function does not exist; found 0, named 2 (want 1), matched the empty URL |
| `5c698c0` | `TestFetchGuideFileRefusesDotOnlyAndEmptyNamesWithoutAsking`, `TestSyncOfAStoredButRefusedURLIsABadRequestNotAnOutage` | fail: 8 of 8 names accepted, 8 requests sent; 3 of 3 answer 502, want 400 |
| `19bbd50` | `TestFirstLineCapsAtWholeCharacters` | fails: 200 characters returned, and a character cut in two |

All of these pass at the fix commit and at HEAD. The tests of the new library
functions (`TestInstallationForRepo`, `TestRepoToken`, `TestRepoVisible`,
`TestRepoIsPublic*`, `TestFetchCommitsSince*`) ship with the fixes, as above.

Mutation checks, each restored from git:

| Mutation | Result |
| --- | --- |
| M1 hand the installation token to every project that names the repo | `TestOnlyTheProjectHoldingARepoSyncsItWithTheInstallationToken` fails: status 200, want 404 |
| M2 remove the advisory lock | the api concurrency test still passed, a weak guard; the store test of `eb6ce39` fails: returned while the other link was uncommitted |
| M3 ask GitHub before authorising the caller | no test failed, a missing guard; the test of `6a40814` fails: 1 request to GitHub for a link the caller may not make |
| M4 follow a `rel="next"` link to another host | the library and api tests fail: the other host received a request carrying the token |

Gates, at `6a40814` (this fragment is the only commit after it): `go build ./...`
and `go vet` over the three packages clean; `gofmt -l internal cmd` empty;
`go test ./internal/github/... ./internal/api/... ./internal/store/... -count=1`
298 top-level tests pass, none fail or skip; the same with `-race` on api and
store passes; `migrate` run twice says "nothing to do"; the largest changed Go
file is `github/github.go`, 284 lines; the `scripts/tests/check_*.py` guards,
`sh scripts/check-redaction.sh` and `sh scripts/check-docs-recorded.sh` exit 0.
No client file changed, so Flutter was not run. Nothing was pushed, and no CI run
is claimed.
