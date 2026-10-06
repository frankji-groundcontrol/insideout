# 2026-10-06 — W3-N2 minified console errors

During the Wave 3 narrow dock tap pass (390×844), the browser console
showed three identical uncaught `Error`s. The stack is minified
(`main.dart.js:85738`). No visible UI failure was recorded.

Source:
[shots-wave3 captures](../review/2026-10-06-frontend-cobalt/shots-wave3/captures.md)
(open-item N2, caveat c, and the W3-N2 line).

## Why it stayed open

The waves 4–7 pass did not symbolize a dev build. The local gate
(analyze with no errors or warnings, 143 tests) does not exercise that
console path. The errors had no recorded user-visible impact, so they
were filed instead of folded into the reliability slice.

## Follow-up

Reproduce on a symbolized `flutter run -d chrome` build during the
same narrow dock tap matrix. If the throw is still there, fix that
call site. If it is gone, close this issue from the symbolized log.
