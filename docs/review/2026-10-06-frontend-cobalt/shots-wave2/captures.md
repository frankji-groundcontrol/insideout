# InsideOut Web Client — Wave 2 Visual Capture Index

- **Date:** 2026-10-06 (Wave 2 re-shoot)
- **Build:** release Flutter web (CanvasKit) rebuilt from `main` (`flutter build web --release`, `client/build/web`, `main.dart.js` 12:06) served at `http://127.0.0.1:5174/`
- **Driver:** `scripts/sim/web_driver.py` (headless Chrome, CDP) with the `--disable-remote-fonts` launch flag temporarily commented out so CanvasKit could fetch Noto Sans SC (flag restored after the pass — `git diff` clean). Narrow shots used a CDP `Emulation.setDeviceMetricsOverride` (390×844) applied on a fresh session before loading the app (never resized mid-session, see A11).
- **Viewport note:** desktop shots are 1440×813 (headless Chrome `--window-size=1440,900` yields an 813px-tall viewport), narrow shots 390×844. Login via the app's own modal (`demo@insideout.local`); all navigation in-app.
- **Environment note:** the process originally holding :8080 was a stale Sep-12 `go run ./cmd/api` orphan (no `/api/v1` routes) — it was replaced with the current server (`scripts/dev.sh -C server go run ./cmd/insideout`) before login could work. The demo DB behind it was seeded ~2026-08-07 and run multiple times: dashboard shows three "01/02/03 InsideOut Demo Team" workspaces and timestamps render as raw ISO (data differs from the baseline pass; layout/occlusion verdicts unaffected).

## Shot index

| File | Page | Theme | Viewport | What it shows |
|---|---|---|---|---|
| `01-landing-default.png` | landing (logged out) | light | 1440×813 | Hero, 开始打磨/登录 buttons, step diagram; no boot-splash ghost |
| `02-login-modal-light.png` | login modal | light | 1440×813 | AuthDoor modal; cobalt disc + "InsideOut" wordmark sit INSIDE the modal card (A1 evidence) |
| `03-dashboard-light.png` | /dashboard | light | 1440×813 | Chapter card paragraph fully visible; last workspace card partly under the dock until scrolled |
| `04-workspace-light.png` | /workspace/<id> | light | 1440×813 | 删除这个工作区 row fully visible above dock; FAB overlaps project-card date (new anomaly N1) |
| `05-project-light.png` | /projects/<id> | light | 1440×813 | GitHub 同步 card bottom edge and 发布进展 button clear of the dock (A2) |
| `06-ideas-wall-light.png` | ideas wall | light | 1440×813 | Idea card fully visible; dock tooltip "想法" captured mid-hover (expected behavior, not an anomaly) |
| `07-prd-desk-light.png` | /prd/<id> | light | 1440×813 | Top-bar icons on a light pill over the navy coach panel — clearly visible (A9 evidence) |
| `08-roadmap-light.png` | roadmap | light | 1440×813 | Empty state; dock clean, no stuck badge on 提交 |
| `09-versions-light.png` | /prd/<id>/versions | light | 1440×813 | Readiness checklist incl. 约束与依赖 / 非目标 fully above the dock (A2) |
| `10-profile-light.png` | /profile | light | 1440×813 | Stuck "未完成；提交" badge on the 提交 dock icon, pointer far away (A5 evidence) |
| `11-profile-dark.png` | /profile | dark | 1440×813 | Same stuck badge after theme toggle — persists across themes (A5 evidence) |
| `12-landing-dark.png` | landing (logged out) | dark | 1440×813 | Dark hero after logout; theme-toggle tooltip visible |
| `13-login-modal-dark.png` | login modal | dark | 1440×813 | Dark AuthDoor modal, branding inside the card — no ghost (A1, dark) |
| `14-dashboard-dark.png` | /dashboard | dark | 1440×813 | Dark dashboard; dock at bottom; last card partly under dock at first paint |
| `15-dashboard-dark-scrolled.png` | /dashboard (scrolled) | dark | 1440×813 | Scrolled to bottom: all three workspace cards fully visible above the dock (A2 residual cleared) |
| `16-workspace-dark.png` | /workspace/<id> | dark | 1440×813 | Same as 04 in dark; FAB/date overlap still present |
| `17-project-dark.png` | /projects/<id> | dark | 1440×813 | Same as 05 in dark |
| `18-ideas-wall-dark.png` | ideas wall | dark | 1440×813 | Same as 06 in dark; dock clean |
| `19-prd-desk-dark.png` | /prd/<id> | dark | 1440×813 | Coach panel dark; top-bar icons visible on dark pill |
| `20-roadmap-dark.png` | roadmap | dark | 1440×813 | Two "Demo" filter chips; dock clean, no badge |
| `21-versions-dark.png` | /prd/<id>/versions | dark | 1440×813 | Checklist fully visible; dock clean |
| `23-landing-narrow-dark.png` | landing (logged out) | dark | 390×844 | Responsive hero, no overflow, no gutter, no tofu |
| `24-dashboard-narrow-dark.png` | /dashboard | dark | 390×844 | Chapter card fully above the dock (A10, dashboard instance) |
| `25-ideas-narrow-dark.png` | ideas wall | dark | 390×844 | Idea card completely visible; dock clear below it (A10 evidence) |
| `26-ideas-narrow-dark-scrolled.png` | ideas wall (scrolled) | dark | 390×844 | Identical — page content fits entirely above the dock; nothing to scroll |

## Anomaly verdicts (Wave 2 scope)

- **A1 — boot-splash ghost above auth modals: FIXED.** `02-login-modal-light.png` and `13-login-modal-dark.png`: the cobalt disc + "InsideOut" wordmark are rendered inside the AuthDoor modal card itself; nothing floats on the scrim above/behind the modal.
- **A2 — dock overlaps page content: FIXED.** The dock now sits at the bottom of the viewport and pages reserve clearance: dashboard chapter paragraph complete (`03`/`14` vs baseline mid-sentence clip), workspace 删除这个工作区 row fully visible (`04`/`16`), project GitHub card bottom edge + 发布进展 button above the dock (`05`/`17`), PRD sections unobstructed (`07`), versions checklist incl. 约束与依赖 / 非目标 fully visible (`09`/`21`). Residual: at first paint a list whose last item sits below the fold shows it partially under the dock until the user scrolls (`14`); `15-dashboard-dark-scrolled.png` shows the scrolled-to-bottom state clears the dock completely. Judged fixed; the residual is standard floating-dock behavior, worth a polish note only.
- **A5 — stuck "未完成；提交" badge on the dock: NOT FIXED.** `10-profile-light.png` and `11-profile-dark.png`: the badge pill (icon + "未完成；提交") is stuck expanded on the 提交 dock icon on /profile with the pointer far away, and it survives a theme toggle. Still flaky/state-dependent — it did NOT reproduce on roadmap (`08`/`20`), versions (`09`/`21`), ideas (`06`/`18`), or dashboard (`03`/`14`) this pass. (Note: a prior interim summary marked A5 fixed based on `06-ideas-wall-light.png`; that page's clean dock is confirmed, but the profile evidence above reproduces the exact baseline defect.)
- **A9 — top-bar icons invisible over the navy coach panel in light theme: FIXED.** `07-prd-desk-light.png`: language/theme/person/logout icons render on a light pill background at the top-right over the navy panel — clearly visible (baseline `15` showed dark-on-navy).
- **A10 — dock swallows idea cards at 390px: FIXED.** `25-ideas-narrow-dark.png`: the idea card's title, body text, and chips are fully visible; the dock sits below the card with clear separation. `26-ideas-narrow-dark-scrolled.png` shows the page fits entirely above the dock (no scrollable remainder). Dashboard instance also fixed (`24-dashboard-narrow-dark.png` — chapter card paragraph complete, vs baseline `19` where the pill hid part of it).

## New anomalies (no fixes attempted)

- **N1 — quick-capture FAB occludes the project card's date.** A blue circular "+" FAB (new in Wave 2; absent from baseline `06`) floats at the bottom-right on the workspace page and covers the project card's date text (`04-workspace-light.png`, `16-workspace-dark.png` ≈(1395,686)). Minor.
- **N2 — dock icon taps do not navigate at 390px width.** In the narrow session, taps on dock icons (记下/想法) were received — hover tooltips and the active-route highlight render — but no navigation fired (reproduced across 4 attempts: touch tap and mouse, dsf 2 and 1). The same taps work at 1440px. Other bottom-region targets (workspace card) tap fine at 390px, so it is dock-specific. Extends baseline A6 (dead dock clicks) to the narrow layout; functional, not visual.
- **N3 — demo-data duplicates (environment, not UI).** Three identical "InsideOut Demo Team" workspaces on the dashboard and raw ISO timestamps on cards/updates (baseline showed relative "8 小时前" on the workspace card). Traced to the local demo DB seeded 2026-08-07 with multiple seed runs, not to a rendering change.

Expected behavior, recorded as requested: the hover tooltip above the dock in `06-ideas-wall-light.png` is normal Tooltip-on-hover behavior (it dismisses when the pointer moves away — see `08-roadmap-light.png`, taken after moving the pointer off); not an anomaly.

## Not re-verified / out of scope

A3/A4/A6/A7/A8/A11/A12/A13/A14 from the baseline list are outside the Wave 2 fix set; several were observed still present (A3 raw ISO timestamps, A4 "证据" eyebrow heading on project page, A7 empty workspace rows, A14 heading-only PRD sections) and are listed in N3/data notes where relevant. No fixes attempted, per the verification-only brief.
