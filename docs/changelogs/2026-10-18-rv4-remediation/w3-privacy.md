# Wave 3, privacy notice and retention record — review-4 remediation

Branch `rv4/w3-privacy`, base `8c74218`: the red test `d39aff2`, two commits
that fix rv4-PUB-02, the retention-erasure decision record `72735de`, this
fragment, and one repair commit (`ca5115a`, verifier round 1) that takes the
link back to the files this cluster owns. Findings are cited as `rv4-<ID>`
because review IDs restart per review. Row 3.5 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md).
No address, host, key or provider id is spelled in this fragment; the contact
in the notice is the placeholder `<privacy-contact>`.

## What changed

- **A privacy notice, and a link to it where people sign up** (rv4-PUB-02, red
  `d39aff2`, fix `e225bed`, prose fix-up `8b16bca`, scope repair `ca5115a`).
  `PRIVACY.md` at the repository root says what is stored (account, sign-in
  state, workspaces, work, coach conversations with their run records, the
  request log), what lives only in server memory (presence, sign-in
  throttling), who can see it inside the service, which outside services the
  code can send user data to and what each one receives, retention exactly as
  built, what deleting your data means today, and the contact. The outside
  services are the model provider, Resend, GitHub and the Google-hosted fonts
  and CanvasKit; each is named with the variable that switches it on
  (`INSIDEOUT_LLM_API_KEY`, `INSIDEOUT_MAIL_API_KEY`, a repository bound to a
  project), and the two keys-off defaults are stated (offline template coach,
  no mail sent). The auth door gets an optional `footer` slot, and the landing
  page fills it with a `PrivacyNoticeLink` for both the sign-in and the sign-up
  prompt. The address is one named constant, `privacyNoticeUrl`, in
  `auth_door.dart`. The strings are `privacy.link` and `privacy.linkCopied` in
  `en_shell` and `zh_shell`. A tap copies the address and swaps the label for
  "Link copied"; it does not open a tab (the repair bullet below).
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
- **Repair round 1: the link copies the address and does not open a tab**
  (rv4-PUB-02, `ca5115a`). The first version opened the notice in a new tab
  through a conditional-import pair, `client/lib/api/open_url_stub.dart` and
  `open_url_web.dart`, with a Chrome test, `client/test/web/open_url_web_test.dart`.
  None of the three is in this cluster's `files_owned`, and the verifier's scope
  check failed on them (every other check passed). They are deleted.
  `PrivacyNoticeLink` keeps one path: copy `privacyNoticeUrl`, swap the label
  for `privacy.linkCopied`. The trailing icon is `link`, not `open_in_new`,
  because nothing opens now. The assertions of
  `auth_door_privacy_link_test.dart` are unchanged; its header comment and the
  title of the tap test say what the link does now. rv4-PUB-02 is `PARTIAL`
  for this reason: a web user taps the link and pastes the address by hand.
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
the slot is optional. **How the link behaves**: it copies the address and says
so in place of its label, because a snackbar would be drawn on the page under
the door's dim barrier. Opening a tab was built first and withdrawn.
`url_launcher` is not a dependency, and `pubspec.yaml` and `pubspec.lock`
belong to `w3-client-assets`, so the opener was a conditional-import pair
modelled on `download_stub` and `download_web` (an anchor click with
`noopener noreferrer`, https only). That pair sits outside this cluster's list
(see the scope note). The existing `openTextInNewTab` was considered and
rejected: it opens a text blob, so a redirect page would leave `window.opener`
pointing at the app. **Retention and erasure**: policy only. With no recorded
owner decision, the honest deliverable is the options, their costs and a
default, not an invented window; PRIVACY.md therefore states today's behaviour
and promises neither a route nor a response time.

## Evidence

| Check | Result |
|---|---|
| Red `d39aff2`, `flutter test test/auth_door_privacy_link_test.dart` | `+0 -7`: the link is absent for login and register in en and zh (4), the tap test, the constant scan and the PRIVACY.md checks fail. Output kept in `/private/tmp/insideout-rv4/reports/w3-privacy-red-PUB-02.txt`. |
| Green at `e225bed`, and at HEAD `ca5115a` after the repair, same command | `+7: All tests passed!` (`w3-privacy-green-PUB-02.txt` for round 0, `w3-privacy-r1-green-PUB-02.txt` at `ca5115a`). Every assertion is the red commit's; `ca5115a` changed the file's header comment and the title of the tap test only. |
| Scope check (verifier check 4), red at `f25bc1e` | 11 changed paths since the base; **3 outside `files_owned`**: `client/lib/api/open_url_stub.dart`, `client/lib/api/open_url_web.dart`, `client/test/web/open_url_web_test.dart`. Exit 1 (`w3-privacy-r1-scope-red.txt`). |
| Scope check, green at `ca5115a` | 8 changed paths, every one in `files_owned`. Exit 0 (`w3-privacy-r1-scope-green.txt`). This is a property of the file set, so no unit test can be red then green on it. The VM behaviour of the link did not change (the VM already took the copy path, because its opener stub returned false), which is why the focused test is green on both sides. |
| Chrome lane, `flutter test --platform chrome test/web` (`ci.yml`, line 273) | `+6: All tests passed!` (`w3-privacy-r1-gate-chrome-lane.txt`): the `download_web` and `sse_transport_web` tests, three each. It was `+8`; the two `open_url_web` tests left with their file. |
| Mutations at `ca5115a`, each restored with `git checkout -- <path>` (`w3-privacy-r1-mutations.txt`) | M1 drop the `footer:` argument in `landing_page.dart`: `+2 -5` (the four link tests and the tap test). M2 the tap copies another address: `+6 -1` (the tap test). M3 the label never swaps to "Link copied": `+6 -1` (the tap test). M4 rename the `privacyNoticeUrl` constant: `+6 -1` (the constant scan). The tree is clean afterwards. |
| `flutter pub get --enforce-lockfile` | Ok; `client/pubspec.lock` and `pubspec.yaml` are untouched. |
| `flutter analyze --no-fatal-infos` | 14 infos, the CI baseline (all in `lib/api/client_core.dart`, `client_prd.dart`, `client_stream.dart`, `requests.dart`); none in a touched file (`w3-privacy-r1-gate-analyze.txt`). |
| `flutter test` and `TZ=Asia/Shanghai flutter test` | 490 passed in each, 0 failed (`w3-privacy-r1-gate-test.txt`, `-test-tz.txt`). The count equals the one before the repair because the deleted browser-only file was skipped on the VM. |
| `dart format --output=none --set-exit-if-changed lib test` | 271 files, 0 changed (274 before the three files left). |
| PRIVACY.md address gate, `sh scripts/check-redaction.sh` | Clean; prints nothing. |
| `sh scripts/check-docs-recorded.sh` | Prints nothing. |
| Every `scripts/tests/check_*.py` | 9 scripts, 0 failed. They print their own `ok` summary lines; no failure output. |
| Files over 350 lines among the changed `.go` and `.dart` | None: the 5 changed files run from 57 to 285 lines (the largest is `landing_page.dart`; `auth_door.dart` is 216). |
| Layout | A throwaway golden render of the door at 900 by 900, both prompts (the test font draws blocks for glyphs): the link sits centered at the bottom of the card below the switch line, no overflow. The scratch files were deleted and never committed. Taken in round 0; the only visual change since is the trailing glyph, `link` in place of `open_in_new`. |

## What was not exercised

- The link in a built web app. No `flutter build web` was run and no browser
  tapped the link. The tap path is the widget test with the platform channel
  replaced by a handler that records the clipboard text, so a real browser's
  clipboard write was not exercised, and Flutter shows "Link copied" without
  any success signal from the browser.
- A native build. The tap path is the same code on iOS, Android and desktop
  now, but no device or simulator was started.
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
- The link copies and does not open a tab. A web user taps it and pastes the
  address by hand. This is the price of keeping the branch inside the cluster's
  `files_owned`; the scope note says how to get the tab back.

## Scope note: the three files that left the branch

`files_owned` names eight paths. Round 0 added three more, so that the link
could open a tab: `client/lib/api/open_url_stub.dart`,
`client/lib/api/open_url_web.dart` and `client/test/web/open_url_web_test.dart`.
It reported rv4-PUB-02 `PARTIAL` by scope and the verifier's scope check
failed on exactly those paths (round 1; every other check passed). The
verifier preferred that the orchestrator widen the list and otherwise that the
worker remove the files. The brief for this repair round carries the same list
of eight, so the files are removed (`ca5115a`). The branch now changes only
those eight paths (`w3-privacy-r1-scope-green.txt`, which runs `git diff
--name-only <base> HEAD` against the list).

To get the tab-opening link back, a cluster has to own the three paths and
`auth_door.dart`. The code is in history: `git show
e225bed:client/lib/api/open_url_web.dart` and the other two files at the same
commit, plus the one-line import and the `_open` method in `auth_door.dart` at
`e225bed`. Or revert `ca5115a` (`git show ca5115a | git apply -R --check`
passes on this branch). Nothing else is needed: `package:web` is already a
direct dependency and `pubspec.yaml` is untouched.

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
`PARTIAL`: `PRIVACY.md` and the sign-in and sign-up link exist in both
locales, a tap copies the address instead of opening a tab, and restoring the
tab needs the three paths named in the scope note (a wave-4 candidate).
INTEGRATOR TODO: PRIVACY.md: after `w3-client-assets` merges, reconcile the
"Google-operated hosts" row (remove it, or restate it for whatever still
loads from a CDN).
INTEGRATOR TODO: PRIVACY.md and docs/issues/2026-10-18-rv4-retention-erasure.md:
update both when the owner answers the record's five questions.
