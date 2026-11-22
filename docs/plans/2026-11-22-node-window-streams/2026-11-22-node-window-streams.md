# 2026-11-22 — Node windows no longer load forever

> "forever loading"

The owner's screenshot shows a node window stuck on "Loading…". The cause is
the browser's six-connection limit, which Server-Sent Events streams use up
([BUG-031](../../issues/2026-11-22-bug-031-node-window-loaded-forever.md)).

## Decisions

These are the coordinator's, stated to the owner:

- **Windows:** a window shares the page's streams instead of opening its
  own. It reloads on the page's progress ticks. This supersedes the
  "Streams" decision of
  [full roadmap in node windows](../2026-11-22-full-roadmap-in-node-windows/2026-11-22-full-roadmap-in-node-windows.md).
- **Hidden tabs:** a hidden tab releases its streams and reconnects and
  reloads when shown.
- **Out of scope:** one multiplexed stream per tab. It is recorded as the
  BUG-031 follow-up.

## Checklist

- [x] `RoadmapController`: `live` flag, `changes` ticks, `setVisible`.
- [x] `RoadmapPage`: windows built with `live: false` and reloading on the
      page's changes; `AppLifecycleListener` on the page.
- [x] Tests: controller visibility, no-stream window, window reload.
- [x] Gates green, a live check with four tabs, records written, PR merged.
