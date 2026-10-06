#!/usr/bin/env python3
"""Attach skeptic verdicts to reviews-all.json -> result.json for register.py.

Verdicts join on (file, id); the file is copied verbatim from the finding so
skeptic line-suffix drift cannot break the join.
"""

from __future__ import annotations

import json
import pathlib

REV = pathlib.Path("<repo>/docs/review/2026-10-06-frontend-cobalt")

# (id, verdict, new_severity, reason)
VERDICTS: list[tuple[str, str, str | None, str]] = [
    ("dock-pill-mid-screen-root-cause", "CONFIRMED", None,
     "product_dock.dart:28 Center + :26 SafeArea inside the bottomNavigationBar slot (app.dart:135) with extendBody (app.dart:71) centers the pill at viewport mid-height; :15 height=88.0 is a dead constant with zero references. Fix: Align(alignment: Alignment.bottomCenter, heightFactor: 1.0)."),
    ("dock-renders-at-viewport-center", "CONFIRMED", None,
     "Same defect as dock-pill-mid-screen-root-cause (semantic duplicate kept for traceability; merge at fix time)."),
    ("pm-dock-mid-viewport", "CONFIRMED", None,
     "Same defect as dock-pill-mid-screen-root-cause, described from the wiring side app.dart:135 (semantic duplicate kept for traceability)."),
    ("full-dock-crosses-workspaces", "CONFIRMED", None,
     "product_dock.dart:91 focus.projectId ?? _projectOnRoute(context); ideas_page.dart:38 and workspace_page.dart:52 never reseed LoopFocus; LoopFocus.clear runs only on logout (app.dart:120), so dock state leaks across workspaces."),
    ("dock-dead-clicks-evidence-version", "DOWNGRADED", "Medium",
     "stay() does navigate to /projects/<id> when a project is known (product_dock.dart:93-94, route router.dart:37). Dead clicks occur only in a fresh session at /dashboard with null focus; the Version dead click is self-navigation of the already-selected item."),
    ("dock-unfinished-blue-on-blue", "CONFIRMED", None,
     "Recomputed contrast: light statusWarnFg #1D4ED8 on brand #3B5BFF = 1.32:1; dark #C7D2FE on brand = 3.41:1; both below 4.5:1. product_dock.dart:166 border, :172 icon, :177 label sit on a transparent container over the brand pill."),
    ("test-dock-harness-hides-overlap", "CONFIRMED", None,
     "product_dock_test.dart:17 mounts a bare Scaffold without extendBody/bottomNavigationBar, so the harness structurally cannot expose the mid-screen overlap it is meant to guard."),
    ("test-no-goldens-modal-collision", "CONFIRMED", None,
     "widget_test.dart:12 setSurfaceSize(Size(1200, 5000)) inflates the viewport past every breakpoint; zero golden tests exist anywhere in the repo."),
    ("pm-authdoor-header-collision", "CONFIRMED", None,
     "auth_door.dart:38 ColoredBox(color: Colors.black54) scrims the full page; the brand header is a Column sibling above the :72 card so the scrim covers it; landing_page.dart:50,181 stack layers."),
    ("rel-no-network-timeouts", "CONFIRMED", None,
     "client.dart:30-34 constructs Dio() with no connect/receive/send timeout; the only .timeout( in lib is web_fonts_web.dart:46; main.dart:33 boot gate awaits a future that never completes when the API hangs."),
    ("rel-hydrate-wipes-tokens", "CONFIRMED", None,
     "session.dart:58-60 catch (_) { user = null; await clear(); } is untyped; clear() (:123-127) deletes stored tokens, so an offline app start silently logs the user out."),
    ("rel-coach-stream-no-timeout", "CONFIRMED", None,
     "client.dart:502 readCoachSse has no deadline; sse_transport_web.dart:27 has no AbortController; coach_panel.dart:212 enabled: !sending && retry == 0 leaves the composer permanently disabled while a hung stream holds sending=true."),
    ("full-write-actions-fail-silently", "CONFIRMED", None,
     "Write calls with no try/catch and no user feedback: ideas_page.dart:63,83,110,134; versions_page.dart:57; dashboard_page.dart:71; prd_page.dart:81."),
    ("feat-capture-convert-no-error-handling", "CONFIRMED", None,
     "Subset of full-write-actions-fail-silently (semantic duplicate kept for traceability)."),
    ("feat-idea-status-raw-enum", "CONFIRMED", None,
     "ideas_page.dart:326 idea.content.isEmpty ? idea.status : idea.content; quick capture always passes content:'' so cards render raw enum strings; the localized key exists unused at zh.dart:35."),
    ("no-shadow-system", "CONFIRMED", None,
     "cobalt.dart:37-60 declares color fields only; cobalt_theme.dart:49,56 set elevation: 0; the only shadows in lib are ad-hoc one-offs (product_dock.dart:31, auth_dock.dart:79)."),
    ("radius-decree-inverse", "CONFIRMED", None,
     "cobalt.dart:72-75 verbatim 12.0/20.0/28.0/999.0; radiusPill is applied to containers (cobalt_theme.dart:29,93,97,101 and StadiumBorder :85,:109), the inverse of the Cobalt-Neo decree."),
    ("no-press-the-block", "CONFIRMED", None,
     "ink_motion.dart:4 inkEase is the only motion primitive in lib; no hover/press state mechanism exists anywhere outside roadmap_canvas cursor tracking."),
    ("hairline-border-weight", "CONFIRMED", None,
     "field_card.dart:29 BorderSide with no width (defaults to 1.0); strokeSubtle #C7D2FE on white = 1.49:1; every container border shares this. DESIGN.md:121 currently decrees hairlines - doc drift tracked under design-md-contract-stale."),
    ("blue-tinted-dark-ramp", "CONFIRMED", None,
     "cobalt.dart:133-137 verbatim; hue950 #1E2A5A doubles as surfaceSunken; ground-to-raised 1.09:1 and ground-to-sunken 1.42:1, so dark layers are visually indistinguishable."),
    ("light-mode-default", "CONFIRMED", None,
     "appearance.dart:36 dark = false; :56-57 only an explicit stored value flips it; main.dart:22 sets no override; no platformBrightness read anywhere in lib."),
    ("stroke-tokens-fail-non-text-contrast", "CONFIRMED", None,
     "Recomputed against both surfaces, all below 3:1 (WCAG non-text): light subtle 1.40/1.49, light strong 2.18/2.32, dark subtle 1.59/2.26, dark strong 1.45/2.07; tokens outline components at cobalt_theme.dart:58,74,94,98,110,112."),
    ("minimap-color-only-invisible-marks", "CONFIRMED", None,
     "roadmap_canvas.dart:172-183 mark() encodes state by color alone with no label or icon; :213 height: 4; contrast locked 1.40/1.59, pending 2.18/2.26, done 2.90 (dark) - all below 3:1."),
    ("admin-hardcoded-red-dark-unreadable", "CONFIRMED", None,
     "admin_section.dart:167 verbatim Color(0xFFB3261E) on surfaceRaised #141820 = 2.72:1; the fgDanger token already exists (cobalt.dart:143) and is wired to ColorScheme.error (cobalt_theme.dart:14), so the fix is a token swap."),
    ("no-dock-clearance-coach-and-canvas", "CONFIRMED", None,
     "Coach composer bottom padding is only 24px (coach_panel.dart:205 fromLTRB(16,8,16,24)); roadmap_page.dart:226 Column has no bottom inset; dashboard_page.dart:137 reserves 140 for comparison."),
    ("missing-viewport-meta", "CONFIRMED", None,
     "grep -ri viewport client/web returns zero matches; mobile browsers fall back to a 980px layout viewport, breaking every responsive assumption."),
    ("create-project-fab-invisible", "CONFIRMED", None,
     "workspace_page.dart:115 builds the FAB unconditionally and :84 is the only createProject call in lib; screenshots 06-workspace-dark.png and 12-workspace-light.png show no FAB in any corner - consistent with the full-height bottomNavigationBar pushing the endFloat position off-screen. Re-verify after the dock Align fix."),
    ("topbar-icons-invisible-over-navy", "CONFIRMED", None,
     "app.dart:90-130 Positioned overlay has no backdrop; icons use onSurfaceVariant/fgSecondary #3A3F4A over hue950 #1E2A5A = 1.30:1. Note the light-theme coach panel is also navy (coach_panel.dart:112), so this is not dark-mode-only."),
    ("project-page-evidence-heading", "CONFIRMED", None,
     "project_page.dart:150-153 renders displaySmall 52px uppercase '证据' as the page lead while :155 renders the actual project name at headlineSmall 26px; string source zh.dart:325."),
]


def main() -> None:
    data = json.loads((REV / "reviews-all.json").read_text(encoding="utf-8"))
    by_id: dict[str, dict] = {}
    for review in data["reviews"]:
        for finding in review.get("findings") or []:
            by_id[ finding["id"] ] = finding

    verdicts: list[dict] = []
    missing: list[str] = []
    for fid, verdict, new_sev, reason in VERDICTS:
        finding = by_id.get(fid)
        if finding is None:
            missing.append(fid)
            continue
        row = {
            "id": fid,
            "file": finding.get("file", ""),
            "summary": finding.get("summary", ""),
            "verdict": verdict,
            "reason": reason,
        }
        if new_sev:
            row["new_severity"] = new_sev
        verdicts.append(row)

    if missing:
        raise SystemExit(f"finding ids not found in reviews-all.json: {missing}")

    out = {
        "reviews": data["reviews"],
        "verdicts": verdicts,
        "critic": data.get("critic") or {},
    }
    (REV / "result.json").write_text(json.dumps(out, indent=1, ensure_ascii=False), encoding="utf-8")
    print(f"wrote result.json: {len(out['reviews'])} reviews, {len(verdicts)} verdicts")


if __name__ == "__main__":
    main()
