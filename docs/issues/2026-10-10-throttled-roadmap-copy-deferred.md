# Throttled roadmap answers still show the generic error copy

**Closed 2026-10-11** — red-first in `client/test/guarded_action_test.dart`
(both codes proved failing on the generic copy), fixed in
`guardWrite` itself: a backoff code now shows the coach's own
service-down vs rate-limited copy with its countdown, every other error
keeps the generic line; full client suite 458/458.

## What

A roadmap build or expand that the server answers with `429 APP_THROTTLE`
or `503 CIRCUIT_OPEN` surfaces in the client through `guardWrite`'s generic
snackbar — no countdown, no "circuit" wording. The coach panel already maps
these codes to specific copy, so the vocabulary exists.

Found by the round-1 s5a verifier's scope notes (2026-10-10 sweep,
`verdict-s5a-coach-core-r1.json`); scheduled as the optional cluster r2n in
the sweep's round-2 backlog and deferred because its file
(`roadmap_page.dart` error handling) collides with r2a/r2o ownership in the
same wave, and the defect is cosmetic.

## Fix prompt

Red first: pump the roadmap page against a fake API whose
build/expand throws `ApiException` with code `APP_THROTTLE` (then
`CIRCUIT_OPEN`); expect the specific copy the coach panel shows, not the
generic `guardWrite` text. Then map the codes where the coach panel does and
reuse that helper. Keep `guardWrite` for everything else.
