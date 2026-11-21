#!/usr/bin/env python3
"""Fail when a relative Markdown link points at a file that is not there.

Usage: python3 -I scripts/tests/check_doc_links.py

rv4-PM-06: the closure record claimed "zero broken doc links" while a plain
relative-link scan found 76. This is that scan, run over every tracked or
untracked-but-not-ignored `*.md` file with the same rule the closure gate
uses: an inline `](target)` whose target is not an http(s) URL, a mailto: or
a bare `#anchor` must name a path that exists relative to the file holding it.

IDX-09 (review 5): a link from an undated document (no YYYY-MM-DD path part)
with a `#fragment` into a markdown file must name a heading there, slugged the
way GitHub does, or an explicit `<a id>`/`<a name>` anchor. A rewrite that
removed a heading left a link pointing at nothing.

Living docs must have zero. Dated records are history, so a link there to a
path that later moved is a true record of what was cited; they get a ceiling
(an explicit, shrinking allowlist) instead of a free pass: the count may never
grow, and a repair must lower the ceiling in the same change.
"""
import pathlib
import re
import subprocess
import sys
import unicodedata
import urllib.parse

ROOT = pathlib.Path(__file__).resolve().parents[2]
LINK = re.compile(r"\]\((?!https?:|#|mailto:)([^)#\s]+)")
FRAGMENT = re.compile(r"\]\((?!https?:|mailto:)([^)#\s]*)#([^)\s]+)\)")
HEADING = re.compile(r"^ {0,3}#{1,6}\s+(.*?)(?:\s+#+)?\s*$")
ANCHOR = re.compile(r"<a\s[^>]*?(?:id|name)=[\"']([^\"']+)[\"']")
FENCE = re.compile(r"^\s*(```|~~~)")
DATED = re.compile(r"(^|/)\d{4}-\d{2}-\d{2}")

# Broken links tolerated per dated-record prefix. Measured at rv4 wave 3 as 24
# and 18 (every one a path the Nuxt app or Ink & Seal theme once had), then
# repaired; the buckets stay so a future dated record cannot start a debt.
DATED_CEILING = {
    "docs/changelogs/": 0,
    "docs/design-system/CHANGELOG.md": 0,
}


def tracked_markdown():
    """Tracked plus untracked-but-not-ignored markdown files."""
    names = set()
    for extra in ([], ["-o", "--exclude-standard"]):
        out = subprocess.run(["git", "ls-files", *extra, "*.md"], cwd=ROOT, capture_output=True, text=True, check=True)
        names.update(n for n in out.stdout.split("\n") if n and (ROOT / n).is_file())
    return sorted(names)


def slug(text):
    """GitHub's heading id: inline markup dropped, lower case, punctuation removed, spaces to hyphens."""
    text = re.sub(r"!?\[([^\]]*)\]\([^)]*\)", r"\1", text)  # links and images keep their text
    text = re.sub(r"<[^>]+>|`|\*\*?|(?<!\w)_+|_+(?!\w)", "", text).lower()
    keep = (c for c in text if c in " -_" or unicodedata.category(c)[0] in "LNM")
    return "".join(keep).replace(" ", "-")


def anchors(path, cache={}):
    """Every fragment a markdown file answers to, duplicates numbered like GitHub."""
    if path not in cache:
        found, seen, fence = set(), {}, False
        for line in path.read_text(encoding="utf-8").split("\n"):
            if FENCE.match(line):
                fence = not fence
            if fence:
                continue
            found.update(a.lower() for a in ANCHOR.findall(line))
            m = HEADING.match(line)
            if m:
                base = slug(m.group(1))
                n = seen.get(base, 0)
                seen[base] = n + 1
                found.add(base if n == 0 else f"{base}-{n}")
        cache[path] = found
    return cache[path]


def broken_fragments(files):
    found = []
    for name in files:
        if DATED.search(name):
            continue
        path = ROOT / name
        for target, frag in FRAGMENT.findall(path.read_text(encoding="utf-8")):
            dest = (path.parent / target) if target else path
            if dest.suffix == ".md" and dest.is_file():
                if urllib.parse.unquote(frag).lower() not in anchors(dest.resolve()):
                    found.append((name, f"{target}#{frag}"))
    return found


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
    # GitHub's slugs for two headings this repository links to.
    assert slug("Step 1 — `DATABASE_URL` (required)") == "step-1--database_url-required"
    assert slug("Step 0 · **Platform** and [destination](x.md)") == "step-0--platform-and-destination"
    files = tracked_markdown()
    broken = broken_links(files)
    living = [b for b in broken if dated_bucket(b[0]) is None]
    failures = [f"{name}: broken relative link -> {target}" for name, target in living]
    fragments = broken_fragments(files)
    failures += [f"{name}: no heading for fragment -> {target}" for name, target in fragments]
    for prefix, ceiling in DATED_CEILING.items():
        count = sum(1 for name, _ in broken if dated_bucket(name) == prefix)
        if count > ceiling:
            failures.append(f"{prefix}: {count} broken relative links, over the dated-record ceiling of {ceiling}")
    print(f"{len(broken)} broken relative links ({len(living)} in living docs, "
          f"{len(broken) - len(living)} in dated records), {len(fragments)} broken fragments in undated docs, "
          f"across {len(files)} .md files")
    for message in failures:
        print(message, file=sys.stderr)
    sys.exit(1 if failures else 0)
