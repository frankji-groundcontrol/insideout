# 2026-11-22 — The full roadmap of a node inside its window

> "I want to see complete experiences like my own, not simplified to do"

The owner's screenshot shows their own Canvas page. The node window shipped
in [node windows](../2026-11-22-node-windows/2026-11-22-node-windows.md) shows
a simplified detail card: status, owner, progress, evidence. The owner wants
the complete roadmap experience in the window, the same page they use
themselves, scoped to the node and seen as its person in charge.

## Decisions

These are the coordinator's, stated to the owner:

- **The page:** the window body is the real roadmap page for the node:
  `RoadmapPage(rootId: node)` in an embedded mode. It includes:
  - the breadcrumb and title;
  - the person-in-charge banner;
  - the Progress / Bands / Canvas / "Priya's" toggles inside the window;
  - the "New item" action for those who may write;
  - card menus;
  - Canvas pan and zoom.

  It opens on Progress.
- **Sizing:** the embedded page lays out to the window's size. Its
  `MediaQuery` size is the window's, so a small window gets the phone layout.
  No `AppScaffold` chrome or dock inside the window.
- **Clicks inside:**
  - A click on a node inside a window opens another window on the same page
    layer.
  - A double-click, or the window's "Open roadmap", navigates the main page
    to the node.
  - A breadcrumb link inside a window re-roots that window rather than
    navigating the page.
- **Size:** windows open at about 760×600, clamped to the viewport.
- **Streams:** each embedded page keeps its own live streams, and closes them
  when its window closes.

## Checklist

- [ ] F1 (client): embedded `RoadmapPage`, its window sizing, clicks inside
      windows, the breadcrumb re-root, tests.
- [ ] Gates green, a live check on the Spark data, records written, PR merged.
