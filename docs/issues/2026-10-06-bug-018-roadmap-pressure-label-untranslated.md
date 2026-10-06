# BUG-018 — roadmap pressure renders as a raw localization key

## Symptom

On the web roadmap progress view, each node shows its deadline and then
the literal string `roadmap.pressure.normal` instead of a word such as
"正常". Seen 2026-10-06 on the live sim web client (release build,
Chinese UI) for every node of Mira's project. The API value is the
enum `normal`; CLI and MCP print that enum, which is fine. Only the
web label is missing.

## Status

Fixed 2026-10-06. `roadmap.pressure.normal` is now "normal" / "正常"
in the two locale tables. The progress-view widget test renders a
`normal` item and asserts the raw key is absent. The release bundle
on :5174 predates the fix, so the running web client still shows the
key until it is rebuilt.
