# 2026-11-22 — Node windows no longer load forever

Plan: [node window streams](../plans/2026-11-22-node-window-streams/2026-11-22-node-window-streams.md).
Owner report: "forever loading", a node window stuck on "Loading…"
([design QA](../design-qa/2026-11-22-full-roadmap-in-node-windows.md#forever-loading)).
Bug record: [BUG-031](../issues/2026-11-22-bug-031-node-window-loaded-forever.md).

| Change | Commits |
| --- | --- |
| A node's window holds no live stream. It reloads when the page's progress stream reports a change. A hidden tab releases its progress and presence streams and reconnects and reloads when it is shown again. Before this change, every window and tab held its own Server-Sent Events streams. A browser gives one server only six HTTP/1.1 connections, so two tabs plus two windows left new requests queued forever. | `234f25e1` |

## Notes

- Presence inside a window is the page's presence: the window shows no
  presence chips of its own.
- Several visible tabs can still use up the connections on plain HTTP. The
  follow-up is recorded in BUG-031.

## Verification

- `flutter analyze`: 8 infos (baseline). `flutter test`: 757 passed.
- Full gates on a fresh database: build, vet, gofmt, Go tests with and without a database (0 skipped), dart format, doc checks and redaction all pass.
- Live as Maya, with four tabs on the roadmap:
  - Hiding three of the tabs closed their six streams on the server.
  - The front tab then loaded and opened three node windows (Matching,
    Compatibility scoring, Safety & trust), each with its breadcrumb,
    banner and Progress.
  - Showing a hidden tab reloaded it and reconnected its streams; hiding it
    again closed them.
