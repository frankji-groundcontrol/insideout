# 20 · Feature Implementation

**Lens:** Advertised features vs implementation status (IMPLEMENTED/PARTIAL/DEAD/DOCS-ONLY)
**Score:** **7.2 / 10**
**Bottom line:** The advertised web feature set is overwhelmingly real and traceable to code: all sixteen checked features are IMPLEMENTED or PARTIAL, none are DEAD or DOCS-ONLY at the page level. The core loop (capture → convert → coach → PRD desk → roadmap → commit → export) has a working implementation for every step. What separates this from a clean bill is the last mile of each feature: the idea wall's primary one-line capture path renders a raw English status enum, evidence display silently empties on any failure and covers only 5 done leaves against a README promise of automatic attach, audience pickers leak raw enum English into the zh-CN UI, the coach's critic findings drop severity and never clear, and Material chrome stays English because MaterialApp never receives a locale. All fixable in small diffs before the neo-brutalist pivot.

Reviewed branch `main` at `f185e94`; a skeptic pass re-checked every Critical/High finding — corrections marked [Correction]. Screenshot evidence lives in `./shots/`.

---

## Scope

Every frontend-facing claim in `README.md` and `PRODUCT.md` was checked against the implementing code in `client/lib`, with the live screenshots under `shots/` consulted where useful. Sixteen advertised features were graded IMPLEMENTED / PARTIAL / DEAD / DOCS-ONLY. Verdict first: the advertised feature set is real — nothing advertised is DEAD or DOCS-ONLY; the gaps are in the last mile of otherwise-working features, and nearly all are small diffs.

## What is good

- Coach SSE is fully wired end-to-end with frame-level handling for `delta`, `prd_updated`, `stage_changed`, `fact_recorded`, and `critic_findings`, plus eager 401-refresh semantics (`client/lib/api/sse.dart:116-148,167-190`).
- The Progress view is rendered from server truth (GET /roadmap/progress plus a live SSE stream) with Now / needsDeadline / Next / Done lanes and pressure coloring, matching PRODUCT.md's time-first default (`client/lib/features/project/roadmap_progress_view.dart:15-53`, `client/lib/features/project/roadmap_page.dart:56-72`).
- The human Commit surface is complete: audience-scoped readiness with gaps, carry-into-commit of unresolved items, diff counts, and a fanned stack visualization (`client/lib/features/prd/versions_page.dart:44-65`, `client/lib/features/prd/versions_stack.dart:150-201`).
- Presence and remote cursors are genuinely live: dual SSE loops with capped backoff, throttled cursor writes (200ms), self-cursor filtering, and dead-session cleanup (`client/lib/features/project/roadmap_page.dart:49-95`, `client/lib/features/project/roadmap_canvas.dart:83-107`).
- Auth breadth is real: password, magic-link (with token-consuming landing page), 6-digit code with resend, forgot/reset, and invite-code prefill driven by the public signup-mode config (`client/lib/features/auth/auth_form.dart:11,77-107`, `client/lib/features/auth/magic_link_page.dart:30-37`).

## Findings, worst first

### feat-idea-status-raw-enum — High

Idea cards render the raw status enum as the card body whenever content is empty — and quick capture always stores empty content. The README's first web-loop step is "jot a one-line idea"; every quick-captured idea has empty content, so the wall displays the raw server enum ("inbox", English) as the card's body text in an otherwise zh-CN interface — on the highest-traffic screen of the product.

Evidence: `client/lib/features/workspace/ideas_page.dart:326`

```dart
Text(
  idea.content.isEmpty ? idea.status : idea.content, — ideas_page.dart:63: `await context.read<Session>().api.createIdea(widget.workspaceId, title, '');` The localized key exists but is never used here: zh.dart:35 'idea.status.inbox': '收集箱',
```

**Fix:** Render `l10n.t('idea.status.${idea.status}')` as the fallback (or capture a content snippet); one-line change at ideas_page.dart:326.

**Verdict:** CONFIRMED — ideas_page.dart:326 `idea.content.isEmpty ? idea.status : idea.content`; quick capture always passes `content:''` so cards render raw enum strings; the localized key exists unused at zh.dart:35.

### feat-capture-convert-no-error-handling — High

Idea capture and convert-to-PRD have no error handling — a failed call is a silent no-op with an unhandled async error. A network blip or 500 during "记下" or "转为 PRD" leaves the user with no feedback: the button appears dead, the idea vanishes from the field only on success, and the convert navigation simply never happens. Contrast `project_page.dart:86-101`, which surfaces the identical failure class via a `localizedError` snackbar.

Evidence: `client/lib/features/workspace/ideas_page.dart:60`

```dart
Future<void> _captureNow() async {
  final title = _capture.text.trim();
  if (title.isEmpty) return;
  await context.read<Session>().api.createIdea(widget.workspaceId, title, '');
  _capture.clear();
  await _reload();
} — ideas_page.dart:133-140 onConvert: (idea) async { final ids = await context.read<Session>().api.convertIdea(idea.id); ... } — no try/catch in either path.
```

**Fix:** Wrap both paths in try/catch and show a localizedError SnackBar, mirroring ProjectPage._decide; keep the field text on failure.

**Verdict:** CONFIRMED — subset of `full-write-actions-fail-silently` (semantic duplicate kept for traceability).

### feat-evidence-overpromised — Medium

Evidence display covers at most 5 done leaf nodes, swallows every failure silently, and the README advertises automatic attach while the UI only offers a manual sync click. A user binds a repo, pushes commits, and opens the Evidence page: nothing appears until they discover and click 立即同步, in-progress nodes never show evidence at all, a sixth done item is invisible, and any API failure renders as an empty page with no error — the exact opposite of the README's "progress you can verify, not claims you have to trust".

Evidence: `client/lib/features/project/project_page.dart:57`

```
project_page.dart:63: .take(5) — project_page.dart:72-74 `} on Exception { return const []; }` — sync is a manual `github.sync` button calling api.syncGithub (project_page.dart:211-232). README.md:18-19: "commits, PRs, and deployments attach themselves to roadmap nodes as evidence"
```

**Fix:** Surface the load error instead of returning const [], expand beyond take(5), and either ship auto-sync or reword README to "sync on demand".

**First check:** server webhook/auto-sync capability in server/internal.

### feat-audience-pickers-raw-enum — Medium

Audience pickers render raw English enum labels in both the audience view and the export page, despite localized keys existing and being used correctly on the versions page. A zh-CN user opening the 读者视图 or export page sees "decision / management / delivery / validation" raw chips — the only untranslated surface on those pages, and one of them is the page a boss (decision audience) would see.

Evidence: `client/lib/features/prd/audience_view_page.dart:55`

```dart
audience_view_page.dart:55: `label: Text(a),` — export_page.dart:67: `DropdownMenuItem(value: a, child: Text(a)),` — keys exist and are used properly at versions_stack.dart:47: `Text(l10n.t('versions.audience.$a'))`; zh.dart:250-253.
```

**Fix:** Reuse `l10n.t('versions.audience.$a')` in both pickers; two-line change.

### feat-material-locale-never-set — Medium

Neither MaterialApp sets `locale:` or `supportedLocales:`, so all Material-localized chrome (deadline date picker, dialog buttons) stays English in the zh-CN-first product. A zh-CN user sets a roadmap deadline via showDatePicker and gets English calendar chrome ("CANCEL / OK", month names) — the exact widgets used on the documented deadline-setting path.

Evidence: `client/lib/app.dart:36`

```dart
return MaterialApp.router(
  title: 'InsideOut',
  theme: light,
  darkTheme: dark,
  themeMode: appearance.themeMode,
  routerConfig: router,
); — admin_section.dart:103 even hardcodes an always-English close button: `TextButton(onPressed: () => Navigator.of(ctx).pop(), child: const MaterialLocalizationsNo()),`
```

**Fix:** Pass `locale: Locale(appearance.locale)` and `supportedLocales: [Locale('zh','CN'), Locale('en','US')]` to both MaterialApp instances; delete the MaterialLocalizationsNo workaround.

### feat-coach-error-contract-partial — Medium

The coach error path destroys the exception object before localization and picks retry copy by English substring sniffing; the fact-recorded snackbar interpolates a raw Map. A 429 from the coach shows the user a generic apology (wrong copy), a genuinely unavailable service is distinguished only if the English server text contains "unavailable", and every recorded fact pops a "{text: ...}" raw-map snackbar.

Evidence: `client/lib/features/prd/coach_panel.dart:95`

```dart
} on ApiException catch (e) {
  setState(() => error = e.message); — coach_panel.dart:194 `child: Text(localizedError(error!, l10n.t), ...)` — error is a String, and errors_l10n.dart:16-41 only handles ApiException/DioException objects, so a String always yields error.generic. coach_panel.dart:199: `l10n.t(error != null && error!.contains('unavailable') ? 'coach.serviceUnavailable' : 'coach.rateLimited', ...` — coach_panel.dart:83-88 onFactRecorded interpolates the whole `Map<String, dynamic> fact` into the snackbar.
```

**Fix:** Keep the ApiException in state and pass the object to localizedError; branch retry copy on a typed error code instead of substring; extract a human string field from the fact map.

### feat-critic-findings-incomplete — Medium

Critic findings render only the issue text under a raw 'critic' label, drop severity and recommendation, and accumulate forever across sends and stage changes. After the critic stage, the user sees unstructured one-line quotes with no severity and no way to act on them, and findings from earlier stages keep re-rendering under later answers — a partial implementation of the advertised critique contract (PRODUCT.md:139-141 promises a "severity-ranked overview … with a recommendation and a keep-as-is option").

Evidence: `client/lib/features/prd/coach_panel.dart:188`

```dart
for (final finding in findings)
  if (finding['issue'] is String) _Quote(text: finding['issue'] as String, role: 'critic', ink: ink), — coach_panel.dart:89-93: onCriticFindings only setState-assigns; nothing ever resets `findings`. PRODUCT.md:139-141 promises "severity-ranked overview ... with a recommendation and a keep-as-is option".
```

**Fix:** Clear findings at the start of each _send and on stage_changed; render severity (and recommendation when present) as chips above each finding.

### feat-bands-done-in-now — Medium

The client-side bucket fallback files done work into the "Now" band, contradicting both the bands' own labels and PRODUCT.md's Now semantics. During the window before the first progress snapshot loads (or after it errors), the Bands view shows completed work under "Now" — the one view meant to answer "where are we?" — and contradicts the Progress tab right next to it.

Evidence: `client/lib/features/project/roadmap_bands.dart:38`

```dart
static String bucket(RoadmapNode n) {
  return switch (n.status) {
    'in_progress' || 'done' => 'now',
    'pending' => 'next',
    _ => 'later',
  };
} — this fallback runs whenever `progress` (server truth) is null.
```

**Fix:** Bucket 'done' into its own handling (count chip or a Done band) in the fallback, or gate the bands view on progress data being present.

### feat-raw-rfc3339-timestamps — Low

Project updates render raw RFC3339 timestamps while the codebase ships timeAgo/isoDate helpers used everywhere else. Every project update card shows an ISO string to non-technical readers; a polish regression on the page decision-makers visit.

Evidence: `client/lib/features/project/project_page.dart:307`

```dart
Text(u.createdAt, style: theme.textTheme.bodySmall), — screenshot shots/07-project-dark.png shows "2026-10-06T00:31:05+08:00" on the Evidence page. Contrast ideas_page.dart:332 (`timeAgo(idea.createdAt, l10n.t)`) and roadmap_progress_view.dart:113 (`isoDate(item.deadline!)`).
```

**Fix:** Wrap with `isoDate(u.createdAt)` (or timeAgo); one-line change.

### feat-landing-fake-hashes-dead-arm — Low

The landing page's "shipped" preview shows fake commit hashes that are not valid hex, alongside a dead duplicate branch in the landing diagram. For a product whose entire pitch is verifiable evidence, the marketing canvas displays impossible hex strings ('g', 'h', 'i', 'j', 'k', 'l') as commit hashes — a detail a technical visitor will notice on the first screen.

Evidence: `client/lib/features/landing/step_peek.dart:220`

```dart
const hashes = ['a1b2c3d', 'e4f5g6h', 'i7j8k9l'], — assembly_diagram.dart:167-170 `if (i == currentStep - 1) return ink;` / `if (i < currentStep - 1) return ink;`
```

**Fix:** Use plausible hex samples ('a1b2c3d', 'e4f5a6b', 'c7d8e9f') and merge the identical if-arms.

### feat-versions-dashboard-raw-error — Low

The Commit surface and dashboard render raw exception text on load failure, bypassing the localizedError standard used by every other page. A zh-CN user hitting a failed versions load sees raw English Dio/HTTP text as the entire page content on the most authority-sensitive screen of the product.

Evidence: `client/lib/features/prd/versions_page.dart:73`

```dart
if (snap.hasError) return Center(child: Text('${snap.error}')); — dashboard_page.dart:98: identical pattern; the project standard is `Text(localizedError(snap.error!, l10n.t))` (ideas_page.dart:125, roadmap_page.dart:211).
```

**Fix:** Route both through localizedError; two-line change.

### feat-shadepill-duplicates-theme-chip — Low

A feature file carries its own duplicated theme widget (ShadePill) instead of the shared `cobalt_chip.dart` ShadeChip, doubling the component surface the design pivot must migrate. When the neo-brutalist pivot re-skins chips (2px borders, square corners, tint fills), ShadePill will be missed or migrate inconsistently, reintroducing rounded pills on the ideas wall.

Evidence: `client/lib/features/workspace/ideas_page.dart:348`

```
ideas_page.dart:348-366 defines `class ShadePill`; theme/cobalt_chip.dart already ships `ShadeChip` used by project_page.dart:304 and step_peek.dart.
```

**Fix:** Delete ShadePill and use ShadeChip (or a cobalt.dart token-driven variant) before the pivot starts.

## Lead verification

Advertised-feature verification matrix — each README/PRODUCT.md claim graded against the implementing code:

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

Claims to stop advertising until fixed: README.md:18-19 ("commits, PRs, and deployments attach themselves to roadmap nodes as evidence" — manual sync, cap 5, silent failure); PRODUCT.md:138-141 critique handling (renders issue text only, unranked and un-clearable); the `idea.privateHint` copy (frontend cannot enforce or verify it — depends entirely on server-side filtering of `listIdeas`; keep only if a test proves non-author, non-admin members cannot see another author's inbox ideas).

## Recommendations

- Fix the ideas-wall card body to use the existing idea.status.* l10n keys, and stop passing empty content from quick capture (`ideas_page.dart:63,326`).
- Wrap idea capture and convert-to-PRD in try/catch with localizedError snackbars, matching the ProjectPage._decide pattern.
- Surface evidence-load errors on the project page, expand past take(5), and either ship auto-sync or reword the README's "attach themselves" claim to "sync on demand".
- Localize every audience picker via versions.audience.* (`audience_view_page.dart:55`, `export_page.dart:67`) and pass locale plus supportedLocales to both MaterialApp instances.
- Harden the coach panel: keep the ApiException object for localizedError, replace the English "unavailable" substring sniff with a typed code, extract a display string from fact maps, clear findings per send, and render severity chips.
- Bucket done work out of the Now band in RoadmapBands' client-side fallback, or gate the bands view on server progress data.
- Format project update timestamps with isoDate/timeAgo (`project_page.dart:307`) and route versions/dashboard error branches through localizedError.
- Before the neo-brutalist pivot, delete the duplicated ShadePill and merge the dead diagram arms so each component migrates exactly once.

## What this role would do first

Fix the two ideas-wall Highs together in `client/lib/features/workspace/ideas_page.dart` — render the localized status fallback at line 326 and wrap `_captureNow`/`onConvert` in try/catch with localizedError snackbars (lines 60-66, 133-140) — since they sit on the first screen of the README's advertised web loop and are each one-to-five-line diffs.
