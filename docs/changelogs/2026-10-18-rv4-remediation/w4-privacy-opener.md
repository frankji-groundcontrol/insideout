# Wave 4, the privacy link opens the notice again, and the notice is tied to the CSP — review-4 remediation

Branch `rv4/w4-privacy-opener`, base `582a4bd`: the red test `f6db60d`, the fix
`68eb299`, the guard `2ff6535`, and this fragment. Findings are cited as
`rv4-<ID>` because review IDs restart per review. Row R3-21 of the
[open-findings record](../../issues/2026-10-18-review-4-open-findings.md) and
the whole-change review row rv4-FR-F12; the plan is
[40-remediation-plan.md](../../review/2026-10-18-multirole-4/40-remediation-plan.md).
No address, host, key or provider id of a deployment is spelled here; the notice
address is referred to as `privacyNoticeUrl`.

## What changed

- **The privacy link opens the notice in a new tab on the web** (rv4-PUB-02,
  red `f6db60d`, fix `68eb299`). This restores what [w3-privacy](w3-privacy.md)
  built as `e225bed` and had to drop in `ca5115a` because the opener files sat
  outside its owned paths; the verifier had run that opener green, so it is a
  restore, not a design. `client/lib/api/open_url_web.dart` (new, 19 lines):
  `openExternalUrl(url)` returns false unless the scheme is `https`; otherwise
  it appends an anchor (`target="_blank"`, `rel="noopener noreferrer"`) to the
  body, clicks it, removes it and returns true. An anchor click, not
  `window.open`: it is a navigation the user's own tap started, so a popup
  blocker leaves it alone, and `noopener` cannot turn the call's answer into
  null. `client/lib/api/open_url_stub.dart` (new, 4 lines) returns false on every
  other platform. `auth_door.dart` picks one of the two with the
  `if (dart.library.ui_web)` conditional import its sibling pairs use
  (`download_*`, `sse_transport*`, `secret_query*`); `PrivacyNoticeLink` calls
  `openExternalUrl(privacyNoticeUrl)` first and keeps the old behaviour as the
  fallback (copy the address, show the "copied" message in place of the label).
  `privacyNoticeUrl` is still one named constant. No dependency changed:
  `package:web` was already in `pubspec.yaml`, and `pubspec.lock` is untouched.
- **One deviation from `e225bed`: the glyph says what a tap does.** The
  trailing icon is `open_in_new` in a browser (`kIsWeb`), `link` elsewhere
  and `check` after a copy. `e225bed` drew `open_in_new` on every platform,
  which on a build that only copies would promise a tab.
- **A guard ties the published notice to the CSP** (rv4-FR-F12, `2ff6535`).
  `scripts/tests/check_privacy_csp_rv4.py` (new, 169 lines, stdlib, run as
  `python3 -I`; CI runs it through the `check_*.py` glob, so `ci.yml` needs no
  edit). It reads the `Content-Security-Policy` line(s) of `client/nginx.conf`
  and fails when (1) `PRIVACY.md` names a host the CSP does not allow and that
  is not in the guard's `SERVER_SIDE` set (`api.github.com` and `github.com`:
  hosts only the server, or a reader who clicks a link, reaches); (2) the CSP
  allows a host the notice does not name, or an open source (`https:`, `*`);
  (3) while the CSP allows `'self'` alone, the notice describes a Google font or
  renderer host, lists a font or renderer as a recipient-table row, or no longer
  has a sentence saying the fonts and CanvasKit come from the deployment's own
  origin. Every run also re-proves on the live files that six such regressions
  turn it red and that a notice naming the host a CSP allows stays green, so it
  follows the CSP instead of banning a fixed list.

## Tests

- `client/test/web/open_url_web_test.dart` (new, Chrome): an https address
  returns true and puts exactly one click on the document, with `href` the
  address, `target` `_blank`, a `rel` holding both `noopener` and `noreferrer`,
  the anchor connected to the document at click time and disconnected after;
  `javascript:`, `data:`, `http:`, a relative path and the empty string return
  false and click nothing.
- `client/test/web/privacy_link_web_test.dart` (new, Chrome): the real
  `PrivacyNoticeLink`, tapped under Chrome, clicks one anchor aimed at
  `privacyNoticeUrl` with target `_blank`, copies nothing and keeps its label;
  the glyph is `open_in_new`, not `link`. It also proves that the conditional
  import resolves to the web opener in a browser build, which the VM suite cannot.
- `client/test/auth_door_privacy_link_test.dart` (edited): the header and the
  tap test's title say that this VM suite is compiled with the stub, so a tap
  copies; the glyph is pinned (`link` before the tap, `check` after). Every
  original assertion is kept.

## Evidence

| Check | Result |
|---|---|
| Red `f6db60d`, `flutter test test/web --platform chrome` | Exit 1, `Failed to compile`: `Error when reading 'lib/api/open_url_web.dart': No such file or directory`, and `Method not found: 'openExternalUrl'` twice. One compile error fails the whole lane (`red-PUB-02-lane-exit.txt`). |
| Red on behaviour, against the copy-only widget | The Chrome reporter prints no text for a failed `testWidgets` ("See exception logs above." with nothing above), so a throwaway copy of the widget test printed the exception and was deleted. Tap: `Expected: an object with length of <1>  Actual: []` (no anchor clicked). Glyph: `Found 0 widgets with icon` `open_in_new` (`red-PUB-02-widget-chrome.txt`). |
| Green at `68eb299`, same lane, with `CHROME_EXECUTABLE` set to Google Chrome 155 | `+10: All tests passed!`, 0 skipped. It was `+6`; the two opener tests and the two widget tests are the four new ones (`gate-chrome-lane.txt`). |
| VM, `flutter test` and `TZ=Asia/Shanghai flutter test` | 537 passed in each, 0 failed. The VM privacy suite is green before and after by construction: it is compiled with the stub, so its tap path was already the copy path. |
| Mutations of PUB-02, each restored (`mutations-PUB-02.txt`) | 12 of 12 red: any scheme may open; no `rel`; `rel` without `noreferrer`; no target; anchor never removed; anchor never appended; the tap never calls the opener; the conditional import never selects the web opener; copies even after a successful open; glyph flipped (Chrome side and VM side); fallback copies another address. |
| FR-F12 premise check | Base `PRIVACY.md` is already repaired: no Google row, no "until that ships" sentence; it says CanvasKit and the fonts come from the deployment's own origin and nothing is sent to a font or renderer host (`da93d78` rewrote the paragraph; `b0b260c` had made the CSP self-only). `PRIVACY.md` and `nginx.conf` are not edited. |
| FR-F12 guard, green at HEAD | `privacy vs CSP (rv4) ok: the CSP allows 'self' only; the notice agrees; 6 regressions still go red`. |
| FR-F12 guard, red against the reviewed commit `3a1c4ac` (its `PRIVACY.md` and `nginx.conf`, passed as arguments) | Exit 1, six messages: the three hosts `fonts.googleapis.com`, `fonts.gstatic.com`, `www.gstatic.com` named though the CSP allows none; a Google font or renderer host described; the Google row listed as a recipient; no own-origin sentence (`frf12-guard-reviewed-vs-base.txt`). |
| FR-F12 mutations, each restored with `git checkout -- PRIVACY.md` (`frf12-mutations.txt`) | Red, with the unmutated file green before and after: the reviewed Google row put back (5 messages); a Google font host in prose; "Google Fonts" with no host; an unnamed font CDN row; the own-origin sentence dropped; a non-Google CDN host named; and, on a scratch copy of `nginx.conf` outside the repo, a CSP that allows `fonts.gstatic.com` while the notice does not name it. |
| `flutter pub get --enforce-lockfile`; `flutter analyze --no-fatal-infos` | Ok; `pubspec.yaml` and `pubspec.lock` untouched. 14 infos, the CI baseline, all in `client_core.dart`, `client_prd.dart`, `client_stream.dart` and `requests.dart`; none in a touched file. |
| `dart format --output=none --set-exit-if-changed lib test` | 286 files, 0 changed. |
| `flutter build web --release --no-web-resources-cdn --dart-define=API_BASE=/api/v1` (what `ci.yml` runs) | Built; `main.dart.js` contains `noopener noreferrer`. |
| `gofmt -l internal cmd`; files over 350 lines among the changed `.dart` and `.py` | Prints nothing; the 7 changed files run from 4 to 220 lines. |
| Every `scripts/tests/check_*.py` with `python3 -I`; `sh scripts/check-redaction.sh`; `sh scripts/check-docs-recorded.sh` | 15 scripts, 0 failed (`check_doc_links.py` again after this fragment was committed: 0 broken links); the other two print nothing and exit 0. |

## What CI runs, and what only CI can prove

`ci.yml` (read only) runs, in the client job: `flutter pub get --enforce-lockfile`,
the analyzer with the info baseline of 14, `dart format`, `flutter test`,
`flutter test --platform chrome test/web` and `flutter build web --release
--no-web-resources-cdn`; its contract-guard step runs every
`scripts/tests/check_*.py`. So the new Chrome tests and the new guard are in CI
without a config change. This run used Flutter 3.44.8, Chrome 155 on macOS and
Python 3.9; CI pins Flutter 3.44.0 and Python 3.12 (the scripts target 3.9 or
newer) and uses the runner image's Chrome. **Pending first CI run after push:**
the Chrome lane on the runner, the web build on the pinned Flutter, and the
guard on Python 3.12.

## What was not exercised

- A real navigation. The Chrome tests cancel the anchor's click in a capture
  listener, so they prove the click, the href, the target and the rel, not that a
  tab opened. A popup blocker's behaviour with an anchor click was not observed
  in any browser here.
- A native build. The stub path is the VM suite; no device or simulator ran.
- The CSP. None of the directives in the `client/nginx.conf` policy governs a
  link the user follows, so the opener needs no CSP change; this was read from
  the policy line, not observed in a browser.

## Ceilings, recorded on purpose

- The https check is on the scheme only. The one caller passes a constant;
  a later caller that passes user input needs a host policy as well.
- The FR-F12 guard finds hosts with a TLD list and the wording of its
  `GOOGLE` and `CLIENT_LOAD` patterns. A claim written with none of them, or a
  contradiction between two prose sentences, passes; a human reads the notice.
- It is the browser half only. R3-23 (the processor list against the server's
  configuration names for the model key, the mail key and the GitHub App) is
  still open and unowned.
- When the CSP allows an off-origin host the guard asks only that the notice
  names it; it cannot judge what the notice says about it.
- FR-F12 has no red commit. The base was already green, so the row is closed
  by the guard and the mutation proof above, not by a red-then-green pair.

## Left for the integrator

- INTEGRATOR TODO: docs/issues/2026-10-18-review-4-open-findings.md: R3-21
  (line 146) is closed by `f6db60d` and `68eb299`; rv4-PUB-02 goes from
  `PARTIAL` to `FIXED`, and the sentence "the new-tab opener is open, R3-21" in
  the rv4-PUB-02, rv4-DATA-06 row (line 281) no longer holds.
- INTEGRATOR TODO: docs/changelogs/2026-10-18-rv4-remediation/wave-3-dispositions.md:
  the rv4-PUB-02 `PARTIAL` row (line 63) and the `PARTIAL` lists in `wave-3.md`
  (lines 77 and 92) are superseded; the wave-4 record should say so. rv4-FR-F12
  belongs in the wave-4 dispositions as "already repaired by `da93d78`, closed
  by guard `2ff6535` and a mutation proof".
- INTEGRATOR TODO: docs/changelogs/2026-10-18-rv4-remediation/index.md: add the
  row for this fragment (wave 4, `w4-privacy-opener`, rv4-PUB-02 and rv4-FR-F12).
- INTEGRATOR TODO: docs/changelogs/README.md and docs/changelogs/index/2026-10.md:
  index this fragment if the month index lists fragments.
- INTEGRATOR TODO: docs/learning/README.md: a candidate lesson, "the Chrome
  lane prints no text for a failed `testWidgets`; to see it, run a throwaway
  copy of the test that wraps the body in `try`/`catch` and `print`s the
  exception, and delete the copy" (the method used for the behaviour-level red
  above).
- Not touched, on purpose: `PRIVACY.md` (already true), `client/nginx.conf`
  (another cluster owns it), `landing_page.dart` and the l10n files.
