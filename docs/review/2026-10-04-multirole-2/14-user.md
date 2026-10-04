# 14 · User (seat)

**Lens:** 
**Score:** **6.5 / 10**
**Bottom line:** The front door finally works (invite deep link, 401 recovery, board promises kept, privacy hint truthful, real export file, Progress default, all five new keys in both locales) — but the walls are the failure paths: Drop never visibly drops, the coach SSE transport sidesteps the 401 refresh so mid-coach expiry loops on 'stream failed', every page-level error is untranslated English server text, and half the mutating actions fail with no signal.

Reviewed main @ 26db979; a skeptic re-checked every High finding from this review. USER-09, USER-10 and USER-11 were all confirmed High by the skeptic; each was independently reproduced at code level by the coordinator.

---

This review was persisted findings-only; the section below is assembled verbatim from its structured findings (the prose wrapper was not retained). 12 findings as filed, worst first.

## Findings

### USER-09 · High ·  **Skeptic: CONFIRMED.**

**Drop never visibly drops: no confirmation dialog (the translated idea.dropConfirm string is dead), the soft-deleted card stays on the wall forever, and an empty-content card then displays the raw status 'dropped'.**

- Where: `client/lib/features/workspace/ideas_page.dart:107`
- Evidence: ideas_page.dart:107-110 onDrop with no dialog; en.dart:194 'idea.dropConfirm': 'Drop this idea?' referenced nowhere; store ListIdeas has no status filter; ideas_page.dart:304 renders the literal 'dropped'.
- Fix: Exclude status='dropped' from ListIdeas (or filter client-side), confirm with the existing idea.dropConfirm key, and localize the dropped state.
- Diff size: ~25 lines + tests

### USER-10 · High ·  **Skeptic: CONFIRMED.**

**The coach's SSE POST bypasses the Wave-1 401 auto-refresh entirely: on web it is a raw fetch with the token snapshotted at send time, so an access token expiring mid-coaching produces a permanent 'stream failed' loop until a full page reload.**

- Where: `client/lib/api/client.dart:450`
- Evidence: client.dart:451-460 token snapshot; sse_transport_web.dart:23-24 throw ApiException('stream failed') with no refresh attempt.
- Fix: Catch 401 in sendCoach/fetchSseStream, await the same single-flight refresh, retry once with the rotated token, else clearTokens so the router lands on the auth door.
- Diff size: ~50-70 lines

### USER-11 · High ·  **Skeptic: CONFIRMED.**

**Carried USER-07, still open after five waves: every FutureBuilder error branch and the coach/admin panels render raw English server text as page content in an otherwise fully zh-CN UI.**

- Where: `client/lib/features/workspace/workspace_page.dart:86`
- Evidence: workspace_page.dart:86 snap.hasError ? Text('${snap.error}') — identical at 12+ call sites (ideas, prd, export, versions, project, roadmap ×2, dashboard, settings, coach_panel, admin_section).
- Fix: One l10n-aware error mapper (ApiException code/status → key, generic fallback) routed through all error branches.
- Diff size: ~120-180 lines

### USER-12 · Medium · 

**Export lets you combine the Print format with an audience projection, a combination the server rejects with a raw English 400; the audience dropdown is never disabled or reset on format switch.**

- Where: `client/lib/features/prd/export_page.dart:60`
- Evidence: export_page.dart:48-69 dropdown live for print; prds.go:301 400 'audience projections export as markdown (format=markdown) for now'.
- Fix: Disable or auto-clear the audience dropdown when format == 'print'; relabel the run button from common.save to a generate verb.
- Diff size: ~10 lines

### USER-13 · Medium · 

**The coach clears the draft before the send completes, so any failure — network blip, backoff, expired token — destroys the member's typed message with no way to recover it.**

- Where: `client/lib/features/prd/coach_panel.dart:72`
- Evidence: coach_panel.dart:72 'draft.clear();' before the awaited sendCoach at :74-88; the error path never restores the draft.
- Fix: Capture the text; clear only after success, or restore in the catch. ~5 lines.
- Diff size: ~5 lines

### USER-14 · Medium · 

**Most mutating actions have no error handling or feedback at all: capture, convert-to-PRD, PRD save, repo set, GitHub sync, version commit all fail silently when the API throws.**

- Where: `client/lib/features/workspace/ideas_page.dart:55`
- Evidence: ideas_page.dart:55-61 no try/catch; ideas_page.dart:111-118; prd_page.dart:78-86; project_page.dart:97-102; versions_page.dart:52-60.
- Fix: Shared runGuarded helper with localized success/failure snackbar around each mutation.
- Diff size: ~60-100 lines

### USER-15 · Medium · 

**The workspace invite card copies only the bare join code — not a link — and nothing explains that under invitation-only signup teammates first need a site-level invitation minted on a different surface (Profile → Admin); no /join?code= deep link.**

- Where: `client/lib/features/workspace/workspace_page.dart:99`
- Evidence: workspace_page.dart:99-107 copies ws.code; site invitations live only in admin_section.dart:96-101; router has no join route.
- Fix: Copy a composed URL (or code + one-line hint pointing to Profile → Admin); consider a /dashboard?join= pre-fill.
- Diff size: ~15 lines

### USER-16 · Medium · 

**USER-01's doc half landed but the UI half did not: a wrong or expired invitation code still yields the generic 'Could not create account. Please try again.' — advice that cannot succeed.**

- Where: `client/lib/features/auth/auth_form.dart:103`
- Evidence: auth_form.dart:71-75 _errorKey maps every register failure to one key; en.dart:110 'register.error'.
- Fix: Branch on the 403/invalid-invite response to a distinct register.inviteError key.
- Diff size: ~10 lines

### USER-17 · Medium · 

**Raw English enum values leak into the zh UI in four spots: export audience options, project status dropdown, update-kind dropdown and chips, plus raw ISO timestamps where sibling surfaces use timeAgo.**

- Where: `client/lib/features/prd/export_page.dart:65`
- Evidence: export_page.dart:65-66 Text(a); project_page.dart:198-199 Text(s); :259 Text(k); :130 ShadeChip(label: u.kind), :133 raw createdAt; prd_desk.dart:71 raw updatedAt.
- Fix: Route each through the existing l10n keys (project.status.*, project.updateKind.*, versions.audience.*) and timeAgo.
- Diff size: ~15 lines

### USER-18 · Low · 

**Coach errors render in regular ink (indistinguishable from quotes) and the localized retry message is chosen by substring-matching the English server message.**

- Where: `client/lib/features/prd/coach_panel.dart:179`
- Evidence: coach_panel.dart:179 TextStyle(color: ink); :184 error!.contains('unavailable') picks the message.
- Fix: Use the error color; decide serviceUnavailable vs rateLimited from ApiException.code.
- Diff size: ~6 lines

### USER-19 · Low · 

**The privacy hint promises 'workspace admins excepted', but no surface ever shows an admin other members' ideas — ListIdeas is author-only even for admins.**

- Where: `client/lib/features/workspace/ideas_page.dart:205`
- Evidence: ideas_page.dart:205 renders idea.privateHint; store ListIdeas filters author_id = $2 for every actor; GetIdeaForMember is the only admin-visible path.
- Fix: Add an admin 'all ideas' toggle (separate store query + UI) or soften the hint.
- Diff size: ~20 lines or 1 line

### USER-20 · Low · 

**When a PRD has no linked conversation the coach panel shows a dead-end sentence with the composer hidden and no way to start a conversation from the UI.**

- Where: `client/lib/features/prd/coach_panel.dart:152`
- Evidence: coach_panel.dart:152-156 noConversation text; :188 guards the input row; no create-conversation call anywhere in the client.
- Fix: Add a 'Start coaching' button POSTing a create-conversation endpoint (or lazy-create server-side on first send).
- Diff size: ~15 lines

