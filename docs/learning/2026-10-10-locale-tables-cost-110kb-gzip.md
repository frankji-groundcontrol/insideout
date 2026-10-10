# zh Material strings cost the web bundle ~110 KB gzip

2026-10-10, the review-register sweep (from the c4 worker's report).

Loading `GlobalMaterialLocalizations` for zh (so Material chrome — dialogs,
pickers, tooltips — speaks Chinese beside the app's own zh strings) pulls in
every Flutter locale table the widget library can reach: the release web
bundle grew about 490 KB raw / 110 KB gzip. The sweep accepted the cost
deliberately — half the audience reads zh — and pinned nothing.

The lesson generalizes: **locale coverage is a bundle-size decision, not a
free default.** When a surface adds a locale, build the release bundle and
diff the sizes in the same change that adds the strings, and record the
number in the changelog (this one is in
[review-sweep-client.md](../changelogs/2026-10-10-close-open-gaps/review-sweep-client.md)).
If the bundle ever matters more than the coverage, the fallback is
per-locale deferred loading, which this project has not built.
