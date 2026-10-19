# Wave 4, first-run wording and `insideout logout` — review-4 remediation

Branch `rv4/w4-copy-sweep`, base `88c9d272`. Findings are cited as `rv4-<ID>`
because review IDs restart per review. Two roots. The wording root is one
red/fix pair shared by the two findings it still had open; logout is its own
pair. No address, host, secret or token is spelled in this fragment.

| Finding | Status | Red | Fix |
| --- | --- | --- | --- |
| `rv4-USER-10` | FIXED (the two names; an invitation that also joins a workspace was not this change) | `39e540b5` | `1d5f68a3` |
| `rv4-PM-08` | FIXED (the hosted blurb and the join line; the README install order was already true on this base) | `39e540b5` | `1d5f68a3` |
| `rv4-FR-F70` | FIXED (CLI half; `reset-password` also drops the saved login) | `654f297b` | `ce674f43` |
| `rv4-USER-12` | DEFERRED_OWNER | — | — |
| `rv4-FEAT-05` | NOT_REAL on this base (convert half already repaired; export headings are not this cluster's files) | — | — |

## What changed

- **Two codes, two names** (`rv4-USER-10`, and the join line of `rv4-PM-08`).
  `insideout help` says `join` takes a workspace code. `insideout register`
  with no email prints `invitation code` and still documents the flag as
  `--invite` (the flag name is the API surface and did not change). The Auth
  row of [cli-mcp-and-guidance.md](../../usage/cli-mcp-and-guidance.md) uses
  `--invite <invitation-code>`, and
  [database-and-rls.md](../../architecture/database-and-rls.md) says a workspace
  is joined by its workspace code. `scripts/tests/check_first_run_copy.py`
  reads the two words from the English client catalogue and fails if those
  files say "invite code".
- **`setup --hosted` teaches the same sign-in as `scripts/setup.sh`**
  (`rv4-PM-08`). It prints `insideout whoami` and
  `insideout register --invite <invitation-code> you@example.com`, and says
  the invitation code comes from the server's administrator. It no longer
  tells a reader to run `insideout setup --local`, which exits 2 when there
  is no repository checkout.
- **`insideout logout`** (`rv4-FR-F70`). It sends the saved refresh token to
  `Client.Logout` (`POST /auth/logout`), then deletes the config file when
  the file holds nothing else, or clears only the token fields when it holds
  more. No saved login prints `not signed in` and exits 0. An unreachable
  server still deletes the local file and says the server session could not
  be revoked and expires by itself. A refusal (for example 503) keeps the
  file so a retry sends the same token. `insideout reset-password` clears a
  saved login for that server after the reset succeeds: the command already
  said every session was revoked, and the dead pair was staying on disk.
  The verb is in `cliVerbs`, the dispatch switch and
  [cli-mcp-and-guidance.md](../../usage/cli-mcp-and-guidance.md), next to the
  path `<OS config dir>/insideout/config.json`. `auth_logout` on the MCP
  server still clears process memory only; that code is not in this cluster's
  files, and the doc now says the saved file stays until `insideout logout`.

## Already true on this base, so not edited

- README install order: the prebuilt one-liner comes before `scripts/install.sh`.
  The platform line says macOS and Linux only and that Windows is not
  supported, matching the opening comments of `scripts/bootstrap.sh` and
  `scripts/install.sh`.
- README English says "invitation code" where the Chinese lines say 邀请码
  (about lines 52, 180 and 207). Those Chinese lines were left as they are.
- `docs/usage/installation.md` already says "invitation code".
- `idea convert [--locale en|zh]` is already a verb
  (`server/cmd/insideout/cli_idea.go`) and the Ideas row of the CLI doc. That
  is the convert half of `rv4-FEAT-05`.

## Evidence

Red commits are tests only (`git show --stat`). Outputs below are the failing
lines, not a full package run.

| Finding | Red | Green |
| --- | --- | --- |
| `rv4-USER-10`, `rv4-PM-08` | `check_first_run_copy.py` exit 1: join summary `join a workspace with its invite code`; register usage `[--invite code]`; hosted blurb missing `insideout whoami` and the register line; both docs say "invite code". `TestRvCopyJoinHelpNamesTheWorkspaceCode`, `TestRvCopyRegisterUsageNamesTheInvitation` and `TestRvCopyHostedBlurbTeachesWhoamiAndTheInvitation` fail the same way. | `first-run copy ok: 7 checks`. Those three tests pass. |
| `rv4-FR-F70` | `TestRvCopyLogoutEndsTheSavedLogin`: `handled false exit 0`. `TestRvCopyResetPasswordDropsTheSavedLogin`: `saved login still present after reset-password`. `TestRvCopyLogoutIsDocumentedNextToLogin`: the CLI doc names neither `config.json` nor logout. | `TestRvCopyLogout*` and `TestRvCopyResetPasswordDropsTheSavedLogin` pass. |

## Not taken

- **`rv4-USER-12` is the owner's route.** The copy already says to ask the
  server's administrator, and the README's temporary route is an "Invite
  request" issue with no e-mail address, or the repository owner through
  GitHub. The question still open: which permanent route may a stranger use
  to ask for an invitation — a role mailbox (`<privacy-contact>`), a GitHub
  issue template or Discussions category, or a form?
- **An invitation that carries a workspace** (`rv4-USER-10`'s other half) is
  an API and schema change. `invite` takes an e-mail only. Not attempted.
- **`rv4-FEAT-05` export headings** stay English in
  `server/internal/export/render.go`. Not this cluster's file.

## Integrator

- `docs/changelogs/2026-10-18-rv4-remediation/index.md`: list this fragment.
- `server/internal/export/render.go`: Chinese PRD export headings are still
  the English table (`rv4-FEAT-05`).
- The register form still shows the invitation field when signup is closed
  (client files, not owned). The closed-server 403 string is already distinct.
- `auth_logout` in the MCP server still does not delete the saved login file.
  The CLI verb does. Point the tool at the same file only if that server
  should end the saved session too.
