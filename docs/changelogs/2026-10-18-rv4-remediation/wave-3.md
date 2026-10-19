# Review-4 remediation, wave 3: coach trust, bounds, the auth network, supply chain and the client

The wave-3 record of the [folder index](index.md): the outcome, the clusters, the
gates, the lane logs, what the records pass closed, the carry-forward and the
operator notes. The per-finding table is
[wave-3-dispositions.md](wave-3-dispositions.md). Each cluster's own account
(what changed, the evidence, the ceilings) is its fragment, linked below.

## Wave 3

Thirteen clusters: the eleven plan rows 3.1 to 3.11 plus `w3-client-apply` and
`w3-users-read-grant`, which wave 2 added. The integrator cherry-picked 169
commits with `-x` onto the wave-2 records head `8c74218`, in the order
coach-trust, stream-bounds, auth-network, release-supply, privacy,
context-deadlines, test-hardening, first-run-copy, github-evidence,
client-assets, client-apply, users-read-grant, docs-truth (no conflicts, no
baseline failure). Two pre-gate commits follow, `4060d64` and `3a1c4ac`: 171
commits above `8c74218`, head `3a1c4ac`, 321 above `d53035a`. The records
follow-ups are `da93d78`; the per-ID state and this section are the commit
titled `docs(rv4): wave 3 dispositions`. SHAs are on `rv4/integration`; the
fragments cite the worker-branch SHAs, and each integration commit is
patch-identical to its worker commit (its body names the original in the
`cherry-pick -x` line).

The wave changes 1,168 files (33,677 insertions, 899 deletions). 872 of them are
the added font mirror under `client/web/fonts` (23 MB), so the other 296 files
hold 19,418 insertions and 899 deletions. It adds four migrations
(`20261018170000_sessions_family.sql`, `20261018180000_github_repo_binding.sql`,
`20261018200000_users_read_grants.sql`, `20261018210000_index_foreign_keys.sql`;
the slots `20261018150000`, `20261018160000` and `20261018190000` stay unused)
and no new dependency. The only module change is the pre-gate bump of
`golang.org/x/text` from v0.40.0 to v0.41.0 with `toolchain go1.26.9` in
`server/go.mod`; the `client/pubspec.yaml` change is a comment and
`pubspec.lock` is unchanged. Workflow files changed: `dependabot.yml` (new),
`ci.yml`, `release.yml`, `security.yml` (new), `smoke.yml` and
`ubuntu-install.yml`.

Outcome lists (64 rows): `FIXED` 48; `PARTIAL` 13; `DEFERRED_OWNER` 3
(rv4-DATA-06, rv4-TEST-07, rv4-CLOSE-09); `NOT_REAL` none; `BLOCKED` none.
`rv4-PM-04` and `rv4-PM-05` were reverted in the repair round of
`w3-release-supply` (`c201f92`, `12bc405`) and are not in the 64: they are not
fixed and move to the proposed wave-4 `w4-release-version`.
The wave closes these earlier residue rows: R1-06, R1-08 (through rv4-FEAT-04,
rv4-COMP-05 and rv4-SEC-06), R2-01, R2-02, R2-06 and R2-08, and half of R1-12
(the `firstLine` half). It leaves 50 new residue rows, R3-01 to R3-50, in the
[open-findings record](../../issues/2026-10-18-review-4-open-findings.md#wave-3-open-residue-integrator-2026-10-19),
and updates the wave-1 and wave-2 rows it touched. The owner decisions the wave
adds are in the last table of that record
([owner decisions](../../issues/2026-10-18-review-4-open-findings.md#owner-decisions-no-engineering-action-until-decided)).

Nothing here was exercised against a real third party: GitHub, a model provider
and a mail provider were driven by stubs or not at all (each fragment says which).
Every property of a workflow file is **pending first CI run after push**, because
no runner has seen the workflows (residue R1-17).

### Clusters

| Cluster | Rows | What landed | Verifier | Fragment |
| --- | --- | --- | --- | --- |
| `w3-coach-trust` | 3.1 | a skipped critic is announced (a `critic_skipped` frame, `criticSkipped` and `criticRounds` in the view, a finalize card that says no review ran) and the gate stays; the confirm card shows the finding resolutions the coach parked; declining a proposed stage is a server write (`POST /api/v1/conversations/{id}/stage/decline`, human only); a provider stream is a whole turn only when the provider says so; parallel Responses tool calls stay apart; a stored fact needs a quote of 12 characters and text the quote supports; the coach fences through `agentcontext.FenceData` | round 1 | [w3-coach-trust.md](w3-coach-trust.md) |
| `w3-stream-bounds` | 3.2 | passive SSE streams capped at 20 per user and 100 per project (429 `APP_STREAM_LIMIT`), cursor posts at 25 per second per user; `healthz` bound to 2 s and plain routes given a 25 s deadline (seven exempt patterns); a stream rechecks membership on every heartbeat and ends at access-token expiry, and the web streams refresh and reopen on a 401; the singleton lock retries every second for two minutes; a PRD section is capped at 10,000 characters on write and cut in every prompt | round 1 | [w3-stream-bounds.md](w3-stream-bounds.md) |
| `w3-auth-network` | 3.3 | the throttle and the resend cooldown key on the TCP peer unless `INSIDEOUT_TRUSTED_PROXY_HOPS` says otherwise (default 0); the compose server port is loopback-only; auth mail leaves a bounded background queue (4 workers, depth 64, 30 s a job) so a known address answers like an unknown one; a spent refresh token ends the whole session family after a 10 s grace | round 2 (one repair: the compose loopback guard read nothing and could not fail) | [w3-auth-network.md](w3-auth-network.md) |
| `w3-release-supply` | 3.4 | the release waits for a green `ci` run on the exact SHA, attests the archives and never overwrites a published tag; every tarball carries `LICENSE` and `THIRD_PARTY_NOTICES.md`; token narrowing, dependabot, govulncheck, gitleaks and key-file ignores; every Action pinned by full commit SHA; Go and Flutter archives checksum-verified and the CI Go floor read from `go.mod` | round 2 (one repair: scope, the PM-04 and PM-05 reverts) | [w3-release-supply.md](w3-release-supply.md) |
| `w3-privacy` | 3.5 | `PRIVACY.md` published and linked from the sign-in and sign-up door (on the web the link copies the notice address); the retention and erasure decision record with its five owner questions | round 2 (one repair: scope, three opener files outside the owned paths) | [w3-privacy.md](w3-privacy.md) |
| `w3-docs-truth` | 3.6 | every `PARTIAL` promise is a dated issue and the docs say PostgreSQL 18; unlisted variables documented; the offline coach said to echo only; the README's evidence claim, install order and invite route; the CLI, MCP and client-config architecture doc; HANDOFF and the board show nine open slices; broken links to 0; ledger counts recomputed; guards for all of it | round 2 (one repair: the rv3 board row) | [w3-docs-truth.md](w3-docs-truth.md) |
| `w3-test-hardening` | 3.7 | `internal/testlock` cross-process locks for `signup_mode` and the breaker row; `INSIDEOUT_TEST_STRICT`; every foreign key indexed; API coverage 63.0% to 72.3%; the CLI and the MCP tools driven through the real API; the replay-cutoff question recorded | round 1 | [w3-test-hardening.md](w3-test-hardening.md) |
| `w3-first-run-copy` | 3.8 | `idea convert --locale`; the workspace code and the account invitation have different names; a closed server's register 403 says so; the installers state macOS and Linux and lead with the prebuilt installer; the hosted setup blurb teaches login then `whoami`; the guard `check_first_run_copy.py` | round 1 | [w3-first-run-copy.md](w3-first-run-copy.md) |
| `w3-context-deadlines` | 3.9 | the caller's context reaches `apiclient`, the MCP handlers and the CLI (exit 130 on Ctrl-C after a 400 ms grace); a lost terminal-state write leaves a log line; quick calls get 20 s and AI calls 5 minutes; a node expand gets the model's deadline | round 2 (one repair: the CLI half was unproven by tests; the repair is a test) | [w3-context-deadlines.md](w3-context-deadlines.md) |
| `w3-github-evidence` | 3.10 | commit sync reads with the repo's App installation token instead of the server-wide `GITHUB_TOKEN`; only a successful `deployment_status` is release evidence; a repo binds to one project, and only once GitHub shows it to the server; `FetchGuideFile` refuses dot-only names; `firstLine` cuts between characters | round 1 | [w3-github-evidence.md](w3-github-evidence.md) |
| `w3-client-assets` | 3.11 | the web client loads nothing from a third-party host: CanvasKit bundled, the engine's whole font table mirrored under `client/web/fonts`, a CSP of `default-src 'self'` and `connect-src 'self'`; a release APK allows no cleartext HTTP | round 2 (one repair: `nginx.conf` still named the Google font host, and the ownership of `client/web/flutter_bootstrap.js`) | [w3-client-assets.md](w3-client-assets.md) |
| `w3-client-apply` | 2.3, 2.4 (R2-01) | the web updates feed no longer re-creates the nodes the server applies when a proposal is accepted | round 1 | [w3-client-apply.md](w3-client-apply.md) |
| `w3-users-read-grant` | 2.6 (R2-02) | `users.password_hash` leaves the app role's table reads; login reads the credential through one definer function (migration `20261018200000_users_read_grants.sql`) | round 1 | [w3-users-read-grant.md](w3-users-read-grant.md) |

Seven verifiers accepted in round 1; six returned their cluster once and
accepted the repair in round 2. Three repairs changed what a row says:
`w3-release-supply` reverted rv4-PM-04 and rv4-PM-05, `w3-privacy` made
rv4-PUB-02 `PARTIAL` (the web link copies the address), and `w3-client-assets`
made rv4-COMP-03 `FIXED`; `w3-docs-truth` gained the repair row
rv4-CLOSE-04-BOARD.

### Dispositions

The 64 rows, with red and fix SHAs, the status and what is left open for each,
are in [wave-3-dispositions.md](wave-3-dispositions.md). In short:

- `FIXED` 48. Each has a red test that failed before its fix and passes after
  it, except rv4-TEST-03 (a coverage task: the red is the measured baseline,
  API coverage 63.0% to 72.3%, plus two mutation checks) and rv4-CLOSE-10
  (closed by the integrator's pre-gate patch, no red commit). Six rows say `-`
  in the Red column: those two, rv4-TEST-04 (test-only commits) and the three
  `DEFERRED_OWNER` rows.
- `PARTIAL` 13: rv4-AIDEV-07, rv4-SEC-04, rv4-PUB-03, rv4-SEC-07, rv4-PUB-02,
  rv4-PM-09, rv4-TEST-04, rv4-TEST-06, rv4-CLOSE-08, rv4-FEAT-05, rv4-USER-12,
  rv4-QUAL-04 and rv4-COMP-05. Each names the part that is not done and the R3
  row that carries it.
- `DEFERRED_OWNER` 3: rv4-DATA-06 (five retention and erasure questions),
  rv4-TEST-07 and rv4-CLOSE-09 (one read of the hosted migration table).
  Two more rows carry an owner half beside a finished engineering half:
  rv4-AIDEV-02 (keep or tighten the finalize gate) and rv4-COMP-05 (what proves
  the linker controls the repository).

### Gates (integrator, at `3a1c4ac`)

On a fresh private PostgreSQL 18 that was torn down afterwards:

- `gofmt -l .` prints nothing; `go build ./...`, `go vet ./...` and
  `go mod tidy -diff` (in `server/`) are clean.
- The migrations apply from empty, including the four of this wave, and a second
  `migrate` does nothing.
- `go test ./... -race -count=1 -v`: 28 packages ok, no failure, the only skip
  `TestReadSecretTerminal`. It ran twice, before and after the pre-gate toolchain
  bump (go1.26.3, then go1.26.9).
- Flutter: `pub get --enforce-lockfile` ok (`pubspec.lock` unchanged); `analyze`
  0 errors or warnings and 14 infos; 537 tests pass, also under
  `TZ=Asia/Shanghai`.
- Every `scripts/tests/check_*.py` guard exits 0 once the pre-gate fixups are in
  (the tree holds 14 files, five of them new this wave; the integrator's report
  counts 15); `check_docs_truth.py` and `check_docs_truth_rv4.py` were red
  before them (the testlock layout line, five API file names, two variables).
  `sh scripts/check-redaction.sh` and `sh scripts/check-docs-recorded.sh` print
  nothing.
- `release.yml` packaging, run by hand: four tarballs, each listing `insideout`,
  `insideout-mcp`, `LICENSE` and `THIRD_PARTY_NOTICES.md`; `insideout version`
  prints the stamped tag; four checksums.
- `govulncheck` v1.9.0 over `./...` (the `security.yml` step): 0 affecting
  vulnerabilities after the `go.mod` bump, 19 before it.
- `flutter build web --release --no-web-resources-cdn` (the `ci.yml` client
  step) builds; `index.html` and `nginx.conf` name no Google CDN host.
- Pending first CI run after push, because the tool is not installed here or the
  step needs a runner: `gitleaks`, `actionlint`, `docker compose config`, the
  nginx jobs, `attest-build-provenance`, the `ci`-green gate on the tag SHA
  (it needs `actions: read`), the `ubuntu-install.yml` changes and dependabot.

### Lanes the coordinator ran at the wave-3 head

Run before the records commits, on the integration head, with a private database
and no key set. The coordinator's lane logs show:

| Lane | Observed |
| --- | --- |
| Chrome web tests (`client/test/web`) | 6 tests, all passed (the SSE transport and the download helpers) |
| `scripts/test_env_catalog.py` and `scripts/test_env_writes.py` | 48 passed and 37 passed, 0 failed |
| `dart format` | 282 files, 0 changed |
| key suites | 52 tests run, OK, 33 skipped |
| nginx checks | `check_nginx_headers.py` (a mirrored font answers 200 and a missing one 404, never the shell; the bundled CanvasKit `.wasm` is gzip'd), `check_nginx_reresolve.py` and `check_nginx_webhook_body.py` all OK |
| web release build | built |
| live smoke | 52 passed, 0 failed, all surfaces green; the mail and LLM provider legs skipped because no key was set |
| upgrade replay | green: 31 deployed files replayed, 13 pending migrations met the seeded data |
| roleless lane | `TestRolelessProbeMatchesCatalog` and the roleless tests pass |

None of these touched a real mail, model or GitHub endpoint. A live check that
flips a global row must restore it: one wave-3 run left `signup_mode` open and
the next full run failed until the value was put back (lesson in the follow-ups
below).

### Applied before the gates (coordinator note)

The integrator already applied these seven items itself, in the two pre-gate
commits `4060d64` and `3a1c4ac`; the records commits do not redo them.

1. `.env.example`: the `INSIDEOUT_TRUSTED_PROXY_HOPS` entry (auth-network).
2. Docs truth, dated links: 42 broken links repaired in `docs/changelogs/*` and
   `docs/design-system/CHANGELOG.md`, and the ceilings in
   `scripts/tests/check_doc_links.py` set to 0 (closes rv4-PM-06).
3. Docs truth, PostgreSQL 18 in the unowned files `docs/PLAN.md`,
   `skills/insideout-setup/*` and `scripts/tests/check_docs_truth.py`.
4. Docs truth, the rv3 header: `docs/changelogs/2026-10-11-rv3-remediation/index.md`
   states the locale default as `zh` (closes rv4-CLOSE-10) and the recomputed
   63 closed by code, 10 recorded. The patch carried no `track.yaml` hunk, so
   the rv3 track outcome was restated in the records follow-ups.
5. The testlock layout line and the new API file names (`client_ip.go`,
   `stream_caps.go`, `stream_recheck.go`, `request_deadline.go`,
   `conversations_trust.go`) in `docs/architecture/backend.md`.
6. `docs/usage/environment.md`: rows for `INSIDEOUT_TRUSTED_PROXY_HOPS` and
   `INSIDEOUT_TEST_STRICT`.
7. Release-supply: `toolchain go1.26.9` and `golang.org/x/text` v0.41.0 in
   `server/go.mod`. `govulncheck` v1.9.0 went from 19 affecting vulnerabilities
   to 0; the cluster had measured 29 under go1.26.0 on 2026-10-19, so the first
   `security.yml` run would have been red without it.

The coordinator also asked for one combined disposition of rv4-PM-08: the README
half (docs-truth) and the installers, bootstrap messages and client copy
(first-run-copy). It is one row in the dispositions table.

### Follow-ups closed in `da93d78`

Residue from the worker hand-overs that is only a document, a comment or an index
edit (43 files). The six code files touched hold comments only. No test, script,
workflow or migration changed; each item was read back and the guards pass.

| Closed | Where | Finding |
| --- | --- | --- |
| `GITHUB_TOKEN` is no longer read by commit sync; private repos use the App installation token (the setup check still reads the key, R3-27) | `.env.example` and `docker-compose.yml` (comments), `docs/usage/environment.md`, `docs/usage/key-testing-tutorial.md`, `docs/usage/keys/` | rv4-FEAT-04 |
| The deployment guide: loopback compose port, the client-IP trust chain and `INSIDEOUT_TRUSTED_PROXY_HOPS`, the stream and request bounds, the singleton passage (WARN, retry for two minutes, ERROR after), the self-only CSP and bundled CanvasKit, the users read-grant rollback | `docs/usage/deployment.md` | rv4-SEC-02, rv4-REL-01 to rv4-REL-04, rv4-COMP-03, rv4-DATA-03 |
| Architecture: the `apiclient` layout and deadlines, the session family and its grace, the mail queue, `critic_skipped` and `stage/decline`, `_credential_for_email`, a web feed that applies once, the GitHub evidence rules | `docs/architecture/{backend,prd-coach-agent,database-and-rls,product-subsystems,cli-mcp-and-client-config}.md` | rv4-AIDEV-02, rv4-CLOSE-03, rv4-SEC-03, rv4-SEC-04, rv4-FEAT-02, rv4-FEAT-04, rv4-FEAT-07, rv4-QUAL-04 |
| `releasing.md` rewritten against the integrated `release.yml` (gate, attestation, no-overwrite publish, archive contents), marked pending first CI run; the build stamp, the version policy and the rollback rule (rv4-PM-04, rv4-PM-05) stay open for the proposed `w4-release-version`, so the page says it is not a finished runbook | `docs/usage/releasing.md` | rv4-PUB-03, rv4-PUB-04 |
| The privacy notice and the third-party notices name the self-hosted fonts, CanvasKit and the GitHub App rows | `PRIVACY.md`, `THIRD_PARTY_NOTICES.md`, `README.md` | rv4-PUB-02, rv4-COMP-03 |
| "Invitation code" for the account invitation, the stranger's route to a code, the idea-convert locale flag, per-request deadlines and exit 130, the strict stream rule, the Go toolchain line, the Android cleartext note | `README.md`, `docs/usage/{installation,cli-mcp-and-guidance,local-development}.md`, `docs/usage/keys/llm/llm-schema.md` | rv4-USER-10, rv4-USER-12, rv4-CLOSE-02, rv4-QUAL-04, rv4-AIDEV-05, rv4-PUB-07, rv4-COMP-06 |
| The named-lock rule for a test on a global row | `docs/practices/2026-07-21-verify-against-real-database.md` | rv4-TEST-02 |
| Closure pointers: the web feed's apply and the users read grant are closed in wave 3 | [wave-2.md](wave-2.md), [w2-proposal-apply.md](w2-proposal-apply.md) | FEAT-02-CLIENT-APPLY, rv4-DATA-03 |
| Comment-only code edits: the decline call, the one fence rule, `api` never imports `agent`, the reuse grace name, the web font note | `client/lib/api/requests.dart`, `agentcontext.go`, `api/server.go`, `apiclient/refresh.go`, `native_fonts.dart`, `web_fonts_web.dart` | rv4-CLOSE-03, rv4-LEAD-FENCE-DEDUP, rv4-SEC-04, rv4-COMP-03 |
| The rv3 track outcome restated as 63 closed by code and 10 recorded | `docs/plans/2026-10-10-rv3-remediation.track.yaml` | rv4-CLOSE-04 |
| New: seven reusable notes (a release script that fails open, a handler that never reads its body, removing the client's copy when the server takes over a write, the escape-sweep row, the named lock, CanvasKit and CSS fonts, one owner per shared resource) and the issues index for the six promise records and the retention record | [learning](../../learning/README.md), [issues](../../issues/README.md) | rv4-PUB-03, rv4-PUB-09, rv4-REL-06, rv4-FULL-07, rv4-FEAT-02, rv4-CLOSE-02, rv4-TEST-02, rv4-CLOSE-08, rv4-COMP-03 |

The remaining items need code, a test, a script, a workflow or a migration; they
are the residue rows R3-01 to R3-50, each with an owner and a fix prompt.
`PRODUCT.md` (462 lines) and `docs/SETENV.md` (368) stay over the line budget and
are on the existing
[modularity issue](../../issues/2026-10-18-product-and-setenv-over-line-budget.md);
the dated plan `docs/plans/2026-07-24-roadmap-canvas-collab.md` (430) is over it
too and is on no issue yet (R3-38).

## Carry forward into wave 4

1. **`w4-release-version` (proposed, not started).** It owns
   `docs/usage/releasing.md`, `scripts/install.sh`, `server/Dockerfile` and
   `scripts/bootstrap.sh`, and takes rv4-PM-04, rv4-PM-05, the bootstrap parts of
   rv4-PUB-03 and rv4-SEC-07 (R3-14, R3-15) and R1-04, R1-05, R1-16. Its lead is
   the work that `w3-release-supply` had to revert: `git show 5320251 3b54d25`
   on the integration branch, and a patch the coordinator kept outside the
   repository. Decide the version policy first (item 1 of the release owner
   decisions).
2. **`w4-privacy-opener` (proposed).** Restores a privacy link that opens a tab on
   the web and is tested in Chrome (R3-21): `open_url_stub.dart`,
   `open_url_web.dart`, `client/test/web/open_url_web_test.dart` and
   `auth_door.dart`.
3. **`w4-coach-completion` (candidate).** Derive "the conversation is complete"
   from server state instead of the `NO OPEN QUESTIONS` marker line (R3-01).
4. **Lint the briefs before launching.** List every repo path a cluster's brief
   names and check each is in its owned files; two clusters of this wave were
   returned for exactly that
   ([lesson](../../learning/2026-10-19-one-owner-per-shared-resource-in-a-multi-worker-wave.md)).
5. **The first CI run after a push is the test of every workflow property.**
   Read the release gate (it needs `actions: read`), the attestation,
   `security.yml` (`govulncheck` v1.9.0, `gitleaks` v8.30.1), dependabot,
   `ubuntu-install.yml`, the client web build flag and the nginx checker. A
   failing one becomes a finding; `ci.yml` is at 349 of 350 lines, so a fix that
   adds a line there needs the split first (R3-20,
   [the ci.yml split](../../issues/2026-10-18-ci-yml-split.md)).
6. **Conventions the wave added.** A test on a global row takes the named lock
   (`testlock.SignupMode`, `testlock.Acquire`). A new exported `apiclient.Client`
   method needs a row in the escape sweep; extend an existing method instead when
   you can. A new client verb builds its client with `clientFromFlag` and threads
   the command context. A new authenticated route is a decision in
   `agentRoutePolicy`; a route that decides something is `humanOnly`. A new
   `users` column needs its own `GRANT SELECT`. A new foreign key needs a
   supporting index. A new Action reference is pinned by full SHA.
7. **Leads for the final review (not findings).** The access JWT's missing
   `iss` and `aud` claims and a logout that cannot cut it short (R3-09); the
   strict stream parsers against a real gateway (R3-02); the 10 s reuse grace
   against a slow client; the `users` read grant against the previous image; a
   hosted deployment without `INSIDEOUT_TRUSTED_PROXY_HOPS`; and the
   whole-change adversarial review that closes the plan.

### Operator notes

No new dependency. Four migrations; each re-runs to the same state. What a
running install sees:

- **Set `INSIDEOUT_TRUSTED_PROXY_HOPS=1` on the hosted server before it deploys.**
  The default is 0, which trusts the TCP peer only. The bundled nginx passes
  `X-Forwarded-For` through untouched, so any count above 0 on the compose stack
  would trust a value the client wrote; leave the compose default alone. Without
  the variable behind a proxy every client shares one throttle bucket: eight
  failed logins lock everyone, refresh is capped at 30 a minute, and the symptom
  is a WARN logged once per process.
- **The compose server port is loopback-only.** A host that reached the server on
  its LAN address through compose now has to go through the web container or a
  proxy.
- **The previous image does not survive the users read grant.** Once the
  migration has run, the previous image's `GetUserByID`, `UpdateProfile` and
  `CreateUser` fail with permission denied, so a rolling deploy has a window in
  which an old instance fails them, and rolling the image back alone leaves it
  failing. To restore the previous image, run the rollback `GRANT` of
  [w3-users-read-grant.md](w3-users-read-grant.md) as the owner; it reopens the
  finding until the new image is back. The hosted role was not inspected: the
  migration's `DO` assertion fails it loudly if `PUBLIC` or an inherited role can
  still read the column, so check that before the deploy.
- **The GitHub App must deliver `deployment_status`.** Only a successful one is
  release evidence now. A repository binds to one project; existing duplicate
  bindings are not merged, so the earliest-created owned project holds it and
  the others stop receiving pushes. The canonicalising update bumps
  `updated_at` and cannot be undone.
- **The server no longer reads `GITHUB_TOKEN` for commit sync.** A private repo
  needs the App installed. New codes: `GITHUB_REPO_NOT_VISIBLE` (400),
  `GITHUB_REPO_ALREADY_LINKED` (409) and `GITHUB_REPO_URL_INVALID` (400); the
  web client shows generic text for them.
- **Auth mail leaves a queue in memory** (4 workers, depth 64, 30 s a job). A
  queued mail is lost when the process exits and a full queue drops the newest.
- **Stricter stream parsers.** A model gateway that never sends `message_stop`
  (Anthropic shape) or `response.completed` (Responses shape) now fails every
  turn with a transient error. No real provider was driven; if one fails, the
  escape hatches are the two conditions named in R3-02.
- **Streams and requests.** A stream refusal is 429 `APP_STREAM_LIMIT`; a stream
  ends when its access token expires (15 minutes) and the web client reopens it;
  a PRD section over 10,000 characters answers 400 and refuses the whole save.
- **The singleton guard** now logs a WARN, retries every second for two minutes
  and logs an INFO on takeover; the ERROR comes only if the window ends, so an
  alert on it should read "within two minutes".
- **The web client.** The font mirror adds 21.8 MB of woff2 and 0.64 MB of
  licences to `build/web` (41.7 to 65.1 MiB). The Flutter pin is 3.44.x; the build
  here used 3.44.8 against the 3.44.0 pin, with identical engine font tables. On
  an Android emulator the app reaches the host at `10.0.2.2`; on a device run
  `adb reverse tcp:8080 tcp:8080`.
- **Re-run the lanes on the final head.** The upgrade replay, the smoke script and
  the roleless lane ran at the wave-3 head before the records commits; run them
  again after wave 4 (`/tmp/rv4-patch/replay_lane.sh` and `roleless_lane.sh` are
  the coordinator's scripts, outside the repository).
- **`docs/usage/releasing.md`** was rewritten against the integrated workflow and
  is marked pending first CI run after push. It says it is not a finished
  runbook: the build stamp, the version policy and the rollback rule (rv4-PM-04,
  rv4-PM-05) stay open for the proposed `w4-release-version`.

Nothing is pushed, merged or deployed.
