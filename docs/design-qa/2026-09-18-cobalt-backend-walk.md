# 2026-09-18 — Cobalt Flutter against a real API

Signed-in Flutter web walk on `http://localhost:5173` with
`API_BASE=http://127.0.0.1:8080/api/v1` (local seed database, not a
fake `ApiClient`). Path URLs. Light/dark and zh-CN/en-US were toggled
on Home. Screenshots under `/opt/cursor/artifacts/screenshots/live-*.png`.
Coach send: `/opt/cursor/artifacts/cobalt-live-coach-sse.mp4`.

## What the walk showed

Home listed the server workspace (numeral `01`). Ideas kept
`Live wire capture 0918` after refresh. Convert opened a real PRD id.
Coach send after the `Uint8List` decode fix returned the offline
template reply (no `Utf8Decoder` TypeError). Human Commit
`Live wire version` is revision 2 on the stack. Roadmap shows
`Ship the live wire check` in the Next band (new nodes are `pending`).
Evidence loaded the seed project update. Dock hops used workspace /
PRD / project UUIDs, not `/dashboard`.

## Follow-ups (not new language)

Offline Coach sends one `delta` then `message_end` within ~15 ms, so
the reply looks instant. `curl -N` still saw `message_start` / `delta`
/ `message_end` as separate flushed frames. Hosted signup stays
invitation-only; this walk did not open a hosted session.

Record: [`docs/changelogs/2026-09-18-cobalt-backend-wire.md`](../changelogs/2026-09-18-cobalt-backend-wire.md).
