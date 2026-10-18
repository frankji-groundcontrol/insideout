# 14 · User

**Lens:** role  
**Score:** **5.8 / 10**  
**Bottom line:** The core loop (idea, convert, PRD, commit, readiness) runs clean and the client UI is honest and bilingual, but the front door is broken: a self-hoster cannot create the first account as documented, a binary-only user's first command (help, version, a typo) fails with a server-config error, plain login and the setup wizard's saved token do not keep you signed in, and setup --check can never fail on the API.

Tree: `main` @ `9f0b1c7`. Every High this lens filed went to an adversarial skeptic cluster; where the skeptic changed a severity or corrected a claim it is marked **[Correction]** below. The full register is [01-findings-register.md](01-findings-register.md).

## Skeptic verdicts on this lens's Highs

| ID | Filed | Verdict | Final | Reason |
|---|---|---|---|---|
| USER-01 | High | CONFIRMED | High | I reproduced the whole chain on a freshly migrated database with 0 users and signup_mode='invitation_only'. (1) `insideout promote-admin you@example.com` logged level=ERROR msg=promote-admin error='not found' and exited 1. SetSystemRole is a bare `UPDATE insideout.users SET system_role ... WHERE email = $1`, and zero rows … |
| USER-02 | High | DOWNGRADED | Medium | **[Correction]** The claim is true as stated. I built the binary from this tree and ran it with a clean env (no DATABASE_URL). All seven invocations (--help, -h, help, version, --version, a mistyped verb, and bare) printed {"msg":"config","error":"config: DATABASE_URL is required"} and exited 1. runClientCommand in cli.go returns (false,0) for … |
| USER-03 | High | DOWNGRADED | Medium | **[Correction]** Reproduced live with the built CLI against the real server on my private Postgres (port 55461, signup opened on that private DB only): `register`, then `login` with the password on stdin. login printed the JWT on stdout and wrote nothing, and no insideout config.json exists under the sandbox HOME. `whoami` right after failed … |
| USER-04 | High | CONFIRMED | High | Reproduced end to end against the built server and CLI. A login on the default-TTL server returned both accessToken and refreshToken, and the decoded access token had exp-iat of 900 seconds, matching the 15m default in config.go:89 and docker-compose.yml:89. flow.Run saves only clientconfig.Config{APIBase, Token: … |
| USER-05 | High | DOWNGRADED | Medium | **[Correction]** The behaviour is confirmed live with the built binary, INSIDEOUT_ENV_FILE pointed at an empty file and a sandbox HOME. With INSIDEOUT_API at a refused port, `setup --check` printed 'unreachable (offline?) — not failed', then 'all configured checks passed', exit 0, and the same happened with a bogus INSIDEOUT_TOKEN set. Against … |
| USER-06 | High | DOWNGRADED | Medium | **[Correction]** The claim holds. runCrudCommand calls fs.Parse(args), which stops at the first non-flag token, so `<id> --section ...` leaves three positionals and need(1,...) fails. I built the CLI from a scratch clone and ran both orders with INSIDEOUT_API pointing at a dead localhost port. README order: prints `usage: insideout prd update … |
| USER-07 | High | DOWNGRADED | Medium | **[Correction]** The claim holds exactly as cited. admin.go:133 only logs the send error. admin.go:136-140 then returns 201 with only {invitation, code, signupUrl}. A scratch DB-backed test with mail in dev mode (no API key; Send logs and returns nil) got status 201 and the response keys signupUrl,code,invitation, with no mail-status flag. The … |

---

## Scope

Review 4, role **user**: a first-time user and an invited collaborator, on `main @ 9f0b1c7` in the clean worktree. I read the README quick start, `scripts/bootstrap.sh`, `docs/usage/installation.md` and `local-development.md`, the CLI entry (`server/cmd/insideout`), the setup probes, and the Flutter auth, coach, admin and project screens with both string tables. I also drove the real thing: a private Postgres 18.4 cluster with CI-parity roles, a freshly built `insideout` and `insideout-mcp`, a server on localhost, a sandboxed `HOME`. I walked: migrate, seed, promote-admin, login, invite, invited register, workspace, project, idea, convert to PRD, coach turn (offline and with a dead provider), commit, view, readiness, `setup --check`, the MCP unauthenticated call. The cluster and servers are stopped. No network beyond localhost, no `.env` read.

## Scores

| Aspect | Score | Why |
| --- | --- | --- |
| Install (binary + PATH) | 7.5 | `bootstrap.sh` fails closed on checksum; idempotent PATH edit. Banner over-promises `setup --local`. |
| First run, hosted | 5.0 | Works only with an invitation; no stated way to get one; the CLI has no help; `login` does not stay logged in. |
| First run, self-host | 4.0 | The documented bootstrap (`promote-admin`) cannot work on a fresh database. |
| Invited collaborator | 5.5 | Invite and register work live. The admin is told "emailed" when it may not be; two different "invite codes". |
| Core loop (capture, convert, commit, view) | 8.0 | Ran clean end to end; JSON is the API's own; commit lists the unresolved sections honestly. |
| Offline / error honesty | 6.0 | The UI discloses echo mode well; docs call it "canned replies... stages, tools". `setup --check` cannot fail. |
| i18n (en/zh) | 8.5 | Key sets match in all four tables (92/106/52/140); no literal English strings found in `features/`. |
| **Overall** | **5.8** | Works for someone who already knows the verbs and holds a token; the front door is where it breaks. |

## What is good

- `scripts/bootstrap.sh:60-80` refuses to install on a missing or mismatched checksum line and cleans up via `trap`; `:123-132` prints the next step.
- `client/lib/features/prd/coach_panel.dart:212-220` plus `client/lib/api/sse.dart:228` show a banner when the stream says `offline:true`, and `client/lib/l10n/en_prd.dart:96` calls the replies "echoes, not AI answers". That is the honest sentence; the docs should copy it.
- `client/lib/l10n/` keeps en and zh in lockstep (I diffed the key sets), and `features/` has no hard-coded English labels.
- `server/cmd/insideout/cli_auth.go` (header) keeps stdout token-only and prompts on stderr, so `INSIDEOUT_TOKEN="$(insideout login …)"` is a real contract; usage strings name every required flag.
- The product loop ran first time: `idea create` -> `idea convert` -> `prd update` -> `commit` -> `readiness`. `commit` returned the carried-open sections (`background`, `risks`) instead of pretending the PRD was done.
- The MCP server returns `isError:true` with a readable string when unauthenticated, and lists 52 tools.

## Findings (worst first)

**USER-01 High, self-host: the first account cannot be made as documented.** Registration returns 403 without an invite (`server/internal/api/auth.go:95`). `signup-mode open` needs an admin token. `promote-admin` only updates an existing row: `server/internal/store/invitations.go:193: return ErrNotFound`. I ran it on a fresh database: `promote-admin` logged `"error":"not found"`. README:182, installation.md:132-137 and `cli_setup_local_boot.go:40` ("First admin: ... promote-admin you@example.com") all send a new user there. Only `seed` (which creates `demo@insideout.local` with a published password) breaks the loop, and no first-run path mentions it. Fix: make `promote-admin` upsert a user when none exists (prompt for the password), or auto-promote the first registrant when `users` is empty.

**USER-02 High: `insideout --help`, `help`, `-h`, `version` and any mistyped verb print `config: DATABASE_URL is required`.** `server/cmd/insideout/main.go:42: cfg, err := config.Load()` runs after `runClientCommand` declines the argument. Confirmed with five arguments on the installed binary; a bare `insideout` does the same. A binary-only user's first command ends in a server-config error. Fix: before `config.Load()`, handle help/version/unknown verbs (print the verb list from `cli.go`'s switch, exit 2), and only fall through to server mode for the server subcommands.

**USER-03 High: plain `login` leaves you logged out.** `loginCommand` prints the token and saves nothing; `installation.md:56-57` lists `insideout login you@example.com` then `insideout whoami` as one sequence. Run live: `whoami` -> `unauthorized (check INSIDEOUT_TOKEN or login)`, pointing back at the step just done. `installation.md:60` also promises `authentication required`, which is not the message. Fix: `login` writes the token to the 0600 clientconfig (as `setup` does) and prints one stderr line saying so; fix the doc string.

**USER-04 High: a saved token dies after 15 minutes and nothing renews it.** Access tokens are 900 s (decoded `exp - iat`; default `INSIDEOUT_ACCESS_TTL` in environment.md:99). `server/internal/apiclient/apiclient.go:39: the refresh token stays unused by`. The wizard's "no export lines to copy" token (installation.md:48) therefore works for one coffee, then every verb fails with the env-var hint, though the user's token lives in a file. Fix: store the refresh token in clientconfig and refresh once on 401; failing that, the 401 text must say "token expired, run `insideout login`".

**USER-05 High: `insideout setup --check` cannot fail on the thing a hosted user has.** The probe sends no credentials: `server/internal/setup/probe/probe.go:67: ...+"/me", nil)`, so it always sees 401 and prints "authentication required (as expected)" even when signed in or expired. A refused connection is a skip: with `INSIDEOUT_API=http://127.0.0.1:1/api/v1` it printed "unreachable (offline?) — not failed", "all configured checks passed", exit 0. The file header (`cli_setup_check.go:3-7`) says it uses the saved token and fails on a rejected one; it does not. Fix: send the clientconfig token and report signed-in / expired; treat an unreachable explicitly configured base as fail, with a `--offline` flag to skip.

**USER-06 High: the README's `prd update` example prints usage only.** `README.md:170: insideout prd update <prd-id> --section goals="Ship the MVP"`. Go's `flag` stops at the first positional, so the flag is never parsed. Confirmed: this order gives the usage line; flags-first works. Fix: put the id last in the README (one line), or reorder args in `runCrudCommand`.

**USER-07 High, collaborator: the admin is told the invite was emailed when it may not have been.** `server/internal/api/admin.go:133: s.log.Error("admin: send invitation", "error", err)` swallows the failure and returns 201; with no mail key it is a dev-mode no-op. The dialog always says `client/lib/l10n/en_account.dart:83: 'admin.codeOnce': 'This code is shown once and was emailed to the invitee:'` (the client already knows `mailEnabled`). The admin walks away, the collaborator never gets anything. Fix: return `mailSent: bool` in the response and show "copy this link and send it yourself" when false.

**USER-08 Medium: the docs describe offline mode as something it is not.** `docs/usage/local-development.md:66: end-to-end (stages, tools, SSE) with canned replies`. `server/internal/agent/template.go` returns the user's own last message and empty tool calls; I confirmed over SSE (`"text":"I want a habit tracker..."`). Stages never move and no section ever fills, yet the same doc recommends it as the local default. The banner only appears after the first message. Fix: reword the docs to "echo", show the banner before the first send, and say that coaching needs `INSIDEOUT_LLM_API_KEY`.

**USER-09 Medium: the binary installer advertises a mode the binary cannot run.** `scripts/bootstrap.sh:127: insideout setup --local --postgres bundled|own   # own server + PostgreSQL`. On a machine with no checkout it exits 2: `no repository checkout found above ...` (`cli_setup_local_env.go:33`). Fix: add "(needs a repository checkout)" to the banner and to the README line at 74-77.

**USER-10 Medium: an invited collaborator needs two codes, and both are called "invite code".** The admin invitation is email-bound and carries no workspace (`admin.go:99-110` takes only an email). After registering, the new user lands on "You haven't joined any workspace yet... enter an invite code" (`en_workspace.dart:16`), a different 32-hex workspace code (`:13 'Invite code'`). Fix: rename the workspace one "Workspace code" in both languages, and let the invite optionally carry a workspace so the first screen is the workspace.

**USER-11 Low: the Edit-project description field is labelled "What's new?"** `client/lib/features/project/project_dialogs.dart:38: label: t('project.updatePlaceholder'),` (the status-post label, reused at `:122`). Add a `project.descPlaceholder` key.

**USER-12 Low: a hosted newcomer is stuck at the door.** Every 403 on register shows `register.inviteError` ("missing, invalid or expired"), even when registration is closed (`auth_form.dart:136`). The README and installation.md say to "use an invite code" and never say who to ask; `README.md:44` says "open your deployment's URL" and gives none (it is in `deployment.md:172` only). Fix: one sentence naming the contact path, and a distinct string for "registration is closed".

## Lead verification

| Lead | Result | Evidence |
| --- | --- | --- |
| L1 siblings | Extended: no new siblings | Grep of every `time.Date(2026...)` in `server/**/*_test.go` and the client tests: all pin `now` or use relative `DateTime.now()` offsets (`project_updates_display_test.dart:12`); only `projects_test.go:51` mixes a pinned clock with `time.Now()` in the code under test. |
| L3 / L4 | Not re-run (outside my lens) | Core loop and CLI/MCP runs on the same cluster raised no failures. |
| L10 | Confirmed from the user seat | `bootstrap.sh:60-80` verifies against the same-origin `checksums.txt`; it protects against corruption, not a compromised release. |
| L15 | Confirmed | `.env.example` carries a placeholder; nothing secret reached the first-run path. |
| L18 | Not in lens | |

## Recommendations

1. Make the first account creatable: `promote-admin` upserts, or the first registrant on an empty `users` table becomes admin (USER-01).
2. Add `help`, `-h`, `--help`, `version` and unknown-verb handling to the client dispatcher before `config.Load()` (USER-02).
3. Make `login` persist its token to clientconfig and print where it saved it; fix the `whoami` doc text (USER-03).
4. Keep sessions alive: save and use the refresh token, or have the 401 message say "token expired, run `insideout login`" (USER-04).
5. Make `setup --check` send the saved token and fail on an unreachable configured API (USER-05).
6. Return `mailSent` from the invite endpoint and change the admin dialog to match (USER-07).
7. Fix the README `prd update` example; rewrite the offline-coach sentences as "echo"; add the checkout caveat to the bootstrap banner (USER-06, 08, 09).
8. Rename the two "invite code" concepts and add a project description label (USER-10, 11).

## The first three minutes, rewritten

**Minute 0:00, install.** `curl ... | bash` prints the same output, then its last lines say: "Next: `insideout login <email>` (you need an invitation; ask the person who runs your instance). Self-hosting? That needs a repository checkout: `git clone ... && ./scripts/setup.sh`."

**Minute 0:45, first command.** `insideout` with no arguments prints a 12-line verb list grouped as sign-in, ideas, PRDs, roadmap, commit, agent, admin, with the server subcommands marked "server only". `insideout version` prints a version. A mistyped verb says `unknown command "idas"` and suggests `ideas`.

**Minute 1:15, sign in.** `insideout login me@example.test` prompts for the password and prints "signed in as me; token saved to `<config path>`; it renews itself". `insideout whoami` then answers with the account. If the invite is missing, the 403 reads "an invitation is required: ask your instance admin for a link".

**Minute 2:00, self-host variant.** `insideout setup --local --postgres bundled` ends with "Create your admin: `insideout promote-admin you@example.com` (creates the account, asks for a password)". The next command works. `setup --check` then says "signed in as you; API ok; database ok; coach: offline (echo only, set `INSIDEOUT_LLM_API_KEY` for AI coaching)".

**Minute 2:30, the loop.** `insideout idea create --title "Habit tracker" <ws>`, `insideout idea convert <id>`, and `insideout prd update --section goals="Ship the MVP" <id>` are the three lines in the README, in the order that works. An invited collaborator opens the emailed link (or the one the admin copied, since the dialog said "not emailed"), registers, and lands in the inviter's workspace instead of an empty screen asking for a second "invite code".

## What this role would do first

I would fix USER-01 and USER-02 in one sitting, because they are what a newcomer hits before anything else. USER-01 is about 25 lines in `runPromoteAdmin` plus `store.SetSystemRole` (an upsert path with a password prompt); USER-02 is a 30-line pre-dispatch block in `main.go`. Then I would close the session gap, USER-03 and USER-04 together (login writes the token and refresh token; the client refreshes on 401), since a CLI that forgets you after 15 minutes makes every later fix look broken. After that comes the invite honesty fix, USER-07, which is eight lines and stops the quiet failure for every collaborator. Everything else is copy and docs.
