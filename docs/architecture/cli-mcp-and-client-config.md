# CLI, MCP and client configuration

InsideOut has one HTTP contract, `/api/v1`, and four ways to reach it: the web
client ([frontend](frontend.md)), the `insideout` CLI, the `insideout-mcp` stdio
server for agents, and the shared client code under both binaries. This
document is the architecture of the last three, the non-web clients: which
package decides what, how a client finds its server and its identity, how a
saved session renews, and what the server must keep true for them. For how to
use them see [CLI, MCP and the guidance system](../usage/cli-mcp-and-guidance.md);
for the token an MCP process runs under see [agent token scope](agent-token-scope.md).

## Surfaces

| Surface | Code (under `server/`) | Role |
|---|---|---|
| `insideout`, operator verbs | `cmd/insideout/main.go`, `seed.go`, `promote_admin.go` | No argument runs the API server; `migrate`, `seed` and `promote-admin` run beside it. These are the only verbs that load `config.Load()` (`DATABASE_URL`, `INSIDEOUT_JWT_SECRET`) and open a pool. |
| `insideout`, client verbs | `cmd/insideout/cli*.go` | The product CLI, one verb per row of `cliVerbs` in `cli.go`. `runClientCommand` dispatches them before any server environment or database is touched; `help`, `version` and a mistyped verb are answered by `runFrontDoor` for the same reason. |
| `insideout setup` | `cmd/insideout/cli_setup*.go`, `internal/setup/` | The first-run front door: a catalog of the variables InsideOut knows, the `.env` writer, a pure flow state machine, native probes and the huh form. Its hosted branch signs in and saves the session through `clientconfig.Save`; its local branch writes `.env` through the `envwrite` package. |
| `insideout-mcp` | `cmd/insideout-mcp/`, `internal/mcpserver/` | The stdio MCP server: one tool per CLI verb over the same contract, an agent profile, and fenced results. `main` only wires stdio; the server is built (and tested) in `mcpserver.Build`. |
| shared client | `internal/apiclient/` | The HTTP client both binaries speak through: typed calls, the 401 renewal, the agent-token exchange, the hosted default. |
| client configuration | `internal/clientconfig/` | `Resolve`: which server and which identity. The 0600 config file lives here too. |

The API server does not import the clients; the two sides share only the
contract. The plan that shaped the CLI and MCP projections is
[CLI and MCP parity](../plans/2026-08-21-cli-mcp-parity.md).

## One contract, three projections

`cliVerbs` in `cmd/insideout/cli.go` is the verb inventory: `insideout help`
prints it, and `TestVerbTableMatchesDispatch` fails when it and the dispatch
switches drift apart. The MCP server registers one tool per verb
(`addProductTools`, `addAdminTools`, `addAuthTools`), and a read tool returns
the API's own JSON rather than a reshaping of it, so a surface cannot invent a
field. What no test checks is that the three lists (CLI verbs, MCP tools,
`apiclient` methods) agree with each other: that parity is by convention, and
review 4 records it as rv4-ARCH-07 in
[the architect review](../review/2026-10-18-multirole-4/10-architect.md). Ask
`tools/list` for the MCP tool set; a count written in prose drifted twice.

A client verb runs under one context (`runInterruptible` in `cli.go`): the
first Ctrl-C or SIGTERM cancels it, the request in flight aborts, and the
command ends with status 130 and `insideout: interrupted` on stderr. A command
that is still running 400 ms later, or a second signal, ends the process at once.
Before this the default signal action killed the process wherever it stood,
even between the server rotating a refresh token and the CLI saving the new
pair (rv4-QUAL-04). One client-building site is not yet on that context:
the sign-in probe of `setup --check` (`runHealthCheck` in `cli_setup_check.go`),
so a first Ctrl-C does not abort its request and the process ends at the 400 ms
grace instead (open residue R3-24 in
[the open findings](../issues/2026-10-18-review-4-open-findings.md)).

## Finding the server and the identity: `clientconfig.Resolve`

Every CLI verb, every `--api` flag and the MCP server resolve credentials through
one function, `clientconfig.Resolve(flagAPI)`. Nothing else reads
`INSIDEOUT_API`, `INSIDEOUT_TOKEN` or the config file.

| | First that is set wins |
|---|---|
| **Base** (the server) | the `--api` flag, then `INSIDEOUT_API`, then the saved login's `api_base`, then `apiclient.DefaultBase` (the hosted API) |
| **Token** (the identity) | `INSIDEOUT_TOKEN`, an explicit credential for whichever base is in effect; else the saved token, but only when the effective base is the one the token was saved for |
| **Refresh token** | the saved one, and only together with the saved token; `INSIDEOUT_TOKEN` never has one |

A config saved without an `api_base` was saved for the hosted default. The rules
that follow from the table:

- A token saved for server A is never sent to server B. A mismatched base means
  signed out for that server, not a silent fall back to the saved token.
- `Load` reads the file and nothing else; overlaying the environment per field
  would pair one server's saved token with another server's base. A missing
  file, or no config directory at all, means nothing was saved.
- A config file that exists but cannot be read or parsed is an error naming the
  path, and the verb stops. It never falls back to the hosted default.
- The file is `os.UserConfigDir()/insideout/config.json`, holding `api_base`,
  `token` and `refresh_token`. `Save` creates the directory 0700, writes a temp
  file beside the target, sets it 0600, then renames it over the target, so the
  file is never observable half-written or world-readable.
- Three writers exist: `insideout login` and `register` (`cli_auth.go`, the path
  goes to stderr and the token is the only stdout), `insideout setup`
  (`cli_setup.go`), and session renewal below, which both binaries run.

## Session renewal

An access token is short-lived (`INSIDEOUT_ACCESS_TTL`, 15 minutes by default);
the saved session also holds the server's rotating one-time refresh token. A
client built from the saved session (`Creds.Client()`) renews itself on 401:

- The transport answers a 401 on an authenticated, non-`/auth/` request once:
  renew, replay the request with the new access token, return what that gets.
  A request whose body cannot be replayed is not retried.
- Renewals are serialized in a process (the refresh token is one-time, so a burst
  of expired calls renews once and the rest adopt the result). Before renewing,
  the `Reload` hook re-reads the stored pair, so a second process that already
  rotated the token is adopted instead of racing with a retired one. A renewal
  that loses the race anyway polls the store for the winner's pair for 500 ms
  (`renewalGrace`) before it takes the refusal as the end of the session. The
  server backs that wait: a spent refresh token offered again within
  `refreshReuseGrace` (10 s, `internal/api/auth.go`) of its rotation is refused
  without ending the session family, and only a later reuse revokes the family
  (rv4-SEC-04). The 500 ms poll stays the ordinary path; a lock file beside the
  config is the upgrade path if that grace is ever shortened below the wait.
- `Save` stores the rotated pair only while the file still holds a login for the
  same server; a newer `login --api <other>` in another terminal is neither
  overwritten nor adopted. If the write fails the old refresh token is already
  dead on the server, so the client says so on stderr once.
- Never armed: `INSIDEOUT_TOKEN` and a login made inside an MCP process with the
  `auth_*` tools. Renewing either with the saved session would swap one identity
  for another.

## The MCP process

`insideout-mcp [--api URL]` resolves its server and token like the CLI, so a
client entry needs no `env`. The server is fixed when the process starts; the
saved login's tokens only ever go to the server they were issued by.

- Two clients exist inside it. `humanClient` is `Resolve` plus the process's own
  login (an `auth_*` tool's token is process state, never conversation output);
  only the `auth_*` tools and the token exchange use it. `client` is
  `apiclient.AgentScoped(humanClient(api))`, and every product tool calls
  through that: the login traded for an agent-scope token, held in one slot per
  server and user and minted again with about a third of its life left. A failed
  trade is a client that fails every call; no tool falls back to the full login.
- The registry has a profile. When the process holds a JWT login,
  `applyAgentProfile` removes the tools in `withheldFromAgents` (Commit, the
  proposal decision, the instance switches, the deletes, the password tools).
  Without a login the registry stays whole so the process can still sign in.
  That removal is advice to the model; the server's 403 for an agent-scope
  token is what holds. See [agent token scope](agent-token-scope.md), which also
  states the deploy order (server first, then the new binary).
- Each call rides the context of what is running it. `apiclient.Client` has no
  `http.Client` timeout; `SetContext` binds a context and every request is cut
  by a deadline picked from its route (`deadline.go`): 20 s for a call the
  server answers from its database, 5 minutes for the two that wait on the
  planner's model (`POST /prds/{id}/build` and `POST /roadmap/{id}/expand`;
  `aiRoutes` is a hand-kept list, so a new model-calling route has to be added
  to it). In the MCP process the tool call's context is that context, so a
  host that cancels a call frees the server request; the agent-scope trade rides
  the same context (rv4-QUAL-06, rv4-REL-06, rv4-FULL-07).
- Every product result opens with a one-line label and carries its data inside
  one `<prd_data>` fence (`fence.go`, the same fence the coach's prompts use,
  `agentcontext.FenceData`), with tag-like text broken so the data cannot close
  the fence or fake another. The `auth_*` tools answer in the server's own
  words and are not fenced.

## Install and release

- From source: `scripts/install.sh` needs a Go toolchain, builds both binaries
  and installs them into `~/.local/bin` (or the first argument, or
  `INSIDEOUT_INSTALL_DIR`), adding that directory to the shell's PATH once.
- Prebuilt: `release.yml` runs on a `v*` tag and builds linux and darwin for
  amd64 and arm64 with `CGO_ENABLED=0`, tars each pair, writes a sha256
  `checksums.txt` and publishes `bootstrap.sh` beside them. `bootstrap.sh`
  refuses any OS other than macOS and Linux, needs `shasum` or `sha256sum`, and
  installs nothing when the checksum line is missing or does not match.
  There is no Windows build. `insideout setup --local` needs a repository
  checkout (it reads `docker-compose.yml`, `.env.example` and the migrations),
  and `--postgres bundled` also needs Docker.
- The hosted default is `apiclient.DefaultBase`, a package-level `var` on one
  line of `internal/apiclient/default_base.go` so a build can retarget it:
  `go build -ldflags "-X <module>/server/internal/apiclient.DefaultBase=<url>"`.
  `scripts/default-api.sh` reads that line. The release workflow passes only
  `-X main.version`, so binaries released from a fork aim at the upstream hosted
  API until the fork adds the flag; wiring it is an owner decision (rv4-ARCH-05).

## The single-replica contract

The API server keeps some state in process memory: the sign-in throttle and the
email resend cooldown, the conversation turn lock, and the live fan-out behind
presence and roadmap progress (`internal/presence`, `internal/progresshub`).
That state is correct only with exactly one server process per database, so the
contract is stated and checked rather than assumed
([operator note](../usage/deployment.md#run-exactly-one-server-replica)).

- When `Handler()` builds the middleware chain, `startSingletonGuard`
  (`internal/api/single_replica.go`) logs the contract once, then tries the
  Postgres advisory lock `insideout:singleton` (`pg_try_advisory_lock`) on a
  connection hijacked out of the pool. Hijacked, the connection sits outside the
  pool budget (the pool still serves its full size) and shutdown does not wait
  on it.
- The holder keeps the lock for the process lifetime; Postgres ends the session,
  and so the lock, when the process exits. A second process finds the lock held
  and logs an ERROR naming it.
- It is advisory: it never blocks startup or serving, and a database that
  cannot be reached for the check only logs a warning. It is separate from the
  `insideout:migrate` lock, which only serializes migration runs.

## Known limits

- CLI, MCP and `apiclient` parity is by convention (above).
- Hosted-default retargeting is wired in the code but not in the release workflow.
- The single-replica guard detects a second process; it does not prevent one.
- Prebuilt binaries cover macOS and Linux on amd64 and arm64 only.
