#!/usr/bin/env python3
"""Fail when the markdown link graph lets a record get lost (review 5, IDX-03).

Usage: python3 -I scripts/tests/check_doc_graph.py

Over every tracked or untracked-but-not-ignored markdown file, fail on:
1. a file no link path reaches from the routers (README.md, CLAUDE.md,
   AGENTS.md, docs/index.md);
2. an index (README.md or index.md under docs/ or hosts/) that does not link
   each direct child file and folder; a link into a folder covers it, and the
   folder's index/ month files count as the index (board, changelogs);
3. a bare link (no ./, ../ or /) matching more than one markdown file by path
   suffix, which is how Marksman resolves it (IDX-05): write ./name.md.
Fenced and inline code are not links.
"""
import collections
import fnmatch
import os
import re
import subprocess
import sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
ROUTERS = ("README.md", "CLAUDE.md", "AGENTS.md", "docs/index.md")
INDEX_ROOTS = ("docs/", "hosts/")

# Children an index may leave out, each with its reason: {index: {child glob: reason}}.
INDEX_EXCEPTIONS = {
    "docs/plans/README.md": {
        "*.track.yaml": "machine-written tracker state beside its plan; the plan is the indexed record",
        "*.track.history.jsonl": "machine-written tracker history beside its plan",
    },
    "docs/review/2026-10-10-multirole-3/README.md": {
        "checks": "dated record keeps its words; the report files cite these check scripts by path",
    },
    "docs/issues/README.md": {
        "2026-11-21-store-error-ladders.md": "open, reached from backend.md; delete once the Open table lists it",
    },
}

INLINE = re.compile(r"\]\(\s*<?([^)\s>]+)>?(?:\s+\"[^\"]*\")?\s*\)")
REFDEF = re.compile(r"^\s{0,3}\[[^\]]+\]:\s*<?(\S+?)>?\s*$")
CODE = re.compile(r"(`+).+?(?<!`)\1(?!`)")
FENCE = re.compile(r"^\s*(`{3,}|~{3,})")
SCHEME = re.compile(r"[a-zA-Z][a-zA-Z0-9+.-]*:")


def git_files(*args):
    out = subprocess.run(["git", "ls-files", *args], cwd=ROOT, capture_output=True, text=True, check=True)
    return [f for f in out.stdout.split("\n") if f and os.path.isfile(os.path.join(ROOT, f))]


def link_targets(text):
    """Link targets outside fenced and inline code."""
    fence = None
    for line in text.split("\n"):
        m = FENCE.match(line)
        if fence:
            if m and m.group(1)[0] == fence[0] and len(m.group(1)) >= len(fence):
                fence = None
            continue
        if m:
            fence = m.group(1)
            continue
        line = CODE.sub("", line)
        yield from INLINE.findall(line)
        ref = REFDEF.match(line)
        if ref:
            yield ref.group(1)


def resolve(src, target):
    """Repository path a relative link reaches, or None for URLs and anchors."""
    if SCHEME.match(target) or target.startswith("#"):
        return None
    path = target.split("#")[0].split("?")[0]
    if not path:
        return None
    if path.startswith("/"):
        return os.path.normpath(path.lstrip("/"))
    return os.path.normpath(os.path.join(os.path.dirname(src), path))


def main():
    every = sorted(set(git_files() + git_files("-o", "--exclude-standard")))
    markdown = [f for f in every if f.endswith(".md")]
    md_set = set(markdown)
    dirs = {os.path.dirname(f) for f in every}
    dirs |= {d[: i] for d in list(dirs) for i in range(len(d)) if d[i] == "/"}
    links = collections.defaultdict(set)  # file -> resolved targets (files or dirs)
    failures = []
    for f in markdown:
        text = open(os.path.join(ROOT, f), encoding="utf-8", errors="replace").read()
        for target in link_targets(text):
            full = resolve(f, target)
            if full is None:
                continue
            links[f].add(full)
            bare = target.split("#")[0]
            if bare and not bare.startswith(("./", "../", "/")) and bare.endswith(".md"):
                hits = [m for m in markdown if m == bare or m.endswith("/" + bare)]
                if len(hits) > 1:
                    failures.append(f"{f}: bare link {bare} matches {len(hits)} files; write ./{bare}")

    # 1. Reachability from the routers; a folder link reaches its README.md or index.md.
    def edges(f):
        for t in links[f]:
            if t in md_set:
                yield t
            elif t in dirs:
                yield from (x for x in (t + "/README.md", t + "/index.md") if x in md_set)
    seen = {r for r in ROUTERS if r in md_set}
    queue = collections.deque(seen)
    while queue:
        for nxt in edges(queue.popleft()):
            if nxt not in seen:
                seen.add(nxt)
                queue.append(nxt)
    failures += [f"{f}: unreachable from the routers {', '.join(ROUTERS)}" for f in markdown if f not in seen]

    # 2. Every index links each direct child file and folder.
    indexes = [f for f in markdown if f.startswith(INDEX_ROOTS) and os.path.basename(f) in ("README.md", "index.md")]
    for idx in indexes:
        folder = os.path.dirname(idx)
        covered = set(links[idx])
        for f in markdown:
            if os.path.dirname(f) == os.path.join(folder, "index"):
                covered |= links[f]
        children = {f for f in every if os.path.dirname(f) == folder}
        children |= {d for d in dirs if os.path.dirname(d) == folder and d}
        children -= {idx}
        if os.path.basename(idx) == "index.md":
            children -= {os.path.join(folder, "README.md")}
        allowed = INDEX_EXCEPTIONS.get(idx, {})
        for child in sorted(children):
            if child in covered or any(c.startswith(child + "/") for c in covered):
                continue
            if any(fnmatch.fnmatch(os.path.relpath(child, folder), glob) for glob in allowed):
                continue
            failures.append(f"{idx}: does not link its child {os.path.relpath(child, folder)}")

    print(f"{len(markdown)} markdown files, {len(seen)} reachable, {len(indexes)} indexes checked")
    for message in failures:
        print(message, file=sys.stderr)
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
