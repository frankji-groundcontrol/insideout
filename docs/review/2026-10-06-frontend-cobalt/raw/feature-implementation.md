# Feature implementation review — frontend vs. README/PRODUCT claims

This review checked every frontend-facing claim in `README.md` and `PRODUCT.md` against the
implementing code in `client/lib`, and checked the live screenshots under
`docs/review/2026-10-06-frontend-cobalt/shots/` where useful. Verdict first: the advertised
feature set is real. Every one of the sixteen checked features is IMPLEMENTED or PARTIAL at
the page level; nothing advertised is DEAD or DOCS-ONLY. The gaps are in the last mile of
otherwise-working features, and nearly all are small diffs.

## Feature matrix

| Advertised feature (README / PRODUCT.md) | Verdict | Implementing code |
| --- | --- | --- |
| Idea inbox capture | IMPLEMENTED | `ideas_page.dart:60-66,195-224` (screenshot 08 confirms) |
| Idea type wall | IMPLEMENTED | `ideas_page.dart:233-249` — flat responsive `Wrap` of posters |
| Convert idea to PRD | IMPLEMENTED (no error handling) | `ideas_page.dart:133-140` → `/prd/:id` |
| AI coach conversation over SSE (stages, suggestions, critic findings) | IMPLEMENTED | `api/sse.dart:116-148` handles all five frame types; `coach_panel.dart:147-150,171-190` |
| PRD desk editing + eight sections | IMPLEMENTED | `prd_desk.dart:87-111`; exactly 8 keys in `models.dart:251-260` |
| Roadmap canvas with presence + remote cursors | IMPLEMENTED | `roadmap_page.dart:56-95,193-203`; `roadmap_canvas.dart:83-107` |
| Now / Next / Later bands | IMPLEMENTED | `roadmap_bands.dart:38-86` |
| Time-first Progress view with deadlines/pressure | IMPLEMENTED | `roadmap_progress_view.dart:15-53`, fed by server truth + SSE; deadline editing in `roadmap_node_card.dart:203-224` |
| Human Commit + versions stack + readiness | IMPLEMENTED | `versions_page.dart:44-65`; `versions_stack.dart:150-201` (readiness gaps, diff counts, carry-into-commit) |
| Audience views (decision / management / delivery / validation) | PARTIAL | `audience_view_page.dart` — functional, but raw English enum chips at line 55 |
| PRD export (markdown / print) | IMPLEMENTED | `export_page.dart:49-99` — markdown download + print-HTML-to-tab with fallback |
| Evidence attach + GitHub sync display | PARTIAL | `project_page.dart:191-287` — manual sync, done-leaves only, cap 5, silent failure |
| Workspace invitations + member roles | IMPLEMENTED | `settings_page.dart:101-131`; join-by-code at `dashboard_page.dart:56-75` |
| Admin signup-mode + invitation ledger | IMPLEMENTED | `admin_section.dart:38-67,139-148,169-197` (three modes, used/revoked states, revoke) |
| Dark mode | IMPLEMENTED | `appearance.dart:46,67-71`; light/dark themes wired at `app.dart:22-23,40` |
| zh-CN / en-US locale switch | IMPLEMENTED (with Material-locale caveat) | `appearance.dart:61-65`; `app.dart:100-104`; but no `locale:` on MaterialApp |
| Magic-link / code / password auth | IMPLEMENTED | `auth_form.dart:11,77-107`; `magic_link_page.dart:30-37`; forgot/reset pages present |

## What works and is worth protecting

The coaching pipeline is the deepest implementation in the client and the hardest to get
right: `readCoachSse` parses multibyte-safe SSE frames and dispatches `delta`,
`prd_updated`, `stage_changed`, `fact_recorded`, and `critic_findings` to typed handlers
(`api/sse.dart:116-148`), with an eager 401-refresh wrapper that mirrors the dio interceptor
policy for the raw-fetch web path (`sse.dart:167-190`). The PRD desk reflects streamed
section updates back into the section controllers (`prd_page.dart:179-186`). The roadmap
runs two independent SSE loops (progress + presence) with capped exponential backoff and a
200ms-throttled cursor write, and the canvas renders labeled remote pointers with dead-session
cleanup. The Commit flow honors the "human authority stays explicit" principle in PRODUCT.md:
the dialog carries unresolved readiness gaps into the commit, the stack renders per-audience
readiness with blocking gaps called out, and nothing auto-commits. These are the features the
neo-brutalist pivot must not break.

## Where the implementation falls short of the advertisement

**The idea wall's primary path shows a raw English enum.** Quick capture always stores empty
content (`ideas_page.dart:63`), and the poster body falls back to `idea.status` verbatim
(`ideas_page.dart:326`) — so every one-line idea, the README's very first web-loop step,
displays "inbox" instead of 收集箱 even though the localized key exists (`zh.dart:35`). The
same two paths (`_captureNow`, `onConvert`) have no try/catch: a failed capture or convert is
a silent no-op on the highest-traffic screen, while `project_page.dart:86-101` demonstrates
the correct `localizedError` snackbar pattern two features away.

**Evidence over-promises and under-delivers.** README.md:18-19 says commits, PRs, and
deployments "attach themselves to roadmap nodes as evidence". The web client offers a manual
立即同步 button, displays evidence only for up to five done leaf nodes
(`project_page.dart:63` `.take(5)`), and swallows every failure into an empty page
(`project_page.dart:72-74`). On the page whose entire pitch is "progress you can verify,
not claims you have to trust", a failed token or a sixth done item is indistinguishable from
"GitHub binding is broken". There is also no human attach/correct affordance, which the
evidence trust layer in PRODUCT.md:184-195 ("small actions to confirm, correct, remove, or
discuss") will eventually require.

**zh-CN leaks.** The audience switcher on the reader-view page and the export dropdown render
raw `Text(a)` enum labels (`audience_view_page.dart:55`, `export_page.dart:67`) while the
versions page right next door localizes the same four values correctly
(`versions_stack.dart:47`). Worse, neither `MaterialApp` receives `locale:` or
`supportedLocales:` (`app.dart:28-42`), so the deadline date picker — on the documented
deadline-setting path — renders English month names and CANCEL/OK to a zh-first user, and
`admin_section.dart:103` hardcodes an always-English close button via a
`MaterialLocalizationsNo` workaround instead of fixing the root cause.

**The coach's error and critique contracts are partial.** The panel flattens `ApiException`
to `e.message` before calling `localizedError` (`coach_panel.dart:95-99`), which only
handles exception objects — so a 429 becomes a generic apology — and then distinguishes
"service unavailable" from "rate limited" by sniffing the English substring "unavailable"
(line 199). The fact-recorded snackbar interpolates a raw `Map` (line 86). Critic findings
render only the issue string under a raw 'critic' label, discarding the severity and
recommendation fields the frame carries, and the list is never cleared, so findings from an
earlier stage shadow later conversation (lines 89-93, 188-190). PRODUCT.md:139-141 promises a
severity-ranked overview handled individually; the UI currently shows an unranked, immortal
quote list.

**Bands fallback contradicts Now semantics.** When server progress truth is absent,
`RoadmapBands.bucket` files `done` work into the Now band (`roadmap_bands.dart:38-44`),
which is exactly what PRODUCT.md:53-54 says cannot be: done work is not Now work. The
Progress tab agrees with the server; the Bands tab, briefly, does not.

**Smaller but visible.** Project updates render raw RFC3339 timestamps (`project_page.dart:307`;
screenshot 07 confirms "2026-10-06T00:31:05+08:00") in a codebase that ships `timeAgo` and
`isoDate`. The versions and dashboard pages render raw exception text as whole-page content
(`versions_page.dart:73`, `dashboard_page.dart:98`), bypassing the `localizedError` standard
the other pages follow. The landing canvas — the product's evidence-pitch storefront — shows
fake commit hashes containing non-hex characters ('e4f5g6h', 'i7j8k9l',
`step_peek.dart:220`) and a dead duplicated branch (`assembly_diagram.dart:167-170`). And
`ideas_page.dart` carries its own `ShadePill` (lines 348-366) duplicating the shared
`cobalt_chip.dart` `ShadeChip`, which means the incoming neo-brutalist pivot would migrate
chips twice unless this is collapsed first.

## Features to stop advertising until fixed

1. **README.md:18-19 — "commits, PRs, and deployments attach themselves to roadmap nodes as
   evidence"**: the web client is manual-sync-only, shows at most five done leaves, and
   silently empties on failure. Reword to "sync on demand" (or ship auto-sync plus a real
   evidence surface) before the next release notes repeat the claim.
2. **PRODUCT.md:138-141 critique handling ("severity-ranked overview … recommendation and a
   keep-as-is option")**: the frontend renders issue text only, unranked and un-clearable.
   Either scope this as target-only explicitly or land the severity rendering; as written it
   reads like shipped behavior.
3. **The idea-privacy hint (`idea.privateHint`, visible in screenshot 08: "仅自己可见——想法默认
   只有作者能看到")**: the frontend cannot enforce or verify this — it depends entirely on
   server-side filtering of `listIdeas`. Keep the copy only if a test proves a non-author,
   non-admin member cannot see another author's inbox ideas; otherwise soften the hint.

Everything else advertised — the eight-section PRD desk, coach streaming with stages,
presence and cursors, Now/Next/Later plus the server-truth Progress view, the human Commit
with readiness, audience projections, markdown/print export, invitations and the admin
ledger, dark mode, the locale switch, and all three auth ways — is implemented and safe to
keep advertising. The recommended order is to fix the ideas-wall pair first (FEAT-01/02),
since that is the first screen of the advertised loop, then the evidence surface, then the
zh-CN leaks, and to fold the component dedup (FEAT-12) into the pre-pivot checklist so the
neo-brutalist migration touches each component exactly once.
