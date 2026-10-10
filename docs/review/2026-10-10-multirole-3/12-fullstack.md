# 12 · Fullstack

**Score:** **8.4 / 10** · 8 findings (1 High+)

**Bottom line:** The fullstack surfaces at 754c502 are genuinely hardened — every control I traced does what its label says except in small corners, the async/setState contracts the recent wave tightened hold almost everywhere, env names agree across compose/.env.example/dev.sh/code with one catalog omission, the Docker/nginx path serves a from-source build with a correct CSP, and en/zh key sets are exactly identical (375/375, verified). The worst defect I found is in the one funnel that has never run live: the invitation email's HTML tells the invitee the link expires in 168 minutes when it lives 7 days — a 60x misstatement in the first message every new user receives. Beyond that: two pages still write TextEditingControllers after awaits without a mounted guard, the dock's Capture/Ideas/Version/Commit items dead-tap on their own destination (only Evidence got the pulse fix), and the PRODUCT.md-to-implementation delta is admitted only for roles while autosave, personas, product branches, decision log, the facts surface, lenses, and dependencies have no implementation and no tracker.

---

## Fullstack review — InsideOut @ `754c502`

### Verdict

This is a codebase whose fullstack seams are unusually well tended. Tracing every newly touched control the ask named — the export page, the versions stack, required-field block forms, dock deep links — plus the server routes behind them and the deploy path, I found the label-vs-behavior discipline to hold everywhere except small corners, and the async contracts the 2026-10-10 wave hardened to hold everywhere except two pages that write controllers after awaits. The worst defect sits in the one funnel that has never run live: the invitation email's HTML tells the invitee the link expires in **168 minutes** when it lives **7 days**.

### What I checked and how

- **Controls vs labels.** Read `client/lib/features/prd/export_page.dart`, `versions_stack.dart`, `versions_page.dart`, `versions_commit_dialog.dart`, `theme/block_form.dart`, `block_modal.dart`, `chrome/product_dock.dart`, `features/workspace/ideas_page.dart`, `settings_page.dart`, `features/auth/*`, and `client/lib/app.dart`; cross-checked the export flow against `server/internal/api/prds.go` `handleExportPrd` (the client's comment about the audience-400 is accurate — prds.go:311 rejects `audience` unless format is markdown).
- **UI-thread vs async contracts.** Read `api/client_core.dart` (single-flight 401 refresh), `api/client_stream.dart` (`_untilStopped` + `stop` futures), `coach_panel.dart` (`gone` completer), `roadmap_page.dart` (stop completer, backoff, terminal 401/403/404), `prd_page.dart`, `settings_page.dart`.
- **Env consistency.** Executed greps extracting variable names from `.env.example`, `docker-compose.yml`, `server/internal/config/config.go`, and `server/internal/agent/coach.go`: all names agree except one catalog omission (FULL-07). `scripts/dev.sh` and `scripts/env_catalog.py` read as documented — the catalog derives its schema from the skeleton, which is why the omission matters.
- **Docker/nginx serving path.** `client/Dockerfile` builds the web bundle from source at image build (`flutter build web --release --dart-define=API_BASE=/api/v1`), copies it into nginx-unprivileged, and `.dockerignore` excludes the local `build/` — so the image always serves the current client. `client/nginx.conf` carries the BUG-027 variable-upstream re-resolve, the 8 MiB webhook override, an exact CSP, and `try_files … /index.html`, which with `usePathUrlStrategy()` (web_url_strategy_web.dart) makes deep links like `/prd/:id/versions` work on refresh; I checked `client/web/index.html`'s Google-Fonts origins against the CSP entries and they match. The three nginx check scripts run in CI (coordinator lead 6).
- **i18n parity.** Executed a key-set diff: `grep`-extracted keys from `en_{shell,account,workspace,prd}.dart` vs `zh_{...}.dart`, sorted and `comm`-diffed — **375 keys on each side, zero one-sided keys**. `test/l10n_unread_keys_test.dart` backstops this in the suite (421/421 green per the coordinator's run today; I did not re-run the suite, and nothing I claim rests on it).

### Findings

| # | Sev | Bucket | Where | Finding |
|---|-----|--------|-------|---------|
| FULL-01 | High | FIX | `server/internal/api/admin.go:131` | Invitation email HTML misstates expiry 60x: `int(ttl.Hours())` is passed into `simpleEmailHTML`, whose template hardcodes "minutes" (`auth_email_flows.go:342`), so a 7-day invite reads "expires in 168 minutes" in HTML while the text body correctly says days. |
| FULL-02 | Medium | FIX | `client/lib/features/prd/prd_page.dart:67`, `settings_page.dart:55` | Controllers written after awaited loads without a mounted guard (prd_page's guard is at line 81, *after* `titleCtrl.text = loaded.title`); settings `_load` has none at all, and its reload also clobbers unsaved edits. |
| FULL-03 | Medium | ADD | `PRODUCT.md:201` | The PRODUCT.md→implementation delta is untracked: autosave, structured personas, Product Branch/Merge, Decision Log, facts surface, roadmap lenses, dependencies, and external feedback have no code and no dated issue; only the roles gap is admitted. |
| FULL-04 | Low | FIX | `client/lib/chrome/product_dock.dart:185` | Capture/Ideas/Version/Commit dead-tap on their own destination (`_activate` early-returns; only Evidence pulses) — the failure mode the file's own comment says was fixed for Version. |
| FULL-05 | Low | IMPROVE | `client/lib/features/prd/export_page.dart:80` | Export page titled "Download Markdown" while offering Print/HTML too; the desk menu item reuses the string. |
| FULL-06 | Low | IMPROVE | `server/internal/api/auth_email_flows.go:341` | Auth email button is pre-Cobalt green `#1f6f54` with 8px radius; DESIGN.md's law is cobalt-500 `#3B5BFF`, squared. |
| FULL-07 | Low | IMPROVE | `docker-compose.yml:101` | `INSIDEOUT_GH_PRIVATE_KEY_FILE` interpolated by compose and read by config.go, but absent from the `.env.example` skeleton, so `env.sh edit --list` never shows it. |
| FULL-08 | Low | HARDEN | `server/internal/api/auth_email_flows_test.go` | No copy assertions for any of the four email templates (executed grep: zero matches), and the funnel has never run against the hosted mail provider — the gap that let FULL-01 survive. |

Each finding carries file:line and a verbatim quote in the structured record, with a concrete fix.

### Strengths

- **Export page state discipline** — `export_page.dart:103-113` resets the audience and drops the stale preview/error on every format or audience change, with comments explaining the server contract it mirrors; the download/copy fallback (`download_stub.dart` returning false off-web) keeps the button functional on every platform.
- **Stream lifecycle contracts** — `client_stream.dart`'s `_untilStopped` turns a leaving user into a cancelled model call for both native dio and web fetch paths; `coach_panel.dart:56`'s `gone` completer and `roadmap_page.dart`'s stop-on-dispose + 500ms→5s backoff with terminal 401/403/404 are textbook.
- **One error contract for reads and writes** — `widgets/guarded_action.dart` (localized SnackBar, input preserved on failure) and `theme/block_state.dart`'s single `readGate` for every FutureBuilder branch eliminate the silent-failure and raw-exception classes.
- **Env tooling that cannot drift silently** — `scripts/env_catalog.py` derives the schema from the skeleton instead of a second hardcoded list; my cross-check found exactly one leak, not a scattering.
- **Deploy path integrity** — from-source image builds, per-request upstream resolution (BUG-027), and a CSP whose origins I verified against `index.html`.
- **i18n parity as a invariant** — 375/375 keys both locales, with an unread-keys test keeping new strings honest.

### Leads 14 and 15 (assigned to me)

- **Lead 14 (PRODUCT.md promises without end-to-end implementation):** ran it. Implemented and spot-verified: deadline pressure ladder and three-item Next cap (`roadmaptime.go:15-75`), audience projections (`/prds/{id}/view`, export), human Commit with audience/summary/unresolved (`prd_commits.go`, versions stack), agent context/checkpoint/propose routes (`agent_routes.go:18-21`), MCP commit tool. Not implemented anywhere: working-version autosave, the structured persona model, Product Branch/Merge, the Decision Log, the facts trust surface (recording exists in `agent/ledger.go`, no route/no client UI), roadmap lenses, dependency modeling, and external feedback invitations. Filed as FULL-03 with the tracker as the fix.
- **Lead 15 (mail funnel never exercised):** confirmed from the material — signup is invitation-only (`client_auth.dart:94` default `invitation_only`), the client honestly banners mail-off (`mail_off_notice.dart`), and no hosted send has ever run. My concrete output is FULL-01 (a bug only live mail would have surfaced) and FULL-08 (the test/live-send hardening to close the class).

### The cheapest user-visible fix

`admin.go:131`: pass `int(ttl.Minutes())` instead of `int(ttl.Hours())` — or render a unit-aware duration — so the invitation email's HTML stops telling invitees their 7-day link dies in 168 minutes. One line, plus one copy-pinning test. While in that template, the second-cheapest is the same file's green button (FULL-06): `#1f6f54` → `#3B5BFF`.

## What this role would do first

Apply the one-line invite-email fix (admin.go:131: pass minutes, not hours, into simpleEmailHTML) with a test asserting the text and HTML expiry copy agree — it is the cheapest user-visible fix and closes the worst finding before invitations ever go out.
