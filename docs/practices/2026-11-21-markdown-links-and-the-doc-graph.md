# Markdown links and the documentation graph

Write every relative link so it names one file, and keep every record
reachable from an index (review 5, `IDX-03`, `IDX-05`, `IDX-11`).

## Trigger

- You add or move a markdown file, or write a link in one.
- You add a dated folder (a review, a plan, a changelog) with its own README.

## Rules

1. A link to a sibling or child starts with `./`: `[plan](./40-remediation-plan.md)`,
   `[keys](./keys/README.md)`. A link upward starts with `../`. Marksman
   matches a link with neither prefix as a path suffix across the whole
   workspace, so `README.md` or `01-findings-register.md` without `./` is
   ambiguous the moment a second file has that name.
2. A new file gets a link from its folder's index: the `README.md` or
   `index.md` beside it, or (for the board and the changelogs) the month
   file under that folder's `index/`. A new folder gets a link into it.
3. Text that only looks like a link, such as `[3-9][0-9]` in a regular
   expression, goes in a code span.

[`.marksman.toml`](../../.marksman.toml) sets `title_from_heading = false`, so a
link to a document resolves to the file even when the document carries two
H1 headings (three dated registers do).

## Verify

```bash
python3 -I scripts/tests/check_doc_graph.py   # unreachable files, index gaps, ambiguous bare links
python3 -I scripts/tests/check_doc_links.py   # missing targets and heading fragments
```

Both run in CI through the `scripts/tests/check_*.py` glob. An index child
that should not be linked goes in `INDEX_EXCEPTIONS` in
`check_doc_graph.py` with its reason.

## Failure signals

- Marksman reports "Ambiguous link to document" in an editor.
- `check_doc_graph.py` prints `bare link ... matches N files` or
  `does not link its child`.
