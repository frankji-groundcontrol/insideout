# Wave 3, first-run copy — review-4 remediation

Branch `rv4/w3-first-run-copy`, `41511f9..3cb80a1` (eleven commits: five red
tests and six fixes, the last of them a repair of this cluster's own CLOSE-02
fix), then this fragment. Row 3.8 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md)
plus the `LEAD-SETUP-BLURB` lead; findings `rv4-USER-10`, `rv4-USER-12`,
`rv4-CLOSE-02`, `rv4-FEAT-05` (convert half only), `rv4-PM-08` (code half only)
and `rv4-LEAD-SETUP-BLURB`. Findings are cited as `rv4-<ID>` because review IDs
restart per review. No secret, invitation code, personal address or hosted API
base is spelled in this fragment (the base is `<hosted-api-base>`, the release
download base `<release-base>`).

| Finding | State | Red | Fix |
| --- | --- | --- | --- |
| `rv4-USER-10` | fixed | `c08ab68` | `31e8fa7` |
| `rv4-USER-12` | partial: the string is fixed, the contact route is the owner's | `1c6b552` | `9fe4b8a` |
| `rv4-CLOSE-02` | fixed | `7bae58f` | `e0a53b8`, repair `3cb80a1` |
| `rv4-FEAT-05` | partial by design: convert half fixed with CLOSE-02, export-heading half open | `7bae58f` | `e0a53b8`, `3cb80a1` |
| `rv4-PM-08` | fixed for the code half (README hunks belong to `w3-docs-truth`) | `dd14a30` | `48164b2` |
| `rv4-LEAD-SETUP-BLURB` | fixed | `16006a5` | `7d5670e` |

## What changed

- **Two codes, two names** (rv4-USER-10, red `c08ab68`, fix `31e8fa7`). An
  invited collaborator needs two unrelated codes and both were called "invite
  code" (邀请码 in Chinese): the email-bound invitation that lets an account be
  created, and the workspace code that lets a signed-in account into a team.
  The workspace one is now "Workspace code" / 工作区代码 in the join dialog (its
  title and field), on the workspace page (the code and the hint above it) and
  in the empty-dashboard prompt. The account invitation keeps "Invitation code"
  / 邀请码 in the register field, its error and the admin dialog. The admin
  dialog's line about the code it created said "This code is shown once" and, in
  Chinese, 验证码 (a verification code, a third and wrong word); it now says
  "This invitation code" / 邀请码. Catalog keys are internal and unchanged. The
  new `invite_wording_test.dart` pins one name per code per language and refuses
  the invitation's word in every `workspace.*` string except
  `workspace.inviteAccountHint`, which is about the invitation on purpose
  (it tells a teammate to get an account first).
- **A closed server says so** (rv4-USER-12, red `1c6b552`, fix `9fe4b8a`). The
  register endpoint answers 403 both when registration is closed and when the
  invitation is missing, wrong or expired, and the body carries no code to tell
  them apart. Every 403 showed the invitation copy, so a visitor to a closed
  server was told to check a code that cannot help. The form already fetches
  the sign-up mode from `/auth/config`; `auth_form.dart` now keeps it
  (`signupMode`) and the new `auth_form_errors.dart` maps `registerErrorKey`:
  a 403 on a `closed` server is `register.closedError`, any other 403 stays
  `register.inviteError` (including while the mode is still unknown, so a slow
  or failed config fetch never claims the door is closed), and everything else
  stays `register.error`. The decision follows the structured mode, not the
  English text of the 403. New copy: "Registration is closed on this server.
  Ask the server's administrator to open sign-up." / 此服务器已关闭注册。请联系服务器管理员开放注册。
- **`idea convert` speaks a language** (rv4-CLOSE-02 and the convert half of
  rv4-FEAT-05, red `7bae58f`, fix `e0a53b8`, repair `3cb80a1`). `insideout idea
  convert` and the MCP tool `idea_convert` sent an empty body, so every idea they
  converted opened a Chinese coach conversation (the server's default), however
  it was written; the web client already sent the app locale. Now the CLI has
  `--locale en|zh` and the tool has a `locale` argument; without one the
  process language decides (`LC_ALL`, `LC_MESSAGES`, `LANG`: the first that is
  set; Chinese is Chinese, anything else or nothing is English). The body always
  carries the answer, `{"locale":"en"}` or `{"locale":"zh"}`. A value other than
  en or zh (a tag such as `en-US` or `zh_CN.UTF-8` is accepted and narrowed) is
  a usage error before anything is sent: exit 2 on the CLI, a tool error over
  MCP. The server turns every tag that is not English into zh, so a typo such as
  `fr` would otherwise open a Chinese conversation without a word of warning.
  `apiclient.ConvertLocale` holds the one rule for both front doors, and the
  tool's schema names en and zh in the `locale` description, since an agent
  learns the argument from `tools/list` alone.
- **The installers say who they are for** (rv4-PM-08 code half, red `dd14a30`,
  fix `48164b2`). `bootstrap.sh` exits on any system but macOS and Linux and
  nothing said so up front; `install.sh` told a stranger with no Go toolchain
  to go and install one. Both scripts now open by naming macOS and Linux as the
  supported systems and Windows as not supported, and `install.sh`'s "no Go"
  error offers the prebuilt installer first (macOS and Linux, arm64 and
  amd64, with the one-liner), then the from-source route ("install Go 1.26+",
  the wording `check_go_floor.py` reads).
- **The hosted blurb teaches the saved login** (LEAD-SETUP-BLURB, red `16006a5`,
  fix `7d5670e`). `scripts/setup.sh --hosted` taught
  `export INSIDEOUT_TOKEN="$(insideout login ...)"`: an access token that
  nothing renews, although `insideout login` already saves a login that does
  (it carries the refresh token, and the CLI and `insideout-mcp` both read it).
  The blurb now teaches `insideout login you@example.com` then `insideout
  whoami`, gives the sign-up route (an invitation code from the server's
  administrator, then `insideout register --invite <invitation-code>
  you@example.com`), and mentions `INSIDEOUT_TOKEN` last, as the headless or CI
  alternative that wins over the saved login and is never renewed. The closing
  messages of `install.sh` and `bootstrap.sh` carry the same sign-up route.
- **`scripts/tests/check_first_run_copy.py`** (new, six checks) pins the last two
  items. It runs the scripts where they stop before doing any work
  (`install.sh` at its Go check with no `go` on `PATH`, `bootstrap.sh` at its OS
  check with a fake `uname`, `setup.sh --hosted` before it reads any env file),
  on a scratch `PATH` and `HOME`, and derives the prebuilt one-liner from
  `bootstrap.sh` itself. It also reads the CLI's verb table, so a script that
  quotes `insideout <verb>` for a verb that does not exist, or `--invite` after
  `register` stopped listing it, fails. CI runs it with the other
  `check_*.py` guards by glob.

## Evidence

Under `/private/tmp/insideout-rv4/reports/w3-first-run-copy-evidence/`: a red
and a green file per finding, each red re-run in a scratch `git archive` of the
red commit (`close02-red.txt`, `user10-red.txt`, `user12-red.txt`,
`pm08-red.txt`, `blurb-red.txt`, `close02-repair-red.txt`) and each green run at
the branch head (`close02-green.txt`, `user10-green.txt`, `user12-green.txt`,
`first-run-copy-green.txt`). The CLOSE-02 runs used a private PostgreSQL
cluster; a skipped test counts as red, and the green file has no SKIP line.

| Finding | Red (at the red commit) | Green (at the head) |
| --- | --- | --- |
| rv4-USER-10 | `+0 -6: Some tests failed.`; `workspace.joinCodePlaceholder is a workspace string: Invite code`; `Expected: contains '工作区代码' Actual: '邀请码'`; `admin.codeOnce` lacks "invitation code" / 邀请码 | `+6: All tests passed!` |
| rv4-USER-12 | `+6 -2: Some tests failed.`; `Found 0 widgets with text "Registration is closed on this server. Ask ...` and the same for 此服务器已关闭注册。请联系服务器管理员开放注册。 (the three control cases pass on the base) | closed test `+8`; with `auth_form_register_test.dart` and `l10n_test.dart` `+16: All tests passed!` |
| rv4-CLOSE-02 | MCP: `the API saw ["POST /ideas/i1/convert {}"], want exactly [... {"locale":"en"}]` (all four cases), `idea_convert declares no locale argument`; CLI: `flag provided but not defined: -locale`, and the no-flag cases send `{}`; `apiclient` does not compile (`undefined: ConvertLocale`) | `TestIdeaConvertSendsTheLocale`, `TestIdeaConvertRefusesALocaleItCannotCoachIn`, `TestIdeaConvertStoresTheLocaleOverTheRealServer` (real PostgreSQL), the three MCP tests and the three `apiclient` tests PASS; `ok` for all three packages |
| rv4-CLOSE-02 repair | at `e0a53b8`: `escape_test.go:257: Client.IdeaConvertWithLocale has no row in the escape sweep: add one` | `TestSweepCoversEveryClientMethod` and `TestHostileValuesStayOneSegmentOrOneQueryValue` PASS |
| rv4-PM-08 | five messages: `scripts/bootstrap.sh: the opening comment does not say macOS and Linux are the supported systems`, `... does not say Windows is not supported` (also for `install.sh`), `scripts/install.sh with no go on PATH does not print the prebuilt one-liner` | `first-run copy ok: 6 checks` |
| LEAD-SETUP-BLURB | eight messages: `scripts/setup.sh --hosted teaches an exported INSIDEOUT_TOKEN from \`insideout login\` as the sign-in; the saved login renews, an exported token does not`, `... does not give the sign-up route ...` (three scripts), `... does not say an invitation code comes from the server's administrator` (three scripts) | `first-run copy ok: 6 checks` |

Cluster gates at the head: `go build ./...` and `go vet` on the three package
trees; `go test ./internal/apiclient/... ./internal/mcpserver/... ./cmd/...
-count=1 -race` (`apiclient` 4.6 s, `mcpserver` 1.8 s, `cmd/insideout` 239 s);
`flutter analyze --no-fatal-infos` (14 infos, all in `lib/api/`, the CI
baseline; none in a touched file); `flutter test` and `TZ=Asia/Shanghai flutter
test` (497 tests each); every `scripts/tests/check_*.py` guard;
`check-redaction.sh`; `gofmt`; the 350-line gate.

### Live proof

Built binaries of the head, a real server on loopback against the private
database (sign-up mode `invitation_only`), a scratch `HOME`, and the stored
conversation read back over the API (`GET /conversations/{id}`):

| Command | Result |
| --- | --- |
| CLI `idea convert --locale en` | exit 0, stored locale `en` |
| CLI `idea convert --locale zh` with `LANG=en_US.UTF-8` | exit 0, stored locale `zh` |
| CLI `idea convert`, `LANG=zh_CN.UTF-8` | exit 0, stored locale `zh` |
| CLI `idea convert`, `LANG=en_US.UTF-8` | exit 0, stored locale `en` |
| CLI `idea convert`, no language in the environment | exit 0, stored locale `en` |
| CLI `idea convert --locale fr` | exit 2, `idea convert: locale "fr" is not supported: use en or zh` |
| MCP `idea_convert` with `locale` en, zh (under an en environment), none (zh environment), none (no environment) | stored locale `en`, `zh`, `zh`, `en` |
| MCP `idea_convert` with `locale` fr | `isError: true`, `locale "fr" is not supported: use en or zh` |

The quoted sign-up lines were run before the repair `3cb80a1`, which touches
only the convert path, so they are not repeated on the final binary:
`insideout register <email>` without `--invite` on that server answered `403
Forbidden: an invitation is required to register`; with `--invite` the account
was created and `insideout whoami` worked; `insideout login <email>` printed
`insideout: signed in to <api-base>; login saved to <path>` on stderr (the base
was the loopback server), the bare token on stdout, and `insideout whoami` then
answered from the saved login. The guard re-checks at every run that each
quoted verb exists and that `register` still lists `--invite`.

## Decisions and deviations

- **Only en and zh.** The coach speaks two languages and `store.NormalizeLocale`
  maps everything else to zh, so the narrowing is the client's job; a locale the
  verb cannot honour is refused rather than silently turned into Chinese.
- **The default is English, not the server's Chinese.** With no flag and no
  language in the environment the verb sends `en`. The environment is the best
  evidence of the person's language, and with none the CLI's own prose is
  English. If the owner prefers the server's default, it is the last `return` in
  `ConvertLocale`.
- **One method with a variadic.** `IdeaConvert(ideaID string, locale ...string)`
  keeps the escape sweep's one row per URL shape and every existing caller
  compiling; the first fix tried a second exported method and the sweep
  (`TestSweepCoversEveryClientMethod`, which requires a row for every exported
  `Client` method) rightly failed. `escape_test.go` belongs to
  `w3-test-hardening` and was not touched.
- **No contact address is invented.** The new copy names a role, "the server's
  administrator", never a mailbox or a person; see the owner decision below.
- **Tunables.** None added: the two language codes and the order of the
  environment variables are constants in `ConvertLocale`.
- **Three test files outside `files_owned` were edited**, because the rename
  forces them and they asserted the old words: `client/test/l10n_test.dart`
  (`'Workspace code: AB12CD'`), `client/test/phone_layout_test.dart` (the
  Chinese join title 用工作区代码加入) and `client/test/block_form_required_test.dart`
  (`find.text('Workspace code')`). Each is a one-line change to the string
  the test looks for, in `31e8fa7`.

## Owner decision (DEFERRED_OWNER)

Which route may a stranger use to ask for an invitation to the hosted
service? The maintainer's personal address cannot be published, and the
copy now says only "the server's administrator". The options are a role
mailbox (`<privacy-contact>`), a GitHub issue template or Discussions
category, or a form. Once it is chosen it needs one sentence in each of: the
README Install section, `docs/usage/installation.md`, the sign-up blocks of
`setup.sh`, `install.sh` and `bootstrap.sh` (the guard pins the phrase "invitation
code ... from the server's administrator", so that test moves with them), and
the web copy (`register.inviteError`, `register.closedError`). Until then
rv4-USER-12 stays partial.

## Known ceilings

- **rv4-FEAT-05's other half is open by design.** The export headings are
  fixed English in `server/internal/export/render.go`, outside this cluster's
  files; the review's fix there is a locale argument on export. Only the
  convert half is done here.
- **rv4-USER-10's second half is a product slice.** The review also asks that an
  invitation may carry a workspace, so the first screen after registering is the
  workspace and one code does both jobs. That is an API and schema change
  (`admin.go` takes only an email today), not copy, and was not attempted.
  The two names are now at least different.
- **The invitation field still shows on a closed server.** `inviteRequired` is
  true for `closed` as well as `invitation_only`, so a visitor to a closed
  server can still type a code that cannot work before the new message
  explains why. Hiding the field, or replacing the form with the closed notice
  up front, is a design choice left open.
- **`insideout setup --hosted` has its own, shorter blurb.** The Go-side copy
  in `server/cmd/insideout/cli_setup.go` (around line 153) prints "Sign in
  interactively: insideout setup" and "Or with the plain CLI: insideout login
  you@example.com". It is consistent (login first) but has no `whoami` and no
  sign-up route; it is not this cluster's file.
- **The "try it without an account" suggestion cannot go in the bootstrap
  closing message.** `check_ops_hardening.py` forbids `setup --local` there
  because it needs a repository checkout, and the prebuilt installer has none.
  The suggestion belongs in the README and the installation guide.
- **The first-run copy is checked by shape, not by a reader.** The guard proves
  the commands exist and the route is stated; whether a stranger understands "ask
  the server's administrator" is the owner decision above.
- **Not reproduced: a one-second delay per CLI call.** One full-package run
  of `cmd/insideout` showed about 1.0 s for each `idea convert` case that
  completes its request; the isolated run takes 0.01 s per case and the built
  binary 0.02 s. It is a harness artefact of the large package, not a product
  delay, and was not chased.

## Integrator notes (wave 3)

- `docs/changelogs/2026-10-18-rv4-remediation/index.md`: list this fragment
  (plan row 3.8 and LEAD-SETUP-BLURB) and the shared changelog indexes.
- `README.md`: the account invitation is still called an "invite code" around
  lines 158 and 185; it should read "invitation code", from the server's
  administrator once the owner has chosen the route. The installer order and the
  macOS/Linux wording should match `scripts/bootstrap.sh` and `scripts/install.sh`
  (the README hunks of rv4-PM-08 are `w3-docs-truth`'s).
- `docs/usage/installation.md` (about line 84): "Register with an invite code"
  should say "invitation code"; the saved-login wording is already consistent
  with the new blurb.
- `docs/usage/cli-mcp-and-guidance.md` (line 81, the Ideas row): `idea convert
  [--locale en|zh] <id>`, and the MCP tool `idea_convert` has a `locale`
  argument.
- `server/cmd/insideout/cli.go` (line 58): "join a workspace with its invite
  code" should say "workspace code".
- `server/cmd/insideout/cli_setup.go`: the Go-side hosted blurb, as above.
- Review register: rv4-USER-12 (partial, owner decision above), rv4-FEAT-05
  (partial, export headings in `export/render.go`) and the product slice of
  rv4-USER-10 need ids and owners.
- Learning: a new exported method on `apiclient.Client` needs a row in the
  escape sweep; run the whole package, not only the new tests, before calling a
  client change done.
