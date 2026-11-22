# 2026-11-22 — The full roadmap of a node inside its window

Plan: [full roadmap in node windows](../plans/2026-11-22-full-roadmap-in-node-windows/2026-11-22-full-roadmap-in-node-windows.md).
Owner comment: "I want to see complete experiences like my own, not simplified
to do" … "for pop-ups"
([design QA](../design-qa/2026-11-22-full-roadmap-in-node-windows.md)).

| Change | Commits |
| --- | --- |
| A node's window is now the node's real roadmap page, embedded, rather than a copy: `RoadmapPage(embedded: true)`. It shows the breadcrumb, the person-in-charge banner, the Progress / Bands / Canvas / "Priya's" toggles, "New item" for writers, card menus, and Canvas pan and zoom, and opens on Progress. The page reads the window as its screen through `MediaQuery`, so a small window gets the phone layout. A tap inside a window opens another window on the same layer. A double-click or "Open roadmap" navigates the main page. A breadcrumb ancestor re-roots the window. Each window's live presence stream closes with the window (since replaced: windows hold no stream, see [node window streams](./2026-11-22-node-window-streams.md)). Windows open at 760×600, clamped to the viewport. | `49a51578` |
| In a window, the big title and lead line give way to the title bar, the toggles share one row with "New item", and the lists leave no room for a dock the window does not have. The window title has its own key, so tests read it on desktop and phone alike. | `92207229` |

## Notes

- **Small windows:** at the 320×240 minimum the Canvas is cramped but
  nothing overflows. The window can be resized or the Canvas fitted.
- **Navigation from a window:** the project link in a window's breadcrumb,
  and the node menu's "Open" and "delete this root", still navigate the main
  page.
- **Phone:** reopening a node that is already open stacks a second sheet.

## Verification

- Go build, vet and gofmt: clean.
- `go test ./...`: passes without a database, and on a fresh database with 0
  skipped tests.
- `flutter analyze`: 8 infos (baseline). `flutter test`: 755 passed.
- Doc and redaction checks: pass.
- Live as Maya: Matching opens as a window holding Leo's view and his Progress,
  and Canvas inside the window shows Matching's steps with their own zoom
  controls.
