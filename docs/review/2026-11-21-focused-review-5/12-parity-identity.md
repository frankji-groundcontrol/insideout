# 12 · Surface parity: identity and workspace

**Lens:** REST, web, MCP and CLI for sign-in, accounts, admin, workspaces, ideas
**Score:** **6.2 / 10**
**Bottom line:** Sign-in is consistent across REST, web, CLI and MCP, and the agent-token wall is enforced on the server. Two defects cause wrong behaviour today: `insideout forgot-password` with stdin at /dev/null sends nothing and exits 0 (reproduced), and `idea update` without content erases the idea body on CLI and MCP. Workspace settings, members, profile edit and account deletion have no CLI verb, account deletion has no web control either, and SKILL.md claims parity the code does not have.

Read at `main` @ `4d186da` together with the uncommitted working tree. A skeptic re-checked every High in this file; the verdicts are in the table near the end and in the [findings register](./01-findings-register.md).

---

## Scope read

I read the route tables in `server/internal/api` (auth, me, account deletion, reset watch, admin, workspaces, members, ideas, agent scope), the Go client in `server/internal/apiclient`, every MCP tool file in `server/internal/mcpserver`, the CLI dispatch and verb files in `server/cmd/insideout`, the Flutter client in `client/lib/api` with the pages that call it, and the three documents that describe these surfaces: `docs/usage/cli-mcp-and-guidance.md`, `SKILL.md` and `server/cmd/insideout/help.go`. I ran `go test` for the three client-surface packages and reproduced one CLI defect against an unreachable port. No server, database or migration was touched. The capability matrix (37 rows) is in the structured output.

## What is good

- The agent-token wall is one fail-closed table on the server (`server/internal/api/agent_scope.go:39`), so every client meets the same 403.
- The five sign-in paths send the same camelCase bodies from the Go client (`server/internal/apiclient/auth.go:12`) and the web client (`client/lib/api/client_auth.dart`).
- The CLI help listing is generated from the verb table (`server/cmd/insideout/cli.go:42`) and a test pins it to the dispatch switch.
- Emailed-auth requests answer a uniform 200 (`server/internal/api/auth_email_flows.go:36`), so no surface can reveal which accounts exist.
- CLI logout revokes the server session before clearing the saved login (`server/internal/apiclient/logout.go:27`).

## Findings, worst first

**PARA-01 (High)** `server/cmd/insideout/cli_auth.go:55`: `if isTerminal(os.Stdin) && !confirmMailSend(email) {`. The helper tests for a character device, and `/dev/null` is one. With stdin detached the command prints `Email not sent.`, sends nothing and exits 0. Reproduced; it is also why `TestEveryCLIClientEndsWithTheCommandContext/runAuthCommand` is red. Fix: use `term.IsTerminal(int(os.Stdin.Fd()))`, as `readSecret` does at line 268.

**PARA-02 (High)** `server/internal/apiclient/crud.go:24`: the client always sends `"content": content`, and `server/internal/store/ideas.go:148` writes `SET title = $2, content = $3`. The CLI and MCP document content as optional, so a title-only update erases the idea body. The web form always sends both. Fix: keep the stored content when the key is absent, as commit 269391f did for a project's ownerId.

**PARA-03 (Medium)** `client/lib/api/client_auth.dart:144`: `updateMe` is the last `/me` method. `DELETE /api/v1/me` has no web control, no Go client method and no CLI verb; `PRIVACY.md:110` tells the user to send the request by hand. Fix: add a profile-page control.

**PARA-04 (Medium)** `server/internal/mcpserver/session_withhold.go:17`: three session tools are hidden from `tools/list` while a login is held, yet `server.go:97` says `call auth_login_code_request and auth_login_code_verify again, or auth_logout`. `docs/usage/cli-mcp-and-guidance.md:198` says `the code and magic-link tools stay`, and `SKILL.md:110` says forty tools; the signed-in list is 37. Fix: make the hints say to restart the process and correct both documents.

**PARA-05 (Medium)** `client/lib/api/client_auth.dart:69`: `forgotPassword` returns nothing. The server reports `mail: accepted|logged|dropped`, but only `insideout forgot-password` and the setup wizard show it. The web, the MCP tools and `insideout login --code|--magic` report success even when the queue dropped the mail. Fix: carry the word to every caller.

**PARA-06 (Medium)** `SKILL.md:107`: `Every verb exists 1:1 as CLI command ↔ MCP tool ↔ API route`, and lines 119-120 say the CLI can delete a project, workspace, member or update. The Go client issues DELETE only for ideas and roadmap nodes. Nine identity and workspace routes are web-only. Fix: correct the text.

**PARA-07 (Medium)** `server/internal/mcpserver/server.go:64`: seven tools and the write half of `signup_mode` are registered only while the process has no login, which is the one state where their routes answer 401. They can never succeed. Fix: stop registering them; keep `signup_mode` as a read.

**PARA-08 (Medium)** `server/internal/apiclient/auth.go:109`: `"/auth/reset-watch/"+id` bypasses the escaping helper, and four new methods have no escape-sweep row, so the package test is red. Fix: use `apiPath` and add the rows.

**PARA-09 (Low)** `server/internal/store/users.go:125`: `avatar_url = $5` is written even when the key is omitted. The web never sends it, so each profile save clears it; `coverUrl` behaves the same.

**PARA-10 (Low)** `server/internal/api/auth_reset_watch.go:22`: the cooldown branch returns no watch id, although the comment at line 16 says it always does. The CLI then prints that a link is on its way.

**PARA-11 (Low)** `SKILL.md:65`: `projects create <ws-id> --title T` exits 2 because flags must precede the id. `cli_idea.go:16` lists two of five sub-verbs, `help.go:49` names only `INSIDEOUT_TOKEN`, and `docs/architecture/agent-token-scope.md:57` counts 72 routes where the table has 74.

**PARA-12 (Low)** `server/internal/mcpserver/product_tools.go:243`: `ttl_hours` is declared as a string and read as a number at line 246, so a schema-conforming value is ignored.

## Lead verification

| Lead | Verdict | The line that decided it |
| --- | --- | --- |
| 7 | Confirmed, with root causes for two of the three failures | `cli_auth.go:55` (null stdin read as a terminal) and `escape_test.go:257: Client.ResetWatchStatus has no row in the escape sweep` |
| 13 | Partly refuted | 89 distinct `HandleFunc("METHOD /path"` literals plus the webhook constant, not about 107; 52 MCP tools confirmed (37 product, 4 draft, 9 auth, 2 admin); 41 rows in the CLI verb table |
| 14 | Checked | None of PARA-01 to PARA-12 is open in `docs/issues/2026-10-18-review-4-open-findings.md`; account deletion is recorded there as decided (rv4-PUB-02), which covers the route, not its surfaces |
| 5, 8, 12 | Not touched | Outside this lens or not re-run |

## Recommendations

1. Land PARA-01 and PARA-08 so the reset-watch work can be committed green.
2. Adopt one PATCH rule for this domain: an omitted key keeps the stored value.
3. Write down which identity capabilities are web-only on purpose, and make `SKILL.md` agree.
4. Register only MCP tools that can succeed, and name only listed tools in MCP hints.
5. Show the mail status word on every surface that requests an emailed code or link.
6. Give account deletion a control on the profile page.

Not checked: live behaviour of the web pages, the real-database integration tests, the setup wizard internals, and whether the reset-watch route is rate limited.

## What this role would do first

Change the terminal check at `server/cmd/insideout/cli_auth.go:55` to `term.IsTerminal(int(os.Stdin.Fd()))` and rerun `go test ./cmd/insideout`. It is a one-line change that removes a silent no-op on the path agents and scripts use, and it turns one of the three red tests green.

## Surface matrix

| Capability | REST | Web | MCP | CLI | Note |
| --- | --- | --- | --- | --- | --- |
| Register | POST /api/v1/auth/register | register via session.register, features/auth/auth_form.dart:123 | auth_register (removed once a login is held) | register [--invite code] <email> [display-name] | MCP withholding is deliberate (agent-token-scope.md: no password in the conversation). CLI defaults username to the email local part; REST and MCP require it. |
| Password login | POST /api/v1/auth/login | login via session.login, features/auth/auth_form.dart:131 | auth_login (removed once a login is held) | login <email> | Consistent bodies. Wrong credentials answer 401 `invalid email or password` everywhere. |
| Login code request | POST /api/v1/auth/login-code | requestLoginCode, features/auth/auth_form.dart:117 | auth_login_code_request | login --code <email> (first half) | The `mail` status word is ignored on all three; only the setup wizard prints it (PARA-05). |
| Login code verify | POST /api/v1/auth/login-code/verify | session.verifyLoginCode, features/auth/auth_form.dart:187 | auth_login_code_verify (hidden from tools/list while a login is held) | login --code <email> (second half) | MCP hints still name the hidden tool (PARA-04). |
| Magic link request | POST /api/v1/auth/magic-link | requestMagicLink, features/auth/auth_form.dart:114 | auth_magic_link_request | login --magic <email> (first half) | Same mail-status gap as the code request. |
| Magic link consume | POST /api/v1/auth/magic-link/consume | session.consumeMagicLink, features/auth/magic_link_page.dart:39 | auth_magic_link_consume (hidden while a login is held) | login --magic <email> (paste link or token) | Consistent. |
| Refresh session | POST /api/v1/auth/refresh | interceptor, api/client_core.dart:127 | none as a tool; in-process renewal | none as a verb; automatic in apiclient/refresh.go | Deliberate: renewal is automatic on every surface. |
| Logout | POST /api/v1/auth/logout | logout via session.logout, app.dart:170 | auth_logout (hidden while a login is held) | logout | Behaviour differs: web and CLI revoke the server session; the MCP tool only clears process memory (auth_tools.go:166) and never calls the route. Documented in cli-mcp-and-guidance.md:195. |
| Forgot password | POST /api/v1/auth/forgot-password | forgotPassword, features/auth/forgot_password_page.dart:50 | auth_forgot_password (removed once a login is held) | forgot-password <email> | Only the CLI confirms the address, prints the mail status and waits on the watch. Null stdin breaks it (PARA-01). |
| Reset watch | GET /api/v1/auth/reset-watch/{id} | none | none | none as a verb; polled inside forgot-password (cli_reset_watch.go) and the setup wizard | Changelog 2026-10-21 names only the CLI. Web absence is reasonable (the link lands there); MCP absence is not recorded either way. |
| Reset password | POST /api/v1/auth/reset-password | resetPassword, features/auth/reset_password_page.dart:64 | auth_reset_password (removed once a login is held) | reset-password | Consistent; 401 `invalid or expired reset link` on all. |
| Read signup mode | GET /api/v1/auth/config | authConfig, features/auth/auth_form.dart:66 and features/profile/admin_section.dart:50 | signup_mode with no argument (removed once a login is held) | signup-mode | A signed-in MCP cannot read this public value because the whole tool is withheld (PARA-07). |
| Who am I | GET /api/v1/me | me() in session | whoami | whoami | Consistent raw JSON. |
| Edit profile | PATCH /api/v1/me | updateMe, features/profile/profile_page.dart:59 | none | none | MCP: deliberate (agent_scope.go:113 humanOnly). CLI: accidental, no client method. Locale is not a server-side profile field; it is a web-only setting and a convert-time argument. |
| Delete account | DELETE /api/v1/me | none | none | none | MCP deliberate (me_delete.go:19). Web and CLI absence unrecorded; PRIVACY.md:110 describes the raw request (PARA-03). |
| Mint agent token | POST /api/v1/agent/token | none | none as a tool; traded automatically on every call (apiclient.AgentScoped) | none | Web: deliberate. CLI: no verb, although agent-token-scope.md:103 describes handing an agent token to a process as INSIDEOUT_TOKEN. |
| Set signup mode | PUT /api/v1/admin/settings | setSignupMode, features/profile/admin_section.dart:69 | signup_mode with mode (never usable) | signup-mode <mode> | GET /api/v1/admin/settings has no caller on any surface. |
| Create invitation | POST /api/v1/admin/invitations | createInvitation(email), features/profile/admin_section.dart:82 | invite_create (never usable) | invite [--ttl hours] <email> | Web cannot set a lifetime; MCP ttl_hours is mistyped (PARA-12). |
| List invitations | GET /api/v1/admin/invitations | listInvitations, features/profile/admin_section.dart:51 | invite_list (never usable) | invites | Capped at 200 rows with no pagination (store/invitations.go:119). |
| Revoke invitation | POST /api/v1/admin/invitations/{id}/revoke | revokeInvitation, features/profile/admin_section.dart:140 | invite_revoke (never usable) | invite revoke <id> | Consistent on web and CLI. |
| Mail probe | POST /api/v1/admin/mail/probe | none | none | none | Operator route, described for manual use in docs/usage/keys/mail; deliberate. |
| Promote admin | none | none | none | promote-admin <email> (server operator, owner database connection) | Deliberately out-of-band (cli-mcp-and-guidance.md:184). |
| List workspaces | GET /api/v1/workspaces | listWorkspaces (dashboard) | workspaces | workspaces | Ordered by updated_at descending; no pagination. |
| Create workspace | POST /api/v1/workspaces | createWorkspace, features/dashboard/dashboard_page.dart:59 | workspace_create | workspaces create --title T [--desc D] | Consistent; MCP uses `description`, CLI flag is `--desc`. |
| Join workspace | POST /api/v1/workspaces/join | joinWorkspace, features/dashboard/dashboard_page.dart:84 | workspace_join | join <code> | Consistent. |
| Get workspace | GET /api/v1/workspaces/{id} | getWorkspace, features/workspace/workspace_page.dart:69 | none | none | humanOnly (agent_scope.go:121); the list answer carries the same fields. |
| Update workspace | PATCH /api/v1/workspaces/{id} | updateWorkspace, features/workspace/settings_page.dart:86 | none | none | MCP deliberate; CLI accidental (PARA-06). Web save clears coverUrl (PARA-09). |
| Delete workspace | DELETE /api/v1/workspaces/{id} | deleteWorkspace, features/workspace/settings_page.dart:107 | none | none | MCP deliberate. SKILL.md:120 wrongly says the CLI can. |
| List members | GET /api/v1/workspaces/{id}/members | listMembers, features/workspace/settings_page.dart:74 | none | none | humanOnly (agent_scope.go:122); CLI absence unrecorded. |
| Change member role | PATCH /api/v1/workspaces/{id}/members/{userId} | updateMemberRole, features/workspace/settings_page.dart:118 | none | none | MCP deliberate; CLI accidental. |
| Remove member or leave | DELETE /api/v1/workspaces/{id}/members/{userId} | removeMember, features/workspace/settings_page.dart:117 | none | none | MCP deliberate; SKILL.md:120 wrongly says the CLI can. |
| List ideas | GET /api/v1/workspaces/{id}/ideas | listIdeas, features/workspace/ideas_page.dart:45 | idea_list | ideas <workspace-id> | Author's own ideas only, created_at descending, no pagination. |
| Create idea | POST /api/v1/workspaces/{id}/ideas | createIdea, features/workspace/ideas_page.dart:87 (title only) | idea_create | idea create --title T [--content C] <workspace-id> | Title limit 200 on all; web captures the body in a later edit. |
| Get one idea | GET /api/v1/ideas/{iid} | getIdea in api/client_workspace.dart:101, no page calls it | none | none | humanOnly (agent_scope.go:112); the list answer carries full content. |
| Update idea | PATCH /api/v1/ideas/{iid} | updateIdea, features/workspace/ideas_page.dart:135 (always sends both fields) | idea_update | idea update --title T [--content C] <id> | Omitting content on CLI or MCP erases the body (PARA-02). |
| Delete idea | DELETE /api/v1/ideas/{iid} | dropIdea, features/workspace/ideas_page.dart:108 | idea_delete (never usable) | idea delete <id> | Soft drop on all. |
| Convert idea to PRD | POST /api/v1/ideas/{iid}/convert | convertIdea with the app locale, features/workspace/ideas_page.dart:163 | idea_convert (locale) | idea convert [--locale en\|zh] <id> | Default differs: REST treats a missing or unknown locale as zh; CLI and MCP default to en from the environment and reject other tags (documented in apiclient/prd.go:39). |

## Skeptic verdicts on this lens's Highs

| ID | Verdict | Reason | What the skeptic ran |
| --- | --- | --- | --- |
| PARA-01 | DOWNGRADED to Medium | The defect is real and reproduced, and it is the cause of the red test: with the one-line fix applied through a build overlay the test passes. It is less than High on three counts. The documented path is a person at a terminal, where the prompt works. The failure needs a detached stdin that is a character device (/dev/null); a piped stdin sends the request. The command prints `Email not sent.` and nothing is lost, since a reset can be asked for again. The red test is in the uncommitted working tree, not on the pushed tip, so the red-gate-on-pushed-tip criterion is not met yet. It becomes High the moment this tree is committed without the fix. | ./insideout forgot-password --api http://127.0.0.1:9/api/v1 a@example.org </dev/null -> `Send email to a@example.org? [y/N] Email not sent.` exit=0; with `printf '' \|` -> connection refused, exit=1; go test ./cmd/insideout -run TestEveryCLIClientEndsWithTheCommandContext -> FAIL `err = <nil>, want the cancelled context's error`; same test with -overlay swapping in term.IsTerminal -> ok |
| PARA-02 | CONFIRMED | Reproduced on the real database through the Go client method that both the CLI verb and the MCP tool call. An idea with a body was updated with a new title and no content; the stored content came back empty. The raw API does the same when the content key is left out, so the loss is in the server rule as well as the client. The CLI usage prints `[--content C]`, the MCP tool declares content optional, and SKILL.md:68 sends agents to `idea_update` to sharpen an idea. The idea row has no revision history, so the body is not recoverable. The web form always sends both fields, so only the CLI and MCP lose data. Not an open record under docs/issues. | go test ./internal/api/ -overlay /tmp/insideout-rv5/client-and-cli/overlay.json -run TestRV5ProbeIdeaUpdateTitleOnly -v -> `before content="the body the user typed"; after title="Sharper title" content=""`; `PATCH {title only, content key omitted} -> 200 ... "content":""` |

## Ordered recommendations

1. Land PARA-01 and PARA-08 first so the uncommitted reset-watch work can be committed with a green `go test ./...`; together they are about ten lines.
2. Adopt one rule for PATCH in this domain: an omitted key keeps the stored value. Commit 269391f already did this for a project's ownerId; apply it to idea content, avatarUrl and coverUrl (PARA-02, PARA-09).
3. Decide in one line of docs/usage/cli-mcp-and-guidance.md which identity capabilities are web-only on purpose (workspace settings, members, profile, account deletion), then make SKILL.md say the same instead of claiming 1:1 parity.
4. Shrink the MCP registry to tools that can succeed (PARA-07) and make every MCP hint name only tools that tools/list currently offers (PARA-04).
5. Carry the `mail` status word to every surface that requests an emailed code or link, or remove it; a status that only one verb prints is worse than none (PARA-05).
6. Add a profile-page control for account deletion so PRIVACY.md no longer has to describe an HTTP request (PARA-03).
