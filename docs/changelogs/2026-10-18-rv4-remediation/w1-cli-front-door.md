# Wave 1, CLI front door — review-4 remediation

Branch `rv4/w1-cli-front-door`, `1dba3fe..c98bab8` (6 commits: three red/fix
pairs), then this fragment. Findings are cited as `rv4-<ID>` because review
IDs restart per review. Rows 1.1, 1.2 and 1.5 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md).
No address, password or database URL is spelled in this fragment.

## What changed

- **`promote-admin` creates the first admin** (rv4-USER-01, row 1.1, red
  `2d74337`, fix `86b4342`). A fresh install is invitation-only and only an
  admin can mint an invitation, yet `promote-admin` only ran an UPDATE: with
  no users it answered "not found" and nobody could become the first admin.
  When the address has no account the command now reads a new password,
  hashes it with `auth.HashPassword`, creates the account with
  `store.CreateUser` and then promotes it. An account that already exists is
  promoted and its password is never touched. The code is the new
  `promote_admin.go`; `main.go` only routes to it.
  - The password comes from stdin only: hidden and typed twice on a terminal
    (a typo must not lock out the only admin), one line when piped. A second
    argument is refused with exit 2, so a password cannot land in argv or in
    shell history.
  - Same address rule and 8-character floor as the register handler. The
    address is trimmed and lowercased first, as register and login do, so a
    mixed-case argument finds the lowercase row. That is also the
    promote-admin half of the Low finding rv4-DATA-10, which the plan batches
    into the hygiene commit.
  - Without `DATABASE_OWNER_URL` the app role cannot update another user's
    row (RLS), so the role change matches nothing. After creating the account
    the command says so and names `DATABASE_OWNER_URL`; setting it and running
    the command again promotes the existing account with no second prompt.
  - The success line still states the fact (`role`, `created`) and never the
    address; the existing `TestRunPromoteAdmin_LogsTheFactNotTheAddress` stays
    green.
- **Help, version and unknown verbs answer before the server config**
  (rv4-USER-02, rv4-FULL-03, rv4-PM-04, row 1.2, red `104d327`, fix `21a7731`).
  `--help`, `-h`, `help`, `version`, `--version` and any mistyped verb fell
  through to `config.Load()` and died with `config: DATABASE_URL is required`,
  the first thing a new user types. `runFrontDoor` (new `help.go`) now runs in
  `main()` after the product verbs and before `config.Load()`:
  - `help`, `-h`, `--help` print the verb list and exit 0.
  - `version`, `--version` print `insideout <version>` and exit 0.
    `version.go` declares `var version = "dev"`; `release.yml` stamps the tag
    with `-ldflags "-X main.version=${GITHUB_REF_NAME}"`.
  - Anything else that is not a verb prints `insideout: unknown command` with
    the verb and a pointer to `insideout help` on stderr and exits 2, so a typo
    such as `migrat` no longer asks for a database.
  - The server verbs (`migrate`, `seed`, `promote-admin`) and a bare
    `insideout` still go on to `config.Load()`. The container runs
    `ENTRYPOINT ["/insideout"]` with no `CMD`, so no argument must keep
    meaning "run the server" (`TestBareInvocationStillStartsTheServer`).
  - One inventory: `cliVerbs` in `cli.go` holds group, name and a one-line
    summary for all 40 verbs. `help` prints from it and `runFrontDoor` asks it
    which verbs are server verbs. `TestVerbTableMatchesDispatch` parses the
    string-literal `case` arms of `runClientCommand`, `main` and `runFrontDoor`
    and fails when they and the table disagree in either direction, and
    `TestHelpListsEveryDispatchedVerb` checks the printed output the same way.
    Review 3 (ARCH-04) found hand-written verb lists going stale; the header
    comment of `main.go` keeps not enumerating the product verbs and now
    points at the table.
- **The release job runs the tests first** (rv4-TEST-01, rv4-PM-05, row 1.5,
  red `aa677b6`, fix `c98bab8`). `release.yml` runs on every `v*` tag. It built
  four targets and published with no test step, so a tag on a red commit
  shipped. The first step after the Go setup is now
  `go test ./... -race -count=1`; a failing step skips everything after it, so
  nothing is built or uploaded. No `continue-on-error`, no `|| true`.
  - The red test is `scripts/tests/check_release_gate.py` (stdlib only, run
    with `python3 -I`, no YAML parser). It requires a `go test ... -race` step
    before the first `go build` or `gh release`, rejects a fail-open form,
    requires the `./cmd/insideout` build to carry `-X main.version=`, and
    requires `var version` in `version.go`. The linker silently ignores `-X`
    for a symbol that does not exist, so the guard pins the name as well as
    the flag.

## Design choices

- **1.1 create-then-promote with the two existing store calls.** Rejected: a
  new store method that inserts the row as admin (new SQL and a second insert
  path that must keep tracking the register handler), and a bootstrap
  environment variable or one-time token (new config surface and a new secret
  to leak). Two existing, already-tested calls also make a re-run after a
  failed promotion idempotent.
- **1.2 table plus drift test, not a table-driven dispatcher.** Re-routing 40
  verbs through a registry is a larger diff than the finding needs. The
  inventory is one table and a test makes the switches answer to it.
- **1.5 the plan's literal step, in the same job.** A separate test job would
  need a second runner and a `needs:` edge for no extra protection here.

## Recorded limits

- **The release runner has no Postgres service.** The `DATABASE_URL`-gated
  suites skip there, so the new step proves everything that does not need a
  database; the full suite, with its Postgres services, is `ci.yml` on the
  same commit. A tag pushed without a green `ci` run on that commit can still
  publish. Two stronger forms are outside this cluster and need a decision:
  a Postgres service in the release job, or a gate that requires a green `ci`
  run for the tagged commit (it needs `actions: read`).
- **The container image and `scripts/install.sh` build `insideout` unstamped**
  (`server/Dockerfile`, `scripts/install.sh`), so `insideout version` prints
  `dev` there. Follow-up for the owners of those files: an `ARG VERSION` and
  the same `-ldflags` in the Dockerfile.
- **The terminal prompt has no automated test.** The piped path is tested; the
  hidden, typed-twice path needs a pty and was checked by hand (see
  Verification).
- **CI-config properties are pending first CI run after push.** `actionlint`
  is not installed on this machine. `release.yml` was parsed with Ruby's YAML
  loader (steps: checkout, setup-go, test, build, publish, in that order) and
  read by the stdlib guard, but the workflow itself has not run. Its first
  real proof is the first tag run; nothing here claims it green.

## Left for the integrator

None of the following is false after this change; each can now say more. The
files are outside this cluster, and the shared indexes are the integrator's.

- First-admin wording, where the first admin is described: `README.md` lines
  182 and 186 (the wave-3 README rows own those hunks),
  `docs/usage/installation.md` (152, 215), `docs/usage/local-development.md`
  (120), `docs/SETENV.md` (81), `docs/architecture/backend.md` (72) and
  `skills/insideout-setup/SKILL.md` (104): `promote-admin <email>` now creates
  the account and reads its password from stdin when none exists.
- `docs/usage/cli-mcp-and-guidance.md` (line 108 and the verb table): add
  `help` and `version`, and the create-when-absent behaviour.
- Per-ID state in `docs/issues/2026-10-18-review-4-open-findings.md` and
  `docs/issues/README.md` for rv4-USER-01, rv4-USER-02, rv4-FULL-03,
  rv4-PM-04, rv4-TEST-01 and rv4-PM-05, and the promote-admin half of
  rv4-DATA-10.
- This folder's `index.md` wave-1 row, the task board
  (`docs/plans/README.md`), `docs/HANDOFF.md`, `docs/changelogs/README.md` and
  `docs/changelogs/index/2026-10.md`.

## Verification

The red outputs were taken at each red commit, before its fix existed, and the
green outputs after it. They are kept as `w1-cli-front-door-red-<ID>.txt` and
`w1-cli-front-door-green-<ID>.txt` under `/private/tmp/insideout-rv4/reports/`.

- **rv4-USER-01**: three tests fail at `2d74337` (the first admin is not
  created, a password on argv is not refused with exit 2, a short password is
  not refused); two pass because they pin what must not change (an existing
  account's password, the log line). The three failing tests and the
  existing-password pin run the real `main()` in a child process
  (`main_exec_test.go`), so they check exit codes, output and the database
  row; the log-line pin is the existing in-process test. Three tests arrived
  with the fix and have no red of their own: the piped password prompt, the
  missing owner pool, the address normalisation.
- **rv4-USER-02, rv4-FULL-03, rv4-PM-04**: three tests fail at `104d327`
  (help and version exit 1 with the config error, unknown verbs exit 1 instead
  of 2, the help listing is missing); the two guards for what must not change
  pass. `TestVerbTableMatchesDispatch` arrived with the fix and was
  mutation-checked in both directions: deleting a table row fails it with
  "a dispatch switch handles ... but cliVerbs does not list it", and adding a
  row no switch handles fails it with "cliVerbs lists ... but no dispatch
  switch handles it".
- **rv4-TEST-01, rv4-PM-05**: the guard fails three checks at `aa677b6` (no
  `go test -race` step, build not stamped, no `var version`) and passes at
  `c98bab8`.
- **Live, against a private throwaway Postgres cluster** (never the shared
  one): a built binary under `env -i` gives `--help`, `-h`, `help`, `version`
  and `--version` exit 0 and `bogus` exit 2 with no mention of
  `DATABASE_URL`; a bare run still goes on to server startup (here it stops
  on the missing config, the environment being empty), which is what the
  container needs; a build with `-X main.version=...` prints the stamped
  value. The first-admin path end to end: `promote-admin` with the password on
  stdin created the account (`created` true, no address in the log line),
  `insideout login` returned a token, `insideout invite` created an
  invitation (HTTP 201) and `insideout signup-mode` printed
  `invitation_only`. The terminal path, driven on a real pty with a stdlib
  expect-style script: mismatched entries print "the two passwords differ",
  exit 1 and create nothing; matching entries create and promote; neither
  password is echoed.
- **Gates**: `gofmt -l` prints nothing; `go build ./...`; `go vet ./cmd/...
  ./internal/store/...`; `go test ./cmd/insideout/... -count=1` and with
  `-race` (the only skip is the pre-existing pty-gated
  `TestReadSecretTerminal`); `go test ./internal/store/... -run
  'Promote|Invitation|SetSystemRole'` with the database (4 pass, none skip);
  every `scripts/tests/check_*.py`; `sh scripts/check-redaction.sh`;
  `sh scripts/check-docs-recorded.sh`; every changed file is under 350 lines.
