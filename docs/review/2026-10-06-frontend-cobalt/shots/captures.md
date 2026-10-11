# InsideOut Web Client — Visual Capture Index

- **Date:** 2026-10-06
- **Build:** release Flutter web (CanvasKit) served at `http://127.0.0.1:5174/` from `client/build/web`
- **Driver:** `scripts/sim/web_driver.py` (headless Chrome, CDP, vision-driven clicks)
- **Viewport:** 1440×900 (desktop shots), 390×844 @2x (narrow shots)
- **Method notes:** Chrome was launched *without* the driver's usual `--disable-remote-fonts` flag so CanvasKit could fetch Noto Sans SC and render CJK (the boot-gate timeout fix from [BUG-016](../../../issues/2026-10-06-bug-016-web-boot-gate-hangs-on-font-load.md) keeps the app booting). Demo login used: `demo@insideout.local`. In-app navigation only — direct deep links 404 (see anomaly A9). Narrow shots were taken with CDP mobile emulation; taps via `Input.dispatchTouchEvent`.

*Corrected 2026-10-11: the deep-link 404 anomaly is **A8**, not A9 (A9 is the
coach-panel top-bar contrast issue); the same mis-tag recurs in the Console
section below.*

## Shot index

| File | Theme | What it shows |
|---|---|---|
| `01-landing-logged-out.png` | light | Landing hero, logged out: 把想法，一件件打磨出来。+ 开始打磨 / 登录 buttons, step-1 teaser |
| `02-login-modal.png` | light | Login modal over dimmed landing (邮箱/密码, 登录, 邮件链接/验证码, 忘记密码?/还没有账号?) |
| `03-register-modal.png` | light | Register modal (邮箱/昵称/密码, 至少 8 个字符, 创建账号, 已经有账号了?) |
| `04-dashboard-light.png` | light | /dashboard after login: 这一圈 tour chapter card + invite/workspace cards, bottom dock |
| `05-dashboard-dark.png` | dark | Same dashboard in dark theme |
| `06-workspace-dark.png` | dark | /workspace/<id>: invite-code banner, 想法/设置/编辑/删除 rows, Onboarding Revamp project card |
| `07-project-dark.png` | dark | /projects/<id>: 证据 heading, GitHub 同步 card, 进展 feed, 发布进展 |
| `08-ideas-wall-dark.png` | dark | /workspace/<id>/ideas: capture input, one idea card (IN-APP CHANGELOG WIDGET, PRD chip) |
| `09-prd-desk-dark.png` | dark | /prd/<id>: PRD editor sections + PRD 教练 coach panel (dark navy) |
| `10-roadmap-dark.png` | dark | /projects/<id>/roadmap: 进展/色带/画布 tabs, empty state |
| `11-versions-dark.png` | dark | /prd/<id>/versions (dock 提交 destination): audience tabs 验证/决策/管理/交付, readiness checklist |
| `12-workspace-light.png` | light | Workspace page, light theme (same content as 06) |
| `13-project-light.png` | light | Project page, light theme (same content as 07) |
| `14-ideas-wall-light.png` | light | Ideas wall, light theme |
| `15-prd-desk-light.png` | light | PRD desk + coach panel, light theme (coach panel stays navy) |
| `16-roadmap-light.png` | light | Roadmap, light theme |
| `17-versions-light.png` | light | Versions/readiness page, light theme |
| `18-profile-light.png` | light | /profile: 个人资料 form + 管理 invite section (最近邀请 list) |
| `19-dashboard-narrow.png` | light | Dashboard at 390×844 (780×1688 px image) |
| `20-ideas-wall-narrow.png` | light | Ideas wall at 390×844 (780×1688 px image) |
| `21-narrow-resize-gutter.png` | light | Evidence shot: white gutter after shrinking viewport post-boot (see A11) |

## Dock walk results (click each icon, 1440×900)

| Dock item | Result |
|---|---|
| 记下 (+) | Navigates to `/workspace/<id>/ideas` (same as 想法) |
| 想法 | Navigates to ideas wall |
| 版本 | From project roadmap → `/prd/<id>/versions`; from /prd desk → dead click; from roadmap (light pass) → landed on /prd desk instead. Inconsistent (A7) |
| 提交 | Navigates to `/prd/<id>/versions` (same page as 版本; no separate commits page exists) |
| 路线图 | Navigates to `/projects/<id>/roadmap` |
| 证据 | Dead click from project roadmap — no navigation, no feedback (A7) |

*Corrected 2026-10-11: the dead/inconsistent dock clicks tagged (A7) in this
table are **A6** in the anomaly log; A7 is the empty workspace rows.
12-first-run-user.md quotes the mis-tagged row verbatim, so that quote keeps
the original tag.*

## Console (after walk, `web_driver.py console` = reload + 20 s drain)

- Only one message: `[error] Failed to load resource: 404` for `/workspace/<id>/ideas` — the reload of a deep client route hits the static server's 404 (see A9).
- No JS exceptions, no Flutter errors, no failed API calls.

## Anomaly log

Coordinates are in the image's own pixels (1440×900 desktop shots; 780×1688 narrow shots).

- **A1 — Stray boot-splash overlay above auth modals.** A cobalt disc + "InsideOut" wordmark floats on top of the dimmed scrim, behind/above the login and register modals. Looks like a leftover `BootLoadingPage` layer that never hid once a modal opened over the landing. `02-login-modal.png` ≈(718,255), `03-register-modal.png` ≈(718,255).
- **A2 — Bottom dock overlaps page content on every screen.** The centered floating dock pill sits on top of content everywhere: dashboard chapter-card paragraph clipped mid-sentence (`04`/`05` ≈(590,440)); workspace 删除这个工作区 row half-hidden (`06`/`12` ≈(720,443)); project GitHub 同步 card bottom edge (`07`/`13` ≈(720,440)); PRD desk document body (`09`/`15` ≈(720,443)); versions checklist lines 约束与依赖 / 非目标 obscured (`11`/`17` ≈(720,440)). Systemic — pages don't reserve bottom space for the dock.
- **A3 — Raw ISO-8601 timestamps shown to users.** Project 进展 entry shows `2026-10-06T00:31:05+08:00` (`07`/`13` ≈(120,588)); PRD desk shows `v1 · 2026-10-06T00:31:05+08:00` (`09`/`15` ≈(120,232)). Elsewhere the app uses friendly relative time ("8 小时前", `06`/`12`), so formatting is inconsistent.
- **A4 — Project page heading reads "证据".** The /projects/<id> page leads with a giant "证据" (Evidence) display heading above the actual project name, with the subtitle "Git 与项目更新落在同一块场地上。" — reads like a wrong or leftover title for a project overview (`07` ≈(75,95), `13` ≈(75,95)).
- **A5 — Stuck "未完成；提交" badge on the dock.** A white pill reading "未完成；提交" renders on top of the 提交 dock icon even when the pointer is far away (`10-roadmap-dark.png` ≈(735,443); `18-profile-light.png` ≈(735,443)). Absent in `16` — flaky. Either a tooltip that never dismisses or a status badge mis-positioned over the icon.
- **A6 — Dead / inconsistent dock clicks.** (a) 证据 does nothing from the project roadmap page (no nav, no toast). (b) 版本 is a dead click on the /prd desk but navigates from /projects/<id>/roadmap. (c) In the light pass, 版本 from the roadmap landed on the /prd *desk* instead of the versions page (earlier dark pass went to /versions from the same origin). (d) On the ideas wall, clicking the idea card body or title does nothing — only the small "PRD" chip opens the PRD desk (`08`/`14` ≈(255,457) dead).
- **A7 — Workspace rows are empty shells.** The 想法 / 设置 / 编辑 / 删除这个工作区 rows contain only a label — no content, chevron, or affordance (`06`/`12` ≈(720,192)–(720,443)). Destructive 删除这个工作区 is styled identically to the neutral rows. Project card also duplicates status: "进行中" chip plus "进行中 0 · 已完成 0" (`06`/`12` ≈(60,568) & (95,590)).
- **A8 — Deep links / refresh 404.** Direct navigation or reload of any client route (e.g. `/workspace/<id>/ideas`) returns the python http.server "Error response 404" page — no SPA fallback in this static serving setup; the app only works when entered at `/`. Confirmed by the console pass 404.
- **A9 — Coach panel inverts the top bar in light theme.** The PRD 教练 panel stays dark navy in light mode (fine, deliberate), but the global top-bar icons over it render dark-on-navy — nearly invisible (`15-prd-desk-light.png` ≈(1287–1410,28): language/theme/person/logout icons).
- **A10 — Dock swallows idea cards at 390 px.** On the narrow ideas wall the centered dock pill covers the idea card's body text; the card is unreadable behind it (`20` ≈(390,845)). Same on dashboard, where the pill hides part of the chapter card (`19` ≈(390,845)).
- **A11 — Viewport shrink leaves a white gutter (canvas stuck).** After booting at desktop width and then emulating a narrow viewport, the Flutter canvas stays at an intermediate ~245 px column with a white band on the right; dispatching a `resize` event does not fix it (`21-narrow-resize-gutter.png` right third). Booting directly at 390 px lays out correctly (`19`/`20`), so this affects window-resize/orientation-change, not first load.
- **A12 — Logged-out landing has no top-area 登录 entry.** Per capture brief the login entry was expected in the top area; the logged-out landing top bar only has logo + language + theme icons, and 登录 lives mid-hero next to 开始打磨 (`01` ≈(772,272)). Discoverable, but not where the brief expected; also worth a product look at hero whitespace (large empty band between CTA row and diagram, `01` ≈(720,300–420)).
- **A13 — Register modal footer highlight wash.** 已经有账号了? shows a wide light-blue highlight wash behind it wider than the text (`03` ≈(565–878,645–678)) — hover or stuck ripple; low confidence.
- **A14 — PRD section cards appear empty.** All PRD editor sections (背景与问题陈述, 目标用户与画像, 目标与成功指标) render heading-only cards with no body content (`09`/`15`). Could be genuinely empty demo content, but combined with A7 the workspace/project/PRD surfaces read largely placeholder.
