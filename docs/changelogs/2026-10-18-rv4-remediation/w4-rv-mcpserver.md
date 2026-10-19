# Wave 4, MCP session: in-process renewal, session-swap refusal, fenced file tools

Branch `rv4/w4-rv-mcpserver`, base `90cd9466` (the accepted `w4-rv-apiclient`
result). Six commits, three red/fix pairs. Cluster `w4-rv-mcpserver` of the
whole-change review. Findings are cited as `rv4-<ID>` because review IDs
restart per review; `FR` is that review's prefix. Nothing is pushed, merged
or deployed. No address, host, secret or token is spelled here.

| Finding | Status | Red | Fix |
| --- | --- | --- | --- |
| `rv4-FR-F158`, `rv4-FR-F163`, `rv4-FR-F168`, `rv4-FR-F175` | `FIXED` | `26dd530f` | `f620915b` |
| `rv4-FR-F176` | `FIXED` (withhold-from-the-list variant `DEFERRED_OWNER`, below) | `6a23d8c7` | `362ca341` |
| `rv4-FR-F184` | `FIXED` | `2eda2321` | `9f6a107c` |

The four session findings share one root and one red test
(`TestRvMcpInProcessSignInRenewsAndRearms`, plus the fallback, the final-401
wording and the password-tool restore). F176 and F184 have their own tests.

## What changed

- **An in-process sign-in keeps its refresh token and renews** (F158, F163,
  F168, F175). `auth_register`, `auth_login`, `auth_login_code_verify` and
  `auth_magic_link_consume` store `RefreshToken()` next to the access token.
  `humanClient` arms `SetRenewal` with in-memory `Reload`/`Save` hooks over the
  session struct. The hooks do not read or write the saved login file. The
  final-401 hint on that client is: call `auth_login_code_request` and
  `auth_login_code_verify` again, or `auth_logout` to fall back to the saved
  login. `AgentScoped` does not copy the hint, so `clientCtx` sets it again on
  the client the tool actually calls.
- **A refused in-process credential stops shadowing the saved login.** When the
  agent-token trade ends in `ErrUnauthorized` and a saved or env token exists,
  the in-process pair is marked dead and that call uses the saved login. With
  nothing saved, the error names the auth tools instead of `insideout login`.
- **Each tool call builds its own client, and the next sign-in re-arms.** A
  `Client` serves one caller at a time, and a refusal stays on that client
  until `SetRenewal`. Product and auth handlers also take one process lock for
  the call. A new sign-in replaces the pair, so the following call is armed
  even after a refusal. A renewal cancelled mid-flight still runs `Save`; the
  hook only stores the pair.
- **Signing out with no JWT left puts the password tools back** (F168).
  `DeleteTools` is not otherwise undone. `auth_logout` calls
  `applyAgentProfile` again; if the credential still in force is not a JWT, the
  withheld tools are registered once more. A self-hosted process without mail
  can call `auth_login` again.
- **A held login cannot be swapped until `auth_logout` confirms** (F176).
  `auth_login_code_verify`, `auth_magic_link_consume`, `auth_login` and
  `auth_register` return "a login is already held; run auth_logout first"
  while a credential is in use. `auth_logout` requires `confirm=true` when that
  credential is a JWT. The tools stay on the advertised list (52 with no
  login, 40 with a JWT). After a confirmed logout, one following sign-in may
  replace the saved or env login; the sign-in then locks the gate again.
- **`guide` and `prd_export` say where the file content is** (F184). The fence
  stays. Both descriptions say the file content is the text between the
  `<prd_data>` tags. `TestRvMcpGuideFileIsTheTextBetweenTheTags` strips that
  fence and parses the guide YAML.

## DEFERRED_OWNER

Question: should the agent profile remove `auth_login_code_verify`,
`auth_magic_link_consume` and `auth_logout` from the advertised tool list, or
keep them listed and refuse at call time?

- Option A: withhold them with the password tools. A prompt cannot call them.
  A human restarts the process to change identity. This changes the "40 tools
  for a JWT login" count and the kept-on-purpose list.
- Option B (shipped): keep them listed, refuse a swap until `auth_logout` with
  `confirm=true`. A model can still pass `confirm=true` if the prompt says to.
  That residual is why A is an owner decision rather than a silent change.

## Verification

`go test ./internal/mcpserver/ -count=1` after each fix. The red runs failed
on the claims above: no `/auth/refresh` and the `insideout login` advice; a
magic-link consume that signed the process in; descriptions that still said
to commit the raw tool result. The same tests pass after the fixes. The race
gate is the whole `mcpserver` and `apiclient` packages against a private
database on the port assigned to this cluster; that port is torn down
afterwards. The apiclient package is unchanged.

## Integrator

- INTEGRATOR TODO: `server/cmd/insideout/contract_real_api_mcp_test.go`: after a
  JWT `auth_login`, `auth_logout` needs `confirm=true` (the assertion and the
  cleanup). This cluster does not own that file.
- INTEGRATOR TODO: `docs/usage/cli-mcp-and-guidance.md`: an in-process MCP
  sign-in keeps its refresh token and renews in memory; a final refusal falls
  back to the saved login; swap tools refuse until `auth_logout` with
  `confirm=true`; `guide` and `prd_export` file content is the text between
  the `<prd_data>` tags; password tools return after logout when no JWT remains.
- INTEGRATOR TODO: `docs/architecture/agent-token-scope.md` (Limits): name the
  session-swap residual (tools stay listed; `confirm=true` can be passed by
  the model; withholding them is the owner decision above).
- INTEGRATOR TODO: `docs/changelogs/2026-10-18-rv4-remediation/index.md`: add
  this fragment's row. Shared indexes were left untouched.
