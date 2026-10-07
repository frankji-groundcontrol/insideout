# 2026-10-07 — A documented curl|bash URL is a release asset contract

## What happened

The release pipeline (slice 4) built tarballs + `checksums.txt` and
passed every gate — including an end-to-end bootstrap install. Only
while writing the docs join did the gap surface: the documented
one-liner

```bash
curl -fsSL <release-url>/bootstrap.sh | bash
```

downloads `bootstrap.sh` **from the release**, but the job never
uploaded `bootstrap.sh`. Every binary gate stayed green because the
gates exercised bootstrap.sh from the checkout, exactly as the
staged `file://` test did — the missing link was between a URL in
prose and the upload list in the workflow.

## The lesson

When documentation names a download URL, that URL is part of the
release contract: the named asset must appear in the upload list,
at the exact path the URL implies (`releases/latest/download/<name>`
serves top-level release assets). Gates that test the *script* do
not test the *URL* — write them from the documented command, not
from the file's location in the repo.

Also: ship the installer with the release rather than pointing at
`raw.githubusercontent.com/<branch>` — then the bootstrap a user
curls always matches the assets it verifies and installs, and the
URL survives branch renames.

## When to apply

Any release pipeline, download page, or README quick-start that
tells a user to fetch a file by URL: diff the URL against the
workflow's upload list before landing, and prefer staging the
end-to-end test from a directory laid out exactly like the release
(including the installer) — the mismatch would have shown up as a
404 in that shape.
