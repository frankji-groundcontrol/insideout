# Wave 3, privacy notice and retention record — review-4 remediation

Branch `rv4/w3-privacy`, `8c74218..72735de` (four commits: the red test, two
commits that fix rv4-PUB-02, and the retention-erasure decision record), then
this fragment. Findings are cited as `rv4-<ID>` because review IDs restart per
review. Row 3.5 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md).
No address, host, key or provider id is spelled in this fragment; the contact
in the notice is the placeholder `<privacy-contact>`.

## What changed

- **A privacy notice, and a link to it where people sign up** (rv4-PUB-02, red
  `d39aff2`, fix `e225bed`, prose fix-up `8b16bca`). `PRIVACY.md` at the
  repository root says what is stored (account, sign-in state, workspaces,
  work, coach conversations with their run records, the request log), what
  lives only in server memory (presence, sign-in throttling), who can see it
  inside the service, which outside services the code can send user data to
  and what each one receives, retention exactly as built, what deleting your
  data means today, and the contact. The outside services are the model
  provider, Resend, GitHub and the Google-hosted fonts and CanvasKit; each is
  named with the variable that switches it on (`INSIDEOUT_LLM_API_KEY`,
  `INSIDEOUT_MAIL_API_KEY`, a repository bound to a project), and the two
  keys-off defaults are stated (offline template coach, no mail sent). The auth
  door gets an optional `footer` slot, and the landing page fills it with a
  `PrivacyNoticeLink` for both the sign-in and the sign-up prompt. The address
  is one named constant, `privacyNoticeUrl`, in `auth_door.dart`. The strings
  are `privacy.link` and `privacy.linkCopied` in `en_shell` and `zh_shell`.
- **The first draft of the notice had five claims that were wrong, and the
  second commit fixes them** (`8b16bca`). Each claim was re-read against its
  source after the first commit: an idea is visible to its author and the
  workspace admins (the draft said "until converted", which no policy says);
  admins list every invitation, not only their own; dropping an idea sets
  `status = 'dropped'` and keeps the text, it does not delete it;
  conversations go only with their workspace, because no route deletes a PRD
  or a conversation; and row-level security limits reads per user but does not
  cover every table (sessions, one-time tokens and invitations sit outside
  it). Also added: the native builds keep tokens in the system's secure
  storage, and the mail body is logged only when the operator opts in.
- **Retention and erasure are a decision record, not code** (rv4-DATA-06,
  `72735de`, status `DEFERRED_OWNER`). The owner-decisions record holds no
  window and no erasure rule, and the harness forbids a migration and a
  delete-account route, so no red/fix pair exists. The record
  [`2026-10-18-rv4-retention-erasure.md`](../../issues/2026-10-18-rv4-retention-erasure.md)
  has the options for erasing an account, the 11 `NO ACTION` user references
  with their migrations, what an erasure must cover, retention proposals with
  a recommended default, why the reaper cannot purge conversations under FORCE
  RLS today, the five-row question for the owner, and a bounded fix prompt for
  the cluster that follows the answer.

## Design spike (row 3.5, marked `dir`)

Four choices. **Where the notice lives**: a file at the repository root,
linked by an absolute URL, over a page bundled into the web build. The build
carries no Markdown, the bundled assets belong to `w3-client-assets` (running
in parallel), a notice should be readable without signing in and without the
app, and a root file is the one reviewers and operators edit beside the code;
the cost is that the link depends on the repository host, and a fork changes
one constant. **Where the link sits**: the auth door's new `footer` slot, so
the sign-in and sign-up prompts share one widget and one constant, and the
three other doors (reset, magic link, forgot password) are untouched because
the slot is optional. **How it opens**: `url_launcher` is not a dependency and
`pubspec.yaml` and `pubspec.lock` are outside this cluster's list, so the
opener is a conditional-import pair modelled on `download_stub`/`download_web`:
`open_url_web.dart` clicks an anchor with `noopener noreferrer` (a navigation
the user's own tap started, so a popup blocker leaves it alone) and refuses
anything but https; the stub returns false. Where there is no browser (a test,
a native build) the link copies the address and says so in place of its label,
because a snackbar would be drawn on the page under the door's dim barrier.
**Retention and erasure**: policy only. With no recorded owner decision, the
honest deliverable is the options, their costs and a default, not an invented
window; PRIVACY.md therefore states today's behaviour and promises neither a
route nor a response time.

## Evidence

| Check | Result |
|---|---|
| Red `d39aff2`, `flutter test test/auth_door_privacy_link_test.dart` | `+0 -7`: the link is absent for login and register in en and zh (4), the tap test, the constant scan and the PRIVACY.md checks fail. Output kept in `/private/tmp/insideout-rv4/reports/w3-privacy-red-PUB-02.txt`. |
| Green at `e225bed` and at HEAD, same command | `+7: All tests passed!` (`w3-privacy-green-PUB-02.txt`). The test file is byte-identical to the red commit. |
| Chrome lane, `flutter test --platform chrome test/web/open_url_web_test.dart` | `+2`: an https address clicks an anchor with `target=_blank` and `noopener noreferrer`; `javascript:`, `data:`, `http:`, a path and the empty string are refused and nothing is clicked. The whole CI lane, `flutter test --platform chrome test/web` (`ci.yml`, line 273), is `+8: All tests passed!`. |
| Mutation: drop the `footer:` argument in `landing_page.dart` | `+2 -5` (the four link tests and the tap test go red; the constant scan and the PRIVACY.md checks stay green). Restored; the tree is clean. |
| `flutter pub get --enforce-lockfile` | Ok; `client/pubspec.lock` and `pubspec.yaml` are untouched. |
| `flutter analyze --no-fatal-infos` | 14 infos, the CI baseline; none in a touched file. |
| `flutter test` and `TZ=Asia/Shanghai flutter test` | 490 passed in each, 0 failed. The browser-only tests are skipped on the VM by design. |
| `dart format --output=none --set-exit-if-changed lib test` | 274 files, 0 changed. |
| PRIVACY.md address gate, `sh scripts/check-redaction.sh` | Clean; prints nothing. |
| `sh scripts/check-docs-recorded.sh` | Prints nothing. |
| Every `scripts/tests/check_*.py` | 9 scripts, 0 failed. They print their own `ok` summary lines; no failure output. |
| Files over 350 lines among the changed `.go` and `.dart` | None: the 8 changed files run from 4 to 285 lines (the largest is `landing_page.dart`; `auth_door.dart` is 216). |
| Layout | A throwaway golden render of the door at 900 by 900, both prompts (the test font draws blocks for glyphs): the link sits centered at the bottom of the card below the switch line, no overflow. The scratch files were deleted and never committed. |

## What was not exercised

- The link in a built web app. The Chrome test cancels the click with a
  capture listener, so no real tab opened, and no `flutter build web` was run.
- A native build. The copy-and-say-so path runs on the VM and is the same code
  on iOS, Android and desktop (where `dart.library.ui_web` is absent), but no
  device or simulator was started.
- A running server. This cluster needs no database; every claim in PRIVACY.md
  about the server was read from the code and the migrations, not observed.
- Whether the Google CDNs are in fact contacted on first load (the plan marks
  COMP-03 unverified); the notice states what `nginx.conf` allows and
  `index.html` references.

## Ceilings, recorded on purpose

- The notice can drift from the code. No guard ties the processor list to the
  variables that switch each one on. None was added: the row asks for a notice
  and a link, and CI runs every `scripts/tests/check_*.py` by glob, so a later
  guard needs no change to `ci.yml`.
- The Google-hosted fonts row holds only until row 3.11 (`w3-client-assets`)
  bundles the fonts and CanvasKit with the build. The row says so; the
  integrator removes it then (see the last TODO below).
- `<privacy-contact>` is a placeholder on purpose. The notice is not final for
  a hosted launch until the owner answers row 5 of the record.
- The notice is English only (the docs-prose rule), and both locales of the
  door link to it. A Chinese notice is a separate decision.
- `privacyNoticeUrl` points at the `main` branch on the repository host. A
  rename or a fork changes that one constant.

## Scope note: files outside the literal list

`files_owned` named eight paths. Two new non-test files were needed for a link
that actually opens: `client/lib/api/open_url_stub.dart` and
`client/lib/api/open_url_web.dart` (Dart's conditional import needs two files,
and `url_launcher` cannot be added from this cluster). One new test file,
`client/test/web/open_url_web_test.dart`, covers the browser half. No existing
file outside the list was edited. rv4-PUB-02 is therefore reported `PARTIAL`
by scope only: the behaviour is complete, and the integrator accepts or
rejects the two extra source files.

The harness prose named the decision record
`2026-10-18-rv4-retention-erasure-decision.md`; `files_owned` says
`2026-10-18-rv4-retention-erasure.md`. The second, the list, is what was
written, and PRIVACY.md links to it.

## Left for the integrator

INTEGRATOR TODO: README.md: link `PRIVACY.md` beside `SECURITY.md` in the
trust and policy links (README is `w3-docs-truth`'s).
INTEGRATOR TODO: docs/issues/README.md: add an index line for
`2026-10-18-rv4-retention-erasure.md` (open, owner decision, rv4-DATA-06).
INTEGRATOR TODO: docs/usage/environment.md: say that `INSIDEOUT_LLM_API_KEY`
and `INSIDEOUT_MAIL_API_KEY` are what let user data leave the server (the
processors table in `PRIVACY.md`), and that a hosted operator must publish a
monitored contact and replace `<privacy-contact>` there.
INTEGRATOR TODO: docs/changelogs/2026-10-18-rv4-remediation/index.md: add a row
for `w3-privacy.md`.
INTEGRATOR TODO: docs/issues/2026-10-18-review-4-open-findings.md: record
rv4-DATA-06 as `DEFERRED_OWNER` (decision record above) and rv4-PUB-02 as
fixed, `PARTIAL` by scope only.
INTEGRATOR TODO: PRIVACY.md: after `w3-client-assets` merges, reconcile the
"Google-operated hosts" row (remove it, or restate it for whatever still
loads from a CDN).
INTEGRATOR TODO: PRIVACY.md and docs/issues/2026-10-18-rv4-retention-erasure.md:
update both when the owner answers the record's five questions.
