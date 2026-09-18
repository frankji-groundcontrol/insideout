# 2026-09-18 — `client/lib/api/client.dart` over the ~350-line budget

`ApiClient` is ~529 lines: Dio interceptors, auth verbs, workspace /
project / idea / PRD / Coach / roadmap. The 2026-09-18 live-API slice
only changed Coach SSE and could not split the file.

## Target

Three files, same public `ApiClient` type:

- `client_core.dart` — construction, interceptors, `_parse` / `_exec`
- `client_product.dart` — workspace, idea, PRD, commit
- `client_stream.dart` — Coach + presence SSE

## Fix prompt

Split along those seams. Keep `ApiClient` as the type pages import
(part files or a thin barrel). Pin existing `requests_test` /
`feature_wiring_test` / `sse_test`. No new routes or request shapes.
