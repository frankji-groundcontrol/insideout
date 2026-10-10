# 2026-10-06 — W3-N2 minified console errors

> **Closed, not reproducible 2026-10-10.** A release web build with source maps (`flutter build web --release
> --source-maps`) was run against a local API and driven in a real browser at
> 390x844 through the dock tap matrix (Capture, Ideas, Roadmap, Evidence from the
> workspace, Commit and Version from the PRD) and the page walks. The console held
> no `Error`, exception or uncaught rejection, only the headless renderer's
> CPU-only warning and Flutter's bootstrap debug lines. The build the errors came
> from no longer exists: the dock was rewritten the same week, and the roadmap,
> PRD and project pages have since been rebuilt. Reopen with a symbolized stack if
> it appears again. See [web walk](../changelogs/2026-10-10-close-open-gaps/web-walk-and-phone-layouts.md).
>
> The original record follows, unedited.

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
