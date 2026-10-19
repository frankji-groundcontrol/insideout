# Wave 4, env truth — review-4 remediation

Branch `rv4/w4-env-truth`, `aa3622b..6e8b3e8` (five commits: three tests and two
fixes), then this fragment. Cluster `w4-env-truth`; finding `rv4-FEAT-04` (the
residue wave 3 listed as R3-27) and one carry-forward row, `rv4-FULL-09`
(residue R3-35). Findings are cited as `rv4-<ID>` because review IDs restart per
review. No value, name or address is spelled in this fragment.

## What changed

- **`GITHUB_TOKEN` text says what the code does** (rv4-FEAT-04, red `f299f44`
  and `ea90c83`, fix `2d46e7b`). Since wave 3 the server does not read the
  token: commit sync reads a private repository with the GitHub App's
  installation token and a public one anonymously. The one reader left is
  `insideout setup --check` (`checkGitHub`, which calls the GitHub API with the
  token to validate it). Four places still said otherwise on the base:
  - `scripts/env.sh` (the check block): the comment, the `ok` line and the
    `info` line said an empty token "keeps GitHub sync at the unauthenticated
    rate limit" and tagged the key "Go server". They now say the token is
    optional and read only by `insideout setup --check`; the empty line adds
    that the server ignores it.
  - `scripts/env-write.sh`: the prompt asked for "GITHUB_TOKEN (optional, enter
    skips)". It now says "optional, read only by insideout setup --check; enter
    skips", and the comment above it says the same.
  - The skip message of `checkGitHub` read "no GITHUB_TOKEN — evidence sync runs
    unauthenticated". It now reads "no GITHUB_TOKEN — nothing to validate
    (optional; only this check reads it)", and the function has a doc comment
    that says why: an absent token leaves nothing to validate and degrades
    nothing. (The brief put the message at about line 120; on this base it was
    line 175.)
  - `docs/usage/environment.md` (the variable's row and the "Compose also
    interpolates" paragraph, which said the token "is passed through but the
    server ignores it") and `docs/usage/keys/github/github-token.md` section 2.
    They now say the server never reads the token and compose does not pass it,
    so a token in the server's environment changes nothing; a private
    repository needs the GitHub App installed on it (`INSIDEOUT_GH_APP_ID` plus
    a private key), a public one syncs anonymously.
  - Not edited because they already told the truth after the wave-3 records
    pass: `docs/usage/keys/README.md`, `docs/usage/keys/github/README.md`,
    `docs/usage/keys/github/gh-app-id.md`, section 7 of
    `docs/usage/key-testing-tutorial.md` and `.env.example`.
- **Compose stops passing `GITHUB_TOKEN`** (rv4-FEAT-04, fix `2d46e7b`). The
  brief allowed keeping the pass-through line or dropping it, with a reason that
  can be proved. Dropped:
  - A search of the server tree for the literal `"GITHUB_TOKEN"` outside tests
    finds one file, `server/cmd/insideout/cli_setup_check.go`, a host-side verb.
    Nothing under `server/internal` reads it.
  - The image is distroless with entrypoint `/insideout`; the container runs the
    server and its `health` self-probe. `setup --check` is a verb for the
    operator's machine (it reads the repository's env file, signs in, pings the
    database) and no document tells anyone to run it in the container. What the
    container does run for itself (the server and its `health` self-probe) never
    reads the value.
  - The line was a promise nothing kept: it invited the belief that a token in
    `.env` changes what a compose deployment does. The host path is unchanged:
    `dev.sh` still exports `.env`, so `setup --check` on the host still sees
    the token.
  - `scripts/test_env_catalog.py` stays green (48 passed): its dead-variable
    lint counts the server's non-test Go as a consumer and `cli_setup_check.go`
    still names the variable. `scripts/test_env_writes.py` (37 passed) does not
    pin the prompt.
  - The comment above the line changed in the same commit, and the choice is
    encoded as guard assertion 4 (below): if the server ever reads the token
    again, the literal appears outside the setup verbs and compose may pass it
    again; until then a server-service variable nobody reads is a failure.
- **`INSIDEOUT_AI_TURN_DEADLINE` reaches the server under compose**
  (rv4-FULL-09 residue R3-35, red `f299f44`, fix `39e1985`).
  `server/internal/agent/coach.go` reads it, `.env.example` and the environment
  reference list it, and compose never passed it, so setting it in `.env` did
  nothing under compose. `INSIDEOUT_AI_TURN_DEADLINE: ${INSIDEOUT_AI_TURN_DEADLINE:-}`
  now sits with the other two coach knobs and their comment is extended (empty
  means the server defaults 4 / 15s / 12m, and the server treats an empty value
  as unset). In `docs/usage/environment.md` the row loses "and
  `docker-compose.yml` does not interpolate it yet, so a compose deployment runs
  the default" (it now says compose passes it, empty when `.env` leaves it
  unset), and the variable joins the "Compose also interpolates" paragraph.
  `docs/usage/keys/llm/README.md` (about line 32) was read and still reads true;
  it is not edited.
  - A `${VAR:-}` line hands the server an empty string, not an absent variable.
    The line is safe only because `turnDeadlineFromEnv` treats an empty,
    unparsable or non-positive value as unset (the 12 m default). A
    characterization test pins that (below); it passed before the line existed.
- **Unchanged.** The key catalog entry, the env scripts' behaviour (only text and
  one prompt changed), the `hosts/` manifests and the CLI probe itself
  (`probe.GitHubProbe`, its wording on success and failure) are as they were:
  whether the variable should be retired is the owner's call (below).

## Tests

- `scripts/tests/check_env_truth_rv4.py` (new, 215 lines, `python3 -I`). Each
  premise is derived from the tree, so a demand disappears with its premise.
  Exit 1 names every offending claim on stderr; exit 2 means a premise could not
  be derived.
  1. (R3-27) No env script, compose file, env example or CLI source ties
     `GITHUB_TOKEN` to sync (a rate limit, an unauthenticated run) or to the Go
     server, however the sentence is wrapped, and a prompt that asks for the
     token says that only `insideout setup --check` reads it. Premise: no server
     code outside the setup verbs (`cli_setup*.go`) reads the literal.
  2. (R3-35) The compose server service passes every `INSIDEOUT_*` variable the
     server reads: the string literals under `server/internal/config` and
     `server/internal/agent`, the definition `check_docs_truth_rv4.py` uses (23
     today). No second hard-coded list.
  3. (R3-35) No living doc (`README.md`, `PRODUCT.md`, `docs/SETENV.md`,
     `.env.example`, `docs/usage/**`, `docs/architecture/**`) says compose "does
     not interpolate", "pass", "forward" or "carry" a variable that
     `docker-compose.yml` interpolates (30 today). The claim is matched by its
     object; in a table row, "it" is the variable of the row's first cell.
  4. (R3-27) Every variable the server service passes is read by some server
     code other than the setup verbs.
- `server/cmd/insideout/cli_setup_check_envtruth_test.go`,
  `TestRvEnvTruthGitHubSkipSaysWhoReadsTheToken`: runs `setup --check` against
  an API that answers 401 with no token anywhere, and requires the `github` line
  to be a skip that says the token is optional and that only this check reads it,
  and not to claim an unauthenticated sync. No database.
- `server/internal/agent/coach_turn_deadline_env_test.go`,
  `TestRvEnvTruthTurnDeadlineEnvKeepsTheDefaultUnlessValid`: an empty,
  unparsable, zero and negative value keep the 12 m default, and `30m` wins.

### Three things about how the guard was built

- **Assertion 3 cannot be red on the base.** There, compose omitted the variable
  and the row said compose does not interpolate it, which was true. It goes red
  the moment the compose line lands without the doc change, and that is how it
  is proven (mutations 12 and 13 below; it was also checked once on the base
  with a temporary compose line). The red that this cluster shows for
  TURN_DEADLINE is assertion 2.
- **`ea90c83` (a second test commit, before any fix).** The first assertion 3
  matched a whole sentence and flagged a true sentence of the new
  `GITHUB_TOKEN` row ("`docker-compose.yml` does not pass it ... `INSIDEOUT_GH_APP_ID`
  plus a private key"). It now matches the object of the claim.
- **`6e8b3e8` (a third test commit, after the fixes).** Mutation probing of the
  fixed tree found two holes in assertion 1, which matched claim words per
  physical line. Prose wraps at about 80 columns, so "GITHUB_TOKEN lifts the
  GitHub API rate" followed by "# limit for commit sync" slipped through; and
  `.env.example` describes the variable in prose above a commented
  `#GITHUB_TOKEN=` line, so the name and the claim never share a line and the
  file the brief lists as covered was in effect unscanned. A run of consecutive
  comment lines is now one text; the sentences that name the variable are
  checked, and so is every sentence of a comment block that ends on its
  assignment. Sentences split on a period plus whitespace, so the compose GitHub
  comment ("guide loads unauthenticated (public repos only)") does not trip over
  its neighbour. The commit is green on the fixed tree (it adds no fix) and its
  proof is mutations 6 to 9 below.

## Evidence

Logs are under `/private/tmp/insideout-rv4/reports/`: `w4-env-truth-guard-red-first.log`,
`-guard-red-final-at-base.log`, `-go-red.log`, `-mutation-guard-round1.log`,
`-mutation-guard-round2.log`, `-mutation-go.log`, `-mutation-agent.log`,
`-gate-cmd-race.log` and the harness `-mutate.py`.

Red on the base (the final guard against the untouched base tree gives the same
seven lines as the first red commit):

```text
scripts/env.sh:195 ties GITHUB_TOKEN to sync or to the Go server ('GITHUB_TOKEN — optional; empty keeps GitHub sync at the unauthenticated rate limit / ...'), but only `insideout setup --check` reads it
scripts/env.sh:196 ties GITHUB_TOKEN to sync or to the Go server ('if [ -n "${GITHUB_TOKEN:-}" ]; then ok GITHUB_TOKEN ... (Go server)"'), but only `insideout setup --check` reads it
scripts/env.sh:197 ties GITHUB_TOKEN to sync or to the Go server ('else info GITHUB_TOKEN "empty — GitHub sync runs at the unauthenticated rate limit (Go server, optio'), ...
server/cmd/insideout/cli_setup_check.go:175 ties GITHUB_TOKEN to sync or to the Go server ('return []checkOutcome{{"github", "skip", "no GITHUB_TOKEN — evidence sync runs unauthenticated"}}'), ...
scripts/env-write.sh:268 asks for GITHUB_TOKEN without saying that only `insideout setup --check` reads it
docker-compose.yml: the server service does not pass INSIDEOUT_AI_TURN_DEADLINE (read at server/internal/agent/coach.go), so setting it in .env has no effect under compose
docker-compose.yml: the server service passes GITHUB_TOKEN, but no server code reads it (the host-side `insideout setup --check` does not run in the container)
exit=1
```

```text
--- FAIL: TestRvEnvTruthGitHubSkipSaysWhoReadsTheToken
github line "  · skip     github — no GITHUB_TOKEN — evidence sync runs unauthenticated" claims an unauthenticated sync, which this token does not control
... want it to say "optional"
... want it to say "only this check reads it"
```

Green on the head: `env truth (rv4) ok: compose carries all 23 server-read INSIDEOUT_*
variables; 30 interpolated variables checked against the docs`, the Go test
passes, and `TestRvEnvTruthTurnDeadlineEnvKeepsTheDefaultUnlessValid` passes
under `-race -count=1`.

Mutation, each applied alone to a scratch copy of the head; a REAL row puts back
text the base carried, a REGRESSION row is a plausible reintroduction where the
base text was already true. All fifteen guard mutations turn red, and the
control run (no mutation) stays green; the repository itself is never touched
(`-mutation-guard-round2.log`):

| # | Mutation | Red |
| --- | --- | --- |
| 1 | REAL: `env.sh` comment, "empty keeps GitHub sync at the unauthenticated rate limit" | assertion 1, `scripts/env.sh:195` |
| 2 | REAL: `env.sh` `ok` line tagged "(Go server)" | assertion 1, `scripts/env.sh:196` |
| 3 | REAL: `env.sh` `info` line, "GitHub sync runs at the unauthenticated rate limit" | assertion 1, `scripts/env.sh:197` |
| 4 | REAL: the CLI skip message, "evidence sync runs unauthenticated" | assertion 1, `cli_setup_check.go` |
| 5 | REAL: `env-write.sh` prompt without "setup --check" | the prompt check, `scripts/env-write.sh:268` |
| 6 | REGRESSION: compose comment, one sentence tying the token to the rate limit | assertion 1, `docker-compose.yml` |
| 7 | REGRESSION: the same claim wrapped mid-phrase ("rate" / "limit") | assertion 1, `docker-compose.yml` |
| 8 | REGRESSION: `.env.example` prose above `#GITHUB_TOKEN=` claims an unauthenticated rate limit | assertion 1, `.env.example` |
| 9 | REGRESSION: a wrapped claim in the Go doc comment of `checkGitHub` | assertion 1, `cli_setup_check.go` |
| 10 | REAL: compose without the TURN_DEADLINE line | assertion 2 |
| 11 | REGRESSION: compose without `INSIDEOUT_TRUSTED_PROXY_HOPS` | assertion 2 |
| 12 | REAL: the environment row says compose "does not interpolate it yet" | assertion 3 |
| 13 | REGRESSION: a paragraph says compose does not pass the queue-wait and concurrency knobs | assertion 3 |
| 14 | REAL: compose passes `GITHUB_TOKEN` again | assertion 4 |
| 15 | REGRESSION: compose passes a variable nothing reads | assertion 4 |

The first round of mutation probing ran ten mutations and caught nine
(`-mutation-guard-round1.log`). The miss was a compose claim wrapped across two
comment lines, which is the hole `6e8b3e8` closes; the row numbers above are
those of round two, where rows 6 to 9 are the cases the per-line guard could not
see and rows 11 and 15 are new. The message of `6e8b3e8` says "16 mutations"; the
harness has fifteen plus the control run, and this paragraph is the correction.

The Go tests, mutated the same way:

| Mutation | Red |
| --- | --- |
| the old skip message put back in `cli_setup_check.go` | `TestRvEnvTruthGitHubSkipSaysWhoReadsTheToken`, three messages |
| `turnDeadlineFromEnv` accepts a non-positive duration | the `zero` and `negative` cases |
| `turnDeadlineFromEnv` falls back to `0` instead of the default | the `empty`, `unparsable`, `zero` and `negative` cases |

Gates on the head, with a private PostgreSQL 18 cluster that was torn down
afterwards (`-gate-cmd-race.log`): `go build ./... && go vet ./...` clean;
`go test ./cmd/... -race -count=1` is `ok` (417 passes counting subtests, no
failure, no data race; one skip, `TestReadSecretTerminal`, which needs a terminal
on stdin and is not a database gate); `go test ./internal/agent -run TestRvEnvTruth -race -count=1`
is `ok`; `check_docs_truth_rv4.py`, `check_docs_truth.py` and
`check_ops_hardening.py` pass; `test_env_catalog.py` (48) and
`test_env_writes.py` (37) pass; `docker-compose.yml` parses as YAML and its
server environment has `INSIDEOUT_AI_TURN_DEADLINE` and no `GITHUB_TOKEN`;
`gofmt -l internal cmd` is empty; no Go file changed since the integration
branch is over 350 lines; every `scripts/tests/check_*.py` passes;
`check-redaction.sh` and `check-docs-recorded.sh` exit 0; `bash -n` passes for
both env scripts. No CI configuration was touched.

## Decisions and deviations

- **Dropped, not kept, the compose line.** The proof is in the first section. The
  cost is that an operator who execs `/insideout setup --check` inside the server
  container would no longer find a token there from compose. Nothing documents
  that path, the probe validates a personal token that belongs to the operator's
  machine, and the host remains the place for that verb.
- **A deviation from the brief's wording of assertion 1.** The brief spelled the
  claims "at the unauthenticated rate limit" and "runs unauthenticated". The
  guard matches the words `unauthenticated`, `rate limit` and `Go server` in a
  sentence (or comment block) that is about the token, which also catches the
  `(Go server)` tag the base put on the `ok` line.
- **`scripts/env-lib.sh`.** The register row for R3-27 lists it. Its only mention
  of the token is a historical comment (a hand-kept name list that had drifted)
  and it makes no claim about who reads the token, so there was nothing to fix.
- **The compose pass-through of the other coach knobs** needed no change: they
  already relied on the server treating an empty value as unset, which is the
  same behaviour the new characterization test pins for the deadline.

## Owner question (DEFERRED_OWNER `GITHUB-TOKEN-RETIRE`)

Should `GITHUB_TOKEN` stay as a CLI-only token probe, or should the variable and
its key guide be retired? Both states are truthful now, so waiting costs nothing.

- **Keep a CLI-only probe.** Nothing more to do. The texts say the token is
  optional and read only by `setup --check`; the key guide, catalog entry, env
  script row and prompt, and the host manifests stay.
- **Retire the variable.** Remove `checkGitHub` and the `github` line of
  `setup --check` with their tests; the env-script row and prompt; the catalog
  entry (`scripts/env_catalog.py`, its tests and
  `server/internal/setup/catalog/catalog_test.go`); the `.env.example` entry; the
  key guide `docs/usage/keys/github/github-token.md` and the links to it from
  `docs/usage/keys/README.md`, `docs/usage/keys/github/README.md`,
  `docs/usage/keys/github/gh-app-id.md`, `docs/usage/key-testing-tutorial.md`,
  `docs/usage/environment.md` and `docs/architecture/product-subsystems.md`; the
  key-placement test `scripts/tests/keys/github/test_github-token.py`; and, per
  host, the `github/tokens.yaml` manifest and its row in the host keys README
  (`hosts/` is outside this cluster). About twenty-five files, a handful of them
  under `hosts/`.

This cluster did not choose.

## Known ceilings and follow-ups

- **The claim words are English.** Assertion 1 matches `unauthenticated`,
  `rate limit` and `Go server`. A claim in other words ("sixty requests an
  hour") or in the Chinese half of a bilingual comment is not seen. Upgrade: add
  the phrase when one appears.
- **Assertion 3 matches one family of denials** ("compose does not
  interpolate/pass/forward/carry X"). A doc that says "compose ignores X" is not
  seen.
- **"Read by the server" means a string literal** under `server/internal/config`
  and `server/internal/agent` (assertion 2) or anywhere in the server tree
  outside the setup verbs (assertion 4). A name assembled at run time is not
  seen; none exists today.

## Integrator notes (wave 4)

Shared indexes and records outside this cluster's files; nothing below was
edited here.

- INTEGRATOR TODO: `docs/changelogs/2026-10-18-rv4-remediation/index.md`: list
  this fragment.
- INTEGRATOR TODO: `docs/changelogs/index/2026-10.md` and
  `docs/changelogs/README.md`: one line for the wave-4 env-truth fragment
  (rv4-FEAT-04 residue R3-27, rv4-FULL-09 residue R3-35).
- INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md`: close row
  R3-35 (line 160; red `f299f44`, fix `39e1985`). Row R3-27 (line 152): the
  prompt, the skip message and the `env.sh` text now say who reads the token (red
  `f299f44`, fix `2d46e7b`) and the row's `env-lib.sh` part needs no change;
  replace the row with the open owner question `GITHUB-TOKEN-RETIRE` (the options
  are above).
- INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-dispositions.md`: the
  rv4-FEAT-04 row (line 187) names R3-27 as open and the rv4-FULL-09 row (line
  158) names R3-35; mark R3-27's text fixed and R3-35 fixed, and add a row for
  `GITHUB-TOKEN-RETIRE` as `DEFERRED_OWNER`.
- INTEGRATOR TODO: `docs/changelogs/2026-10-18-rv4-remediation/wave-3-dispositions.md`:
  rows rv4-FEAT-04 (line 95) and rv4-FULL-09 (line 66) name the residues this
  cluster closes; mark them superseded.
- INTEGRATOR TODO: `docs/usage/key-setup-tutorial.md` (not a file of this
  cluster): lines 126-128 say "The testing tutorial separates manual token sync
  from signed App webhook delivery so each credential is proved by the consumer
  that actually reads it." There is no token sync. Replace "manual token sync"
  with "the manual `insideout sync` path (a private repository is read with the
  App's installation token, a public one anonymously)".
