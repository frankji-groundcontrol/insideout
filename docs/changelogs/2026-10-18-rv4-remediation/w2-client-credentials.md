# Wave 2, client credentials — review-4 remediation

Branch `rv4/w2-client-credentials`, `4b2a1db..3d33ea8` (22 commits: nine red
tests, nine fixes, one follow-up red/fix pair found by the live proof, and two
chores), then this fragment. Rows 2.1 and 2.2 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md);
findings `rv4-ARCH-01`, `rv4-ARCH-02`, `rv4-ARCH-05`, `rv4-SEC-05` (the client
half), `rv4-FULL-02`, `rv4-USER-03`, `rv4-USER-04`, `rv4-USER-05` and
`rv4-FULL-05`. Findings are cited as `rv4-<ID>` because review IDs restart per
review. No value, name or address is spelled in this fragment.

## What changed

- **One resolver decides base and token** (rv4-ARCH-01, red `fe6aac4`, fix
  `bf2160e`). `clientconfig.Resolve(flagAPI)` returns the API base and the
  bearer for exactly that base. The CLI verbs, every `--api` flag and
  `insideout-mcp` call it; the two `envOr` copies, `clientAPIBase`,
  `clientToken`, `sessionToken` and `applyAPIFlag` are deleted, and the `api`
  parameter is gone from every `run*Command`. Clients are built after flag
  parsing (`clientFromFlag`), not before, so the flag is no longer lost.
  Before: 37 verb forms ignored `--api` (some even refused the flag), 7 verbs
  (`build`, `expand`, `build-confirm`, `expand-confirm`, `repo`, `sync`,
  `guide`) went to the hosted default although the saved login named another
  server, and the MCP binary built its client with an empty base. The red
  commit message says "38 verb forms"; the count is 37 plus the 7 saved-base
  verbs. History is not rewritten, so this line is the correction.
- **A token belongs to the base it was saved for, and a bad config is loud**
  (rv4-ARCH-02, red `b73b79c`, fix `fd8ea13`). `Load` now returns exactly what
  the file holds; it used to overlay `INSIDEOUT_API` and `INSIDEOUT_TOKEN` per
  field, which paired one server's saved token with another server's base. 88
  verb forms (44 bearer-sending verbs, `--api` and `INSIDEOUT_API`) sent the
  token saved for server A to server B, and a config that could not be parsed
  was swallowed: each of the 6 verbs the test covers sent its request anyway
  (five exited 0), and `setup --check` passed. Now an unreadable config is an
  error, and the verb fails with it before any request (`apiclient.Failing`).
- **`DefaultBase` is retargetable at link time** (rv4-ARCH-05, red `a9bef27`,
  chore `1a1e955`, fix `e7f68b0`). It is a package-level `var` with a literal,
  so `-ldflags "-X ...apiclient.DefaultBase=..."` works (a `const` ignored it
  without a word). `scripts/default-api.sh` and `scripts/check-redaction.sh`
  read the declaration as a `const` or a `var`; that chore comes first so
  every gate stays green between the commits. One test parses
  `default_base.go` and fails unless `DefaultBase` is a package-level `var`
  with a string literal (the only form `-X` replaces); another checks that
  `scripts/default-api.sh` prints exactly the compiled value. The green file
  builds the CLI with `-X` and shows `insideout setup --hosted` printing the
  retargeted value; live proof 6 builds both binaries with `-X` and reaches a
  real server with no config, environment or flag.
- **The MCP process fixes its base once** (rv4-SEC-05, `PARTIAL`; red
  `543d2d5`, fix `5308d65`). `Build` resolves the base at start, so an
  `auth_login` session on server A is not carried to server B when another
  terminal re-points the saved login. The other half of SEC-05 (admin tools
  offered to everyone) is row 2.5 / AIDEV-04 in `admin_tools.go` and
  `product_tools.go`, which this cluster does not own.
- **Setup keeps the refresh token** (rv4-FULL-02, red `28f8904`, fix
  `4d2b03f`). `flow.Run` saved only the 15-minute access token for every
  sign-in method (password, emailed code, magic link); it now saves the
  refresh token with it (`Config.RefreshToken`, `Client.RefreshToken()`).
- **A saved session renews itself once on 401** (rv4-USER-04, red `1cfdaf1`,
  fix `a4eec44`; follow-up red `49ebe57`, fix `3d33ea8`). See the next section.
- **Plain `login` and `register` save the session** (rv4-USER-03, red
  `493f5cc`, fix `a628664`). They write the 0600 config (base, access token,
  refresh token), print its path on stderr and only the bare access token on
  stdout, so `insideout login you@example.com` then `insideout whoami` works as
  the docs say. The docs said it; the code printed a token and saved nothing.
- **`setup --check` asks as the signed-in user** (rv4-USER-05, red `6c2ac89`,
  fix `bb76711`). With a stored credential it calls `whoami` through the same
  client as every verb (renewal included) and reports "signed in"; a rejected
  login now fails the check (exit 1). It used to read `401` as "ok (as
  expected)" for a login the server rejects.
- **The web session survives a refresh that merely failed** (rv4-FULL-05, red
  `3679610`, fix `3f3ed61`). `client_core.dart` and `coachSseWithRefresh` clear
  the tokens only when the refresh error is a 401 or 403 (`refreshEndsSession`,
  the rule `SessionState.hydrate` already followed); a dropped connection or a
  5xx keeps the session for the next try.
- **`scripts/mcp-config.sh` is removed** (`b156cfc`, part of rv4-ARCH-01). The
  shadow resolver is gone and every live reference is fixed in the same
  cluster: the README, `SKILL.md`, the setup skill and its module reference,
  `docs/usage/{README,installation,cli-mcp-and-guidance}.md`, `scripts/install.sh`
  and the `setup` message now show the static registration,
  `{"mcpServers":{"insideout":{"command":"insideout-mcp"}}}`. The binary reads
  the saved login itself, so the snippet needs no `env` and no token. The plan's
  gate, `git grep -n 'mcp-config.sh'` outside the dated records, prints nothing.

## The credential contract

- **Base**: `--api`, else `INSIDEOUT_API`, else the saved login's `api_base`,
  else `apiclient.DefaultBase`.
- **Token**: `INSIDEOUT_TOKEN` when set (an explicit credential for whichever
  base is in effect, never renewed), else the saved token, but only when the
  effective base equals the base it was saved for (a config without `api_base`
  was saved for the hosted default). The refresh token travels with the saved
  token and only with it.
- **Errors**: a config file that exists and cannot be read is an error naming
  the file; a missing file, or no config directory at all, means nothing was
  saved.
- **One saved login per machine.** `login --api B` replaces the login saved for
  A; there are no profiles. A failed sign-in keeps the saved login. A save that
  fails prints a warning and still exits 0 with the token on stdout.
- **MCP**: the base is fixed at `Build`; a session started with an `auth_*`
  tool is a plain unarmed client for that base (it neither renews with, nor
  overwrites, the saved file).

## Renewal on 401

`apiclient/refresh.go` wraps the HTTP transport. A 401 is answered once when
the client carries the saved session (`SetRenewal`), the request has a bearer,
the path is not under `/auth/`, and the body can be replayed: renew, replay with
the new access token, return that answer.

1. Look at the store first: another command may already have renewed, and its
   pair is adopted without another call.
2. Otherwise `POST /auth/refresh` with the stored refresh token. On 200 the
   rotated pair is adopted and saved atomically (0600, temp file and rename).
3. On 401 or 403 the one-time token is gone. Wait up to `renewalGrace` (500 ms)
   for another process's rotated pair to reach the store, adopt it, else give up
   and say to log in again. Any other status leaves the original 401 standing.
4. One renewal per process at a time; one replay per request; never for a token
   from `INSIDEOUT_TOKEN`, another server, or an in-process login.

**The race the live proof found.** The server rotates the refresh token on
every use and refuses a replay, but does not revoke the family. When several
processes find the access token expired together, one wins; the winner's pair
reaches the file a few milliseconds after the server answered, and a loser that
looked in that gap told the user to log in again. Against a real server with a
3-second access TTL, 4 simultaneous `insideout whoami` in 10 trials: 21 of 40
succeeded and 19 failed (exactly one refresh accepted per trial). With the wait
in step 3: 40 of 40. The first red test (`49ebe57`) failed before the fix, and
the grace test (`TestWaitForTheWinnersPairEndsWhenNothingArrives`) is added in
the fix commit, because it names the new `renewalGrace` and would not compile
before it.

## Decisions and deviations

- **`setup --check`, unreachable stays a skip.** The register text asks for
  failure on 401; that is done. A server that cannot be reached is still
  reported as `skip`, not `fail`, because the check is also run offline right
  after install (the same rule the anonymous probe already had). Any other
  error, including a 5xx, fails.
- **No credential, anonymous probe.** With nothing stored `--check` keeps the
  old behaviour: reachable and "authentication required" is a pass.
- **`INSIDEOUT_TOKEN` keeps winning and is never renewed.** `scripts/install.sh`
  no longer teaches exporting it; it now says to run `insideout login`.
- **The `unauthorized` message** now says whether the client had a token at all
  ("not signed in") or the session expired or was revoked, and names the
  override: an `INSIDEOUT_TOKEN` in the environment hides the saved login.

## Evidence

Each red/green pair is a file under `/private/tmp/insideout-rv4/reports/`,
named `w2-client-credentials-red-<ID>.txt` and `-green-<ID>.txt` (the follow-up
is `-red-USER-04-race` and `-green-USER-04-race`). A red file is the test
failing on the commit before the fix, with the failing assertions; the tests
that are green on purpose (the other side of each rule) are named in its notes.
No skipped test counts as a pass: the DB-gated tests ran against a private
PostgreSQL cluster.

Live proof with the real binaries (`insideout`, `insideout-mcp`) against two
real servers on loopback that share one private database, in scrubbed
environments, `-live-<n>-<slug>.txt`:

| # | Proves |
| --- | --- |
| 1 `token-bound-to-base` | A's valid token is not sent to B with `--api` or `INSIDEOUT_API`; B logs a 401 with no user; `login --api B` moves the saved login |
| 2 `refresh-on-401` | expiry, 401, refresh 200, replay 200; the rotated pair is on disk and used next |
| 3 `login-persists` | 0600 file with base, token and refresh token; path on stderr; stdout is the bare token |
| 4 `setup-check` | valid ok; stale pair and bogus env token fail; closed port skips; no credential is the anonymous probe |
| 5 `corrupt-config` | corrupt, wrong-type and mode-000 config fail loudly for whoami, roadmap list, setup --check and login; the server saw nothing |
| 6 `mcp-binary` | 52 tools over stdio; follows the saved login; renews and saves in-process; `--api B` is unauthenticated; ldflags retarget works for both binaries |
| 7 `concurrent-renewal` | the 21/40 against 40/40 measurement above |

The second server logs the singleton-guard ERROR (two processes on one
database); that is expected for a two-server proof and was not a finding.

## Known ceilings

- The renewal wait is a poll, not a cross-process lock. It holds while the
  winner saves within 500 ms and the server only refuses a replayed token.
  If the server ever revokes a token family on reuse, or bursts still lose,
  serialize renewals with a lock file beside the config (marked `ponytail:` in
  the code). Each loser also sends one refused refresh, which counts against the
  server's per-IP auth throttle (30 a minute).
- A session an MCP `auth_*` tool opened lives one access TTL (15 minutes by
  default); renewing it needs the tools to hand the refresh token over
  (`auth_tools.go`, not this cluster's file).
- `roadmap.go` `Guide` builds its own request: it renews through the transport
  but a final 401 prints the plain status instead of the shared wording.
- Cosmetic: a request that cannot start (an unreadable config) is wrapped as
  `Get "/me": ...`; the message inside names the file and the fix.

## Out-of-ownership edits

`client/test/sse_test.dart` (one line: the stubbed refresh failure is now a 401
`ApiException`, because a bare `Exception` no longer ends the session),
`scripts/default-api.sh` and `scripts/check-redaction.sh` (the const-or-var
reader), and `server/internal/apiclient/escape_test.go` (one line: the names
`RefreshToken` and `SetRenewal` join the accessor map, because the wave-1
completeness test lists every exported `*Client` method).

## Integrator notes (wave 2)

- Shared indexes (`docs/changelogs/README.md`, the 2026-10 month index,
  `docs/HANDOFF.md`, the plan board, `docs/issues/README.md`) are the
  integrator's.
- `docs/architecture/backend.md` line 24 still says "env overrides the 0600
  config"; the rule is now the contract above.
- `docs/usage/environment.md` near line 88 and `scripts/setup.sh` line 53 (it
  still teaches `export INSIDEOUT_TOKEN="$(insideout login ...)"`) predate the
  saved session.
- `.github/workflows/release.yml` (lines 48 to 52) can pass
  `-X ...apiclient.DefaultBase=<repository variable>` to both builds, guarded
  against an empty variable.

## Verification

Gates on `3d33ea8` (the fragment is documentation only):

- `gofmt -l` prints nothing; `go build ./...`, `go vet ./...` and
  `go mod tidy -diff` are clean (no new package, no new dependency).
- `go test -race -count=1` on the touched packages: `apiclient`,
  `clientconfig`, `mcpserver` and `internal/setup/...` (290 `PASS` lines for
  tests and subtests, none skipped) and `cmd/insideout` (330 `PASS`, 1 skipped:
  the terminal-only `TestReadSecretTerminal`), with `DATABASE_URL` and
  `DATABASE_OWNER_URL` set to the private cluster.
- `go test ./... -count=1` ran once as the last Go gate against the private
  cluster: 27 packages ok and 3 without test files, 1357 `PASS`, 0 `FAIL`, the
  same one terminal-only skip.
- `flutter pub get --enforce-lockfile` (`pubspec.lock` untouched),
  `flutter analyze --no-fatal-infos` (no error or warning; the 14 infos sit in
  lines this cluster did not change), and `TZ=Asia/Shanghai flutter test`: 475
  tests pass.
- `scripts/tests/check_*.py`, `sh scripts/check-redaction.sh` and
  `sh scripts/check-docs-recorded.sh` pass. No touched file is over 330 lines.
