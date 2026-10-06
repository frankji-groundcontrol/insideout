# InsideOut Web Client — Wave 3 Visual Capture Index

- **Date:** 2026-10-06 (Wave 3 re-shoot — Cobalt-Neo decree verification)
- **Build:** release Flutter web (CanvasKit) rebuilt from `main` (`flutter build web --release`, `client/build/web`) served at `http://127.0.0.1:5174/`; API on `:8080` healthy.
- **Driver:** `scripts/sim/web_driver.py` (headless Chrome, CDP) with the `--disable-remote-fonts` launch flag temporarily commented out so CanvasKit could fetch Noto Sans SC (flag restored after the pass — `git diff` clean). Narrow shots used CDP `Emulation.setDeviceMetricsOverride` (390×844) applied **before** loading the app on a fresh session (the app must boot under the override — re-applying it to a live page leaves hit-testing dead, an automation-only caveat).
- **What changed since Wave 2:** Cobalt-Neo tokens applied (0px radius, 2px borders, 4px4px zero-blur cast shadows, PressableBlock press-the-block motion, amber `#FFD166` square warn dot, neutral dark ramp); `client/lib/chrome/product_dock.dart` rewritten (GestureDetector-based PressableBlock items; the old stuck "未完成；提交" badge overlay is gone — unfinished state now renders as an 8×8 amber square at the item's top-right plus a swapped item label).

## Shot index

| File | Page | Theme | Viewport | What it shows |
|---|---|---|---|---|
| `01-landing-light.png` | landing (logged out) | light | 1440×813 | Squared 开始打磨/登录 blocks, 2px borders, cast shadows (decree) |
| `02-login-modal-light.png` | login modal | light | 1440×813 | AuthDoor modal: 0-radius card, 2px border, cast shadow, squared buttons; no A1 ghost |
| `03-dashboard-light.png` | /dashboard | light | 1440×813 | Squared chapter/workspace cards; dock squared at bottom |
| `04-dock-rest-crop-light.png` | dashboard dock (crop) | light | 1440×813 | Rest state: every dock item block on full cast shadow |
| `05-dock-hover-crop-light.png` | dashboard dock (crop) | light | 1440×813 | Hover on 证据: translate(2,2) + shrunk castSm shadow + tooltip |
| `06-dock-pressed-crop-light.png` | dashboard dock (crop) | light | 1440×813 | Press-hold on 想法: block shifted (4,4), shadow gone (pressed into page) |
| `07-workspace-light.png` | /workspace/<id> | light | 1440×813 | Squared rows with cast shadows; squared FAB overlaps project-card date (N1) |
| `08-project-light.png` | /projects/<id> | light | 1440×813 | Squared GitHub 同步 card; content clear of dock |
| `09-ideas-wall-light.png` | ideas wall | light | 1440×813 | Squared capture bar + idea card; card fully above dock |
| `10-prd-desk-light.png` | /prd/<id> | light | 1440×813 | Squared PRD sections; top-bar icons on light pill over navy panel (A9 stays fixed) |
| `11-roadmap-light.png` | roadmap | light | 1440×813 | Squared chrome; empty state |
| `12-versions-light.png` | /prd/<id>/versions | light | 1440×813 | Squared checklist card; dock clean |
| `13-profile-light.png` | /profile | light | 1440×813 | Squared form; A5 evidence — see `14` |
| `14-dock-crop-profile-light.png` | profile dock (crop) | light | 1440×813 | 提交 item: label swapped to "未完成：提交" + 8×8 amber square top-right (new intended warn design, A5) |
| `15-profile-dark.png` | /profile | dark | 1440×813 | Dark form, squared |
| `16-dock-crop-profile-dark.png` | profile dock (crop) | dark | 1440×813 | Same amber square + swapped label in dark; strong ink cast shadows |
| `17-login-modal-dark.png` | login modal | dark | 1440×813 | Dark AuthDoor, squared, cast shadow; no A1 ghost |
| `18-dashboard-dark.png` | /dashboard | dark | 1440×813 | Neutral dark ramp surfaces (no blue cast), squared cards, ink shadows |
| `19-workspace-dark.png` | /workspace/<id> | dark | 1440×813 | Dark squared rows; FAB/date overlap (N1) |
| `20-project-dark.png` | /projects/<id> | dark | 1440×813 | Dark squared cards |
| `21-ideas-wall-dark.png` | ideas wall | dark | 1440×813 | Dark squared card, dock clean |
| `22-prd-desk-dark.png` | /prd/<id> | dark | 1440×813 | Dark editor + navy coach panel; squared chips |
| `23-roadmap-dark.png` | roadmap | dark | 1440×813 | Dark empty state; dock clean |
| `24-versions-dark.png` | /prd/<id>/versions | dark | 1440×813 | Dark squared checklist; dock clean |
| `25-profile-dark.png` | /profile | dark | 1440×813 | Dark profile (companion to `15`) |
| `26-cards-crop-project-dark.png` | project cards (crop) | dark | 1440×813 | Neutral dark ramp close-up: squared card, 2px border, squared buttons |
| `27-dashboard-narrow-dark.png` | /dashboard | dark | 390×844 | Squared chapter card fully above dock; squared dock; N2 tap-test page |
| `28-ideas-narrow-dark.png` | ideas wall | dark | 390×844 | Idea card fully visible above dock (A10 stays fixed); squared surfaces |
| `29-profile-narrow-dark.png` | /profile | dark | 390×844 | Squared form fields; dock squared; plain 提交 label (LoopFocus state reset — see A5 note) |

## Decree verdicts (Cobalt-Neo)

- **Radius 0px everywhere: PASS.** No rounded corners anywhere: modal (`02`/`17`), landing buttons (`01`), all cards/rows/chips (`03`, `07`–`13`, `18`–`25`), dock body and items (`04`), narrow layouts (`27`–`29`). The Wave 2 circular FAB is now a square block (`07`).
- **2px borders: PASS.** Cards, rows, inputs, dock body, and modal all carry the 2px ink border (visible on every surface; light theme shows it as dark edge, e.g. `01`–`03`).
- **Cast shadows (4,4, blur 0): PASS.** Offset hard shadows visible under every card/modal/dock surface in both themes (`03`, `07`, `17`, `18`, `26`).
- **Press-the-block motion: PASS.** Dock item hover = translate(2,2) with shadow shrunk to castSm (`05` vs `04`); press-hold = translate(4,4) with the shadow gone (`06`).
- **Amber #FFD166 square warn dot: PASS.** 8×8 amber square at the unfinished item's top-right (`14`, `16`, and the warm narrow dock in the N2 test run).
- **Neutral dark ramp: PASS.** Dark surfaces read neutral charcoal, not blue (`18`, `26`); brand blue only on interactive accents.
- **A1 (boot-splash ghost): stays FIXED** — `02`, `17`.

## Open-item verdicts

- **A5 — stuck "未完成；提交" badge: FIXED.** The old stuck overlay badge no longer exists. What renders now is the intended design: when the app knows the commit step is unfinished, the 提交 item's own label swaps to "未完成：提交" and an 8×8 amber square appears at its top-right (`14` light, `16` dark). No stuck pill, pointer far away, stable across the theme toggle. Nuance: the marker is LoopFocus-state-dependent — it showed on /profile in the desktop pass (after /versions loaded commit state) but not on the narrow /profile (`29`) after intervening navigation reset the state; on a cold dashboard it is absent. State-dependent visibility is per the implementation, not the old stuck-overlay defect.
- **N2 — dock taps dead at 390px: FIXED (rewritten dock).** Warm-state tap matrix at 390×844, mouse AND touch, all six items navigate correctly: 记下→`/workspace/<id>/ideas`, 想法→ideas, 版本→`/prd/<id>`, 提交→`/prd/<id>/versions`, 路线图→`/projects/<id>/roadmap`, 证据→`/projects/<id>` (baseline A6a's dead 证据 click now routes to the project page). Caveats observed: (a) on a **cold** dashboard (no LoopFocus ids yet) every item falls back to `stay()` → `/dashboard`, i.e. no visible navigation and no feedback — by design but UX-notable, and this is what made the Wave 2-era narrow dock look dead; (b) when the 提交 label expands ("未完成：提交"), the dock widens and item x-positions shift — taps must aim at the current layout; (c) console showed 3 identical uncaught `Error`s (minified stack at `main.dart.js:85738`) during the tap-test session — no visible impact, not symbolized.
- **N1 — FAB overlaps project-card date: NOT FIXED.** The quick-capture FAB (now a squared block per the decree) still covers the project card's date at the workspace page bottom-right (`07-workspace-light.png`, `19-workspace-dark.png` ≈(1395,686)).

## New anomalies (no fixes attempted)

- **W3-N1 — dock width grows past the viewport-safe center when 提交 is unfinished.** With the wide "未完成：提交" label, the dock at 390px nearly spans the full screen width (measured body x≈30–365 CSS of 390) — visually heavy and it shifts sibling item positions (contributed to the tap-test misses). Layout nit, not a functional defect.
- **W3-N2 — three uncaught console Errors** (same minified trace, 3×) observed during the narrow tap-test session; no user-visible impact. Suggest symbolized follow-up in a dev build.

Automation caveats (not app anomalies): the hover tooltips captured above the dock (`05`, `28` "记下", `29` "个人资料") are synthetic-pointer hover artifacts; re-applying the viewport override to a live page without reload deadens hit-testing (boot under the override instead).
