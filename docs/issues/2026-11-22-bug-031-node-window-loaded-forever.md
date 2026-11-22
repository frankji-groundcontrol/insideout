# BUG-031 — a node's window showed "Loading…" forever

**Status:** Fixed 2026-11-22 ([changelog](../changelogs/2026-11-22-node-window-streams.md)).
The follow-up below stays open.

## Symptom

On the local dev build, a node window opened over the roadmap page stayed on
"Loading…". Its view toggle read "Mine" and there was no breadcrumb, so not
even the node read had come back. With three tabs on the roadmap, the page
itself showed "Network problem — check the connection".

## Root cause

A browser opens at most six HTTP/1.1 connections to one server. Each
Server-Sent Events stream keeps one of them open for as long as it runs. The
roadmap page holds two streams: progress and presence. Every embedded page
in a node window opened its own presence stream too, and every other tab on
the roadmap held its own two streams. Two tabs and two windows already use
all six connections. After that, every other request (the window's node
read, glance and progress) waits in the browser's queue and never reaches
the server. The server log showed the window's requests arriving only once
the page reloaded and gave its streams up.

Production behind HTTPS is served over HTTP/2, which multiplexes many
streams over one connection, so this was mostly a local, plain-HTTP failure.
It is still worth avoiding: a hidden tab has no use for a live stream.

## Fix

- A window holds no stream. Its embedded page builds its controller with
  `live: false` and reloads when the page's progress stream ticks
  (`RoadmapController.changes`, handed to windows through
  `RoadmapWindowScope.changes`).
- A hidden tab lets its streams go. The page listens to
  `AppLifecycleListener` and calls `RoadmapController.setVisible`. Hiding
  ends both stream sessions and starts no new ones. Showing the tab again
  reconnects and reloads.

Tests: `roadmap_controller_test.dart` ("a hidden tab lets its streams go…",
"a window controller opens no stream") and
`roadmap_window_embedded_test.dart` ("a window holds no live stream…").

## Follow-up (open)

Several visible tabs, or other pages that stream, can still use up the six
connections on plain HTTP. Two fixes would remove the limit:

- Serve the local dev API over HTTP/2 (TLS), as production does.
- Carry progress and presence on one multiplexed stream per tab, for
  example one `/projects/{id}/live` SSE with both event kinds, or share one
  stream across tabs through a `BroadcastChannel` leader.

**Fix prompt:** "Merge the roadmap's progress and presence SSE streams into
one per-tab stream (server route plus `RoadmapController`), keep the CLI and
MCP snapshots unchanged, and prove four visible tabs plus three node windows
load on plain HTTP."
