# Wave 4, `setup --check` — review-4 remediation

Branch `rv4/w4-cli-setup-check`, `582a4bd..dc98a1f` (four commits: two red tests
and two fixes), then this fragment. Cluster `w4-cli-setup-check`; findings
`rv4-QUAL-04` (the residue wave 3 listed as R3-24) and `rv4-FR-F69` (an
additional row of this cluster; the register has no row of that id on the
base). Findings are cited as `rv4-<ID>` because review IDs restart per review.
No value, name or address is spelled in this fragment: loopback addresses and
their ports are written `<loopback>:<port>`.

## What changed

- **The probes run under the CLI context** (rv4-QUAL-04, red `a176d1d`, fix
  `67965f0`). `runHealthCheck` built its 60 s deadline on
  `context.Background()` and `checkSignedIn` asked the API with a client that
  had no context, so the check ignored the context wave 3 made every other verb
  run under. A Ctrl-C reached `cliCtx` and nothing waiting on the check
  noticed: the request, the database ping and the live model probe ran until
  they answered or the deadline struck, and the process ended only because
  `interruptGrace` (400 ms) forced `os.Exit(130)` in the middle of a probe,
  with nothing printed. Now the deadline is `context.WithTimeout(cliCtx, 60s)`;
  `checkSignedIn(ctx, creds)` binds its client with the existing `SetContext`
  (no new exported `apiclient` method), so the request, a session renewal
  included (it runs under `req.Context()`), ends with the interrupt; the
  anonymous API probe, the database ping and the live model probe already took
  the check's context and now inherit the CLI's.
  - A probe the interrupt cuts short prints `skip ... cancelled (interrupted
    before it answered)`. It is a skip, not a failure: the target was never
    judged. It no longer reads "unreachable (offline?)", a database failure or,
    when the answer comes late, `ok`.
  - A run that was interrupted ends with exit 1 and the line `interrupted — the
    run did not finish, so it is not a pass`; `runInterruptible` turns that 1
    into 130 and prints `insideout: interrupted`. It is not exit 0: with every
    probe a skip the run would say "all configured checks passed", and
    `runInterruptible` leaves a zero exit alone, so a SIGTERM from a supervisor
    or `timeout` would have reported a pass for a check that never finished.
  - `commandInterrupted` tests `context.Canceled` only, which only the CLI
    context produces. The check's own 60 s deadline is `DeadlineExceeded` and
    keeps its old reading (a timeout is "could not reach").
- **Only a target that could not be reached is the offline skip** (rv4-FR-F69,
  red `11f58eb`, fix `dc98a1f`). `checkSignedIn` took every `*url.Error` for
  "unreachable (offline?) — not failed", and the anonymous lane took every
  error `probe.APIProbe` returned, because `APIProbe` wraps every transport
  failure as `ErrUnreachable`. A certificate this machine does not trust, a
  scheme that is not http(s), an https address on a plain-http port and a peer
  that hangs up all passed the check with exit 0: the cases a check of the API
  address exists to catch, and room for a middlebox in front of the API with
  the check saying nothing. `targetUnreachable` is now a whitelist and both
  lanes use it: a resolver error (`*net.DNSError`), a `*net.OpError` whose
  operation is `dial` or `proxyconnect`, or any `net.Error` that reports
  `Timeout()` means "could not reach" and skips. Everything else fails, and the
  anonymous lane no longer puts the probe's `unreachable: ` prefix in front of a
  cause that is not that, so the line names what went wrong (`x509: certificate
  signed by unknown authority`, `unsupported protocol scheme`). For the
  certificate, scheme and plain-port cases the failure comes before any request
  is sent, so the credential never reaches such a peer; the red test counts
  what the peer served.
- **Unchanged.** An unreachable API is a skip (exit 0), a rejected token a
  failure, a signed-in user `ok`, a 5xx a failure; the GitHub probe keeps its
  wording and behaviour; `cli_setup_check_test.go` and
  `cli_setup_check_creds_test.go` are not edited and stay green; `apiclient` and
  `probe` are not edited.

## Tests (new files, `TestRvSetupChk...`)

- `cli_setup_check_ctx_test.go` (QUAL-04). An in-process matrix of four probes
  (anonymous API, signed-in API, database, live model probe) and two moments:
  the CLI context cancelled before the check starts, and cancelled when the
  probe's request reaches a listener that is slow on purpose (the wave-3
  `holdingAPI` and `cancelOnArrival`; for the database a TCP listener that takes
  the connection and never speaks). Each case requires the check back within
  2 s, the probe's line to say `skip` and `cancelled`, exit 1, `interrupted`
  and no "all configured checks passed" in the output and, for the first
  moment, that nothing reached the listener. A child test sends a real SIGINT
  to `insideout setup --check`, signed in and anonymous, and requires exit 130.
- `cli_setup_check_tls_test.go` (FR-F69). Four peers times two lanes: a TLS
  server under a certificate no trust store knows, https against a plain-http
  server, the scheme `htp://`, and a listener that closes every connection.
  Each case requires exit 1, a `fail` line without "not failed" and, for the
  two servers, that the peer served no request.
- `cli_setup_check_class_test.go` (FR-F69, added with the fix because it needs
  `targetUnreachable`). A nineteen-row table for the classifier: the errors a
  test cannot make a local server produce (a name that does not resolve, a down
  proxy, a deadline) beside the ones it can, pinned both ways; and a lane test
  that the failing line names the cause (`certificate`) and does not say
  `unreachable`.

### The test that would have passed without the fix

The first draft of the Ctrl-C child test was green on the base: with the
default 400 ms `interruptGrace` the forced `os.Exit(130)` ends the process
while the probe still waits (0.42 s, measured; the report is printed only after
the last probe, so nothing is printed). Wave 3 met the same trap. The child therefore sets the grace to one hour through the existing
test hook (`graceEnv`). On the base it then ends with exit 0 once the held API
gives up at its 3 s backstop, and the test is red; exit 130 can now only come
from the cancelled context aborting the probe and the check returning by itself.

## Evidence

Logs are under `/private/tmp/insideout-rv4/reports/`:
`w4-cli-setup-check-{qual04,frf69}-{red,green}.log`, `-qual04-race15.log`,
`-frf69-race40.log`, `-mutation.log` and `-gate-race.log`. None of the new
tests needs the database.

QUAL-04, red on the base (`-run 'TestRvSetupChk(AnInterrupted|CtrlC)'`: twelve
failing ids, 30 s because every case waits out the 3 s backstop):

```text
the check took 3.0s, want it to end with the cancelled context, not wait for the probe or the 60 s deadline
api line = "ok       api — <loopback>:<port>/api/v1 reachable and already authenticated", want a skip that says the probe was cancelled
db line = "fail     db — unreachable: failed to connect to ...: timeout: context deadline exceeded", want a skip ...
llm line = "ok       llm — provider answered /models (HTTP 200)", want a skip that says the probe was cancelled
exit 0, want 1: a check cut short must not pass
after SIGINT the command ended with <nil>, want exit status 130 (stderr "")
```

Green after the fix: the matrix takes 0.02 s, the child 0.03 s, and
`-race -count=15` of the new tests is `ok`.

FR-F69, red on the base and on the QUAL-04 fix (all eight live cases):

```text
exit 0, api line "skip     api — https://<loopback>:<port>/api/v1 unreachable (offline?) — not failed"; want a failed check, not an offline skip
exit 0, api line "skip     api — htp://<loopback>:<port>/api/v1 unreachable (offline?) — not failed"; want a failed check, not an offline skip
```

Green after the fix: every case passes, and `-race -count=40` over the new
tests and the existing check tests is `ok`.

Live proof with the real binary (a `go build` of the head, an isolated HOME, no
checkout, only the API address and a token in the environment; a throwaway
self-signed certificate and a server that holds each request for 60 s, both on
loopback and stopped afterwards):

| Run | Result |
| --- | --- |
| untrusted certificate, signed in | `fail api — apiclient: GET /me: Get "https://<loopback>:<port>/api/v1/me": tls: failed to verify certificate: x509: certificate signed by unknown authority`, exit 1 |
| untrusted certificate, anonymous | `fail api — Get "https://<loopback>:<port>/api/v1/me": tls: failed to verify certificate: x509: certificate signed by unknown authority`, exit 1 |
| scheme `htp://`, signed in | `fail api — apiclient: GET /me: Get "htp://<loopback>:<port>/api/v1/me": unsupported protocol scheme "htp"`, exit 1 |
| nothing listening, signed in and anonymous | `skip api — http://<loopback>:<port>/api/v1 unreachable (offline?) — not failed`, exit 0 |
| Ctrl-C one second into a held request, signed in and anonymous | `skip api — cancelled (interrupted before it answered)`, the `interrupted — the run did not finish` line, stderr `insideout: interrupted`, exit 130, 0.00 s after the signal |

Mutation, each applied alone to the head and run against the tests that should
catch it (`-mutation.log`); all fifteen turn red and the file is restored:

| Mutation | Red |
| --- | --- |
| deadline back on `context.Background()` | the matrix (all eight cases) and both child cases |
| `checkSignedIn` without `SetContext` | the two signed-in matrix cases and the signed-in child |
| anonymous lane without the cancelled case | the anonymous matrix cases |
| signed-in lane without the cancelled case | the signed-in matrix cases |
| no run-level `interrupted` block (exit 0) | the matrix and both child cases |
| database without the cancelled case | the database matrix cases |
| live model probe without the cancelled case | the model-probe matrix cases |
| anonymous lane skips every wrapped error | the anonymous TLS cases and the cause-naming test |
| signed-in lane skips every error | the signed-in TLS cases and the cause-naming test |
| `proxyconnect` not whitelisted | table row "the proxy is down" |
| timeouts not whitelisted | table rows "deadline passed" and "went quiet" |
| resolver errors not whitelisted | table rows "no such host" and "a bare resolver error" |
| anonymous failure keeps the `unreachable: ` prefix | the cause-naming test |
| any `*net.OpError` counts as unreachable | table row "the peer reset the connection" |
| any `net.Error` counts as unreachable | the TLS cases, the table and the cause-naming test (24 ids) |

Gates on the head, with a private PostgreSQL 18 cluster that was torn down
afterwards (`-gate-race.log`): `go build ./... && go vet ./...` clean;
`go test ./cmd/... ./internal/apiclient/... -race -count=1` is `ok` (584 passes
counting subtests, no failure, no data race; one skip, `TestReadSecretTerminal`,
which needs a terminal on stdin and has nothing to do with this cluster or the
database); `gofmt -l` clean; no Go file over 350 lines; the fourteen
`scripts/tests/check_*.py` pass; `check-redaction.sh` and
`check-docs-recorded.sh` exit 0.

## Decisions and deviations

- **A whitelist, not a blacklist.** "Fail on TLS errors" would pass every error
  nobody thought of, which is the shape of the original defect. `errors.As`
  walks the `%w: %w` chain, so the `APIProbe` wrap does not hide the cause.
- **`proxyconnect`.** net/http reports a dial failure toward a proxy as an
  `*net.OpError` with that operation, not `dial`. A proxy that is down is "could
  not reach", like the target. A proxy that answers `CONNECT` with an error
  status is not a dial failure and fails.
- **Cancelled is a skip, exit 1.** A fail would say the target is broken; a
  skip with exit 0 would let an interrupted run pass (see above). The extra
  `interrupted` line is what carries the exit code.
- **Behaviour change to expect.** A peer that closes or resets the connection
  used to skip and now fails, with the cause on the line. A server that
  restarts at the wrong moment now fails a check that used to skip; a re-run is
  cheap, and skipping on EOF would be the middlebox hole again.
- **Read timeouts still skip.** A server that accepts the connection and does
  not answer within the deadline is a `net.Error` timeout, one of the "timed
  out" cases the skip was written for.

## Known ceilings and follow-ups

- **GitHub lane.** Untouched, as briefed. With a configured `GITHUB_TOKEN` a
  Ctrl-C still prints `api.github.com unreachable (offline?) — not failed` for
  the probe it cut short (the run still ends 1, then 130, through the run-level
  line). The lane also has the FR-F69 shape: every transport error is the
  offline skip, so an untrusted certificate on the way to the host passes. Fix,
  in `checkGitHub`: a `commandInterrupted(ctx)` case first, then
  `errors.Is(err, probe.ErrUnreachable) && targetUnreachable(err)` for the skip.
- **Live model probe (`--llm`, legacy, opt-in).** The interrupt is handled.
  `llmModelsProbe` wraps its transport error with `%w: %v`, which drops the
  cause chain, so `targetUnreachable` cannot classify it and an untrusted
  provider certificate still prints `skip ... provider unreachable`. Fix: wrap
  with `%w: %w` and guard the skip with `targetUnreachable`. Neither finding
  names it, so it is left for a decision.
- **The hang-up peer.** The listener that closes every connection has no handler
  to count, so "served nothing" is true by construction for it; only the
  certificate and plain-port servers prove that nothing reached the peer. Over
  plain http the request is already written when the hang-up is seen.
- **Not driven:** a session renewal in flight when the Ctrl-C arrives. The
  renewal runs under `req.Context()` (`internal/apiclient/refresh.go`), so
  binding the client covers it by construction; this cluster's tests hold the
  first request, not the refresh round trip.

## Integrator notes (wave 4)

Shared indexes and records outside this cluster's files; nothing below was
edited here.

- INTEGRATOR TODO: `docs/changelogs/2026-10-18-rv4-remediation/index.md`: list
  this fragment.
- INTEGRATOR TODO: `docs/changelogs/index/2026-10.md` and
  `docs/changelogs/README.md`: one line for the wave-4 `setup --check` fragment
  (rv4-QUAL-04 residue, rv4-FR-F69).
- INTEGRATOR TODO: `docs/architecture/cli-mcp-and-client-config.md`: lines
  46-50 say one client-building site, the sign-in probe of `setup --check`, is
  "not yet on that context" (open residue R3-24). Replace them with: the check
  runs its probes under the same context; a probe the first Ctrl-C cuts short
  prints `skip ... cancelled` and the run ends 1, which becomes 130.
- INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md`: close row
  R3-24 (fix `67965f0`, red `a176d1d`); add the GitHub-lane and live-model-probe
  follow-ups above as new rows with an owner.
- INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-dispositions.md`: row
  rv4-QUAL-04 (line 183) from `PARTIAL` to `FIXED` (red `a176d1d`, fix
  `67965f0`); add a row for rv4-FR-F69 (cluster `w4-cli-setup-check`, `FIXED`,
  red `11f58eb`, fix `dc98a1f`).
- INTEGRATOR TODO: `docs/changelogs/2026-10-18-rv4-remediation/wave-3-dispositions.md`:
  row rv4-QUAL-04 (line 91) names a residue this cluster closes; mark it
  superseded.
- INTEGRATOR TODO: `docs/usage/installation.md`: lines 198-199 (the offline-skip
  sentence): add that a certificate this machine does not trust, a scheme that
  is not http(s) or a connection the peer breaks fails the check, and that
  Ctrl-C ends the check with status 130, the probe it cut short reporting
  `skip ... cancelled`.
