# 11 · Feature completeness: the Flutter client

**Lens:** every route, control, state and string in the web app
**Score:** **6.2 / 10**
**Bottom line:** The Flutter client delivers the single-author loop (capture, convert, coach, commit, build, roadmap, evidence) with disciplined loading, empty and error states, full en/zh key parity (417 keys each) and both themes. It is not complete for a team: the coach transcript does not update during a session, a teammate or approving admin has no path to a PRD they did not author, no surface can set or show a project's owner, there is no per-person task tree, and every member is shown write controls that only an owner or admin can use.

Read at `main` @ `4d186da` together with the uncommitted working tree. A skeptic re-checked every High in this file; the verdicts are in the table near the end and in the [findings register](./01-findings-register.md).

---

## Scope read

I read `client/lib/router.dart`, all of `client/lib/api/`, `client/lib/session/`, `client/lib/chrome/`, the l10n tables, and the auth, dashboard, workspace, profile, project and roadmap pages line by line. In `client/lib/features/prd/` I read the PRD page, desk actions, coach panel, versions, revisions, snapshot and export files in full, and skimmed `prd_desk.dart`, `versions_stack.dart`, `audience_view_page.dart`, `home_field.dart` and the landing files for API calls and state handling only. I compared the client against the route registrations in `server/internal/api`, and read `docs/architecture/frontend.md`, the README feature list, `docs/usage/user-guide.md` and the uncommitted reset-password work. I did not run the Flutter tests or the app.

## What is good

- One read gate serves every data page (`client/lib/theme/block_state.dart:156`): loading, a localized error and a retry button are uniform.
- One write guard (`client/lib/widgets/guarded_action.dart:18`) turns a failed write into a localized message and keeps the typed input.
- The locale tables are at exact parity: 417 keys in English and 417 in Chinese, none one-sided, none undefined.
- Both themes are built from tokens (`client/lib/app.dart:26-27`); no page hard-codes a colour.
- The PRD desk already mirrors server authority (`client/lib/features/prd/prd_desk_actions.dart:121-123`).
- Every one of the 76 client calls has a server route.

## Findings, worst first

| ID | Sev | Where | Deciding line | Fix |
|---|---|---|---|---|
| FE-01 | High | `client/lib/features/prd/coach_panel.dart:112` | `stream = '';` and the page refreshes only `getConversation` at turn end | Re-read messages when a turn ends and show the sent text at once |
| FE-02 | High | `client/lib/features/workspace/ideas_page.dart:177` | `context.go('/prd/${idea.prdId}');` is the only way into a PRD, and ideas list to their author only | Add a workspace PRD list route and page section, and a PRD link on the project page |
| FE-03 | High | `client/lib/router.dart:43` | `'/projects/:id/roadmap': (c, s) =>` is the only roadmap route | Node route, Mine filter, Open as tree; recorded in `docs/issues/2026-10-11-focus-surface.md` |
| FE-04 | High | `client/lib/features/project/project_page.dart:271` | `ownerId: data['ownerId'] as String?,` | Owner select in the project edit dialog; show the owner |
| FE-05 | Medium | `client/lib/features/workspace/workspace_page.dart:220` | Delete workspace button with no role test | Gate workspace, settings, project and node controls on role |
| FE-06 | Medium | `client/lib/features/auth/reset_password_page.dart:67` | `error = l10n.t('reset.error')` for every failure | Length check, status-aware copy, widget test |
| FE-07 | Medium | `client/lib/features/profile/profile_page.dart:140` | page ends; `DELETE /api/v1/me` has no caller | Password-confirmed delete block on the profile page |
| FE-08 | Medium | `client/lib/chrome/loop_focus.dart:35` | `int ideasPulse = 0;` has no listener | Listen in the ideas page; test the visible effect |
| FE-09 | Medium | `client/lib/features/prd/prd_page.dart:140` | `result = await api.confirmBuildFromPrd(widget.id, tree);` with no review | Show the drafted tree before confirming |
| FE-10 | Low | `client/lib/features/prd/coach_column.dart:59` | `role,` printed raw | Localize role and evidence kind |
| FE-11 | Low | `client/lib/features/project/roadmap_node_card.dart:50` | `c.status == 'done'` only | Count validated as finished |
| FE-12 | Low | `docs/architecture/frontend.md:24` | a deployed hostname, shown here as `<host>` | Redact in six docs |

FE-01 is the most visible defect. During a session the panel shows the stored history loaded with the page plus one streaming buffer. The member's own message is never appended, and the buffer is emptied on the next send, so a three-turn exchange shows one reply and no questions until the page is reloaded.

FE-02 and FE-04 are the two gaps that make the product single-player. A PRD can be opened only from its author's idea card. The admin who alone may approve it has no list to find it in. A project's owner, the only non-admin who may write the roadmap, cannot be seen or changed on any surface.

FE-03 confirms the scout's reading from the user's seat. The roadmap is one page per project with three views of the whole tree. There is no way to open a node as its own tree, no filter to the signed-in person's nodes, and the owner is a line of text on a card.

## The uncommitted reset work

The reset page now offers two choices, reset here or copy a code for the CLI, in both locales, and both paths work. It is not complete. Every failure reads as a dead link, including a password under eight characters, which the server refuses before it uses the token (`server/internal/api/auth_email_flows.go:139`). A link opened without a token shows a disabled button and no reason. No test names the page. The reset watch has no web part and needs none: using the link completes the watch on the server. The forgot-password page does ignore the new `mail` word, so the web still says a link is on its way when the server reports the mail was dropped.

## Server routes with no client call

Of 88 `/api/v1` routes, 12 have no web caller. Six belong to agents or operators and are not needed in the browser: `GET /agent/context`, `POST /agent/checkpoint`, `POST /agent/propose`, `POST /agent/token`, `GET /projects/{pid}/guide`, `GET /auth/reset-watch/{id}`. Two are replaced by other calls: `GET /projects/{pid}/presence` (the stream is used) and `GET /admin/settings` (the page reads `/auth/config`). Two a web user needs: `DELETE /me` (FE-07) and `POST /admin/mail/probe`. The webhook and `/healthz` are not user routes. No client call lacks a route.

## Lead verification

| Lead | Result | Deciding line |
|---|---|---|
| 2 | Confirmed | `server/internal/store/memberships.go:232` "requirePlanWriter allows the project owner or a workspace admin" |
| 3 | Confirmed | `server/internal/api/roadmap.go:17-27` registers no subtree or owner-scoped read |
| 4 | Confirmed | `client/lib/router.dart:43`; `roadmap_canvas.dart:74` is a layout helper; owner picker at `roadmap_node_menu.dart:152-161` |
| 5 | Extended | The same absence holds for the project owner: no owner parameter in the Go client, MCP tools or CLI |
| 6 | Confirmed by record | `docs/issues/2026-10-11-focus-surface.md` |
| 7 | Not re-run | The three uncommitted client files add no analyzer-relevant construct that I saw |
| 9 | Confirmed for the client | Largest client files are 349 lines (`roadmap_node_menu.dart`, `product_dock.dart`) |
| 13 | Not reproduced | I count 90 route patterns in `server/internal/api` (88 under `/api/v1`, `/healthz`, the webhook), not about 107 |
| 14 | Followed | FE-03 cites the open record; none of the other findings appears in the review-4 register or its open list |

## Recommendations

1. Repair the coach transcript (FE-01).
2. Add the workspace PRD list and the project-to-PRD link (FE-02), and keep the intended path through sign-in.
3. Add the project owner picker (FE-04), then the node route and Mine filter (FE-03) with the server decision on node-owner writes.
4. Gate write controls on role in four files (FE-05).
5. Finish the reset page before committing it (FE-06).
6. Add account deletion to the profile page (FE-07) and wire the ideas pulse (FE-08).
7. Smaller repairs: reload the timeline after a GitHub sync (`project_github_card.dart:62`), stop `listNodeEvidence` turning a 403 into an empty list (`client_project.dart:184-189`), and stop listing done work under Later (`roadmap_bands.dart:46`).

## What this role would do first

I would fix the coach transcript. The coach conversation is the product's central surface, the change is about 25 lines in two files, and a two-turn widget test would have caught it. Then I would add the PRD list, because without it the review step works only for a person reviewing their own document.

## Property checks

| Property | Status | Evidence |
| --- | --- | --- |
| Landing page with locale and theme toggles | implemented | client/lib/router.dart:26; toggles at client/lib/features/landing/landing_page.dart:99 and :114. Skimmed, not read line by line. |
| Login, register, magic link and sign-in code | implemented | client/lib/features/auth/auth_form.dart:105-135 calls login, register, requestMagicLink, requestLoginCode and loginWithCode with per-mode error keys; magic_link_page.dart:39 consumes the token. No locale or theme toggle on the auth door. |
| Forgot password page | partial | forgot_password_page.dart:50 calls forgotPassword and shows the mail-off notice; client_auth.dart:76-81 discards the response, so the new `watch` id and the `mail: dropped` word are ignored. |
| Reset password page with web and CLI choices (uncommitted) | partial | reset_password_page.dart:27 `enum _ResetStep { choose, web, cli, done }` works for both choices and both locales; every failure maps to reset.error (:67), an empty token gives a disabled button with no explanation (:181), and no test names ResetPasswordPage. |
| Reset watch in the web client | missing | No client call to GET /auth/reset-watch/{id} (grep `reset-watch` over client/lib is empty). By design it is a CLI feature; the browser's part is using the link, which marks the watch completed server-side. |
| Dashboard: list, create and join workspaces | implemented | dashboard_page.dart:32, :59, :84 with readGate at :116 and an empty state in home_field.dart:197. |
| Profile edit | implemented | profile_page.dart:59 updateMe with busy, error and saved states (:120-130). |
| Account deletion from the web | missing | profile_page.dart ends at :140; server route at server/internal/api/me_delete.go:11 has no client call (FE-07). |
| Admin section: signup mode and invitations | implemented | admin_section.dart:50-51, :69, :82, :140; shown only to admins (profile_page.dart:140). The ledger is cut at ten rows (:224); POST /admin/mail/probe has no control. |
| Workspace board: projects, invite code, create project | partial | workspace_page.dart:67-76 and :100; edit and delete workspace are shown to every member (:214-225) and there is no PRD list (FE-02, FE-05). |
| Ideas wall: capture, edit, drop, convert | implemented | ideas_page.dart:84-110, :133, :161-171 with confirm on drop and an empty state at :277; the dock pulse it is said to answer has no listener (FE-08). |
| Workspace settings and members | partial | settings_page.dart:83-121 wires save, delete, role change and remove; none is role-gated and remove member has no confirm (:112-120, :247-263). |
| Project (Evidence) page: edit, delete, add update | partial | project_page.dart:261-299; no owner control (:271), no PRD link, controls not role-gated. |
| GitHub card: link repo and sync | partial | project_github_card.dart:47-55 and :62-76; the sync branch never calls onChanged, so new commits do not appear until a reload. |
| Updates feed and agent proposal decisions | implemented | project_updates_feed.dart:57-74 single decision with busy and decided guards; edit and delete are offered on every update regardless of author (:131). |
| Roadmap Progress view | implemented | roadmap_page.dart:243-268 with live stream (:94-99) and distinct empty sentences; items are read-only cards (roadmap_progress_view.dart:130). |
| Roadmap Bands view | partial | roadmap_bands.dart:54-66 follows the server's Now and Next; everything else, including done and validated nodes, is listed under Later (:42-47, :63-66). |
| Roadmap Canvas with presence and cursors | implemented | roadmap_page.dart:101-124 and :300-327; retry loop with back-off at :78-92. |
| Node menu: edit, owner, status, deadline, acceptance, move, AI expand, observe, validate, delete | implemented | roadmap_node_menu.dart:22-41 and :254-268; every item calls a real route through guardWrite. Not role-gated (FE-05). |
| Per-person task tree: open a node as its own tree, my nodes, manager drill-in | missing | client/lib/router.dart:43 is the only roadmap route; no owner filter in roadmap_page.dart:36; recorded in docs/issues/2026-10-11-focus-surface.md (FE-03). |
| PRD desk: edit, save, status moves, build roadmap | partial | prd_page.dart:113-125 and prd_desk_actions.dart:113-128 with role-aware status menu; build confirms the AI draft unseen (prd_page.dart:136-140, FE-09). prd_desk.dart skimmed. |
| Coach panel: streaming chat, stage proposal, findings, back-off | partial | coach_panel.dart:120-169 streams and handles throttle and offline states; the transcript is not updated during a session (FE-01) and the role label is raw (FE-10). |
| Versions and human commit | implemented | versions_page.dart:37-51 and :69-80 with readGate at :90. versions_stack.dart skimmed. |
| Revisions and snapshot | implemented | revisions_page.dart:35, :54-80; snapshot_flow.dart:40-47. A snapshot taken from the PRD desk gives no success feedback. |
| Audience view | implemented | audience_view_page.dart:54 calls prdView and :69 uses readGate. Skimmed. |
| Export (Markdown and print) | implemented | export_page.dart:37-57 with running and error states and a copy fallback at :61-68. |
| Product dock | partial | product_dock.dart:176-210 disables items that lack an id; Capture and Ideas share one target and their on-wall tap is inert (FE-08); Version and Commit share one target. |
| Every English string has a Chinese counterpart and the reverse | implemented | Key census: 417 keys in client/lib/l10n/en_*.dart and 417 in zh_*.dart, no one-sided key, no undefined literal key. Two raw enums bypass the tables (FE-10). |
| Day and night themes honoured | implemented | client/lib/app.dart:26-27 builds both themes from Cobalt tokens; grep for colour literals outside client/lib/theme finds only Colors.transparent (4 uses). |
| Members without authority see controls disabled or hidden | partial | Done on the PRD desk (prd_page.dart:324 `canDecide: myRole == null \|\| myRole == 'admin'`); absent on workspace, settings, project and roadmap pages (FE-05). The workspace role set is admin and member only (server/db/migrations/20260720135751_workspaces.sql:25); no viewer role exists. |
| Every client call has a server route | implemented | 76 distinct client calls read from client/lib/api/client_*.dart and requests.dart; each matches a registration in server/internal/api (for example roadmap.go:17-27, prds.go:16-26, conversations.go:16-21). |

## Skeptic verdicts on this lens's Highs

| ID | Verdict | Reason | What the skeptic ran |
| --- | --- | --- | --- |
| FE-01 | CONFIRMED | Reproduced on the real PrdPage with a scripted API, two turns. After turn 1 the question is absent and only the streamed reply shows. After turn 2 the first question, the first reply and the second question are all absent; only the second reply shows. listMessages was called 0 times after page load; the turn-end hook called getConversation only. The transcript has two sources: widget.messages (assigned only at load and at _startCoach, prd_page.dart:87 and :194) and the stream buffer that _send resets. Coach chat is the core PRD flow, so wrong behaviour on the documented path holds at High. Not recorded under docs/issues (README and the review-4 open findings have no transcript entry). | cd client && flutter test /tmp/insideout-rv5/client-and-cli/coach_probe_test.dart -> `PROBE after turn 2: Q1=0 R1=0 Q2=0 R2=1 listMessages calls since load=0 getConversation=2` |
| FE-02 | CONFIRMED | Real, and the same root cause as PARB-03: this is its web framing and adds no separate harm, so merge the two. The only navigations to /prd/<id> outside the PRD pages are ideas_page.dart:170 and :177; the dock needs an id already on the route or remembered (product_dock.dart:182). The API probe shows the admin's ideas list does not carry the member's idea or PRD id, and no list route does. The admin can approve once the id is known (200), so the capability exists but cannot be discovered in the product. Approval gates the roadmap build, so the documented author-submits, admin-approves path cannot complete in-product when the author is not the admin. Not an open record under docs/issues. | go test ./internal/api/ -overlay /tmp/insideout-rv5/client-and-cli/overlay.json -run TestRV5ProbeAdminFindsPrd -v -> `admin GET /api/v1/workspaces/<ws>/ideas -> 200, names PRD id: false, names idea id: false`; `admin GET /api/v1/workspaces/<ws>/prds -> 404`; `admin GET /prds/<known id> -> 200` |
| FE-03 | CONFIRMED | The defect is real and High for the same reasons as TREEX-02 and TREEX-03, whose two claims it combines (single route, no owner filter); it adds no separate harm, and its last clause about refused edits is the backend authority gap that R1 measured. One statement in it is wrong: it says it restates the open record docs/issues/2026-10-11-focus-surface.md. That record is about a per-session Focus pointer and team Now aggregation and states 'Focus is a pointer, never ownership'; it does not describe a subtree route, an owner filter or a manager view. So the per-person tree has no open record and should be filed, not treated as already tracked. | grep -n roadmap client/lib/router.dart -> only line 43 is a route; grep -n -i 'owner\|mine\|filter\|userId' over roadmap_bands, roadmap_progress_view, roadmap_header, roadmap_canvas -> only the 'Owner:' text line in roadmap_progress_view.dart:148. Read docs/issues/2026-10-11-focus-surface.md in full. |
| FE-04 | DOWNGRADED to Medium | The facts hold: the edit dialog returns only title, description and status, the page re-sends the stored ownerId unchanged, no client model or string shows a project owner, and no CLI verb or MCP tool edits a project at all. The severity does not. The owner's stated requirement is about delegating nodes, not handing over projects, and neither PRODUCT.md nor docs/usage promises a project hand-over control, so this is not a promised or owner-required capability. The failure scenario also overstates the dead end: the probe shows a plain member can create their own project and write its roadmap, and that an admin's PATCH ownerId does hand write access over, so promotion to workspace admin is not the only route. What remains is a decided server capability (rv4-TEST-03, 2026-10-20: only an admin may give a project away) with no user surface, and an owner nobody can see, which will bite as soon as teams rely on project ownership for roadmap write access. That is Medium. | go test ./internal/store/ -overlay /tmp/insideout-rv5/tree-surfaces/overlay.json -run TestRv5TreeSurfacesProjectOwner -v -count=1 -> 'A member adds node to admin-created project: err=forbidden'; 'B admin hands project to member via store: err=<nil> owner_is_member=true'; 'C member adds node after handover: err=<nil>'; 'D plain member creates own project: err=<nil>'; 'E member adds node to own project: err=<nil>'. |

## Ordered recommendations

1. Repair the coach transcript first (FE-01); it is a small change to the core feature and needs one two-turn widget test.
2. Give teammates a way into PRDs: a workspace PRD list with status and a PRD link on the project page (FE-02), and keep the intended path through sign-in so a shared link survives login (client/lib/session/auth_redirect.dart:17 always sends a signed-out visitor to /login and auth_form.dart then goes to /dashboard).
3. Make ownership usable before building the per-person tree: add the project owner picker (FE-04), then a node route and a Mine filter (FE-03), together with the server decision on whether a node owner may write inside their subtree.
4. Derive two booleans per page, canAdminWorkspace and canWritePlan, and gate the workspace, settings, project and node-menu controls with them (FE-05).
5. Finish the uncommitted reset page before it is committed: length check, status-aware error copy, an empty-token message and a widget test (FE-06). Also read the `mail` word the server now returns from forgot-password, magic-link and login-code so the web says when a mail was dropped, as the CLI does.
6. Add the account-deletion block to the profile page (FE-07) and wire the ideas-wall pulse (FE-08).
7. Smaller repairs: reload the timeline after a GitHub sync (project_github_card.dart:62 never calls onChanged), run listNodeEvidence through _parse (client_project.dart:184-189 turns a 403 or 404 into an empty list), stop listing done work under Later (roadmap_bands.dart:46), and say when the invitation ledger is cut at ten rows (admin_section.dart:224).
