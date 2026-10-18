#!/usr/bin/env python3
"""Fail when a relative Markdown link points at a file that is not there.

Usage: python3 -I scripts/tests/check_doc_links.py

rv4-PM-06: the closure record claimed "zero broken doc links" while a plain
relative-link scan found 76. This is that scan, run over every tracked `*.md`
file with the same rule the closure gate uses: an inline `](target)` whose
target is not an http(s) URL, a mailto: or a bare `#anchor` must name a path
that exists relative to the file holding it (the `#fragment` is dropped; a
fragment is not checked).

Living docs must have zero. Dated records are history, so a link there to a
path that later moved is a true record of what was cited; they get a ceiling
(an explicit, shrinking allowlist) instead of a free pass: the count may never
grow, and a repair must lower the ceiling in the same change.
"""
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
LINK = re.compile(r"\]\((?!https?:|#|mailto:)([^)#\s]+)")

# Broken links tolerated per dated-record prefix (measured at rv4 wave 3).
DATED_CEILING = {
    "docs/changelogs/": 24,
    "docs/design-system/CHANGELOG.md": 18,
}


def tracked_markdown():
    out = subprocess.run(["git", "ls-files", "*.md"], cwd=ROOT, capture_output=True, text=True, check=True)
    return sorted(out.stdout.split())


def broken_links(files):
    found = []
    for name in files:
        path = ROOT / name
        for target in LINK.findall(path.read_text(encoding="utf-8")):
            if not (path.parent / target).exists():
                found.append((name, target))
    return found


def dated_bucket(name):
    for prefix in DATED_CEILING:
        if name.startswith(prefix):
            return prefix
    return None


if __name__ == "__main__":
    files = tracked_markdown()
    broken = broken_links(files)
    living = [b for b in broken if dated_bucket(b[0]) is None]
    failures = [f"{name}: broken relative link -> {target}" for name, target in living]
    for prefix, ceiling in DATED_CEILING.items():
        count = sum(1 for name, _ in broken if dated_bucket(name) == prefix)
        if count > ceiling:
            failures.append(f"{prefix}: {count} broken relative links, over the dated-record ceiling of {ceiling}")
    print(f"{len(broken)} broken relative links ({len(living)} in living docs, "
          f"{len(broken) - len(living)} in dated records) across {len(files)} tracked .md files")
    for message in failures:
        print(message, file=sys.stderr)
    sys.exit(1 if failures else 0)
